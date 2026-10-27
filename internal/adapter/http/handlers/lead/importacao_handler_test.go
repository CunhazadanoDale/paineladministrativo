package lead

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	"github.com/google/uuid"
)

type importacaoLeadsFalso struct {
	input    portsin.ImportarLeadsInput
	chamadas int
}

func (f *importacaoLeadsFalso) Importar(_ context.Context, input portsin.ImportarLeadsInput) (*domain.RelatorioImportacao, error) {
	f.input = input
	f.chamadas++
	relatorio, _ := domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
	relatorio.Importadas = len(input.Linhas) - 1
	relatorio.Rejeitar(3, "Loja", "email do lead é inválido")

	return relatorio, nil
}

func TestImportarLeadsRepassaEtapaSimulacaoELinhas(t *testing.T) {
	usecase := &importacaoLeadsFalso{}
	etapaID := uuid.New()
	corpo := "nome;email;telefone;origem\r\nConstru\xe7\xe3o \xc1guia;contato@aguia.com.br;(83) 99999-0000;feira\r\nLoja;x;;\r\n"

	resposta := apoioteste.ExecutarHandler(NewImportacaoHandler(usecase).Importar, http.MethodPost,
		"/api/v1/leads/importacao?etapa_id="+etapaID.String()+"&simular=true", "", corpo)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.input.EtapaID != etapaID || !usecase.input.Simular || len(usecase.input.Linhas) != 2 {
		t.Fatalf("input = %+v", usecase.input)
	}
	if primeira := usecase.input.Linhas[0]; primeira.Numero != 2 || primeira.Nome != "Construção Águia" || primeira.Telefone != "(83) 99999-0000" || primeira.Origem != "feira" {
		t.Errorf("primeira linha = %+v", primeira)
	}

	var envelope dto.Resposta[dto.RelatorioImportacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if !envelope.Dados.Simulacao || envelope.Dados.Total != 2 || len(envelope.Dados.Rejeitadas) != 1 || envelope.Dados.Rejeitadas[0].Chave != "Loja" {
		t.Errorf("relatório = %+v", envelope.Dados)
	}
}

func TestImportarLeadsRecusaConsultaOuArquivoInvalido(t *testing.T) {
	casos := []struct {
		nome    string
		caminho string
		corpo   string
		status  int
	}{
		{"etapa inválida", "/api/v1/leads/importacao?etapa_id=abc", "nome;email;telefone;origem\n", http.StatusBadRequest},
		{"simular inválido", "/api/v1/leads/importacao?simular=sim", "nome;email;telefone;origem\n", http.StatusBadRequest},
		{"cabeçalho errado", "/api/v1/leads/importacao", "nome;email\n", http.StatusBadRequest},
		{"acima de 5MB", "/api/v1/leads/importacao", "nome;email;telefone;origem" + strings.Repeat("\n"+strings.Repeat("a", 190)+";;;", 30000), http.StatusRequestEntityTooLarge},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			usecase := &importacaoLeadsFalso{}
			resposta := apoioteste.ExecutarHandler(NewImportacaoHandler(usecase).Importar, http.MethodPost, caso.caminho, "", caso.corpo)

			apoioteste.VerificarEnvelopeDeErro(t, resposta, caso.status)
			if usecase.chamadas != 0 {
				t.Error("não deveria chegar ao usecase")
			}
		})
	}
}

func TestModeloDeLeadsDevolveCSVComCabecalho(t *testing.T) {
	resposta := apoioteste.ExecutarHandler(NewImportacaoHandler(&importacaoLeadsFalso{}).Modelo, http.MethodGet, "/api/v1/leads/importacao/modelo", "", "")

	if resposta.Code != http.StatusOK || resposta.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("status %d, tipo %q", resposta.Code, resposta.Header().Get("Content-Type"))
	}
	if !strings.Contains(resposta.Header().Get("Content-Disposition"), "modelo-leads.csv") {
		t.Errorf("Content-Disposition = %q", resposta.Header().Get("Content-Disposition"))
	}
	if !strings.HasPrefix(resposta.Body.String(), "\xef\xbb\xbfnome;email;telefone;origem\r\n") {
		t.Errorf("corpo = %q", resposta.Body.String())
	}
}
