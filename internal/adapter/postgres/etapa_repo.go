package postgres

import (
	"context"
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
	return &EtapaRepository{
		db: db,
	}
}


func (e *EtapaRepository) Create(ctx context.Context, etapa *lead.Etapa) (uuid.UUID, error) {
	panic("unimplemented")
}

// Delete implements [leads.EtapaRepository].
func (e *EtapaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// ExistsByFunil implements [leads.EtapaRepository].
func (e *EtapaRepository) ExistsByFunil(ctx context.Context, funilID uuid.UUID) (bool, error) {
	panic("unimplemented")
}

// GetByID implements [leads.EtapaRepository].
func (e *EtapaRepository) GetByID(ctx context.Context, id uuid.UUID) (*lead.Etapa, error) {
	panic("unimplemented")
}

// GetNextEtapa implements [leads.EtapaRepository].
func (e *EtapaRepository) GetNextEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	panic("unimplemented")
}

// GetPreviousEtapa implements [leads.EtapaRepository].
func (e *EtapaRepository) GetPreviousEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	panic("unimplemented")
}

// ListByFunilID implements [leads.EtapaRepository].
func (e *EtapaRepository) ListByFunilID(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	panic("unimplemented")
}

// ListByFunilOrdenado implements [leads.EtapaRepository].
func (e *EtapaRepository) ListByFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	panic("unimplemented")
}

// Reordenar implements [leads.EtapaRepository].
func (e *EtapaRepository) Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*lead.Etapa) error {
	panic("unimplemented")
}

// Update implements [leads.EtapaRepository].
func (e *EtapaRepository) Update(ctx context.Context, etapa *lead.Etapa) error {
	panic("unimplemented")
}
