package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.TabelaPrecoRepository = (*TabelaPrecoRepository)(nil)

type TabelaPrecoRepository struct {
	db *sqlx.DB
}

func NewTabelaPrecoRepository(db *sqlx.DB) *TabelaPrecoRepository {
	return &TabelaPrecoRepository{db: db}
}

type tabelaPrecoItemLinha struct {
	ID            uuid.UUID `db:"id"`
	ProdutoID     uuid.UUID `db:"produto_id"`
	Modelo        string    `db:"modelo"`
	PrecoCentavos int64     `db:"preco_centavos"`
	VigenteDe     time.Time `db:"vigente_de"`
	Observacao    string    `db:"observacao"`
	UsuarioID     uuid.UUID `db:"usuario_id"`
	CriadoEm      time.Time `db:"criado_em"`
}

type precoVigenteLinha struct {
	ProdutoID     uuid.UUID  `db:"produto_id"`
	ProdutoNome   string     `db:"produto_nome"`
	ProdutoCodigo *string    `db:"produto_codigo"`
	Unidade       string     `db:"unidade_medida"`
	PrecoProduto  *int64     `db:"preco_produto"`
	ItemID        *uuid.UUID `db:"item_id"`
	PrecoTabela   *int64     `db:"preco_tabela"`
	VigenteDe     *time.Time `db:"vigente_de"`
}

const colunasTabelaPrecoItem = `
	id, produto_id, modelo, preco_centavos, vigente_de, observacao, usuario_id, criado_em
`

func (r *TabelaPrecoRepository) Criar(ctx context.Context, item *domaintabelapreco.Item) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	var ativo bool
	if err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM produto WHERE id = $1 FOR SHARE`, item.ProdutoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domaintabelapreco.ErroProdutoNaoEncontrado()
		}
		return tratarErroDeGravacao(err)
	}
	if !ativo {
		return domaintabelapreco.ErroProdutoInativo()
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO tabela_preco_item (`+colunasTabelaPrecoItem+`) VALUES (
			:id, :produto_id, :modelo, :preco_centavos, :vigente_de, :observacao, :usuario_id, :criado_em
		)
	`, paraLinhaTabelaPrecoItem(item)); err != nil {
		var erroPostgres *pq.Error
		if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation && erroPostgres.Constraint == "uq_tabela_preco_item" {
			return domaintabelapreco.ErroVigenciaRepetida()
		}
		return tratarErroDeGravacao(err)
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (r *TabelaPrecoRepository) Listar(ctx context.Context, filtro portsout.ItemFiltro) ([]*domaintabelapreco.Item, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 4)
	if filtro.ProdutoID != nil {
		argumentos = append(argumentos, *filtro.ProdutoID)
		condicoes = append(condicoes, "produto_id = $"+strconv.Itoa(len(argumentos)))
	}
	if filtro.Modelo != "" {
		argumentos = append(argumentos, filtro.Modelo.String())
		condicoes = append(condicoes, "modelo = $"+strconv.Itoa(len(argumentos)))
	}

	query := `SELECT ` + colunasTabelaPrecoItem + ` FROM tabela_preco_item`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY vigente_de DESC, criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*tabelaPrecoItemLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	itens := make([]*domaintabelapreco.Item, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, &domaintabelapreco.Item{
			ID:            linha.ID,
			ProdutoID:     linha.ProdutoID,
			Modelo:        domainpedido.ModeloDe(linha.Modelo),
			PrecoCentavos: linha.PrecoCentavos,
			VigenteDe:     linha.VigenteDe,
			Observacao:    linha.Observacao,
			UsuarioID:     linha.UsuarioID,
			CriadoEm:      linha.CriadoEm,
		})
	}

	return itens, nil
}

