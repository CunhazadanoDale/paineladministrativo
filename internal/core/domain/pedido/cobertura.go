package pedido

import (
	"bytes"
	"slices"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type DisponivelNoLocal struct {
	LocalID    uuid.UUID
	Disponivel int
}

type ReservaPlanejada struct {
	LocalID    uuid.UUID
	Quantidade int
}

func DistribuirReserva(falta int, locais []DisponivelNoLocal) []ReservaPlanejada {
	ordenados := slices.Clone(locais)
	slices.SortStableFunc(ordenados, func(a, b DisponivelNoLocal) int {
		if a.Disponivel != b.Disponivel {
			return b.Disponivel - a.Disponivel
		}

		return bytes.Compare(a.LocalID[:], b.LocalID[:])
	})

	planejadas := make([]ReservaPlanejada, 0, 1)
	for _, local := range ordenados {
		if falta <= 0 {
			break
		}
		if local.Disponivel <= 0 {
			continue
		}

		quantidade := min(local.Disponivel, falta)
		planejadas = append(planejadas, ReservaPlanejada{LocalID: local.LocalID, Quantidade: quantidade})
		falta -= quantidade
	}

	return planejadas
}

type PlanoDeBaixa struct {
	BaixarReserva  *domainestoque.Reserva
	Liberar        []*domainestoque.Reserva
	Sair           int
	ReservarDeNovo int
}

func PlanejarBaixa(item *EntregaItem, reservas []*domainestoque.Reserva) (PlanoDeBaixa, error) {
	if item == nil || item.Quantidade <= 0 {
		return PlanoDeBaixa{}, domain.ErroValidacao("item da entrega sem quantidade")
	}

	reservado := 0
	for _, reserva := range reservas {
		switch {
		case reserva == nil || !reserva.Ativa():
			return PlanoDeBaixa{}, domain.ErroConflito("reserva do item já encerrada, recarregue e tente novamente")
		case reserva.Origem != domainestoque.OrigemReservaPedidoItem || reserva.OrigemID == nil || *reserva.OrigemID != item.PedidoItemID:
			return PlanoDeBaixa{}, domain.ErroValidacao("reserva de outro item")
		case reserva.LocalID != item.LocalID:
			return PlanoDeBaixa{}, domain.ErroValidacao("reserva de outro local")
		}
		reservado += reserva.Quantidade
	}

	if len(reservas) == 1 && reservado == item.Quantidade {
		return PlanoDeBaixa{BaixarReserva: reservas[0]}, nil
	}

	return PlanoDeBaixa{
		Liberar:        reservas,
		Sair:           item.Quantidade,
		ReservarDeNovo: max(reservado-item.Quantidade, 0),
	}, nil
}
