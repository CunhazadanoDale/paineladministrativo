package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.ProdutoRepository = (*ProdutoRepository)(nil)

type ProdutoRepository struct {
	db *sqlx.DB
}

func NewProdutoRepository(db *sqlx.DB) *ProdutoRepository {
	return &ProdutoRepository{db: db}
}

type produtoLinha struct {
	ID               uuid.UUID `db:"id"`
	CategoriaID      uuid.UUID `db:"categoria_id"`
	Nome             string    `db:"nome"`
	Slug             string    `db:"slug"`
	Descricao        string    `db:"descricao"`
	Codigo           *string   `db:"codigo"`
	UnidadeMedida    string    `db:"unidade_medida"`
	Preco            *int64    `db:"preco"`
	PrecoPromocional *int64    `db:"preco_promocional"`
	QuantidadeAtual  int       `db:"quantidade_atual"`
	EstoqueMinimo    *int      `db:"estoque_minimo"`
	PesoKg           *float64  `db:"peso_kg"`
	Destaque         bool      `db:"destaque"`
	Ativo            bool      `db:"ativo"`
	CriadoEm         time.Time `db:"criado_em"`
	AtualizadoEm     time.Time `db:"atualizado_em"`
}

const colunasProduto = `
	id, categoria_id, nome, slug, descricao, codigo, unidade_medida, preco,
	preco_promocional, quantidade_atual, estoque_minimo, peso_kg, destaque,
	ativo, criado_em, atualizado_em
`

func (p *ProdutoRepository) Criar(ctx context.Context, produto *domainestoque.Produto) (uuid.UUID, error) {
	query := `
		INSERT INTO produto (
			id, categoria_id, nome, slug, descricao, codigo, unidade_medida, preco,
			preco_promocional, quantidade_atual, estoque_minimo, peso_kg, destaque,
			ativo, criado_em, atualizado_em
		)
		VALUES (
			:id, :categoria_id, :nome, :slug, :descricao, :codigo, :unidade_medida, :preco,
			:preco_promocional, :quantidade_atual, :estoque_minimo, :peso_kg, :destaque,
			:ativo, :criado_em, :atualizado_em
		)
	`

	if _, err := p.db.NamedExecContext(ctx, query, paraLinhaProduto(produto)); err != nil {
		return uuid.Nil, tratarErro(err)
	}

	return produto.ID, nil
}

func (p *ProdutoRepository) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	query := `SELECT ` + colunasProduto + ` FROM produto WHERE id = $1`

	var linha produtoLinha
	if err := p.db.GetContext(ctx, &linha, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioProduto(&linha), nil
}

