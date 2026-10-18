package comissao

import (
	"math"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const diaDoPagamento = 15

type Fechamento struct {
	ID                    uuid.UUID
	EmpresaID             uuid.UUID
	RepresentanteID       uuid.UUID
	Competencia           Competencia
	ComissoesCentavos     int64
	AjustesCentavos       int64
	TotalCentavos         int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
	SolicitacaoID         uuid.UUID
	StatusContaAPagar     string
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type FechamentoComItens struct {
	Fechamento *Fechamento
	Comissoes  []*Comissao
	Ajustes    []*Ajuste
}

type SaldoAberto struct {
	RepresentanteID   uuid.UUID
	ComissoesCentavos int64
	AjustesCentavos   int64
}

type FalhaFechamento struct {
	RepresentanteID uuid.UUID
	Motivo          string
}

type ResultadoFechamento struct {
	Apuracao    ResultadoApuracao
	Fechados    []*Fechamento
	Falhas      []FalhaFechamento
	Competencia Competencia
}

func ConferirCompetenciaDeFechamento(competencia Competencia, agora time.Time) error {
	if !competencia.Inicio().Before(CompetenciaDe(Hoje(agora)).Inicio()) {
		return domain.ErroValidacao("só é possível fechar competência de mês já encerrado")
	}

	return nil
}

func VencimentoDoFechamento(competencia Competencia) time.Time {
	return competencia.Seguinte().Inicio().AddDate(0, 0, diaDoPagamento-1)
}

func NovoFechamento(empresaID uuid.UUID, competencia Competencia, saldo SaldoAberto, categoriaID, usuarioID uuid.UUID, agora time.Time) (*Fechamento, error) {
	switch {
	case empresaID == uuid.Nil:
		return nil, domain.ErroValidacao("empresa do fechamento é obrigatória")
	case saldo.RepresentanteID == uuid.Nil:
		return nil, domain.ErroValidacao("representante do fechamento é obrigatório")
	case categoriaID == uuid.Nil:
		return nil, domain.ErroValidacao("categoria financeira do fechamento é obrigatória")
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário não informado")
	case saldo.ComissoesCentavos < 0:
		return nil, domain.ErroValidacao("soma das comissões do fechamento não pode ser negativa")
	case saldo.AjustesCentavos > 0 && saldo.ComissoesCentavos > math.MaxInt64-saldo.AjustesCentavos:
		return nil, domain.ErroValidacao("total do fechamento excede o limite")
	}

	total := saldo.ComissoesCentavos + saldo.AjustesCentavos
	if total <= 0 {
		return nil, ErroNadaAPagar()
	}

	return &Fechamento{
		ID:                    uuid.New(),
		EmpresaID:             empresaID,
		RepresentanteID:       saldo.RepresentanteID,
		Competencia:           competencia,
		ComissoesCentavos:     saldo.ComissoesCentavos,
		AjustesCentavos:       saldo.AjustesCentavos,
		TotalCentavos:         total,
		Vencimento:            VencimentoDoFechamento(competencia),
		CategoriaFinanceiraID: categoriaID,
		UsuarioID:             usuarioID,
		CriadoEm:              agora,
	}, nil
}
