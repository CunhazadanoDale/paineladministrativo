package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.EstoqueRepository = (*EstoqueRepository)(nil)

type EstoqueRepository struct {
	db *sqlx.DB
}

func NewEstoqueRepository(db *sqlx.DB) *EstoqueRepository {
	return &EstoqueRepository{db: db}
}

type saldoLocalLinha struct {
	ProdutoID    uuid.UUID `db:"produto_id"`
	LocalID      uuid.UUID `db:"local_id"`
	Quantidade   int       `db:"quantidade"`
	Reservado    int       `db:"reservado"`
	AtualizadoEm time.Time `db:"atualizado_em"`
}

type saldoLocalDetalheLinha struct {
	saldoLocalLinha
	LocalNome     string    `db:"local_nome"`
	EmpresaID     uuid.UUID `db:"empresa_id"`
	ProdutoNome   string    `db:"produto_nome"`
	ProdutoCodigo *string   `db:"produto_codigo"`
	UnidadeMedida string    `db:"unidade_medida"`
}

const consultaSaldoLocalDetalhe = `
	SELECT s.produto_id, s.local_id, s.quantidade, s.reservado, s.atualizado_em,
	       l.nome AS local_nome, l.empresa_id,
	       p.nome AS produto_nome, p.codigo AS produto_codigo, p.unidade_medida
	FROM produto_saldo_local s
	JOIN estoque_local l ON l.id = s.local_id
	JOIN produto p ON p.id = s.produto_id
`

func (e *EstoqueRepository) Carregar(ctx context.Context, produtoID uuid.UUID, localIDs []uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	tx, err := e.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var produto produtoLinha
	if err := tx.GetContext(ctx, &produto, `SELECT `+colunasProduto+` FROM produto WHERE id = $1`, produtoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	locais := pq.Array(textosDeIDs(localIDs))

	var saldos []*saldoLocalLinha
	if err := tx.SelectContext(ctx, &saldos, `
		SELECT produto_id, local_id, quantidade, reservado, atualizado_em
		FROM produto_saldo_local
		WHERE produto_id = $1 AND local_id = ANY($2::uuid[])
	`, produtoID, locais); err != nil {
		return nil, err
	}

	var lotes []*loteLinha
	if err := tx.SelectContext(ctx, &lotes, `
		SELECT `+colunasLote+`
		FROM produto_lote
		WHERE produto_id = $1 AND local_id = ANY($2::uuid[]) AND quantidade_atual > 0
		ORDER BY entrada_em ASC, id ASC
	`, produtoID, locais); err != nil {
		return nil, err
	}

	var totalAlocado int
	if err := tx.GetContext(ctx, &totalAlocado,
		`SELECT COALESCE(SUM(quantidade), 0) FROM produto_saldo_local WHERE produto_id = $1`, produtoID,
	); err != nil {
		return nil, err
	}

	estoque := &domainestoque.EstoqueProduto{
		Produto:      paraDominioProduto(&produto),
		Saldos:       make(map[uuid.UUID]domainestoque.SaldoLocal, len(saldos)),
		Lotes:        make([]*domainestoque.Lote, 0, len(lotes)),
		TotalAlocado: totalAlocado,
	}
	for _, saldo := range saldos {
		estoque.Saldos[saldo.LocalID] = paraDominioSaldoLocal(saldo)
	}
	for _, lote := range lotes {
		estoque.Lotes = append(estoque.Lotes, paraDominioLote(lote))
	}

	return estoque, nil
}

func (e *EstoqueRepository) Gravar(ctx context.Context, operacao *domainestoque.OperacaoEstoque) error {
	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := gravarOperacaoEstoque(ctx, tx, operacao); err != nil {
		return err
	}

	return tratarErroDeGravacao(tx.Commit())
}

func gravarOperacaoEstoque(ctx context.Context, tx *sqlx.Tx, operacao *domainestoque.OperacaoEstoque) error {
	if err := avancarVersaoEstoque(ctx, tx, operacao); err != nil {
		return err
	}
	if err := exigirLocaisAtivos(ctx, tx, operacao.LotesCriados); err != nil {
		return err
	}

	for _, saldo := range operacao.Saldos {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO produto_saldo_local (produto_id, local_id, quantidade, reservado, atualizado_em)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (produto_id, local_id) DO UPDATE
			SET quantidade = EXCLUDED.quantidade,
			    reservado = EXCLUDED.reservado,
			    atualizado_em = EXCLUDED.atualizado_em
		`, saldo.ProdutoID, saldo.LocalID, saldo.Quantidade, saldo.Reservado, saldo.AtualizadoEm); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	if operacao.ReservaCriada != nil {
		if err := inserirReserva(ctx, tx, operacao.ReservaCriada); err != nil {
			return tratarErroDeGravacao(err)
		}
	}
	if operacao.ReservaEncerrada != nil {
		if err := encerrarReserva(ctx, tx, operacao.ReservaEncerrada); err != nil {
			return err
		}
	}

	if operacao.Movimento != nil {
		if err := inserirMovimento(ctx, tx, operacao.Movimento); err != nil {
			return tratarErroDeGravacao(err)
		}

		for _, consumo := range operacao.Movimento.Consumos {
			if err := consumirLote(ctx, tx, operacao.Movimento.ID, consumo); err != nil {
				return err
			}
		}
	}

	for _, lote := range operacao.LotesCriados {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO produto_lote (
				id, produto_id, local_id, movimento_id, lote_origem_id, custo_unitario_centavos,
				quantidade_inicial, quantidade_atual, entrada_em, criado_em
			)
			VALUES (
				:id, :produto_id, :local_id, :movimento_id, :lote_origem_id, :custo_unitario_centavos,
				:quantidade_inicial, :quantidade_atual, :entrada_em, :criado_em
			)
		`, paraLinhaLote(lote)); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	return nil
}

func avancarVersaoEstoque(ctx context.Context, tx *sqlx.Tx, operacao *domainestoque.OperacaoEstoque) error {
	resultado, err := tx.ExecContext(ctx, `
		UPDATE produto
		SET quantidade_atual = $1,
		    quantidade_reservada = $2,
		    versao_estoque = versao_estoque + 1,
		    atualizado_em = $3
		WHERE id = $4
		  AND versao_estoque = $5
	`,
		operacao.Saldo.Quantidade(),
		operacao.Reservado.Quantidade(),
		operacao.AtualizadoEm,
		operacao.ProdutoID,
		operacao.VersaoLida,
	)
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domain.ErroConflito("estoque do produto alterado por outra operação, recarregue e tente novamente")
	}

	return nil
}

