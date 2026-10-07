package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound  = errors.New("registro não encontrado")
	ErrValidacao = errors.New("erro de validação")
	ErrPermissao = errors.New("perfil sem permissão para esta operação")
	ErrConflito  = errors.New("operação em conflito com o estado atual")
)

func ErroValidacao(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrValidacao, mensagem)
}

func ErroNaoEncontrado(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, mensagem)
}

func ErroPermissao(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrPermissao, mensagem)
}

func ErroConflito(mensagem string) error {
	return fmt.Errorf("%w: %s", ErrConflito, mensagem)
}
