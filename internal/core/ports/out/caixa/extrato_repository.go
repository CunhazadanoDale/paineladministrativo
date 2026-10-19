package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type ImportacaoFiltro struct {
	domain.PaginacaoFiltro
	EmpresaIDs      []uuid.UUID
	ContaBancariaID *uuid.UUID
}

type LancamentoFiltro struct {
	domain.PaginacaoFiltro
	ContaBancariaID uuid.UUID
	De              *time.Time
	Ate             *time.Time
	Situacao        *domaincaixa.SituacaoLancamento
}

type ExtratoRepository interface {
	ContasDoBanco(ctx context.Context, banco domainfinanceiro.Banco) ([]*domainfinanceiro.ContaBancaria, error)
	ArquivoImportado(ctx context.Context, arquivoID uuid.UUID) (bool, error)
	ContarExistentes(ctx context.Context, contaBancariaID uuid.UUID, identificadores []string) (int, error)
	Importar(ctx context.Context, preparado *domaincaixa.ExtratoPreparado, versaoDaAbertura int) error
	ObterImportacao(ctx context.Context, id uuid.UUID) (*domaincaixa.ImportacaoExtrato, error)
	ListarImportacoes(ctx context.Context, filtro ImportacaoFiltro) ([]*domaincaixa.ImportacaoExtrato, error)
	DesfazerImportacao(ctx context.Context, importacao *domaincaixa.ImportacaoExtrato) error
	ListarLancamentos(ctx context.Context, filtro LancamentoFiltro) ([]*domaincaixa.LancamentoExtrato, error)
}
