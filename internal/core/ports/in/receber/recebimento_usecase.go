package receber

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type RepasseInput struct {
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
}

type ReceberInput struct {
	UsuarioID        uuid.UUID
	TituloID         uuid.UUID
	Versao           int
	Tipo             string
	ContaBancariaID  *uuid.UUID
	RecebidoEm       time.Time
	ValorCentavos    int64
	JurosCentavos    int64
	DescontoCentavos int64
	Repasse          *RepasseInput
}

type EstornarRecebimentoInput struct {
	UsuarioID     uuid.UUID
	TituloID      uuid.UUID
	RecebimentoID uuid.UUID
	Versao        int
	Motivo        string
	EstornadoEm   time.Time
}

type RecebimentoUseCase interface {
	Receber(ctx context.Context, input ReceberInput) (*domainreceber.Titulo, *domainreceber.Recebimento, error)
	ListarRecebimentos(ctx context.Context, tituloID, usuarioID uuid.UUID) ([]*domainreceber.Recebimento, error)
	EstornarRecebimento(ctx context.Context, input EstornarRecebimentoInput) (*domainreceber.Titulo, error)
}
