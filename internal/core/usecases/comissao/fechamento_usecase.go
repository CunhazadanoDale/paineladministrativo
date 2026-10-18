package comissao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

var _ portsin.FechamentoUseCase = (*FechamentoUsecaseImpl)(nil)

type FechamentoUsecaseImpl struct {
	comissoes  portsout.ComissaoRepository
	categorias portsout.CategoriaFinanceiraRepository
	apuracao   apurador
	permissoes
}

func NewFechamentoUsecase(
	comissoes portsout.ComissaoRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *FechamentoUsecaseImpl {
	return &FechamentoUsecaseImpl{
		comissoes:  comissoes,
		categorias: categorias,
		apuracao:   apurador{comissoes: comissoes, representantes: representantes},
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
		},
	}
}

func (u *FechamentoUsecaseImpl) Fechar(ctx context.Context, input portsin.FecharInput) (domaincomissao.ResultadoFechamento, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}

	agora := time.Now().UTC()
	competencia, err := domaincomissao.NovaCompetencia(input.Competencia)
	if err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}
	if err := domaincomissao.ConferirCompetenciaDeFechamento(competencia, agora); err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}
	if err := u.exigirCategoriaDeDespesa(ctx, input.CategoriaFinanceiraID); err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}

	empresaID := input.EmpresaID
	limite := competencia.Seguinte().Inicio()
	apuracao, err := u.apuracao.apurar(ctx, portsout.EntregaFiltro{Empresa: &empresaID, SaidaAntesDe: &limite}, input.UsuarioID)
	if err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}
	if apuracao.RestamEntregas {
		return domaincomissao.ResultadoFechamento{}, domaincomissao.ErroApuracaoPendente()
	}

	saldos, err := u.comissoes.SaldosAbertos(ctx, portsout.SaldoFiltro{
		EmpresaID: input.EmpresaID, Ate: competencia, RepresentanteID: input.RepresentanteID,
	})
	if err != nil {
		return domaincomissao.ResultadoFechamento{}, err
	}
	if input.RepresentanteID != nil && len(saldos) == 0 {
		return domaincomissao.ResultadoFechamento{}, domaincomissao.ErroNadaAPagar()
	}

	resultado := domaincomissao.ResultadoFechamento{
		Apuracao:    apuracao,
		Competencia: competencia,
		Fechados:    []*domaincomissao.Fechamento{},
		Falhas:      []domaincomissao.FalhaFechamento{},
	}
	for _, saldo := range saldos {
		fechamento, err := u.fecharRepresentante(ctx, input, competencia, saldo, agora)
		if err == nil {
			resultado.Fechados = append(resultado.Fechados, fechamento)
			continue
		}
		if !erroDeNegocio(err) {
			return domaincomissao.ResultadoFechamento{}, err
		}
		resultado.Falhas = append(resultado.Falhas, domaincomissao.FalhaFechamento{RepresentanteID: saldo.RepresentanteID, Motivo: err.Error()})
	}

	return resultado, nil
}

func (u *FechamentoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarFechamentosInput) ([]*domaincomissao.Fechamento, error) {
	perfil, err := u.exigeLeituraDeComissoes(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa, input.RepresentanteID)
	if err != nil {
		return nil, err
	}

	filtro := portsout.FechamentoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		RepresentanteID: escopo.representanteID,
	}
	if filtro.Competencia, err = competenciaOpcional(input.Competencia); err != nil {
		return nil, err
	}

	return u.comissoes.ListarFechamentos(ctx, filtro)
}

func (u *FechamentoUsecaseImpl) Obter(ctx context.Context, fechamentoID, usuarioID uuid.UUID) (*domaincomissao.FechamentoComItens, error) {
	perfil, err := u.exigeLeituraDeComissoes(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	fechamento, err := u.comissoes.ObterFechamento(ctx, fechamentoID)
	if err != nil {
		return nil, err
	}
	if fechamento == nil {
		return nil, domaincomissao.ErroFechamentoNaoEncontrado()
	}
	if !perfil.acesso.Permite(fechamento.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}
	if perfil.restritoAoRepresentante() && fechamento.RepresentanteID != *perfil.representanteID {
		return nil, erroDeOutroRepresentante()
	}

	comissoes, ajustes, err := u.comissoes.ItensDoFechamento(ctx, fechamento.ID)
	if err != nil {
		return nil, err
	}

	return &domaincomissao.FechamentoComItens{Fechamento: fechamento, Comissoes: comissoes, Ajustes: ajustes}, nil
}

func (u *FechamentoUsecaseImpl) fecharRepresentante(
	ctx context.Context,
	input portsin.FecharInput,
	competencia domaincomissao.Competencia,
	saldo domaincomissao.SaldoAberto,
	agora time.Time,
) (*domaincomissao.Fechamento, error) {
	representante, err := u.representantes.Obter(ctx, saldo.RepresentanteID)
	if err != nil {
		return nil, err
	}
	if representante == nil {
		return nil, domaincomissao.ErroRepresentanteDoFechamentoNaoEncontrado()
	}
	if representante.FornecedorID == nil {
		return nil, domaincomissao.ErroRepresentanteSemFornecedor()
	}

	fechamento, err := domaincomissao.NovoFechamento(input.EmpresaID, competencia, saldo, input.CategoriaFinanceiraID, input.UsuarioID, agora)
	if err != nil {
		return nil, err
	}

	competenciaDaConta := domainsolicitacao.CompetenciaDe(competencia.Inicio())
	solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID:             input.EmpresaID,
		SolicitanteID:         input.UsuarioID,
		FornecedorID:          representante.FornecedorID,
		CategoriaFinanceiraID: input.CategoriaFinanceiraID,
		Origem:                domainsolicitacao.OrigemComissao,
		OrigemID:              fechamento.ID,
		ValorCentavos:         fechamento.TotalCentavos,
		Vencimento:            fechamento.Vencimento,
		Competencia:           &competenciaDaConta,
		Observacao:            "comissão de " + representante.Nome + " — competência " + competencia.String(),
	}, agora)
	if err != nil {
		return nil, err
	}
	fechamento.SolicitacaoID = solicitacao.ID
	fechamento.StatusContaAPagar = solicitacao.Status.String()

	if err := u.comissoes.Fechar(ctx, &portsout.GravacaoFechamento{
		Fechamento: fechamento, Solicitacao: solicitacao, Historico: historico,
	}); err != nil {
		return nil, err
	}

	return fechamento, nil
}

func (u *FechamentoUsecaseImpl) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa do fechamento é obrigatória")
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domain.ErroValidacao("empresa do fechamento não encontrada")
	}
	if !empresa.Ativa {
		return domain.ErroValidacao("empresa inativa não fecha comissão")
	}

	return nil
}

func (u *FechamentoUsecaseImpl) exigirCategoriaDeDespesa(ctx context.Context, categoriaID uuid.UUID) error {
	if categoriaID == uuid.Nil {
		return domain.ErroValidacao("categoria financeira do fechamento é obrigatória")
	}

	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domain.ErroValidacao("categoria financeira não encontrada")
	}
	if !categoria.Lancavel() {
		return domain.ErroValidacao("categoria financeira deve ser uma subcategoria ativa")
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa {
		return domain.ErroValidacao("categoria financeira do fechamento deve ser de despesa")
	}

	return nil
}
