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
	BaixarReserva *domainestoque.Reserva
	Liberar       []*domainestoque.Reserva
	Sair          int
	Reservar      []ReservaPlanejada
}

func (p PlanoDeBaixa) Locais() []uuid.UUID {
	vistos := make(map[uuid.UUID]bool)
	locais := make([]uuid.UUID, 0, 1+len(p.Liberar)+len(p.Reservar))
	incluir := func(id uuid.UUID) {
		if !vistos[id] {
			vistos[id] = true
			locais = append(locais, id)
		}
	}
	if p.BaixarReserva != nil {
		incluir(p.BaixarReserva.LocalID)
	}
	for _, reserva := range p.Liberar {
		incluir(reserva.LocalID)
	}
	for _, reserva := range p.Reservar {
		incluir(reserva.LocalID)
	}

	return locais
}

func PlanejarBaixa(item *EntregaItem, restanteDoItem int, reservas []*domainestoque.Reserva) (PlanoDeBaixa, error) {
	if item == nil || item.Quantidade <= 0 {
		return PlanoDeBaixa{}, domain.ErroValidacao("item da entrega sem quantidade")
	}
	if item.Quantidade > restanteDoItem {
		return PlanoDeBaixa{}, domain.ErroValidacao("quantidade da entrega maior que o restante do item")
	}

	doLocal := make([]*domainestoque.Reserva, 0, len(reservas))
	deOutrosLocais := make([]*domainestoque.Reserva, 0, len(reservas))
	reservadoNoLocal := 0
	for _, reserva := range reservas {
		switch {
		case reserva == nil || !reserva.Ativa():
			return PlanoDeBaixa{}, domain.ErroConflito("reserva do item já encerrada, recarregue e tente novamente")
		case reserva.Origem != domainestoque.OrigemReservaPedidoItem || reserva.OrigemID == nil || *reserva.OrigemID != item.PedidoItemID:
			return PlanoDeBaixa{}, domain.ErroValidacao("reserva de outro item")
		case reserva.LocalID == item.LocalID:
			doLocal = append(doLocal, reserva)
			reservadoNoLocal += reserva.Quantidade
		default:
			deOutrosLocais = append(deOutrosLocais, reserva)
		}
	}

	plano := PlanoDeBaixa{}
	cabeAindaReservado := restanteDoItem - item.Quantidade
	for _, reserva := range deOutrosLocais {
		if reserva.Quantidade <= cabeAindaReservado {
			cabeAindaReservado -= reserva.Quantidade
			continue
		}

		plano.Liberar = append(plano.Liberar, reserva)
		if cabeAindaReservado > 0 {
			plano.Reservar = append(plano.Reservar, ReservaPlanejada{LocalID: reserva.LocalID, Quantidade: cabeAindaReservado})
			cabeAindaReservado = 0
		}
	}

	if len(doLocal) == 1 && reservadoNoLocal == item.Quantidade {
		plano.BaixarReserva = doLocal[0]

		return plano, nil
	}

	plano.Liberar = append(doLocal, plano.Liberar...)
	plano.Sair = item.Quantidade
	if sobra := min(max(reservadoNoLocal-item.Quantidade, 0), cabeAindaReservado); sobra > 0 {
		plano.Reservar = append(plano.Reservar, ReservaPlanejada{LocalID: item.LocalID, Quantidade: sobra})
	}

	return plano, nil
}
