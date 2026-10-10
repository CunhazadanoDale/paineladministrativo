package solicitacao

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

const (
	campoArquivo          = "arquivo"
	folgaMultipart        = 1 << 20
	memoriaMultipart      = 1 << 20
	multipartDesconhecido = "application/octet-stream"
)

type ArquivoHandler struct {
	usecase portsin.ArquivoUseCase
}

func NewArquivoHandler(usecase portsin.ArquivoUseCase) *ArquivoHandler {
	return &ArquivoHandler{usecase: usecase}
}

func (h *ArquivoHandler) Enviar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(portsin.TamanhoMaximoArquivo+folgaMultipart))

	if err := r.ParseMultipartForm(memoriaMultipart); err != nil {
		var erroTamanho *http.MaxBytesError
		if errors.As(err, &erroTamanho) {
			dto.EscreverErro(w, http.StatusRequestEntityTooLarge, "arquivo excede o limite de 10MB")
			return
		}
		resposta.ResponderErro(w, domain.ErroValidacao("requisição deve conter o campo arquivo em multipart/form-data"))
		return
	}

	conteudo, cabecalho, err := r.FormFile(campoArquivo)
	if err != nil {
		resposta.ResponderErro(w, domain.ErroValidacao("arquivo ausente no corpo da requisição"))
		return
	}
	defer conteudo.Close()

	arquivo, err := h.usecase.Enviar(
		r.Context(),
		usuario.ID,
		cabecalho.Filename,
		tipoDoArquivo(cabecalho),
		cabecalho.Size,
		conteudo,
	)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[solicitacaodto.ArquivoResponse]{
		Dados: solicitacaodto.NovoArquivoResponse(arquivo),
	})
}

func (h *ArquivoHandler) Baixar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	arquivo, conteudo, err := h.usecase.Baixar(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	defer conteudo.Close()

	w.Header().Set("Content-Type", arquivo.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(arquivo.Nome)))
	w.Header().Set("Content-Length", strconv.FormatInt(arquivo.Tamanho, 10))

	if _, err := io.Copy(w, conteudo); err != nil {
		middleware.AnotarErro(w, fmt.Errorf("envio do arquivo interrompido: %w", err))
	}
}

func (h *ArquivoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.ListarPorProprietario(r.Context(), usuario.ID, paginacao)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[solicitacaodto.ArquivoResponse]{
		Dados:   solicitacaodto.NovoArquivoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *ArquivoHandler) Remover(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id, usuario.ID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func tipoDoArquivo(cabecalho *multipart.FileHeader) string {
	tipo := cabecalho.Header.Get("Content-Type")
	if tipo != "" && tipo != multipartDesconhecido {
		return tipo
	}

	if detectado := mime.TypeByExtension(filepath.Ext(cabecalho.Filename)); detectado != "" {
		return detectado
	}

	return multipartDesconhecido
}
