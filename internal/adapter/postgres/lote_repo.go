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

var _ portsout.LoteRepository = (*LoteRepository)(nil)

type LoteRepository struct {
	db *sqlx.DB
}

func NewLoteRepository(db *sqlx.DB) *LoteRepository {
	return &LoteRepository{db: db}
}

type loteLinha struct {
	ID                uuid.UUID  `db:"id"`
	ProdutoID         uuid.UUID  `db:"produto_id"`
	LocalID           uuid.UUID  `db:"local_id"`
	MovimentoID       uuid.UUID  `db:"movimento_id"`
	LoteOrigemID      *uuid.UUID `db:"lote_origem_id"`
	Custo             *int64     `db:"custo_unitario_centavos"`
	QuantidadeInicial int        `db:"quantidade_inicial"`
	QuantidadeAtual   int        `db:"quantidade_atual"`
	EntradaEm         time.Time  `db:"entrada_em"`
	CriadoEm          time.Time  `db:"criado_em"`
}

const colunasLote = `
	id, produto_id, local_id, movimento_id, lote_origem_id, custo_unitario_centavos,
	quantidade_inicial, quantidade_atual, entrada_em, criado_em
`

func (l *LoteRepository) Listar(ctx context.Context, filtro portsout.LoteFiltro) ([]*domainestoque.Lote, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasLote + ` FROM produto_lote`
	argumentos := []any{filtro.ProdutoID}
	condicoes := []string{condicao("produto_id", 1)}

	if filtro.LocalID != nil {
		argumentos = append(argumentos, *filtro.LocalID)
		condicoes = append(condicoes, condicao("local_id", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes,
			"local_id IN (SELECT id FROM estoque_local WHERE empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[]))")
	}
	if filtro.ComSaldo {
		condicoes = append(condicoes, "quantidade_atual > 0")
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += " WHERE " + unirCondicoes(condicoes) + ` ORDER BY entrada_em ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*loteLinha
	if err := l.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Lote, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioLote(linha))
	}

	return itens, nil
}

func paraLinhaLote(lote *domainestoque.Lote) loteLinha {
	return loteLinha{
		ID:                lote.ID,
		ProdutoID:         lote.ProdutoID,
		LocalID:           lote.LocalID,
		MovimentoID:       lote.MovimentoID,
		LoteOrigemID:      lote.LoteOrigemID,
		Custo:             centavosDoCusto(lote.Custo),
		QuantidadeInicial: lote.QuantidadeInicial,
		QuantidadeAtual:   lote.QuantidadeAtual,
		EntradaEm:         lote.EntradaEm,
		CriadoEm:          lote.CriadoEm,
	}
}

func paraDominioLote(linha *loteLinha) *domainestoque.Lote {
	return &domainestoque.Lote{
		ID:                linha.ID,
		ProdutoID:         linha.ProdutoID,
		LocalID:           linha.LocalID,
		MovimentoID:       linha.MovimentoID,
		LoteOrigemID:      linha.LoteOrigemID,
		Custo:             custoDosCentavos(linha.Custo),
		QuantidadeInicial: linha.QuantidadeInicial,
		QuantidadeAtual:   linha.QuantidadeAtual,
		EntradaEm:         linha.EntradaEm,
		CriadoEm:          linha.CriadoEm,
	}
}
