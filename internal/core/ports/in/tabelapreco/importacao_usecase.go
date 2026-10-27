package tabelapreco

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type LinhaImportacao struct {
	Numero     int
	Codigo     string
	Modelo     string
	Preco      string
	VigenteDe  string
	Observacao string
}

type ImportarItensInput struct {
	UsuarioID uuid.UUID
	Linhas    []LinhaImportacao
	Simular   bool
}

type ImportacaoUseCase interface {
	Importar(ctx context.Context, input ImportarItensInput) (*domain.RelatorioImportacao, error)
}
