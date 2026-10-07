package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ArquivoRepository interface {
	Create(ctx context.Context, arquivo *domainsolicitacao.Arquivo) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error)
	ListByProprietario(ctx context.Context, proprietarioID uuid.UUID, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error)
	VinculadoASolicitacao(ctx context.Context, arquivoID uuid.UUID) (bool, error)
	SolicitacaoDoArquivo(ctx context.Context, arquivoID uuid.UUID) (*uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