func exigirLocaisAtivos(ctx context.Context, tx *sqlx.Tx, lotes []*domainestoque.Lote) error {
	if len(lotes) == 0 {
		return nil
	}

	localIDs := make([]uuid.UUID, 0, len(lotes))
	for _, lote := range lotes {
		localIDs = append(localIDs, lote.LocalID)
	}

	var inativos int
	if err := tx.GetContext(ctx, &inativos, `
		SELECT COUNT(*) FROM (
			SELECT ativo FROM estoque_local WHERE id = ANY($1::uuid[]) FOR SHARE
		) locais
		WHERE NOT ativo
	`, pq.Array(textosDeIDs(localIDs))); err != nil {
		return tratarErroDeGravacao(err)
	}
	if inativos > 0 {
		return domain.ErroValidacao("local inativo não recebe entrada de estoque")
	}

	return nil
}

func consumirLote(ctx context.Context, tx *sqlx.Tx, movimentoID uuid.UUID, consumo domainestoque.ConsumoLote) error {
	resultado, err := tx.ExecContext(ctx, `
		UPDATE produto_lote
		SET quantidade_atual = quantidade_atual - $1
		WHERE id = $2
		  AND quantidade_atual >= $1
	`, consumo.Quantidade, consumo.LoteID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domain.ErroConflito("lote consumido por outra operação, recarregue e tente novamente")
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO produto_movimento_lote (movimento_id, lote_id, quantidade, custo_unitario_centavos)
		VALUES ($1, $2, $3, $4)
	`, movimentoID, consumo.LoteID, consumo.Quantidade, centavosDoCusto(consumo.Custo)); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (e *EstoqueRepository) SaldosDoProduto(ctx context.Context, produtoID uuid.UUID, empresaIDs []uuid.UUID) ([]*domainestoque.SaldoLocalDetalhe, error) {
	query := consultaSaldoLocalDetalhe + ` WHERE s.produto_id = $1`
	argumentos := []any{produtoID}
	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` AND l.empresa_id = ANY($2::uuid[])`
	}
	query += ` ORDER BY l.nome ASC, l.id ASC`

	var linhas []*saldoLocalDetalheLinha
	if err := e.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	return paraDominioSaldosDetalhados(linhas), nil
}

func (e *EstoqueRepository) SaldosDoLocal(ctx context.Context, filtro portsout.SaldoDoLocalFiltro) ([]*domainestoque.SaldoLocalDetalhe, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := consultaSaldoLocalDetalhe + ` WHERE s.local_id = $1`
	if filtro.ComSaldo {
		query += ` AND s.quantidade > 0`
	}
	argumentos := []any{filtro.LocalID, filtro.Size, offset}
	query += ` ORDER BY p.nome ASC, p.id ASC` + limiteOffset(argumentos)

	var linhas []*saldoLocalDetalheLinha
	if err := e.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	return paraDominioSaldosDetalhados(linhas), nil
}

func (e *EstoqueRepository) TotalAlocado(ctx context.Context, produtoID uuid.UUID) (int, error) {
	var total int
	if err := e.db.GetContext(ctx, &total,
		`SELECT COALESCE(SUM(quantidade), 0) FROM produto_saldo_local WHERE produto_id = $1`, produtoID,
	); err != nil {
		return 0, err
	}

	return total, nil
}

func paraDominioSaldoLocal(linha *saldoLocalLinha) domainestoque.SaldoLocal {
	return domainestoque.SaldoLocal{
		ProdutoID:    linha.ProdutoID,
		LocalID:      linha.LocalID,
		Quantidade:   linha.Quantidade,
		Reservado:    linha.Reservado,
		AtualizadoEm: linha.AtualizadoEm,
	}
}

func paraDominioSaldosDetalhados(linhas []*saldoLocalDetalheLinha) []*domainestoque.SaldoLocalDetalhe {
	itens := make([]*domainestoque.SaldoLocalDetalhe, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, &domainestoque.SaldoLocalDetalhe{
			SaldoLocal:    paraDominioSaldoLocal(&linha.saldoLocalLinha),
			LocalNome:     linha.LocalNome,
			EmpresaID:     linha.EmpresaID,
			ProdutoNome:   linha.ProdutoNome,
			ProdutoCodigo: linha.ProdutoCodigo,
			UnidadeMedida: domainestoque.UnidadeMedidaDe(linha.UnidadeMedida),
		})
	}

	return itens
}
