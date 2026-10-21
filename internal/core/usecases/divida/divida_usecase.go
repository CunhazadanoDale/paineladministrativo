package divida

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
)

var _ portsin.DividaUseCase = (*DividaUsecaseImpl)(nil)

const tamanhoMaximoDescricaoHistoricoSolicitacao = 500

type DividaUsecaseImpl struct {
	fornecedores portsout.FornecedorRepository
	categorias   portsout.CategoriaFinanceiraRepository
	contas       portsout.ContaBancariaRepository
	permissoes
}

func NewDividaUsecase(
	dividas portsout.DividaRepository,
	fornecedores portsout.FornecedorRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *DividaUsecaseImpl {
	return &DividaUsecaseImpl{
		fornecedores: fornecedores,
		categorias:   categorias,
		contas:       contas,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
			dividas:  dividas,
		},
	}
}

func (u *DividaUsecaseImpl) Criar(ctx context.Context, input portsin.CriarDividaInput) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return nil, err
	}
	if err := u.exigirCredorAtivo(ctx, input.FornecedorID); err != nil {
		return nil, err
	}
	informativos, err := u.informativosValidados(ctx, nil, input.EmpresaID, input.Dados)
	if err != nil {
		return nil, err
	}

	parcelas := make([]domaindivida.ParcelaInput, 0, len(input.Parcelas))
	for _, parcela := range input.Parcelas {
		parcelas = append(parcelas, domaindivida.ParcelaInput{
			Vencimento:          parcela.Vencimento,
			AmortizacaoCentavos: parcela.AmortizacaoCentavos,
			JurosCentavos:       parcela.JurosCentavos,
		})
	}

	agora := agoraEmUTC()
	divida, err := domaindivida.NovaDivida(input.EmpresaID, input.UsuarioID, domaindivida.DadosDivida{
		DadosInformativos:    informativos,
		FornecedorID:         input.FornecedorID,
		SaldoInicialCentavos: input.SaldoInicialCentavos,
		DataSaldo:            input.DataSaldo,
		Parcelas:             parcelas,
	}, agora)
	if err != nil {
		return nil, err
	}

	gravacao := &portsout.Gravacao{
		Divida: divida,
		Novo:   true,
		Historicos: []*domaindivida.Historico{
			domaindivida.NovoHistorico(divida.ID, input.UsuarioID, domaindivida.AcaoCriada, nil, divida.Status,
				fmt.Sprintf("saldo devedor %s em %s, %d parcelas", formatarReais(divida.SaldoInicialCentavos),
					divida.DataSaldo.Format("02/01/2006"), len(divida.Parcelas)), agora),
		},
	}
	if err := u.anexarGeracaoDoMes(gravacao, input.UsuarioID, agora); err != nil {
		return nil, err
	}
	if err := u.dividas.Gravar(ctx, gravacao); err != nil {
		return nil, err
	}

	return u.visao(divida, agora), nil
}

func (u *DividaUsecaseImpl) Obter(ctx context.Context, dividaID, usuarioID uuid.UUID) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	divida, err := u.dividaVisivel(ctx, perfil, dividaID)
	if err != nil {
		return nil, err
	}

	return u.visao(divida, agoraEmUTC()), nil
}

func (u *DividaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarDividasInput) ([]*domaindivida.Divida, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}

	filtro := portsout.DividaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		FornecedorID:    input.FornecedorID,
	}
	if input.CredorTipo != "" {
		if filtro.CredorTipo, err = domaindivida.NovoTipoCredor(input.CredorTipo); err != nil {
			return nil, err
		}
	}
	if input.Status != "" {
		if filtro.Status, err = domaindivida.NovoStatus(input.Status); err != nil {
			return nil, err
		}
	}

	return u.dividas.Listar(ctx, filtro)
}

func (u *DividaUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarDividaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.DividaID, input.Versao,
		func(divida *domaindivida.Divida, agora time.Time) (*operacao, error) {
			informativos, err := u.informativosValidados(ctx, divida, divida.EmpresaID, input.Dados)
			if err != nil {
				return nil, err
			}
			if err := divida.Alterar(informativos, agora); err != nil {
				return nil, err
			}

			return &operacao{acao: domaindivida.AcaoAlterada}, nil
		})
}

