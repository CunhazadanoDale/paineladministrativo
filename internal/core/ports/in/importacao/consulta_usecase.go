package importacao

import (
	"context"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type AlertasInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	FolgaDias *int
}

type ConsultaUseCase interface {
	Alertas(ctx context.Context, input AlertasInput) (*domainimportacao.Alertas, error)
	ImportacoesDoPedido(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]domainimportacao.AtendimentoDoPedido, error)
}
