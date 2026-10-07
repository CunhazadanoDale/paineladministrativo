package solicitacao

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type aprovadorUseCaseFalso struct {
	erro        error
	aprovador   *domainsolicitacao.Aprovador
	itens       []*domainsolicitacao.Aprovador
	designado   uuid.UUID
	removidoID  uuid.UUID
	inputListar domain.PaginacaoFiltro
}

func (f *aprovadorUseCaseFalso) Designar(_ context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	if f.erro != nil {
		return uuid.Nil, f.erro
	}
	f.designado = usuarioID

	return f.aprovador.ID, nil
}

func (f *aprovadorUseCaseFalso) Obter(_ context.Context, _ uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.aprovador, nil
}

func (f *aprovadorUseCaseFalso) Remover(_ context.Context, id uuid.UUID) error {
	f.removidoID = id

	return f.erro
}

func (f *aprovadorUseCaseFalso) Listar(_ context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = filtro

	return f.itens, nil
}

func (f *aprovadorUseCaseFalso) EhDesignado(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.erro == nil, f.erro
}

var _ portsin.AprovadorUseCase = (*aprovadorUseCaseFalso)(nil)

func novoAprovadorDeTeste() *domainsolicitacao.Aprovador {
	return &domainsolicitacao.Aprovador{
		ID:        uuid.New(),
		UsuarioID: uuid.New(),
		CriadoEm:  time.Now().UTC(),
	}
}

func TestDesignarAprovadorValidoRespondeCreated(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{aprovador: novoAprovadorDeTeste()}
	usuarioID := uuid.New()

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Designar, http.MethodPost,
		"/api/v1/aprovadores", "", `{"usuario_id":"`+usuarioID.String()+`"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.designado != usuarioID {
		t.Errorf("usuário designado = %s, esperado %s", usecase.designado, usuarioID)
	}
}

func TestDesignarAprovadorRepetidoResponde409(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{
		erro:      domain.ErroConflito("usuário já designado como aprovador"),
		aprovador: novoAprovadorDeTeste(),
	}

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Designar, http.MethodPost,
		"/api/v1/aprovadores", "", `{"usuario_id":"`+uuid.NewString()+`"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "usuário já designado como aprovador" {
		t.Errorf("mensagem %q sem o detalhe do conflito", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestDesignarAprovadorSemPermissaoResponde403(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{
		erro:      domain.ErroPermissao("perfil sem permissão para designar aprovadores"),
		aprovador: novoAprovadorDeTeste(),
	}

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Designar, http.MethodPost,
		"/api/v1/aprovadores", "", `{"usuario_id":"`+uuid.NewString()+`"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
}

func TestDesignarAprovadorComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{aprovador: novoAprovadorDeTeste()}

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Designar, http.MethodPost,
		"/api/v1/aprovadores", "", `{"usuario_id":"nao-e-uuid"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.designado != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestListarAprovadoresRepassaPaginacao(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{aprovador: novoAprovadorDeTeste()}

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Listar, http.MethodGet,
		"/api/v1/aprovadores?pagina=3&tamanho=50", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.Page != 3 || usecase.inputListar.Size != 50 {
		t.Errorf("paginação = %+v, esperada {pagina: 3, tamanho: 50}", usecase.inputListar)
	}
}

func TestRemoverAprovadorInexistenteResponde404(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{
		erro:      domain.ErroNaoEncontrado("aprovador não encontrado"),
		aprovador: novoAprovadorDeTeste(),
	}
	id := uuid.New()

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Remover, http.MethodDelete,
		"/api/v1/aprovadores/"+id.String(), id.String(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusNotFound)
	if usecase.removidoID != id {
		t.Errorf("id removido = %s, esperado %s", usecase.removidoID, id)
	}
}

func TestRemoverAprovadorValidoResponde204(t *testing.T) {
	usecase := &aprovadorUseCaseFalso{aprovador: novoAprovadorDeTeste()}
	id := uuid.New()

	resposta := apoioteste.ExecutarHandler(NewAprovadorHandler(usecase).Remover, http.MethodDelete,
		"/api/v1/aprovadores/"+id.String(), id.String(), "")

	if resposta.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d: %s", resposta.Code, http.StatusNoContent, resposta.Body.String())
	}
}