func (u *DividaUsecaseImpl) AlterarParcela(ctx context.Context, input portsin.AlterarParcelaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.DividaID, input.Versao,
		func(divida *domaindivida.Divida, agora time.Time) (*operacao, error) {
			parcela, detalhe, err := divida.AlterarParcela(input.ParcelaID, input.Vencimento, input.JurosCentavos, agora)
			if err != nil {
				return nil, err
			}

			return &operacao{
				acao:     domaindivida.AcaoParcelaAlterada,
				detalhe:  detalhe,
				gravacao: portsout.Gravacao{ParcelasAlteradas: []*domaindivida.Parcela{parcela}},
			}, nil
		})
}

func (u *DividaUsecaseImpl) Cancelar(ctx context.Context, input portsin.CancelarDividaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.DividaID, input.Versao,
		func(divida *domaindivida.Divida, agora time.Time) (*operacao, error) {
			parcelas, contas, err := divida.Cancelar(input.Motivo, agora)
			if err != nil {
				return nil, err
			}

			return &operacao{
				acao:    domaindivida.AcaoCancelada,
				detalhe: *divida.Motivo,
				gravacao: portsout.Gravacao{
					ParcelasAlteradas: parcelas,
					ContasCanceladas:  contasCanceladas(contas, input.UsuarioID, "dívida cancelada: "+*divida.Motivo),
				},
			}, nil
		})
}

func (u *DividaUsecaseImpl) ListarHistorico(ctx context.Context, dividaID, usuarioID uuid.UUID) ([]*domaindivida.Historico, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if _, err := u.dividaVisivel(ctx, perfil, dividaID); err != nil {
		return nil, err
	}

	return u.dividas.ListarHistorico(ctx, dividaID)
}

func (u *DividaUsecaseImpl) GerarContas(ctx context.Context, input portsin.GerarContasInput) (portsin.ResultadoGeracao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}
	if strings.TrimSpace(input.Competencia) == "" {
		return portsin.ResultadoGeracao{}, domain.ErroValidacao("competência da geração é obrigatória")
	}
	competencia, err := domainsolicitacao.NovaCompetencia(input.Competencia)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}
	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}

	limite := domaindivida.FimDoMes(competencia.Inicio())
	ids, err := u.dividas.ComParcelasAGerar(ctx, limite, escopo.lista())
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}

	resultado := portsin.ResultadoGeracao{
		Competencia: competencia.String(),
		Geradas:     []portsin.GeracaoDaDivida{},
		Recusadas:   []portsin.DividaRecusada{},
	}
	for _, id := range ids {
		geracao, err := u.gerarDaDivida(ctx, id, input.UsuarioID, limite)
		switch {
		case errors.Is(err, domain.ErrValidacao) || errors.Is(err, domain.ErrConflito):
			resultado.Recusadas = append(resultado.Recusadas, portsin.DividaRecusada{DividaID: id, Motivo: motivoDoErro(err)})
		case err != nil:
			return portsin.ResultadoGeracao{}, err
		case !geracao.Vazia():
			resultado.Geradas = append(resultado.Geradas, portsin.GeracaoDaDivida{
				DividaID: id, Parcelas: len(geracao.Parcelas), Contas: len(geracao.Contas),
			})
		}
	}

	return resultado, nil
}

func (u *DividaUsecaseImpl) gerarDaDivida(ctx context.Context, dividaID, usuarioID uuid.UUID, limite time.Time) (*domaindivida.Geracao, error) {
	divida, err := u.dividas.Obter(ctx, dividaID)
	if err != nil {
		return nil, err
	}
	if divida == nil {
		return &domaindivida.Geracao{}, nil
	}

	agora := agoraEmUTC()
	versao := divida.Versao
	geracao, err := divida.GerarContasAte(limite, usuarioID, agora)
	if err != nil || geracao.Vazia() {
		return geracao, err
	}

	if err := u.dividas.Gravar(ctx, &portsout.Gravacao{
		Divida:            divida,
		VersaoLida:        versao,
		StatusAnterior:    divida.Status,
		ParcelasAlteradas: geracao.Parcelas,
		ContasAPagar:      geracao.Contas,
		Historicos: []*domaindivida.Historico{
			domaindivida.NovoHistorico(divida.ID, usuarioID, domaindivida.AcaoContasGeradas, nil, divida.Status, geracao.Detalhe(), agora),
		},
	}); err != nil {
		return nil, err
	}

	return geracao, nil
}