func (p *ProdutoRepository) ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Produto, error) {
	query := `SELECT ` + colunasProduto + ` FROM produto WHERE slug = $1`

	var linha produtoLinha
	if err := p.db.GetContext(ctx, &linha, query, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioProduto(&linha), nil
}

func (p *ProdutoRepository) Listar(ctx context.Context, filtro portsout.ProdutoFiltro) ([]*domainestoque.Produto, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasProduto + ` FROM produto`
	condicoes := make([]string, 0, 5)
	argumentos := make([]any, 0, 7)

	if filtro.CategoriaID != nil {
		argumentos = append(argumentos, *filtro.CategoriaID)
		posicao := len(argumentos)
		condicoes = append(condicoes,
			"(categoria_id = $"+strconv.Itoa(posicao)+
				" OR categoria_id IN (SELECT id FROM categoria WHERE categoria_pai_id = $"+strconv.Itoa(posicao)+"))")
	}
	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}
	if filtro.Destaque != nil {
		argumentos = append(argumentos, *filtro.Destaque)
		condicoes = append(condicoes, condicao("destaque", len(argumentos)))
	}
	if filtro.EstoqueBaixo {
		condicoes = append(condicoes, "estoque_minimo IS NOT NULL AND estoque_minimo > 0 AND quantidade_atual <= estoque_minimo")
	}
	if filtro.ComSaldo {
		condicoes = append(condicoes, "quantidade_atual > 0")
	}
	if strings.TrimSpace(filtro.Busca) != "" {
		argumentos = append(argumentos, "%"+escaparBusca(strings.TrimSpace(filtro.Busca))+"%")
		posicao := len(argumentos)
		condicoes = append(condicoes,
			"(nome ILIKE $"+strconv.Itoa(posicao)+" OR codigo ILIKE $"+strconv.Itoa(posicao)+")")
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome ASC` + limiteOffset(argumentos)

	var linhas []*produtoLinha
	if err := p.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Produto, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioProduto(linha))
	}

	return itens, nil
}

func (p *ProdutoRepository) Resumo(ctx context.Context) (domainestoque.ResumoProdutos, error) {
	query := `
		SELECT
			COUNT(*) AS total_produtos,
			COUNT(*) FILTER (WHERE ativo) AS produtos_ativos,
			COALESCE(
				SUM(preco * quantidade_atual) FILTER (WHERE ativo AND preco IS NOT NULL),
				0
			) AS valor_estoque_centavos,
			COUNT(*) FILTER (
				WHERE ativo
				  AND estoque_minimo IS NOT NULL
				  AND estoque_minimo > 0
				  AND quantidade_atual <= estoque_minimo
			) AS produtos_estoque_baixo
		FROM produto
	`

	var linha produtoResumoLinha
	if err := p.db.GetContext(ctx, &linha, query); err != nil {
		return domainestoque.ResumoProdutos{}, err
	}

	return domainestoque.ResumoProdutos{
		TotalProdutos:        linha.TotalProdutos,
		ProdutosAtivos:       linha.ProdutosAtivos,
		ValorEstoqueCentavos: linha.ValorEstoqueCentavos,
		ProdutosEstoqueBaixo: linha.ProdutosEstoqueBaixo,
	}, nil
}

type produtoResumoLinha struct {
	TotalProdutos        int64 `db:"total_produtos"`
	ProdutosAtivos       int64 `db:"produtos_ativos"`
	ValorEstoqueCentavos int64 `db:"valor_estoque_centavos"`
	ProdutosEstoqueBaixo int64 `db:"produtos_estoque_baixo"`
}

func (p *ProdutoRepository) Atualizar(ctx context.Context, produto *domainestoque.Produto) (bool, error) {
	query := `
		UPDATE produto
		SET categoria_id = $1,
		    nome = $2,
		    slug = $3,
		    descricao = $4,
		    codigo = $5,
		    unidade_medida = $6,
		    preco = $7,
		    preco_promocional = $8,
		    estoque_minimo = $9,
		    peso_kg = $10,
		    destaque = $11,
		    ativo = $12,
		    atualizado_em = $13
		WHERE id = $14
	`

	linha := paraLinhaProduto(produto)
	resultado, err := p.db.ExecContext(ctx, query,
		linha.CategoriaID,
		linha.Nome,
		linha.Slug,
		linha.Descricao,
		linha.Codigo,
		linha.UnidadeMedida,
		linha.Preco,
		linha.PrecoPromocional,
		linha.EstoqueMinimo,
		linha.PesoKg,
		linha.Destaque,
		linha.Ativo,
		linha.AtualizadoEm,
		linha.ID,
	)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}

	return linhas > 0, nil
}

func (p *ProdutoRepository) Movimentar(ctx context.Context, produto *domainestoque.Produto, movimento *domainestoque.Movimento) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	efeito := movimento.Tipo.Efeito(movimento.Quantidade)
	saldoAnterior := movimento.SaldoApos.Quantidade() - efeito

	atualizado, err := atualizarSaldoComGuarda(ctx, tx, produto.ID, movimento, saldoAnterior)
	if err != nil {
		return err
	}
	if !atualizado {
		return domain.ErroConflito("produto com saldo alterado por outra operação, recarregue e tente novamente")
	}

	linha := paraLinhaMovimento(movimento)
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO produto_movimento (id, produto_id, tipo, quantidade, saldo_apos, usuario_id, documento_ref, observacao, criado_em)
		VALUES (:id, :produto_id, :tipo, :quantidade, :saldo_apos, :usuario_id, :documento_ref, :observacao, :criado_em)
	`, linha); err != nil {
		return tratarErro(err)
	}

	return tx.Commit()
}

