package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type SolicitacaoFiltro struct {
	domain.PaginacaoFiltro
	Status      domainsolicitacao.Status
	Solicitante *uuid.UUID
	Aprovador   *uuid.UUID
}

type SolicitacaoRepository interface {
	Criar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, arquivoIDs []uuid.UUID, historico *domainsolicitacao.Historico) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Solicitacao, error)
	Listar(ctx context.Context, filtro SolicitacaoFiltro) ([]*domainsolicitacao.Solicitacao, error)
	AtualizarStatus(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error)
	CriarPagamento(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error
	ListarArquivos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Arquivo, error)
	ListarHistorico(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Historico, error)
	ObterPagamento(ctx context.Context, solicitacaoID uuid.UUID) (*domainsolicitacao.Pagamento, error)
}
