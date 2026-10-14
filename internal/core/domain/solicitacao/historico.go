package solicitacao

import (
	"time"

	"github.com/google/uuid"
)

type Historico struct {
	ID            uuid.UUID `db:"id"`
	SolicitacaoID uuid.UUID `db:"solicitacao_id"`
	UsuarioID     uuid.UUID `db:"usuario_id"`
	DeStatus      *Status   `db:"de_status"`
	ParaStatus    Status    `db:"para_status"`
	Evento        Evento    `db:"evento"`
	Descricao     string    `db:"descricao"`
	CriadoEm      time.Time `db:"criado_em"`
}

func NovoHistorico(solicitacaoID, usuarioID uuid.UUID, deStatus *Status, paraStatus Status, evento Evento, descricao string) *Historico {
	return &Historico{
		ID:            uuid.New(),
		SolicitacaoID: solicitacaoID,
		UsuarioID:     usuarioID,
		DeStatus:      deStatus,
		ParaStatus:    paraStatus,
		Evento:        evento,
		Descricao:     descricao,
		CriadoEm:      time.Now().UTC(),
	}
}
