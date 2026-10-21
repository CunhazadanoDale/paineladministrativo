package divida

import (
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type StatusParcela string

const (
	StatusParcelaPrevista  StatusParcela = "prevista"
	StatusParcelaGerada    StatusParcela = "gerada"
	StatusParcelaCancelada StatusParcela = "cancelada"
)

func StatusParcelaDe(valor string) StatusParcela {
	return StatusParcela(valor)
}

func (s StatusParcela) String() string {
	return string(s)
}

type Situacao string

const (
	SituacaoAVencer        Situacao = "a_vencer"
	SituacaoVencida        Situacao = "vencida"
	SituacaoPaga           Situacao = "paga"
	SituacaoContaCancelada Situacao = "conta_cancelada"
	SituacaoCancelada      Situacao = "cancelada"
)

func (s Situacao) String() string {
	return string(s)
}

type Conta struct {
	SolicitacaoID uuid.UUID
	Status        domainsolicitacao.Status
}

func (c *Conta) paga() bool {
	return c != nil && c.Status == domainsolicitacao.StatusPago
}

func (c *Conta) encerrada() bool {
	return c != nil && (c.Status == domainsolicitacao.StatusCancelado || c.Status == domainsolicitacao.StatusRejeitado)
}

func (c *Conta) aberta() bool {
	return c != nil && (c.Status == domainsolicitacao.StatusPendenteAprovacao || c.Status == domainsolicitacao.StatusAprovado)
}

type ParcelaInput struct {
	Vencimento          time.Time
	AmortizacaoCentavos int64
	JurosCentavos       int64
}

type Parcela struct {
	ID                  uuid.UUID
	DividaID            uuid.UUID
	Numero              int
	Vencimento          time.Time
	AmortizacaoCentavos int64
	JurosCentavos       int64
	Status              StatusParcela
	GeradaEm            *time.Time
	SubstitutaID        *uuid.UUID
	ContaAmortizacao    *Conta
	ContaJuros          *Conta
	CriadoEm            time.Time
	AtualizadoEm        time.Time
}

func (p *Parcela) Contas() []*Conta {
	contas := make([]*Conta, 0, 2)
	if p.ContaAmortizacao != nil {
		contas = append(contas, p.ContaAmortizacao)
	}
	if p.ContaJuros != nil {
		contas = append(contas, p.ContaJuros)
	}

	return contas
}

func (p *Parcela) Situacao(hoje time.Time) Situacao {
	if p.Status == StatusParcelaCancelada {
		return SituacaoCancelada
	}

	contas := p.Contas()
	pagas := 0
	for _, conta := range contas {
		if conta.encerrada() {
			return SituacaoContaCancelada
		}
		if conta.paga() {
			pagas++
		}
	}
	if p.Status == StatusParcelaGerada && len(contas) > 0 && pagas == len(contas) {
		return SituacaoPaga
	}
	if p.Vencimento.Before(hoje) {
		return SituacaoVencida
	}

	return SituacaoAVencer
}

func (p *Parcela) AmortizacaoEmAberto() int64 {
	if p.Status == StatusParcelaCancelada || p.ContaAmortizacao.paga() {
		return 0
	}

	return p.AmortizacaoCentavos
}

func (p *Parcela) JurosEmAberto() int64 {
	if p.Status == StatusParcelaCancelada || p.ContaJuros.paga() {
		return 0
	}

	return p.JurosCentavos
}

func (p *Parcela) TemContaPaga() bool {
	return p.ContaAmortizacao.paga() || p.ContaJuros.paga()
}

func (p *Parcela) ContasAbertas() []*Conta {
	abertas := make([]*Conta, 0, 2)
	for _, conta := range p.Contas() {
		if conta.aberta() {
			abertas = append(abertas, conta)
		}
	}

	return abertas
}

func (p *Parcela) cancelar(agora time.Time) {
	p.Status = StatusParcelaCancelada
	p.AtualizadoEm = agora
}
