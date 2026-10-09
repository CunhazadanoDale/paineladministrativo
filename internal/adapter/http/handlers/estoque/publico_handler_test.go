package estoque

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsinsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type publicoUseCaseFalso struct {
	erro          error
	categorias    []*domainestoque.Categoria
	produtos      []*domainestoque.Produto
	produto       *domainestoque.Produto
	inputProdutos portsin.ListarProdutosPublicosInput
	inputSlug     string
	inputFiltro   domain.PaginacaoFiltro
}

func (f *publicoUseCaseFalso) ListarCategorias(_ context.Context) ([]*domainestoque.Categoria, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.categorias, nil
}

func (f *publicoUseCaseFalso) ListarProdutos(_ context.Context, input portsin.ListarProdutosPublicosInput) ([]*domainestoque.Produto, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	f.inputProdutos = input

	return f.produtos, nil
}

func (f *publicoUseCaseFalso) ObterProdutoPorSlug(_ context.Context, slug string) (*domainestoque.Produto, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	f.inputSlug = slug

	return f.produto, nil
}

func (f *publicoUseCaseFalso) ListarDestaques(_ context.Context, filtro domain.PaginacaoFiltro) ([]*domainestoque.Produto, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	f.inputFiltro = filtro

	return f.produtos, nil
}

var _ portsin.PublicoUseCase = (*publicoUseCaseFalso)(nil)

type arquivoUseCaseFalso struct {
	erro     error
	arquivo  *domainsolicitacao.Arquivo
	conteudo []byte
}

func (f *arquivoUseCaseFalso) Enviar(context.Context, uuid.UUID, string, string, int64, io.Reader) (*domainsolicitacao.Arquivo, error) {
	return nil, nil
}

func (f *arquivoUseCaseFalso) Baixar(context.Context, uuid.UUID, uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	return nil, nil, nil
}

func (f *arquivoUseCaseFalso) BaixarPublico(context.Context, uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	if f.erro != nil {
		return nil, nil, f.erro
	}

	return f.arquivo, io.NopCloser(bytes.NewReader(f.conteudo)), nil
}

func (f *arquivoUseCaseFalso) ListarPorProprietario(context.Context, uuid.UUID, domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	return nil, nil
}

func (f *arquivoUseCaseFalso) Remover(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

var _ portsinsolicitacao.ArquivoUseCase = (*arquivoUseCaseFalso)(nil)

func TestListarCategoriasPublicasMontaArvore(t *testing.T) {
	raiz := novaCategoriaDeTeste(t)

	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("não gerei o slug da subcategoria: %v", err)
	}
	filha, err := domainestoque.NovaCategoria("Cimento", slug, raiz, 2, "")
	if err != nil {
		t.Fatalf("não criei a subcategoria de teste: %v", err)
	}

	usecase := &publicoUseCaseFalso{categorias: []*domainestoque.Categoria{filha, raiz}}

	resposta := executaComParametros(t, NewPublicoHandler(usecase).ListarCategorias,
		http.MethodGet, "/api/v1/publico/categorias", nil, "", false)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}

	var corpo struct {
		Dados []struct {
			Nome   string `json:"nome"`
			Slug   string `json:"slug"`
			Filhas []struct {
				Nome string `json:"nome"`
				Slug string `json:"slug"`
			} `json:"filhas"`
		} `json:"dados"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("não decodifiquei a resposta: %v", err)
	}

	if len(corpo.Dados) != 1 {
		t.Fatalf("categorias raiz = %d, esperado 1", len(corpo.Dados))
	}
	if corpo.Dados[0].Nome != "Alvenaria" || corpo.Dados[0].Slug != "alvenaria" {
		t.Errorf("raiz = %+v, esperado Alvenaria", corpo.Dados[0])
	}
	if len(corpo.Dados[0].Filhas) != 1 || corpo.Dados[0].Filhas[0].Slug != "cimento" {
		t.Errorf("filhas = %+v, esperado a subcategoria Cimento", corpo.Dados[0].Filhas)
	}
}

func TestListarProdutosPublicosRespondePaginadoEEncaminhaFiltro(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	preco, err := domainestoque.NovoPreco(2590)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}
	produto.Preco = &preco
	produto.Imagens = []*domainestoque.Imagem{{
		ID:    uuid.New(),
		Ordem: 0,
		Alt:   "Saco de cimento",
	}}

	usecase := &publicoUseCaseFalso{produtos: []*domainestoque.Produto{produto}}

	resposta := executaComParametros(t, NewPublicoHandler(usecase).ListarProdutos,
		http.MethodGet, "/api/v1/publico/produtos?categoria=alvenaria&pagina=2", nil, "", false)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputProdutos.CategoriaSlug != "alvenaria" {
		t.Errorf("categoria = %q, esperado o slug da query", usecase.inputProdutos.CategoriaSlug)
	}
	if usecase.inputProdutos.Filtro.Page != 2 {
		t.Errorf("página = %d, esperado 2", usecase.inputProdutos.Filtro.Page)
	}

	var corpo struct {
		Dados []struct {
			Nome                     string `json:"nome"`
			Slug                     string `json:"slug"`
			PrecoCentavos            *int64 `json:"preco_centavos"`
			PrecoPromocionalCentavos *int64 `json:"preco_promocional_centavos"`
			Imagens                  []struct {
				ID  uuid.UUID `json:"id"`
				Alt string    `json:"alt"`
			} `json:"imagens"`
		} `json:"dados"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("não decodifiquei a resposta: %v", err)
	}

	if len(corpo.Dados) != 1 {
		t.Fatalf("produtos = %d, esperado 1", len(corpo.Dados))
	}
	if corpo.Dados[0].PrecoCentavos == nil || *corpo.Dados[0].PrecoCentavos != 2590 {
		t.Errorf("preço = %+v, esperado 2590 centavos", corpo.Dados[0].PrecoCentavos)
	}
	if corpo.Dados[0].PrecoPromocionalCentavos != nil {
		t.Error("produto sem promoção não deveria ter preço promocional")
	}
	if len(corpo.Dados[0].Imagens) != 1 || corpo.Dados[0].Imagens[0].Alt != "Saco de cimento" {
		t.Errorf("imagens = %+v, esperado a imagem do produto", corpo.Dados[0].Imagens)
	}
}