type operacao struct {
	acao     domaindivida.Acao
	detalhe  string
	gravacao portsout.Gravacao
}

func (u *DividaUsecaseImpl) operar(
	ctx context.Context,
	usuarioID, dividaID uuid.UUID,
	versao int,
	executar func(*domaindivida.Divida, time.Time) (*operacao, error),
) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	divida, err := u.dividaVisivel(ctx, perfil, dividaID)
	if err != nil {
		return nil, err
	}
	if err := divida.ConferirVersao(versao); err != nil {
		return nil, err
	}

	agora := agoraEmUTC()
	anterior := divida.Status
	resultado, err := executar(divida, agora)
	if err != nil {
		return nil, err
	}

	gravacao := resultado.gravacao
	gravacao.Divida = divida
	gravacao.VersaoLida = versao
	gravacao.StatusAnterior = anterior
	var statusAnterior *domaindivida.Status
	if anterior != divida.Status {
		statusAnterior = &anterior
	}
	gravacao.Historicos = append([]*domaindivida.Historico{
		domaindivida.NovoHistorico(divida.ID, usuarioID, resultado.acao, statusAnterior, divida.Status, resultado.detalhe, agora),
	}, gravacao.Historicos...)
	if err := u.dividas.Gravar(ctx, &gravacao); err != nil {
		return nil, err
	}

	return u.visao(divida, agora), nil
}

func (u *DividaUsecaseImpl) anexarGeracaoDoMes(gravacao *portsout.Gravacao, usuarioID uuid.UUID, agora time.Time) error {
	divida := gravacao.Divida
	geracao, err := divida.GerarContasAte(domaindivida.FimDoMes(domaindivida.Hoje(agora)), usuarioID, agora)
	if err != nil || geracao.Vazia() {
		return err
	}

	gravacao.ContasAPagar = append(gravacao.ContasAPagar, geracao.Contas...)
	gravacao.Historicos = append(gravacao.Historicos,
		domaindivida.NovoHistorico(divida.ID, usuarioID, domaindivida.AcaoContasGeradas, nil, divida.Status, geracao.Detalhe(), agora))

	return nil
}

func (u *DividaUsecaseImpl) visao(divida *domaindivida.Divida, agora time.Time) *portsin.Visao {
	return &portsin.Visao{Divida: divida, Hoje: domaindivida.Hoje(agora)}
}

func (u *DividaUsecaseImpl) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa da dívida é obrigatória")
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domaindivida.ErroEmpresaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domaindivida.ErroEmpresaInativa()
	}

	return nil
}

func (u *DividaUsecaseImpl) exigirCredorAtivo(ctx context.Context, fornecedorID uuid.UUID) error {
	if fornecedorID == uuid.Nil {
		return domain.ErroValidacao("credor da dívida é obrigatório")
	}

	fornecedor, err := u.fornecedores.Obter(ctx, fornecedorID)
	if err != nil {
		return err
	}
	if fornecedor == nil {
		return domaindivida.ErroFornecedorNaoEncontrado()
	}
	if !fornecedor.Ativo {
		return domaindivida.ErroFornecedorInativo(fornecedor.Nome)
	}

	return nil
}

