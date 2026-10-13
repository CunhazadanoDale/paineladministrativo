package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type DadosClienteInput struct {
	Nome                    string
	NomeFantasia            *string
	Tipo                    string
	GrupoID                 *uuid.UUID
	UF                      string
	Cidade                  string
	EmpresaRelacionamentoID *uuid.UUID
	RepresentanteID         *uuid.UUID
	EmailNF                 *string
	Telefone                *string
	Observacoes             string
	ExigeNumeroPedidoNaNF   bool
	ProibeAntecipacao       bool
}

type CriarClienteInput struct {
	UsuarioID uuid.UUID
	Documento string
	Dados     DadosClienteInput
}

type AlterarClienteInput struct {
	UsuarioID uuid.UUID
	ClienteID uuid.UUID
	Dados     DadosClienteInput
}

type ListarClientesInput struct {
	UsuarioID       uuid.UUID
	Busca           string
	GrupoID         *uuid.UUID
	RepresentanteID *uuid.UUID
	Tipo            string
	UF              string
	Ativo           *bool
	LeadID          *uuid.UUID
	Filtro          domain.PaginacaoFiltro
}

type ClienteUseCase interface {
	Criar(ctx context.Context, input CriarClienteInput) (*domaincliente.Cliente, error)
	Obter(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error)
	Listar(ctx context.Context, input ListarClientesInput) ([]*domaincliente.Cliente, error)
	Alterar(ctx context.Context, input AlterarClienteInput) (*domaincliente.Cliente, error)
	Ativar(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error)
	Desativar(ctx context.Context, clienteID, usuarioID uuid.UUID) (*domaincliente.Cliente, error)
	Remover(ctx context.Context, clienteID, usuarioID uuid.UUID) error
}
