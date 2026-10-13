package cliente

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type clienteUseCaseFalso struct {
	erro         error
	cliente      *domaincliente.Cliente
	inputCriar   portsin.CriarClienteInput
	inputAlterar portsin.AlterarClienteInput
	inputListar  portsin.ListarClientesInput
	removido     uuid.UUID
}

func (f *clienteUseCaseFalso) Criar(_ context.Context, input portsin.CriarClienteInput) (*domaincliente.Cliente, error) {
	f.inputCriar = input
	return f.cliente, f.erro
}

func (f *clienteUseCaseFalso) Obter(_ context.Context, _, _ uuid.UUID) (*domaincliente.Cliente, error) {
	return f.cliente, f.erro
}

func (f *clienteUseCaseFalso) Listar(_ context.Context, input portsin.ListarClientesInput) ([]*domaincliente.Cliente, error) {
	f.inputListar = input
	return []*domaincliente.Cliente{f.cliente}, f.erro
}

func (f *clienteUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarClienteInput) (*domaincliente.Cliente, error) {
	f.inputAlterar = input
	return f.cliente, f.erro
}

func (f *clienteUseCaseFalso) Ativar(_ context.Context, _, _ uuid.UUID) (*domaincliente.Cliente, error) {
	return f.cliente, f.erro
}

func (f *clienteUseCaseFalso) Desativar(_ context.Context, _, _ uuid.UUID) (*domaincliente.Cliente, error) {
	return f.cliente, f.erro
}

func (f *clienteUseCaseFalso) Remover(_ context.Context, clienteID, _ uuid.UUID) error {
	f.removido = clienteID
	return f.erro
}

var _ portsin.ClienteUseCase = (*clienteUseCaseFalso)(nil)

func clienteDeTeste(t *testing.T) *domaincliente.Cliente {
	t.Helper()

	documento, err := domaincliente.NovoDocumento("11.222.333/0001-81")
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	cliente, err := domaincliente.NovoCliente(documento, domaincliente.DadosCliente{
		Nome:                  "Construtora Águia",
		Tipo:                  domaincliente.TipoClienteConstrutora,
		UF:                    domainempresa.UFDe("PB"),
		Cidade:                "João Pessoa",
		ExigeNumeroPedidoNaNF: true,
	}, nil)
	if err != nil {
		t.Fatalf("cliente de teste inválido: %v", err)
	}

	return cliente
}

func TestCriarClienteRepassaCorpoEDevolveDocumentoFormatado(t *testing.T) {
	cliente := clienteDeTeste(t)
	usecase := &clienteUseCaseFalso{cliente: cliente}
	grupoID := uuid.New()

	resposta := executaProtegido(t, NewClienteHandler(usecase).Criar, http.MethodPost, "/api/v1/clientes", "",
		`{"documento":"11.222.333/0001-81","nome":"Construtora Águia","tipo":"construtora","grupo_id":"`+grupoID.String()+
			`","uf":"PB","cidade":"João Pessoa","email_nf":"nf@aguia.com.br","exige_numero_pedido_na_nf":true,"proibe_antecipacao":true}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	input := usecase.inputCriar
	if input.Documento != "11.222.333/0001-81" || input.Dados.Nome != "Construtora Águia" || input.Dados.GrupoID == nil || *input.Dados.GrupoID != grupoID ||
		!input.Dados.ExigeNumeroPedidoNaNF || !input.Dados.ProibeAntecipacao || input.UsuarioID == uuid.Nil {
		t.Errorf("input repassado = %+v", input)
	}

	var envelope dto.Resposta[clientedto.ClienteResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	dados := envelope.Dados
	if dados.Documento != "11222333000181" || dados.DocumentoFormatado != "11.222.333/0001-81" || dados.TipoPessoa != "pj" || !dados.ExigeNumeroPedidoNaNF {
		t.Errorf("cliente devolvido = %+v", dados)
	}
}

func TestAlterarClienteNaoAceitaDocumento(t *testing.T) {
	cliente := clienteDeTeste(t)
	usecase := &clienteUseCaseFalso{cliente: cliente}

	resposta := executaProtegido(t, NewClienteHandler(usecase).Alterar, http.MethodPut, "/api/v1/clientes/"+cliente.ID.String(), cliente.ID.String(),
		`{"documento":"33400004000116","nome":"Outro","tipo":"loja","uf":"PB","cidade":"JP"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputAlterar.ClienteID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com documento no corpo")
	}

	resposta = executaProtegido(t, NewClienteHandler(usecase).Alterar, http.MethodPut, "/api/v1/clientes/"+cliente.ID.String(), cliente.ID.String(),
		`{"nome":"Outro","tipo":"loja","uf":"PB","cidade":"JP"}`)
	if resposta.Code != http.StatusOK || usecase.inputAlterar.ClienteID != cliente.ID {
		t.Errorf("alterar: status %d, input %+v", resposta.Code, usecase.inputAlterar)
	}
}

func TestListarClientesRepassaFiltrosEValidaConsultas(t *testing.T) {
	usecase := &clienteUseCaseFalso{cliente: clienteDeTeste(t)}
	grupo, representante, lead := uuid.New(), uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewClienteHandler(usecase).Listar, http.MethodGet,
		"/api/v1/clientes?busca=aguia&tipo=construtora&uf=pb&ativo=true&pagina=2&tamanho=5&grupo="+grupo.String()+
			"&representante="+representante.String()+"&lead="+lead.String(), "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	filtro := usecase.inputListar
	if filtro.Busca != "aguia" || filtro.Tipo != "construtora" || filtro.UF != "pb" || filtro.Ativo == nil || !*filtro.Ativo ||
		*filtro.GrupoID != grupo || *filtro.RepresentanteID != representante || *filtro.LeadID != lead || filtro.Filtro.Page != 2 {
		t.Errorf("filtro repassado = %+v", filtro)
	}

	for _, consulta := range []string{"grupo=x", "representante=x", "lead=x", "ativo=talvez"} {
		resposta := executaProtegido(t, NewClienteHandler(usecase).Listar, http.MethodGet, "/api/v1/clientes?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}

func TestErrosDoClienteViramStatus(t *testing.T) {
	cliente := clienteDeTeste(t)
	casos := []struct {
		erro   error
		status int
	}{
		{domain.ErroConflito("já existe um cliente com este documento"), http.StatusConflict},
		{domain.ErroPermissao("cliente de outro representante"), http.StatusForbidden},
		{domain.ErroNaoEncontrado("cliente não encontrado"), http.StatusNotFound},
		{domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído"), http.StatusBadRequest},
	}

	for _, caso := range casos {
		usecase := &clienteUseCaseFalso{cliente: cliente, erro: caso.erro}
		resposta := executaProtegido(t, NewClienteHandler(usecase).Remover, http.MethodDelete, "/api/v1/clientes/"+cliente.ID.String(), cliente.ID.String(), "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, caso.status)
	}

	usecase := &clienteUseCaseFalso{cliente: cliente}
	resposta := executaProtegido(t, NewClienteHandler(usecase).Remover, http.MethodDelete, "/api/v1/clientes/"+cliente.ID.String(), cliente.ID.String(), "")
	if resposta.Code != http.StatusNoContent || usecase.removido != cliente.ID {
		t.Errorf("remover: status %d, cliente %s", resposta.Code, usecase.removido)
	}
}
