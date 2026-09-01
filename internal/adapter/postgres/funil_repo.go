package postgres

import (
	"context"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ leads.FunilRepository = (*FunilRepo)(nil)

type FunilRepo struct {
	db *sqlx.DB
}

func NewFunilRepo(db *sqlx.DB) *FunilRepo {
	return &FunilRepo{
		db: db,
	}
}

// Create implements [leads.FunilRepository].
func (f *FunilRepo) Create(ctx context.Context, funil *lead.Funil) (uuid.UUID, error) {
	panic("unimplemented")
}

// Delete implements [leads.FunilRepository].
func (f *FunilRepo) Delete(ctx context.Context, funilID uuid.UUID) error {
	panic("unimplemented")
}

// ExistsByID implements [leads.FunilRepository].
func (f *FunilRepo) ExistsByID(ctx context.Context, funilID uuid.UUID) (bool, error) {
	panic("unimplemented")
}

// GetByID implements [leads.FunilRepository].
func (f *FunilRepo) GetByID(ctx context.Context, funilID uuid.UUID) (*lead.Funil, error) {
	panic("unimplemented")
}

// List implements [leads.FunilRepository].
func (f *FunilRepo) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	panic("unimplemented")
}

// ListAtivos implements [leads.FunilRepository].
func (f *FunilRepo) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	panic("unimplemented")
}

// Update implements [leads.FunilRepository].
func (f *FunilRepo) Update(ctx context.Context, funil *lead.Funil) error {
	panic("unimplemented")
}
