package postgres

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ leads.LeadHistoryRepository = (*LeadHistoryRepository)(nil)

type LeadHistoryRepository struct {
	db *sqlx.DB
}

func NewLeadHistoryRepository(db *sqlx.DB) *LeadHistoryRepository {
	return &LeadHistoryRepository{db: db}
}

func (l *LeadHistoryRepository) ListByLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error) {
	query := `
		SELECT id, lead_id, etapa_anterior_id, etapa_atual_id, movido_em
		FROM lead_historico
		WHERE lead_id = $1
		ORDER BY movido_em DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	var itens []lead.LeadHistorico
	if err := l.db.SelectContext(ctx, &itens, query, leadID, paginacao.Size, (paginacao.Page-1)*paginacao.Size); err != nil {
		return nil, err
	}
	return itens, nil
}
