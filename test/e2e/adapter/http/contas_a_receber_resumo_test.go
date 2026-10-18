//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
)

func TestResumoEBoletosPendentesDeContasAReceber(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMia := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	hoje := domainreceber.Hoje(time.Now().UTC())

	corpo := c.corpo(c.mia, "1")
	corpo["vencimento"] = hoje.AddDate(0, 0, 3).Format("2006-01-02")
	corpo["conta_bancaria_id"] = c.contaMia
	emBanco := c.criar(t, corpo, tokenMia)
	corpo = c.corpo(c.mia, "2")
	corpo["vencimento"] = hoje.AddDate(0, 0, 10).Format("2006-01-02")
	c.criar(t, corpo, tokenMia)
	c.criar(t, c.corpo(c.red, "1"), tokenRed)

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/boletos-pendentes", nil, tokenMia)
	conferirStatus(t, resposta, http.StatusOK)
	if pendentes := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(pendentes) != 1 || pendentes[0].ID != emBanco.ID {
		t.Fatalf("boletos pendentes = %+v", pendentes)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/boletos-pendentes?dias=91", nil, tokenMia),
		http.StatusBadRequest, "erro de validação: dias do alerta de boleto devem estar entre 0 e 90")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/boletos-pendentes?dias=x", nil, tokenMia), http.StatusBadRequest)

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber/"+emBanco.ID.String()+"/boleto-enviado", map[string]any{
		"versao": 1, "canal": "email", "enviado_em": time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}, tokenMia), http.StatusOK)
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/boletos-pendentes", nil, tokenMia)
	if pendentes := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(pendentes) != 0 {
		t.Fatalf("o boleto enviado sai do alerta: %+v", pendentes)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/contas-a-receber/resumo", nil, tokenMia)
	conferirStatus(t, resposta, http.StatusOK)
	resumo := decodificarEnvelope[receberdto.ResumoContasAReceberResponse](t, resposta).Dados
	if resumo.Totais.AReceber.Quantidade != 2 || resumo.Totais.AReceber.SaldoCentavos != 300000 || resumo.Totais.Vencido.Quantidade != 0 {
		t.Fatalf("resumo da MIA = %+v", resumo)
	}
	for _, linha := range resumo.Semanas {
		if linha.EmpresaID != c.mia {
			t.Fatalf("o resumo da MIA não mostra outra empresa: %+v", linha)
		}
	}
	if inicio, err := time.Parse("2006-01-02", resumo.De); err != nil || inicio.Weekday() != time.Monday {
		t.Fatalf("o período começa na segunda: %q", resumo.De)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/contas-a-receber/resumo?empresa="+c.mia.String(), nil, tokenRed), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/contas-a-receber/resumo?de=2026-10-20&ate=2026-10-01", nil, tokenMia),
		http.StatusBadRequest, "erro de validação: fim do resumo não pode ser anterior ao início")

	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/contas-a-receber/resumo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if todos := decodificarEnvelope[receberdto.ResumoContasAReceberResponse](t, resposta).Dados; todos.Totais.AReceber.Quantidade != 3 {
		t.Fatalf("resumo do administrador = %+v", todos.Totais)
	}
}
