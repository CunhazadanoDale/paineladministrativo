package caixa

import (
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	HorizontePadraoDaProjecao = 30
	HorizonteMaximoDaProjecao = 90
	PeriodoMaximoDoMovimento  = 92
)

type SaldoDePartida struct {
	ExtratoCentavos    int64
	EmTransitoCentavos int64
	ContasConsideradas int
	ContasSemAbertura  int
}

func (s SaldoDePartida) TotalCentavos() int64 {
	return s.ExtratoCentavos + s.EmTransitoCentavos
}

type Previsto struct {
	Data          time.Time
	ValorCentavos int64
	Estimado      bool
}

type DadosProjecao struct {
	Hoje        time.Time
	Ate         time.Time
	Partida     SaldoDePartida
	Entradas    []Previsto
	Saidas      []Previsto
	Recorrentes []Previsto
}

type DiaProjetado struct {
	Data                     time.Time
	EntradasCentavos         int64
	EntradasVencidasCentavos int64
	SaidasCentavos           int64
	SaidasEstimadasCentavos  int64
	SaidasVencidasCentavos   int64
	RecorrentesCentavos      int64
	SaldoCentavos            int64
}

func (d *DiaProjetado) variacao() int64 {
	return d.EntradasCentavos + d.EntradasVencidasCentavos -
		d.SaidasCentavos - d.SaidasEstimadasCentavos - d.SaidasVencidasCentavos - d.RecorrentesCentavos
}

type Projecao struct {
	De      time.Time
	Ate     time.Time
	Partida SaldoDePartida
	Dias    []DiaProjetado
}

func (p *Projecao) SaldoFinalCentavos() int64 {
	if len(p.Dias) == 0 {
		return p.Partida.TotalCentavos()
	}

	return p.Dias[len(p.Dias)-1].SaldoCentavos
}

func HorizonteDaProjecao(hoje time.Time, ate *time.Time) (time.Time, error) {
	if ate == nil {
		return hoje.AddDate(0, 0, HorizontePadraoDaProjecao), nil
	}

	fim := DataDe(*ate)
	if fim.Before(hoje) {
		return time.Time{}, domain.ErroValidacao("a projeção começa hoje; ate não pode ser anterior a hoje")
	}
	if fim.After(hoje.AddDate(0, 0, HorizonteMaximoDaProjecao)) {
		return time.Time{}, domain.ErroValidacao("a projeção vai até " + strconv.Itoa(HorizonteMaximoDaProjecao) + " dias à frente")
	}

	return fim, nil
}

func Projetar(dados DadosProjecao) *Projecao {
	hoje := DataDe(dados.Hoje)
	ate := DataDe(dados.Ate)

	quantidade := int(ate.Sub(hoje).Hours()/24) + 1
	dias := make([]DiaProjetado, quantidade)
	for i := range dias {
		dias[i].Data = hoje.AddDate(0, 0, i)
	}

	indice := func(data time.Time) (int, bool) {
		data = DataDe(data)
		if data.After(ate) {
			return 0, false
		}
		if data.Before(hoje) {
			return -1, true
		}

		return int(data.Sub(hoje).Hours() / 24), true
	}

	for _, entrada := range dados.Entradas {
		posicao, ok := indice(entrada.Data)
		switch {
		case !ok:
		case posicao < 0:
			dias[0].EntradasVencidasCentavos += entrada.ValorCentavos
		default:
			dias[posicao].EntradasCentavos += entrada.ValorCentavos
		}
	}
	for _, saida := range dados.Saidas {
		posicao, ok := indice(saida.Data)
		switch {
		case !ok:
		case posicao < 0:
			dias[0].SaidasVencidasCentavos += saida.ValorCentavos
		case saida.Estimado:
			dias[posicao].SaidasEstimadasCentavos += saida.ValorCentavos
		default:
			dias[posicao].SaidasCentavos += saida.ValorCentavos
		}
	}
	for _, recorrente := range dados.Recorrentes {
		posicao, ok := indice(recorrente.Data)
		if !ok {
			continue
		}
		dias[max(posicao, 0)].RecorrentesCentavos += recorrente.ValorCentavos
	}

	saldo := dados.Partida.TotalCentavos()
	for i := range dias {
		saldo += dias[i].variacao()
		dias[i].SaldoCentavos = saldo
	}

	return &Projecao{De: hoje, Ate: ate, Partida: dados.Partida, Dias: dias}
}

type MovimentoDiario struct {
	Data             time.Time
	ContaBancariaID  uuid.UUID
	EntradasCentavos int64
	SaidasCentavos   int64
	SaldoCentavos    int64
}

func PeriodoDoMovimentoDiario(hoje time.Time, de, ate *time.Time) (time.Time, time.Time, error) {
	fim := hoje
	if ate != nil {
		fim = DataDe(*ate)
	}
	inicio := CompetenciaDe(fim)
	if de != nil {
		inicio = DataDe(*de)
	}

	if fim.Before(inicio) {
		return time.Time{}, time.Time{}, domain.ErroValidacao("data final anterior à inicial")
	}
	if fim.Sub(inicio) > PeriodoMaximoDoMovimento*24*time.Hour {
		return time.Time{}, time.Time{}, domain.ErroValidacao("o movimento diário aceita no máximo " + strconv.Itoa(PeriodoMaximoDoMovimento) + " dias por vez")
	}

	return inicio, fim, nil
}
