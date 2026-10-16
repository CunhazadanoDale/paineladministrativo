package receber

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type Alteracao struct {
	Titulo            *domainreceber.Titulo
	VersaoLida        int
	StatusAnterior    domainreceber.Status
	Historico         *domainreceber.Historico
	ConferirCategoria bool
	ConferirConta     bool
}

type EntradaDoPedido struct {
	Pedido         *domainpedido.Pedido
	VersaoLida     int
	StatusAnterior domainpedido.Status
	Historico      *domainpedido.Historico
}

type Baixa struct {
	Titulo         *domainreceber.Titulo
	VersaoLida     int
	StatusAnterior domainreceber.Status
	Recebimento    *domainreceber.Recebimento
	Historico      *domainreceber.Historico
	Entrada        *EntradaDoPedido
	ContaDoRepasse *ContaAPagarGerada
}

type EstornoDaBaixa struct {
	Titulo         *domainreceber.Titulo
	VersaoLida     int
	StatusAnterior domainreceber.Status
	Recebimento    *domainreceber.Recebimento
	Historico      *domainreceber.Historico
}

type TituloFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	ClienteID       *uuid.UUID
	Status          domainreceber.Status
	Origem          domainreceber.Origem
	OrigemID        *uuid.UUID
	ContaBancariaID *uuid.UUID
	VencimentoDe    *time.Time
	VencimentoAte   *time.Time
	VencidosEm      *time.Time
	OrigemCancelada bool
}

type TituloReceberRepository interface {
	Criar(ctx context.Context, titulo *domainreceber.Titulo, historico *domainreceber.Historico) error
	Atualizar(ctx context.Context, alteracao *Alteracao) error
	Obter(ctx context.Context, id uuid.UUID) (*domainreceber.Titulo, error)
	Listar(ctx context.Context, filtro TituloFiltro) ([]*domainreceber.Titulo, error)
	ListarHistorico(ctx context.Context, tituloID uuid.UUID) ([]*domainreceber.Historico, error)
	GerarDeOrigem(ctx context.Context, geracao *domainreceber.Geracao) (domainreceber.ResultadoGeracao, error)
	CancelarPorOrigem(ctx context.Context, cancelamento domainreceber.CancelamentoDeOrigem) (domainreceber.ResultadoCancelamento, error)
	Receber(ctx context.Context, baixa *Baixa) error
	EstornarRecebimento(ctx context.Context, estorno *EstornoDaBaixa) error
	ListarRecebimentos(ctx context.Context, tituloID uuid.UUID) ([]*domainreceber.Recebimento, error)
}
