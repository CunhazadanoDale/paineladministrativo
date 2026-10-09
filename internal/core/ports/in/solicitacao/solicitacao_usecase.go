package solicitacao

import (
	"context"
	"io"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

const (
	EscopoMinhas     = "minhas"
	EscopoAprovacao  = "aprovacao"
	EscopoFinanceiro = "financeiro"
	EscopoTodas      = "todas"

	TamanhoMaximoArquivo = 10 << 20
)

type CriarSolicitacaoInput struct {
	SolicitanteID  uuid.UUID
	ValorCentavos  int64
	PrazoPagamento time.Time
	Observacao     string
	FormaPagamento string
	ArquivoIDs     []uuid.UUID
}

type RegistrarPagamentoInput struct {
	SolicitacaoID        uuid.UUID
	UsuarioID            uuid.UUID
	ValorCentavos        int64
	ComprovanteArquivoID *uuid.UUID
	PagoEm               time.Time
}

type ListarSolicitacoesInput struct {
	UsuarioID uuid.UUID
	Escopo    string
	Status    string
	Filtro    domain.PaginacaoFiltro
}

type SolicitacaoUseCase interface {
	Criar(ctx context.Context, input CriarSolicitacaoInput) (uuid.UUID, error)
	Obter(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Solicitacao, error)
	Listar(ctx context.Context, input ListarSolicitacoesInput) ([]*domainsolicitacao.Solicitacao, error)
	Aprovar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID) error
	Rejeitar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, motivo string) error
	Cancelar(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) error
	RegistrarPagamento(ctx context.Context, input RegistrarPagamentoInput) error
	ListarHistorico(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Historico, error)
	ListarArquivos(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Arquivo, error)
	ObterPagamento(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) (*domainsolicitacao.Pagamento, error)
}

type ArquivoUseCase interface {
	Enviar(ctx context.Context, proprietarioID uuid.UUID, nome, contentType string, tamanho int64, conteudo io.Reader) (*domainsolicitacao.Arquivo, error)
	Baixar(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error)
	BaixarPublico(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error)
	ListarPorProprietario(ctx context.Context, proprietarioID uuid.UUID, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error)
	Remover(ctx context.Context, id, usuarioID uuid.UUID) error
}

type AprovadorUseCase interface {
	Designar(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error)
	Remover(ctx context.Context, id uuid.UUID) error
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error)
	EhDesignado(ctx context.Context, usuarioID uuid.UUID) (bool, error)
}
