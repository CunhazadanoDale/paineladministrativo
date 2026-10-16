package notafiscal

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
)

const (
	modeloNFe                  = "55"
	ambienteProducao           = "1"
	prefixoIDNFe               = "NFe"
	tamanhoProtocolo           = 15
	itensMaximos               = 990
	sequenciaMaximaEvento      = 20
	tamanhoMaximoNaturezaOp    = 60
	tamanhoMaximoNome          = 200
	tamanhoMaximoCodigo        = 60
	tamanhoMaximoDescricao     = 120
	tamanhoMaximoUnidade       = 6
	tamanhoMaximoPedido        = 60
	tamanhoMaximoNumeroDup     = 60
	tamanhoMaximoJustificativa = 1000
	tamanhoMaximoCidade        = 120
	tamanhoMaximoEmail         = 255
	formatoData                = "2006-01-02"
	formatoAnoMes              = "0601"
)

var (
	statusAutorizacao = map[string]bool{"100": true, "150": true}
	statusEvento      = map[string]bool{"135": true, "136": true, "155": true}
)

type nfeProcXML struct {
	NFe struct {
		InfNFe infNFeXML `xml:"infNFe"`
	} `xml:"NFe"`
	ProtNFe struct {
		InfProt infProtXML `xml:"infProt"`
	} `xml:"protNFe"`
}

type infNFeXML struct {
	ID  string `xml:"Id,attr"`
	Ide struct {
		CUF    string `xml:"cUF"`
		CNF    string `xml:"cNF"`
		NatOp  string `xml:"natOp"`
		Mod    string `xml:"mod"`
		Serie  string `xml:"serie"`
		NNF    string `xml:"nNF"`
		DhEmi  string `xml:"dhEmi"`
		TpNF   string `xml:"tpNF"`
		TpEmis string `xml:"tpEmis"`
		CDV    string `xml:"cDV"`
		TpAmb  string `xml:"tpAmb"`
		FinNFe string `xml:"finNFe"`
	} `xml:"ide"`
	Emit struct {
		CNPJ string `xml:"CNPJ"`
		CPF  string `xml:"CPF"`
	} `xml:"emit"`
	Dest struct {
		CNPJ          string `xml:"CNPJ"`
		CPF           string `xml:"CPF"`
		IDEstrangeiro string `xml:"idEstrangeiro"`
		XNome         string `xml:"xNome"`
		EnderDest     struct {
			XMun string `xml:"xMun"`
			UF   string `xml:"UF"`
		} `xml:"enderDest"`
		Email string `xml:"email"`
	} `xml:"dest"`
	Det []struct {
		NItem string `xml:"nItem,attr"`
		Prod  struct {
			CProd  string `xml:"cProd"`
			XProd  string `xml:"xProd"`
			NCM    string `xml:"NCM"`
			CFOP   string `xml:"CFOP"`
			UCom   string `xml:"uCom"`
			QCom   string `xml:"qCom"`
			VUnCom string `xml:"vUnCom"`
			VProd  string `xml:"vProd"`
			VDesc  string `xml:"vDesc"`
			XPed   string `xml:"xPed"`
		} `xml:"prod"`
	} `xml:"det"`
	Total struct {
		ICMSTot struct {
			VProd  string `xml:"vProd"`
			VFrete string `xml:"vFrete"`
			VDesc  string `xml:"vDesc"`
			VNF    string `xml:"vNF"`
		} `xml:"ICMSTot"`
	} `xml:"total"`
	Cobr struct {
		Dup []struct {
			NDup  string `xml:"nDup"`
			DVenc string `xml:"dVenc"`
			VDup  string `xml:"vDup"`
		} `xml:"dup"`
	} `xml:"cobr"`
	Compra struct {
		XPed string `xml:"xPed"`
	} `xml:"compra"`
}

