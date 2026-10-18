package nfe

import (
	"fmt"
	"strconv"
	"strings"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
)

const tamanhoCPF = 11

type Item struct {
	Codigo     string
	CFOP       string
	Quantidade string
	Valor      string
	Pedido     string
}

type Duplicata struct {
	Numero     string
	Vencimento string
	Valor      string
}

type NFe struct {
	CNPJEmitente      string
	Destinatario      string
	NomeDestinatario  string
	UF                string
	Cidade            string
	Email             string
	Finalidade        string
	Emissao           string
	Numero            int
	Serie             int
	PedidoCompra      string
	Itens             []Item
	Duplicatas        []Duplicata
	StatusAutorizacao string
}

func Nova(cnpjEmitente, destinatario string, numero int) NFe {
	return NFe{
		CNPJEmitente:     cnpjEmitente,
		Destinatario:     destinatario,
		NomeDestinatario: "Construtora Exemplo SPE",
		UF:               "PB",
		Cidade:           "Joao Pessoa",
		Email:            "nf@exemplo.com.br",
		Finalidade:       "1",
		Emissao:          "2026-10-10T09:30:00-03:00",
		Numero:           numero,
		Serie:            1,
		Itens: []Item{
			{Codigo: "PISO-01", CFOP: "5102", Quantidade: "10.5000", Valor: "1050.00"},
			{Codigo: "REJ-02", CFOP: "5102", Quantidade: "2", Valor: "40.00", Pedido: "PC-77"},
		},
		Duplicatas: []Duplicata{
			{Numero: "001", Vencimento: "2026-11-09", Valor: "545.00"},
			{Numero: "002", Vencimento: "2026-12-09", Valor: "545.00"},
		},
		StatusAutorizacao: "100",
	}
}

func ChaveDe(cnpjEmitente, emissao string, serie, numero int) string {
	base := fmt.Sprintf("25%s%s%s55%03d%09d1%08d", emissao[2:4], emissao[5:7], cnpjEmitente, serie, numero, 12345678)
	for digito := 0; digito <= 9; digito++ {
		candidata := base + strconv.Itoa(digito)
		if _, err := domainnotafiscal.NovaChave(candidata); err == nil {
			return candidata
		}
	}

	return base + "0"
}

func (n NFe) Chave() string {
	return ChaveDe(n.CNPJEmitente, n.Emissao, n.Serie, n.Numero)
}

func (n NFe) XML() []byte {
	chave := n.Chave()

	var itens strings.Builder
	for indice, item := range n.Itens {
		pedido := ""
		if item.Pedido != "" {
			pedido = "<xPed>" + item.Pedido + "</xPed>"
		}
		fmt.Fprintf(&itens, `<det nItem="%d"><prod><cProd>%s</cProd><cEAN>SEM GTIN</cEAN><xProd>Produto %s</xProd><NCM>69072100</NCM><CFOP>%s</CFOP><uCom>M2</uCom><qCom>%s</qCom><vUnCom>100.0000000000</vUnCom><vProd>%s</vProd>%s</prod></det>`,
			indice+1, item.Codigo, item.Codigo, item.CFOP, item.Quantidade, item.Valor, pedido)
	}

	cobranca := ""
	if len(n.Duplicatas) > 0 {
		var duplicatas strings.Builder
		for _, dup := range n.Duplicatas {
			fmt.Fprintf(&duplicatas, "<dup><nDup>%s</nDup><dVenc>%s</dVenc><vDup>%s</vDup></dup>", dup.Numero, dup.Vencimento, dup.Valor)
		}
		cobranca = "<cobr>" + duplicatas.String() + "</cobr>"
	}

	compra := ""
	if n.PedidoCompra != "" {
		compra = "<compra><xPed>" + n.PedidoCompra + "</xPed></compra>"
	}

	tagDestino := "CNPJ"
	if len(n.Destinatario) == tamanhoCPF {
		tagDestino = "CPF"
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><nfeProc xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00"><NFe><infNFe Id="NFe%s" versao="4.00"><ide><cUF>25</cUF><cNF>12345678</cNF><natOp>VENDA DE MERCADORIA</natOp><mod>55</mod><serie>%d</serie><nNF>%d</nNF><dhEmi>%s</dhEmi><tpNF>1</tpNF><tpEmis>1</tpEmis><cDV>%s</cDV><tpAmb>1</tpAmb><finNFe>%s</finNFe></ide><emit><CNPJ>%s</CNPJ><xNome>Emitente</xNome></emit><dest><%s>%s</%s><xNome>%s</xNome><enderDest><xMun>%s</xMun><UF>%s</UF></enderDest><email>%s</email></dest>%s<total><ICMSTot><vProd>1090.00</vProd><vFrete>0.00</vFrete><vDesc>0.00</vDesc><vNF>1090.00</vNF></ICMSTot></total>%s%s</infNFe></NFe><protNFe versao="4.00"><infProt><tpAmb>1</tpAmb><chNFe>%s</chNFe><dhRecbto>2026-10-10T09:31:02-03:00</dhRecbto><nProt>325260000012345</nProt><cStat>%s</cStat></infProt></protNFe></nfeProc>`,
		chave, n.Serie, n.Numero, n.Emissao, chave[43:], n.Finalidade, n.CNPJEmitente,
		tagDestino, n.Destinatario, tagDestino, n.NomeDestinatario, n.Cidade, n.UF, n.Email,
		itens.String(), cobranca, compra, chave, n.StatusAutorizacao))
}

type Evento struct {
	Chave     string
	Autor     string
	Codigo    string
	Sequencia int
	Texto     string
}

func Cancelamento(chave, autor string) Evento {
	return Evento{Chave: chave, Autor: autor, Codigo: "110111", Sequencia: 1, Texto: "Nota emitida com erro de valor"}
}

func CartaCorrecao(chave, autor string, sequencia int) Evento {
	return Evento{Chave: chave, Autor: autor, Codigo: "110110", Sequencia: sequencia, Texto: "Corrige o endereco de entrega"}
}

func (e Evento) XML() []byte {
	detalhe := "<descEvento>Cancelamento</descEvento><nProt>325260000012345</nProt><xJust>" + e.Texto + "</xJust>"
	if e.Codigo == "110110" {
		detalhe = "<descEvento>Carta de Correcao</descEvento><xCorrecao>" + e.Texto + "</xCorrecao>"
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><procEventoNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00"><evento versao="1.00"><infEvento Id="ID%s%s%02d"><cOrgao>25</cOrgao><tpAmb>1</tpAmb><CNPJ>%s</CNPJ><chNFe>%s</chNFe><dhEvento>2026-10-11T14:00:00-03:00</dhEvento><tpEvento>%s</tpEvento><nSeqEvento>%d</nSeqEvento><verEvento>1.00</verEvento><detEvento versao="1.00">%s</detEvento></infEvento></evento><retEvento versao="1.00"><infEvento><tpAmb>1</tpAmb><cStat>135</cStat><chNFe>%s</chNFe><tpEvento>%s</tpEvento><nSeqEvento>%d</nSeqEvento><nProt>325260000099999</nProt></infEvento></retEvento></procEventoNFe>`,
		e.Codigo, e.Chave, e.Sequencia, e.Autor, e.Chave, e.Codigo, e.Sequencia, detalhe, e.Chave, e.Codigo, e.Sequencia))
}
