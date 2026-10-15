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
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

type pedidoUseCaseFalso struct {
	erro           error
	pedido         *domainpedido.Pedido
	reservas       []*domainestoque.Reserva
	inputCriar     portsin.CriarPedidoInput
	inputAlterar   portsin.AlterarPedidoInput
	inputListar    portsin.ListarPedidosInput
	inputTransicao portsin.TransicaoPedidoInput
	inputEntrada   portsin.RegistrarEntradaInput
	inputCancelar  portsin.CancelarPedidoInput
}

func (f *pedidoUseCaseFalso) Criar(_ context.Context, input portsin.CriarPedidoInput) (*domainpedido.Pedido, error) {
	f.inputCriar = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) Obter(context.Context, uuid.UUID, uuid.UUID) (*domainpedido.Pedido, error) {
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) Listar(_ context.Context, input portsin.ListarPedidosInput) ([]*domainpedido.Pedido, error) {
	f.inputListar = input
	return []*domainpedido.Pedido{f.pedido}, f.erro
}

func (f *pedidoUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarPedidoInput) (*domainpedido.Pedido, error) {
	f.inputAlterar = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) Confirmar(_ context.Context, input portsin.TransicaoPedidoInput) (*domainpedido.Pedido, error) {
	f.inputTransicao = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) ReservarPendencias(_ context.Context, input portsin.TransicaoPedidoInput) (*domainpedido.Pedido, error) {
	f.inputTransicao = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) RegistrarEntrada(_ context.Context, input portsin.RegistrarEntradaInput) (*domainpedido.Pedido, error) {
	f.inputEntrada = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) Cancelar(_ context.Context, input portsin.CancelarPedidoInput) (*domainpedido.Pedido, error) {
	f.inputCancelar = input
	return f.pedido, f.erro
}

func (f *pedidoUseCaseFalso) ObterComReservas(context.Context, uuid.UUID, uuid.UUID) (*domainpedido.Pedido, []*domainestoque.Reserva, error) {
	return f.pedido, f.reservas, f.erro
}

func (f *pedidoUseCaseFalso) ListarHistorico(context.Context, uuid.UUID, uuid.UUID) ([]*domainpedido.Historico, error) {
	return nil, f.erro
}

var _ portsin.PedidoUseCase = (*pedidoUseCaseFalso)(nil)

func pedidoDeTeste(t *testing.T) *domainpedido.Pedido {
	t.Helper()

	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), domainpedido.DadosPedido{
		ClienteID: uuid.New(),
		Modelo:    domainpedido.ModeloImportacao,
		Itens:     []domainpedido.ItemInput{{ProdutoID: uuid.New(), Quantidade: 10, PrecoUnitarioCentavos: 1500}},
	})
	if err != nil {
		t.Fatalf("pedido de teste inválido: %v", err)
	}

	return pedido
}

