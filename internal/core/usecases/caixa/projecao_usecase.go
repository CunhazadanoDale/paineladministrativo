package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.ProjecaoUseCase = (*ProjecaoUsecaseImpl)(nil)

type ProjecaoUsecaseImpl struct {
	projecoes    portsout.ProjecaoRepository
	recorrencias portsout.DespesaRecorrenteRepository
	permissoes
	escopoPorEmpresa
}

func NewProjecaoUsecase(
	projecoes portsout.ProjecaoRepository,
	recorrencias portsout.DespesaRecorrenteRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ProjecaoUsecaseImpl {
	return &ProjecaoUsecaseImpl{
		projecoes:        projecoes,
		recorrencias:     recorrencias,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

func (u *ProjecaoUsecaseImpl) Projetar(ctx context.Context, input portsin.ProjetarInput) (*domaincaixa.Projecao, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	hoje := domaincaixa.Hoje(time.Now().UTC())
	ate, err := domaincaixa.HorizonteDaProjecao(hoje, input.Ate)
	if err != nil {
		return nil, err
	}

	partida, err := u.projecoes.SaldoDePartida(ctx, empresaIDs)
	if err != nil {
		return nil, err
	}
	entradas, err := u.projecoes.TitulosAReceber(ctx, empresaIDs, ate)
	if err != nil {
		return nil, err
	}
	saidas, err := u.projecoes.ContasAPagar(ctx, empresaIDs, ate)
	if err != nil {
		return nil, err
	}
	recorrentes, err := u.recorrentesPrevistas(ctx, empresaIDs, hoje, ate)
	if err != nil {
		return nil, err
	}

	return domaincaixa.Projetar(domaincaixa.DadosProjecao{
		Hoje:        hoje,
		Ate:         ate,
		Partida:     *partida,
		Entradas:    entradas,
		Saidas:      saidas,
		Recorrentes: recorrentes,
	}), nil
}

func (u *ProjecaoUsecaseImpl) MovimentoDiario(ctx context.Context, input portsin.MovimentoDiarioInput) ([]*domaincaixa.MovimentoDiario, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	filtro := portsout.MovimentoDiarioFiltro{}
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

	filtro.De, filtro.Ate, err = domaincaixa.PeriodoDoMovimentoDiario(domaincaixa.Hoje(time.Now().UTC()), input.De, input.Ate)
	if err != nil {
		return nil, err
	}

	return u.projecoes.MovimentoDiario(ctx, filtro)
}

func (u *ProjecaoUsecaseImpl) recorrentesPrevistas(ctx context.Context, empresaIDs []uuid.UUID, hoje, ate time.Time) ([]domaincaixa.Previsto, error) {
	type candidata struct {
		chave portsout.RecorrenciaNaCompetencia
		item  domaincaixa.Previsto
	}

	var candidatas []candidata
	for competencia := domaincaixa.CompetenciaDe(hoje).AddDate(0, -1, 0); !competencia.After(ate); competencia = competencia.AddDate(0, 1, 0) {
		daCompetencia := domainsolicitacao.CompetenciaDe(competencia)
		vigentes, err := u.recorrencias.ListarVigentes(ctx, daCompetencia, empresaIDs)
		if err != nil {
			return nil, err
		}

		for _, despesa := range vigentes {
			if !despesa.VigenteEm(daCompetencia) {
				continue
			}
			vencimento := despesa.VencimentoNa(daCompetencia).Data()
			if vencimento.After(ate) {
				continue
			}
			candidatas = append(candidatas, candidata{
				chave: portsout.RecorrenciaNaCompetencia{RecorrenciaID: despesa.ID, Competencia: competencia},
				item:  domaincaixa.Previsto{Data: vencimento, ValorCentavos: despesa.Valor.Centavos(), Estimado: despesa.Estimado},
			})
		}
	}
	if len(candidatas) == 0 {
		return nil, nil
	}

	chaves := make([]portsout.RecorrenciaNaCompetencia, 0, len(candidatas))
	for _, item := range candidatas {
		chaves = append(chaves, item.chave)
	}
	geradas, err := u.projecoes.RecorrenciasGeradas(ctx, chaves)
	if err != nil {
		return nil, err
	}

	previstas := make([]domaincaixa.Previsto, 0, len(candidatas))
	for _, item := range candidatas {
		if !geradas[item.chave] {
			previstas = append(previstas, item.item)
		}
	}

	return previstas, nil
}
