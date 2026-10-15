package solicitacao

import (
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type PendenciasResponse struct {
	Aprovacao  *int `json:"aprovacao"`
	Financeiro *int `json:"financeiro"`
	Vencidas   *int `json:"vencidas"`
}

func NovaPendenciasResponse(pendencias domainsolicitacao.Pendencias) PendenciasResponse {
	return PendenciasResponse{
		Aprovacao:  pendencias.Aprovacao,
		Financeiro: pendencias.Financeiro,
		Vencidas:   pendencias.Vencidas,
	}
}

type TotalResumoResponse struct {
	Quantidade    int   `json:"quantidade"`
	ValorCentavos int64 `json:"valor_centavos"`
}

type SemanaResumoResponse struct {
	Semana           string              `json:"semana"`
	EmpresaID        uuid.UUID           `json:"empresa_id"`
	Previsto         TotalResumoResponse `json:"previsto"`
	APagar           TotalResumoResponse `json:"a_pagar"`
	Vencido          TotalResumoResponse `json:"vencido"`
	EstimadoCentavos int64               `json:"estimado_centavos"`
}

type AtrasoResumoResponse struct {
	EmpresaID uuid.UUID           `json:"empresa_id"`
	Vencido   TotalResumoResponse `json:"vencido"`
}

type TotaisResumoResponse struct {
	Previsto         TotalResumoResponse `json:"previsto"`
	APagar           TotalResumoResponse `json:"a_pagar"`
	Vencido          TotalResumoResponse `json:"vencido"`
	EstimadoCentavos int64               `json:"estimado_centavos"`
}

type ResumoContasAPagarResponse struct {
	De        string                 `json:"de"`
	Ate       string                 `json:"ate"`
	Semanas   []SemanaResumoResponse `json:"semanas"`
	Atrasadas []AtrasoResumoResponse `json:"atrasadas"`
	Totais    TotaisResumoResponse   `json:"totais"`
}

func NovoResumoContasAPagarResponse(resumo domainsolicitacao.ResumoContasAPagar) ResumoContasAPagarResponse {
	resposta := ResumoContasAPagarResponse{
		De:        resumo.De.Format(formatoDataPagamento),
		Ate:       resumo.Ate.Format(formatoDataPagamento),
		Semanas:   make([]SemanaResumoResponse, 0, len(resumo.Semanas)),
		Atrasadas: make([]AtrasoResumoResponse, 0, len(resumo.Atrasadas)),
		Totais: TotaisResumoResponse{
			Previsto:         novoTotalResumo(resumo.Totais.Previsto),
			APagar:           novoTotalResumo(resumo.Totais.APagar),
			Vencido:          novoTotalResumo(resumo.Totais.Vencido),
			EstimadoCentavos: resumo.Totais.EstimadoCentavos,
		},
	}

	for _, linha := range resumo.Semanas {
		resposta.Semanas = append(resposta.Semanas, SemanaResumoResponse{
			Semana:           linha.Semana.Format(formatoDataPagamento),
			EmpresaID:        linha.EmpresaID,
			Previsto:         novoTotalResumo(linha.Previsto),
			APagar:           novoTotalResumo(linha.APagar),
			Vencido:          novoTotalResumo(linha.Vencido),
			EstimadoCentavos: linha.EstimadoCentavos,
		})
	}
	for _, atraso := range resumo.Atrasadas {
		resposta.Atrasadas = append(resposta.Atrasadas, AtrasoResumoResponse{
			EmpresaID: atraso.EmpresaID,
			Vencido:   novoTotalResumo(atraso.Vencido),
		})
	}

	return resposta
}

func novoTotalResumo(total domainsolicitacao.TotalResumo) TotalResumoResponse {
	return TotalResumoResponse{Quantidade: total.Quantidade, ValorCentavos: total.ValorCentavos}
}
