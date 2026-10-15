package solicitacao

type Evento string

const (
	EventoCriada       Evento = "criada"
	EventoAprovada     Evento = "aprovada"
	EventoRejeitada    Evento = "rejeitada"
	EventoCancelada    Evento = "cancelada"
	EventoPaga         Evento = "paga"
	EventoEditada      Evento = "editada"
	EventoClassificada Evento = "classificada"
	EventoAdiada       Evento = "adiada"
	EventoEstornada    Evento = "estornada"
)

func EventoDe(valor string) Evento {
	return Evento(valor)
}

func (e Evento) String() string {
	return string(e)
}
