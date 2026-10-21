package divida

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ContaCancelada struct {
	domaindivida.ContaCancelada
	Historico *domainsolicitacao.Historico
}

type Gravacao struct {
	Divida            *domaindivida.Divida
	Novo              bool
	VersaoLida        int
	StatusAnterior    domaindivida.Status
	ParcelasAlteradas []*domaindivida.Parcela
	ParcelasNovas     []*domaindivida.Parcela
	ConferirContas    bool
	ContasCanceladas  []ContaCancelada
	Substituta        *domaindivida.Divida
	ContasAPagar      []domaindivida.ContaGerada
	Historicos        []*domaindivida.Historico
}

type DividaFiltro struct {
	domain.PaginacaoFiltro
	Empresa      *uuid.UUID
	EmpresaIDs   []uuid.UUID
	FornecedorID *uuid.UUID
	CredorTipo   domaindivida.TipoCredor
	Status       domaindivida.Status
}

type DividaRepository interface {
	Gravar(ctx context.Context, gravacao *Gravacao) error
	Obter(ctx context.Context, id uuid.UUID) (*domaindivida.Divida, error)
	Listar(ctx context.Context, filtro DividaFiltro) ([]*domaindivida.Divida, error)
	ListarHistorico(ctx context.Context, dividaID uuid.UUID) ([]*domaindivida.Historico, error)
	ComParcelasAGerar(ctx context.Context, limite time.Time, empresaIDs []uuid.UUID) ([]uuid.UUID, error)
}
