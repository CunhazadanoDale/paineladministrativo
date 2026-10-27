package domain

import (
	"errors"
	"strconv"
	"strings"
)

const LimiteDeLinhasDaImportacao = 5000

type LinhaRejeitada struct {
	Linha  int
	Chave  string
	Motivo string
}

type RelatorioImportacao struct {
	Simulacao  bool
	Total      int
	Importadas int
	Rejeitadas []LinhaRejeitada
}

func NovoRelatorioImportacao(simulacao bool, total int) (*RelatorioImportacao, error) {
	if total > LimiteDeLinhasDaImportacao {
		return nil, ErroValidacao("arquivo com mais de " + strconv.Itoa(LimiteDeLinhasDaImportacao) + " linhas")
	}

	return &RelatorioImportacao{Simulacao: simulacao, Total: total, Rejeitadas: []LinhaRejeitada{}}, nil
}

func (r *RelatorioImportacao) Rejeitar(linha int, chave, motivo string) {
	r.Rejeitadas = append(r.Rejeitadas, LinhaRejeitada{Linha: linha, Chave: chave, Motivo: motivo})
}

func MotivoDeRejeicao(err error) (string, error) {
	for _, sentinela := range []error{ErrValidacao, ErrConflito, ErrNotFound} {
		if errors.Is(err, sentinela) {
			return strings.TrimPrefix(err.Error(), sentinela.Error()+": "), nil
		}
	}

	return "", err
}