type infProtXML struct {
	TpAmb    string `xml:"tpAmb"`
	ChNFe    string `xml:"chNFe"`
	DhRecbto string `xml:"dhRecbto"`
	NProt    string `xml:"nProt"`
	CStat    string `xml:"cStat"`
}

type procEventoXML struct {
	Evento struct {
		InfEvento struct {
			TpAmb      string `xml:"tpAmb"`
			CNPJ       string `xml:"CNPJ"`
			ChNFe      string `xml:"chNFe"`
			DhEvento   string `xml:"dhEvento"`
			TpEvento   string `xml:"tpEvento"`
			NSeqEvento string `xml:"nSeqEvento"`
			DetEvento  struct {
				XJust     string `xml:"xJust"`
				XCorrecao string `xml:"xCorrecao"`
			} `xml:"detEvento"`
		} `xml:"infEvento"`
	} `xml:"evento"`
	RetEvento struct {
		InfEvento struct {
			TpAmb      string `xml:"tpAmb"`
			CStat      string `xml:"cStat"`
			ChNFe      string `xml:"chNFe"`
			TpEvento   string `xml:"tpEvento"`
			NSeqEvento string `xml:"nSeqEvento"`
			NProt      string `xml:"nProt"`
		} `xml:"infEvento"`
	} `xml:"retEvento"`
}

func LerNFe(conteudo []byte) (*Documento, error) {
	raiz, err := elementoRaiz(conteudo)
	if err != nil {
		return nil, err
	}
	switch raiz {
	case "nfeProc":
	case "NFe":
		return nil, domain.ErroValidacao("XML sem protocolo de autorização: envie o nfeProc autorizado")
	default:
		return nil, domain.ErroValidacao("XML não é uma NF-e autorizada (nfeProc)")
	}

	var proc nfeProcXML
	if err := xml.Unmarshal(conteudo, &proc); err != nil {
		return nil, domain.ErroValidacao("XML da NF-e malformado")
	}

	infNFe := proc.NFe.InfNFe
	identificador := strings.TrimSpace(infNFe.ID)
	if !strings.HasPrefix(identificador, prefixoIDNFe) {
		return nil, domain.ErroValidacao(mensagemChaveInvalida)
	}
	chave, err := NovaChave(strings.TrimPrefix(identificador, prefixoIDNFe))
	if err != nil {
		return nil, err
	}

	if err := conferirIdentificacao(infNFe, chave); err != nil {
		return nil, err
	}

	documento := &Documento{Chave: chave, Serie: chave.Serie(), Numero: chave.Numero()}

	if documento.EmitidaEm, err = momento(infNFe.Ide.DhEmi, "data de emissão (dhEmi)"); err != nil {
		return nil, err
	}
	if documento.EmitidaEm.Format(formatoAnoMes) != chave.AnoMes() {
		return nil, domain.ErroValidacao("ano e mês da chave não batem com a data de emissão")
	}
	if documento.TipoOperacao, err = TipoOperacaoDoCodigo(infNFe.Ide.TpNF); err != nil {
		return nil, err
	}
	if documento.Finalidade, err = FinalidadeDoCodigo(infNFe.Ide.FinNFe); err != nil {
		return nil, err
	}
	if documento.NaturezaOperacao, err = textoObrigatorio(infNFe.Ide.NatOp, "natureza da operação (natOp)", tamanhoMaximoNaturezaOp); err != nil {
		return nil, err
	}
	if documento.EmitenteCNPJ, err = emitente(infNFe, chave); err != nil {
		return nil, err
	}
	if documento.Destinatario, err = destinatario(infNFe); err != nil {
		return nil, err
	}
	if documento.Itens, err = itens(infNFe); err != nil {
		return nil, err
	}
	if documento.Duplicatas, err = duplicatas(infNFe); err != nil {
		return nil, err
	}
	if documento.Totais, err = totais(infNFe); err != nil {
		return nil, err
	}
	if documento.NumeroPedidoCliente, err = numeroPedidoCliente(infNFe.Compra.XPed, documento.Itens); err != nil {
		return nil, err
	}
	if documento.Protocolo, documento.AutorizadaEm, err = autorizacao(proc.ProtNFe.InfProt, chave); err != nil {
		return nil, err
	}

	return documento, nil
}

