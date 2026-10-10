package disco

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
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

	temporario, err := os.CreateTemp(filepath.Dir(caminho), ".envio-*")
	if err != nil {
		return err
	}

	if err := gravar(temporario, conteudo); err != nil {
		_ = os.Remove(temporario.Name())
		return err
	}

	if err := os.Rename(temporario.Name(), caminho); err != nil {
		_ = os.Remove(temporario.Name())
		return err
	}

	return nil
}

func (s *Storage) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	caminho, err := s.caminho(chave)
	if err != nil {
		return nil, err
	}

	arquivo, err := os.Open(caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, domain.ErroNaoEncontrado("arquivo não encontrado no armazenamento")
	}

	return arquivo, err
}

func (s *Storage) Remover(_ context.Context, chave string) error {
	caminho, err := s.caminho(chave)
	if err != nil {
		return err
	}

	if err := os.Remove(caminho); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return nil
}

func (s *Storage) caminho(chave string) (string, error) {
	relativo := filepath.FromSlash(chave)
	if !filepath.IsLocal(relativo) {
		return "", domain.ErroValidacao("chave de arquivo inválida para o armazenamento")
	}

	return filepath.Join(s.dir, relativo), nil
}

func gravar(destino *os.File, conteudo io.Reader) error {
	if _, err := io.Copy(destino, conteudo); err != nil {
		_ = destino.Close()
		return err
	}

	return destino.Close()
}
