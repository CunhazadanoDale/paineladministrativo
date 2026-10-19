package pedido

import (
	"strconv"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EmpresaDoDocumentoResponse struct {
	ID           uuid.UUID `json:"id"`
	RazaoSocial  string    `json:"razao_social"`
	NomeFantasia string    `json:"nome_fantasia"`
	CNPJ         string    `json:"cnpj"`
	UF           string    `json:"uf"`
}

type ClienteDoDocumentoResponse struct {
	ID        uuid.UUID `json:"id"`
	Nome      string    `json:"nome"`
	Documento string    `json:"documento"`
	Cidade    string    `json:"cidade"`
	UF        string    `json:"uf"`
	Telefone  *string   `json:"telefone"`
	Email     *string   `json:"email"`
}

type ObraDoDocumentoResponse struct {
	ID       uuid.UUID `json:"id"`
	Nome     string    `json:"nome"`
	Endereco string    `json:"endereco"`
}

type RepresentanteDoDocumentoResponse struct {
	ID       uuid.UUID `json:"id"`
	Nome     string    `json:"nome"`
	Email    *string   `json:"email"`
	Telefone *string   `json:"telefone"`
}

type ItemDoDocumentoResponse struct {
	ItemID                   uuid.UUID `json:"item_id"`
	ProdutoID                uuid.UUID `json:"produto_id"`
	Codigo                   *string   `json:"codigo"`
	Nome                     string    `json:"nome"`
	Unidade                  string    `json:"unidade_medida"`
	Quantidade               int       `json:"quantidade"`
	PrecoUnitarioCentavos    int64     `json:"preco_unitario_centavos"`
	PrecoTabelaCentavos      *int64    `json:"preco_tabela_centavos"`
	DescontoUnitarioCentavos *int64    `json:"desconto_unitario_centavos"`
	SubtotalCentavos         int64     `json:"subtotal_centavos"`
}

type DocumentoPropostaResponse struct {
	PedidoID              uuid.UUID                         `json:"pedido_id"`
	Numero                int                               `json:"numero"`
	Revisao               int                               `json:"revisao"`
	Identificacao         string                            `json:"identificacao"`
	Status                string                            `json:"status"`
	Modelo                string                            `json:"modelo"`
	EmitidoEm             string                            `json:"emitido_em"`
	Validade              *string                           `json:"validade"`
	Vencida               bool                              `json:"vencida"`
	Empresa               EmpresaDoDocumentoResponse        `json:"empresa"`
	Cliente               ClienteDoDocumentoResponse        `json:"cliente"`
	Obra                  *ObraDoDocumentoResponse          `json:"obra"`
	Representante         *RepresentanteDoDocumentoResponse `json:"representante"`
	Itens                 []ItemDoDocumentoResponse         `json:"itens"`
	TotalCentavos         int64                             `json:"total_centavos"`
	DescontoCentavos      int64                             `json:"desconto_centavos"`
	Condicoes             []string                          `json:"condicoes"`
	FreteTipo             *string                           `json:"frete_tipo"`
	FreteEstimadoCentavos *int64                            `json:"frete_estimado_centavos"`
	NumeroPedidoCliente   *string                           `json:"numero_pedido_cliente"`
	Observacoes           string                            `json:"observacoes"`
}

type ModeloDeVendaResponse struct {
	Modelo    string           `json:"modelo"`
	Condicao  CondicaoResponse `json:"condicao"`
	Condicoes []string         `json:"condicoes"`
}

func NovoDocumentoPropostaResponse(documento *domainpedido.DocumentoProposta) DocumentoPropostaResponse {
	resposta := DocumentoPropostaResponse{
		PedidoID:            documento.PedidoID,
		Numero:              documento.Numero,
		Revisao:             documento.Revisao,
		Identificacao:       identificacaoDaProposta(documento.Numero, documento.Revisao),
		Status:              documento.Status.String(),
		Modelo:              documento.Modelo.String(),
		EmitidoEm:           documento.EmitidoEm.Format(formatoData),
		Vencida:             documento.Vencida,
		Empresa:             EmpresaDoDocumentoResponse(documento.Empresa),
		Cliente:             ClienteDoDocumentoResponse(documento.Cliente),
		Itens:               make([]ItemDoDocumentoResponse, 0, len(documento.Itens)),
		TotalCentavos:       documento.TotalCentavos,
		DescontoCentavos:    documento.DescontoCentavos,
		Condicoes:           documento.Condicoes,
		NumeroPedidoCliente: documento.NumeroPedidoCliente,
		Observacoes:         documento.Observacoes,
	}
	if documento.Validade != nil {
		validade := documento.Validade.Format(formatoData)
		resposta.Validade = &validade
	}
	if documento.Obra != nil {
		obra := ObraDoDocumentoResponse(*documento.Obra)
		resposta.Obra = &obra
	}
	if documento.Representante != nil {
		representante := RepresentanteDoDocumentoResponse(*documento.Representante)
		resposta.Representante = &representante
	}
	if documento.Frete != nil {
		tipo := documento.Frete.Tipo().String()
		resposta.FreteTipo = &tipo
		resposta.FreteEstimadoCentavos = documento.Frete.Centavos()
	}
	for _, item := range documento.Itens {
		resposta.Itens = append(resposta.Itens, ItemDoDocumentoResponse(item))
	}

	return resposta
}

func NovoModeloDeVendaResponses(modelos []domainpedido.ModeloDeVenda) []ModeloDeVendaResponse {
	respostas := make([]ModeloDeVendaResponse, 0, len(modelos))
	for _, modelo := range modelos {
		respostas = append(respostas, ModeloDeVendaResponse{
			Modelo: modelo.Modelo.String(),
			Condicao: CondicaoResponse{
				EntradaBP:        modelo.Condicao.Entrada().PontosBase(),
				PrazoBoletoDias:  modelo.Condicao.PrazoBoletoDias(),
				PrazoEntregaDias: modelo.Condicao.PrazoEntregaDias(),
			},
			Condicoes: modelo.Condicoes,
		})
	}

	return respostas
}

func identificacaoDaProposta(numero, revisao int) string {
	return "Proposta " + strconv.Itoa(numero) + "-R" + strconv.Itoa(revisao)
}
