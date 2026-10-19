package pedido

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
)

func TestDocumentoDaPropostaIdentificaNumeroERevisao(t *testing.T) {
	pedido := pedidoDeTeste(t)
	pedido.Numero = 57
	handler := NewPedidoHandler(&pedidoUseCaseFalso{pedido: pedido})

	registrador := executaProtegido(t, handler.Documento, http.MethodGet, "/api/v1/pedidos/x/proposta", pedido.ID.String(), "")
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}

	var corpo dto.Resposta[pedidodto.DocumentoPropostaResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	documento := corpo.Dados
	if documento.Identificacao != "Proposta 57-R1" || documento.Empresa.RazaoSocial != "MIA Materiais Ltda" ||
		documento.Validade == nil || len(documento.Itens) != 1 || len(documento.Condicoes) == 0 {
		t.Fatalf("documento = %+v", documento)
	}
}

func TestModelosDeVendaListaAsCondicoesPadrao(t *testing.T) {
	handler := NewPedidoHandler(&pedidoUseCaseFalso{})

	registrador := executaProtegido(t, handler.ModelosDeVenda, http.MethodGet, "/api/v1/modelos-venda", "", "")
	var corpo dto.Resposta[[]pedidodto.ModeloDeVendaResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if registrador.Code != http.StatusOK || len(corpo.Dados) != 2 || corpo.Dados[1].Modelo != "importacao" || corpo.Dados[1].Condicao.EntradaBP != 3000 {
		t.Fatalf("status %d, modelos %+v", registrador.Code, corpo.Dados)
	}
}
