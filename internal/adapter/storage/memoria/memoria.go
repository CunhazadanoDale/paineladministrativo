package memoria

import (
	"bytes"
	"context"
	"io"
	"sync"

	portssolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
)

var _ portssolicitacao.Storage = (*Storage)(nil)

type Storage struct {
	mu        sync.RWMutex
	conteudos map[string][]byte
	tipos     map[string]string
}

func Novo() *Storage {
	return &Storage{
		conteudos: map[string][]byte{},
		tipos:     map[string]string{},
	}
}

func (s *Storage) Enviar(_ context.Context, chave string, conteudo io.Reader, contentType string) error {
	dados, err := io.ReadAll(conteudo)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.conteudos[chave] = dados
	s.tipos[chave] = contentType

	return nil
}

func (s *Storage) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dados, ok := s.conteudos[chave]
	if !ok {
		return nil, io.EOF
	}

	return io.NopCloser(bytes.NewReader(dados)), nil
}

func (s *Storage) Remover(_ context.Context, chave string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conteudos, chave)
	delete(s.tipos, chave)

	return nil
}
