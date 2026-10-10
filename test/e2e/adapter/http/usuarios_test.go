//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/google/uuid"
)

const (
	emailAdministrador = "admin@exemplo.com"
	senhaAdministrador = "Mia@2026Admin"
)

func autenticar(t *testing.T, servidor *httptest.Server, email, senha string) usuariosdto.SessaoResponse {
	t.Helper()

	resposta := enviaSemToken(t, servidor, http.MethodPost, "/api/v1/usuarios/autenticar", map[string]string{
		"email": email,
		"senha": senha,
	})
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarEnvelope[usuariosdto.SessaoResponse](t, resposta).Dados
}

func criarCargo(t *testing.T, servidor *httptest.Server, nome string, administrador bool) uuid.UUID {
	t.Helper()

	return criarCargoComPerfil(t, servidor, nome, administrador, false)
}

func criarCargoComPerfil(t *testing.T, servidor *httptest.Server, nome string, administrador, financeiro bool) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/cargos", map[string]any{
		"nome":          nome,
		"descricao":     "Cargo criado no teste",
		"administrador": administrador,
		"financeiro":    financeiro,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[usuariosdto.CargoResponse](t, resposta).Dados.ID
}

func criarUsuario(t *testing.T, servidor *httptest.Server, nome, email, senha string, cargoID uuid.UUID) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/usuarios", map[string]any{
		"nome":     nome,
		"email":    email,
		"senha":    senha,
		"cargo_id": cargoID,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[usuariosdto.UsuarioResponse](t, resposta).Dados.ID
}

func TestLoginDevolveTokenDeAdministrador(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	sessao := autenticar(t, servidor, emailAdministrador, senhaAdministrador)

	if sessao.Token == "" {
		t.Fatal("login não devolveu token")
	}
	if sessao.ExpiraEm.Before(time.Now().UTC()) {
		t.Errorf("expiração %v já vencida", sessao.ExpiraEm)
	}
	if !sessao.Administrador {
		t.Error("administrador do seed saiu sem perfil administrador")
	}
	if sessao.Usuario.Email != emailAdministrador {
		t.Errorf("usuário %q, esperado %q", sessao.Usuario.Email, emailAdministrador)
	}
	if sessao.Usuario.UltimoLogin.IsZero() {
		t.Error("login não registrou o último acesso")
	}
}

func TestLoginComSenhaErradaDevolveErro(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := enviaSemToken(t, servidor, http.MethodPost, "/api/v1/usuarios/autenticar", map[string]string{
		"email": emailAdministrador,
		"senha": "senhaErrada123",
	})
	conferirStatus(t, resposta, http.StatusNotFound)

	erro := decodificarErro(t, resposta)
	if erro.Erro.Mensagem != "email ou senha inválidos" {
		t.Errorf("mensagem %q, esperada %q", erro.Erro.Mensagem, "email ou senha inválidos")
	}
}

func TestRotaProtegidaSemTokenDevolveNaoAutenticado(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	protegidas := []string{"/api/v1/leads", "/api/v1/funils", "/api/v1/usuarios", "/api/v1/cargos"}

	for _, caminho := range protegidas {
		resposta := enviaSemToken(t, servidor, http.MethodGet, caminho, nil)

		if resposta.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status %d, esperado %d", caminho, resposta.StatusCode, http.StatusUnauthorized)
		}
		if resposta.Header.Get("WWW-Authenticate") == "" {
			t.Errorf("%s: resposta 401 sem cabeçalho WWW-Authenticate", caminho)
		}
	}

	resposta := enviaSemToken(t, servidor, http.MethodGet, "/health", nil)
	if resposta.StatusCode != http.StatusOK {
		t.Errorf("health com status %d, esperado %d", resposta.StatusCode, http.StatusOK)
	}
}

