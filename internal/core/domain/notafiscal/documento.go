package notafiscal

import (
	"time"
)

type Documento struct {
	Chave               Chave
	Serie               int
	Numero              int
	EmitidaEm           time.Time
	TipoOperacao        TipoOperacao
	Finalidade          Finalidade
	NaturezaOperacao    string
	EmitenteCNPJ        string
	Destinatario        Destinatario
	Itens               []ItemDocumento
	Duplicatas          []DuplicataDocumento
	Totais              Totais
	NumeroPedidoCliente *string
	Protocolo           string
	AutorizadaEm        time.Time
}

type Destinatario struct {
	Documento string
	Nome      string
	UF        string
	Cidade    string
	Email     string
}

type ItemDocumento struct {
	NumeroItem              int
	Codigo                  string
	Descricao               string
	NCM                     string
	CFOP                    string
	Unidade                 string
	QuantidadeDecimilesimos int64
	ValorUnitario           string
	ValorCentavos           int64
	DescontoCentavos        int64
	PedidoCliente           *string
}

type DuplicataDocumento struct {
	Numero        string
	Vencimento    time.Time
	ValorCentavos int64
}

type Totais struct {
	ProdutosCentavos int64
	FreteCentavos    int64
	DescontoCentavos int64
	TotalCentavos    int64
}

type DocumentoEvento struct {
	Chave         Chave
	AutorCNPJ     string
	Tipo          TipoEvento
	Sequencia     int
	OcorridoEm    time.Time
	Protocolo     string
	Justificativa string
}

func (d *Documento) CFOPPredominante() string {
	var predominante *ItemDocumento
	for indice := range d.Itens {
		item := &d.Itens[indice]
		if predominante == nil || item.ValorCentavos > predominante.ValorCentavos ||
			(item.ValorCentavos == predominante.ValorCentavos && item.NumeroItem < predominante.NumeroItem) {
			predominante = item
		}
	}
	if predominante == nil {
		return ""
	}

	return predominante.CFOP
}
