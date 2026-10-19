//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	tabelaprecodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/tabelapreco"
)

func TestTabelaDePrecosComVigencia(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	produtoID := produtoDoTeste(t, servidor)
	if _, err := banco.Exec(`UPDATE produto SET preco = 9900 WHERE id = $1`, produtoID); err != nil {
		t.Fatalf("definir preço do produto: %v", err)
	}
	_, tokenComercial := usuarioComercial(t, servidor, "Caio Lima", "caio@exemplo.com")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabi", "fabi@exemplo.com")
	tokenSemPerfil := tokenComPerfil(t, servidor, "Operador", "Otto", "otto@exemplo.com", false, false)

	brasilia := time.FixedZone("BRT", -3*60*60)
	hoje := time.Now().In(brasilia)
	hoje = time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)
	ontem := hoje.AddDate(0, 0, -1).Format("2006-01-02")
	daquiAUmMes := hoje.AddDate(0, 1, 0).Format("2006-01-02")

	vigentes := func(token, consulta string) []tabelaprecodto.PrecoVigenteResponse {
		t.Helper()
		resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/tabela-precos/vigentes?"+consulta, nil, token)
		conferirStatus(t, resposta, http.StatusOK)
		return decodificarPagina[tabelaprecodto.PrecoVigenteResponse](t, resposta).Dados
	}

	semTabela := vigentes(tokenComercial, "modelo=pronta_entrega&produto="+produtoID.String())
	if len(semTabela) != 1 || semTabela[0].Origem != "produto" || *semTabela[0].PrecoCentavos != 9900 {
		t.Fatalf("sem tabela = %+v", semTabela)
	}

	corpo := map[string]any{"produto_id": produtoID, "modelo": "pronta_entrega", "preco_centavos": 12000, "vigente_de": ontem, "observacao": "tabela de outubro"}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/tabela-precos", corpo, tokenComercial), http.StatusForbidden)
	resposta := envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", corpo)
	conferirStatus(t, resposta, http.StatusCreated)
	vigorando := decodificarEnvelope[tabelaprecodto.ItemResponse](t, resposta).Dados
	if vigorando.VigenteDe != ontem || vigorando.PrecoCentavos != 12000 || vigorando.Modelo != "pronta_entrega" {
		t.Fatalf("preço lançado = %+v", vigorando)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", corpo), http.StatusConflict)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", map[string]any{
		"produto_id": produtoID, "modelo": "consignado", "preco_centavos": 1, "vigente_de": ontem,
	}), http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", map[string]any{
		"produto_id": produtoID, "modelo": "pronta_entrega", "preco_centavos": 13000, "vigente_de": daquiAUmMes,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	futuro := decodificarEnvelope[tabelaprecodto.ItemResponse](t, resposta).Dados

	hojeVigente := vigentes(tokenFinanceiro, "modelo=pronta_entrega&produto="+produtoID.String())
	if len(hojeVigente) != 1 || hojeVigente[0].Origem != "tabela" || *hojeVigente[0].PrecoCentavos != 12000 || *hojeVigente[0].ItemID != vigorando.ID {
		t.Fatalf("vigente hoje = %+v", hojeVigente)
	}
	noFuturo := vigentes(tokenComercial, "modelo=pronta_entrega&produto="+produtoID.String()+"&data="+daquiAUmMes)
	if *noFuturo[0].PrecoCentavos != 13000 || noFuturo[0].Data != daquiAUmMes {
		t.Fatalf("vigente daqui a um mês = %+v", noFuturo)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/tabela-precos/vigentes?modelo=pronta_entrega", nil, tokenSemPerfil), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/tabela-precos/vigentes", nil, tokenComercial), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/tabela-precos?produto="+produtoID.String(), nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if historico := decodificarPagina[tabelaprecodto.ItemResponse](t, resposta).Dados; len(historico) != 2 || historico[0].ID != futuro.ID {
		t.Fatalf("histórico = %+v", historico)
	}

	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/tabela-precos/"+vigorando.ID.String(), nil), http.StatusConflict)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, "/api/v1/tabela-precos/"+futuro.ID.String(), nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/tabela-precos/"+futuro.ID.String(), nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/tabela-precos/"+futuro.ID.String(), nil), http.StatusNotFound)
}