func (r *TabelaPrecoRepository) Vigentes(ctx context.Context, filtro portsout.VigenteFiltro) ([]*domaintabelapreco.PrecoVigente, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	argumentos := []any{filtro.Modelo.String(), filtro.Data}
	condicoes := []string{"p.ativo"}
	if filtro.ProdutoID != nil {
		argumentos = append(argumentos, *filtro.ProdutoID)
		condicoes = append(condicoes, "p.id = $"+strconv.Itoa(len(argumentos)))
	}
	if filtro.Busca != "" {
		argumentos = append(argumentos, "%"+escaparBusca(filtro.Busca)+"%")
		posicao := strconv.Itoa(len(argumentos))
		condicoes = append(condicoes, "(p.nome ILIKE $"+posicao+" OR p.codigo ILIKE $"+posicao+")")
	}
	argumentos = append(argumentos, filtro.Size, offset)

	var linhas []*precoVigenteLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT p.id AS produto_id, p.nome AS produto_nome, p.codigo AS produto_codigo, p.unidade_medida,
			p.preco AS preco_produto, t.id AS item_id, t.preco_centavos AS preco_tabela, t.vigente_de
		FROM produto p
		LEFT JOIN LATERAL (
			SELECT id, preco_centavos, vigente_de
			FROM tabela_preco_item
			WHERE produto_id = p.id AND modelo = $1 AND vigente_de <= $2
			ORDER BY vigente_de DESC
			LIMIT 1
		) t ON true
		WHERE `+unirCondicoes(condicoes)+`
		ORDER BY p.nome ASC, p.id ASC`+limiteOffset(argumentos), argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	vigentes := make([]*domaintabelapreco.PrecoVigente, 0, len(linhas))
	for _, linha := range linhas {
		vigente := &domaintabelapreco.PrecoVigente{
			ProdutoID:     linha.ProdutoID,
			ProdutoNome:   linha.ProdutoNome,
			ProdutoCodigo: linha.ProdutoCodigo,
			Unidade:       linha.Unidade,
			Modelo:        filtro.Modelo,
			Data:          filtro.Data,
		}
		vigente.Resolver(linha.PrecoTabela, linha.ItemID, linha.VigenteDe, linha.PrecoProduto)
		vigentes = append(vigentes, vigente)
	}

	return vigentes, nil
}

func (r *TabelaPrecoRepository) RemoverFuturo(ctx context.Context, id uuid.UUID, hoje time.Time) error {
	resultado, err := r.db.ExecContext(ctx, `DELETE FROM tabela_preco_item WHERE id = $1 AND vigente_de > $2`, id, hoje)
	if err != nil {
		return tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErro(err)
	}
	if linhas > 0 {
		return nil
	}

	var existe bool
	if err := r.db.GetContext(ctx, &existe, `SELECT EXISTS (SELECT 1 FROM tabela_preco_item WHERE id = $1)`, id); err != nil {
		return tratarErroDeConsulta(err)
	}
	if !existe {
		return domaintabelapreco.ErroItemNaoEncontrado()
	}

	return domaintabelapreco.ErroVigenciaJaComecou()
}

func paraLinhaTabelaPrecoItem(item *domaintabelapreco.Item) *tabelaPrecoItemLinha {
	return &tabelaPrecoItemLinha{
		ID:            item.ID,
		ProdutoID:     item.ProdutoID,
		Modelo:        item.Modelo.String(),
		PrecoCentavos: item.PrecoCentavos,
		VigenteDe:     item.VigenteDe,
		Observacao:    item.Observacao,
		UsuarioID:     item.UsuarioID,
		CriadoEm:      item.CriadoEm,
	}
}

func (r *TabelaPrecoRepository) PrecoVigente(ctx context.Context, produtoID uuid.UUID, modelo domainpedido.Modelo, data time.Time) (*int64, error) {
	var preco *int64
	if err := r.db.GetContext(ctx, &preco, `
		SELECT COALESCE(
			(SELECT preco_centavos FROM tabela_preco_item
			 WHERE produto_id = $1 AND modelo = $2 AND vigente_de <= $3
			 ORDER BY vigente_de DESC
			 LIMIT 1),
			(SELECT preco FROM produto WHERE id = $1)
		)
	`, produtoID, modelo.String(), data); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	return preco, nil
}
