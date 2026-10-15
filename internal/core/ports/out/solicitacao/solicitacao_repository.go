package solicitacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type SolicitacaoFiltro struct {
	domain.PaginacaoFiltro
	Status      domainsolicitacao.Status
	Solicitante *uuid.UUID
	Empresa     *uuid.UUID
	EmpresaIDs  []uuid.UUID
	Fornecedor  *uuid.UUID
	Categoria   *uuid.UUID
	Competencia *domainsolicitacao.Competencia
	Origem      domainsolicitacao.Origem
	OrigemID    *uuid.UUID
	VenceDe     *time.Time
	VenceAte    *time.Time
	VencidasEm  *time.Time
}

type SolicitacaoRepository interface {
	Criar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, arquivoIDs []uuid.UUID, historico *domainsolicitacao.Historico) (uuid.UUID, error)
	GerarDeRecorrencia(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Solicitacao, error)
	Listar(ctx context.Context, filtro SolicitacaoFiltro) ([]*domainsolicitacao.Solicitacao, error)
	Contar(ctx context.Context, filtro SolicitacaoFiltro) (int, error)
	AtualizarStatus(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error)
	Classificar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error
	Editar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error
	Adiar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error
	CriarPagamento(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error
	ListarArquivos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Arquivo, error)
	ListarHistorico(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Historico, error)
	ObterPagamento(ctx context.Context, solicitacaoID uuid.UUID) (*domainsolicitacao.Pagamento, error)
	ListarPagamentos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Pagamento, error)
	Estornar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error
}
