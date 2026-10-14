package pedido

import (
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const tamanhoMaximoDetalheHistorico = 1000

type Acao string

const (
	AcaoCriado               Acao = "criado"
	AcaoAlterado             Acao = "alterado"
	AcaoConfirmado           Acao = "confirmado"
	AcaoEntradaRegistrada    Acao = "entrada_registrada"
	AcaoPendenciasReservadas Acao = "pendencias_reservadas"
	AcaoEntregaCriada        Acao = "entrega_criada"
	AcaoEntregaSaiu          Acao = "entrega_saiu"
	AcaoEntregaConcluida     Acao = "entrega_concluida"
	AcaoEntregaCancelada     Acao = "entrega_cancelada"
	AcaoCancelado            Acao = "cancelado"
)

func AcaoDe(valor string) Acao {
	return Acao(valor)
}

func (a Acao) String() string {
	return string(a)
}

type Historico struct {
	ID             uuid.UUID
	PedidoID       uuid.UUID
	UsuarioID      uuid.UUID
	Acao           Acao
	StatusAnterior *Status
	StatusNovo     Status
	Detalhe        string
	CriadoEm       time.Time
}

func NovoHistorico(pedidoID, usuarioID uuid.UUID, acao Acao, anterior *Status, novo Status, detalhe string, agora time.Time) *Historico {
	return &Historico{
		ID:             uuid.Must(uuid.NewV7()),
		PedidoID:       pedidoID,
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
