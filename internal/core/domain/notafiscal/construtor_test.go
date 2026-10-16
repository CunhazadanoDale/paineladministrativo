package notafiscal_test

import (
	"fmt"
	"strconv"
	"strings"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
)

const (
	cnpjEmitente     = "11222333000181"
	cnpjDestinatario = "11444777000161"
	cpfDestinatario  = "52998224725"
	cnpjAlfanumerico = "12ABC34501DE35"
)

type itemXML struct {
	numero, codigo, cfop, quantidade, valor, desconto, pedido string
}

type duplicataXML struct {
	numero, vencimento, valor string
}

type nfeXML struct {
	cnpjEmitente   string
	destinoTag     string
	destino        string
	finalidade     string
	tipoOperacao   string
	ambiente       string
	modelo         string
	statusProt     string
	emissao        string
	dhEmi          string
	numero         int
	serie          int
	itens          []itemXML
	duplicatas     []duplicataXML
	pedidoCompra   string
	semProtocolo   bool
	chaveProtocolo string
}

func novaNFeXML() nfeXML {
	return nfeXML{
		cnpjEmitente: cnpjEmitente,
		destinoTag:   "CNPJ",
		destino:      cnpjDestinatario,
		finalidade:   "1",
		tipoOperacao: "1",
		ambiente:     "1",
		modelo:       "55",
		statusProt:   "100",
		emissao:      "2026-10-10T09:30:00-03:00",
		numero:       123,
		serie:        1,
		itens: []itemXML{
			{numero: "1", codigo: "PISO-01", cfop: "5102", quantidade: "10.5000", valor: "1050.00", desconto: "", pedido: ""},
			{numero: "2", codigo: "REJ-02", cfop: "5102", quantidade: "2", valor: "40.00", desconto: "1.50", pedido: "PC-77"},
		},
		duplicatas: []duplicataXML{
			{numero: "001", vencimento: "2026-11-09", valor: "544.25"},
			{numero: "002", vencimento: "2026-12-09", valor: "544.25"},
		},
	}
}

func chaveDe(cnpj, emissao string, serie, numero int, modelo string) string {
	anoMes := emissao[2:4] + emissao[5:7]
	base := fmt.Sprintf("25%s%s%s%03d%09d1%08d", anoMes, cnpj, modelo, serie, numero, 12345678)
	for digito := 0; digito <= 9; digito++ {
		candidata := base + strconv.Itoa(digito)
		if _, err := domainnotafiscal.NovaChave(candidata); err == nil {
			return candidata
		}
	}

	return base + "0"
}

func (n nfeXML) chave() string {
	return chaveDe(n.cnpjEmitente, n.emissao, n.serie, n.numero, n.modelo)
}

