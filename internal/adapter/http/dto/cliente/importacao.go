package cliente

import (
	"io"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
)

var CabecalhoImportacao = []string{
	"documento", "nome", "nome_fantasia", "tipo", "grupo", "uf", "cidade", "email_nf", "telefone",
	"representante_documento", "exige_numero_pedido_na_nf", "proibe_antecipacao", "observacoes",
}

var ExemploImportacao = []string{
	"11.222.333/0001-81", "Construtora Exemplo LTDA", "Exemplo", "construtora", "Grupo Exemplo", "PB", "João Pessoa",
	"nf@exemplo.com.br", "83 3333-0000", "", "nao", "nao", "",
}

func LerLinhasCSV(conteudo io.Reader) ([]portsin.LinhaImportacao, error) {
	linhasCSV, err := dto.LerCSV(conteudo, CabecalhoImportacao, domaincliente.LimiteDeLinhasDaImportacao)
	if err != nil {
		return nil, err
	}

	linhas := make([]portsin.LinhaImportacao, 0, len(linhasCSV))
	for _, linha := range linhasCSV {
		campos := linha.Campos
		linhas = append(linhas, portsin.LinhaImportacao{
			Numero:                 linha.Numero,
			Documento:              campos[0],
			Nome:                   campos[1],
			NomeFantasia:           campos[2],
			Tipo:                   campos[3],
			Grupo:                  campos[4],
			UF:                     campos[5],
			Cidade:                 campos[6],
			EmailNF:                campos[7],
			Telefone:               campos[8],
			RepresentanteDocumento: campos[9],
			ExigeNumeroPedidoNaNF:  campos[10],
			ProibeAntecipacao:      campos[11],
			Observacoes:            campos[12],
		})
	}

	return linhas, nil
}

type LinhaRejeitadaResponse struct {
	Linha     int    `json:"linha"`
	Documento string `json:"documento"`
	Motivo    string `json:"motivo"`
}

type RelatorioImportacaoResponse struct {
	Simulacao  bool                     `json:"simulacao"`
	Total      int                      `json:"total"`
	Importadas int                      `json:"importadas"`
	Rejeitadas []LinhaRejeitadaResponse `json:"rejeitadas"`
}

func NovoRelatorioImportacaoResponse(relatorio *domaincliente.RelatorioImportacao) RelatorioImportacaoResponse {
	rejeitadas := make([]LinhaRejeitadaResponse, 0, len(relatorio.Rejeitadas))
	for _, rejeitada := range relatorio.Rejeitadas {
		rejeitadas = append(rejeitadas, LinhaRejeitadaResponse(rejeitada))
	}

	return RelatorioImportacaoResponse{
		Simulacao:  relatorio.Simulacao,
		Total:      relatorio.Total,
		Importadas: relatorio.Importadas,
		Rejeitadas: rejeitadas,
	}
}

type ConverterLeadRequest struct {
	Documento string `json:"documento"`
	DadosClienteRequest
}