func LerEvento(conteudo []byte) (*DocumentoEvento, error) {
	raiz, err := elementoRaiz(conteudo)
	if err != nil {
		return nil, err
	}
	if raiz != "procEventoNFe" {
		return nil, domain.ErroValidacao("XML não é um evento de NF-e autorizado (procEventoNFe)")
	}

	var proc procEventoXML
	if err := xml.Unmarshal(conteudo, &proc); err != nil {
		return nil, domain.ErroValidacao("XML do evento malformado")
	}

	pedido := proc.Evento.InfEvento
	retorno := proc.RetEvento.InfEvento

	if strings.TrimSpace(pedido.TpAmb) != ambienteProducao || strings.TrimSpace(retorno.TpAmb) != ambienteProducao {
		return nil, domain.ErroValidacao("evento de homologação não é aceito")
	}
	if !statusEvento[strings.TrimSpace(retorno.CStat)] {
		return nil, domain.ErroValidacao("evento não registrado pela SEFAZ")
	}

	chave, err := NovaChave(pedido.ChNFe)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(retorno.ChNFe) != chave.Valor() {
		return nil, domain.ErroValidacao("chave do retorno do evento diferente da chave do evento")
	}

	tipo, err := TipoEventoDoCodigo(pedido.TpEvento)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(retorno.TpEvento) != strings.TrimSpace(pedido.TpEvento) {
		return nil, domain.ErroValidacao("tipo do retorno do evento diferente do tipo do evento")
	}

	sequencia, err := strconv.Atoi(strings.TrimSpace(pedido.NSeqEvento))
	if err != nil || sequencia < 1 || sequencia > sequenciaMaximaEvento {
		return nil, domain.ErroValidacao("sequência do evento (nSeqEvento) inválida")
	}
	if sequenciaRetorno, err := strconv.Atoi(strings.TrimSpace(retorno.NSeqEvento)); err != nil || sequenciaRetorno != sequencia {
		return nil, domain.ErroValidacao("sequência do retorno do evento diferente da sequência do evento")
	}

	ocorridoEm, err := momento(pedido.DhEvento, "data do evento (dhEvento)")
	if err != nil {
		return nil, err
	}

	protocolo, err := protocoloValido(retorno.NProt, "protocolo do evento")
	if err != nil {
		return nil, err
	}

	textoJustificativa := pedido.DetEvento.XJust
	if tipo == TipoEventoCartaCorrecao {
		textoJustificativa = pedido.DetEvento.XCorrecao
	}
	justificativa, err := textoObrigatorio(textoJustificativa, "justificativa ou correção do evento", tamanhoMaximoJustificativa)
	if err != nil {
		return nil, err
	}

	return &DocumentoEvento{
		Chave:         chave,
		AutorCNPJ:     strings.ToUpper(strings.TrimSpace(pedido.CNPJ)),
		Tipo:          tipo,
		Sequencia:     sequencia,
		OcorridoEm:    ocorridoEm,
		Protocolo:     protocolo,
		Justificativa: justificativa,
	}, nil
}

func elementoRaiz(conteudo []byte) (string, error) {
	decodificador := xml.NewDecoder(bytes.NewReader(conteudo))
	for {
		token, err := decodificador.Token()
		if errors.Is(err, io.EOF) {
			return "", domain.ErroValidacao("XML vazio")
		}
		if err != nil {
			return "", domain.ErroValidacao("XML malformado")
		}
		if inicio, ok := token.(xml.StartElement); ok {
			return inicio.Name.Local, nil
		}
	}
}

