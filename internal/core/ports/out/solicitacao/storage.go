package solicitacao

import (
	"context"
	"io"
)

type Storage interface {
	Enviar(ctx context.Context, chave string, conteudo io.Reader, contentType string) error
	Baixar(ctx context.Context, chave string) (io.ReadCloser, error)
	Remover(ctx context.Context, chave string) error
}
