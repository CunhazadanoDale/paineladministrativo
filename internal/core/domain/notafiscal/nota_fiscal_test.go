package notafiscal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

var agoraDoTeste = time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)

func documentoDoTeste(t *testing.T, ajustes ...func(*nfeXML)) *domainnotafiscal.Documento {
	t.Helper()

	xml := novaNFeXML()
	for _, ajuste := range ajustes {
		ajuste(&xml)
	}

	documento, err := domainnotafiscal.LerNFe(xml.bytes())
	if err != nil {
		t.Fatalf("LerNFe = %v", err)
	}

	return documento
}

func importacaoDoTeste(documento *domainnotafiscal.Documento) domainnotafiscal.ImportacaoInput {
	clienteID := uuid.New()

	return domainnotafiscal.ImportacaoInput{
		Documento:         documento,
		EmpresaID:         uuid.New(),
		ClienteID:         &clienteID,
		ProdutosPorCodigo: map[string]uuid.UUID{"PISO-01": uuid.New()},
		ArquivoID:         uuid.New(),
		UsuarioID:         uuid.New(),
		Agora:             agoraDoTeste,
	}
}

func notaDoTeste(t *testing.T) *domainnotafiscal.NotaFiscal {
	t.Helper()

	nota, err := domainnotafiscal.NovaNotaFiscal(importacaoDoTeste(documentoDoTeste(t)))
	if err != nil {
		t.Fatalf("NovaNotaFiscal = %v", err)
	}

	return nota
}

func TestNovaNotaFiscalMontaANotaDoDocumento(t *testing.T) {
	documento := documentoDoTeste(t)
	input := importacaoDoTeste(documento)

	nota, err := domainnotafiscal.NovaNotaFiscal(input)
	if err != nil {
		t.Fatalf("NovaNotaFiscal = %v", err)
	}

	if nota.ID == uuid.Nil || nota.Chave.Valor() != documento.Chave.Valor() || nota.EmpresaID != input.EmpresaID {
		t.Errorf("identificação = %v %q %v", nota.ID, nota.Chave.Valor(), nota.EmpresaID)
	}
	if nota.Natureza != domainnotafiscal.NaturezaVenda || nota.EntreEmpresas || nota.CFOPPredominante != "5102" {
		t.Errorf("natureza %q entre empresas %v CFOP %q", nota.Natureza, nota.EntreEmpresas, nota.CFOPPredominante)
	}
	if nota.ClienteID == nil || *nota.ClienteID != *input.ClienteID || nota.EmpresaDestinatariaID != nil {
		t.Errorf("destino = cliente %v empresa %v", nota.ClienteID, nota.EmpresaDestinatariaID)
	}
	if nota.Status != domainnotafiscal.StatusAutorizada || nota.Versao != 1 || !nota.CriadoEm.Equal(agoraDoTeste) {
		t.Errorf("status %q versão %d criado %v", nota.Status, nota.Versao, nota.CriadoEm)
	}
	if nota.ValorTotalCentavos != 108850 || nota.DestinatarioDocumento != cnpjDestinatario || nota.Protocolo != "325260000012345" {
		t.Errorf("valores = total %d destinatário %q protocolo %q", nota.ValorTotalCentavos, nota.DestinatarioDocumento, nota.Protocolo)
	}

	if len(nota.Itens) != 2 || nota.Itens[0].NotaFiscalID != nota.ID {
		t.Fatalf("itens = %+v", nota.Itens)
	}
	if nota.Itens[0].ProdutoID == nil || *nota.Itens[0].ProdutoID != input.ProdutosPorCodigo["PISO-01"] {
		t.Errorf("produto do primeiro item = %v", nota.Itens[0].ProdutoID)
	}
	pendentes := nota.ItensSemProduto()
	if len(pendentes) != 1 || pendentes[0].Codigo != "REJ-02" {
		t.Errorf("itens sem produto = %+v", pendentes)
	}

	if len(nota.Duplicatas) != 2 || nota.Duplicatas[0].ID == uuid.Nil || nota.Duplicatas[0].NotaFiscalID != nota.ID {
		t.Errorf("duplicatas = %+v", nota.Duplicatas)
	}
}

