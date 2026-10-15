package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type DadosDespesaRecorrenteInput struct {
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Descricao             string
	ValorCentavos         int64
	Estimado              bool
	FormaPagamento        string
	DiaVencimento         int
	VencimentoMesSeguinte bool
	Inicio                string
	Fim                   string
}

type CriarDespesaRecorrenteInput struct {
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
	Dados     DadosDespesaRecorrenteInput
}

type AlterarDespesaRecorrenteInput struct {
	UsuarioID uuid.UUID
	DespesaID uuid.UUID
	Dados     DadosDespesaRecorrenteInput
	Versao    int
}

type MudarSituacaoDespesaRecorrenteInput struct {
	UsuarioID uuid.UUID
	DespesaID uuid.UUID
	Versao    int
}

type ListarDespesasRecorrentesInput struct {
	UsuarioID  uuid.UUID
	Empresa    *uuid.UUID
	Ativa      *bool
	Fornecedor *uuid.UUID
	Filtro     domain.PaginacaoFiltro
}

type GerarDespesasRecorrentesInput struct {
	UsuarioID   uuid.UUID
	Competencia string
	Empresa     *uuid.UUID
}

type RecorrenciaRecusada struct {
	DespesaRecorrenteID uuid.UUID
	Motivo              string
}

type ResultadoGeracao struct {
	Competencia  string
	Geradas      []uuid.UUID
	JaExistentes int
	Recusadas    []RecorrenciaRecusada
}

type DespesaRecorrenteUseCase interface {
	Criar(ctx context.Context, input CriarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error)
	Obter(ctx context.Context, despesaID, usuarioID uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error)
	Listar(ctx context.Context, input ListarDespesasRecorrentesInput) ([]*domainsolicitacao.DespesaRecorrente, error)
	Alterar(ctx context.Context, input AlterarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error)
	Ativar(ctx context.Context, input MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error)
	Desativar(ctx context.Context, input MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error)
	Remover(ctx context.Context, despesaID, usuarioID uuid.UUID) error
	Gerar(ctx context.Context, input GerarDespesasRecorrentesInput) (ResultadoGeracao, error)
}
