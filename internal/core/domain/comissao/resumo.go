package comissao

import "github.com/google/uuid"

type LinhaResumo struct {
	EmpresaID         uuid.UUID
	RepresentanteID   uuid.UUID
	RepresentanteNome string
	ComissoesCentavos int64
	AjustesCentavos   int64
	EmAbertoCentavos  int64
	FechadoCentavos   int64
	PagoCentavos      int64
}

type ResumoComissoes struct {
	Competencia Competencia
	Linhas      []LinhaResumo
}

func (r ResumoComissoes) TotalCentavos() int64 {
	var total int64
	for _, linha := range r.Linhas {
		total += linha.ComissoesCentavos + linha.AjustesCentavos
	}

	return total
}
