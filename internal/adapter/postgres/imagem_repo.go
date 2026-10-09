package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.ImagemRepository = (*ImagemRepository)(nil)

type ImagemRepository struct {
	db *sqlx.DB
}

func NewImagemRepository(db *sqlx.DB) *ImagemRepository {
	return &ImagemRepository{db: db}
}

type imagemLinha struct {
	ID        uuid.UUID `db:"id"`
	ProdutoID uuid.UUID `db:"produto_id"`
	ArquivoID uuid.UUID `db:"arquivo_id"`
	Ordem     int       `db:"ordem"`
	Alt       string    `db:"alt"`
	CriadoEm  time.Time `db:"criado_em"`
}

const colunasImagem = `
	id, produto_id, arquivo_id, ordem, alt, criado_em
`

const ordenacaoImagem = ` ORDER BY ordem ASC, criado_em ASC, id ASC`

func (i *ImagemRepository) Criar(ctx context.Context, imagem *domainestoque.Imagem) (uuid.UUID, error) {
	query := `
		INSERT INTO produto_imagem (id, produto_id, arquivo_id, ordem, alt, criado_em)
		VALUES (:id, :produto_id, :arquivo_id, :ordem, :alt, :criado_em)
	`

	if _, err := i.db.NamedExecContext(ctx, query, paraLinhaImagem(imagem)); err != nil {
		return uuid.Nil, tratarErro(err)
	}

	return imagem.ID, nil
}

func (i *ImagemRepository) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Imagem, error) {
	query := `SELECT ` + colunasImagem + ` FROM produto_imagem WHERE id = $1`

	var linha imagemLinha
	if err := i.db.GetContext(ctx, &linha, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioImagem(&linha), nil
}

func (i *ImagemRepository) ListarPorProduto(ctx context.Context, produtoID uuid.UUID) ([]*domainestoque.Imagem, error) {
	query := `SELECT ` + colunasImagem + ` FROM produto_imagem WHERE produto_id = $1` + ordenacaoImagem

	return i.listar(ctx, query, produtoID)
}

func (i *ImagemRepository) ListarPorProdutos(ctx context.Context, produtoIDs []uuid.UUID) ([]*domainestoque.Imagem, error) {
	if len(produtoIDs) == 0 {
		return []*domainestoque.Imagem{}, nil
	}

	argumentos := make([]any, 0, len(produtoIDs))
	posicoes := make([]string, 0, len(produtoIDs))
	for _, produtoID := range produtoIDs {
		argumentos = append(argumentos, produtoID)
		posicoes = append(posicoes, "$"+strconv.Itoa(len(argumentos)))
	}

	query := `SELECT ` + colunasImagem + ` FROM produto_imagem WHERE produto_id IN (` +
		strings.Join(posicoes, ", ") + `)` + ordenacaoImagem

	return i.listar(ctx, query, argumentos...)
}

func (i *ImagemRepository) Remover(ctx context.Context, id uuid.UUID) (bool, error) {
	resultado, err := i.db.ExecContext(ctx, `DELETE FROM produto_imagem WHERE id = $1`, id)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}

	return linhas > 0, nil
}

func (i *ImagemRepository) listar(ctx context.Context, query string, argumentos ...any) ([]*domainestoque.Imagem, error) {
	var linhas []*imagemLinha
	if err := i.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Imagem, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioImagem(linha))
	}

	return itens, nil
}

func paraLinhaImagem(imagem *domainestoque.Imagem) imagemLinha {
	return imagemLinha{
		ID:        imagem.ID,
		ProdutoID: imagem.ProdutoID,
		ArquivoID: imagem.ArquivoID,
		Ordem:     imagem.Ordem,
		Alt:       imagem.Alt,
		CriadoEm:  imagem.CriadoEm,
	}
}

func paraDominioImagem(linha *imagemLinha) *domainestoque.Imagem {
	return &domainestoque.Imagem{
		ID:        linha.ID,
		ProdutoID: linha.ProdutoID,
		ArquivoID: linha.ArquivoID,
		Ordem:     linha.Ordem,
		Alt:       linha.Alt,
		CriadoEm:  linha.CriadoEm,
	}
}
