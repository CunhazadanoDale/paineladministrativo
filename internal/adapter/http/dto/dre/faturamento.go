package dre

import (
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type TotalFaturamentoResponse struct {
	Quantidade          int   `json:"quantidade"`
	ValorCentavos       int64 `json:"valor_centavos"`
	TicketMedioCentavos int64 `json:"ticket_medio_centavos"`
}

type FaturamentoEmpresaResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	TotalFaturamentoResponse
}

type MesFaturamentoResponse struct {
	Mes         string                       `json:"mes"`
	Empresas    []FaturamentoEmpresaResponse `json:"empresas"`
	Consolidado TotalFaturamentoResponse     `json:"consolidado"`
}

type ResumoFaturamentoResponse struct {
	De          string                       `json:"de"`
	Ate         string                       `json:"ate"`
	Meses       []MesFaturamentoResponse     `json:"meses"`
	Empresas    []FaturamentoEmpresaResponse `json:"empresas"`
	Consolidado TotalFaturamentoResponse     `json:"consolidado"`
}

func NovoResumoFaturamentoResponse(resumo domaindre.ResumoFaturamento) ResumoFaturamentoResponse {
	resposta := ResumoFaturamentoResponse{
		De:          resumo.De.String(),
		Ate:         resumo.Ate.String(),
		Meses:       make([]MesFaturamentoResponse, 0, len(resumo.Meses)),
		Empresas:    novosFaturamentosPorEmpresa(resumo.Empresas),
		Consolidado: novoTotalFaturamento(resumo.Consolidado),
	}

	for _, mes := range resumo.Meses {
		resposta.Meses = append(resposta.Meses, MesFaturamentoResponse{
			Mes:         mes.Mes.String(),
			Empresas:    novosFaturamentosPorEmpresa(mes.Empresas),
			Consolidado: novoTotalFaturamento(mes.Consolidado),
		})
	}

	return resposta
}

func novosFaturamentosPorEmpresa(empresas []domaindre.FaturamentoEmpresa) []FaturamentoEmpresaResponse {
	respostas := make([]FaturamentoEmpresaResponse, 0, len(empresas))
	for _, empresa := range empresas {
		respostas = append(respostas, FaturamentoEmpresaResponse{
			EmpresaID:                empresa.EmpresaID,
			TotalFaturamentoResponse: novoTotalFaturamento(empresa.Total),
		})
	}

	return respostas
}

func novoTotalFaturamento(total domaindre.TotalFaturamento) TotalFaturamentoResponse {
	return TotalFaturamentoResponse{
		Quantidade:          total.Quantidade,
		ValorCentavos:       total.ValorCentavos,
		TicketMedioCentavos: total.TicketMedioCentavos(),
	}
}
