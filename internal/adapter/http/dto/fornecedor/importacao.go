package fornecedor

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
)

var CabecalhoImportacao = []string{
	"pais", "documento", "identificacao_estrangeira", "nome", "nome_fantasia", "moeda_padrao",
	"uf", "cidade", "contato_nome", "email", "telefone", "observacoes",
}

var ExemploImportacao = []string{
	"BR", "11.222.333/0001-81", "", "Cerâmica Paraíba LTDA", "Cerâmica PB", "BRL",
	"PB", "João Pessoa", "Maria", "vendas@ceramicapb.com.br", "83 3333-0000", "",
}

func NovasLinhasImportacao(linhasCSV []dto.LinhaCSV) []portsin.LinhaImportacao {
	linhas := make([]portsin.LinhaImportacao, 0, len(linhasCSV))
	for _, linha := range linhasCSV {
		campos := linha.Campos
		linhas = append(linhas, portsin.LinhaImportacao{
			Numero:                   linha.Numero,
			Pais:                     campos[0],
			Documento:                campos[1],
			IdentificacaoEstrangeira: campos[2],
			Nome:                     campos[3],
			NomeFantasia:             campos[4],
			MoedaPadrao:              campos[5],
			UF:                       campos[6],
			Cidade:                   campos[7],
			ContatoNome:              campos[8],
			Email:                    campos[9],
			Telefone:                 campos[10],
			Observacoes:              campos[11],
		})
	}

	return linhas
}
