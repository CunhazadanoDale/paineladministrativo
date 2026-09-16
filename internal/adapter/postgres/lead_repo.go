package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ leads.LeadRepository = (*LeadRepository)(nil)

type LeadRepository struct {
	db *sqlx.DB
}

func NewLeadRepository(db *sqlx.DB) *LeadRepository {
	return &LeadRepository{db: db}
}

// CountByEtapa implements [leads.LeadRepository].
func (l *LeadRepository) CountByEtapa(ctx context.Context, etapaID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM lead WHERE etapa_id = $1`
	if err := l.db.GetContext(ctx, &count, query, etapaID); err != nil {
		return 0, err
	}
	return count, nil
}

// CountByFunil implements [leads.LeadRepository].
func (l *LeadRepository) CountByFunil(ctx context.Context, funilID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM lead l
		JOIN etapa e ON e.etapa_id = l.etapa_id
		WHERE e.funil_id = $1
	`
	if err := l.db.GetContext(ctx, &count, query, funilID); err != nil {
		return 0, err
	}
	return count, nil
}

// Create implements [leads.LeadRepository].
func (l *LeadRepository) Create(ctx context.Context, lead *lead.Lead) (uuid.UUID, error) {
	query := `
		INSERT INTO lead (id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id)
		VALUES (:id, :nome, :email, :telefone, :ativo, :origem, :criado_em, :atualizado_em, :etapa_id)
		RETURNING id
	`

	rows, err := l.db.NamedQueryContext(ctx, query, lead)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do lead")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

// Delete implements [leads.LeadRepository].
func (l *LeadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := l.db.ExecContext(ctx, `DELETE FROM lead WHERE id = $1`, id)
	return err
}

// GetByID implements [leads.LeadRepository].
func (l *LeadRepository) GetByID(ctx context.Context, id uuid.UUID) (*lead.Lead, error) {
	query := `
		SELECT id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id
		FROM lead
		WHERE id = $1
	`

	var item lead.Lead
	if err := l.db.GetContext(ctx, &item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// ListAtivos implements [leads.LeadRepository].
func (l *LeadRepository) ListAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	query := `
		SELECT id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id
		FROM lead
		WHERE ativo = TRUE
		ORDER BY criado_em DESC
		LIMIT $1 OFFSET $2
	`

	var itens []*lead.Lead
	if err := l.db.SelectContext(ctx, &itens, query, paginacao.Size, (paginacao.Page-1)*paginacao.Size); err != nil {
		return nil, err
	}
	return itens, nil
}

// ListByEtapa implements [leads.LeadRepository].
func (l *LeadRepository) ListByEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error) {
	query := `
		SELECT id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id
		FROM lead
		WHERE etapa_id = $1
		ORDER BY atualizado_em DESC
	`

	var itens []*lead.Lead
	if err := l.db.SelectContext(ctx, &itens, query, etapaID); err != nil {
		return nil, err
	}
	return itens, nil
}

// ListByFunnil implements [leads.LeadRepository].
func (l *LeadRepository) ListByFunnil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error) {
	query := `
		SELECT l.id, l.nome, l.email, l.telefone, l.ativo, l.origem, l.criado_em, l.atualizado_em, l.etapa_id
		FROM lead l
		JOIN etapa e ON e.etapa_id = l.etapa_id
		WHERE e.funil_id = $1
		ORDER BY l.atualizado_em DESC
	`

	var itens []*lead.Lead
	if err := l.db.SelectContext(ctx, &itens, query, funilID); err != nil {
		return nil, err
	}
	return itens, nil
}

// Search implements [leads.LeadRepository].
func (l *LeadRepository) Search(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	searchQuery := `
		SELECT id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id
		FROM lead
		WHERE ativo = TRUE
		  AND (
			LOWER(nome) LIKE '%' || LOWER($1) || '%'
			OR LOWER(email) LIKE '%' || LOWER($1) || '%'
			OR telefone LIKE '%' || $1 || '%'
		  )
		ORDER BY criado_em DESC
		LIMIT $2 OFFSET $3
	`

	var itens []*lead.Lead
	if err := l.db.SelectContext(ctx, &itens, searchQuery, query, paginacao.Size, (paginacao.Page-1)*paginacao.Size); err != nil {
		return nil, err
	}
	return itens, nil
}

// Update implements [leads.LeadRepository].
func (l *LeadRepository) Update(ctx context.Context, lead *lead.Lead) error {
	query := `
		UPDATE lead
		SET nome = :nome,
		    email = :email,
		    telefone = :telefone,
		    ativo = :ativo,
		    origem = :origem,
		    atualizado_em = :atualizado_em,
		    etapa_id = :etapa_id
		WHERE id = :id
	`

	_, err := l.db.NamedExecContext(ctx, query, lead)
	return err
}

// UpdateEtapa implements [leads.LeadRepository].
func (l *LeadRepository) UpdateEtapa(ctx context.Context, leadID uuid.UUID, newEtapaID uuid.UUID) error {
	query := `
		UPDATE lead
		SET etapa_id = $1,
		    atualizado_em = NOW()
		WHERE id = $2
	`

	_, err := l.db.ExecContext(ctx, query, newEtapaID, leadID)
	return err
}
