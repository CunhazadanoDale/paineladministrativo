package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type ImportarExtratoInput struct {
	UsuarioID       uuid.UUID
	ArquivoID       uuid.UUID
	ContaBancariaID *uuid.UUID
	Simular         bool
}

type RelatorioImportacao struct {
	Simulacao  bool
	Importacao *domaincaixa.ImportacaoExtrato
	Total      int
	Ignoradas  []domaincaixa.LinhaIgnorada
}

type ListarImportacoesInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ContaBancariaID *uuid.UUID
	Filtro          domain.PaginacaoFiltro
}

type ListarLancamentosInput struct {
	UsuarioID       uuid.UUID
	ContaBancariaID uuid.UUID
	De              *time.Time
	Ate             *time.Time
	Filtro          domain.PaginacaoFiltro
}

type ExtratoUseCase interface {
	Importar(ctx context.Context, input ImportarExtratoInput) (*RelatorioImportacao, error)
	ListarImportacoes(ctx context.Context, input ListarImportacoesInput) ([]*domaincaixa.ImportacaoExtrato, error)
	DesfazerImportacao(ctx context.Context, importacaoID, usuarioID uuid.UUID) error
	ListarLancamentos(ctx context.Context, input ListarLancamentosInput) ([]*domaincaixa.LancamentoExtrato, error)
}
