package financeiro

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	financeirodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

type contaUseCaseFalso struct {
	portsin.ContaBancariaUseCase
	conta          *domainfinanceiro.ContaBancaria
	erro           error
	inputCriar     portsin.CriarContaBancariaInput
	inputAlterar   portsin.AlterarContaBancariaInput
	inputSituacao  portsin.MudarSituacaoContaBancariaInput
	inputListar    portsin.ListarContasBancariasInput
	obtidaPor      uuid.UUID
	removida       uuid.UUID
	removidaPor    uuid.UUID
	chamadas       int
	ativacoes      int
	desativacoes   int
	chamadasListar int
}

func (f *contaUseCaseFalso) Criar(_ context.Context, input portsin.CriarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	f.inputCriar = input
	f.chamadas++
	return f.conta, f.erro
}

func (f *contaUseCaseFalso) Obter(_ context.Context, _, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	f.obtidaPor = usuarioID
	return f.conta, f.erro
}

func (f *contaUseCaseFalso) Listar(_ context.Context, input portsin.ListarContasBancariasInput) ([]*domainfinanceiro.ContaBancaria, error) {
	f.inputListar = input
	f.chamadasListar++
	return []*domainfinanceiro.ContaBancaria{f.conta}, f.erro
}

func (f *contaUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	f.inputAlterar = input
	return f.conta, f.erro
}

func (f *contaUseCaseFalso) Ativar(_ context.Context, input portsin.MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	f.inputSituacao = input
	f.ativacoes++
	return f.conta, f.erro
}

func (f *contaUseCaseFalso) Desativar(_ context.Context, input portsin.MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error) {
	f.inputSituacao = input
	f.desativacoes++
	return f.conta, f.erro
}

func (f *contaUseCaseFalso) Remover(_ context.Context, contaID, usuarioID uuid.UUID) error {
	f.removida = contaID
	f.removidaPor = usuarioID
	return f.erro
}

func (f *contaUseCaseFalso) Bancos() []domainfinanceiro.Banco {
	return domainfinanceiro.Bancos()
}

func contaDeTeste(t *testing.T) *domainfinanceiro.ContaBancaria {
	t.Helper()

	agencia, err := domainfinanceiro.NovaAgencia("0001", nil)
	if err != nil {
		t.Fatalf("agência inválida: %v", err)
	}
	numero, err := domainfinanceiro.NovaConta("12345-6", nil)
	if err != nil {
		t.Fatalf("conta inválida: %v", err)
	}
	conta, err := domainfinanceiro.NovaContaBancaria(domainfinanceiro.DadosContaBancaria{
		EmpresaID: uuid.New(),
		Banco:     domainfinanceiro.Cora,
		Tipo:      domainfinanceiro.TipoContaCorrente,
		Agencia:   agencia,
		Conta:     numero,
		Apelido:   "Cora movimento",
	})
	if err != nil {
		t.Fatalf("conta de teste inválida: %v", err)
	}

	return conta
}

