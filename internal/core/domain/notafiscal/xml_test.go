package notafiscal_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
)

func TestLerNFeExtraiODocumento(t *testing.T) {
	xml := novaNFeXML()

	documento, err := domainnotafiscal.LerNFe(xml.bytes())
	if err != nil {
		t.Fatalf("LerNFe = %v", err)
	}

	if documento.Chave.Valor() != xml.chave() || documento.Numero != 123 || documento.Serie != 1 {
		t.Errorf("chave %q número %d série %d", documento.Chave.Valor(), documento.Numero, documento.Serie)
	}
	if !documento.EmitidaEm.Equal(time.Date(2026, 10, 10, 12, 30, 0, 0, time.UTC)) {
		t.Errorf("emitida em %v", documento.EmitidaEm)
	}
	if documento.TipoOperacao != domainnotafiscal.TipoOperacaoSaida || documento.Finalidade != domainnotafiscal.FinalidadeNormal {
		t.Errorf("tipo %q finalidade %q", documento.TipoOperacao, documento.Finalidade)
	}
	if documento.EmitenteCNPJ != cnpjEmitente || documento.NaturezaOperacao != "VENDA DE MERCADORIA" {
		t.Errorf("emitente %q natureza %q", documento.EmitenteCNPJ, documento.NaturezaOperacao)
	}

	destinatario := documento.Destinatario
	if destinatario.Documento != cnpjDestinatario || destinatario.Nome != "Construtora Exemplo SPE" ||
		destinatario.UF != "PB" || destinatario.Cidade != "Joao Pessoa" || destinatario.Email != "nf@exemplo.com.br" {
		t.Errorf("destinatário = %+v", destinatario)
	}

	if len(documento.Itens) != 2 {
		t.Fatalf("itens = %d, esperado 2", len(documento.Itens))
	}
	primeiro, segundo := documento.Itens[0], documento.Itens[1]
	if primeiro.Codigo != "PISO-01" || primeiro.QuantidadeDecimilesimos != 105000 || primeiro.ValorCentavos != 105000 ||
		primeiro.DescontoCentavos != 0 || primeiro.NCM != "69072100" || primeiro.Unidade != "M2" || primeiro.PedidoCliente != nil {
		t.Errorf("primeiro item = %+v", primeiro)
	}
	if segundo.DescontoCentavos != 150 || segundo.PedidoCliente == nil || *segundo.PedidoCliente != "PC-77" {
		t.Errorf("segundo item = %+v", segundo)
	}

	if documento.NumeroPedidoCliente == nil || *documento.NumeroPedidoCliente != "PC-77" {
		t.Errorf("pedido do cliente = %v, esperado o xPed do item", documento.NumeroPedidoCliente)
	}

	if len(documento.Duplicatas) != 2 || documento.Duplicatas[0].Numero != "001" || documento.Duplicatas[0].ValorCentavos != 54425 ||
		!documento.Duplicatas[1].Vencimento.Equal(time.Date(2026, 12, 9, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("duplicatas = %+v", documento.Duplicatas)
	}

	totais := documento.Totais
	if totais.TotalCentavos != 108850 || totais.ProdutosCentavos != 109000 || totais.DescontoCentavos != 150 || totais.FreteCentavos != 0 {
		t.Errorf("totais = %+v", totais)
	}

	if documento.Protocolo != "325260000012345" || documento.AutorizadaEm.IsZero() {
		t.Errorf("protocolo %q autorizada %v", documento.Protocolo, documento.AutorizadaEm)
	}
	if documento.CFOPPredominante() != "5102" {
		t.Errorf("CFOP predominante = %q", documento.CFOPPredominante())
	}
}

func TestLerNFePrefereOPedidoDaCompraEDestinatarioCPF(t *testing.T) {
	xml := novaNFeXML()
	xml.pedidoCompra = "OC-2026-15"
	xml.destinoTag = "CPF"
	xml.destino = cpfDestinatario
	xml.duplicatas = nil

	documento, err := domainnotafiscal.LerNFe(xml.bytes())
	if err != nil {
		t.Fatalf("LerNFe = %v", err)
	}
	if documento.NumeroPedidoCliente == nil || *documento.NumeroPedidoCliente != "OC-2026-15" {
		t.Errorf("pedido do cliente = %v", documento.NumeroPedidoCliente)
	}
	if documento.Destinatario.Documento != cpfDestinatario {
		t.Errorf("destinatário = %q", documento.Destinatario.Documento)
	}
	if len(documento.Duplicatas) != 0 {
		t.Errorf("duplicatas = %d, esperado nenhuma", len(documento.Duplicatas))
	}
}

func TestLerNFeAceitaEmitenteAlfanumerico(t *testing.T) {
	xml := novaNFeXML()
	xml.cnpjEmitente = cnpjAlfanumerico

	documento, err := domainnotafiscal.LerNFe(xml.bytes())
	if err != nil {
		t.Fatalf("LerNFe = %v", err)
	}
	if documento.EmitenteCNPJ != cnpjAlfanumerico {
		t.Errorf("emitente = %q", documento.EmitenteCNPJ)
	}
}

func TestLerNFeRecusaXMLInvalido(t *testing.T) {
	casos := map[string]func(*nfeXML){
		"sem protocolo":               func(x *nfeXML) { x.semProtocolo = true },
		"homologação":                 func(x *nfeXML) { x.ambiente = "2" },
		"não autorizada":              func(x *nfeXML) { x.statusProt = "302" },
		"nfc-e":                       func(x *nfeXML) { x.modelo = "65" },
		"protocolo de outra":          func(x *nfeXML) { x.chaveProtocolo = chaveDe(cnpjEmitente, "2026-10-10", 1, 999, "55") },
		"destinatário inválido":       func(x *nfeXML) { x.destino = "11444777000162" },
		"estrangeiro":                 func(x *nfeXML) { x.destinoTag = "idEstrangeiro"; x.destino = "AB123" },
		"sem itens":                   func(x *nfeXML) { x.itens = nil },
		"item repetido":               func(x *nfeXML) { x.itens[1].numero = "1" },
		"quantidade zero":             func(x *nfeXML) { x.itens[0].quantidade = "0" },
		"valor com três casas":        func(x *nfeXML) { x.itens[0].valor = "1.005" },
		"cfop inválido":               func(x *nfeXML) { x.itens[0].cfop = "51A2" },
		"duplicata repetida":          func(x *nfeXML) { x.duplicatas[1].numero = "001" },
		"duplicata zerada":            func(x *nfeXML) { x.duplicatas[0].valor = "0.00" },
		"vencimento inválido":         func(x *nfeXML) { x.duplicatas[0].vencimento = "09/11/2026" },
		"emissão sem fuso":            func(x *nfeXML) { x.dhEmi = "2026-10-10T09:30:00" },
		"pedido do item longo":        func(x *nfeXML) { x.itens[0].pedido = strings.Repeat("P", 61) },
		"emissão fora da chave":       func(x *nfeXML) { x.dhEmi = "2026-09-10T09:30:00-03:00" },
		"finalidade inexistente":      func(x *nfeXML) { x.finalidade = "9" },
		"entrada que não é devolução": func(x *nfeXML) { x.tipoOperacao = "0" },
		"sem protNFe":                 func(x *nfeXML) { x.semBlocoProtocolo = true },
	}

	for nome, ajuste := range casos {
		t.Run(nome, func(t *testing.T) {
			xml := novaNFeXML()
			ajuste(&xml)

			if _, err := domainnotafiscal.LerNFe(xml.bytes()); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}
}

func TestLerNFeRecusaArquivoQueNaoEhNFe(t *testing.T) {
	casos := map[string]string{
		"vazio":      "",
		"malformado": "<nfeProc><NFe>",
		"outro xml":  `<?xml version="1.0"?><pedido><id>1</id></pedido>`,
		"entidade":   `<?xml version="1.0"?><!DOCTYPE nfeProc [<!ENTITY x SYSTEM "file:///etc/passwd">]><nfeProc>&x;</nfeProc>`,
	}

	for nome, conteudo := range casos {
		if _, err := domainnotafiscal.LerNFe([]byte(conteudo)); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}
}

func TestLerEvento(t *testing.T) {
	chave := novaNFeXML().chave()

	cancelamento, err := domainnotafiscal.LerEvento(novoEventoXML(chave).bytes())
	if err != nil {
		t.Fatalf("LerEvento cancelamento = %v", err)
	}
	if cancelamento.Tipo != domainnotafiscal.TipoEventoCancelamento || cancelamento.Sequencia != 1 ||
		cancelamento.Protocolo != "325260000099999" || cancelamento.Justificativa != "Nota emitida com erro de valor" ||
		cancelamento.AutorCNPJ != cnpjEmitente || cancelamento.Chave.Valor() != chave {
		t.Errorf("cancelamento = %+v", cancelamento)
	}

	carta := novoEventoXML(chave)
	carta.tipo = "110110"
	carta.sequencia = "2"
	carta.texto = "Corrige o endereco de entrega"
	correcao, err := domainnotafiscal.LerEvento(carta.bytes())
	if err != nil {
		t.Fatalf("LerEvento carta = %v", err)
	}
	if correcao.Tipo != domainnotafiscal.TipoEventoCartaCorrecao || correcao.Sequencia != 2 || correcao.Justificativa != "Corrige o endereco de entrega" {
		t.Errorf("carta de correção = %+v", correcao)
	}
}

func TestLerEventoRecusaInvalido(t *testing.T) {
	chave := novaNFeXML().chave()
	casos := map[string]func(*eventoXML){
		"rejeitado":      func(e *eventoXML) { e.status = "573" },
		"tipo diferente": func(e *eventoXML) { e.tipoRetorn = "110110" },
		"manifestação":   func(e *eventoXML) { e.tipo = "210200" },
		"sequência zero": func(e *eventoXML) { e.sequencia = "0" },
		"sem texto":      func(e *eventoXML) { e.texto = " " },
		"chave inválida": func(e *eventoXML) { e.chave = chave[:43] + "X" },
	}

	for nome, ajuste := range casos {
		evento := novoEventoXML(chave)
		ajuste(&evento)
		if _, err := domainnotafiscal.LerEvento(evento.bytes()); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}

	if _, err := domainnotafiscal.LerEvento(novaNFeXML().bytes()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("NF-e como evento = %v, esperado validação", err)
	}
}

func TestLerNFeAceitaEntradaDeDevolucao(t *testing.T) {
	xml := novaNFeXML()
	xml.tipoOperacao = "0"
	xml.finalidade = "4"

	documento, err := domainnotafiscal.LerNFe(xml.bytes())
	if err != nil {
		t.Fatalf("LerNFe = %v", err)
	}
	if documento.TipoOperacao != domainnotafiscal.TipoOperacaoEntrada || documento.Finalidade != domainnotafiscal.FinalidadeDevolucao {
		t.Errorf("tipo %q finalidade %q", documento.TipoOperacao, documento.Finalidade)
	}
}

func TestLerNFeExplicaOMotivoDaRecusa(t *testing.T) {
	casos := map[string]struct {
		ajuste func(*nfeXML)
		trecho string
	}{
		"sem protNFe":                 {func(x *nfeXML) { x.semBlocoProtocolo = true }, "sem protocolo de autorização"},
		"entrada que não é devolução": {func(x *nfeXML) { x.tipoOperacao = "0" }, "só é aceita como devolução"},
		"homologação":                 {func(x *nfeXML) { x.ambiente = "2" }, "homologação"},
	}

	for nome, caso := range casos {
		xml := novaNFeXML()
		caso.ajuste(&xml)
		_, err := domainnotafiscal.LerNFe(xml.bytes())
		if err == nil || !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%s: erro = %v, esperado %q", nome, err, caso.trecho)
		}
	}
}
