package disco

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type leitorComFalha struct{}

func (leitorComFalha) Read([]byte) (int, error) {
	return 0, errors.New("conexão interrompida")
}

func TestEnviarEBaixarAceitaPontosNoMeioDoNome(t *testing.T) {
	storage := Novo(t.TempDir())
	ctx := context.Background()
	chave := "dono/arquivo/nota..final.pdf"

	if err := storage.Enviar(ctx, chave, strings.NewReader("conteúdo"), "application/pdf"); err != nil {
		t.Fatalf("envio falhou: %v", err)
	}

	leitor, err := storage.Baixar(ctx, chave)
	if err != nil {
		t.Fatalf("download falhou: %v", err)
	}
	defer leitor.Close()

	conteudo, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("leitura falhou: %v", err)
	}
	if string(conteudo) != "conteúdo" {
		t.Errorf("conteúdo %q, esperado %q", conteudo, "conteúdo")
	}
}

func TestChaveForaDoDiretorioRecebeErroDeValidacao(t *testing.T) {
	storage := Novo(t.TempDir())
	ctx := context.Background()

	for _, chave := range []string{"../fora.pdf", "dono/../../fora.pdf", "", "/absoluta.pdf"} {
		if err := storage.Enviar(ctx, chave, strings.NewReader("x"), "application/pdf"); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("Enviar(%q) = %v, esperado erro de validação", chave, err)
		}
		if _, err := storage.Baixar(ctx, chave); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("Baixar(%q) = %v, esperado erro de validação", chave, err)
		}
	}
}

func TestBaixarArquivoAusenteRecebeNaoEncontrado(t *testing.T) {
	storage := Novo(t.TempDir())

	if _, err := storage.Baixar(context.Background(), "dono/arquivo/sumiu.pdf"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado não encontrado", err)
	}
}

func TestEnvioInterrompidoNaoDeixaArquivo(t *testing.T) {
	dir := t.TempDir()
	storage := Novo(dir)
	chave := "dono/arquivo/nota.pdf"

	if err := storage.Enviar(context.Background(), chave, leitorComFalha{}, "application/pdf"); err == nil {
		t.Fatal("envio com falha de leitura deveria falhar")
	}

	entradas, err := os.ReadDir(filepath.Join(dir, "dono", "arquivo"))
	if err != nil {
		t.Fatalf("leitura do diretório falhou: %v", err)
	}
	if len(entradas) != 0 {
		t.Errorf("%d arquivos sobraram após o envio interrompido", len(entradas))
	}
}

func TestRemoverArquivoAusenteNaoFalha(t *testing.T) {
	storage := Novo(t.TempDir())

	if err := storage.Remover(context.Background(), "dono/arquivo/sumiu.pdf"); err != nil {
		t.Errorf("remoção de arquivo ausente = %v, esperado sucesso", err)
	}
}
