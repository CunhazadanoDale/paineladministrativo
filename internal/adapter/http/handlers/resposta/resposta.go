package resposta

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

func MapearErro(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidacao):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrPermissao):
		return http.StatusForbidden, detalhe(err, domain.ErrPermissao)
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrConflito):
		return http.StatusConflict, detalhe(err, domain.ErrConflito)
	default:
		return http.StatusInternalServerError, "erro interno do servidor"
	}
}

func detalhe(err error, sentinela error) string {
	return strings.TrimPrefix(err.Error(), sentinela.Error()+": ")
}

func ResponderErro(w http.ResponseWriter, err error) {
	status, mensagem := MapearErro(err)

	dto.EscreverErro(w, status, mensagem)
}

func CorpoJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	defer r.Body.Close()

	decodificador := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(destino); err != nil {
		ResponderErro(w, domain.ErroValidacao("corpo da requisição inválido: "+err.Error()))
		return false
	}

	return true
}

func ParametroUUID(w http.ResponseWriter, r *http.Request, nome string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(nome))
	if err != nil {
		ResponderErro(w, domain.ErroValidacao("parâmetro "+nome+" inválido"))
		return uuid.Nil, false
	}

	return id, true
}

func UsuarioDoContexto(w http.ResponseWriter, r *http.Request) (*domainusuarios.Usuario, bool) {
	usuario, ok := middleware.UsuarioDoContexto(r.Context())
	if !ok {
		dto.EscreverErro(w, http.StatusUnauthorized, "token ausente ou inválido")
		return nil, false
	}

	return usuario, true
}

func UsuarioEId(w http.ResponseWriter, r *http.Request) (*domainusuarios.Usuario, uuid.UUID, bool) {
	usuario, ok := UsuarioDoContexto(w, r)
	if !ok {
		return nil, uuid.Nil, false
	}

	id, ok := ParametroUUID(w, r, "id")
	if !ok {
		return nil, uuid.Nil, false
	}

	return usuario, id, true
}

func ConsultaBooleana(r *http.Request, nome string) bool {
	valor := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(nome)))

	return valor == "true" || valor == "1"
}

func ConsultaBooleanaOpcional(r *http.Request, nome string) (*bool, error) {
	valor := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(nome)))

	switch valor {
	case "":
		return nil, nil
	case "true", "1":
		verdadeiro := true
		return &verdadeiro, nil
	case "false", "0":
		falso := false
		return &falso, nil
	default:
		return nil, domain.ErroValidacao("consulta " + nome + " deve ser true ou false")
	}
}

func ConsultaUUID(r *http.Request, nome string) (*uuid.UUID, error) {
	valor := strings.TrimSpace(r.URL.Query().Get(nome))
	if valor == "" {
		return nil, nil
	}

	id, err := uuid.Parse(valor)
	if err != nil {
		return nil, domain.ErroValidacao("consulta " + nome + " inválida")
	}

	return &id, nil
}

func ConsultaPaginacao(r *http.Request) domain.PaginacaoFiltro {
	return dto.NovaPaginacaoQuery(r.URL.Query()).ParaFiltro().Normalizada()
}

func ConsultaTexto(r *http.Request, nome string) string {
	return strings.TrimSpace(r.URL.Query().Get(nome))
}
