package financeiro

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type ContaBancariaFiltro struct {
	domain.PaginacaoFiltro
	EmpresaIDs []uuid.UUID
	Banco      *domainfinanceiro.Banco
	Tipo       *domainfinanceiro.TipoConta
	Ativa      *bool
}

type ContaBancariaRepository interface {
	Criar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error
	Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error)
	Listar(ctx context.Context, filtro ContaBancariaFiltro) ([]*domainfinanceiro.ContaBancaria, error)
	Atualizar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error
	Ativar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error
	Desativar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error
	Remover(ctx context.Context, id uuid.UUID) error
}