func conferirIdentificacao(infNFe infNFeXML, chave Chave) error {
	ide := infNFe.Ide

	switch {
	case strings.TrimSpace(ide.Mod) != modeloNFe || chave.Modelo() != modeloNFe:
		return domain.ErroValidacao("só NF-e modelo 55 é aceita")
	case strings.TrimSpace(ide.TpAmb) != ambienteProducao:
		return domain.ErroValidacao("nota emitida em homologação não é aceita")
	case strings.TrimSpace(ide.CUF) != chave.CodigoUF():
		return domain.ErroValidacao("UF da chave diferente da UF da nota")
	case numeroDe(ide.Serie) != chave.Serie():
		return domain.ErroValidacao("série da chave diferente da série da nota")
	case numeroDe(ide.NNF) != chave.Numero() || chave.Numero() == 0:
		return domain.ErroValidacao("número da chave diferente do número da nota")
	case strings.TrimSpace(ide.TpEmis) != chave.TipoEmissao():
		return domain.ErroValidacao("tipo de emissão da chave diferente do da nota")
	case fmt.Sprintf("%08d", numeroDe(ide.CNF)) != chave.CodigoNumerico():
		return domain.ErroValidacao("código numérico da chave diferente do da nota")
	case strings.TrimSpace(ide.CDV) != chave.DigitoVerificador():
		return domain.ErroValidacao("dígito verificador da chave diferente do da nota")
	}

	return nil
}

func emitente(infNFe infNFeXML, chave Chave) (string, error) {
	cnpj := strings.ToUpper(strings.TrimSpace(infNFe.Emit.CNPJ))
	if cnpj == "" {
		return "", domain.ErroValidacao("emitente da nota precisa ter CNPJ")
	}
	if cnpj != chave.CNPJEmitente() {
		return "", domain.ErroValidacao("CNPJ do emitente diferente do CNPJ da chave")
	}

	return cnpj, nil
}

func destinatario(infNFe infNFeXML) (Destinatario, error) {
	dest := infNFe.Dest
	if strings.TrimSpace(dest.IDEstrangeiro) != "" {
		return Destinatario{}, domain.ErroValidacao("destinatário estrangeiro não é aceito: o cliente precisa de CPF ou CNPJ")
	}

	valor := dest.CNPJ
	if strings.TrimSpace(valor) == "" {
		valor = dest.CPF
	}
	if strings.TrimSpace(valor) == "" {
		return Destinatario{}, domain.ErroValidacao("destinatário da nota sem CPF ou CNPJ")
	}

	documento, err := domaincliente.NovoDocumento(valor)
	if err != nil {
		return Destinatario{}, domain.ErroValidacao("documento do destinatário inválido")
	}

	nome, err := textoObrigatorio(dest.XNome, "nome do destinatário", tamanhoMaximoNome)
	if err != nil {
		return Destinatario{}, err
	}

	return Destinatario{
		Documento: documento.Valor(),
		Nome:      nome,
		UF:        strings.ToUpper(strings.TrimSpace(dest.EnderDest.UF)),
		Cidade:    limitado(dest.EnderDest.XMun, tamanhoMaximoCidade),
		Email:     limitado(dest.Email, tamanhoMaximoEmail),
	}, nil
}

func itens(infNFe infNFeXML) ([]ItemDocumento, error) {
	if len(infNFe.Det) == 0 {
		return nil, domain.ErroValidacao("nota sem itens")
	}
	if len(infNFe.Det) > itensMaximos {
		return nil, domain.ErroValidacao("nota com itens demais")
	}

	lidos := make([]ItemDocumento, 0, len(infNFe.Det))
	numeros := make(map[int]bool, len(infNFe.Det))
	for _, det := range infNFe.Det {
		numero := numeroDe(det.NItem)
		if numero < 1 || numero > itensMaximos || numeros[numero] {
			return nil, domain.ErroValidacao("número do item (nItem) inválido ou repetido")
		}
		numeros[numero] = true

		item, err := lerItem(numero, det.Prod.CProd, det.Prod.XProd, det.Prod.NCM, det.Prod.CFOP, det.Prod.UCom,
			det.Prod.QCom, det.Prod.VUnCom, det.Prod.VProd, det.Prod.VDesc, det.Prod.XPed)
		if err != nil {
			return nil, err
		}
		lidos = append(lidos, item)
	}

	return lidos, nil
}

