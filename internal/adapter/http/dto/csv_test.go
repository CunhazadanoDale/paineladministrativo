package dto

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

var cabecalhoDeTeste = []string{"codigo", "nome"}

func TestLerCSVDevolveLinhasComNumeroDoArquivo(t *testing.T) {
	conteudo := "\xef\xbb\xbfCODIGO; Nome\r\nA1;\"Tijolo; 8 furos\"\r\n\r\nB2;Areia\r\n"

	linhas, err := LerCSV(strings.NewReader(conteudo), cabecalhoDeTeste, 10)

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(linhas) != 2 || linhas[0].Numero != 2 || linhas[0].Campos[1] != "Tijolo; 8 furos" || linhas[1].Numero != 4 || linhas[1].Campos[0] != "B2" {
		t.Errorf("linhas = %+v", linhas)
	}
}

func TestLerCSVConverteWindows1252(t *testing.T) {
	linhas, err := LerCSV(strings.NewReader("codigo;nome\nA1;Cimento \x96 saco \x80 50kg \xe7\xe3\n"), cabecalhoDeTeste, 10)

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if linhas[0].Campos[1] != "Cimento – saco € 50kg çã" {
		t.Errorf("campo convertido = %q", linhas[0].Campos[1])
	}
}

func TestLerCSVRecusaArquivoInvalido(t *testing.T) {
	casos := []struct {
		nome     string
		conteudo string
		mensagem string
	}{
		{"vazio", "", "erro de validação: arquivo vazio"},
		{"cabeçalho trocado", "codigo;descricao\n", "erro de validação: cabeçalho deve ser codigo;nome"},
		{"colunas a menos", "codigo;nome\nA1\n", "erro de validação: linha 2 deve ter 2 colunas separadas por ;"},
		{"aspas abertas", "codigo;nome\n\"A1;Areia\n", "erro de validação: csv inválido na linha 2"},
		{"acima do limite", "codigo;nome\nA;1\nB;2\nC;3\n", "erro de validação: arquivo com mais de 2 linhas"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := LerCSV(strings.NewReader(caso.conteudo), cabecalhoDeTeste, 2)

			if !errors.Is(err, domain.ErrValidacao) || err.Error() != caso.mensagem {
				t.Errorf("erro = %v, esperado %q", err, caso.mensagem)
			}
		})
	}
}

func TestModeloCSVTrazBOMCabecalhoEExemplo(t *testing.T) {
	if modelo := ModeloCSV(cabecalhoDeTeste, []string{"A1", "Areia"}); modelo != "\xef\xbb\xbfcodigo;nome\r\nA1;Areia\r\n" {
		t.Errorf("modelo = %q", modelo)
	}
}
