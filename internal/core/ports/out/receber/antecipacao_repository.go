package receber

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type TituloAlterado struct {
	Titulo         *domainreceber.Titulo
	VersaoLida     int
	StatusAnterior domainreceber.Status
	Historico      *domainreceber.Historico
}

type GravacaoAntecipacao struct {
	Antecipacao *domainreceber.Antecipacao
	Titulos     []TituloAlterado
}

type EstornoDaAntecipacao struct {
	Antecipacao *domainreceber.Antecipacao
	VersaoLida  int
	Titulos     []TituloAlterado
}

type ContaAPagarGerada struct {
	Solicitacao *domainsolicitacao.Solicitacao
	Historico   *domainsolicitacao.Historico
}

type GravacaoRecompra struct {
	Titulo      TituloAlterado
	Recompra    *domainreceber.Recompra
	ContaAPagar ContaAPagarGerada
}

type AntecipacaoFiltro struct {
	domain.PaginacaoFiltro
	Empresa      *uuid.UUID
	EmpresaIDs   []uuid.UUID
	FornecedorID *uuid.UUID
	Status       domainreceber.StatusAntecipacao
	De           *time.Time
	Ate          *time.Time
}

type AntecipacaoRepository interface {
	Criar(ctx context.Context, gravacao *GravacaoAntecipacao) error
	Obter(ctx context.Context, id uuid.UUID) (*domainreceber.Antecipacao, error)
	ObterVigenteDoTitulo(ctx context.Context, tituloID uuid.UUID) (*domainreceber.Antecipacao, error)
	Listar(ctx context.Context, filtro AntecipacaoFiltro) ([]*domainreceber.Antecipacao, error)
	Estornar(ctx context.Context, estorno *EstornoDaAntecipacao) error
	Recomprar(ctx context.Context, gravacao *GravacaoRecompra) error
}
