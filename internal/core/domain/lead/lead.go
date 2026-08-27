package lead

import "github.com/google/uuid"

type Lead struct {
	ID uuid.UUID `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
	Telefone string `db:"telefone"`
	Ativo bool `db:"ativo"`
	Funil uuid.UUID `db:"funil_id"`
}