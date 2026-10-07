package solicitacao

import (
	"time"

	"github.com/google/uuid"
)

type Aprovador struct {
	ID        uuid.UUID `db:"id"`
	UsuarioID uuid.UUID `db:"usuario_id"`
	CriadoEm  time.Time `db:"criado_em"`
}
