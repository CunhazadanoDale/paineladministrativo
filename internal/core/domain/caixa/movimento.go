package caixa

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type TipoMovimento string

const (
	MovimentoPagamento            TipoMovimento = "pagamento"
	MovimentoRecebimento          TipoMovimento = "recebimento"
	MovimentoAntecipacao          TipoMovimento = "antecipacao"
	MovimentoLancamentoCaixa      TipoMovimento = "lancamento_caixa"
	MovimentoTransferenciaSaida   TipoMovimento = "transferencia_saida"
	MovimentoTransferenciaEntrada TipoMovimento = "transferencia_entrada"
)

func NovoTipoMovimento(valor string) (TipoMovimento, error) {
	tipo := TipoMovimento(strings.ToLower(strings.TrimSpace(valor)))
	if err := tipo.Validado(); err != nil {
		return "", err
	}
	if tipo == MovimentoLancamentoCaixa {
		return "", domain.ErroValidacao("lançamento de caixa nasce da classificação da linha e não é conciliado à mão")
	}

	return tipo, nil
}

func TipoMovimentoDe(valor string) TipoMovimento {
	return TipoMovimento(valor)
}

func (t TipoMovimento) String() string {
	return string(t)
}

func (t TipoMovimento) Validado() error {
	switch t {
	case MovimentoPagamento, MovimentoRecebimento, MovimentoAntecipacao, MovimentoLancamentoCaixa,
		MovimentoTransferenciaSaida, MovimentoTransferenciaEntrada:
		return nil
	default:
		return domain.ErroValidacao("tipo do movimento deve ser pagamento, recebimento, antecipacao, " +
			"transferencia_saida ou transferencia_entrada")
	}
}

type ReferenciaMovimento struct {
	Tipo TipoMovimento
	ID   uuid.UUID
}

type MovimentoSistema struct {
	Tipo            TipoMovimento
	ID              uuid.UUID
	ContaBancariaID uuid.UUID
	EmpresaID       uuid.UUID
	Data            time.Time
	ValorCentavos   int64
	Descricao       string
	OrigemID        uuid.UUID
	Estornado       bool
	ConciliacaoID   *uuid.UUID
}

func (m *MovimentoSistema) Referencia() ReferenciaMovimento {
	return ReferenciaMovimento{Tipo: m.Tipo, ID: m.ID}
}

func (m *MovimentoSistema) Pendente() bool {
	return !m.Estornado && m.ConciliacaoID == nil
}
