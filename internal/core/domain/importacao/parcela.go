package importacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

const tamanhoMaximoDescricaoParcela = 120

type StatusParcela string

const (
	StatusParcelaAberta        StatusParcela = "aberta"
	StatusParcelaCambioFechado StatusParcela = "cambio_fechado"
	StatusParcelaCancelada     StatusParcela = "cancelada"
)

func StatusParcelaDe(valor string) StatusParcela {
	return StatusParcela(valor)
}

func (s StatusParcela) String() string {
	return string(s)
}

type Parcela struct {
	ID                 uuid.UUID
	ImportacaoID       uuid.UUID
	Descricao          string
	Valor              ValorMoeda
	VencimentoPrevisto time.Time
	Status             StatusParcela
	Cambio             *Cambio
	ValorCentavos      *int64
	Vencimento         *time.Time
	SolicitacaoID      *uuid.UUID
	StatusContaAPagar  *domainsolicitacao.Status
	UsuarioID          uuid.UUID
	CriadoEm           time.Time
	AtualizadoEm       time.Time
}

type DadosParcela struct {
	Descricao            string
	ValorMoedaCentesimos int64
	VencimentoPrevisto   time.Time
}

type FechamentoCambioInput struct {
	CambioMicro *int64
	Vencimento  time.Time
}

func (p *Parcela) Ativa() bool {
	return p.Status != StatusParcelaCancelada
}

func (p *Parcela) aplicarDados(dados DadosParcela, agora time.Time) error {
	descricao, err := textoObrigatorio(dados.Descricao, feminino("descrição da parcela"), tamanhoMaximoDescricaoParcela)
	if err != nil {
		return err
	}
	if dados.ValorMoedaCentesimos <= 0 {
		return domain.ErroValidacao("valor da parcela deve ser maior que zero")
	}
	if dados.VencimentoPrevisto.IsZero() {
		return domain.ErroValidacao(masculino("vencimento previsto da parcela").obrigatorio())
	}

	p.Descricao = descricao
	p.Valor = ValorMoedaDe(dados.ValorMoedaCentesimos)
	p.VencimentoPrevisto = DataDe(dados.VencimentoPrevisto)
	p.AtualizadoEm = agora

	return nil
}

func (p *Parcela) fecharCambio(cambio Cambio, vencimento time.Time, agora time.Time) error {
	if p.Status != StatusParcelaAberta {
		return domain.ErroConflito("o câmbio desta parcela já foi fechado")
	}
	if vencimento.IsZero() {
		return domain.ErroValidacao(masculino("vencimento da conta a pagar").obrigatorio())
	}

	centavos, err := cambio.EmReais(p.Valor)
	if err != nil {
		return err
	}
	if centavos <= 0 {
		return domain.ErroValidacao("valor da parcela em reais deve ser maior que zero")
	}

	data := DataDe(vencimento)
	p.Status = StatusParcelaCambioFechado
	p.Cambio = &cambio
	p.ValorCentavos = &centavos
	p.Vencimento = &data
	p.AtualizadoEm = agora

	return nil
}

func (p *Parcela) contaEncerrada() bool {
	if p.StatusContaAPagar == nil {
		return false
	}

	return contaAPagarEncerrada(*p.StatusContaAPagar)
}

func contaAPagarEncerrada(status domainsolicitacao.Status) bool {
	return status == domainsolicitacao.StatusCancelado || status == domainsolicitacao.StatusRejeitado
}
