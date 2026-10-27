package cliente

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