func TestNovaNotaFiscalEntreEmpresasEhTransferencia(t *testing.T) {
	input := importacaoDoTeste(documentoDoTeste(t))
	empresaDestinataria := uuid.New()
	input.ClienteID = nil
	input.EmpresaDestinatariaID = &empresaDestinataria

	nota, err := domainnotafiscal.NovaNotaFiscal(input)
	if err != nil {
		t.Fatalf("NovaNotaFiscal = %v", err)
	}
	if !nota.EntreEmpresas || nota.Natureza != domainnotafiscal.NaturezaTransferencia || nota.ClienteID != nil {
		t.Errorf("entre empresas %v natureza %q cliente %v", nota.EntreEmpresas, nota.Natureza, nota.ClienteID)
	}
	if len(nota.Duplicatas) != 2 {
		t.Errorf("duplicatas = %d, esperado guardar as duas", len(nota.Duplicatas))
	}
}

func TestNovaNotaFiscalClassificaRemessaEDevolucao(t *testing.T) {
	remessa := importacaoDoTeste(documentoDoTeste(t, func(x *nfeXML) {
		x.itens[0].cfop = "6116"
		x.itens[1].cfop = "6116"
	}))
	nota, err := domainnotafiscal.NovaNotaFiscal(remessa)
	if err != nil || nota.Natureza != domainnotafiscal.NaturezaRemessa {
		t.Errorf("remessa = %v, %v", nota, err)
	}

	devolucao := importacaoDoTeste(documentoDoTeste(t, func(x *nfeXML) { x.finalidade = "4" }))
	nota, err = domainnotafiscal.NovaNotaFiscal(devolucao)
	if err != nil || nota.Natureza != domainnotafiscal.NaturezaDevolucao {
		t.Errorf("devolução = %v, %v", nota, err)
	}
}

func TestNovaNotaFiscalValidaEntrada(t *testing.T) {
	documento := documentoDoTeste(t)
	outroID := uuid.New()

	casos := map[string]func(*domainnotafiscal.ImportacaoInput){
		"sem documento": func(i *domainnotafiscal.ImportacaoInput) { i.Documento = nil },
		"sem empresa":   func(i *domainnotafiscal.ImportacaoInput) { i.EmpresaID = uuid.Nil },
		"sem arquivo":   func(i *domainnotafiscal.ImportacaoInput) { i.ArquivoID = uuid.Nil },
		"sem usuário":   func(i *domainnotafiscal.ImportacaoInput) { i.UsuarioID = uuid.Nil },
		"sem destino":   func(i *domainnotafiscal.ImportacaoInput) { i.ClienteID = nil },
		"dois destinos": func(i *domainnotafiscal.ImportacaoInput) { i.EmpresaDestinatariaID = &outroID },
		"para si mesma": func(i *domainnotafiscal.ImportacaoInput) {
			empresa := i.EmpresaID
			i.ClienteID = nil
			i.EmpresaDestinatariaID = &empresa
		},
	}

	for nome, ajuste := range casos {
		input := importacaoDoTeste(documento)
		ajuste(&input)
		if _, err := domainnotafiscal.NovaNotaFiscal(input); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}
}

func pedidoDaNota(nota *domainnotafiscal.NotaFiscal, status domainpedido.Status) *domainpedido.Pedido {
	return &domainpedido.Pedido{ID: uuid.New(), EmpresaID: nota.EmpresaID, ClienteID: *nota.ClienteID, Status: status}
}

func TestVincularPedidoEEntrega(t *testing.T) {
	nota := notaDoTeste(t)
	pedido := pedidoDaNota(nota, domainpedido.StatusParcialmenteEntregue)
	entrega := &domainpedido.Entrega{ID: uuid.New(), PedidoID: pedido.ID, Status: domainpedido.StatusEntregaSaiu}

	if err := nota.Vincular(pedido, entrega, agoraDoTeste); err != nil {
		t.Fatalf("Vincular = %v", err)
	}
	if nota.PedidoID == nil || *nota.PedidoID != pedido.ID || nota.EntregaID == nil || *nota.EntregaID != entrega.ID {
		t.Errorf("vínculo = pedido %v entrega %v", nota.PedidoID, nota.EntregaID)
	}

	if err := nota.Vincular(pedido, nil, agoraDoTeste); err != nil || nota.EntregaID != nil {
		t.Errorf("vincular só o pedido = %v, entrega %v", err, nota.EntregaID)
	}

	if err := nota.Vincular(nil, nil, agoraDoTeste); err != nil || nota.PedidoID != nil {
		t.Errorf("desfazer = %v, pedido %v", err, nota.PedidoID)
	}
}

