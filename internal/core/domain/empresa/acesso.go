package empresa

import (
	"slices"

	"github.com/google/uuid"
)

type Acesso struct {
	todas    bool
	empresas map[uuid.UUID]struct{}
}

func AcessoTotal() Acesso {
	return Acesso{todas: true}
}

func AcessoRestrito(empresaIDs []uuid.UUID) Acesso {
	empresas := make(map[uuid.UUID]struct{}, len(empresaIDs))
	for _, id := range empresaIDs {
		empresas[id] = struct{}{}
	}

	return Acesso{empresas: empresas}
}

func (a Acesso) Todas() bool {
	return a.todas
}

func (a Acesso) Permite(empresaID uuid.UUID) bool {
	if a.todas {
		return true
	}

	_, ok := a.empresas[empresaID]

	return ok
}

func (a Acesso) EmpresaIDs() []uuid.UUID {
	if a.todas {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(a.empresas))
	for id := range a.empresas {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return slices.Compare(a[:], b[:])
	})

	return ids
}
