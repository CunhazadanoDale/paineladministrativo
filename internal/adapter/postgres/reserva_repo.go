package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ReservaRepository = (*ReservaRepository)(nil)

type ReservaRepository struct {
	db *sqlx.DB
}

func NewReservaRepository(db *sqlx.DB) *ReservaRepository {
	return &ReservaRepository{db: db}
}

type reservaLinha struct {
	ID          uuid.UUID  `db:"id"`
	ProdutoID   uuid.UUID  `db:"produto_id"`
	LocalID     uuid.UUID  `db:"local_id"`
	Quantidade  int        `db:"quantidade"`
	Origem      string     `db:"origem"`
	OrigemID    *uuid.UUID `db:"origem_id"`
	Status      string     `db:"status"`
	UsuarioID   uuid.UUID  `db:"usuario_id"`
	Observacao  string     `db:"observacao"`
	CriadoEm    time.Time  `db:"criado_em"`
	EncerradaEm *time.Time `db:"encerrada_em"`
}

const colunasReserva = `
	id, produto_id, local_id, quantidade, origem, origem_id, status, usuario_id, observacao, criado_em, encerrada_em
`

func (r *ReservaRepository) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Reserva, error) {
	var linha reservaLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasReserva+` FROM produto_reserva WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioReserva(&linha), nil
}

func (r *ReservaRepository) Listar(ctx context.Context, filtro portsout.ReservaFiltro) ([]*domainestoque.Reserva, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	argumentos := []any{filtro.ProdutoID}
	condicoes := []string{condicao("produto_id", 1)}

	if filtro.LocalID != nil {
		argumentos = append(argumentos, *filtro.LocalID)
		condicoes = append(condicoes, condicao("local_id", len(argumentos)))
	}
	if filtro.Status != "" {
		argumentos = append(argumentos, filtro.Status)
		condicoes = append(condicoes, condicao("status", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes,
			"local_id IN (SELECT id FROM estoque_local WHERE empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[]))")
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query := `SELECT ` + colunasReserva + ` FROM produto_reserva WHERE ` + unirCondicoes(condicoes) +
		` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*reservaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainestoque.Reserva, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioReserva(linha))
	}

	return itens, nil
}

func inserirReserva(ctx context.Context, tx *sqlx.Tx, reserva *domainestoque.Reserva) error {
	_, err := tx.NamedExecContext(ctx, `
		INSERT INTO produto_reserva (
			id, produto_id, local_id, quantidade, origem, origem_id, status, usuario_id, observacao, criado_em, encerrada_em
		)
		VALUES (
			:id, :produto_id, :local_id, :quantidade, :origem, :origem_id, :status, :usuario_id, :observacao, :criado_em, :encerrada_em
		)
	`, paraLinhaReserva(reserva))

	return err
}

func encerrarReserva(ctx context.Context, tx *sqlx.Tx, reserva *domainestoque.Reserva) error {
	resultado, err := tx.ExecContext(ctx, `
		UPDATE produto_reserva
		SET status = $1,
		    encerrada_em = $2
		WHERE id = $3
		  AND status = $4
	`, reserva.Status, reserva.EncerradaEm, reserva.ID, domainestoque.StatusReservaAtiva)
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domain.ErroConflito("reserva já encerrada")
	}

	return nil
}

func paraLinhaReserva(reserva *domainestoque.Reserva) reservaLinha {
	return reservaLinha{
		ID:          reserva.ID,
		ProdutoID:   reserva.ProdutoID,
		LocalID:     reserva.LocalID,
		Quantidade:  reserva.Quantidade,
		Origem:      reserva.Origem.String(),
		OrigemID:    reserva.OrigemID,
		Status:      reserva.Status.String(),
		UsuarioID:   reserva.UsuarioID,
		Observacao:  reserva.Observacao,
		CriadoEm:    reserva.CriadoEm,
		EncerradaEm: reserva.EncerradaEm,
	}
}

func paraDominioReserva(linha *reservaLinha) *domainestoque.Reserva {
	return &domainestoque.Reserva{
		ID:          linha.ID,
		ProdutoID:   linha.ProdutoID,
		LocalID:     linha.LocalID,
		Quantidade:  linha.Quantidade,
		Origem:      domainestoque.OrigemReservaDe(linha.Origem),
		OrigemID:    linha.OrigemID,
		Status:      domainestoque.StatusReservaDe(linha.Status),
		UsuarioID:   linha.UsuarioID,
		Observacao:  linha.Observacao,
		CriadoEm:    linha.CriadoEm,
		EncerradaEm: linha.EncerradaEm,
	}
}
