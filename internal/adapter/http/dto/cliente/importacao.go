package cliente

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
)

var CabecalhoImportacao = []string{
	"documento", "nome", "nome_fantasia", "tipo", "grupo", "uf", "cidade", "email_nf", "telefone",
	"representante_documento", "exige_numero_pedido_na_nf", "proibe_antecipacao", "observacoes",
}

var marcaDeOrdemDeBytes = []byte{0xEF, 0xBB, 0xBF}

func LerLinhasCSV(conteudo io.Reader) ([]portsin.LinhaImportacao, error) {
	leitor := bufio.NewReader(conteudo)
	if inicio, err := leitor.Peek(len(marcaDeOrdemDeBytes)); err == nil && bytes.Equal(inicio, marcaDeOrdemDeBytes) {
		_, _ = leitor.Discard(len(marcaDeOrdemDeBytes))
	}

	csvLeitor := csv.NewReader(leitor)
	csvLeitor.Comma = ';'
	csvLeitor.FieldsPerRecord = len(CabecalhoImportacao)
	csvLeitor.TrimLeadingSpace = true

	cabecalho, err := csvLeitor.Read()
	if errors.Is(err, io.EOF) {
		return nil, domain.ErroValidacao("arquivo vazio")
	}
	if err != nil {
		return nil, erroDeLeitura(err)
	}
	if err := exigirUTF8(csvLeitor, cabecalho); err != nil {
		return nil, err
	}
	for indice, coluna := range cabecalho {
		if strings.ToLower(strings.TrimSpace(coluna)) != CabecalhoImportacao[indice] {
			return nil, domain.ErroValidacao("cabeçalho deve ser " + strings.Join(CabecalhoImportacao, ";"))
		}
	}

	linhas := make([]portsin.LinhaImportacao, 0)
	for {
		campos, err := csvLeitor.Read()
		if errors.Is(err, io.EOF) {
			return linhas, nil
		}
		if err != nil {
			return nil, erroDeLeitura(err)
		}
		if err := exigirUTF8(csvLeitor, campos); err != nil {
			return nil, err
		}
		if len(linhas) == domaincliente.LimiteDeLinhasDaImportacao {
			return nil, domain.ErroValidacao("arquivo com mais de " + strconv.Itoa(domaincliente.LimiteDeLinhasDaImportacao) + " linhas")
		}

		numero, _ := csvLeitor.FieldPos(0)
		linhas = append(linhas, portsin.LinhaImportacao{
			Numero:                 numero,
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
}

func exigirUTF8(csvLeitor *csv.Reader, campos []string) error {
	for indice, campo := range campos {
		if !utf8.ValidString(campo) {
			linha, _ := csvLeitor.FieldPos(indice)
			return domain.ErroValidacao("linha " + strconv.Itoa(linha) + " não está em UTF-8; salve o arquivo como CSV UTF-8")
		}
	}

	return nil
}

func erroDeLeitura(err error) error {
	var erroCSV *csv.ParseError
	if errors.As(err, &erroCSV) {
		if errors.Is(erroCSV.Err, csv.ErrFieldCount) {
			return domain.ErroValidacao("linha " + strconv.Itoa(erroCSV.Line) + " deve ter " + strconv.Itoa(len(CabecalhoImportacao)) + " colunas separadas por ;")
		}
		return domain.ErroValidacao("csv inválido na linha " + strconv.Itoa(erroCSV.Line))
	}

	return err
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
