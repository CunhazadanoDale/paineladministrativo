package pedido

import (
	"math"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const quantidadeMaxima = math.MaxInt32

type Item struct {
	ID                 uuid.UUID
	PedidoID           uuid.UUID
	ProdutoID          uuid.UUID
	Quantidade         int
	PrecoUnitario      Preco
	QuantidadeEntregue int
	CriadoEm           time.Time
}

type ItemInput struct {
	ProdutoID             uuid.UUID
	Quantidade            int
	PrecoUnitarioCentavos int64
}

func novoItem(pedidoID uuid.UUID, input ItemInput, agora time.Time) (*Item, error) {
	if input.ProdutoID == uuid.Nil {
		return nil, domain.ErroValidacao("produto do item é obrigatório")
	}
	if input.Quantidade <= 0 {
		return nil, domain.ErroValidacao("quantidade do item deve ser maior que zero")
	}
	if input.Quantidade > quantidadeMaxima {
		return nil, domain.ErroValidacao("quantidade do item acima do limite permitido")
	}

	preco, err := NovoPreco(input.PrecoUnitarioCentavos)
	if err != nil {
		return nil, err
	}

	return &Item{
		ID:            uuid.New(),
		PedidoID:      pedidoID,
		ProdutoID:     input.ProdutoID,
		Quantidade:    input.Quantidade,
		PrecoUnitario: preco,
		CriadoEm:      agora,
	}, nil
}

func (i *Item) Subtotal() (int64, error) {
	return i.PrecoUnitario.Vezes(i.Quantidade)
}

func (i *Item) Restante() int {
	return i.Quantidade - i.QuantidadeEntregue
}

func (i *Item) Completo() bool {
	return i.Restante() == 0
}

func (i *Item) FaltaCobrir(reservadoAtivo int) int {
	return max(i.Restante()-reservadoAtivo, 0)
}
