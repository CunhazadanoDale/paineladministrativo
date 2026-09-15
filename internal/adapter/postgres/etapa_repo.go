package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ leads.EtapaRepository = (*EtapaRepository)(nil)

type EtapaRepository struct {
	db *sqlx.DB
}

func NewEtapaRepository(db *sqlx.DB) *EtapaRepository {
	return &EtapaRepository{db: db}
}

func (e *EtapaRepository) Create(ctx context.Context, etapa *lead.Etapa) (uuid.UUID, error) {
	query := `
		INSERT INTO etapa (etapa_id, nome, ordem, funil_id, ativo)
		VALUES (:etapa_id, :nome, :ordem, :funil_id, :ativo)
		RETURNING etapa_id
	`

	rows, err := e.db.NamedQueryContext(ctx, query, etapa)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação da etapa")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (e *EtapaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := e.db.ExecContext(ctx, `DELETE FROM etapa WHERE etapa_id = $1`, id)
	return err
}

func (e *EtapaRepository) ExistsByFunil(ctx context.Context, funilID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM etapa WHERE funil_id = $1)`
	if err := e.db.GetContext(ctx, &exists, query, funilID); err != nil {
		return false, err
	}
	return exists, nil
}

func (e *EtapaRepository) GetByID(ctx context.Context, id uuid.UUID) (*lead.Etapa, error) {
	query := `SELECT etapa_id, nome, ordem, funil_id, ativo FROM etapa WHERE etapa_id = $1`
	var etapa lead.Etapa
	if err := e.db.GetContext(ctx, &etapa, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &etapa, nil
}

func (e *EtapaRepository) GetNextEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	query := `
		SELECT e.etapa_id, e.nome, e.ordem, e.funil_id, e.ativo
		FROM etapa e
		JOIN etapa atual ON atual.funil_id = e.funil_id
		WHERE atual.etapa_id = $1
		  AND e.ordem > atual.ordem
		ORDER BY e.ordem ASC
		LIMIT 1
	`

	var etapa lead.Etapa
	if err := e.db.GetContext(ctx, &etapa, query, currentEtapaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &etapa, nil
}

func (e *EtapaRepository) GetPreviousEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	query := `
		SELECT e.etapa_id, e.nome, e.ordem, e.funil_id, e.ativo
		FROM etapa e
		JOIN etapa atual ON atual.funil_id = e.funil_id
		WHERE atual.etapa_id = $1
		  AND e.ordem < atual.ordem
		ORDER BY e.ordem DESC
		LIMIT 1
	`

	var etapa lead.Etapa
	if err := e.db.GetContext(ctx, &etapa, query, currentEtapaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &etapa, nil
}

func (e *EtapaRepository) ListByFunilID(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	query := `
		SELECT etapa_id, nome, ordem, funil_id, ativo
		FROM etapa
		WHERE funil_id = $1
		ORDER BY ordem ASC
	`

	var etapas []*lead.Etapa
	if err := e.db.SelectContext(ctx, &etapas, query, funilID); err != nil {
		return nil, err
	}
	return etapas, nil
}

func (e *EtapaRepository) ListByFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	return e.ListByFunilID(ctx, funilID)
}

func (e *EtapaRepository) Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*lead.Etapa) error {
	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	for i, etapa := range etapas {
		if etapa == nil {
			return fmt.Errorf("etapa %d é nula", i)
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE etapa SET ordem = $1 WHERE etapa_id = $2 AND funil_id = $3`,
			i+1,
			etapa.EtapaID,
			funilID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *EtapaRepository) Update(ctx context.Context, etapa *lead.Etapa) error {
	query := `
		UPDATE etapa
		SET nome = :nome, ordem = :ordem, funil_id = :funil_id, ativo = :ativo
		WHERE etapa_id = :etapa_id
	`

	_, err := e.db.NamedExecContext(ctx, query, etapa)
	return err
}
