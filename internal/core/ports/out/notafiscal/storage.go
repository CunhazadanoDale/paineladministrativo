package notafiscal

import (
	"context"
	"io"
)

type Storage interface {
	Baixar(ctx context.Context, chave string) (io.ReadCloser, error)
}
