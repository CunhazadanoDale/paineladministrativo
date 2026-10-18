package receber

import (
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type TotalResumoResponse struct {
	Quantidade    int   `json:"quantidade"`
	SaldoCentavos int64 `json:"saldo_centavos"`
}

type LinhaResumoResponse struct {
	Semana     string              `json:"semana"`
	EmpresaID  uuid.UUID           `json:"empresa_id"`
	AReceber   TotalResumoResponse `json:"a_receber"`
	Antecipado TotalResumoResponse `json:"antecipado"`
	Vencido    TotalResumoResponse `json:"vencido"`
}

type AtrasoResponse struct {
	EmpresaID uuid.UUID           `json:"empresa_id"`
	Vencido   TotalResumoResponse `json:"vencido"`
}

type TotaisResumoResponse struct {
	AReceber   TotalResumoResponse `json:"a_receber"`
	Antecipado TotalResumoResponse `json:"antecipado"`
	Vencido    TotalResumoResponse `json:"vencido"`
}

type ResumoContasAReceberResponse struct {
	De        string                `json:"de"`
	Ate       string                `json:"ate"`
	Semanas   []LinhaResumoResponse `json:"semanas"`
	Atrasados []AtrasoResponse      `json:"atrasados"`
	Totais    TotaisResumoResponse  `json:"totais"`
}

func novoTotal(total domainreceber.TotalResumo) TotalResumoResponse {
	return TotalResumoResponse{Quantidade: total.Quantidade, SaldoCentavos: total.SaldoCentavos}
}

func NovoResumoContasAReceberResponse(resumo domainreceber.ResumoContasAReceber) ResumoContasAReceberResponse {
	resposta := ResumoContasAReceberResponse{
		De:        formatarData(resumo.De),
		Ate:       formatarData(resumo.Ate),
		Semanas:   make([]LinhaResumoResponse, 0, len(resumo.Semanas)),
		Atrasados: make([]AtrasoResponse, 0, len(resumo.Atrasados)),
		Totais: TotaisResumoResponse{
			AReceber:   novoTotal(resumo.Totais.AReceber),
			Antecipado: novoTotal(resumo.Totais.Antecipado),
			Vencido:    novoTotal(resumo.Totais.Vencido),
		},
	}
	for _, linha := range resumo.Semanas {
		resposta.Semanas = append(resposta.Semanas, LinhaResumoResponse{
			Semana:     formatarData(linha.Semana),
			EmpresaID:  linha.EmpresaID,
			AReceber:   novoTotal(linha.AReceber),
			Antecipado: novoTotal(linha.Antecipado),
			Vencido:    novoTotal(linha.Vencido),
		})
	}
	for _, atraso := range resumo.Atrasados {
		resposta.Atrasados = append(resposta.Atrasados, AtrasoResponse{EmpresaID: atraso.EmpresaID, Vencido: novoTotal(atraso.Vencido)})
	}

	return resposta
}
