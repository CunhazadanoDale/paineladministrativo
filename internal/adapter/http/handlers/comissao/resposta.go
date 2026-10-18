package comissao

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func escreverItem[T any, R any](w http.ResponseWriter, status int, item T, err error, converter func(T) R) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[R]{Dados: converter(item)})
}

func escreverPagina[T any, R any](w http.ResponseWriter, paginacao domain.PaginacaoFiltro, itens []T, err error, converter func([]T) []R) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[R]{
		Dados:   converter(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}
