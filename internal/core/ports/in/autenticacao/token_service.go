package autenticacao

import (
	"time"

	"github.com/google/uuid"
)

type TokenService interface {
	Gerar(usuarioID uuid.UUID) (string, time.Time, error)
	Validar(token string) (uuid.UUID, error)
}
