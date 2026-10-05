package usuarios

import "github.com/google/uuid"

type Cargo struct {
	ID        uuid.UUID `db:"id"`
	Nome      string    `db:"nome"`
	Descricao string    `db:"descricao"`
	Ativo     bool      `db:"ativo"`
}
