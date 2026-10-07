package autenticacao

import "errors"

var (
	errUsuarioInvalido = errors.New("usuário inválido para o token")
	errTokenInvalido   = errors.New("token inválido")
)
