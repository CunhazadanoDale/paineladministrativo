package solicitacao

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

type arquivoUseCaseFalso struct {
	erroEnviar   error
	erroBaixar   error
	erroRemover  error
	baixado      *domainsolicitacao.Arquivo
	conteudo     []byte
	proprietario uuid.UUID
	nome         string
	contentType  string
	tamanho      int64
}

func (f *arquivoUseCaseFalso) Enviar(_ context.Context, proprietarioID uuid.UUID, nome, contentType string, tamanho int64, _ io.Reader) (*domainsolicitacao.Arquivo, error) {
	if f.erroEnviar != nil {
		return nil, f.erroEnviar
	}

	f.proprietario = proprietarioID
	f.nome = nome
	f.contentType = contentType
	f.tamanho = tamanho

	return &domainsolicitacao.Arquivo{
		ID:             uuid.New(),
		ProprietarioID: proprietarioID,
		Nome:           nome,
		ContentType:    contentType,
		Tamanho:        tamanho,
	}, nil
}

func (f *arquivoUseCaseFalso) Baixar(_ context.Context, _, usuarioID uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	if f.erroBaixar != nil {
		return nil, nil, f.erroBaixar
	}
	if f.baixado == nil || f.baixado.ProprietarioID != usuarioID {
		return nil, nil, domain.ErroPermissao("perfil sem permissão para baixar este arquivo")
	}

	return f.baixado, io.NopCloser(bytes.NewReader(f.conteudo)), nil
}

func (f *arquivoUseCaseFalso) BaixarPublico(_ context.Context, _ uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	if f.erroBaixar != nil {
		return nil, nil, f.erroBaixar
	}
	if f.baixado == nil {
		return nil, nil, domain.ErroValidacao("arquivo não encontrado")
	}

	return f.baixado, io.NopCloser(bytes.NewReader(f.conteudo)), nil
}

func (f *arquivoUseCaseFalso) ListarPorProprietario(_ context.Context, _ uuid.UUID, _ domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	return nil, nil
}

func (f *arquivoUseCaseFalso) Remover(_ context.Context, _, _ uuid.UUID) error {
	return f.erroRemover
}

var _ portsin.ArquivoUseCase = (*arquivoUseCaseFalso)(nil)

type tokensFalso struct {
	usuarioID uuid.UUID
}

func (t *tokensFalso) Gerar(uuid.UUID, int) (string, time.Time, error) {
	return "token", time.Now().UTC().Add(time.Hour), nil
}

func (t *tokensFalso) Validar(string) (uuid.UUID, int, error) {
	return t.usuarioID, 0, nil
}

var _ portsinautenticacao.TokenService = (*tokensFalso)(nil)

type usuariosFalso struct {
	portsinusuarios.UsuarioUseCase
	usuario *domainusuarios.Usuario
}

func (u *usuariosFalso) GetByID(context.Context, uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.usuario, nil
}

var _ portsinusuarios.UsuarioUseCase = (*usuariosFalso)(nil)

func corpoMultipart(t *testing.T, nomeArquivo string, conteudo []byte) (*bytes.Buffer, string) {
	t.Helper()

	corpo := &bytes.Buffer{}
	escritor := multipart.NewWriter(corpo)

	parte, err := escritor.CreateFormFile("arquivo", nomeArquivo)
	if err != nil {
		t.Fatalf("não montei a parte do arquivo: %v", err)
	}
	if _, err := parte.Write(conteudo); err != nil {
		t.Fatalf("não escrevi o arquivo: %v", err)
	}
	if err := escritor.Close(); err != nil {
		t.Fatalf("não fechei o multipart: %v", err)
	}

	return corpo, escritor.FormDataContentType()
}

func TestEnviarArquivoSemMultipartRecebeErroDeValidacao(t *testing.T) {
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}

	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/arquivos", strings.NewReader(`{"algo":1}`))
	requisicao.Header.Set("Content-Type", "application/json")

	registrador := httptest.NewRecorder()
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, NewArquivoHandler(&arquivoUseCaseFalso{}).Enviar)
	protegido(registrador, requisicao)

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusBadRequest, registrador.Body.String())
	}
}

