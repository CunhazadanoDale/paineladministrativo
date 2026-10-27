package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type LinhaImportacao struct {
	Numero   int
	Nome     string
	Email    string
	Telefone string
	Origem   string
}

type ImportarLeadsInput struct {
	EtapaID uuid.UUID
	Linhas  []LinhaImportacao
	Simular bool
}

type ImportacaoUseCase interface {
	Importar(ctx context.Context, input ImportarLeadsInput) (*domain.RelatorioImportacao, error)
}