func lerItem(numero int, codigo, descricao, ncm, cfop, unidade, quantidade, valorUnitario, valor, desconto, pedido string) (ItemDocumento, error) {
	rotulo := func(campo string) string {
		return fmt.Sprintf("%s do item %d", campo, numero)
	}

	lido := ItemDocumento{NumeroItem: numero}
	var err error

	if lido.Codigo, err = textoObrigatorio(codigo, rotulo("código (cProd)"), tamanhoMaximoCodigo); err != nil {
		return ItemDocumento{}, err
	}
	if lido.Descricao, err = textoObrigatorio(descricao, rotulo("descrição (xProd)"), tamanhoMaximoDescricao); err != nil {
		return ItemDocumento{}, err
	}
	lido.NCM = strings.TrimSpace(ncm)
	if !somenteDigitos(lido.NCM) || (len(lido.NCM) != 2 && len(lido.NCM) != 8) {
		return ItemDocumento{}, domain.ErroValidacao(rotulo("NCM") + " inválido")
	}
	lido.CFOP = strings.TrimSpace(cfop)
	if !somenteDigitos(lido.CFOP) || len(lido.CFOP) != 4 {
		return ItemDocumento{}, domain.ErroValidacao(rotulo("CFOP") + " inválido")
	}
	if lido.Unidade, err = textoObrigatorio(unidade, rotulo("unidade (uCom)"), tamanhoMaximoUnidade); err != nil {
		return ItemDocumento{}, err
	}
	if lido.QuantidadeDecimilesimos, err = QuantidadeDe(quantidade, rotulo("quantidade (qCom)")); err != nil {
		return ItemDocumento{}, err
	}
	if lido.ValorUnitario, err = ValorUnitarioDe(valorUnitario, rotulo("valor unitário (vUnCom)")); err != nil {
		return ItemDocumento{}, err
	}
	if lido.ValorCentavos, err = CentavosDe(valor, rotulo("valor (vProd)")); err != nil {
		return ItemDocumento{}, err
	}
	if lido.DescontoCentavos, err = CentavosOpcionaisDe(desconto, rotulo("desconto (vDesc)")); err != nil {
		return ItemDocumento{}, err
	}
	if lido.PedidoCliente, err = textoOpcional(pedido, rotulo("pedido do cliente (xPed)"), tamanhoMaximoPedido); err != nil {
		return ItemDocumento{}, err
	}

	return lido, nil
}

func duplicatas(infNFe infNFeXML) ([]DuplicataDocumento, error) {
	lidas := make([]DuplicataDocumento, 0, len(infNFe.Cobr.Dup))
	numeros := make(map[string]bool, len(infNFe.Cobr.Dup))
	for _, dup := range infNFe.Cobr.Dup {
		numero, err := textoObrigatorio(dup.NDup, "número da duplicata (nDup)", tamanhoMaximoNumeroDup)
		if err != nil {
			return nil, err
		}
		if numeros[numero] {
			return nil, domain.ErroValidacao("número de duplicata repetido na nota")
		}
		numeros[numero] = true

		vencimento, err := time.Parse(formatoData, strings.TrimSpace(dup.DVenc))
		if err != nil {
			return nil, domain.ErroValidacao("vencimento da duplicata " + numero + " inválido")
		}

		valor, err := CentavosDe(dup.VDup, "valor da duplicata "+numero)
		if err != nil {
			return nil, err
		}
		if valor == 0 {
			return nil, domain.ErroValidacao("valor da duplicata " + numero + " deve ser maior que zero")
		}

		lidas = append(lidas, DuplicataDocumento{Numero: numero, Vencimento: vencimento, ValorCentavos: valor})
	}

	return lidas, nil
}

