package cliente

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const LimiteDeLinhasDaImportacao = 5000

type LinhaRejeitada struct {
	Linha     int
	Documento string
	Motivo    string
}

type RelatorioImportacao struct {
	Simulacao  bool
	Total      int
	Importadas int
	Rejeitadas []LinhaRejeitada
}

func (r *RelatorioImportacao) Rejeitar(linha int, documento, motivo string) {
	r.Rejeitadas = append(r.Rejeitadas, LinhaRejeitada{Linha: linha, Documento: documento, Motivo: motivo})
}

func FlagDeImportacao(valor, campo string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(valor)) {
	case "", "nao", "não", "n":
		return false, nil
	case "sim", "s":
		return true, nil
	default:
		return false, domain.ErroValidacao(campo + " deve ser sim ou nao")
	}
}
