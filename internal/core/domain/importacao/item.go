package importacao

import (
	"math"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const quantidadeMaxima = math.MaxInt32

type Item struct {
	ID                    uuid.UUID
	ImportacaoID          uuid.UUID
	ProdutoID             uuid.UUID
	Quantidade            int
	Valor                 ValorMoeda
	QuantidadeRecebida    *int
	CustoUnitarioCentavos *int64
	MovimentoID           *uuid.UUID
	CriadoEm              time.Time
}

type ItemInput struct {
	ProdutoID            uuid.UUID
	Quantidade           int
	ValorMoedaCentesimos int64
}

func novoItem(importacaoID uuid.UUID, atual *Item, input ItemInput, agora time.Time) (*Item, error) {
	if input.ProdutoID == uuid.Nil {
		return nil, domain.ErroValidacao("produto do item é obrigatório")
	}
	if input.Quantidade <= 0 {
		return nil, domain.ErroValidacao("quantidade do item deve ser maior que zero")
	}
	if input.Quantidade > quantidadeMaxima {
		return nil, domain.ErroValidacao("quantidade do item acima do limite permitido")
	}

	valor, err := NovoValorMoeda(input.ValorMoedaCentesimos)
	if err != nil {
		return nil, err
	}

	item := &Item{
		ID:           uuid.Must(uuid.NewV7()),
		ImportacaoID: importacaoID,
		ProdutoID:    input.ProdutoID,
		Quantidade:   input.Quantidade,
		Valor:        valor,
		CriadoEm:     agora,
	}
	if atual != nil {
		item.ID = atual.ID
		item.CriadoEm = atual.CriadoEm
	}

	return item, nil
}
