package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadImportacaoRepository interface {
	Criar(ctx context.Context, lead *lead.Lead) (uuid.UUID, error)
	EmailsCadastrados(ctx context.Context, emails []string) (map[string]bool, error)
	TelefonesCadastrados(ctx context.Context, telefones []string) (map[string]bool, error)
}
