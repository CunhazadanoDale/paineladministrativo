package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type guardaDeVersao struct {
	tabela        string
	naoEncontrado func() error
	alterado      func() error
	conflito      func(restricao string) error
}

func (g guardaDeVersao) traduzir(err error) error {
	var erroPostgres *pq.Error
	if g.conflito != nil && errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
		if conflito := g.conflito(erroPostgres.Constraint); conflito != nil {
			return conflito
		}
	}

	return tratarErroDeGravacao(err)
}

func (g guardaDeVersao) atualizar(ctx context.Context, consulta sqlx.QueryerContext, id uuid.UUID, query string, argumentos ...any) (int, error) {
	var novaVersao int
	err := sqlx.GetContext(ctx, consulta, &novaVersao, query, argumentos...)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, g.motivoSemAlteracao(ctx, consulta, id)
	}
	if err != nil {
		return 0, g.traduzir(err)
	}

	return novaVersao, nil
}

func (g guardaDeVersao) conferirVersaoTravada(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, versaoLida int) error {
	var versaoAtual int
	err := tx.GetContext(ctx, &versaoAtual, `SELECT versao FROM `+g.tabela+` WHERE id = $1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return g.naoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if versaoAtual != versaoLida {
		return g.alterado()
	}

	return nil
}

func (g guardaDeVersao) remover(ctx context.Context, db *sqlx.DB, id uuid.UUID) error {
	resultado, err := db.ExecContext(ctx, `DELETE FROM `+g.tabela+` WHERE id = $1`, id)
	if err != nil {
		return tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErro(err)
	}
	if linhas == 0 {
		return g.naoEncontrado()
	}

	return nil
}

func (g guardaDeVersao) motivoSemAlteracao(ctx context.Context, consulta sqlx.QueryerContext, id uuid.UUID) error {
	var existe bool
	if err := sqlx.GetContext(ctx, consulta, &existe, `SELECT EXISTS (SELECT 1 FROM `+g.tabela+` WHERE id = $1)`, id); err != nil {
		return tratarErroDeGravacao(err)
	}
	if !existe {
		return g.naoEncontrado()
	}

	return g.alterado()
}