func TestCriarContaBancariaRepassaCorpoEUsuario(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}
	empresaID := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewContaBancariaHandler(usecase).Criar, http.MethodPost, "/api/v1/contas-bancarias", "",
		`{"empresa_id":"`+empresaID.String()+`","banco":"403","tipo":"corrente","agencia":"0001","agencia_digito":"9","conta":"12345","conta_digito":"6","apelido":"Cora movimento"}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	esperado := portsin.CriarContaBancariaInput{
		UsuarioID: usuarioID, EmpresaID: empresaID, Banco: "403", Tipo: "corrente", Agencia: "0001",
		Conta: "12345", Apelido: "Cora movimento",
	}
	input := usecase.inputCriar
	if *input.AgenciaDigito != "9" || *input.ContaDigito != "6" {
		t.Errorf("dígitos = %v/%v", input.AgenciaDigito, input.ContaDigito)
	}
	input.AgenciaDigito, input.ContaDigito = nil, nil
	if input != esperado {
		t.Errorf("input = %+v", input)
	}

	corpo := decodificar(t, registrador)
	if corpo["banco"] != "403" || corpo["banco_nome"] != "Cora" || corpo["tipo"] != "corrente" {
		t.Errorf("banco na resposta = %v", corpo)
	}
	if corpo["agencia"] != "0001" || corpo["agencia_digito"] != nil || corpo["agencia_formatada"] != "0001" {
		t.Errorf("agência na resposta = %v %v %v", corpo["agencia"], corpo["agencia_digito"], corpo["agencia_formatada"])
	}
	if corpo["conta"] != "12345" || corpo["conta_digito"] != "6" || corpo["conta_formatada"] != "12345-6" {
		t.Errorf("conta na resposta = %v %v %v", corpo["conta"], corpo["conta_digito"], corpo["conta_formatada"])
	}
	if corpo["versao"] != float64(1) || corpo["ativa"] != true || corpo["apelido"] != "Cora movimento" {
		t.Errorf("resposta = %v", corpo)
	}
	if _, existe := corpo["agencia_digito"]; !existe {
		t.Error("resposta deveria trazer agencia_digito, mesmo nulo")
	}
}

func TestCriarContaBancariaRecusaCorpoInvalido(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}

	for nome, corpo := range map[string]string{
		"campo desconhecido": `{"apelido":"X","versao":1}`,
		"empresa inválida":   `{"empresa_id":"abc"}`,
		"json quebrado":      `{"apelido":`,
		"corpo vazio":        ``,
	} {
		t.Run(nome, func(t *testing.T) {
			registrador, _ := executaProtegido(t, NewContaBancariaHandler(usecase).Criar, http.MethodPost, "/api/v1/contas-bancarias", "", corpo)
			apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
		})
	}
	if usecase.chamadas != 0 {
		t.Errorf("usecase chamado %d vezes com corpo inválido", usecase.chamadas)
	}
}

func TestCriarContaBancariaTraduzErrosDoUsecase(t *testing.T) {
	casos := []struct {
		erro     error
		status   int
		mensagem string
	}{
		{domainfinanceiro.ErroContaDuplicada(), http.StatusConflict, "já existe uma conta com este banco, agência e número"},
		{domain.ErroPermissao("perfil sem acesso a esta empresa"), http.StatusForbidden, "perfil sem acesso a esta empresa"},
		{domainfinanceiro.ErroEmpresaInativa(), http.StatusBadRequest, "erro de validação: empresa inativa não recebe conta bancária"},
	}

	for _, caso := range casos {
		t.Run(caso.mensagem, func(t *testing.T) {
			usecase := &contaUseCaseFalso{erro: caso.erro}
			registrador, _ := executaProtegido(t, NewContaBancariaHandler(usecase).Criar, http.MethodPost, "/api/v1/contas-bancarias", "", `{"apelido":"X"}`)

			apoioteste.VerificarEnvelopeDeErro(t, registrador, caso.status)
			if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); mensagem != caso.mensagem {
				t.Errorf("mensagem %q, esperada %q", mensagem, caso.mensagem)
			}
		})
	}
}

func TestListarContasBancariasRepassaFiltrosEPaginacao(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}
	empresaID := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewContaBancariaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/contas-bancarias?empresa="+empresaID.String()+"&banco=+403+&tipo=corrente&ativa=false&pagina=3&tamanho=7", "", "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	input := usecase.inputListar
	if input.UsuarioID != usuarioID || *input.Empresa != empresaID || input.Banco != "403" || input.Tipo != "corrente" || *input.Ativa {
		t.Errorf("input = %+v", input)
	}
	if input.Filtro.Page != 3 || input.Filtro.Size != 7 {
		t.Errorf("paginação = %+v", input.Filtro)
	}

	var pagina dto.Paginado[financeirodto.ContaBancariaResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if pagina.Pagina != 3 || pagina.Tamanho != 7 || len(pagina.Dados) != 1 || pagina.Dados[0].Apelido != "Cora movimento" {
		t.Errorf("página = %+v", pagina)
	}
}

func TestListarContasBancariasRecusaConsultaInvalida(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}

	for _, consulta := range []string{"empresa=abc", "ativa=talvez"} {
		registrador, _ := executaProtegido(t, NewContaBancariaHandler(usecase).Listar, http.MethodGet, "/api/v1/contas-bancarias?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	}
	if usecase.chamadasListar != 0 {
		t.Error("usecase não deveria ser chamado com consulta inválida")
	}
}

func TestObterContaBancariaRepassaUsuario(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}
	id := uuid.NewString()

	registrador, usuarioID := executaProtegido(t, NewContaBancariaHandler(usecase).Obter, http.MethodGet, "/api/v1/contas-bancarias/"+id, id, "")
	if registrador.Code != http.StatusOK || usecase.obtidaPor != usuarioID {
		t.Fatalf("status %d, obtida por %s", registrador.Code, usecase.obtidaPor)
	}

	usecase.erro = domainfinanceiro.ErroContaBancariaNaoEncontrada()
	registrador, _ = executaProtegido(t, NewContaBancariaHandler(usecase).Obter, http.MethodGet, "/api/v1/contas-bancarias/"+id, id, "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusNotFound)

	registrador, _ = executaProtegido(t, NewContaBancariaHandler(usecase).Obter, http.MethodGet, "/api/v1/contas-bancarias/abc", "abc", "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestAlterarContaBancariaSoAceitaApelidoEVersao(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}
	id := uuid.New()
	caminho := "/api/v1/contas-bancarias/" + id.String()

	registrador, usuarioID := executaProtegido(t, NewContaBancariaHandler(usecase).Alterar, http.MethodPut, caminho, id.String(), `{"apelido":"Novo","versao":4}`)
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.inputAlterar != (portsin.AlterarContaBancariaInput{UsuarioID: usuarioID, ContaID: id, Apelido: "Novo", Versao: 4}) {
		t.Errorf("input = %+v", usecase.inputAlterar)
	}

	for _, campo := range []string{`"banco":"001"`, `"conta":"1"`, `"empresa_id":"` + uuid.NewString() + `"`} {
		registrador, _ = executaProtegido(t, NewContaBancariaHandler(usecase).Alterar, http.MethodPut, caminho, id.String(), `{"apelido":"X","versao":4,`+campo+`}`)
		apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	}

	usecase.erro = domainfinanceiro.ErroContaBancariaAlterada()
	registrador, _ = executaProtegido(t, NewContaBancariaHandler(usecase).Alterar, http.MethodPut, caminho, id.String(), `{"apelido":"X","versao":1}`)
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusConflict)
	if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); mensagem != "a conta bancária foi alterada por outra pessoa; recarregue e tente de novo" {
		t.Errorf("mensagem %q", mensagem)
	}
}

func TestAtivarEDesativarContaBancariaExigemVersao(t *testing.T) {
	usecase := &contaUseCaseFalso{conta: contaDeTeste(t)}
	id := uuid.New()
	base := "/api/v1/contas-bancarias/" + id.String()
	handler := NewContaBancariaHandler(usecase)

	registrador, usuarioID := executaProtegido(t, handler.Ativar, http.MethodPatch, base+"/ativar", id.String(), `{"versao":2}`)
	if registrador.Code != http.StatusOK || usecase.inputSituacao != (portsin.MudarSituacaoContaBancariaInput{UsuarioID: usuarioID, ContaID: id, Versao: 2}) {
		t.Fatalf("ativar: status %d, input %+v", registrador.Code, usecase.inputSituacao)
	}

	registrador, _ = executaProtegido(t, handler.Desativar, http.MethodPatch, base+"/desativar", id.String(), `{"versao":3}`)
	if registrador.Code != http.StatusOK || usecase.inputSituacao.Versao != 3 {
		t.Fatalf("desativar: status %d, input %+v", registrador.Code, usecase.inputSituacao)
	}

	registrador, _ = executaProtegido(t, handler.Desativar, http.MethodPatch, base+"/desativar", id.String(), ``)
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	if usecase.ativacoes != 1 || usecase.desativacoes != 1 {
		t.Errorf("ativações %d e desativações %d", usecase.ativacoes, usecase.desativacoes)
	}
}

func TestRemoverContaBancaria(t *testing.T) {
	usecase := &contaUseCaseFalso{}
	id := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewContaBancariaHandler(usecase).Remover, http.MethodDelete, "/api/v1/contas-bancarias/"+id.String(), id.String(), "")
	if registrador.Code != http.StatusNoContent || registrador.Body.Len() != 0 {
		t.Errorf("status %d e corpo %q, esperado 204 vazio", registrador.Code, registrador.Body.String())
	}
	if usecase.removida != id || usecase.removidaPor != usuarioID {
		t.Errorf("removida %s por %s", usecase.removida, usecase.removidaPor)
	}

	usecase.erro = domain.ErroPermissao("perfil sem permissão para esta operação")
	registrador, _ = executaProtegido(t, NewContaBancariaHandler(usecase).Remover, http.MethodDelete, "/api/v1/contas-bancarias/"+id.String(), id.String(), "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusForbidden)
}

func TestBancosDevolveOCatalogo(t *testing.T) {
	registrador, _ := executaProtegido(t, NewContaBancariaHandler(&contaUseCaseFalso{}).Bancos, http.MethodGet, "/api/v1/bancos", "", "")

	var envelope dto.Resposta[[]financeirodto.BancoResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if registrador.Code != http.StatusOK || len(envelope.Dados) != 7 || envelope.Dados[5] != (financeirodto.BancoResponse{Codigo: "403", Nome: "Cora"}) {
		t.Errorf("status %d, bancos %+v", registrador.Code, envelope.Dados)
	}
}
