package lead

import "github.com/google/uuid"

type Lead struct {
	ID uuid.UUID `db:"id"`
	Nome  string `db:"nome"`
	Email string `db:"email"`
	Telefone string `db:"telefone"`
	Ativo bool `db:"ativo"`

	FunilID uuid.UUID `db:"funil_id"`
	EtapaID uuid.UUID `db:"etapa_id"`
}