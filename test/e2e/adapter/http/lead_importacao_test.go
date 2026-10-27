//go:build e2e

package http

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
)

func TestImportarLeadsPorCSV(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, token := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	etapaID := criarEtapa(t, servidor, criarFunil(t, servidor, "Prospecção"), "Novo")
	criarLead(t, servidor, etapaID, "Lead antigo")
	caminho := "/api/v1/leads/importacao?etapa_id=" + etapaID.String()

	csv := strings.Join([]string{
		"nome;email;telefone;origem",
		"Construtora Águia;contato@aguia.com.br;(83) 99999-0000;feira",
		"Loja Ômega;;83 3222-1111;indicação",
		"Águia repetida;CONTATO@aguia.com.br;;feira",
		";semnome@exemplo.com;;",
		"Construção Beta;beta@exemplo.com;;site",
	}, "\r\n")

	resposta := enviarCSV(t, servidor, caminho+"&simular=true", csv, token)
	conferirStatus(t, resposta, http.StatusOK)
	if simulacao := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; !simulacao.Simulacao || simulacao.Importadas != 3 || len(simulacao.Rejeitadas) != 2 {
		t.Fatalf("simulação = %+v", simulacao)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/etapas/"+etapaID.String()+"/leads", nil, token)
	if leads := decodificarEnvelope[[]leaddto.LeadResponse](t, resposta).Dados; len(leads) != 1 {
		t.Fatalf("simulação gravou leads: %+v", leads)
	}

	resposta = enviarCSV(t, servidor, caminho, csv, token)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados
	if relatorio.Importadas != 3 || len(relatorio.Rejeitadas) != 2 {
		t.Fatalf("relatório = %+v", relatorio)
	}
	if relatorio.Rejeitadas[0].Linha != 4 || relatorio.Rejeitadas[0].Motivo != "email repetido no arquivo (linha 2)" {
		t.Errorf("primeira rejeição = %+v", relatorio.Rejeitadas[0])
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/etapas/"+etapaID.String()+"/leads", nil, token)
	leads := decodificarEnvelope[[]leaddto.LeadResponse](t, resposta).Dados
	nomes := map[string]bool{}
	for _, lead := range leads {
		nomes[lead.Nome] = true
	}
	if len(leads) != 4 || !nomes["Construção Beta"] || !nomes["Loja Ômega"] {
		t.Errorf("leads da etapa = %+v", leads)
	}

	resposta = enviarCSV(t, servidor, caminho, "nome;email;telefone;origem\nOutra;;(83)99999-0000;\n", token)
	conferirStatus(t, resposta, http.StatusOK)
	if novamente := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; novamente.Importadas != 0 || novamente.Rejeitadas[0].Motivo != "já existe um lead com este telefone" {
		t.Errorf("reimportação = %+v", novamente)
	}

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/leads/importacao", csv, token), http.StatusBadRequest)
	conferirStatus(t, enviarCSV(t, servidor, caminho, "nome;email\n", token), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/leads/importacao/modelo", nil, token)
	conferirStatus(t, resposta, http.StatusOK)
	modelo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatalf("leitura do modelo falhou: %v", err)
	}
	if !strings.HasPrefix(string(modelo), "\xef\xbb\xbfnome;email;telefone;origem\r\n") {
		t.Errorf("modelo = %q", modelo)
	}
}
