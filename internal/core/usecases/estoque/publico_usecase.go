package estoque

import (
	"context"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.PublicoUseCase = (*PublicoUsecaseImpl)(nil)

type PublicoUsecaseImpl struct {
	categorias portsout.CategoriaRepository
	produtos   portsout.ProdutoRepository
	imagens    portsout.ImagemRepository
}

func NewPublicoUsecase(
	categorias portsout.CategoriaRepository,
	produtos portsout.ProdutoRepository,
	imagens portsout.ImagemRepository,
) *PublicoUsecaseImpl {
	return &PublicoUsecaseImpl{
		categorias: categorias,
		produtos:   produtos,
		imagens:    imagens,
	}
}

func (u *PublicoUsecaseImpl) ListarCategorias(ctx context.Context) ([]*domainestoque.Categoria, error) {
	ativo := true
	categorias := make([]*domainestoque.Categoria, 0)

	for pagina := 1; ; pagina++ {
		lote, err := u.categorias.Listar(ctx, portsout.CategoriaFiltro{
			Ativo:           &ativo,
			PaginacaoFiltro: domain.PaginacaoFiltro{Page: pagina, Size: domain.TamanhoPaginaMaximo},
		})
		if err != nil {
			return nil, err
		}

		categorias = append(categorias, lote...)

		if len(lote) < domain.TamanhoPaginaMaximo {
			return categorias, nil
		}
	}
}

func (u *PublicoUsecaseImpl) ListarProdutos(ctx context.Context, input portsin.ListarProdutosPublicosInput) ([]*domainestoque.Produto, error) {
	ativo := true

	filtro := portsout.ProdutoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Ativo:           &ativo,
		ComSaldo:        true,
	}

	if slug := strings.TrimSpace(input.CategoriaSlug); slug != "" {
		categoria, err := u.categorias.ObterPorSlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if categoria == nil || !categoria.Ativo {
			return nil, domain.ErroNaoEncontrado("categoria não encontrada")
		}

		filtro.CategoriaID = &categoria.ID
	}

	produtos, err := u.produtos.Listar(ctx, filtro)
	if err != nil {
		return nil, err
	}

	if err := u.anexarImagens(ctx, produtos); err != nil {
		return nil, err
	}

	return produtos, nil
}

func (u *PublicoUsecaseImpl) ObterProdutoPorSlug(ctx context.Context, slug string) (*domainestoque.Produto, error) {
	produto, err := u.produtos.ObterPorSlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, err
	}
	if produto == nil || !produto.Ativo || produto.Saldo.Vazio() {
		return nil, domain.ErroNaoEncontrado("produto não encontrado")
	}

	produtos := []*domainestoque.Produto{produto}
	if err := u.anexarImagens(ctx, produtos); err != nil {
		return nil, err
	}

	return produto, nil
}

func (u *PublicoUsecaseImpl) ListarDestaques(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainestoque.Produto, error) {
	ativo := true
	destacado := true

	produtos, err := u.produtos.Listar(ctx, portsout.ProdutoFiltro{
		PaginacaoFiltro: filtro.Normalizada(),
		Ativo:           &ativo,
		Destaque:        &destacado,
		ComSaldo:        true,
	})
	if err != nil {
		return nil, err
	}

	if err := u.anexarImagens(ctx, produtos); err != nil {
		return nil, err
	}

	return produtos, nil
}

func (u *PublicoUsecaseImpl) anexarImagens(ctx context.Context, produtos []*domainestoque.Produto) error {
	if len(produtos) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(produtos))
	for _, produto := range produtos {
		ids = append(ids, produto.ID)
	}

	imagens, err := u.imagens.ListarPorProdutos(ctx, ids)
	if err != nil {
		return err
	}

	porProduto := make(map[uuid.UUID][]*domainestoque.Imagem, len(produtos))
	for _, imagem := range imagens {
		porProduto[imagem.ProdutoID] = append(porProduto[imagem.ProdutoID], imagem)
	}

	for _, produto := range produtos {
		produto.Imagens = porProduto[produto.ID]
	}

	return nil
}
