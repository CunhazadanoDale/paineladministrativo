package receber

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
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

func escreverLista[T any, R any](w http.ResponseWriter, itens []T, err error, converter func([]T) []R) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]R]{Dados: converter(itens)})
}

func consultaData(r *http.Request, nome string) (*time.Time, error) {
	data, err := receberdto.ParaData(resposta.ConsultaTexto(r, nome), "parâmetro "+nome)
	if err != nil || data.IsZero() {
		return nil, err
	}

	return &data, nil
}

func consultaVerdadeira(r *http.Request, nome string) (bool, error) {
	valor, err := resposta.ConsultaBooleanaOpcional(r, nome)
	if err != nil || valor == nil {
		return false, err
	}

	return *valor, nil
}

func consultaDias(r *http.Request) (*int, error) {
	valor := resposta.ConsultaTexto(r, "dias")
	if valor == "" {
		return nil, nil
	}

	dias, err := strconv.Atoi(valor)
	if err != nil {
		return nil, domain.ErroValidacao("parâmetro dias deve ser um número inteiro")
	}

	return &dias, nil
}
