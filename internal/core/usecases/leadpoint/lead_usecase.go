package leadpoint

import (
	"context"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	"github.com/google/uuid"
)

var _ leads.LeadUseCase = (*LeadUsecaseImpl)(nil)

type LeadUsecaseImpl struct {
	repo      postgres.LeadRepository
	historico postgres.LeadHistoryRepository
}

// CountByEtapa implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) CountByEtapa(ctx context.Context, etapaID uuid.UUID) (int, error) {
	panic("unimplemented")
}

// CountByFunil implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) CountByFunil(ctx context.Context, funilID uuid.UUID) (int, error) {
	panic("unimplemented")
}

// Create implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) Create(ctx context.Context, lead *lead.Lead) (uuid.UUID, error) {
	panic("unimplemented")
}

// Delete implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// GetByID implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*lead.Lead, error) {
	panic("unimplemented")
}

// ListAtivos implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) ListAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	panic("unimplemented")
}

// ListByEtapa implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) ListByEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error) {
	panic("unimplemented")
}

// ListByFunnil implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) ListByFunnil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error) {
	panic("unimplemented")
}

// Search implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) Search(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	panic("unimplemented")
}

// Update implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) Update(ctx context.Context, lead *lead.Lead) error {
	panic("unimplemented")
}

// UpdateEtapa implements [leads.LeadUseCase].
func (l *LeadUsecaseImpl) UpdateEtapa(ctx context.Context, leadID uuid.UUID, newEtapaID uuid.UUID) error {
	panic("unimplemented")
}