func TestUsuarioSemAdministracaoRecebeProibido(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Operador da obra", false)
	criarUsuario(t, servidor, "Bruno Lima", "bruno@exemplo.com", "senhaForte123", cargoID)

	sessao := autenticar(t, servidor, "bruno@exemplo.com", "senhaForte123")
	if sessao.Administrador {
		t.Fatal("usuário comum saiu como administrador")
	}

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/usuarios", nil, sessao.Token)
	if resposta.StatusCode != http.StatusForbidden {
		t.Errorf("listagem de usuários com status %d, esperado %d: %s", resposta.StatusCode, http.StatusForbidden, resposta.Status)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/cargos", map[string]any{"nome": "Invasor"}, sessao.Token)
	if resposta.StatusCode != http.StatusForbidden {
		t.Errorf("criação de cargo com status %d, esperado %d", resposta.StatusCode, http.StatusForbidden)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/leads", nil, sessao.Token)
	if resposta.StatusCode != http.StatusOK {
		t.Errorf("listagem de leads com status %d, esperado %d", resposta.StatusCode, http.StatusOK)
	}
}

func TestCRUDDeUsuarioComListagemPaginada(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Gerente de obra", false)
	usuarioID := criarUsuario(t, servidor, "Ana Souza", "ana@exemplo.com", "senhaForte123", cargoID)

	pagina := decodificarPagina[usuariosdto.UsuarioResponse](t, envia(t, servidor, http.MethodGet, "/api/v1/usuarios?pagina=1&tamanho=2", nil))
	if pagina.Pagina != 1 || pagina.Tamanho != 2 {
		t.Errorf("paginação %d/%d, esperado 1/2", pagina.Pagina, pagina.Tamanho)
	}
	if len(pagina.Dados) != 2 {
		t.Fatalf("página devolveu %d usuários, esperado 2", len(pagina.Dados))
	}
	if pagina.Dados[0].Email != emailAdministrador || pagina.Dados[1].Email != "ana@exemplo.com" {
		t.Errorf("ordem inesperada: %q, %q", pagina.Dados[0].Email, pagina.Dados[1].Email)
	}

	resposta := envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+usuarioID.String(), map[string]any{
		"nome":     "Ana Alterada",
		"email":    "ana@exemplo.com",
		"cargo_id": cargoID,
		"ativo":    true,
	})
	conferirStatus(t, resposta, http.StatusOK)

	atualizado := decodificarEnvelope[usuariosdto.UsuarioResponse](t, resposta).Dados
	if atualizado.Nome != "Ana Alterada" || !atualizado.Ativo {
		t.Errorf("usuário atualizado %+v, esperado Ana Alterada e ativo", atualizado)
	}

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/usuarios/"+usuarioID.String()+"/senha", map[string]string{
		"nova_senha": "senhaNova123",
	})
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = enviaSemToken(t, servidor, http.MethodPost, "/api/v1/usuarios/autenticar", map[string]string{
		"email": "ana@exemplo.com",
		"senha": "senhaForte123",
	})
	conferirStatus(t, resposta, http.StatusNotFound)

	novaSessao := autenticar(t, servidor, "ana@exemplo.com", "senhaNova123")
	if novaSessao.Administrador {
		t.Error("usuário comum saiu como administrador")
	}

	resposta = envia(t, servidor, http.MethodPatch, "/api/v1/usuarios/"+usuarioID.String()+"/desativar", nil)
	conferirStatus(t, resposta, http.StatusOK)

	busca := decodificarPagina[usuariosdto.UsuarioResponse](t, envia(t, servidor, http.MethodGet, "/api/v1/usuarios?q=ana", nil))
	if len(busca.Dados) != 1 || busca.Dados[0].Ativo {
		t.Errorf("busca devolveu %+v, esperado a usuária desativada", busca.Dados)
	}

	resposta = envia(t, servidor, http.MethodPatch, "/api/v1/usuarios/"+usuarioID.String()+"/ativar", nil)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = envia(t, servidor, http.MethodDelete, "/api/v1/usuarios/"+usuarioID.String(), nil)
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/usuarios/"+usuarioID.String(), nil)
	conferirStatus(t, resposta, http.StatusNotFound)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/leads", nil, novaSessao.Token)
	conferirStatus(t, resposta, http.StatusUnauthorized)
}

func TestExcluirCargoEmUsoDevolveErroDeValidacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Encarregado", false)
	criarUsuario(t, servidor, "Carla Dias", "carla@exemplo.com", "senhaForte123", cargoID)

	resposta := envia(t, servidor, http.MethodDelete, "/api/v1/cargos/"+cargoID.String(), nil)
	conferirStatus(t, resposta, http.StatusBadRequest)

	erro := decodificarErro(t, resposta)
	if erro.Erro.Codigo != http.StatusBadRequest || erro.Erro.Mensagem == "" {
		t.Errorf("erro %+v, esperado código 400 com mensagem", erro.Erro)
	}
}

