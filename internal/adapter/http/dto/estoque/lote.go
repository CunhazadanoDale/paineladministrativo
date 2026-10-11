package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type LoteResponse struct {
	ID                    uuid.UUID  `json:"id"`
	ProdutoID             uuid.UUID  `json:"produto_id"`
	LocalID               uuid.UUID  `json:"local_id"`
	MovimentoID           uuid.UUID  `json:"movimento_id"`
	LoteOrigemID          *uuid.UUID `json:"lote_origem_id"`
	CustoUnitarioCentavos *int64     `json:"custo_unitario_centavos"`
	QuantidadeInicial     int        `json:"quantidade_inicial"`
	QuantidadeAtual       int        `json:"quantidade_atual"`
	EntradaEm             time.Time  `json:"entrada_em"`
}

func NovoLoteResponses(itens []*domainestoque.Lote) []LoteResponse {
	respostas := make([]LoteResponse, 0, len(itens))
	for _, item := range itens {
		resposta := LoteResponse{
			ID:                item.ID,
			ProdutoID:         item.ProdutoID,
			LocalID:           item.LocalID,
			MovimentoID:       item.MovimentoID,
			LoteOrigemID:      item.LoteOrigemID,
			QuantidadeInicial: item.QuantidadeInicial,
			QuantidadeAtual:   item.QuantidadeAtual,
			EntradaEm:         item.EntradaEm,
		}
		if item.Custo != nil {
			centavos := item.Custo.Centavos()
			resposta.CustoUnitarioCentavos = &centavos
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}
