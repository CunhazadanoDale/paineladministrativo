package estoque

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.ImportacaoProdutoUseCase = (*ImportacaoProdutoUsecaseImpl)(nil)

type ImportacaoProdutoUsecaseImpl struct {
	produtos   portsout.ProdutoImportacaoRepository
	categorias portsout.CategoriaPorSlugRepository
	permissoes
}

func NewImportacaoProdutoUsecase(
	produtos portsout.ProdutoImportacaoRepository,
	categorias portsout.CategoriaPorSlugRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ImportacaoProdutoUsecaseImpl {
	return &ImportacaoProdutoUsecaseImpl{
		produtos:   produtos,
		categorias: categorias,
		permissoes: permissoes{usuarios: usuarios, cargos: cargos},
	}
}

type importacaoDeProdutos struct {
	simular            bool
	categorias         map[string]*domainestoque.Categoria
	codigosCadastrados map[string]uuid.UUID
	nomesDoArquivo     map[string]int
	codigosDoArquivo   map[string]int
}

func (u *ImportacaoProdutoUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarProdutosInput) (*domain.RelatorioImportacao, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}
	relatorio, err := domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
	if err != nil {
		return nil, err
	}

	estado := &importacaoDeProdutos{
		simular:          input.Simular,
		categorias:       map[string]*domainestoque.Categoria{},
		nomesDoArquivo:   map[string]int{},
		codigosDoArquivo: map[string]int{},
	}
	if estado.codigosCadastrados, err = u.produtos.IDsPorCodigos(ctx, codigosDasLinhas(input.Linhas)); err != nil {
		return nil, err
	}

	for _, linha := range input.Linhas {
		motivo, err := u.importarLinha(ctx, estado, linha)
		if err != nil {
			return nil, err
		}
		if motivo != "" {
			relatorio.Rejeitar(linha.Numero, strings.TrimSpace(linha.Nome), motivo)
			continue
		}
		relatorio.Importadas++
	}

	return relatorio, nil
}

func (u *ImportacaoProdutoUsecaseImpl) importarLinha(ctx context.Context, estado *importacaoDeProdutos, linha portsin.LinhaImportacaoProduto) (string, error) {
	if strings.TrimSpace(linha.Nome) == "" {
		return "nome do produto é obrigatório", nil
	}
	slug, err := domainestoque.NovoSlug(linha.Nome)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	if anterior, repetido := estado.nomesDoArquivo[slug.Valor()]; repetido {
		return "nome repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
	}
	codigo := strings.TrimSpace(linha.Codigo)
	if codigo != "" {
		if anterior, repetido := estado.codigosDoArquivo[codigo]; repetido {
			return "código repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
		}
		if _, cadastrado := estado.codigosCadastrados[codigo]; cadastrado {
			return "já existe um produto com este código", nil
		}
	}

	existente, err := u.produtos.ObterPorSlug(ctx, slug.Valor())
	if err != nil {
		return "", err
	}
	if existente != nil {
		return "já existe um produto com este nome", nil
	}

	categoria, err := u.categoriaDaLinha(ctx, estado, linha.Categoria)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	campos, err := camposDaLinha(linha)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	produtoInput, err := montarProdutoInput(categoria, slug, campos)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}
	produto, err := domainestoque.NovoProduto(produtoInput)
	if err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	if !estado.simular {
		if _, err := u.produtos.Criar(ctx, produto); err != nil {
			if errors.Is(err, domain.ErrConflito) {
				return "já existe um produto com este nome ou código", nil
			}
			return domain.MotivoDeRejeicao(err)
		}
	}

	estado.nomesDoArquivo[slug.Valor()] = linha.Numero
	if codigo != "" {
		estado.codigosDoArquivo[codigo] = linha.Numero
	}

	return "", nil
}

func (u *ImportacaoProdutoUsecaseImpl) categoriaDaLinha(ctx context.Context, estado *importacaoDeProdutos, nome string) (*domainestoque.Categoria, error) {
	slug, err := domainestoque.NovoSlug(nome)
	if err != nil {
		return nil, domain.ErroValidacao("categoria é obrigatória")
	}
	if categoria, ok := estado.categorias[slug.Valor()]; ok {
		return categoria, nil
	}

	categoria, err := u.categorias.ObterPorSlug(ctx, slug.Valor())
	if err != nil {
		return nil, err
	}
	if categoria == nil {
		return nil, domain.ErroValidacao("categoria " + strings.TrimSpace(nome) + " não encontrada")
	}
	estado.categorias[slug.Valor()] = categoria

	return categoria, nil
}

func camposDaLinha(linha portsin.LinhaImportacaoProduto) (produtoCampos, error) {
	preco, err := domain.CentavosDeImportacao(linha.Preco, "preco")
	if err != nil {
		return produtoCampos{}, err
	}
	precoPromocional, err := domain.CentavosDeImportacao(linha.PrecoPromocional, "preco_promocional")
	if err != nil {
		return produtoCampos{}, err
	}
	estoqueMinimo, err := domain.InteiroDeImportacao(linha.EstoqueMinimo, "estoque_minimo")
	if err != nil {
		return produtoCampos{}, err
	}
	peso, err := domain.DecimalDeImportacao(linha.PesoKg, "peso_kg")
	if err != nil {
		return produtoCampos{}, err
	}
	destaque, err := domain.FlagDeImportacao(linha.Destaque, "destaque")
	if err != nil {
		return produtoCampos{}, err
	}

	var codigo *string
	if limpo := strings.TrimSpace(linha.Codigo); limpo != "" {
		codigo = &limpo
	}

	return produtoCampos{
		nome:                     strings.TrimSpace(linha.Nome),
		descricao:                strings.TrimSpace(linha.Descricao),
		codigo:                   codigo,
		unidadeMedida:            linha.UnidadeMedida,
		precoCentavos:            preco,
		precoPromocionalCentavos: precoPromocional,
		estoqueMinimo:            estoqueMinimo,
		pesoKg:                   peso,
		destaque:                 destaque,
	}, nil
}

func codigosDasLinhas(linhas []portsin.LinhaImportacaoProduto) []string {
	codigos := make([]string, 0, len(linhas))
	for _, linha := range linhas {
		if codigo := strings.TrimSpace(linha.Codigo); codigo != "" {
			codigos = append(codigos, codigo)
		}
	}

	return codigos
}
