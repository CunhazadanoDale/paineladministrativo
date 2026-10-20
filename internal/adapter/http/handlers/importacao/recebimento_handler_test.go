package importacao

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type recebimentoUseCaseFalso struct {
	portsin.RecebimentoUseCase
	alocacoes portsin.DefinirAlocacoesInput
	receber   portsin.ReceberInput
	chamadas  int
}

func (f *recebimentoUseCaseFalso) DefinirAlocacoes(_ context.Context, input portsin.DefinirAlocacoesInput) (*portsin.Visao, error) {
	f.alocacoes = input
	f.chamadas++

	return &portsin.Visao{Importacao: importacaoDeTeste()}, nil
}

func (f *recebimentoUseCaseFalso) Receber(_ context.Context, input portsin.ReceberInput) (*portsin.Visao, error) {
	f.receber = input
	f.chamadas++

	return &portsin.Visao{Importacao: importacaoDeTeste(), ComValores: true}, nil
}

func TestReceberLeOsItensRecebidos(t *testing.T) {
	usecase := &recebimentoUseCaseFalso{}
	handler := NewRecebimentoHandler(usecase)
	id := uuid.NewString()
	item, local := uuid.New(), uuid.New()

	registrador := executaProtegido(t, handler.Receber, http.MethodPost, "/api/v1/importacoes/"+id+"/receber", id,
		`{"versao": 3, "recebida_em": "2026-10-15", "local_id": "`+local.String()+`", "cambio_micro": 5400000,
		  "itens": [{"item_id": "`+item.String()+`", "quantidade_recebida": 0}]}`)
	if registrador.Code != http.StatusOK || usecase.receber.LocalID != local || usecase.receber.Versao != 3 ||
		*usecase.receber.CambioMicro != 5400000 || usecase.receber.RecebidaEm.Format("2006-01-02") != "2026-10-15" {
		t.Fatalf("status %d, entrada %+v", registrador.Code, usecase.receber)
	}
	if quantidade, ok := usecase.receber.Recebidas[item]; !ok || quantidade != 0 {
		t.Errorf("recebidas = %+v", usecase.receber.Recebidas)
	}
	if !strings.Contains(registrador.Body.String(), `"alocacoes":[]`) {
		t.Errorf("detalhe sem alocações: %s", registrador.Body.String())
	}

	chamadas := usecase.chamadas
	apoioteste.VerificarEnvelopeDeErro(t, executaProtegido(t, handler.Receber, http.MethodPost, "/api/v1/importacoes/"+id+"/receber", id,
		`{"versao": 3, "recebida_em": "2026-10-15", "itens": [{"item_id": "`+item.String()+`"}, {"item_id": "`+item.String()+`"}]}`), http.StatusBadRequest)
	apoioteste.VerificarEnvelopeDeErro(t, executaProtegido(t, handler.Receber, http.MethodPost, "/api/v1/importacoes/"+id+"/receber", id,
		`{"versao": 3, "recebida_em": "15/10/2026"}`), http.StatusBadRequest)
	if usecase.chamadas != chamadas {
		t.Error("usecase chamado com entrada inválida")
	}
}

func TestDefinirAlocacoesRepassaALista(t *testing.T) {
	usecase := &recebimentoUseCaseFalso{}
	handler := NewRecebimentoHandler(usecase)
	id := uuid.NewString()
	item, pedidoItem := uuid.New(), uuid.New()

	registrador := executaProtegido(t, handler.DefinirAlocacoes, http.MethodPut, "/api/v1/importacoes/"+id+"/alocacoes", id,
		`{"versao": 2, "alocacoes": [{"importacao_item_id": "`+item.String()+`", "pedido_item_id": "`+pedidoItem.String()+`", "quantidade": 40}]}`)
	if registrador.Code != http.StatusOK || usecase.alocacoes.Versao != 2 || len(usecase.alocacoes.Alocacoes) != 1 ||
		usecase.alocacoes.Alocacoes[0].PedidoItemID != pedidoItem || usecase.alocacoes.Alocacoes[0].Quantidade != 40 {
		t.Errorf("status %d, entrada %+v", registrador.Code, usecase.alocacoes)
	}
	if strings.Contains(registrador.Body.String(), "valor_") {
		t.Errorf("visão sem valores mostrou valores: %s", registrador.Body.String())
	}
}
