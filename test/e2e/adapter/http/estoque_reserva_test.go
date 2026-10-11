//go:build e2e

package http

import (
	"net/http"
	"testing"

	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/google/uuid"
)

func TestReservasSeparamODisponivelEBaixamComASaida(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	produtoID := produtoDoTeste(t, servidor)
	caminho := "/api/v1/produtos/" + produtoID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 5, "custo_unitario_centavos": 1990,
	}), http.StatusCreated)

	vitrine := func() int {
		resposta := envia(t, servidor, http.MethodGet, "/api/v1/publico/produtos", nil)
		conferirStatus(t, resposta, http.StatusOK)

		return len(decodificarPagina[estoquedto.ProdutoPublicoResponse](t, resposta).Dados)
	}
	if itens := vitrine(); itens != 1 {
		t.Fatalf("vitrine antes da reserva = %d produtos, esperado 1", itens)
	}

	reservar := func(quantidade int) *http.Response {
		return envia(t, servidor, http.MethodPost, caminho+"/reservas", map[string]any{
			"local_id": galpao.ID, "quantidade": quantidade, "observacao": "pedido 123",
		})
	}
	conferirStatus(t, reservar(6), http.StatusBadRequest)

	resposta := reservar(3)
	conferirStatus(t, resposta, http.StatusCreated)
	paraBaixar := decodificarEnvelope[estoquedto.ReservaResponse](t, resposta).Dados
	if paraBaixar.Status != "ativa" || paraBaixar.Quantidade != 3 || paraBaixar.Origem != "manual" || paraBaixar.Observacao != "pedido 123" {
		t.Errorf("reserva criada = %+v", paraBaixar)
	}
	resposta = reservar(2)
	conferirStatus(t, resposta, http.StatusCreated)
	paraLiberar := decodificarEnvelope[estoquedto.ReservaResponse](t, resposta).Dados

	resposta = envia(t, servidor, http.MethodGet, caminho+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados; saldo.Saldo != 5 || saldo.Reservado != 5 || saldo.Disponivel != 0 {
		t.Errorf("saldo com tudo reservado = %+v", saldo)
	}
	if itens := vitrine(); itens != 0 {
		t.Errorf("vitrine com tudo reservado = %d produtos, esperado nenhum", itens)
	}

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "saida", "quantidade": 1,
	}), http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "saida", "quantidade": 3, "reserva_id": paraBaixar.ID,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	if baixa := decodificarEnvelope[estoquedto.MovimentoResponse](t, resposta).Dados; baixa.ReservaID == nil || *baixa.ReservaID != paraBaixar.ID {
		t.Errorf("movimento da baixa = %+v", baixa)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "saida", "quantidade": 3, "reserva_id": paraBaixar.ID,
	}), http.StatusConflict)

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/reservas/"+paraLiberar.ID.String()+"/liberar", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if liberada := decodificarEnvelope[estoquedto.ReservaResponse](t, resposta).Dados; liberada.Status != "liberada" || liberada.EncerradaEm == nil {
		t.Errorf("reserva liberada = %+v", liberada)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/reservas/"+paraLiberar.ID.String()+"/liberar", nil), http.StatusConflict)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/reservas/"+uuid.NewString()+"/liberar", nil), http.StatusNotFound)

	resposta = envia(t, servidor, http.MethodGet, caminho+"/reservas?status=ativa", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if ativas := decodificarPagina[estoquedto.ReservaResponse](t, resposta).Dados; len(ativas) != 0 {
		t.Errorf("reservas ativas = %+v, esperado nenhuma", ativas)
	}
	if itens := vitrine(); itens != 1 {
		t.Errorf("vitrine depois da liberação = %d produtos, esperado 1", itens)
	}
}

func TestReservasExigemAdministradorEAcesso(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	deposito := criarLocal(t, servidor, red, "Depósito")
	produtoID := produtoDoTeste(t, servidor)
	caminho := "/api/v1/produtos/" + produtoID.String()
	for _, local := range []uuid.UUID{galpao.ID, deposito.ID} {
		conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
			"local_id": local, "tipo": "entrada", "quantidade": 5, "custo_unitario_centavos": 1990,
		}), http.StatusCreated)
		conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/reservas", map[string]any{
			"local_id": local, "quantidade": 1,
		}), http.StatusCreated)
	}

	cargo := criarCargo(t, servidor, "Operador", false)
	idAna := criarUsuario(t, servidor, "Ana Souza", "ana@exemplo.com", "senhaForte123", cargo)
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idAna.String()+"/empresas", map[string]any{
		"empresa_ids": []uuid.UUID{mia},
	}), http.StatusOK)
	token := autenticar(t, servidor, "ana@exemplo.com", "senhaForte123").Token

	resposta := enviaComToken(t, servidor, http.MethodGet, caminho+"/reservas", nil, token)
	conferirStatus(t, resposta, http.StatusOK)
	if reservas := decodificarPagina[estoquedto.ReservaResponse](t, resposta).Dados; len(reservas) != 1 || reservas[0].LocalID != galpao.ID {
		t.Errorf("reservas da Ana = %+v, esperado só a do galpão da MIA", reservas)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/reservas?local="+deposito.ID.String(), nil, token), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/reservas", map[string]any{
		"local_id": galpao.ID, "quantidade": 1,
	}, token), http.StatusForbidden)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, caminho+"/reservas", nil), http.StatusUnauthorized)
}