func TestSessaoDevolvePerfilDoUsuarioLogado(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Financeiro da obra", false)
	criarUsuario(t, servidor, "Duda Prado", "duda@exemplo.com", "senhaForte123", cargoID)
	sessao := autenticar(t, servidor, "duda@exemplo.com", "senhaForte123")

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, sessao.Token)
	conferirStatus(t, resposta, http.StatusOK)

	perfil := decodificarEnvelope[usuariosdto.UsuarioResponse](t, resposta).Dados
	if perfil.Email != "duda@exemplo.com" {
		t.Errorf("e-mail %q, esperado %q", perfil.Email, "duda@exemplo.com")
	}
	if perfil.CargoID != cargoID {
		t.Errorf("cargo %s, esperado %s", perfil.CargoID, cargoID)
	}
	if !perfil.Ativo {
		t.Error("perfil do sessão saiu inativo")
	}
}

func TestSessaoSemTokenDevolveNaoAutenticado(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil)
	conferirStatus(t, resposta, http.StatusUnauthorized)

	resposta = enviaSemToken(t, servidor, http.MethodPost, "/api/v1/sessao/senha", map[string]string{
		"nova_senha": "senhaNova123",
	})
	conferirStatus(t, resposta, http.StatusUnauthorized)
}

func TestTrocarSenhaDaPropriaSessao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Almoxarife", false)
	criarUsuario(t, servidor, "Tiago Melo", "tiago@exemplo.com", "senhaForte123", cargoID)
	sessao := autenticar(t, servidor, "tiago@exemplo.com", "senhaForte123")

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/sessao/senha", map[string]string{
		"nova_senha": "senhaNova123",
	}, sessao.Token)
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/sessao/senha", map[string]string{
		"senha_atual": "senhaErrada1",
		"nova_senha":  "senhaNova123",
	}, sessao.Token)
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/sessao/senha", map[string]string{
		"senha_atual": "senhaForte123",
		"nova_senha":  "curta",
	}, sessao.Token)
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/sessao/senha", map[string]string{
		"senha_atual": "senhaForte123",
		"nova_senha":  "senhaNova123",
	}, sessao.Token)
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, sessao.Token)
	conferirStatus(t, resposta, http.StatusUnauthorized)

	novaSessao := autenticar(t, servidor, "tiago@exemplo.com", "senhaNova123")

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, novaSessao.Token)
	conferirStatus(t, resposta, http.StatusOK)
}

func TestEncerrarSessaoRevogaOsTokensEmitidos(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Almoxarife", false)
	criarUsuario(t, servidor, "Rita Paz", "rita@exemplo.com", "senhaForte123", cargoID)
	primeira := autenticar(t, servidor, "rita@exemplo.com", "senhaForte123")
	segunda := autenticar(t, servidor, "rita@exemplo.com", "senhaForte123")

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/sessao/encerrar", nil, primeira.Token)
	conferirStatus(t, resposta, http.StatusNoContent)

	for _, token := range []string{primeira.Token, segunda.Token} {
		resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, token)
		conferirStatus(t, resposta, http.StatusUnauthorized)
	}

	nova := autenticar(t, servidor, "rita@exemplo.com", "senhaForte123")

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, nova.Token)
	conferirStatus(t, resposta, http.StatusOK)
}

func TestTrocaDeSenhaPeloAdministradorRevogaOsTokensDoUsuario(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoID := criarCargo(t, servidor, "Almoxarife", false)
	id := criarUsuario(t, servidor, "Caio Reis", "caio@exemplo.com", "senhaForte123", cargoID)
	sessao := autenticar(t, servidor, "caio@exemplo.com", "senhaForte123")

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/usuarios/"+id.String()+"/senha", map[string]string{
		"nova_senha": "senhaNova123",
	}, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/sessao", nil, sessao.Token)
	conferirStatus(t, resposta, http.StatusUnauthorized)
}
