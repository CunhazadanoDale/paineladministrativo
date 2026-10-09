package postgres

import (
	"context"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.MovimentoRepository = (*MovimentoRepository)(nil)

type MovimentoRepository struct {
	db *sqlx.DB
}

func NewMovimentoRepository(db *sqlx.DB) *MovimentoRepository {
	return &MovimentoRepository{db: db}
}

type movimentoLinha struct {
	ID           uuid.UUID `db:"id"`
	ProdutoID    uuid.UUID `db:"produto_id"`
	Tipo         string    `db:"tipo"`
	Quantidade   int       `db:"quantidade"`
	SaldoApos    int       `db:"saldo_apos"`
	UsuarioID    uuid.UUID `db:"usuario_id"`
	DocumentoRef *string   `db:"documento_ref"`
	Observacao   string    `db:"observacao"`
	CriadoEm     time.Time `db:"criado_em"`
}

const colunasMovimento = `
	id, produto_id, tipo, quantidade, saldo_apos, usuario_id, documento_ref, observacao, criado_em
`

func (m *MovimentoRepository) Listar(ctx context.Context, filtro portsout.MovimentoFiltro) ([]*domainestoque.Movimento, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasMovimento + ` FROM produto_movimento`
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 4)

	if filtro.ProdutoID != nil {
		argumentos = append(argumentos, *filtro.ProdutoID)
		condicoes = append(condicoes, condicao("produto_id", len(argumentos)))
	}
	if filtro.Tipo != "" {
		argumentos = append(argumentos, filtro.Tipo)
		condicoes = append(condicoes, condicao("tipo", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*movimentoLinha
	if err := m.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Movimento, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioMovimento(linha))
	}

	return itens, nil
}

func (m *MovimentoRepository) ListarResumo(ctx context.Context, limite int) ([]*domainestoque.MovimentoResumo, error) {
	query := `
		SELECT m.id, m.produto_id, p.nome AS produto_nome, m.tipo, m.quantidade, m.saldo_apos, m.criado_em
		FROM produto_movimento m
		JOIN produto p ON p.id = m.produto_id
		ORDER BY m.criado_em DESC, m.id DESC
		LIMIT $1
	`

	var linhas []*movimentoResumoLinha
	if err := m.db.SelectContext(ctx, &linhas, query, limite); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.MovimentoResumo, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioMovimentoResumo(linha))
	}

	return itens, nil
}

type movimentoResumoLinha struct {
	ID          uuid.UUID `db:"id"`
	ProdutoID   uuid.UUID `db:"produto_id"`
	ProdutoNome string    `db:"produto_nome"`
	Tipo        string    `db:"tipo"`
	Quantidade  int       `db:"quantidade"`
	SaldoApos   int       `db:"saldo_apos"`
	CriadoEm    time.Time `db:"criado_em"`
}

func paraLinhaMovimento(movimento *domainestoque.Movimento) movimentoLinha {
	return movimentoLinha{
		ID:           movimento.ID,
		ProdutoID:    movimento.ProdutoID,
		Tipo:         string(movimento.Tipo),
		Quantidade:   movimento.Quantidade.Valor(),
		SaldoApos:    movimento.SaldoApos.Quantidade(),
		UsuarioID:    movimento.UsuarioID,
		DocumentoRef: movimento.DocumentoRef,
		Observacao:   movimento.Observacao,
		CriadoEm:     movimento.CriadoEm,
	}
}

func paraDominioMovimento(linha *movimentoLinha) *domainestoque.Movimento {
	return &domainestoque.Movimento{
		ID:           linha.ID,
		ProdutoID:    linha.ProdutoID,
		Tipo:         domainestoque.TipoMovimentoDe(linha.Tipo),
		Quantidade:   domainestoque.QuantidadeDe(linha.Quantidade),
		SaldoApos:    domainestoque.SaldoDe(linha.SaldoApos),
		UsuarioID:    linha.UsuarioID,
		DocumentoRef: linha.DocumentoRef,
		Observacao:   linha.Observacao,
		CriadoEm:     linha.CriadoEm,
	}
}

func paraDominioMovimentoResumo(linha *movimentoResumoLinha) *domainestoque.MovimentoResumo {
	return &domainestoque.MovimentoResumo{
		ID:          linha.ID,
		ProdutoID:   linha.ProdutoID,
		ProdutoNome: linha.ProdutoNome,
		Tipo:        domainestoque.TipoMovimentoDe(linha.Tipo),
		Quantidade:  domainestoque.QuantidadeDe(linha.Quantidade),
		SaldoApos:   domainestoque.SaldoDe(linha.SaldoApos),
		CriadoEm:    linha.CriadoEm,
	}
}
