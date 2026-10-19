package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type ClassificarInput struct {
	UsuarioID             uuid.UUID
	LancamentoID          uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Descricao             string
}

type TransferirInput struct {
	UsuarioID             uuid.UUID
	LancamentoID          uuid.UUID
	ContraparteID         uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Descricao             string
}

type ListarCaixaInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ContaBancariaID *uuid.UUID
	De              *time.Time
	Ate             *time.Time
	Filtro          domain.PaginacaoFiltro
}

type LancamentoCaixaUseCase interface {
	Classificar(ctx context.Context, input ClassificarInput) (*domaincaixa.Classificacao, error)
	Transferir(ctx context.Context, input TransferirInput) (*domaincaixa.TransferenciaRegistrada, error)
	Listar(ctx context.Context, input ListarCaixaInput) ([]*domaincaixa.LancamentoCaixa, error)
	ListarTransferencias(ctx context.Context, input ListarCaixaInput) ([]*domaincaixa.Transferencia, error)
}
