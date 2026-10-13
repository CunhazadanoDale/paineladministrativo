//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/google/uuid"
)

func usuarioComercial(t *testing.T, servidor *httptest.Server, nome, email string) (uuid.UUID, string) {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/cargos", map[string]any{
		"nome":      "Comercial " + nome,
		"descricao": "Cargo comercial criado no teste",
		"comercial": true,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	cargo := decodificarEnvelope[usuariosdto.CargoResponse](t, resposta).Dados

	id := criarUsuario(t, servidor, nome, email, "senhaForte123", cargo.ID)

	return id, autenticar(t, servidor, email, "senhaForte123").Token
}

func corpoDeCliente(documento, nome, tipo string, extras map[string]any) map[string]any {
	corpo := map[string]any{"documento": documento, "nome": nome, "tipo": tipo, "uf": "PB", "cidade": "João Pessoa"}
	for chave, valor := range extras {
		corpo[chave] = valor
	}

	return corpo
}

func TestComercialGerenciaClientesObrasEGrupos(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	criarUsuario(t, servidor, "Otto Lima", "otto@exemplo.com", "senhaForte123", criarCargo(t, servidor, "Operador", false))
	tokenOperador := autenticar(t, servidor, "otto@exemplo.com", "senhaForte123").Token

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/cliente-grupos", map[string]any{"nome": "Grupo Águia"}, tokenComercial)
	conferirStatus(t, resposta, http.StatusCreated)
	grupo := decodificarEnvelope[clientedto.GrupoResponse](t, resposta).Dados
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/cliente-grupos", map[string]any{"nome": "grupo aguia"}, tokenComercial), http.StatusConflict)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("11222333000181", "Construtora Águia LTDA", "construtora", map[string]any{
		"grupo_id": grupo.ID, "email_nf": "nf@aguia.com.br", "exige_numero_pedido_na_nf": true,
	}), tokenComercial)
	conferirStatus(t, resposta, http.StatusCreated)
	cliente := decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados
	if cliente.DocumentoFormatado != "11.222.333/0001-81" || cliente.TipoPessoa != "pj" || cliente.GrupoID == nil || !cliente.ExigeNumeroPedidoNaNF {
		t.Errorf("cliente criado = %+v", cliente)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("11.222.333/0001-81", "Outro", "loja", nil), tokenComercial)
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "já existe um cliente com este documento" {
		t.Errorf("mensagem do documento repetido = %q", mensagem)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("529.982.247-25", "José", "construtora", nil), tokenComercial), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("33400004000116", "Loja", "loja", nil), tokenOperador), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/clientes?busca=AGUIA", nil, tokenOperador)
	conferirStatus(t, resposta, http.StatusOK)
	if encontrados := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados; len(encontrados) != 1 || encontrados[0].ID != cliente.ID {
		t.Errorf("busca por aguia = %+v", encontrados)
	}

	caminho := "/api/v1/clientes/" + cliente.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, corpoDeCliente("33400004000116", "Outro", "loja", nil), tokenComercial), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{
		"nome": "Águia Engenharia", "tipo": "construtora", "uf": "CE", "cidade": "Fortaleza", "proibe_antecipacao": true,
	}, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if alterado := decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados; alterado.Nome != "Águia Engenharia" || alterado.Documento != "11222333000181" || !alterado.ProibeAntecipacao || alterado.GrupoID != nil {
		t.Errorf("cliente alterado = %+v", alterado)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/obras", map[string]any{
		"nome": "Residencial Mar Azul", "cnpj": "33.400.004/0001-16", "logradouro": "Av. Beira Mar", "numero": "100",
		"cidade": "Cabedelo", "uf": "pb", "cep": "58310-000",
	}, tokenComercial)
	conferirStatus(t, resposta, http.StatusCreated)
	obra := decodificarEnvelope[clientedto.ObraResponse](t, resposta).Dados
	if obra.ClienteID != cliente.ID || *obra.CNPJFormatado != "33.400.004/0001-16" || *obra.CEP != "58310000" || obra.UF != "PB" {
		t.Errorf("obra criada = %+v", obra)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/obras", map[string]any{
		"nome": "RESIDENCIAL MAR AZUL", "logradouro": "Rua", "cidade": "Cabedelo", "uf": "PB",
	}, tokenComercial), http.StatusConflict)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/obras", nil, tokenOperador)
	conferirStatus(t, resposta, http.StatusOK)
	if obras := decodificarPagina[clientedto.ObraResponse](t, resposta).Dados; len(obras) != 1 {
		t.Errorf("obras do cliente = %+v", obras)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, "/api/v1/obras/"+obra.ID.String()+"/desativar", nil, tokenComercial), http.StatusOK)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, caminho, nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminho, nil), http.StatusBadRequest)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{"nome": "José"}, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/representantes", nil, tokenOperador), http.StatusForbidden)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/clientes", nil), http.StatusUnauthorized)
}

