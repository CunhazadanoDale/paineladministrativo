package domain


type PaginacaoFiltro struct {
	Page int `json:"pagina"`
	Size int `json:"tamanho"`
}


type PaginacaoResponse[T any] struct {
	Dados []T `json:"dados"`
	Page int `json:"pagina"`
	Size int `json:"tamanho"`
	Total int `json:"total"`
}