func TestVincularRecusaPedidoIncompativel(t *testing.T) {
	nota := notaDoTeste(t)

	casos := map[string]func() (*domainpedido.Pedido, *domainpedido.Entrega){
		"outra empresa": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			pedido := pedidoDaNota(nota, domainpedido.StatusConfirmado)
			pedido.EmpresaID = uuid.New()
			return pedido, nil
		},
		"outro cliente": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			pedido := pedidoDaNota(nota, domainpedido.StatusConfirmado)
			pedido.ClienteID = uuid.New()
			return pedido, nil
		},
		"proposta": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			return pedidoDaNota(nota, domainpedido.StatusProposta), nil
		},
		"cancelado": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			return pedidoDaNota(nota, domainpedido.StatusCancelado), nil
		},
		"entrega de outro pedido": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			return pedidoDaNota(nota, domainpedido.StatusConfirmado), &domainpedido.Entrega{ID: uuid.New(), PedidoID: uuid.New()}
		},
		"entrega cancelada": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			pedido := pedidoDaNota(nota, domainpedido.StatusConfirmado)
			return pedido, &domainpedido.Entrega{ID: uuid.New(), PedidoID: pedido.ID, Status: domainpedido.StatusEntregaCancelada}
		},
		"entrega sem pedido": func() (*domainpedido.Pedido, *domainpedido.Entrega) {
			return nil, &domainpedido.Entrega{ID: uuid.New()}
		},
	}

	for nome, montar := range casos {
		pedido, entrega := montar()
		if err := nota.Vincular(pedido, entrega, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}
	if nota.PedidoID != nil {
		t.Errorf("vínculo recusado alterou a nota: %v", nota.PedidoID)
	}
}

func TestVincularRecusaNotaEntreEmpresasECancelada(t *testing.T) {
	input := importacaoDoTeste(documentoDoTeste(t))
	empresaDestinataria := uuid.New()
	clienteID := *input.ClienteID
	input.ClienteID = nil
	input.EmpresaDestinatariaID = &empresaDestinataria
	transferencia, err := domainnotafiscal.NovaNotaFiscal(input)
	if err != nil {
		t.Fatalf("NovaNotaFiscal = %v", err)
	}
	pedido := &domainpedido.Pedido{ID: uuid.New(), EmpresaID: transferencia.EmpresaID, ClienteID: clienteID, Status: domainpedido.StatusConfirmado}
	if err := transferencia.Vincular(pedido, nil, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("transferência com pedido = %v, esperado validação", err)
	}

	cancelada := notaDoTeste(t)
	cancelada.Status = domainnotafiscal.StatusCancelada
	if err := cancelada.Vincular(nil, nil, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("nota cancelada = %v, esperado conflito", err)
	}
}

func TestVincularProduto(t *testing.T) {
	nota := notaDoTeste(t)
	produtoID := uuid.New()

	item, err := nota.VincularProduto(2, produtoID, agoraDoTeste)
	if err != nil || item.ProdutoID == nil || *item.ProdutoID != produtoID {
		t.Fatalf("VincularProduto = %v, %v", item, err)
	}
	if len(nota.ItensSemProduto()) != 0 {
		t.Errorf("itens sem produto = %d", len(nota.ItensSemProduto()))
	}

	if _, err := nota.VincularProduto(9, produtoID, agoraDoTeste); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("item inexistente = %v, esperado não encontrado", err)
	}
	if _, err := nota.VincularProduto(1, uuid.Nil, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("produto vazio = %v, esperado validação", err)
	}
}

