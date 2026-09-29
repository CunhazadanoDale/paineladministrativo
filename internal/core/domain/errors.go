package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound  = errors.New("registro não encontrado")
	ErrValidacao = errors.New("erro de validação")
)

func ErroValidacao(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrValidacao, mensagem)
}

func ErroNaoEncontrado(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, mensagem)
}
