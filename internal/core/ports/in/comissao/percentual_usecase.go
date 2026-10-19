package comissao

import (
	"context"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type DefinirPercentualInput struct {
	UsuarioID  uuid.UUID
	PedidoID   uuid.UUID
	PontosBase int
	Versao     *int
}

type PercentualUseCase interface {
	Obter(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domaincomissao.PercentualPedido, error)
	Definir(ctx context.Context, input DefinirPercentualInput) (*domaincomissao.PercentualPedido, error)
}
