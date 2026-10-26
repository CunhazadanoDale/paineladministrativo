package dto

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const marcaDeOrdemUTF8 = "\xef\xbb\xbf"

var windows1252 = [32]rune{
	'€', 0x81, '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', 0x8D, 'Ž', 0x8F,
	0x90, '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ', 0x9D, 'ž', 'Ÿ',
}

type LinhaCSV struct {
	Numero int
	Campos []string
}

func LerCSV(conteudo io.Reader, cabecalho []string, limiteDeLinhas int) ([]LinhaCSV, error) {
	bruto, err := io.ReadAll(conteudo)
	if err != nil {
		return nil, err
	}

	csvLeitor := csv.NewReader(strings.NewReader(textoUTF8(bruto)))
	csvLeitor.Comma = ';'
	csvLeitor.FieldsPerRecord = len(cabecalho)
	csvLeitor.TrimLeadingSpace = true

	recebido, err := csvLeitor.Read()
	if errors.Is(err, io.EOF) {
		return nil, domain.ErroValidacao("arquivo vazio")
	}
	if err != nil {
		return nil, erroDeLeituraCSV(err, len(cabecalho))
	}
	for indice, coluna := range recebido {
		if strings.ToLower(strings.TrimSpace(coluna)) != cabecalho[indice] {
			return nil, domain.ErroValidacao("cabeçalho deve ser " + strings.Join(cabecalho, ";"))
		}
	}

	linhas := make([]LinhaCSV, 0)
	for {
		campos, err := csvLeitor.Read()
		if errors.Is(err, io.EOF) {
			return linhas, nil
		}
		if err != nil {
			return nil, erroDeLeituraCSV(err, len(cabecalho))
		}
		if len(linhas) == limiteDeLinhas {
			return nil, domain.ErroValidacao("arquivo com mais de " + strconv.Itoa(limiteDeLinhas) + " linhas")
		}

		numero, _ := csvLeitor.FieldPos(0)
		linhas = append(linhas, LinhaCSV{Numero: numero, Campos: campos})
	}
}

func ModeloCSV(cabecalho []string, exemplo []string) string {
	return marcaDeOrdemUTF8 + strings.Join(cabecalho, ";") + "\r\n" + strings.Join(exemplo, ";") + "\r\n"
}

func textoUTF8(conteudo []byte) string {
	if utf8.Valid(conteudo) {
		return strings.TrimPrefix(string(conteudo), marcaDeOrdemUTF8)
	}

	var texto strings.Builder
	texto.Grow(len(conteudo))
	for _, b := range conteudo {
		switch {
		case b >= 0x80 && b < 0xA0:
			texto.WriteRune(windows1252[b-0x80])
		default:
			texto.WriteRune(rune(b))
		}
	}

	return texto.String()
}

func erroDeLeituraCSV(err error, colunas int) error {
	var erroCSV *csv.ParseError
	if errors.As(err, &erroCSV) {
		if errors.Is(erroCSV.Err, csv.ErrFieldCount) {
			return domain.ErroValidacao("linha " + strconv.Itoa(erroCSV.Line) + " deve ter " + strconv.Itoa(colunas) + " colunas separadas por ;")
		}
		return domain.ErroValidacao("csv inválido na linha " + strconv.Itoa(erroCSV.Line))
	}

	return err
}