func TestRepresentanteComLoginSoVeOsPropriosClientes(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	idAna, tokenAna := usuarioComercial(t, servidor, "Ana Souza", "ana@exemplo.com")
	idBia, _ := usuarioComercial(t, servidor, "Bia Lima", "bia@exemplo.com")

	criarRepresentante := func(nome string, usuarioID uuid.UUID) clientedto.RepresentanteResponse {
		resposta := envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{
			"nome": nome, "usuario_id": usuarioID, "comissao_pronta_entrega_bp": 300, "comissao_importacao_bp": 150,
		})
		conferirStatus(t, resposta, http.StatusCreated)

		return decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados
	}
	ana := criarRepresentante("Ana Representações", idAna)
	bia := criarRepresentante("Bia Vendas", idBia)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{"nome": "Outra", "usuario_id": idAna}), http.StatusConflict)

	criarCliente := func(documento, nome string, representanteID uuid.UUID) clientedto.ClienteResponse {
		resposta := envia(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente(documento, nome, "construtora", map[string]any{"representante_id": representanteID}))
		conferirStatus(t, resposta, http.StatusCreated)

		return decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados
	}
	daAna := criarCliente("11222333000181", "Construtora Águia", ana.ID)
	daBia := criarCliente("33400004000116", "Construtora Ômega", bia.ID)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/clientes/"+daBia.ID.String()+"/obras", map[string]any{
		"nome": "Obra da Bia", "logradouro": "Rua", "cidade": "Recife", "uf": "PE",
	}), http.StatusCreated)

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/clientes", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if clientes := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados; len(clientes) != 1 || clientes[0].ID != daAna.ID {
		t.Errorf("clientes da Ana = %+v, esperado só o dela", clientes)
	}

	caminhoAlheio := "/api/v1/clientes/" + daBia.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminhoAlheio, nil, tokenAna), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminhoAlheio, map[string]any{
		"nome": "Tomado", "tipo": "construtora", "uf": "PB", "cidade": "JP",
	}, tokenAna), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminhoAlheio+"/obras", nil, tokenAna), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/clientes?representante="+bia.ID.String(), nil, tokenAna), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/representantes/"+bia.ID.String(), nil, tokenAna), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/representantes", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if representantes := decodificarPagina[clientedto.RepresentanteResponse](t, resposta).Dados; len(representantes) != 1 || representantes[0].ID != ana.ID {
		t.Errorf("representantes vistos pela Ana = %+v", representantes)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("529.982.247-25", "Maria", "pf", nil), tokenAna)
	conferirStatus(t, resposta, http.StatusCreated)
	if nova := decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados; nova.RepresentanteID == nil || *nova.RepresentanteID != ana.ID {
		t.Errorf("cliente criado pela Ana = %+v, esperado com ela como representante", nova)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("111.444.777-35", "Pedro", "pf", map[string]any{
		"representante_id": bia.ID,
	}), tokenAna), http.StatusForbidden)

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/clientes", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if todos := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados; len(todos) != 3 {
		t.Errorf("administrador listou %d clientes, esperado 3", len(todos))
	}
}
