package receber

import (
	"context"
	"strings"
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

var _ portsin.TituloUseCase = (*TituloUsecaseImpl)(nil)

type TituloUsecaseImpl struct {
	clientes   portsout.ClienteRepository
	categorias portsout.CategoriaFinanceiraRepository
	cobranca
	permissoes
}

func NewTituloUsecase(
	titulos portsout.TituloReceberRepository,
	clientes portsout.ClienteRepository,
	contas portsout.ContaBancariaRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *TituloUsecaseImpl {
	return &TituloUsecaseImpl{
		clientes:   clientes,
		categorias: categorias,
		cobranca:   cobranca{contas: contas},
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
			titulos:  titulos,
		},
	}
}

func (u *TituloUsecaseImpl) Criar(ctx context.Context, input portsin.CriarTituloInput) (*domainreceber.Titulo, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return nil, err
	}
	if err := u.exigirClienteAtivo(ctx, input.ClienteID); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, input.Dados)
	if err != nil {
		return nil, err
	}
	if input.ContaBancariaID != nil && *input.ContaBancariaID != uuid.Nil {
		if err := u.exigirContaDeCobranca(ctx, *input.ContaBancariaID, input.EmpresaID); err != nil {
			return nil, err
		}
	}

	titulo, err := domainreceber.NovoTituloManual(input.EmpresaID, input.UsuarioID, domainreceber.DadosTitulo{
		DadosEdicao:     dados,
		ClienteID:       input.ClienteID,
		ContaBancariaID: input.ContaBancariaID,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	historico := domainreceber.NovoHistorico(titulo.ID, input.UsuarioID, domainreceber.EventoCriado, nil, titulo.Status, "", titulo.CriadoEm)
	if err := u.titulos.Criar(ctx, titulo, historico); err != nil {
		return nil, err
	}

	return titulo, nil
}

func (u *TituloUsecaseImpl) Obter(ctx context.Context, tituloID, usuarioID uuid.UUID) (*domainreceber.Titulo, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.tituloVisivel(ctx, perfil, tituloID)
}

func (u *TituloUsecaseImpl) Listar(ctx context.Context, input portsin.ListarTitulosInput) ([]*domainreceber.Titulo, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.TituloFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		ClienteID:       input.ClienteID,
		OrigemID:        input.OrigemID,
		ContaBancariaID: input.ContaBancariaID,
		VencimentoDe:    datasOpcionais(input.VencimentoDe),
		VencimentoAte:   datasOpcionais(input.VencimentoAte),
		OrigemCancelada: input.OrigemCancelada,
	}
	if strings.TrimSpace(input.Status) != "" {
		if filtro.Status, err = domainreceber.NovoStatus(input.Status); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(input.Origem) != "" {
		if filtro.Origem, err = domainreceber.NovaOrigem(input.Origem); err != nil {
			return nil, err
		}
	}
	if input.Vencidos {
		hoje := domainreceber.Hoje(time.Now().UTC())
		filtro.VencidosEm = &hoje
	}

	return u.titulos.Listar(ctx, filtro)
}

func (u *TituloUsecaseImpl) ListarHistorico(ctx context.Context, tituloID, usuarioID uuid.UUID) ([]*domainreceber.Historico, error) {
	if _, err := u.Obter(ctx, tituloID, usuarioID); err != nil {
		return nil, err
	}

	return u.titulos.ListarHistorico(ctx, tituloID)
}

func (u *TituloUsecaseImpl) Editar(ctx context.Context, input portsin.EditarTituloInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}
	if err := titulo.PodeSerEditado(); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, input.Dados)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	descricao, err := titulo.Editar(dados, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if descricao == "" {
		return titulo, nil
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoEditado, descricao, true, false)
}

func (u *TituloUsecaseImpl) Prorrogar(ctx context.Context, input portsin.ProrrogarTituloInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	descricao, err := titulo.Prorrogar(input.Vencimento, input.Motivo, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoProrrogado, descricao, false, false)
}

