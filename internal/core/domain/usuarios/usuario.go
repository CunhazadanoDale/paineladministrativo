package usuarios

import (
	"time"

	"github.com/google/uuid"
)

type Usuario struct {
	ID           uuid.UUID `db:"id"`
	Nome         string    `db:"nome"`
	Email        string    `db:"email"`
	Senha        string    `db:"senha"`
	CargoID      uuid.UUID `db:"cargo_id"`
	Ativo        bool      `db:"ativo"`
	VersaoSessao int       `db:"versao_sessao"`
	UltimoLogin  time.Time `db:"ultimo_login"`
	CriadoEm     time.Time `db:"criado_em"`
	AtualizadoEm time.Time `db:"atualizado_em"`
}
