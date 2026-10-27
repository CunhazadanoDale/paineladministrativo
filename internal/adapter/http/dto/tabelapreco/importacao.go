package tabelapreco

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
)

var CabecalhoImportacao = []string{"codigo", "modelo", "preco", "vigente_de", "observacao"}

var ExemploImportacao = []string{"CIM-50", "pronta_entrega", "39,90", "01/12/2026", "reajuste do fornecedor"}

func NovasLinhasImportacao(linhasCSV []dto.LinhaCSV) []portsin.LinhaImportacao {
	linhas := make([]portsin.LinhaImportacao, 0, len(linhasCSV))
	for _, linha := range linhasCSV {
		linhas = append(linhas, portsin.LinhaImportacao{
			Numero:     linha.Numero,
			Codigo:     linha.Campos[0],
			Modelo:     linha.Campos[1],
			Preco:      linha.Campos[2],
			VigenteDe:  linha.Campos[3],
			Observacao: linha.Campos[4],
		})
	}

	return linhas
}
