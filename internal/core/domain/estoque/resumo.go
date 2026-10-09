package estoque

import (
	"time"

	"github.com/google/uuid"
)

type ResumoProdutos struct {
	TotalProdutos        int64
	ProdutosAtivos       int64
	ValorEstoqueCentavos int64
	ProdutosEstoqueBaixo int64
}

type MovimentoResumo struct {
	ID          uuid.UUID
	ProdutoID   uuid.UUID
	ProdutoNome string
	Tipo        TipoMovimento
	Quantidade  Quantidade
	SaldoApos   Saldo
	CriadoEm    time.Time
}

type ResumoEstoque struct {
	Produtos          ResumoProdutos
	UltimosMovimentos []*MovimentoResumo
}

func ResumoDe(produtos ResumoProdutos, ultimosMovimentos []*MovimentoResumo) ResumoEstoque {
	movimentos := make([]*MovimentoResumo, len(ultimosMovimentos))
	copy(movimentos, ultimosMovimentos)

	return ResumoEstoque{
		Produtos:          produtos,
		UltimosMovimentos: movimentos,
	}
}