func totais(infNFe infNFeXML) (Totais, error) {
	icms := infNFe.Total.ICMSTot
	var lidos Totais
	var err error

	if lidos.TotalCentavos, err = CentavosDe(icms.VNF, "valor total da nota (vNF)"); err != nil {
		return Totais{}, err
	}
	if lidos.ProdutosCentavos, err = CentavosDe(icms.VProd, "valor dos produtos (vProd)"); err != nil {
		return Totais{}, err
	}
	if lidos.FreteCentavos, err = CentavosOpcionaisDe(icms.VFrete, "valor do frete (vFrete)"); err != nil {
		return Totais{}, err
	}
	if lidos.DescontoCentavos, err = CentavosOpcionaisDe(icms.VDesc, "valor do desconto (vDesc)"); err != nil {
		return Totais{}, err
	}

	return lidos, nil
}

func numeroPedidoCliente(compra string, itens []ItemDocumento) (*string, error) {
	pedido, err := textoOpcional(compra, "pedido do cliente (xPed)", tamanhoMaximoPedido)
	if err != nil || pedido != nil {
		return pedido, err
	}

	for _, item := range itens {
		if item.PedidoCliente != nil {
			copia := *item.PedidoCliente
			return &copia, nil
		}
	}

	return nil, nil
}

func autorizacao(prot infProtXML, chave Chave) (string, time.Time, error) {
	if strings.TrimSpace(prot.TpAmb) != ambienteProducao {
		return "", time.Time{}, domain.ErroValidacao("nota emitida em homologação não é aceita")
	}
	if !statusAutorizacao[strings.TrimSpace(prot.CStat)] {
		return "", time.Time{}, domain.ErroValidacao("nota não autorizada pela SEFAZ")
	}
	if strings.ToUpper(strings.TrimSpace(prot.ChNFe)) != chave.Valor() {
		return "", time.Time{}, domain.ErroValidacao("chave do protocolo diferente da chave da nota")
	}

	protocolo, err := protocoloValido(prot.NProt, "protocolo de autorização")
	if err != nil {
		return "", time.Time{}, err
	}

	autorizadaEm, err := momento(prot.DhRecbto, "data da autorização (dhRecbto)")
	if err != nil {
		return "", time.Time{}, err
	}

	return protocolo, autorizadaEm, nil
}

func protocoloValido(valor, campo string) (string, error) {
	protocolo := strings.TrimSpace(valor)
	if len(protocolo) != tamanhoProtocolo || !somenteDigitos(protocolo) {
		return "", domain.ErroValidacao(campo + " inválido")
	}

	return protocolo, nil
}

func momento(valor, campo string) (time.Time, error) {
	instante, err := time.Parse(time.RFC3339, strings.TrimSpace(valor))
	if err != nil {
		return time.Time{}, domain.ErroValidacao(campo + " inválida")
	}

	return instante, nil
}

func numeroDe(valor string) int {
	numero, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil {
		return -1
	}

	return numero
}

func textoObrigatorio(valor, campo string, tamanhoMaximo int) (string, error) {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return "", domain.ErroValidacao(campo + " é obrigatório")
	}
	if utf8.RuneCountInString(limpo) > tamanhoMaximo {
		return "", domain.ErroValidacao(fmt.Sprintf("%s deve ter no máximo %d caracteres", campo, tamanhoMaximo))
	}

	return limpo, nil
}

func textoOpcional(valor, campo string, tamanhoMaximo int) (*string, error) {
	if strings.TrimSpace(valor) == "" {
		return nil, nil
	}

	limpo, err := textoObrigatorio(valor, campo, tamanhoMaximo)
	if err != nil {
		return nil, err
	}

	return &limpo, nil
}

func limitado(valor string, tamanhoMaximo int) string {
	limpo := strings.TrimSpace(valor)
	if utf8.RuneCountInString(limpo) > tamanhoMaximo {
		return ""
	}

	return limpo
}
