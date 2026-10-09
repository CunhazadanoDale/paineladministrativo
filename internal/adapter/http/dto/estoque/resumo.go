package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ResumoResponse struct {
	TotalProdutos        int64                     `json:"total_produtos"`
	ProdutosAtivos       int64                     `json:"produtos_ativos"`
	ValorEstoqueCentavos int64                     `json:"valor_estoque_centavos"`
	ProdutosEstoqueBaixo int64                     `json:"produtos_estoque_baixo"`
	UltimosMovimentos    []MovimentoResumoResponse `json:"ultimos_movimentos"`
}

func NovoResumoResponse(resumo *domainestoque.ResumoEstoque) ResumoResponse {
	return ResumoResponse{
		TotalProdutos:        resumo.Produtos.TotalProdutos,
		ProdutosAtivos:       resumo.Produtos.ProdutosAtivos,
		ValorEstoqueCentavos: resumo.Produtos.ValorEstoqueCentavos,
		ProdutosEstoqueBaixo: resumo.Produtos.ProdutosEstoqueBaixo,
		UltimosMovimentos:    NovoMovimentoResumoResponses(resumo.UltimosMovimentos),
	}
}

type MovimentoResumoResponse struct {
	ID          uuid.UUID `json:"id"`
	ProdutoID   uuid.UUID `json:"produto_id"`
	ProdutoNome string    `json:"produto_nome"`
	Tipo        string    `json:"tipo"`
	Quantidade  int       `json:"quantidade"`
	SaldoApos   int       `json:"saldo_apos"`
	CriadoEm    time.Time `json:"criado_em"`
}

func NovoMovimentoResumoResponses(itens []*domainestoque.MovimentoResumo) []MovimentoResumoResponse {
	respostas := make([]MovimentoResumoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, MovimentoResumoResponse{
			ID:          item.ID,
			ProdutoID:   item.ProdutoID,
			ProdutoNome: item.ProdutoNome,
			Tipo:        item.Tipo.String(),
			Quantidade:  item.Quantidade.Valor(),
			SaldoApos:   item.SaldoApos.Quantidade(),
			CriadoEm:    item.CriadoEm,
		})
	}

	return respostas
}
