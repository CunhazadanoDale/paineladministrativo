package receber

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type DadosEdicaoInput struct {
	Numero                string
	Parcela               int
	Vencimento            time.Time
	ValorCentavos         int64
	CategoriaFinanceiraID *uuid.UUID
	Competencia           string
	Observacao            string
}

type CriarTituloInput struct {
	UsuarioID       uuid.UUID
	EmpresaID       uuid.UUID
	ClienteID       uuid.UUID
	ContaBancariaID *uuid.UUID
	Dados           DadosEdicaoInput
}

type EditarTituloInput struct {
	UsuarioID uuid.UUID
	TituloID  uuid.UUID
	Versao    int
	Dados     DadosEdicaoInput
}

type ProrrogarTituloInput struct {
	UsuarioID  uuid.UUID
	TituloID   uuid.UUID
	Versao     int
	Vencimento time.Time
	Motivo     string
}

type EnviarAoBancoInput struct {
	UsuarioID       uuid.UUID
	TituloID        uuid.UUID
	Versao          int
	ContaBancariaID uuid.UUID
}

type TransicaoTituloInput struct {
	UsuarioID uuid.UUID
	TituloID  uuid.UUID
	Versao    int
}

type RegistrarEnvioBoletoInput struct {
	UsuarioID uuid.UUID
	TituloID  uuid.UUID
	Versao    int
	Canal     string
	EnviadoEm time.Time
}

type CancelarTituloInput struct {
	UsuarioID uuid.UUID
	TituloID  uuid.UUID
	Versao    int
	Motivo    string
}

type ListarTitulosInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ClienteID       *uuid.UUID
	Status          string
	Origem          string
	OrigemID        *uuid.UUID
	ContaBancariaID *uuid.UUID
	VencimentoDe    *time.Time
	VencimentoAte   *time.Time
	Vencidos        bool
	OrigemCancelada bool
	Filtro          domain.PaginacaoFiltro
}

type BoletosPendentesInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Dias      *int
	Filtro    domain.PaginacaoFiltro
}

type TituloUseCase interface {
	Criar(ctx context.Context, input CriarTituloInput) (*domainreceber.Titulo, error)
	Obter(ctx context.Context, tituloID, usuarioID uuid.UUID) (*domainreceber.Titulo, error)
	Listar(ctx context.Context, input ListarTitulosInput) ([]*domainreceber.Titulo, error)
	ListarHistorico(ctx context.Context, tituloID, usuarioID uuid.UUID) ([]*domainreceber.Historico, error)
	Editar(ctx context.Context, input EditarTituloInput) (*domainreceber.Titulo, error)
	Prorrogar(ctx context.Context, input ProrrogarTituloInput) (*domainreceber.Titulo, error)
	EnviarAoBanco(ctx context.Context, input EnviarAoBancoInput) (*domainreceber.Titulo, error)
	RetirarDoBanco(ctx context.Context, input TransicaoTituloInput) (*domainreceber.Titulo, error)
	RegistrarEnvioBoleto(ctx context.Context, input RegistrarEnvioBoletoInput) (*domainreceber.Titulo, error)
	Cancelar(ctx context.Context, input CancelarTituloInput) (*domainreceber.Titulo, error)
	BoletosPendentes(ctx context.Context, input BoletosPendentesInput) ([]*domainreceber.Titulo, error)
}