func TestEnviarArquivoValidoRespondeCreated(t *testing.T) {
	usecase := &arquivoUseCaseFalso{}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	conteudo := []byte("%PDF-1.4 conteudo")
	corpo, tipo := corpoMultipart(t, "orcamento.pdf", conteudo)

	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/arquivos", corpo)
	requisicao.Header.Set("Content-Type", tipo)

	registrador := httptest.NewRecorder()
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, NewArquivoHandler(usecase).Enviar)
	protegido(registrador, requisicao)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusCreated, registrador.Body.String())
	}
	if usecase.proprietario != usuario.ID {
		t.Errorf("proprietário = %s, esperado %s", usecase.proprietario, usuario.ID)
	}
	if usecase.nome != "orcamento.pdf" {
		t.Errorf("nome = %q, esperado %q", usecase.nome, "orcamento.pdf")
	}
	if usecase.tamanho != int64(len(conteudo)) {
		t.Errorf("tamanho = %d, esperado %d", usecase.tamanho, len(conteudo))
	}
	if usecase.contentType != "application/pdf" {
		t.Errorf("content type = %q, esperado application/pdf", usecase.contentType)
	}
}

func TestEnviarArquivoGrandeRespondePayloadTooLarge(t *testing.T) {
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	conteudo := bytes.Repeat([]byte("a"), portsin.TamanhoMaximoArquivo+2*1024*1024)
	corpo, tipo := corpoMultipart(t, "grande.pdf", conteudo)

	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/arquivos", corpo)
	requisicao.Header.Set("Content-Type", tipo)

	registrador := httptest.NewRecorder()
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, NewArquivoHandler(&arquivoUseCaseFalso{}).Enviar)
	protegido(registrador, requisicao)

	if registrador.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusRequestEntityTooLarge, registrador.Body.String())
	}
}

func TestBaixarArquivoDevolveConteudoECabecalhos(t *testing.T) {
	conteudo := []byte("%PDF-1.4 anexo")
	usecase := &arquivoUseCaseFalso{
		baixado: &domainsolicitacao.Arquivo{
			ID:             uuid.New(),
			ProprietarioID: uuid.New(),
			Nome:           "comprovante.pdf",
			ContentType:    "application/pdf",
			Tamanho:        int64(len(conteudo)),
		},
		conteudo: conteudo,
	}
	usuario := &domainusuarios.Usuario{ID: usecase.baixado.ProprietarioID, Ativo: true}

	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/arquivos/"+usecase.baixado.ID.String(), nil)
	requisicao.SetPathValue("id", usecase.baixado.ID.String())

	registrador := httptest.NewRecorder()
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, NewArquivoHandler(usecase).Baixar)
	protegido(registrador, requisicao)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}
	if !bytes.Equal(registrador.Body.Bytes(), conteudo) {
		t.Errorf("conteúdo = %q, esperado %q", registrador.Body.Bytes(), conteudo)
	}
	if disponivel := registrador.Header().Get("Content-Disposition"); !strings.Contains(disponivel, "comprovante.pdf") {
		t.Errorf("content-disposition %q sem o nome do arquivo", disponivel)
	}
	if tipo := registrador.Header().Get("Content-Type"); tipo != "application/pdf" {
		t.Errorf("content-type %q, esperado application/pdf", tipo)
	}
}

func TestBaixarArquivoSemPermissaoRespondeProibido(t *testing.T) {
	usecase := &arquivoUseCaseFalso{erroBaixar: domain.ErroPermissao("perfil sem permissão para baixar este arquivo")}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}

	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/arquivos/"+uuid.New().String(), nil)
	requisicao.SetPathValue("id", uuid.New().String())

	registrador := httptest.NewRecorder()
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, NewArquivoHandler(usecase).Baixar)
	protegido(registrador, requisicao)

	if registrador.Code != http.StatusForbidden {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusForbidden, registrador.Body.String())
	}
}
