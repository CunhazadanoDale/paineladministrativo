package dto

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

type LinhaRejeitadaResponse struct {
	Linha  int    `json:"linha"`
	Chave  string `json:"chave"`
	Motivo string `json:"motivo"`
}

type RelatorioImportacaoResponse struct {
	Simulacao  bool                     `json:"simulacao"`
	Total      int                      `json:"total"`
	Importadas int                      `json:"importadas"`
	Rejeitadas []LinhaRejeitadaResponse `json:"rejeitadas"`
}

func NovoRelatorioImportacaoResponse(relatorio *domain.RelatorioImportacao) RelatorioImportacaoResponse {
	rejeitadas := make([]LinhaRejeitadaResponse, 0, len(relatorio.Rejeitadas))
	for _, rejeitada := range relatorio.Rejeitadas {
		rejeitadas = append(rejeitadas, LinhaRejeitadaResponse(rejeitada))
	}

	return RelatorioImportacaoResponse{
		Simulacao:  relatorio.Simulacao,
		Total:      relatorio.Total,
		Importadas: relatorio.Importadas,
		Rejeitadas: rejeitadas,
	}
}
