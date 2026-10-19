package caixa

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type TipoMovimento string

const (
	MovimentoPagamento   TipoMovimento = "pagamento"
	MovimentoRecebimento TipoMovimento = "recebimento"
	MovimentoAntecipacao TipoMovimento = "antecipacao"
)

func NovoTipoMovimento(valor string) (TipoMovimento, error) {
	tipo := TipoMovimento(strings.ToLower(strings.TrimSpace(valor)))
	if err := tipo.Validado(); err != nil {
		return "", err
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
	case MovimentoPagamento, MovimentoRecebimento, MovimentoAntecipacao:
		return nil
	default:
		return domain.ErroValidacao("tipo do movimento deve ser pagamento, recebimento ou antecipacao")
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
