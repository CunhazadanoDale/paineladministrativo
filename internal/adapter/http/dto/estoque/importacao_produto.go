package estoque

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

var CabecalhoImportacaoProduto = []string{
	"nome", "codigo", "categoria", "unidade_medida", "preco", "preco_promocional", "estoque_minimo", "peso_kg", "descricao", "destaque",
}

var ExemploImportacaoProduto = []string{
	"Cimento CP II 50kg", "CIM-50", "Cimentos", "sc", "39,90", "", "20", "50", "Saco de 50 kg", "nao",
}

func NovasLinhasImportacaoProduto(linhasCSV []dto.LinhaCSV) []portsin.LinhaImportacaoProduto {
	linhas := make([]portsin.LinhaImportacaoProduto, 0, len(linhasCSV))
	for _, linha := range linhasCSV {
		campos := linha.Campos
		linhas = append(linhas, portsin.LinhaImportacaoProduto{
			Numero:           linha.Numero,
			Nome:             campos[0],
			Codigo:           campos[1],
			Categoria:        campos[2],
			UnidadeMedida:    campos[3],
			Preco:            campos[4],
			PrecoPromocional: campos[5],
			EstoqueMinimo:    campos[6],
			PesoKg:           campos[7],
			Descricao:        campos[8],
			Destaque:         campos[9],
		})
	}

	return linhas
}
