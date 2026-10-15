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
	SolicitanteID         uuid.UUID
	EmpresaID             uuid.UUID
	ValorCentavos         int64
	PrazoPagamento        time.Time
	Observacao            string
	FormaPagamento        string
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID *uuid.UUID
	Competencia           string
	Estimado              bool
	ArquivoIDs            []uuid.UUID
}

type EditarSolicitacaoInput struct {
	SolicitacaoID         uuid.UUID
	UsuarioID             uuid.UUID
	ValorCentavos         int64
	PrazoPagamento        time.Time
	Observacao            string
	FormaPagamento        string
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID *uuid.UUID
	Competencia           string
	Estimado              bool
	Versao                int
}

type AdiarSolicitacaoInput struct {
	SolicitacaoID uuid.UUID
	UsuarioID     uuid.UUID
	Vencimento    time.Time
	Motivo        string
	Versao        int
}

type ClassificarSolicitacaoInput struct {
	SolicitacaoID         uuid.UUID
	UsuarioID             uuid.UUID
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID *uuid.UUID
	Competencia           string
	Versao                int
}

type RegistrarPagamentoInput struct {
	SolicitacaoID        uuid.UUID
	UsuarioID            uuid.UUID
	ContaBancariaID      *uuid.UUID
	ValorCentavos        int64
	ComprovanteArquivoID *uuid.UUID
	PagoEm               time.Time
}

type EstornarPagamentoInput struct {
	SolicitacaoID uuid.UUID
	UsuarioID     uuid.UUID
	Motivo        string
	EstornadoEm   time.Time
	Versao        int
}

type ListarSolicitacoesInput struct {
	UsuarioID   uuid.UUID
	Escopo      string
	Status      string
	Empresa     *uuid.UUID
	Fornecedor  *uuid.UUID
	Categoria   *uuid.UUID
	Competencia string
	Origem      string
	VenceDe     *time.Time
	VenceAte    *time.Time
	Vencidas    bool
	Filtro      domain.PaginacaoFiltro
}

type SolicitacaoUseCase interface {
	Criar(ctx context.Context, input CriarSolicitacaoInput) (uuid.UUID, error)
	Obter(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Solicitacao, error)
	Listar(ctx context.Context, input ListarSolicitacoesInput) ([]*domainsolicitacao.Solicitacao, error)
	Aprovar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID) error
	AprovarNaVersao(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, versao int) error
	Rejeitar(ctx context.Context, solicitacaoID, aprovadorID uuid.UUID, motivo string) error
	Cancelar(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) error
	Classificar(ctx context.Context, input ClassificarSolicitacaoInput) error
	Editar(ctx context.Context, input EditarSolicitacaoInput) error
	Adiar(ctx context.Context, input AdiarSolicitacaoInput) error
	RegistrarPagamento(ctx context.Context, input RegistrarPagamentoInput) error
	ListarHistorico(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Historico, error)
	ListarArquivos(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Arquivo, error)
	ObterPagamento(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) (*domainsolicitacao.Pagamento, error)
	ListarPagamentos(ctx context.Context, solicitacaoID, usuarioID uuid.UUID) ([]*domainsolicitacao.Pagamento, error)
	EstornarPagamento(ctx context.Context, input EstornarPagamentoInput) error
	Pendencias(ctx context.Context, usuarioID uuid.UUID) (domainsolicitacao.Pendencias, error)
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
