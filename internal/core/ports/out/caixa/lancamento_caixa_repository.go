package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type LancamentoCaixaFiltro struct {
	EmpresaIDs      []uuid.UUID
	ContaBancariaID *uuid.UUID
	De              *time.Time
	Ate             *time.Time
	Limite          int
	Deslocamento    int
}

type LancamentoCaixaRepository interface {
	Classificar(ctx context.Context, classificacao *domaincaixa.Classificacao) error
	Transferir(ctx context.Context, registrada *domaincaixa.TransferenciaRegistrada) error
	Listar(ctx context.Context, filtro LancamentoCaixaFiltro) ([]*domaincaixa.LancamentoCaixa, error)
	ListarTransferencias(ctx context.Context, filtro LancamentoCaixaFiltro) ([]*domaincaixa.Transferencia, error)
}
