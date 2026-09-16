package postgres

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/jmoiron/sqlx"
)

var _ leads.LeadHistoryRepository = (*LeadHistoryRepository)(nil)

type LeadHistoryRepository struct {
	db *sqlx.DB
}

func NewLeadHistoryRepository(db *sqlx.DB) *LeadHistoryRepository {
	return &LeadHistoryRepository{db: db}
}

// ListByLead implements [leads.LeadHistoryRepository].
func (l *LeadHistoryRepository) ListByLead(ctx context.Context, leadID string, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error) {
	query := `
		SELECT id, lead_id, etapa_anterior_id, etapa_atual_id, movido_em
		FROM lead_historico
		WHERE lead_id = $1
		ORDER BY movido_em DESC
		LIMIT $2 OFFSET $3
	`

	var itens []lead.LeadHistorico
	if err := l.db.SelectContext(ctx, &itens, query, leadID, paginacao.Size, (paginacao.Page-1)*paginacao.Size); err != nil {
		return nil, err
	}
	return itens, nil
}

// RegistrarMovimentacao implements [leads.LeadHistoryRepository].
func (l *LeadHistoryRepository) RegistrarMovimentacao(ctx context.Context, leadID string, etapaAnteriorID string, etapaAtualID string) error {
	query := `
		INSERT INTO lead_historico (id, lead_id, etapa_anterior_id, etapa_atual_id, movido_em)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
	`

	_, err := l.db.ExecContext(ctx, query, leadID, etapaAnteriorID, etapaAtualID)
	return err
}
