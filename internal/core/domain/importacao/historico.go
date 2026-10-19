package importacao

import (
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const tamanhoMaximoDetalheHistorico = 1000

type Acao string

const (
	AcaoCriada        Acao = "criada"
	AcaoAlterada      Acao = "alterada"
	AcaoEmbarcada     Acao = "embarcada"
	AcaoDesembaracada Acao = "desembaracada"
	AcaoCancelada     Acao = "cancelada"
)

func AcaoDe(valor string) Acao {
	return Acao(valor)
}

func (a Acao) String() string {
	return string(a)
}

type Historico struct {
	ID             uuid.UUID
	ImportacaoID   uuid.UUID
	UsuarioID      uuid.UUID
	Acao           Acao
	StatusAnterior *Status
	StatusNovo     Status
	Detalhe        string
	CriadoEm       time.Time
}

func NovoHistorico(importacaoID, usuarioID uuid.UUID, acao Acao, anterior *Status, novo Status, detalhe string, agora time.Time) *Historico {
	return &Historico{
		ID:             uuid.Must(uuid.NewV7()),
		ImportacaoID:   importacaoID,
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
