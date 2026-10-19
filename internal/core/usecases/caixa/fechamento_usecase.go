package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.FechamentoUseCase = (*FechamentoUsecaseImpl)(nil)

type FechamentoUsecaseImpl struct {
	fechamentos  portsout.FechamentoRepository
	aberturas    portsout.AberturaRepository
	conciliacoes portsout.ConciliacaoRepository
	permissoes
	escopoPorEmpresa
}

func NewFechamentoUsecase(
	fechamentos portsout.FechamentoRepository,
	aberturas portsout.AberturaRepository,
	conciliacoes portsout.ConciliacaoRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *FechamentoUsecaseImpl {
	return &FechamentoUsecaseImpl{
		fechamentos:      fechamentos,
		aberturas:        aberturas,
		conciliacoes:     conciliacoes,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

type contextoDoFechamento struct {
	previa   *domaincaixa.PreviaFechamento
	abertura *domaincaixa.Abertura
	ultimo   *domaincaixa.Fechamento
}

func (u *FechamentoUsecaseImpl) Previa(ctx context.Context, input portsin.PreviaInput) (*domaincaixa.PreviaFechamento, error) {
	contexto, err := u.montarPrevia(ctx, input.UsuarioID, input.ContaBancariaID, input.Competencia, false)
	if err != nil {
		return nil, err
	}

	return contexto.previa, nil
}

func (u *FechamentoUsecaseImpl) Fechar(ctx context.Context, input portsin.FecharInput) (*portsin.DetalheFechamento, error) {
	contexto, err := u.montarPrevia(ctx, input.UsuarioID, input.ContaBancariaID, input.Competencia, true)
	if err != nil {
		return nil, err
	}

	fechamento, err := domaincaixa.NovoFechamento(domaincaixa.DadosFechamento{
		Previa:              contexto.previa,
		SaldoBancoInformado: input.SaldoBancoCentavos,
		UsuarioID:           input.UsuarioID,
		Agora:               time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	gravacao := &portsout.GravacaoFechamento{
		Fechamento:       fechamento,
		VersaoDaAbertura: contexto.abertura.Versao,
		SaldoBancoLido:   contexto.previa.SaldoBancoCentavos,
	}
	if contexto.ultimo != nil {
		gravacao.UltimoID = &contexto.ultimo.ID
	}
	if err := u.fechamentos.Fechar(ctx, gravacao); err != nil {
		return nil, err
	}

	return &portsin.DetalheFechamento{
		Fechamento:              fechamento,
		ConciliacoesInvalidadas: contexto.previa.ConciliacoesInvalidadas,
		MovimentosSemLinha:      contexto.previa.MovimentosSemLinha,
	}, nil
}

func (u *FechamentoUsecaseImpl) Obter(ctx context.Context, fechamentoID, usuarioID uuid.UUID) (*portsin.DetalheFechamento, error) {
	fechamento, _, err := u.fechamentoComAcesso(ctx, fechamentoID, usuarioID, u.exigeFinanceiro)
	if err != nil {
		return nil, err
	}

	abertura, err := u.aberturas.Obter(ctx, fechamento.ContaBancariaID)
	if err != nil {
		return nil, err
	}
	if abertura == nil {
		return nil, domaincaixa.ErroContaSemAbertura()
	}

	movimento, err := u.fechamentos.MovimentoDoMes(ctx, fechamento.ContaBancariaID, fechamento.Competencia, abertura.DataCorte)
	if err != nil {
		return nil, err
	}
	semLinha, err := u.movimentosSemLinha(ctx, fechamento.ContaBancariaID, fechamento.Competencia)
	if err != nil {
		return nil, err
	}

	return &portsin.DetalheFechamento{
		Fechamento:              fechamento,
		ConciliacoesInvalidadas: movimento.ConciliacoesInvalidadas,
		MovimentosSemLinha:      semLinha,
	}, nil
}

func (u *FechamentoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarFechamentosInput) ([]*domaincaixa.Fechamento, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	filtro := portsout.FechamentoFiltro{}
	if input.ContaBancariaID != nil {
		conta, err := u.contaComAcesso(ctx, *input.ContaBancariaID, input.UsuarioID)
		if err != nil {
			return nil, err
		}
		filtro.ContaBancariaID = &conta.ID
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}
	filtro.EmpresaIDs = empresaIDs

	if input.Competencia != nil {
		competencia := domaincaixa.CompetenciaDe(*input.Competencia)
		filtro.Competencia = &competencia
	}

	paginacao := input.Filtro.Normalizada()
	filtro.Limite = paginacao.Size
	filtro.Deslocamento = (paginacao.Page - 1) * paginacao.Size

	return u.fechamentos.Listar(ctx, filtro)
}

func (u *FechamentoUsecaseImpl) Reabrir(ctx context.Context, input portsin.ReabrirInput) (*domaincaixa.Fechamento, error) {
	fechamento, _, err := u.fechamentoComAcesso(ctx, input.FechamentoID, input.UsuarioID, u.exigeAdministrador)
	if err != nil {
		return nil, err
	}

	if err := fechamento.Reabrir(input.UsuarioID, input.Motivo, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.fechamentos.Reabrir(ctx, fechamento); err != nil {
		return nil, err
	}

	return fechamento, nil
}

func (u *FechamentoUsecaseImpl) montarPrevia(
	ctx context.Context,
	usuarioID, contaID uuid.UUID,
	competencia time.Time,
	exigirEncerrado bool,
) (*contextoDoFechamento, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, contaID, usuarioID)
	if err != nil {
		return nil, err
	}

	abertura, err := u.aberturas.Obter(ctx, conta.ID)
	if err != nil {
		return nil, err
	}
	ultimo, err := u.fechamentos.Ultimo(ctx, conta.ID)
	if err != nil {
		return nil, err
	}
	if err := domaincaixa.ConferirCompetencia(competencia, abertura, ultimo, domaincaixa.Hoje(time.Now().UTC()), exigirEncerrado); err != nil {
		return nil, err
	}

	competencia = domaincaixa.CompetenciaDe(competencia)
	movimento, err := u.fechamentos.MovimentoDoMes(ctx, conta.ID, competencia, abertura.DataCorte)
	if err != nil {
		return nil, err
	}
	semLinha, err := u.movimentosSemLinha(ctx, conta.ID, competencia)
	if err != nil {
		return nil, err
	}

	return &contextoDoFechamento{
		previa: &domaincaixa.PreviaFechamento{
			ContaBancariaID:      conta.ID,
			Competencia:          competencia,
			SaldoInicialCentavos: domaincaixa.SaldoInicialDaCompetencia(abertura, ultimo),
			MesEncerrado:         competencia.Before(domaincaixa.CompetenciaDe(domaincaixa.Hoje(time.Now().UTC()))),
			MovimentoDoMes:       *movimento,
			MovimentosSemLinha:   semLinha,
		},
		abertura: abertura,
		ultimo:   ultimo,
	}, nil
}

func (u *FechamentoUsecaseImpl) movimentosSemLinha(ctx context.Context, contaID uuid.UUID, competencia time.Time) ([]*domaincaixa.MovimentoSistema, error) {
	de, ate := domaincaixa.CompetenciaDe(competencia), domaincaixa.FimDaCompetencia(competencia)

	return u.conciliacoes.Movimentos(ctx, portsout.MovimentoFiltro{
		ContaBancariaID: contaID,
		De:              &de,
		Ate:             &ate,
		SoPendentes:     true,
		Limite:          limiteDeMovimentosDaConta,
	})
}

func (u *FechamentoUsecaseImpl) fechamentoComAcesso(
	ctx context.Context,
	fechamentoID, usuarioID uuid.UUID,
	exigirPerfil func(context.Context, uuid.UUID) error,
) (*domaincaixa.Fechamento, *domainfinanceiro.ContaBancaria, error) {
	if err := exigirPerfil(ctx, usuarioID); err != nil {
		return nil, nil, err
	}

	fechamento, err := u.fechamentos.Obter(ctx, fechamentoID)
	if err != nil {
		return nil, nil, err
	}
	if fechamento == nil {
		return nil, nil, domaincaixa.ErroFechamentoNaoEncontrado()
	}

	conta, err := u.contaComAcesso(ctx, fechamento.ContaBancariaID, usuarioID)
	if err != nil {
		return nil, nil, err
	}

	return fechamento, conta, nil
}