func TestConferirVersaoEFaltaNumeroPedido(t *testing.T) {
	nota := notaDoTeste(t)

	if err := nota.ConferirVersao(1); err != nil {
		t.Errorf("versão atual = %v", err)
	}
	if err := nota.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão ausente = %v", err)
	}
	if err := nota.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v", err)
	}

	if nota.FaltaNumeroPedido(true) {
		t.Error("nota com xPed não falta número de pedido")
	}
	nota.NumeroPedidoCliente = nil
	if !nota.FaltaNumeroPedido(true) || nota.FaltaNumeroPedido(false) {
		t.Error("falta número de pedido só quando o cliente exige")
	}
}

func eventoDoTeste(t *testing.T, chave string, ajustes ...func(*eventoXML)) *domainnotafiscal.DocumentoEvento {
	t.Helper()

	xml := novoEventoXML(chave)
	for _, ajuste := range ajustes {
		ajuste(&xml)
	}

	documento, err := domainnotafiscal.LerEvento(xml.bytes())
	if err != nil {
		t.Fatalf("LerEvento = %v", err)
	}

	return documento
}

func TestAplicarEventoCancelaANota(t *testing.T) {
	nota := notaDoTeste(t)
	arquivoID, usuarioID := uuid.New(), uuid.New()

	evento, err := nota.AplicarEvento(eventoDoTeste(t, nota.Chave.Valor()), arquivoID, usuarioID, agoraDoTeste)
	if err != nil {
		t.Fatalf("AplicarEvento = %v", err)
	}
	if nota.Status != domainnotafiscal.StatusCancelada {
		t.Errorf("status = %q, esperado cancelada", nota.Status)
	}
	if evento.NotaFiscalID != nota.ID || evento.Tipo != domainnotafiscal.TipoEventoCancelamento || evento.ArquivoID != arquivoID ||
		evento.UsuarioID != usuarioID || evento.Protocolo != "325260000099999" {
		t.Errorf("evento = %+v", evento)
	}

	if _, err := nota.AplicarEvento(eventoDoTeste(t, nota.Chave.Valor()), uuid.New(), usuarioID, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar de novo = %v, esperado conflito", err)
	}
	carta := eventoDoTeste(t, nota.Chave.Valor(), func(e *eventoXML) { e.tipo = "110110" })
	if _, err := nota.AplicarEvento(carta, uuid.New(), usuarioID, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("carta em nota cancelada = %v, esperado conflito", err)
	}
}

func TestAplicarCartaDeCorrecaoNaoMudaANota(t *testing.T) {
	nota := notaDoTeste(t)
	carta := eventoDoTeste(t, nota.Chave.Valor(), func(e *eventoXML) { e.tipo = "110110" })

	evento, err := nota.AplicarEvento(carta, uuid.New(), uuid.New(), agoraDoTeste)
	if err != nil {
		t.Fatalf("AplicarEvento = %v", err)
	}
	if evento.Tipo != domainnotafiscal.TipoEventoCartaCorrecao || nota.Status != domainnotafiscal.StatusAutorizada {
		t.Errorf("tipo %q status %q", evento.Tipo, nota.Status)
	}
}

func TestAplicarEventoRecusaOutraNotaEOutroAutor(t *testing.T) {
	nota := notaDoTeste(t)
	outraChave := chaveDe(cnpjEmitente, "2026-10-10", 1, 777, "55")

	if _, err := nota.AplicarEvento(eventoDoTeste(t, outraChave), uuid.New(), uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("evento de outra nota = %v, esperado validação", err)
	}

	outroAutor := eventoDoTeste(t, nota.Chave.Valor(), func(e *eventoXML) { e.autor = cnpjDestinatario })
	if _, err := nota.AplicarEvento(outroAutor, uuid.New(), uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("evento de outro autor = %v, esperado validação", err)
	}
	if _, err := nota.AplicarEvento(nil, uuid.New(), uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("evento nulo = %v, esperado validação", err)
	}
	if nota.Status != domainnotafiscal.StatusAutorizada {
		t.Errorf("evento recusado mudou o status para %q", nota.Status)
	}
}
