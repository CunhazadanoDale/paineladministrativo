package postgres

import (
	"context"
	"strconv"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.MovimentoRepository = (*MovimentoRepository)(nil)

type MovimentoRepository struct {
	db *sqlx.DB
}

func NewMovimentoRepository(db *sqlx.DB) *MovimentoRepository {
	return &MovimentoRepository{db: db}
}

type movimentoLinha struct {
	ID             uuid.UUID  `db:"id"`
	ProdutoID      uuid.UUID  `db:"produto_id"`
	Tipo           string     `db:"tipo"`
	Quantidade     int        `db:"quantidade"`
	SaldoApos      int        `db:"saldo_apos"`
	LocalID        *uuid.UUID `db:"local_id"`
	LocalDestinoID *uuid.UUID `db:"local_destino_id"`
	SaldoLocalApos *int       `db:"saldo_local_apos"`
	Custo          *int64     `db:"custo_unitario_centavos"`
	ReservaID      *uuid.UUID `db:"reserva_id"`
	UsuarioID      uuid.UUID  `db:"usuario_id"`
	DocumentoRef   *string    `db:"documento_ref"`
	Observacao     string     `db:"observacao"`
	CriadoEm       time.Time  `db:"criado_em"`
}

const colunasMovimento = `
	id, produto_id, tipo, quantidade, saldo_apos, local_id, local_destino_id, saldo_local_apos,
	custo_unitario_centavos, reserva_id, usuario_id, documento_ref, observacao, criado_em
`

func (m *MovimentoRepository) Listar(ctx context.Context, filtro portsout.MovimentoFiltro) ([]*domainestoque.Movimento, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasMovimento + ` FROM produto_movimento`
	condicoes := make([]string, 0, 4)
	argumentos := make([]any, 0, 6)

	if filtro.ProdutoID != nil {
		argumentos = append(argumentos, *filtro.ProdutoID)
		condicoes = append(condicoes, condicao("produto_id", len(argumentos)))
	}
	if filtro.Tipo != "" {
		argumentos = append(argumentos, filtro.Tipo)
		condicoes = append(condicoes, condicao("tipo", len(argumentos)))
	}
	if filtro.LocalID != nil {
		argumentos = append(argumentos, *filtro.LocalID)
		posicao := strconv.Itoa(len(argumentos))
		condicoes = append(condicoes, "(local_id = $"+posicao+" OR local_destino_id = $"+posicao+")")
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, condicaoMovimentoDasEmpresas("local_id", len(argumentos)))
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

func (m *MovimentoRepository) ListarResumo(ctx context.Context, limite int, empresaIDs []uuid.UUID) ([]*domainestoque.MovimentoResumo, error) {
	query := `
		SELECT m.id, m.produto_id, p.nome AS produto_nome, m.tipo, m.quantidade, m.saldo_apos, m.criado_em
		FROM produto_movimento m
		JOIN produto p ON p.id = m.produto_id
	`
	argumentos := []any{limite}
	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` WHERE ` + condicaoMovimentoDasEmpresas("m.local_id", len(argumentos))
	}
	query += ` ORDER BY m.criado_em DESC, m.id DESC LIMIT $1`

	var linhas []*movimentoResumoLinha
	if err := m.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.MovimentoResumo, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioMovimentoResumo(linha))
	}

	return itens, nil
}

func condicaoMovimentoDasEmpresas(coluna string, posicao int) string {
	return "(" + coluna + " IS NULL OR " + coluna +
		" IN (SELECT id FROM estoque_local WHERE empresa_id = ANY($" + strconv.Itoa(posicao) + "::uuid[])))"
}

func inserirMovimento(ctx context.Context, tx *sqlx.Tx, movimento *domainestoque.Movimento) error {
	_, err := tx.NamedExecContext(ctx, `
		INSERT INTO produto_movimento (
			id, produto_id, tipo, quantidade, saldo_apos, local_id, local_destino_id, saldo_local_apos,
			custo_unitario_centavos, reserva_id, usuario_id, documento_ref, observacao, criado_em
		)
		VALUES (
			:id, :produto_id, :tipo, :quantidade, :saldo_apos, :local_id, :local_destino_id, :saldo_local_apos,
			:custo_unitario_centavos, :reserva_id, :usuario_id, :documento_ref, :observacao, :criado_em
		)
	`, paraLinhaMovimento(movimento))

	return err
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
	linha := movimentoLinha{
		ID:             movimento.ID,
		ProdutoID:      movimento.ProdutoID,
		Tipo:           string(movimento.Tipo),
		Quantidade:     movimento.Quantidade.Valor(),
		SaldoApos:      movimento.SaldoApos.Quantidade(),
		LocalID:        movimento.LocalID,
		LocalDestinoID: movimento.LocalDestinoID,
		Custo:          centavosDoCusto(movimento.Custo),
		ReservaID:      movimento.ReservaID,
		UsuarioID:      movimento.UsuarioID,
		DocumentoRef:   movimento.DocumentoRef,
		Observacao:     movimento.Observacao,
		CriadoEm:       movimento.CriadoEm,
	}

	if movimento.SaldoLocalApos != nil {
		saldo := movimento.SaldoLocalApos.Quantidade()
		linha.SaldoLocalApos = &saldo
	}

	return linha
}

func paraDominioMovimento(linha *movimentoLinha) *domainestoque.Movimento {
	movimento := &domainestoque.Movimento{
		ID:             linha.ID,
		ProdutoID:      linha.ProdutoID,
		Tipo:           domainestoque.TipoMovimentoDe(linha.Tipo),
		Quantidade:     domainestoque.QuantidadeDe(linha.Quantidade),
		SaldoApos:      domainestoque.SaldoDe(linha.SaldoApos),
		LocalID:        linha.LocalID,
		LocalDestinoID: linha.LocalDestinoID,
		Custo:          custoDosCentavos(linha.Custo),
		ReservaID:      linha.ReservaID,
		UsuarioID:      linha.UsuarioID,
		DocumentoRef:   linha.DocumentoRef,
		Observacao:     linha.Observacao,
		CriadoEm:       linha.CriadoEm,
	}

	if linha.SaldoLocalApos != nil {
		saldo := domainestoque.SaldoDe(*linha.SaldoLocalApos)
		movimento.SaldoLocalApos = &saldo
	}

	return movimento
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

func centavosDoCusto(custo *domainestoque.Custo) *int64 {
	if custo == nil {
		return nil
	}

	centavos := custo.Centavos()

	return &centavos
}

func custoDosCentavos(centavos *int64) *domainestoque.Custo {
	if centavos == nil {
		return nil
	}

	custo := domainestoque.CustoDe(*centavos)

	return &custo
}
