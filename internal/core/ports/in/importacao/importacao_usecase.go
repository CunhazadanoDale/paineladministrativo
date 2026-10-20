package importacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type ItemInput struct {
	ProdutoID            uuid.UUID
	Quantidade           int
	ValorMoedaCentesimos int64
}

type DadosImportacaoInput struct {
	FornecedorID     uuid.UUID
	Moeda            string
	Referencia       *string
	EmbarquePrevisto *time.Time
	ChegadaPrevista  *time.Time
	Observacoes      string
	Itens            []ItemInput
}

type CriarImportacaoInput struct {
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
	Dados     DadosImportacaoInput
}

type AlterarImportacaoInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	Versao       int
	Dados        DadosImportacaoInput
}

type EmbarcarInput struct {
	UsuarioID         uuid.UUID
	ImportacaoID      uuid.UUID
	Versao            int
	EmbarcadaEm       time.Time
	ChegadaPrevista   time.Time
	DocumentoEmbarque *string
}

type DesembaracarInput struct {
	UsuarioID       uuid.UUID
	ImportacaoID    uuid.UUID
	Versao          int
	ChegadaEm       time.Time
	DesembaracadaEm time.Time
	Declaracao      string
}

type CancelarImportacaoInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	Versao       int
	Motivo       string
}

type ListarImportacoesInput struct {
	UsuarioID    uuid.UUID
	Empresa      *uuid.UUID
	FornecedorID *uuid.UUID
	Status       string
	Numero       *int
	ChegadaAte   *time.Time
	Filtro       domain.PaginacaoFiltro
}

type Visao struct {
	Importacao *domainimportacao.Importacao
	ComValores bool
}

type Lista struct {
	Importacoes []*domainimportacao.Importacao
	ComValores  bool
}

type ImportacaoUseCase interface {
	Criar(ctx context.Context, input CriarImportacaoInput) (*Visao, error)
	Obter(ctx context.Context, importacaoID, usuarioID uuid.UUID) (*Visao, error)
	Listar(ctx context.Context, input ListarImportacoesInput) (*Lista, error)
	Alterar(ctx context.Context, input AlterarImportacaoInput) (*Visao, error)
	Embarcar(ctx context.Context, input EmbarcarInput) (*Visao, error)
	Desembaracar(ctx context.Context, input DesembaracarInput) (*Visao, error)
	Cancelar(ctx context.Context, input CancelarImportacaoInput) (*Visao, error)
	ListarHistorico(ctx context.Context, importacaoID, usuarioID uuid.UUID) ([]*domainimportacao.Historico, error)
}
