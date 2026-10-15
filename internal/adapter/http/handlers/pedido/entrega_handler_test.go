package pedido

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

type entregaUseCaseFalso struct {
	erro           error
	entrega        *domainpedido.Entrega
	inputCriar     portsin.CriarEntregaInput
	inputListar    portsin.ListarEntregasInput
	inputTransicao portsin.TransicaoEntregaInput
}

func (f *entregaUseCaseFalso) Criar(_ context.Context, input portsin.CriarEntregaInput) (*domainpedido.Entrega, error) {
	f.inputCriar = input
	return f.entrega, f.erro
}

func (f *entregaUseCaseFalso) Obter(context.Context, uuid.UUID, uuid.UUID) (*domainpedido.Entrega, error) {
	return f.entrega, f.erro
}

func (f *entregaUseCaseFalso) ListarDoPedido(context.Context, uuid.UUID, uuid.UUID) ([]*domainpedido.Entrega, error) {
	return []*domainpedido.Entrega{f.entrega}, f.erro
}

func (f *entregaUseCaseFalso) Listar(_ context.Context, input portsin.ListarEntregasInput) ([]*domainpedido.Entrega, error) {
	f.inputListar = input
	return []*domainpedido.Entrega{f.entrega}, f.erro
}

func (f *entregaUseCaseFalso) RegistrarSaida(_ context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	f.inputTransicao = input
	return f.entrega, f.erro
}

func (f *entregaUseCaseFalso) Concluir(_ context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	f.inputTransicao = input
	return f.entrega, f.erro
}

func (f *entregaUseCaseFalso) Cancelar(_ context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	f.inputTransicao = input
	return f.entrega, f.erro
}

var _ portsin.EntregaUseCase = (*entregaUseCaseFalso)(nil)

func entregaDeTeste() *domainpedido.Entrega {
	return &domainpedido.Entrega{
		ID:       uuid.New(),
		PedidoID: uuid.New(),
		Status:   domainpedido.StatusEntregaPendente,
		Frete:    domainpedido.FreteDe(domainpedido.TipoFreteFOB, nil),
		Versao:   1,
	}
}

func TestCriarEntregaRepassaItensEPrevisao(t *testing.T) {
	usecase := &entregaUseCaseFalso{entrega: entregaDeTeste()}
	pedidoID, itemID, localID := uuid.New(), uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewEntregaHandler(usecase).Criar, http.MethodPost, "/api/v1/pedidos/"+pedidoID.String()+"/entregas", pedidoID.String(),
		`{"transportadora":"Rodonaves","frete_tipo":"cif","frete_centavos":18000,"previsao":"2026-10-20",`+
			`"itens":[{"pedido_item_id":"`+itemID.String()+`","local_id":"`+localID.String()+`","quantidade":5}]}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	input := usecase.inputCriar
	if input.PedidoID != pedidoID || input.FreteTipo != "cif" || input.FreteCentavos == nil || *input.FreteCentavos != 18000 ||
		input.Previsao == nil || !input.Previsao.Equal(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)) ||
		len(input.Itens) != 1 || input.Itens[0].LocalID != localID || input.Itens[0].Quantidade != 5 {
		t.Errorf("input repassado = %+v", input)
	}

	resposta = executaProtegido(t, NewEntregaHandler(usecase).Criar, http.MethodPost, "/api/v1/pedidos/x/entregas", pedidoID.String(),
		`{"frete_tipo":"fob","previsao":"amanhã"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestTransicoesDaEntregaConvertemAsDatas(t *testing.T) {
	entrega := entregaDeTeste()
	usecase := &entregaUseCaseFalso{entrega: entrega}
	handler := NewEntregaHandler(usecase)
	id := entrega.ID.String()

	resposta := executaProtegido(t, handler.RegistrarSaida, http.MethodPost, "/api/v1/entregas/"+id+"/saiu", id, `{"versao":1}`)
	if resposta.Code != http.StatusOK || usecase.inputTransicao.Versao != 1 || !usecase.inputTransicao.Data.IsZero() {
		t.Errorf("saída sem data: status %d, input %+v", resposta.Code, usecase.inputTransicao)
	}

	resposta = executaProtegido(t, handler.Concluir, http.MethodPost, "/api/v1/entregas/"+id+"/entregue", id, `{"versao":2,"entregue_em":"2026-10-12"}`)
	if resposta.Code != http.StatusOK || !usecase.inputTransicao.Data.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("conclusão: status %d, input %+v", resposta.Code, usecase.inputTransicao)
	}

	resposta = executaProtegido(t, handler.RegistrarSaida, http.MethodPost, "/api/v1/entregas/"+id+"/saiu", id, `{"versao":1,"saiu_em":"12-10-2026"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)

	resposta = executaProtegido(t, handler.Cancelar, http.MethodPost, "/api/v1/entregas/"+id+"/cancelar", id, `{"versao":1,"saiu_em":"2026-10-12"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestListarEntregasValidaConsultas(t *testing.T) {
	usecase := &entregaUseCaseFalso{entrega: entregaDeTeste()}

	resposta := executaProtegido(t, NewEntregaHandler(usecase).Listar, http.MethodGet, "/api/v1/entregas?status=pendente&previsao_ate=2026-10-30", "", "")
	if resposta.Code != http.StatusOK || usecase.inputListar.Status != "pendente" || usecase.inputListar.PrevisaoAte == nil {
		t.Errorf("listar: status %d, input %+v", resposta.Code, usecase.inputListar)
	}

	for _, consulta := range []string{"empresa=x", "previsao_ate=ontem"} {
		resposta := executaProtegido(t, NewEntregaHandler(usecase).Listar, http.MethodGet, "/api/v1/entregas?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}
