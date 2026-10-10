package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.ProdutoUseCase = (*ProdutoUsecaseImpl)(nil)

type ProdutoUsecaseImpl struct {
	produtos   portsout.ProdutoRepository
	categorias portsout.CategoriaRepository
	imagens    portsout.ImagemRepository
	permissoes
}

func NewProdutoUsecase(
	produtos portsout.ProdutoRepository,
	categorias portsout.CategoriaRepository,
	imagens portsout.ImagemRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ProdutoUsecaseImpl {
	return &ProdutoUsecaseImpl{
		produtos:   produtos,
		categorias: categorias,
		imagens:    imagens,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
		},
	}
}

func (u *ProdutoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarProdutoInput) (uuid.UUID, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return uuid.Nil, err
	}

	categoria, err := u.categorias.Obter(ctx, input.CategoriaID)
	if err != nil {
		return uuid.Nil, err
	}
	if categoria == nil {
		return uuid.Nil, domain.ErroNaoEncontrado("categoria do produto não encontrada")
	}

	slug, err := domainestoque.NovoSlug(input.Nome)
	if err != nil {
		return uuid.Nil, err
	}

	existente, err := u.produtos.ObterPorSlug(ctx, slug.Valor())
	if err != nil {
		return uuid.Nil, err
	}
	if existente != nil {
		return uuid.Nil, domain.ErroConflito("já existe um produto com este nome")
	}

	produtoInput, err := montarProdutoInput(categoria, slug, produtoCampos{
		nome:                     input.Nome,
		descricao:                input.Descricao,
		codigo:                   input.Codigo,
		unidadeMedida:            input.UnidadeMedida,
		precoCentavos:            input.PrecoCentavos,
		precoPromocionalCentavos: input.PrecoPromocionalCentavos,
		estoqueMinimo:            input.EstoqueMinimo,
		pesoKg:                   input.PesoKg,
		destaque:                 input.Destaque,
	})
	if err != nil {
		return uuid.Nil, err
	}

	produto, err := domainestoque.NovoProduto(produtoInput)
	if err != nil {
		return uuid.Nil, err
	}

	return u.produtos.Criar(ctx, produto)
}

func (u *ProdutoUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	produto, err := u.produtos.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, domain.ErroNaoEncontrado("produto não encontrado")
	}

	imagens, err := u.imagens.ListarPorProduto(ctx, produto.ID)
	if err != nil {
		return nil, err
	}
	produto.Imagens = imagens

	return produto, nil
}

func (u *ProdutoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarProdutosInput) ([]*domainestoque.Produto, error) {
	return u.produtos.Listar(ctx, portsout.ProdutoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		CategoriaID:     input.CategoriaID,
		Ativo:           input.Ativo,
		Destaque:        input.Destaque,
		EstoqueBaixo:    input.EstoqueBaixo,
		Busca:           input.Busca,
	})
}

func (u *ProdutoUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarProdutoInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	produto, err := u.Obter(ctx, input.ProdutoID)
	if err != nil {
		return err
	}

	categoria, err := u.categorias.Obter(ctx, input.CategoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domain.ErroNaoEncontrado("categoria do produto não encontrada")
	}

	produtoInput, err := montarProdutoInput(categoria, produto.Slug, produtoCampos{
		nome:                     input.Nome,
		descricao:                input.Descricao,
		codigo:                   input.Codigo,
		unidadeMedida:            input.UnidadeMedida,
		precoCentavos:            input.PrecoCentavos,
		precoPromocionalCentavos: input.PrecoPromocionalCentavos,
		estoqueMinimo:            input.EstoqueMinimo,
		pesoKg:                   input.PesoKg,
		destaque:                 input.Destaque,
	})
	if err != nil {
		return err
	}

	if err := produto.AlterarDados(produtoInput, time.Now().UTC()); err != nil {
		return err
	}

	_, err = u.produtos.Atualizar(ctx, produto)

	return err
}

func (u *ProdutoUsecaseImpl) AlternarAtivo(ctx context.Context, input portsin.AlternarAtivoProdutoInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	produto, err := u.Obter(ctx, input.ProdutoID)
	if err != nil {
		return err
	}

	produto.AlternarAtivo(input.Ativo, time.Now().UTC())

	_, err = u.produtos.Atualizar(ctx, produto)

	return err
}

func (u *ProdutoUsecaseImpl) AlternarDestaque(ctx context.Context, input portsin.AlternarDestaqueProdutoInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	produto, err := u.Obter(ctx, input.ProdutoID)
	if err != nil {
		return err
	}

	produto.AlternarDestaque(input.Destaque, time.Now().UTC())

	_, err = u.produtos.Atualizar(ctx, produto)

	return err
}

type produtoCampos struct {
	nome                     string
	descricao                string
	codigo                   *string
	unidadeMedida            string
	precoCentavos            *int64
	precoPromocionalCentavos *int64
	estoqueMinimo            *int
	pesoKg                   *float64
	destaque                 bool
}

func montarProdutoInput(categoria *domainestoque.Categoria, slug domainestoque.Slug, campos produtoCampos) (domainestoque.ProdutoInput, error) {
	unidade, err := domainestoque.NovaUnidadeMedida(campos.unidadeMedida)
	if err != nil {
		return domainestoque.ProdutoInput{}, err
	}

	var preco *domainestoque.Preco
	if campos.precoCentavos != nil {
		valor, err := domainestoque.NovoPreco(*campos.precoCentavos)
		if err != nil {
			return domainestoque.ProdutoInput{}, err
		}
		preco = &valor
	}

	var precoPromocional *domainestoque.Preco
	if campos.precoPromocionalCentavos != nil {
		valor, err := domainestoque.NovoPreco(*campos.precoPromocionalCentavos)
		if err != nil {
			return domainestoque.ProdutoInput{}, err
		}
		precoPromocional = &valor
	}

	var estoqueMinimo *domainestoque.EstoqueMinimo
	if campos.estoqueMinimo != nil {
		valor, err := domainestoque.NovoEstoqueMinimo(*campos.estoqueMinimo)
		if err != nil {
			return domainestoque.ProdutoInput{}, err
		}
		estoqueMinimo = &valor
	}

	var peso *domainestoque.Peso
	if campos.pesoKg != nil {
		valor, err := domainestoque.NovoPeso(*campos.pesoKg)
		if err != nil {
			return domainestoque.ProdutoInput{}, err
		}
		peso = &valor
	}

	return domainestoque.ProdutoInput{
		Categoria:        categoria,
		Nome:             campos.nome,
		Slug:             slug,
		Descricao:        campos.descricao,
		Codigo:           campos.codigo,
		UnidadeMedida:    unidade,
		Preco:            preco,
		PrecoPromocional: precoPromocional,
		EstoqueMinimo:    estoqueMinimo,
		Peso:             peso,
		Destaque:         campos.destaque,
	}, nil
}
