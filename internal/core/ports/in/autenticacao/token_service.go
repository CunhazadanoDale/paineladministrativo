package autenticacao

import (
	"time"

	"github.com/google/uuid"
)

type TokenService interface {
	Gerar(usuarioID uuid.UUID, versaoSessao int) (string, time.Time, error)
	Validar(token string) (uuid.UUID, int, error)
}
