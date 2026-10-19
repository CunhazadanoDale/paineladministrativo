package caixa

import (
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type SaldoDePartidaResponse struct {
	ExtratoCentavos    int64 `json:"extrato_centavos"`
	EmTransitoCentavos int64 `json:"em_transito_centavos"`
	TotalCentavos      int64 `json:"total_centavos"`
	ContasConsideradas int   `json:"contas_consideradas"`
	ContasSemAbertura  int   `json:"contas_sem_saldo_inicial"`
}

type DiaProjetadoResponse struct {
	Data                     string `json:"data"`
	EntradasCentavos         int64  `json:"entradas_centavos"`
	EntradasVencidasCentavos int64  `json:"entradas_vencidas_centavos"`
	SaidasCentavos           int64  `json:"saidas_centavos"`
	SaidasEstimadasCentavos  int64  `json:"saidas_estimadas_centavos"`
	SaidasVencidasCentavos   int64  `json:"saidas_vencidas_centavos"`
	RecorrentesCentavos      int64  `json:"recorrentes_centavos"`
	SaldoCentavos            int64  `json:"saldo_centavos"`
}

type ProjecaoResponse struct {
	De                 string                 `json:"de"`
	Ate                string                 `json:"ate"`
	SaldoDePartida     SaldoDePartidaResponse `json:"saldo_de_partida"`
	SaldoFinalCentavos int64                  `json:"saldo_final_centavos"`
	Dias               []DiaProjetadoResponse `json:"dias"`
}

func NovaProjecaoResponse(projecao *domaincaixa.Projecao) ProjecaoResponse {
	resposta := ProjecaoResponse{
		De:  formatarData(projecao.De),
		Ate: formatarData(projecao.Ate),
		SaldoDePartida: SaldoDePartidaResponse{
			ExtratoCentavos:    projecao.Partida.ExtratoCentavos,
			EmTransitoCentavos: projecao.Partida.EmTransitoCentavos,
			TotalCentavos:      projecao.Partida.TotalCentavos(),
			ContasConsideradas: projecao.Partida.ContasConsideradas,
			ContasSemAbertura:  projecao.Partida.ContasSemAbertura,
		},
		SaldoFinalCentavos: projecao.SaldoFinalCentavos(),
		Dias:               make([]DiaProjetadoResponse, 0, len(projecao.Dias)),
	}
	for _, dia := range projecao.Dias {
		resposta.Dias = append(resposta.Dias, DiaProjetadoResponse{
			Data:                     formatarData(dia.Data),
			EntradasCentavos:         dia.EntradasCentavos,
			EntradasVencidasCentavos: dia.EntradasVencidasCentavos,
			SaidasCentavos:           dia.SaidasCentavos,
			SaidasEstimadasCentavos:  dia.SaidasEstimadasCentavos,
			SaidasVencidasCentavos:   dia.SaidasVencidasCentavos,
			RecorrentesCentavos:      dia.RecorrentesCentavos,
			SaldoCentavos:            dia.SaldoCentavos,
		})
	}

	return resposta
}

type MovimentoDiarioResponse struct {
	Data             string    `json:"data"`
	ContaBancariaID  uuid.UUID `json:"conta_bancaria_id"`
	EntradasCentavos int64     `json:"entradas_centavos"`
	SaidasCentavos   int64     `json:"saidas_centavos"`
	SaldoCentavos    int64     `json:"saldo_centavos"`
}

func NovosMovimentoDiarioResponses(movimentos []*domaincaixa.MovimentoDiario) []MovimentoDiarioResponse {
	respostas := make([]MovimentoDiarioResponse, 0, len(movimentos))
	for _, movimento := range movimentos {
		respostas = append(respostas, MovimentoDiarioResponse{
			Data:             formatarData(movimento.Data),
			ContaBancariaID:  movimento.ContaBancariaID,
			EntradasCentavos: movimento.EntradasCentavos,
			SaidasCentavos:   movimento.SaidasCentavos,
			SaldoCentavos:    movimento.SaldoCentavos,
		})
	}

	return respostas
}
