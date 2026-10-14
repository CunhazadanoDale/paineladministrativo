package financeiro

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type CriarContaBancariaInput struct {
	UsuarioID     uuid.UUID
	EmpresaID     uuid.UUID
	Banco         string
	Tipo          string
	Agencia       string
	AgenciaDigito *string
	Conta         string
	ContaDigito   *string
	Apelido       string
}

type AlterarContaBancariaInput struct {
	UsuarioID uuid.UUID
	ContaID   uuid.UUID
	Apelido   string
	Versao    int
}

type MudarSituacaoContaBancariaInput struct {
	UsuarioID uuid.UUID
	ContaID   uuid.UUID
	Versao    int
}

type ListarContasBancariasInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Banco     string
	Tipo      string
	Ativa     *bool
	Filtro    domain.PaginacaoFiltro
}

type ContaBancariaUseCase interface {
	Criar(ctx context.Context, input CriarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error)
	Obter(ctx context.Context, contaID, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error)
	Listar(ctx context.Context, input ListarContasBancariasInput) ([]*domainfinanceiro.ContaBancaria, error)
	Alterar(ctx context.Context, input AlterarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error)
	Ativar(ctx context.Context, input MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error)
	Desativar(ctx context.Context, input MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error)
	Remover(ctx context.Context, contaID, usuarioID uuid.UUID) error
	Bancos() []domainfinanceiro.Banco
}
