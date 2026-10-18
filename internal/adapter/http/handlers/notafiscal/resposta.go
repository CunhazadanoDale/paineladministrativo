package notafiscal

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const formatoData = "2006-01-02"

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

func consultaInteiro(r *http.Request, nome string) (*int, error) {
	valor := resposta.ConsultaTexto(r, nome)
	if valor == "" {
		return nil, nil
	}

	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 1 {
		return nil, domain.ErroValidacao("parâmetro " + nome + " deve ser um número inteiro positivo")
	}

	return &numero, nil
}

func consultaData(r *http.Request, nome string) (*time.Time, error) {
	valor := resposta.ConsultaTexto(r, nome)
	if valor == "" {
		return nil, nil
	}

	data, err := time.Parse(formatoData, valor)
	if err != nil {
		return nil, domain.ErroValidacao("parâmetro " + nome + " deve estar no formato AAAA-MM-DD")
	}

	return &data, nil
}

func parametroInteiro(r *http.Request, nome string) (int, error) {
	numero, err := strconv.Atoi(strings.TrimSpace(r.PathValue(nome)))
	if err != nil || numero < 1 {
		return 0, domain.ErroValidacao(nome + " deve ser um número inteiro positivo")
	}

	return numero, nil
}
