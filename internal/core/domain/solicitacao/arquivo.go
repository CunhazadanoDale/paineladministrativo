package solicitacao

import (
	"time"

	"github.com/google/uuid"
)

type Arquivo struct {
	ID             uuid.UUID `db:"id"`
	ProprietarioID uuid.UUID `db:"proprietario_id"`
	Nome           string    `db:"nome"`
	Chave          string    `db:"chave"`
	ContentType    string    `db:"content_type"`
	Tamanho        int64     `db:"tamanho"`
	CriadoEm       time.Time `db:"criado_em"`
}
