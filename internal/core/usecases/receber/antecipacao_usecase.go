package receber

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

const maximoDeTitulosPorBordero = 200

var _ portsin.AntecipacaoUseCase = (*AntecipacaoUsecaseImpl)(nil)

type AntecipacaoUsecaseImpl struct {
	antecipacoes portsout.AntecipacaoRepository
	clientes     portsout.ClienteRepository
	fornecedores portsout.FornecedorRepository
	cobranca
	classificacaoDaContaAPagar
	permissoes
}

func NewAntecipacaoUsecase(
	antecipacoes portsout.AntecipacaoRepository,
	titulos portsout.TituloReceberRepository,
	clientes portsout.ClienteRepository,
	fornecedores portsout.FornecedorRepository,
	contas portsout.ContaBancariaRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *AntecipacaoUsecaseImpl {
	return &AntecipacaoUsecaseImpl{
		antecipacoes:               antecipacoes,
		clientes:                   clientes,
		fornecedores:               fornecedores,
		cobranca:                   cobranca{contas: contas},
		classificacaoDaContaAPagar: classificacaoDaContaAPagar{categorias: categorias},
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
			titulos:  titulos,
		},
	}
}

func (u *AntecipacaoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarAntecipacaoInput) (*domainreceber.Antecipacao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return nil, err
	}
	if len(input.TituloIDs) > maximoDeTitulosPorBordero {
		return nil, domain.ErroValidacao("a antecipação aceita no máximo 200 títulos")
	}
	if err := u.exigirFomentoAtivo(ctx, input.FornecedorID); err != nil {
		return nil, err
	}
	if input.ContaBancariaID != uuid.Nil {
		if err := u.exigirContaDeCobranca(ctx, input.ContaBancariaID, input.EmpresaID); err != nil {
			return nil, err
		}
	}

	titulos := make([]*domainreceber.Titulo, 0, len(input.TituloIDs))
	alterados := make([]portsout.TituloAlterado, 0, len(input.TituloIDs))
	for _, tituloID := range input.TituloIDs {
		titulo, err := u.tituloVisivel(ctx, perfil, tituloID)
		if err != nil {
			return nil, err
		}
		if err := u.exigirClienteQuePermiteAntecipacao(ctx, titulo.ClienteID); err != nil {
			return nil, err
		}
		titulos = append(titulos, titulo)
		alterados = append(alterados, portsout.TituloAlterado{Titulo: titulo, VersaoLida: titulo.Versao, StatusAnterior: titulo.Status})
	}

	antecipacao, historicos, err := domainreceber.NovaAntecipacao(input.EmpresaID, input.UsuarioID, domainreceber.DadosAntecipacao{
		FornecedorID:         input.FornecedorID,
		Numero:               input.Numero,
		OperadoEm:            input.OperadoEm,
		TaxaBP:               input.TaxaBP,
		ValorLiquidoCentavos: input.ValorLiquidoCentavos,
		ContaBancariaID:      input.ContaBancariaID,
	}, titulos, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	for indice := range alterados {
		alterados[indice].Historico = historicos[indice]
	}

	if err := u.antecipacoes.Criar(ctx, &portsout.GravacaoAntecipacao{Antecipacao: antecipacao, Titulos: alterados}); err != nil {
		return nil, err
	}

	return antecipacao, nil
}

func (u *AntecipacaoUsecaseImpl) Obter(ctx context.Context, antecipacaoID, usuarioID uuid.UUID) (*domainreceber.Antecipacao, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.antecipacaoVisivel(ctx, perfil, antecipacaoID)
}

func (u *AntecipacaoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarAntecipacoesInput) ([]*domainreceber.Antecipacao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.AntecipacaoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		FornecedorID:    input.FornecedorID,
		De:              datasOpcionais(input.De),
		Ate:             datasOpcionais(input.Ate),
	}
	if strings.TrimSpace(input.Status) != "" {
		if filtro.Status, err = domainreceber.NovoStatusAntecipacao(input.Status); err != nil {
			return nil, err
		}
	}

	return u.antecipacoes.Listar(ctx, filtro)
}

