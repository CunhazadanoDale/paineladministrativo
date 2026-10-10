//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	empresadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/empresa"
	"github.com/google/uuid"
)

func criarEmpresa(t *testing.T, servidor *httptest.Server, cnpj, nomeFantasia string) empresadto.EmpresaResponse {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/empresas", map[string]string{
		"cnpj":              cnpj,
		"razao_social":      nomeFantasia + " LTDA",
		"nome_fantasia":     nomeFantasia,
		"uf":                "pb",
		"regime_tributario": "presumido",
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[empresadto.EmpresaResponse](t, resposta).Dados
}

func TestAdministradorGerenciaEmpresas(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	if mia.CNPJ != "11222333000181" || mia.CNPJFormatado != "11.222.333/0001-81" || mia.UF != "PB" || !mia.Ativa {
		t.Errorf("empresa criada %+v, esperado CNPJ limpo e formatado, UF PB e ativa", mia)
	}
	red := criarEmpresa(t, servidor, "33400004000116", "Red Diamond")

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/empresas", map[string]string{
		"cnpj": "11222333000181", "razao_social": "Outra", "nome_fantasia": "Outra", "uf": "SP", "regime_tributario": "real",
	})
	conferirStatus(t, resposta, http.StatusConflict)

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/empresas", map[string]string{
		"cnpj": "11222333000182", "razao_social": "Outra", "nome_fantasia": "Outra", "uf": "SP", "regime_tributario": "real",
	})
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPut, "/api/v1/empresas/"+mia.ID.String(), map[string]string{
		"razao_social": "MIA Construtora LTDA", "nome_fantasia": "Gigante PB", "uf": "PB", "regime_tributario": "real",
	})
	conferirStatus(t, resposta, http.StatusOK)
	if alterada := decodificarEnvelope[empresadto.EmpresaResponse](t, resposta).Dados; alterada.NomeFantasia != "Gigante PB" || alterada.CNPJ != mia.CNPJ {
		t.Errorf("empresa alterada %+v, esperado nome novo e mesmo CNPJ", alterada)
	}

	resposta = envia(t, servidor, http.MethodPatch, "/api/v1/empresas/"+red.ID.String()+"/desativar", nil)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/empresas?ativas=true", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if ativas := decodificarPagina[empresadto.EmpresaResponse](t, resposta).Dados; len(ativas) != 1 || ativas[0].ID != mia.ID {
		t.Errorf("empresas ativas %+v, esperado só a MIA", ativas)
	}

	resposta = envia(t, servidor, http.MethodDelete, "/api/v1/empresas/"+red.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/empresas/"+red.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusNotFound)
}

func TestUsuarioSoEnxergaEmpresasDoSeuAcesso(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond")

	cargoID := criarCargo(t, servidor, "Comprador", false)
	usuarioID := criarUsuario(t, servidor, "Lia Prado", "lia@exemplo.com", "senhaForte123", cargoID)
	token := autenticar(t, servidor, "lia@exemplo.com", "senhaForte123").Token

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/empresas", nil, token)
	conferirStatus(t, resposta, http.StatusOK)
	if empresas := decodificarPagina[empresadto.EmpresaResponse](t, resposta).Dados; len(empresas) != 0 {
		t.Errorf("usuário novo listou %d empresas, esperado nenhuma", len(empresas))
	}

	caminhoAcesso := "/api/v1/usuarios/" + usuarioID.String() + "/empresas"
	resposta = envia(t, servidor, http.MethodPut, caminhoAcesso, map[string]any{
		"acesso_todas": false,
		"empresa_ids":  []uuid.UUID{mia.ID, mia.ID},
	})
	conferirStatus(t, resposta, http.StatusOK)
	if vinculos := decodificarEnvelope[empresadto.VinculosResponse](t, resposta).Dados; vinculos.AcessoTodas || len(vinculos.EmpresaIDs) != 1 || vinculos.EmpresaIDs[0] != mia.ID {
		t.Errorf("vínculos %+v, esperado só a MIA", vinculos)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/empresas", nil, token)
	conferirStatus(t, resposta, http.StatusOK)
	if empresas := decodificarPagina[empresadto.EmpresaResponse](t, resposta).Dados; len(empresas) != 1 || empresas[0].ID != mia.ID {
		t.Errorf("usuário listou %+v, esperado só a MIA", empresas)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/empresas/"+mia.ID.String(), nil, token)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/empresas/"+red.ID.String(), nil, token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/empresas", map[string]string{
		"cnpj": "44555666000181", "razao_social": "X", "nome_fantasia": "X", "uf": "CE", "regime_tributario": "simples",
	}, token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminhoAcesso, nil, token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = envia(t, servidor, http.MethodDelete, "/api/v1/empresas/"+mia.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPut, caminhoAcesso, map[string]any{"acesso_todas": true, "empresa_ids": []uuid.UUID{}})
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/empresas", nil, token)
	conferirStatus(t, resposta, http.StatusOK)
	if empresas := decodificarPagina[empresadto.EmpresaResponse](t, resposta).Dados; len(empresas) != 2 {
		t.Errorf("usuário com acesso total listou %d empresas, esperado 2", len(empresas))
	}
}

func TestVinculosDeUsuarioInexistenteOuComEmpresaInvalida(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/usuarios/"+uuid.NewString()+"/empresas", nil)
	conferirStatus(t, resposta, http.StatusNotFound)

	cargoID := criarCargo(t, servidor, "Comprador", false)
	usuarioID := criarUsuario(t, servidor, "Lia Prado", "lia@exemplo.com", "senhaForte123", cargoID)

	resposta = envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+usuarioID.String()+"/empresas", map[string]any{
		"empresa_ids": []uuid.UUID{uuid.New()},
	})
	conferirStatus(t, resposta, http.StatusBadRequest)
}
