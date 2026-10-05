package usuarios

import (
	"time"
	"uuid"
)

type Usuario struct {
	ID          uuid.UUID        `db:"id"`
	Nome        string        `db:"nome"`
	Email       string        `db:"email"`
	Senha       string        `db:"senha"`
	CargoID     uuid.UUID        `db:"cargo_id"`
	Ativo       bool          `db:"ativo"`
	UltimoLogin time.Time `db:"ultimo_login"`
	CriadoEm    time.Time     `db:"criado_em"`
	AtualizadoEm time.Time     `db:"atualizado_em"`
}