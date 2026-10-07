package disco

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	portssolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
)

var _ portssolicitacao.Storage = (*Storage)(nil)

type Storage struct {
	dir string
}

func Novo(dir string) *Storage {
	return &Storage{dir: dir}
}

func (s *Storage) Enviar(_ context.Context, chave string, conteudo io.Reader, _ string) error {
	caminho, err := s.caminho(chave)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return err
	}

	destino, err := os.Create(caminho)
	if err != nil {
		return err
	}
	defer destino.Close()

	_, err = io.Copy(destino, conteudo)

	return err
}

func (s *Storage) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	caminho, err := s.caminho(chave)
	if err != nil {
		return nil, err
	}

	return os.Open(caminho)
}

func (s *Storage) Remover(_ context.Context, chave string) error {
	caminho, err := s.caminho(chave)
	if err != nil {
		return err
	}

	if err := os.Remove(caminho); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (s *Storage) caminho(chave string) (string, error) {
	limpa := filepath.Clean("/" + chave)
	if strings.Contains(limpa, "..") {
		return "", os.ErrInvalid
	}

	return filepath.Join(s.dir, filepath.FromSlash(limpa)), nil
}
