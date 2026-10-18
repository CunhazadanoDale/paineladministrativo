package notafiscal

import (
	"context"
	"io"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

const (
	AvisoNumeroPedidoAusente = "numero_pedido_ausente"
	AvisoItensSemProduto     = "itens_sem_produto"
	AvisoClienteCriado       = "cliente_criado"
	AvisoTitulosPendentes    = "titulos_pendentes"
)

type ImportarInput struct {
	UsuarioID uuid.UUID
	ArquivoID uuid.UUID
	PedidoID  *uuid.UUID
	EntregaID *uuid.UUID
}

type Importacao struct {
	Nota           *domainnotafiscal.NotaFiscal
	Avisos         []string
	TitulosGerados []uuid.UUID
	ErroDosTitulos error
}

type SincronizacaoTitulos struct {
	Gerados      []uuid.UUID
	JaExistentes []uuid.UUID
	Cancelados   []uuid.UUID
	Marcados     []uuid.UUID
	Ignorados    int
}

type VincularInput struct {
	UsuarioID    uuid.UUID
	NotaFiscalID uuid.UUID
	Versao       int
	PedidoID     *uuid.UUID
	EntregaID    *uuid.UUID
}

type VincularProdutoInput struct {
	UsuarioID    uuid.UUID
	NotaFiscalID uuid.UUID
	NumeroItem   int
	Versao       int
	ProdutoID    uuid.UUID
}

type ListarNotasInput struct {
	UsuarioID     uuid.UUID
	Empresa       *uuid.UUID
	ClienteID     *uuid.UUID
	PedidoID      *uuid.UUID
	Natureza      string
	Status        string
	EntreEmpresas *bool
	SemPedido     bool
	EmitidaDe     *time.Time
	EmitidaAte    *time.Time
	Numero        *int
	Chave         string
	Filtro        domain.PaginacaoFiltro
}

type ListarItensSemProdutoInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Filtro    domain.PaginacaoFiltro
}

type ListarEntregasSemNotaInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	ClienteID *uuid.UUID
	Filtro    domain.PaginacaoFiltro
}

type NotaFiscalUseCase interface {
	Importar(ctx context.Context, input ImportarInput) (*Importacao, error)
	Obter(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*domainnotafiscal.NotaFiscal, error)
	Listar(ctx context.Context, input ListarNotasInput) ([]*domainnotafiscal.NotaFiscal, error)
	Vincular(ctx context.Context, input VincularInput) (*domainnotafiscal.NotaFiscal, error)
	VincularProduto(ctx context.Context, input VincularProdutoInput) (*domainnotafiscal.NotaFiscal, error)
	BaixarXML(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error)
	ListarItensSemProduto(ctx context.Context, input ListarItensSemProdutoInput) ([]*domainnotafiscal.ItemPendente, error)
	ListarEntregasSemNota(ctx context.Context, input ListarEntregasSemNotaInput) ([]*domainnotafiscal.EntregaSemNota, error)
	SincronizarTitulos(ctx context.Context, notaFiscalID, usuarioID uuid.UUID) (*SincronizacaoTitulos, error)
}