func (n nfeXML) bytes() []byte {
	chave := n.chave()
	emissao := n.emissao
	if n.dhEmi != "" {
		emissao = n.dhEmi
	}

	var itens strings.Builder
	for _, item := range n.itens {
		desconto := ""
		if item.desconto != "" {
			desconto = "<vDesc>" + item.desconto + "</vDesc>"
		}
		pedido := ""
		if item.pedido != "" {
			pedido = "<xPed>" + item.pedido + "</xPed>"
		}
		fmt.Fprintf(&itens, `<det nItem="%s"><prod><cProd>%s</cProd><cEAN>SEM GTIN</cEAN><xProd>Produto %s</xProd><NCM>69072100</NCM><CFOP>%s</CFOP><uCom>M2</uCom><qCom>%s</qCom><vUnCom>100.0000000000</vUnCom><vProd>%s</vProd>%s%s</prod></det>`,
			item.numero, item.codigo, item.codigo, item.cfop, item.quantidade, item.valor, desconto, pedido)
	}

	var duplicatas strings.Builder
	for _, dup := range n.duplicatas {
		fmt.Fprintf(&duplicatas, "<dup><nDup>%s</nDup><dVenc>%s</dVenc><vDup>%s</vDup></dup>", dup.numero, dup.vencimento, dup.valor)
	}
	cobranca := ""
	if duplicatas.Len() > 0 {
		cobranca = "<cobr><fat><nFat>123</nFat></fat>" + duplicatas.String() + "</cobr>"
	}

	compra := ""
	if n.pedidoCompra != "" {
		compra = "<compra><xPed>" + n.pedidoCompra + "</xPed></compra>"
	}

	nfe := fmt.Sprintf(`<NFe xmlns="http://www.portalfiscal.inf.br/nfe"><infNFe Id="NFe%s" versao="4.00"><ide><cUF>25</cUF><cNF>12345678</cNF><natOp>VENDA DE MERCADORIA</natOp><mod>%s</mod><serie>%d</serie><nNF>%d</nNF><dhEmi>%s</dhEmi><tpNF>%s</tpNF><idDest>1</idDest><cMunFG>2507507</cMunFG><tpImp>1</tpImp><tpEmis>1</tpEmis><cDV>%s</cDV><tpAmb>%s</tpAmb><finNFe>%s</finNFe></ide><emit><CNPJ>%s</CNPJ><xNome>MIA LTDA</xNome></emit><dest><%s>%s</%s><xNome>Construtora Exemplo SPE</xNome><enderDest><xLgr>Rua A</xLgr><xMun>Joao Pessoa</xMun><UF>PB</UF></enderDest><email>nf@exemplo.com.br</email></dest>%s<total><ICMSTot><vProd>1090.00</vProd><vFrete>0.00</vFrete><vDesc>1.50</vDesc><vNF>1088.50</vNF></ICMSTot></total>%s%s</infNFe><Signature xmlns="http://www.w3.org/2000/09/xmldsig#"></Signature></NFe>`,
		chave, n.modelo, n.serie, n.numero, emissao, n.tipoOperacao, chave[43:], n.ambiente, n.finalidade,
		n.cnpjEmitente, n.destinoTag, n.destino, n.destinoTag, itens.String(), cobranca, compra)

	if n.semProtocolo {
		return []byte(`<?xml version="1.0" encoding="UTF-8"?>` + nfe)
	}

	chaveProtocolo := chave
	if n.chaveProtocolo != "" {
		chaveProtocolo = n.chaveProtocolo
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><nfeProc xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00">%s<protNFe versao="4.00"><infProt><tpAmb>%s</tpAmb><verAplic>PB</verAplic><chNFe>%s</chNFe><dhRecbto>2026-10-10T09:31:02-03:00</dhRecbto><nProt>325260000012345</nProt><digVal>abc=</digVal><cStat>%s</cStat><xMotivo>Autorizado o uso da NF-e</xMotivo></infProt></protNFe></nfeProc>`,
		nfe, n.ambiente, chaveProtocolo, n.statusProt))
}

type eventoXML struct {
	chave      string
	autor      string
	tipo       string
	sequencia  string
	status     string
	texto      string
	tipoRetorn string
}

func novoEventoXML(chave string) eventoXML {
	return eventoXML{chave: chave, autor: cnpjEmitente, tipo: "110111", sequencia: "1", status: "135", texto: "Nota emitida com erro de valor"}
}

func (e eventoXML) bytes() []byte {
	detalhe := "<descEvento>Cancelamento</descEvento><nProt>325260000012345</nProt><xJust>" + e.texto + "</xJust>"
	if e.tipo == "110110" {
		detalhe = "<descEvento>Carta de Correcao</descEvento><xCorrecao>" + e.texto + "</xCorrecao><xCondUso>condicoes</xCondUso>"
	}
	tipoRetorno := e.tipo
	if e.tipoRetorn != "" {
		tipoRetorno = e.tipoRetorn
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><procEventoNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00"><evento versao="1.00"><infEvento Id="ID%s%s0%s"><cOrgao>25</cOrgao><tpAmb>1</tpAmb><CNPJ>%s</CNPJ><chNFe>%s</chNFe><dhEvento>2026-10-11T14:00:00-03:00</dhEvento><tpEvento>%s</tpEvento><nSeqEvento>%s</nSeqEvento><verEvento>1.00</verEvento><detEvento versao="1.00">%s</detEvento></infEvento></evento><retEvento versao="1.00"><infEvento><tpAmb>1</tpAmb><cStat>%s</cStat><chNFe>%s</chNFe><tpEvento>%s</tpEvento><nSeqEvento>%s</nSeqEvento><dhRegEvento>2026-10-11T14:00:05-03:00</dhRegEvento><nProt>325260000099999</nProt></infEvento></retEvento></procEventoNFe>`,
		e.tipo, e.chave, e.sequencia, e.autor, e.chave, e.tipo, e.sequencia, detalhe, e.status, e.chave, tipoRetorno, e.sequencia))
}
