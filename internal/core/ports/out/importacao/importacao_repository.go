package importacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ContaAPagarGerada struct {
	Solicitacao *domainsolicitacao.Solicitacao
	Historico   *domainsolicitacao.Historico
}

type Gravacao struct {
	Importacao               *domainimportacao.Importacao
	Novo                     bool
	VersaoLida               int
	StatusAnterior           domainimportacao.Status
	SubstituirItens          bool
	FornecedorTrocado        bool
	ParcelasNovas            []*domainimportacao.Parcela
	ParcelasAlteradas        []*domainimportacao.Parcela
	ParcelasRemovidas        []uuid.UUID
	CustosNovos              []*domainimportacao.Custo
	CustosCancelados         []*domainimportacao.Custo
	ContasAPagar             []ContaAPagarGerada
	ContasExigidasEncerradas []uuid.UUID
	Historicos               []*domainimportacao.Historico
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
