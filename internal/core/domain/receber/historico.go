package receber

import (
	"time"

	"github.com/google/uuid"
)

const tamanhoMaximoDescricaoHistorico = 1000

type Evento string

const (
	EventoCriado               Evento = "criado"
	EventoGerado               Evento = "gerado"
	EventoEditado              Evento = "editado"
	EventoProrrogado           Evento = "prorrogado"
	EventoEnviadoBanco         Evento = "enviado_banco"
	EventoRetiradoBanco        Evento = "retirado_banco"
	EventoBoletoEnviado        Evento = "boleto_enviado"
	EventoAntecipado           Evento = "antecipado"
	EventoAntecipacaoEstornada Evento = "antecipacao_estornada"
	EventoRecomprado           Evento = "recomprado"
	EventoRecebido             Evento = "recebido"
	EventoRecebimentoEstornado Evento = "recebimento_estornado"
	EventoCancelado            Evento = "cancelado"
	EventoOrigemCancelada      Evento = "origem_cancelada"
)

func EventoDe(valor string) Evento {
	return Evento(valor)
}

func (e Evento) String() string {
	return string(e)
}

type Historico struct {
	ID         uuid.UUID
	TituloID   uuid.UUID
	UsuarioID  uuid.UUID
	Evento     Evento
	DeStatus   *Status
	ParaStatus Status
	Descricao  string
	CriadoEm   time.Time
}

func NovoHistorico(tituloID, usuarioID uuid.UUID, evento Evento, de *Status, para Status, descricao string, agora time.Time) *Historico {
	return &Historico{
		ID:         uuid.Must(uuid.NewV7()),
		TituloID:   tituloID,
		UsuarioID:  usuarioID,
		Evento:     evento,
		DeStatus:   de,
		ParaStatus: para,
		Descricao:  limitarTexto(descricao, tamanhoMaximoDescricaoHistorico),
		CriadoEm:   agora,
	}
}