func TestListarProdutosPublicosComErroRespondeStatusDoErro(t *testing.T) {
	usecase := &publicoUseCaseFalso{erro: domain.ErroNaoEncontrado("categoria não encontrada")}

	resposta := executaComParametros(t, NewPublicoHandler(usecase).ListarProdutos,
		http.MethodGet, "/api/v1/publico/produtos?categoria=nao-existe", nil, "", false)

	if resposta.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", resposta.Code, http.StatusNotFound)
	}
}

func TestListarDestaquesPublicosRecebePaginacao(t *testing.T) {
	usecase := &publicoUseCaseFalso{produtos: []*domainestoque.Produto{novoProdutoDeTeste(t)}}

	resposta := executaComParametros(t, NewPublicoHandler(usecase).ListarDestaques,
		http.MethodGet, "/api/v1/publico/destaques?tamanho=5", nil, "", false)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputFiltro.Size != 5 {
		t.Errorf("tamanho = %d, esperado 5", usecase.inputFiltro.Size)
	}
}

func TestObterProdutoPublicoPorSlug(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	usecase := &publicoUseCaseFalso{produto: produto}

	resposta := executaComParametros(t, NewPublicoHandler(usecase).ObterProduto,
		http.MethodGet, "/api/v1/publico/produtos/"+produto.Slug.Valor(),
		map[string]string{"slug": produto.Slug.Valor()}, "", false)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputSlug != produto.Slug.Valor() {
		t.Errorf("slug = %q, esperado o da rota", usecase.inputSlug)
	}

	respostaVazia := executaComParametros(t, NewPublicoHandler(usecase).ObterProduto,
		http.MethodGet, "/api/v1/publico/produtos/", map[string]string{"slug": "  "}, "", false)

	if respostaVazia.Code != http.StatusBadRequest {
		t.Errorf("slug vazio = %d, esperado %d", respostaVazia.Code, http.StatusBadRequest)
	}
}

func TestServirImagemPublicaDevolveConteudo(t *testing.T) {
	imagemID := uuid.New()
	imagens := &imagemUseCaseFalso{publica: &domainestoque.Imagem{
		ID:        imagemID,
		ProdutoID: uuid.New(),
		ArquivoID: uuid.New(),
	}}
	arquivos := &arquivoUseCaseFalso{
		arquivo: &domainsolicitacao.Arquivo{
			ID:          uuid.New(),
			ContentType: "image/png",
			Tamanho:     4,
		},
		conteudo: []byte{0x89, 0x50, 0x4E, 0x47},
	}

	resposta := executaComParametros(t, NewImagemPublicaHandler(imagens, arquivos).Servir,
		http.MethodGet, "/api/v1/publico/imagens/"+imagemID.String(),
		map[string]string{"id": imagemID.String()}, "", false)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if conteudo := resposta.Header().Get("Content-Type"); conteudo != "image/png" {
		t.Errorf("content-type %q, esperado image/png", conteudo)
	}
	if !bytes.Equal(resposta.Body.Bytes(), arquivos.conteudo) {
		t.Errorf("corpo = %v, esperado os bytes da imagem", resposta.Body.Bytes())
	}
}

func TestServirImagemPublicaDeProdutoInativoResponde404(t *testing.T) {
	imagemID := uuid.New()
	imagens := &imagemUseCaseFalso{erro: domain.ErroNaoEncontrado("imagem não encontrada")}
	arquivos := &arquivoUseCaseFalso{}

	resposta := executaComParametros(t, NewImagemPublicaHandler(imagens, arquivos).Servir,
		http.MethodGet, "/api/v1/publico/imagens/"+imagemID.String(),
		map[string]string{"id": imagemID.String()}, "", false)

	if resposta.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", resposta.Code, http.StatusNotFound)
	}
}
