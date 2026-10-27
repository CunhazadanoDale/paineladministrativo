package lead

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

var CabecalhoImportacao = []string{"nome", "email", "telefone", "origem"}

var ExemploImportacao = []string{"Construtora Exemplo", "contato@exemplo.com.br", "(83) 99999-0000", "feira de construção"}

func NovasLinhasImportacao(linhasCSV []dto.LinhaCSV) []portsin.LinhaImportacao {
	linhas := make([]portsin.LinhaImportacao, 0, len(linhasCSV))
	for _, linha := range linhasCSV {
		linhas = append(linhas, portsin.LinhaImportacao{
			Numero:   linha.Numero,
			Nome:     linha.Campos[0],
			Email:    linha.Campos[1],
			Telefone: linha.Campos[2],
			Origem:   linha.Campos[3],
		})
	}

	return linhas
}