func (u *AntecipacaoUsecaseImpl) Estornar(ctx context.Context, input portsin.EstornarAntecipacaoInput) (*domainreceber.Antecipacao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	antecipacao, err := u.antecipacaoVisivel(ctx, perfil, input.AntecipacaoID)
	if err != nil {
		return nil, err
	}
	if err := antecipacao.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	titulos := make(map[uuid.UUID]*domainreceber.Titulo, len(antecipacao.Titulos))
	alterados := make([]portsout.TituloAlterado, 0, len(antecipacao.Titulos))
	for _, vinculo := range antecipacao.Titulos {
		titulo, err := u.titulos.Obter(ctx, vinculo.TituloID)
		if err != nil {
			return nil, err
		}
		if titulo == nil {
			return nil, domainreceber.ErroTituloNaoEncontrado()
		}
		titulos[titulo.ID] = titulo
		alterados = append(alterados, portsout.TituloAlterado{Titulo: titulo, VersaoLida: titulo.Versao, StatusAnterior: titulo.Status})
	}

	versaoLida := antecipacao.Versao
	historicos, err := antecipacao.Estornar(input.Motivo, titulos, input.UsuarioID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	for indice := range alterados {
		alterados[indice].Historico = historicos[indice]
	}

	if err := u.antecipacoes.Estornar(ctx, &portsout.EstornoDaAntecipacao{
		Antecipacao: antecipacao, VersaoLida: versaoLida, Titulos: alterados,
	}); err != nil {
		return nil, err
	}

	return antecipacao, nil
}

func (u *AntecipacaoUsecaseImpl) Recomprar(ctx context.Context, input portsin.RecomprarInput) (*domainreceber.Titulo, *domainreceber.Recompra, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, nil, err
	}
	if err := titulo.PodeSerRecomprado(); err != nil {
		return nil, nil, err
	}

	antecipacao, err := u.antecipacoes.ObterVigenteDoTitulo(ctx, titulo.ID)
	if err != nil {
		return nil, nil, err
	}
	var vinculo *domainreceber.AntecipacaoTitulo
	if antecipacao != nil {
		vinculo = antecipacao.Vinculo(titulo.ID)
	}

	agora := time.Now().UTC()
	versaoLida, anterior := titulo.Versao, titulo.Status
	recompra, historico, err := titulo.Recomprar(vinculo, domainreceber.DadosRecompra{
		RecompradoEm:          input.RecompradoEm,
		EncargosCentavos:      input.EncargosCentavos,
		Vencimento:            input.Vencimento,
		CategoriaFinanceiraID: input.CategoriaFinanceiraID,
		Motivo:                input.Motivo,
	}, input.UsuarioID, agora)
	if err != nil {
		return nil, nil, err
	}
	if err := u.exigirCategoriaDeDespesa(ctx, recompra.CategoriaFinanceiraID); err != nil {
		return nil, nil, err
	}

	competencia := domainsolicitacao.CompetenciaDe(recompra.RecompradoEm)
	solicitacao, historicoDaConta, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID:             titulo.EmpresaID,
		SolicitanteID:         input.UsuarioID,
		FornecedorID:          &antecipacao.FornecedorID,
		CategoriaFinanceiraID: recompra.CategoriaFinanceiraID,
		Origem:                domainsolicitacao.OrigemRecompra,
		OrigemID:              recompra.ID,
		ValorCentavos:         recompra.ValorCentavos,
		Vencimento:            recompra.Vencimento,
		Competencia:           &competencia,
		Observacao:            recompra.Observacao(titulo),
	}, agora)
	if err != nil {
		return nil, nil, err
	}

	if err := u.antecipacoes.Recomprar(ctx, &portsout.GravacaoRecompra{
		Titulo:      portsout.TituloAlterado{Titulo: titulo, VersaoLida: versaoLida, StatusAnterior: anterior, Historico: historico},
		Recompra:    recompra,
		ContaAPagar: portsout.ContaAPagarGerada{Solicitacao: solicitacao, Historico: historicoDaConta},
	}); err != nil {
		return nil, nil, err
	}

	return titulo, recompra, nil
}

func (u *AntecipacaoUsecaseImpl) antecipacaoVisivel(ctx context.Context, perfil perfilDoUsuario, antecipacaoID uuid.UUID) (*domainreceber.Antecipacao, error) {
	antecipacao, err := u.antecipacoes.Obter(ctx, antecipacaoID)
	if err != nil {
		return nil, err
	}
	if antecipacao == nil {
		return nil, domainreceber.ErroAntecipacaoNaoEncontrada()
	}
	if !perfil.acesso.Permite(antecipacao.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	return antecipacao, nil
}

func (u *AntecipacaoUsecaseImpl) exigirFomentoAtivo(ctx context.Context, fornecedorID uuid.UUID) error {
	if fornecedorID == uuid.Nil {
		return domain.ErroValidacao("fomento da antecipação é obrigatório")
	}

	fornecedor, err := u.fornecedores.Obter(ctx, fornecedorID)
	if err != nil {
		return err
	}
	if fornecedor == nil {
		return domainreceber.ErroFomentoNaoEncontrado()
	}
	if !fornecedor.Ativo {
		return domainreceber.ErroFomentoInativo()
	}

	return nil
}

func (u *AntecipacaoUsecaseImpl) exigirClienteQuePermiteAntecipacao(ctx context.Context, clienteID uuid.UUID) error {
	cliente, err := u.clientes.Obter(ctx, clienteID)
	if err != nil {
		return err
	}
	if cliente == nil {
		return domainreceber.ErroClienteNaoEncontrado()
	}
	if cliente.ProibeAntecipacao {
		return domainreceber.ErroClienteProibeAntecipacao(cliente.Nome)
	}

	return nil
}
