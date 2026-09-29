package lead

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

func responderJSON(w http.ResponseWriter, status int, conteudo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if conteudo == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(conteudo)
}

func responderVazio(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

func responderErro(w http.ResponseWriter, err error) {
	status, mensagem := mapearErro(err)

	responderJSON(w, status, dto.RespostaErro{
		Erro: dto.ErroInterno{Codigo: status, Mensagem: mensagem},
	})
}

func mapearErro(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidacao):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, "erro interno do servidor"
	}
}

func corpoJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	defer r.Body.Close()

	decodificador := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(destino); err != nil {
		responderErro(w, domain.ErroValidacao("corpo da requisição inválido: "+err.Error()))
		return false
	}

	return true
}

func parametroUUID(w http.ResponseWriter, r *http.Request, nome string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(nome))
	if err != nil {
		responderErro(w, domain.ErroValidacao("parâmetro "+nome+" inválido"))
		return uuid.Nil, false
	}

	return id, true
}

func consultaPaginacao(r *http.Request) domain.PaginacaoFiltro {
	return dto.NovaPaginacaoQuery(r.URL.Query()).ParaFiltro().Normalizada()
}

func consultaBooleana(r *http.Request, nome string) bool {
	valor := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(nome)))

	return valor == "true" || valor == "1"
}

func consultaTexto(r *http.Request, nome string) string {
	return strings.TrimSpace(r.URL.Query().Get(nome))
}
