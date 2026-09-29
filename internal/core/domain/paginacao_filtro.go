package domain

const (
	TamanhoPaginaPadrao = 20
	TamanhoPaginaMaximo = 100
)

type PaginacaoFiltro struct {
	Page int `json:"pagina"`
	Size int `json:"tamanho"`
}

type PaginacaoResponse[T any] struct {
	Dados []T `json:"dados"`
	Page  int `json:"pagina"`
	Size  int `json:"tamanho"`
	Total int `json:"total"`
}

func (p PaginacaoFiltro) Normalizada() PaginacaoFiltro {
	page := p.Page
	if page < 1 {
		page = 1
	}

	size := p.Size
	if size < 1 {
		size = TamanhoPaginaPadrao
	}
	if size > TamanhoPaginaMaximo {
		size = TamanhoPaginaMaximo
	}

	return PaginacaoFiltro{Page: page, Size: size}
}
