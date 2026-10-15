package pedido

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

type consultaUseCaseFalso struct {
	pendencias         []*domainpedido.Pendencia
	pedidos            []*domainpedido.Pedido
	inputPendencias    portsin.ConsultarPendenciasInput
	inputPrazos        portsin.ConsultarPrazosInput
	inputCancelamentos portsin.ConsultarCancelamentosInput
}

func (f *consultaUseCaseFalso) Pendencias(_ context.Context, input portsin.ConsultarPendenciasInput) ([]*domainpedido.Pendencia, error) {
	f.inputPendencias = input
	return f.pendencias, nil
}

func (f *consultaUseCaseFalso) Prazos(_ context.Context, input portsin.ConsultarPrazosInput) ([]*domainpedido.Pedido, error) {
	f.inputPrazos = input
	return f.pedidos, nil
}

func (f *consultaUseCaseFalso) Cancelamentos(_ context.Context, input portsin.ConsultarCancelamentosInput) ([]*domainpedido.Pedido, error) {
	f.inputCancelamentos = input
	return f.pedidos, nil
}

var _ portsin.ConsultaUseCase = (*consultaUseCaseFalso)(nil)

func TestPendenciasDevolvemOsCalculosPorItem(t *testing.T) {
	produto := uuid.New()
	usecase := &consultaUseCaseFalso{pendencias: []*domainpedido.Pendencia{
		{Numero: 7, Vendido: 10, Entregue: 2, Programado: 3, Reservado: 5, DisponivelNaEmpresa: 9, ProdutoNome: "Cimento"},
	}}

	resposta := executaProtegido(t, NewConsultaHandler(usecase).Pendencias, http.MethodGet, "/api/v1/pedidos/pendencias?produto="+produto.String(), "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.inputPendencias.ProdutoID == nil || *usecase.inputPendencias.ProdutoID != produto {
		t.Errorf("produto repassado = %v", usecase.inputPendencias.ProdutoID)
	}
	var pagina dto.Paginado[pedidodto.PendenciaResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if linha := pagina.Dados[0]; linha.Restante != 8 || linha.FaltaCobrir != 3 || linha.PodeCobrirAgora != 3 || linha.Numero != 7 {
		t.Errorf("pendência devolvida = %+v", linha)
	}

	resposta = executaProtegido(t, NewConsultaHandler(usecase).Pendencias, http.MethodGet, "/api/v1/pedidos/pendencias?cliente=x", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestPrazosDevolvemDiasParaOPrazo(t *testing.T) {
	pedido := pedidoDeTeste(t)
	prazo := domainpedido.DataDe(time.Now().UTC()).AddDate(0, 0, 5)
	pedido.PrazoLimite = &prazo
	usecase := &consultaUseCaseFalso{pedidos: []*domainpedido.Pedido{pedido}}

	resposta := executaProtegido(t, NewConsultaHandler(usecase).Prazos, http.MethodGet, "/api/v1/pedidos/prazos?situacao=a_vencer&dias=10", "", "")

	if resposta.Code != http.StatusOK || usecase.inputPrazos.Situacao != "a_vencer" || usecase.inputPrazos.Dias == nil || *usecase.inputPrazos.Dias != 10 {
		t.Fatalf("status %d, input %+v", resposta.Code, usecase.inputPrazos)
	}
	var pagina dto.Paginado[pedidodto.PedidoNoPrazoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if dias := pagina.Dados[0].DiasParaOPrazo; dias == nil || *dias != 5 {
		t.Errorf("dias para o prazo = %v, esperado 5", dias)
	}

	resposta = executaProtegido(t, NewConsultaHandler(usecase).Prazos, http.MethodGet, "/api/v1/pedidos/prazos?dias=muitos", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestCancelamentosConvertemPeriodoEDevolucao(t *testing.T) {
	usecase := &consultaUseCaseFalso{}

	resposta := executaProtegido(t, NewConsultaHandler(usecase).Cancelamentos, http.MethodGet,
		"/api/v1/pedidos/cancelamentos?com_devolucao=true&de=2026-10-01&ate=2026-10-31", "", "")

	input := usecase.inputCancelamentos
	if resposta.Code != http.StatusOK || !input.ComDevolucao || input.De == nil || input.Ate == nil ||
		!input.Ate.Equal(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("status %d, input %+v", resposta.Code, input)
	}

	for _, consulta := range []string{"com_devolucao=talvez", "de=01/10/2026", "ate=ontem", "empresa=x"} {
		resposta := executaProtegido(t, NewConsultaHandler(usecase).Cancelamentos, http.MethodGet, "/api/v1/pedidos/cancelamentos?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}
