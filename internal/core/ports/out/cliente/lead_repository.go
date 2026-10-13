package cliente

import (
	"context"

	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainlead.Lead, error)
}
