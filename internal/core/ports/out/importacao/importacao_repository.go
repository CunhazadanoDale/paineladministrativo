package importacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type Gravacao struct {
	Importacao        *domainimportacao.Importacao
	Novo              bool
	VersaoLida        int
	StatusAnterior    domainimportacao.Status
	SubstituirItens   bool
	FornecedorTrocado bool
	Historicos        []*domainimportacao.Historico
}

type ImportacaoFiltro struct {
	domain.PaginacaoFiltro
	Empresa      *uuid.UUID
	EmpresaIDs   []uuid.UUID
	FornecedorID *uuid.UUID
	Status       domainimportacao.Status
	Numero       *int
	ChegadaAte   *time.Time
}

type ImportacaoRepository interface {
	Gravar(ctx context.Context, gravacao *Gravacao) error
	Obter(ctx context.Context, id uuid.UUID) (*domainimportacao.Importacao, error)
	Listar(ctx context.Context, filtro ImportacaoFiltro) ([]*domainimportacao.Importacao, error)
	ListarHistorico(ctx context.Context, importacaoID uuid.UUID) ([]*domainimportacao.Historico, error)
}
