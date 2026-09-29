package dto

type Resposta[T any] struct {
	Dados T `json:"dados"`
}

type Paginado[T any] struct {
	Dados   []T `json:"dados"`
	Pagina  int `json:"pagina"`
	Tamanho int `json:"tamanho"`
}

type ErroInterno struct {
	Codigo   int    `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

type RespostaErro struct {
	Erro ErroInterno `json:"erro"`
}