func (u *TituloUsecaseImpl) EnviarAoBanco(ctx context.Context, input portsin.EnviarAoBancoInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	if err := titulo.EnviarAoBanco(input.ContaBancariaID, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.exigirContaDeCobranca(ctx, input.ContaBancariaID, titulo.EmpresaID); err != nil {
		return nil, err
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoEnviadoBanco, "", false, true)
}

func (u *TituloUsecaseImpl) RetirarDoBanco(ctx context.Context, input portsin.TransicaoTituloInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	if err := titulo.RetirarDoBanco(time.Now().UTC()); err != nil {
		return nil, err
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoRetiradoBanco, "", false, false)
}

func (u *TituloUsecaseImpl) RegistrarEnvioBoleto(ctx context.Context, input portsin.RegistrarEnvioBoletoInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	if err := titulo.RegistrarEnvioBoleto(input.Canal, input.EnviadoEm, time.Now().UTC()); err != nil {
		return nil, err
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoBoletoEnviado, "canal: "+titulo.BoletoCanal.String(), false, false)
}

func (u *TituloUsecaseImpl) Cancelar(ctx context.Context, input portsin.CancelarTituloInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	anterior := titulo.Status
	if err := titulo.Cancelar(input.Motivo, time.Now().UTC()); err != nil {
		return nil, err
	}

	return u.gravar(ctx, input.UsuarioID, titulo, input.Versao, anterior, domainreceber.EventoCancelado, *titulo.CanceladoMotivo, false, false)
}

func (u *TituloUsecaseImpl) gravar(
	ctx context.Context,
	usuarioID uuid.UUID,
	titulo *domainreceber.Titulo,
	versaoLida int,
	anterior domainreceber.Status,
	evento domainreceber.Evento,
	descricao string,
	conferirCategoria, conferirConta bool,
) (*domainreceber.Titulo, error) {
	historico := domainreceber.NovoHistorico(titulo.ID, usuarioID, evento, &anterior, titulo.Status, descricao, titulo.AtualizadoEm)

	if err := u.titulos.Atualizar(ctx, &portsout.Alteracao{
		Titulo:            titulo,
		VersaoLida:        versaoLida,
		StatusAnterior:    anterior,
		Historico:         historico,
		ConferirCategoria: conferirCategoria,
		ConferirConta:     conferirConta,
	}); err != nil {
		return nil, err
	}

	return titulo, nil
}

func (u *TituloUsecaseImpl) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domainreceber.ErroEmpresaObrigatoria()
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainreceber.ErroEmpresaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainreceber.ErroEmpresaInativa()
	}

	return nil
}

func (u *TituloUsecaseImpl) exigirClienteAtivo(ctx context.Context, clienteID uuid.UUID) error {
	if clienteID == uuid.Nil {
		return domainreceber.ErroClienteObrigatorio()
	}

	cliente, err := u.clientes.Obter(ctx, clienteID)
	if err != nil {
		return err
	}
	if cliente == nil {
		return domainreceber.ErroClienteNaoEncontrado()
	}
	if !cliente.Ativo {
		return domainreceber.ErroClienteInativo()
	}

	return nil
}

func (u *TituloUsecaseImpl) exigirCategoriaDeReceita(ctx context.Context, categoriaID uuid.UUID) error {
	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domainreceber.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if !categoria.Lancavel() {
		return domainreceber.ErroCategoriaFinanceiraNaoLancavel()
	}
	if categoria.Natureza != domainfinanceiro.NaturezaReceita {
		return domainreceber.ErroCategoriaFinanceiraNaoEhReceita()
	}

	return nil
}

func (u *TituloUsecaseImpl) dadosValidados(ctx context.Context, input portsin.DadosEdicaoInput) (domainreceber.DadosEdicao, error) {
	dados := domainreceber.DadosEdicao{
		Numero:                input.Numero,
		Parcela:               input.Parcela,
		Vencimento:            input.Vencimento,
		ValorCentavos:         input.ValorCentavos,
		CategoriaFinanceiraID: input.CategoriaFinanceiraID,
		Observacao:            input.Observacao,
	}
	if strings.TrimSpace(input.Competencia) != "" {
		competencia, err := domainreceber.NovaCompetencia(input.Competencia)
		if err != nil {
			return domainreceber.DadosEdicao{}, err
		}
		dados.Competencia = &competencia
	}
	if dados.CategoriaFinanceiraID != nil && *dados.CategoriaFinanceiraID != uuid.Nil {
		if err := u.exigirCategoriaDeReceita(ctx, *dados.CategoriaFinanceiraID); err != nil {
			return domainreceber.DadosEdicao{}, err
		}
	}

	return dados, nil
}

func datasOpcionais(valor *time.Time) *time.Time {
	if valor == nil || valor.IsZero() {
		return nil
	}

	data := domainreceber.DataDe(*valor)

	return &data
}
