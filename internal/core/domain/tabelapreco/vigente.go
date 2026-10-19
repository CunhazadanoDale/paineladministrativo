package tabelapreco

import (
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type OrigemPreco string

const (
	OrigemPrecoTabela   OrigemPreco = "tabela"
	OrigemPrecoProduto  OrigemPreco = "produto"
	OrigemPrecoSemPreco OrigemPreco = "sem_preco"
)

func (o OrigemPreco) String() string {
	return string(o)
}

type PrecoVigente struct {
	ProdutoID     uuid.UUID
	ProdutoNome   string
	ProdutoCodigo *string
	Unidade       string
	Modelo        domainpedido.Modelo
	Data          time.Time
	PrecoCentavos *int64
	Origem        OrigemPreco
	ItemID        *uuid.UUID
	VigenteDe     *time.Time
}

func (p *PrecoVigente) Resolver(precoTabela *int64, itemID *uuid.UUID, vigenteDe *time.Time, precoProduto *int64) {
	p.PrecoCentavos, p.Origem, p.ItemID, p.VigenteDe = nil, OrigemPrecoSemPreco, nil, nil

	switch {
	case precoTabela != nil:
		p.PrecoCentavos, p.Origem, p.ItemID, p.VigenteDe = precoTabela, OrigemPrecoTabela, itemID, vigenteDe
	case precoProduto != nil:
		p.PrecoCentavos, p.Origem = precoProduto, OrigemPrecoProduto
	}
}