func (u *DividaUsecaseImpl) informativosValidados(
	ctx context.Context, atual *domaindivida.Divida, empresaID uuid.UUID, input portsin.DadosInformativosInput,
) (domaindivida.DadosInformativos, error) {
	credor, err := domaindivida.NovoTipoCredor(input.CredorTipo)
	if err != nil {
		return domaindivida.DadosInformativos{}, err
	}
	taxa, err := domaindivida.NovaTaxa(input.TaxaBP, input.TaxaPeriodo)
	if err != nil {
		return domaindivida.DadosInformativos{}, err
	}
	indexador, err := domaindivida.NovoIndexador(input.Indexador)
	if err != nil {
		return domaindivida.DadosInformativos{}, err
	}

	if atual == nil || atual.CategoriaAmortizacaoID != input.CategoriaAmortizacaoID {
		if err := u.exigirCategoria(ctx, input.CategoriaAmortizacaoID, domainfinanceiro.LinhaForaDRE, "amortização"); err != nil {
			return domaindivida.DadosInformativos{}, err
		}
	}
	if atual == nil || atual.CategoriaJurosID != input.CategoriaJurosID {
		if err := u.exigirCategoria(ctx, input.CategoriaJurosID, domainfinanceiro.LinhaResultadoFinanceiro, "juros"); err != nil {
			return domaindivida.DadosInformativos{}, err
		}
	}
	if input.ContaLiberacaoID != nil && (atual == nil || atual.ContaLiberacaoID == nil || *atual.ContaLiberacaoID != *input.ContaLiberacaoID) {
		if err := u.exigirContaDaEmpresa(ctx, *input.ContaLiberacaoID, empresaID); err != nil {
			return domaindivida.DadosInformativos{}, err
		}
	}

	return domaindivida.DadosInformativos{
		CredorTipo:             credor,
		Descricao:              input.Descricao,
		Contrato:               input.Contrato,
		ValorOriginalCentavos:  input.ValorOriginalCentavos,
		ContratadaEm:           input.ContratadaEm,
		Taxa:                   taxa,
		Indexador:              indexador,
		ContaLiberacaoID:       input.ContaLiberacaoID,
		LiberadaEm:             input.LiberadaEm,
		CategoriaAmortizacaoID: input.CategoriaAmortizacaoID,
		CategoriaJurosID:       input.CategoriaJurosID,
		Observacoes:            input.Observacoes,
	}, nil
}

func (u *DividaUsecaseImpl) exigirCategoria(ctx context.Context, categoriaID uuid.UUID, linha domainfinanceiro.LinhaDRE, componente string) error {
	if categoriaID == uuid.Nil {
		return domain.ErroValidacao("categoria financeira de " + componente + " é obrigatória")
	}

	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domain.ErroValidacao("categoria financeira de " + componente + " não encontrada")
	}
	if !categoria.Lancavel() {
		return domain.ErroValidacao("categoria financeira de " + componente + " deve ser uma subcategoria ativa")
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa || categoria.LinhaDRE != linha {
		return domain.ErroValidacao(fmt.Sprintf("categoria financeira de %s deve ser de despesa na linha %s", componente, linha))
	}

	return nil
}

func (u *DividaUsecaseImpl) exigirContaDaEmpresa(ctx context.Context, contaID, empresaID uuid.UUID) error {
	conta, err := u.contas.Obter(ctx, contaID)
	if err != nil {
		return err
	}
	if conta == nil {
		return domain.ErroValidacao("conta bancária da liberação não encontrada")
	}
	if conta.EmpresaID != empresaID {
		return domain.ErroValidacao("conta bancária da liberação deve ser da empresa da dívida")
	}

	return nil
}

func contasCanceladas(contas []domaindivida.ContaCancelada, usuarioID uuid.UUID, descricao string) []portsout.ContaCancelada {
	if runas := []rune(descricao); len(runas) > tamanhoMaximoDescricaoHistoricoSolicitacao {
		descricao = string(runas[:tamanhoMaximoDescricaoHistoricoSolicitacao])
	}

	canceladas := make([]portsout.ContaCancelada, 0, len(contas))
	for _, conta := range contas {
		anterior := conta.StatusAnterior
		canceladas = append(canceladas, portsout.ContaCancelada{
			ContaCancelada: conta,
			Historico: domainsolicitacao.NovoHistorico(conta.SolicitacaoID, usuarioID, &anterior,
				domainsolicitacao.StatusCancelado, domainsolicitacao.EventoCancelada, descricao),
		})
	}

	return canceladas
}

func motivoDoErro(err error) string {
	texto := err.Error()
	for _, sentinela := range []error{domain.ErrValidacao, domain.ErrConflito} {
		texto = strings.TrimPrefix(texto, sentinela.Error()+": ")
	}

	return texto
}

func formatarReais(centavos int64) string {
	return fmt.Sprintf("R$ %d,%02d", centavos/100, centavos%100)
}

func agoraEmUTC() time.Time {
	return time.Now().UTC()
}
