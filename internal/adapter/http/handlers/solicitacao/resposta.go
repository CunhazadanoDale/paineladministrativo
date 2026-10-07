package solicitacao

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

func responderErro(w http.ResponseWriter, err error) {
	status, mensagem := mapearErro(err)

	dto.EscreverErro(w, status, mensagem)
}

func mapearErro(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidacao):
		return http.StatusBadRequest, detalhe(err, domain.ErrValidacao)
	case errors.Is(err, domain.ErrPermissao):
		return http.StatusForbidden, detalhe(err, domain.ErrPermissao)
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, detalhe(err, domain.ErrNotFound)
	case errors.Is(err, domain.ErrConflito):
		return http.StatusConflict, detalhe(err, domain.ErrConflito)
	default:
		return http.StatusInternalServerError, "erro interno do servidor"
	}
}

func detalhe(err error, sentinela error) string {
	return strings.TrimPrefix(err.Error(), sentinela.Error()+": ")
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

func usuarioDoContexto(w http.ResponseWriter, r *http.Request) (*domainusuarios.Usuario, bool) {
	usuario, ok := middleware.UsuarioDoContexto(r.Context())
	if !ok {
		dto.EscreverErro(w, http.StatusUnauthorized, "token ausente ou inválido")
		return nil, false
	}

	return usuario, true
}

func parametroUUID(w http.ResponseWriter, r *http.Request, nome string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(nome))
	if err != nil {
		responderErro(w, domain.ErroValidacao("parâmetro "+nome+" inválido"))
		return uuid.Nil, false
	}

	return id, true
}

func usuarioEId(w http.ResponseWriter, r *http.Request) (*domainusuarios.Usuario, uuid.UUID, bool) {
	usuario, ok := usuarioDoContexto(w, r)
	if !ok {
		return nil, uuid.Nil, false
	}

	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return nil, uuid.Nil, false
	}

	return usuario, id, true
}

func consultaPaginacao(r *http.Request) domain.PaginacaoFiltro {
	return dto.NovaPaginacaoQuery(r.URL.Query()).ParaFiltro().Normalizada()
}

func consultaTexto(r *http.Request, nome string) string {
	return strings.TrimSpace(r.URL.Query().Get(nome))
}
