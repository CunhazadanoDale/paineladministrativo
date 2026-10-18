package notafiscal

import (
	"context"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	"github.com/google/uuid"
)

type ImportarEventoInput struct {
	UsuarioID uuid.UUID
	ArquivoID uuid.UUID
}

type ImportacaoEvento struct {
	Evento            *domainnotafiscal.Evento
	Nota              *domainnotafiscal.NotaFiscal
	Avisos            []string
	TitulosCancelados []uuid.UUID
	TitulosMarcados   []uuid.UUID
	ErroDosTitulos    error
}

type EventoUseCase interface {
	Importar(ctx context.Context, input ImportarEventoInput) (*ImportacaoEvento, error)
	Listar(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) ([]*domainnotafiscal.Evento, error)
}
