package importacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	folgaPadraoDias = 10
	folgaMaximaDias = 90
)

func NovaFolga(dias *int) (int, error) {
	if dias == nil {
		return folgaPadraoDias, nil
	}
	if *dias < 0 || *dias > folgaMaximaDias {
		return 0, domain.ErroValidacao("folga deve ser de 0 a 90 dias")
	}

	return *dias, nil
}

type AlocacaoAberta struct {
	ImportacaoID     uuid.UUID
	ImportacaoNumero int
	ImportacaoStatus Status
	EmpresaID        uuid.UUID
	ChegadaPrevista  *time.Time
	PedidoID         uuid.UUID
	PedidoNumero     int
	PedidoItemID     uuid.UUID
	ProdutoID        uuid.UUID
	Quantidade       int
	PrazoLimite      *time.Time
}

type PrazoEmRisco struct {
	AlocacaoAberta
	ChegadaComFolga time.Time
	DiasDeAtraso    int
}

type PedidoSemImportacao struct {
	PedidoID     uuid.UUID
	PedidoNumero int
	EmpresaID    uuid.UUID
	PedidoItemID uuid.UUID
	ProdutoID    uuid.UUID
	PrazoLimite  *time.Time
	Restante     int
	Reservado    int
	Alocado      int
}

func (p PedidoSemImportacao) SemCobertura() int {
	return max(p.Restante-p.Reservado-p.Alocado, 0)
}

type Alertas struct {
	Folga         int
	PrazoEmRisco  []PrazoEmRisco
	SemPrevisao   []AlocacaoAberta
	SemImportacao []PedidoSemImportacao
}

func ClassificarAlertas(folga int, alocacoes []AlocacaoAberta, pedidos []PedidoSemImportacao) *Alertas {
	alertas := &Alertas{
		Folga:         folga,
		PrazoEmRisco:  make([]PrazoEmRisco, 0),
		SemPrevisao:   make([]AlocacaoAberta, 0),
		SemImportacao: make([]PedidoSemImportacao, 0),
	}

	for _, alocacao := range alocacoes {
		if alocacao.ChegadaPrevista == nil {
			alertas.SemPrevisao = append(alertas.SemPrevisao, alocacao)
			continue
		}
		if alocacao.PrazoLimite == nil {
			continue
		}

		chegada := alocacao.ChegadaPrevista.AddDate(0, 0, folga)
		if chegada.After(*alocacao.PrazoLimite) {
			alertas.PrazoEmRisco = append(alertas.PrazoEmRisco, PrazoEmRisco{
				AlocacaoAberta:  alocacao,
				ChegadaComFolga: chegada,
				DiasDeAtraso:    int(chegada.Sub(*alocacao.PrazoLimite).Hours() / 24),
			})
		}
	}

	for _, pedido := range pedidos {
		if pedido.SemCobertura() > 0 {
			alertas.SemImportacao = append(alertas.SemImportacao, pedido)
		}
	}

	return alertas
}

type AtendimentoDoPedido struct {
	ImportacaoID     uuid.UUID
	ImportacaoNumero int
	ImportacaoStatus Status
	ChegadaPrevista  *time.Time
	RecebidaEm       *time.Time
	PedidoItemID     uuid.UUID
	ProdutoID        uuid.UUID
	Quantidade       int
	Reservada        *int
}
