package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.CategoriaRepository = (*CategoriaRepository)(nil)

type CategoriaRepository struct {
	db *sqlx.DB
}

func NewCategoriaRepository(db *sqlx.DB) *CategoriaRepository {
	return &CategoriaRepository{db: db}
}

type categoriaLinha struct {
	ID             uuid.UUID  `db:"id"`
	Nome           string     `db:"nome"`
	CategoriaPaiID *uuid.UUID `db:"categoria_pai_id"`
	Slug           string     `db:"slug"`
	Ordem          int        `db:"ordem"`
	Icone          string     `db:"icone"`
	Ativo          bool       `db:"ativo"`
	CriadoEm       time.Time  `db:"criado_em"`
	AtualizadoEm   time.Time  `db:"atualizado_em"`
}

const colunasCategoria = `
	id, nome, categoria_pai_id, slug, ordem, icone, ativo, criado_em, atualizado_em
`

func (c *CategoriaRepository) Criar(ctx context.Context, categoria *domainestoque.Categoria) (uuid.UUID, error) {
	query := `
		INSERT INTO categoria (id, nome, categoria_pai_id, slug, ordem, icone, ativo, criado_em, atualizado_em)
		VALUES (:id, :nome, :categoria_pai_id, :slug, :ordem, :icone, :ativo, :criado_em, :atualizado_em)
	`

	if _, err := c.db.NamedExecContext(ctx, query, paraLinhaCategoria(categoria)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return categoria.ID, nil
}

func (c *CategoriaRepository) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Categoria, error) {
	query := `SELECT ` + colunasCategoria + ` FROM categoria WHERE id = $1`

	var linha categoriaLinha
	if err := c.db.GetContext(ctx, &linha, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioCategoria(&linha), nil
}

func (c *CategoriaRepository) ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Categoria, error) {
	query := `SELECT ` + colunasCategoria + ` FROM categoria WHERE slug = $1`

	var linha categoriaLinha
	if err := c.db.GetContext(ctx, &linha, query, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioCategoria(&linha), nil
}

func (c *CategoriaRepository) Listar(ctx context.Context, filtro portsout.CategoriaFiltro) ([]*domainestoque.Categoria, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasCategoria + ` FROM categoria`
	condicoes := make([]string, 0, 1)
	argumentos := make([]any, 0, 3)

	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY ordem ASC, nome ASC` + limiteOffset(argumentos)

	var linhas []*categoriaLinha
	if err := c.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Categoria, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioCategoria(linha))
	}

	return itens, nil
}

func (c *CategoriaRepository) Atualizar(ctx context.Context, categoria *domainestoque.Categoria) (bool, error) {
	query := `
		UPDATE categoria
		SET nome = $1,
		    categoria_pai_id = $2,
		    slug = $3,
		    ordem = $4,
		    icone = $5,
		    ativo = $6,
		    atualizado_em = $7
		WHERE id = $8
	`

	resultado, err := c.db.ExecContext(ctx, query,
		categoria.Nome,
		categoria.CategoriaPaiID,
		categoria.Slug.Valor(),
		categoria.Ordem,
		categoria.Icone,
		categoria.Ativo,
		categoria.AtualizadoEm,
		categoria.ID,
	)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return linhas > 0, nil
}

func (c *CategoriaRepository) PossuiSubcategorias(ctx context.Context, id uuid.UUID) (bool, error) {
	return existeRegistro(ctx, c.db, `SELECT EXISTS (SELECT 1 FROM categoria WHERE categoria_pai_id = $1)`, id)
}

func (c *CategoriaRepository) PossuiProdutos(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM produto p
			JOIN categoria cat ON cat.id = p.categoria_id
			WHERE cat.id = $1 OR cat.categoria_pai_id = $1
		)
	`

	return existeRegistro(ctx, c.db, query, id)
}

func paraLinhaCategoria(categoria *domainestoque.Categoria) categoriaLinha {
	return categoriaLinha{
		ID:             categoria.ID,
		Nome:           categoria.Nome,
		CategoriaPaiID: categoria.CategoriaPaiID,
		Slug:           categoria.Slug.Valor(),
		Ordem:          categoria.Ordem,
		Icone:          categoria.Icone,
		Ativo:          categoria.Ativo,
		CriadoEm:       categoria.CriadoEm,
		AtualizadoEm:   categoria.AtualizadoEm,
	}
}

func paraDominioCategoria(linha *categoriaLinha) *domainestoque.Categoria {
	return &domainestoque.Categoria{
		ID:             linha.ID,
		Nome:           linha.Nome,
		CategoriaPaiID: linha.CategoriaPaiID,
		Slug:           domainestoque.SlugDe(linha.Slug),
		Ordem:          linha.Ordem,
		Icone:          linha.Icone,
		Ativo:          linha.Ativo,
		CriadoEm:       linha.CriadoEm,
		AtualizadoEm:   linha.AtualizadoEm,
	}
}

func existeRegistro(ctx context.Context, db *sqlx.DB, query string, argumentos ...any) (bool, error) {
	var existe bool
	if err := db.GetContext(ctx, &existe, query, argumentos...); err != nil {
		return false, err
	}

	return existe, nil
}
