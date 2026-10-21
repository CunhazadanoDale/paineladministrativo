package divida

import (
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const tamanhoMaximoDetalheHistorico = 1000

type Acao string

const (
	AcaoCriada          Acao = "criada"
	AcaoAlterada        Acao = "alterada"
	AcaoParcelaAlterada Acao = "parcela_alterada"
	AcaoContasGeradas   Acao = "contas_geradas"
	AcaoParcelaReaberta Acao = "parcela_reaberta"
	AcaoRenegociada     Acao = "renegociada"
	AcaoCancelada       Acao = "cancelada"
)

func AcaoDe(valor string) Acao {
	return Acao(valor)
}

func (a Acao) String() string {
	return string(a)
}

type Historico struct {
	ID             uuid.UUID
	DividaID       uuid.UUID
	UsuarioID      uuid.UUID
	Acao           Acao
	StatusAnterior *Status
	StatusNovo     Status
	Detalhe        string
	CriadoEm       time.Time
}

func NovoHistorico(dividaID, usuarioID uuid.UUID, acao Acao, anterior *Status, novo Status, detalhe string, agora time.Time) *Historico {
	return &Historico{
		ID:             uuid.Must(uuid.NewV7()),
		DividaID:       dividaID,
		UsuarioID:      usuarioID,
		Acao:           acao,
		StatusAnterior: anterior,
		StatusNovo:     novo,
		Detalhe:        limitarDetalhe(detalhe),
		CriadoEm:       agora,
	}
}

func limitarDetalhe(detalhe string) string {
	if utf8.RuneCountInString(detalhe) <= tamanhoMaximoDetalheHistorico {
		return detalhe
	}

	return string([]rune(detalhe)[:tamanhoMaximoDetalheHistorico])
}
