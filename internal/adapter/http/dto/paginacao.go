package dto

import (
	"net/url"
	"strconv"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type PaginacaoQuery struct {
	Pagina  int `json:"pagina"`
	Tamanho int `json:"tamanho"`
}

func NovaPaginacaoQuery(valores url.Values) PaginacaoQuery {
	pagina, _ := strconv.Atoi(valores.Get("pagina"))
	tamanho, _ := strconv.Atoi(valores.Get("tamanho"))

	return PaginacaoQuery{Pagina: pagina, Tamanho: tamanho}
}

func (p PaginacaoQuery) ParaFiltro() domain.PaginacaoFiltro {
	return domain.PaginacaoFiltro{Page: p.Pagina, Size: p.Tamanho}
}