func atualizarSaldoComGuarda(ctx context.Context, tx *sqlx.Tx, produtoID uuid.UUID, movimento *domainestoque.Movimento, saldoAnterior int) (bool, error) {
	argumentos := map[string]any{
		"id":             produtoID,
		"saldo_apos":     movimento.SaldoApos.Quantidade(),
		"saldo_anterior": saldoAnterior,
		"atualizado_em":  movimento.CriadoEm,
	}

	resultado, err := tx.NamedExecContext(ctx, `
		UPDATE produto
		SET quantidade_atual = :saldo_apos,
		    atualizado_em = :atualizado_em
		WHERE id = :id
		  AND quantidade_atual = :saldo_anterior
	`, argumentos)
	if err != nil {
		return false, err
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}

	return linhas > 0, nil
}

func paraLinhaProduto(produto *domainestoque.Produto) produtoLinha {
	linha := produtoLinha{
		ID:              produto.ID,
		CategoriaID:     produto.CategoriaID,
		Nome:            produto.Nome,
		Slug:            produto.Slug.Valor(),
		Descricao:       produto.Descricao,
		Codigo:          produto.Codigo,
		UnidadeMedida:   string(produto.UnidadeMedida),
		QuantidadeAtual: produto.Saldo.Quantidade(),
		Destaque:        produto.Destaque,
		Ativo:           produto.Ativo,
		CriadoEm:        produto.CriadoEm,
		AtualizadoEm:    produto.AtualizadoEm,
	}

	if produto.Preco != nil {
		centavos := produto.Preco.Centavos()
		linha.Preco = &centavos
	}
	if produto.PrecoPromocional != nil {
		centavos := produto.PrecoPromocional.Centavos()
		linha.PrecoPromocional = &centavos
	}
	if produto.EstoqueMinimo != nil {
		minimo := produto.EstoqueMinimo.Quantidade()
		linha.EstoqueMinimo = &minimo
	}
	if produto.Peso != nil {
		quilogramas := produto.Peso.Quilos()
		linha.PesoKg = &quilogramas
	}

	return linha
}

func paraDominioProduto(linha *produtoLinha) *domainestoque.Produto {
	produto := &domainestoque.Produto{
		ID:            linha.ID,
		CategoriaID:   linha.CategoriaID,
		Nome:          linha.Nome,
		Slug:          domainestoque.SlugDe(linha.Slug),
		Descricao:     linha.Descricao,
		Codigo:        linha.Codigo,
		UnidadeMedida: domainestoque.UnidadeMedidaDe(linha.UnidadeMedida),
		Saldo:         domainestoque.SaldoDe(linha.QuantidadeAtual),
		Destaque:      linha.Destaque,
		Ativo:         linha.Ativo,
		CriadoEm:      linha.CriadoEm,
		AtualizadoEm:  linha.AtualizadoEm,
	}

	if linha.Preco != nil {
		preco := domainestoque.PrecoDe(*linha.Preco)
		produto.Preco = &preco
	}
	if linha.PrecoPromocional != nil {
		promocional := domainestoque.PrecoDe(*linha.PrecoPromocional)
		produto.PrecoPromocional = &promocional
	}
	if linha.EstoqueMinimo != nil {
		minimo := domainestoque.EstoqueMinimoDe(*linha.EstoqueMinimo)
		produto.EstoqueMinimo = &minimo
	}
	if linha.PesoKg != nil {
		peso := domainestoque.PesoDe(*linha.PesoKg)
		produto.Peso = &peso
	}

	return produto
}

func escaparBusca(busca string) string {
	busca = strings.ReplaceAll(busca, `\`, `\\`)
	busca = strings.ReplaceAll(busca, `%`, `\%`)
	busca = strings.ReplaceAll(busca, `_`, `\_`)

	return busca
}
