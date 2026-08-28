package lead

import "github.com/google/uuid"

type Funil struct {
	FunilID uuid.UUID `db:"funil_id"`
	Nome    string    `db:"nome"`
}

type Etapa struct {
	EtapaID uuid.UUID `db:"etapa_id"`
	Nome    string    `db:"nome"`
	Ordem  int       `db:"ordem"`
	FunilID uuid.UUID `db:"funil_id"`
}