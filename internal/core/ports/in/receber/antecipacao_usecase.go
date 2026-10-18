package receber

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type CriarAntecipacaoInput struct {
	UsuarioID            uuid.UUID
	EmpresaID            uuid.UUID
	FornecedorID         uuid.UUID
	Numero               *string
	OperadoEm            time.Time
	TaxaBP               int
	ValorLiquidoCentavos int64
	ContaBancariaID      uuid.UUID
	TituloIDs            []uuid.UUID
}

type ListarAntecipacoesInput struct {
	UsuarioID    uuid.UUID
	Empresa      *uuid.UUID
	FornecedorID *uuid.UUID
	Status       string
	De           *time.Time
	Ate          *time.Time
	Filtro       domain.PaginacaoFiltro
}

type EstornarAntecipacaoInput struct {
	UsuarioID     uuid.UUID
	AntecipacaoID uuid.UUID
	Versao        int
	Motivo        string
}

type RecomprarInput struct {
	UsuarioID             uuid.UUID
	TituloID              uuid.UUID
	Versao                int
	RecompradoEm          time.Time
	EncargosCentavos      int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
	Motivo                string
}

type AntecipacaoUseCase interface {
	Criar(ctx context.Context, input CriarAntecipacaoInput) (*domainreceber.Antecipacao, error)
	Obter(ctx context.Context, antecipacaoID, usuarioID uuid.UUID) (*domainreceber.Antecipacao, error)
	Listar(ctx context.Context, input ListarAntecipacoesInput) ([]*domainreceber.Antecipacao, error)
	Estornar(ctx context.Context, input EstornarAntecipacaoInput) (*domainreceber.Antecipacao, error)
	Recomprar(ctx context.Context, input RecomprarInput) (*domainreceber.Titulo, *domainreceber.Recompra, error)
}
