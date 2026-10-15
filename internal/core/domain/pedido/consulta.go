package pedido

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	diasAVencerPadrao = 30
	diasAVencerMaximo = 365
)

type Pendencia struct {
	PedidoID            uuid.UUID
	Numero              int
	EmpresaID           uuid.UUID
	ClienteID           uuid.UUID
	RepresentanteID     *uuid.UUID
	Status              Status
	PrazoLimite         *time.Time
	ItemID              uuid.UUID
	ProdutoID           uuid.UUID
	ProdutoNome         string
	Vendido             int
	Entregue            int
	Programado          int
	Reservado           int
	DisponivelNaEmpresa int
}

func (p Pendencia) Restante() int {
	return p.Vendido - p.Entregue
}

func (p Pendencia) FaltaCobrir() int {
	return max(p.Restante()-p.Reservado, 0)
}

func (p Pendencia) PodeCobrirAgora() int {
	return min(p.FaltaCobrir(), max(p.DisponivelNaEmpresa, 0))
}

type SituacaoPrazo string

const (
	SituacaoPrazoTodos   SituacaoPrazo = ""
	SituacaoPrazoVencido SituacaoPrazo = "vencido"
	SituacaoPrazoAVencer SituacaoPrazo = "a_vencer"
)

func NovaSituacaoPrazo(valor string) (SituacaoPrazo, error) {
	situacao := SituacaoPrazo(strings.ToLower(strings.TrimSpace(valor)))

	switch situacao {
	case SituacaoPrazoTodos, SituacaoPrazoVencido, SituacaoPrazoAVencer:
		return situacao, nil
	default:
		return "", domain.ErroValidacao("situação do prazo deve ser vencido ou a_vencer")
	}
}

type JanelaDePrazo struct {
	De  *time.Time
	Ate time.Time
}

func NovaJanelaDePrazo(situacao SituacaoPrazo, dias *int, agora time.Time) (JanelaDePrazo, error) {
	quantidade := diasAVencerPadrao
	if dias != nil {
		quantidade = *dias
	}
	if quantidade < 0 || quantidade > diasAVencerMaximo {
		return JanelaDePrazo{}, domain.ErroValidacao("dias deve estar entre 0 e 365")
	}

	hoje := Hoje(agora)
	switch situacao {
	case SituacaoPrazoVencido:
		return JanelaDePrazo{Ate: hoje.AddDate(0, 0, -1)}, nil
	case SituacaoPrazoAVencer:
		return JanelaDePrazo{De: &hoje, Ate: hoje.AddDate(0, 0, quantidade)}, nil
	default:
		return JanelaDePrazo{Ate: hoje.AddDate(0, 0, quantidade)}, nil
	}
}

func (p *Pedido) DiasParaOPrazo(agora time.Time) *int {
	if p.PrazoLimite == nil {
		return nil
	}

	dias := int(DataDe(*p.PrazoLimite).Sub(Hoje(agora)).Hours() / 24)

	return &dias
}
