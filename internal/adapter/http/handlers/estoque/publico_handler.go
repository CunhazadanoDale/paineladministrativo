package estoque

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsinsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

type PublicoHandler struct {
	usecase portsin.PublicoUseCase
}

func NewPublicoHandler(usecase portsin.PublicoUseCase) *PublicoHandler {
	return &PublicoHandler{usecase: usecase}
}

type ImagemPublicaHandler struct {
	imagens  portsin.ImagemUseCase
	arquivos portsinsolicitacao.ArquivoUseCase
}

func NewImagemPublicaHandler(
	imagens portsin.ImagemUseCase,
	arquivos portsinsolicitacao.ArquivoUseCase,
) *ImagemPublicaHandler {
	return &ImagemPublicaHandler{
		imagens:  imagens,
		arquivos: arquivos,
	}
}

func (h *PublicoHandler) ListarCategorias(w http.ResponseWriter, r *http.Request) {
	categorias, err := h.usecase.ListarCategorias(r.Context())
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]estoquedto.CategoriaPublicaResponse]{
		Dados: estoquedto.NovaCategoriaPublicaResponses(categorias),
	})
}

func (h *PublicoHandler) ListarProdutos(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	produtos, err := h.usecase.ListarProdutos(r.Context(), portsin.ListarProdutosPublicosInput{
		CategoriaSlug: resposta.ConsultaTexto(r, "categoria"),
		Filtro:        paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.ProdutoPublicoResponse]{
		Dados:   estoquedto.NovaProdutoPublicoResponses(produtos),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *PublicoHandler) ListarDestaques(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	produtos, err := h.usecase.ListarDestaques(r.Context(), paginacao)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.ProdutoPublicoResponse]{
		Dados:   estoquedto.NovaProdutoPublicoResponses(produtos),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *PublicoHandler) ObterProduto(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		resposta.ResponderErro(w, domain.ErroValidacao("slug do produto é obrigatório"))
		return
	}

	produto, err := h.usecase.ObterProdutoPorSlug(r.Context(), slug)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.ProdutoPublicoResponse]{
		Dados: estoquedto.NovaProdutoPublicoResponse(produto),
	})
}

func (h *ImagemPublicaHandler) Servir(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	imagem, err := h.imagens.ObterPublica(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	arquivo, conteudo, err := h.arquivos.BaixarPublico(r.Context(), imagem.ArquivoID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	defer conteudo.Close()

	w.Header().Set("Content-Type", arquivo.ContentType)

	if _, err := io.Copy(w, conteudo); err != nil {
		middleware.AnotarErro(w, fmt.Errorf("envio da imagem interrompido: %w", err))
	}
}