func TestCriarPedidoRepassaCorpoEConverteAData(t *testing.T) {
	pedido := pedidoDeTeste(t)
	usecase := &pedidoUseCaseFalso{pedido: pedido}
	empresa, cliente, produto := uuid.New(), uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewPedidoHandler(usecase).Criar, http.MethodPost, "/api/v1/pedidos", "",
		`{"empresa_id":"`+empresa.String()+`","cliente_id":"`+cliente.String()+`","modelo":"pronta_entrega","prazo_limite":"2026-12-20",`+
			`"itens":[{"produto_id":"`+produto.String()+`","quantidade":3,"preco_unitario_centavos":990}]}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado 201: %s", resposta.Code, resposta.Body.String())
	}
	input := usecase.inputCriar
	if input.EmpresaID != empresa || input.Dados.ClienteID != cliente || input.Dados.Modelo != "pronta_entrega" || len(input.Dados.Itens) != 1 ||
		input.Dados.Itens[0].PrecoUnitarioCentavos != 990 || input.Dados.PrazoLimite == nil ||
		!input.Dados.PrazoLimite.Equal(time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("input repassado = %+v", input)
	}

	var envelope dto.Resposta[pedidodto.PedidoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	dados := envelope.Dados
	if dados.ID != pedido.ID || dados.Condicao.EntradaBP != 3000 || dados.Condicao.PrazoEntregaDias == nil || *dados.Condicao.PrazoEntregaDias != 120 ||
		len(dados.Itens) != 1 || dados.Itens[0].SubtotalCentavos != 15000 || dados.Itens[0].Restante != 10 || dados.Itens[0].Reservado != nil {
		t.Errorf("pedido devolvido = %+v", dados)
	}
}

func TestCriarPedidoRecusaCorpoInvalido(t *testing.T) {
	usecase := &pedidoUseCaseFalso{pedido: pedidoDeTeste(t)}

	for _, corpo := range []string{
		`{"modelo":"pronta_entrega","prazo_limite":"20/12/2026"}`,
		`{"modelo":"pronta_entrega","desconhecido":1}`,
		`{"modelo":`,
	} {
		resposta := executaProtegido(t, NewPedidoHandler(usecase).Criar, http.MethodPost, "/api/v1/pedidos", "", corpo)
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
	if usecase.inputCriar.UsuarioID != uuid.Nil {
		t.Error("corpo inválido chegou ao usecase")
	}
}

func TestObterPedidoMostraReservadoEFaltaPorItem(t *testing.T) {
	pedido := pedidoDeTeste(t)
	itemID := pedido.Itens[0].ID
	reserva := &domainestoque.Reserva{
		ID: uuid.New(), Quantidade: 4, Origem: domainestoque.OrigemReservaPedidoItem, OrigemID: &itemID, Status: domainestoque.StatusReservaAtiva,
	}
	usecase := &pedidoUseCaseFalso{pedido: pedido, reservas: []*domainestoque.Reserva{reserva}}

	resposta := executaProtegido(t, NewPedidoHandler(usecase).Obter, http.MethodGet, "/api/v1/pedidos/"+pedido.ID.String(), pedido.ID.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	var envelope dto.Resposta[pedidodto.PedidoDetalheResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	item := envelope.Dados.Itens[0]
	if item.Reservado == nil || *item.Reservado != 4 || item.FaltaCobrir == nil || *item.FaltaCobrir != 6 || len(envelope.Dados.Reservas) != 1 {
		t.Errorf("detalhe = %+v", envelope.Dados)
	}
}

func TestListarPedidosValidaConsultas(t *testing.T) {
	usecase := &pedidoUseCaseFalso{pedido: pedidoDeTeste(t)}
	empresa := uuid.New()

	resposta := executaProtegido(t, NewPedidoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/pedidos?empresa="+empresa.String()+"&status=confirmado&modelo=importacao&numero=12&pagina=2", "", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	filtro := usecase.inputListar
	if filtro.Empresa == nil || *filtro.Empresa != empresa || filtro.Status != "confirmado" || filtro.Modelo != "importacao" ||
		filtro.Numero == nil || *filtro.Numero != 12 || filtro.Filtro.Page != 2 {
		t.Errorf("filtro repassado = %+v", filtro)
	}

	for _, consulta := range []string{"empresa=x", "cliente=x", "representante=x", "numero=abc", "numero=0"} {
		resposta := executaProtegido(t, NewPedidoHandler(usecase).Listar, http.MethodGet, "/api/v1/pedidos?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}

func TestTransicoesDoPedidoRepassamVersao(t *testing.T) {
	pedido := pedidoDeTeste(t)
	usecase := &pedidoUseCaseFalso{pedido: pedido}
	handler := NewPedidoHandler(usecase)
	caminho := "/api/v1/pedidos/" + pedido.ID.String()

	resposta := executaProtegido(t, handler.Confirmar, http.MethodPost, caminho+"/confirmar", pedido.ID.String(), `{"versao":3}`)
	if resposta.Code != http.StatusOK || usecase.inputTransicao.Versao != 3 || usecase.inputTransicao.PedidoID != pedido.ID {
		t.Errorf("confirmar: status %d, input %+v", resposta.Code, usecase.inputTransicao)
	}

	resposta = executaProtegido(t, handler.RegistrarEntrada, http.MethodPost, caminho+"/entrada", pedido.ID.String(),
		`{"versao":2,"pago_em":"2026-10-10","valor_centavos":4500}`)
	if resposta.Code != http.StatusOK || usecase.inputEntrada.ValorCentavos != 4500 ||
		!usecase.inputEntrada.PagaEm.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("entrada: status %d, input %+v", resposta.Code, usecase.inputEntrada)
	}
	resposta = executaProtegido(t, handler.RegistrarEntrada, http.MethodPost, caminho+"/entrada", pedido.ID.String(), `{"versao":2,"valor_centavos":4500}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)

	resposta = executaProtegido(t, handler.Cancelar, http.MethodPost, caminho+"/cancelar", pedido.ID.String(),
		`{"versao":4,"motivo":"obra parada","valor_a_devolver_centavos":100}`)
	cancelar := usecase.inputCancelar
	if resposta.Code != http.StatusOK || cancelar.Motivo != "obra parada" || cancelar.ValorADevolverCentavos == nil || *cancelar.ValorADevolverCentavos != 100 {
		t.Errorf("cancelar: status %d, input %+v", resposta.Code, cancelar)
	}
}

func TestErrosDoPedidoViramStatus(t *testing.T) {
	pedido := pedidoDeTeste(t)
	casos := []struct {
		erro   error
		status int
	}{
		{domainpedido.ErroVersaoDivergente(), http.StatusConflict},
		{domain.ErroPermissao("perfil sem acesso a esta empresa"), http.StatusForbidden},
		{domain.ErroNaoEncontrado("pedido não encontrado"), http.StatusNotFound},
		{domain.ErroValidacao("cliente inativo não recebe pedido novo"), http.StatusBadRequest},
	}

	for _, caso := range casos {
		usecase := &pedidoUseCaseFalso{pedido: pedido, erro: caso.erro}
		resposta := executaProtegido(t, NewPedidoHandler(usecase).Confirmar, http.MethodPost, "/api/v1/pedidos/x/confirmar", pedido.ID.String(), `{"versao":1}`)
		apoioteste.VerificarEnvelopeDeErro(t, resposta, caso.status)
	}
}
