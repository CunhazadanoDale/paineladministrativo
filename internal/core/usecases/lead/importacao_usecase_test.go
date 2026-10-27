package lead_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	leadusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/lead"
	"github.com/google/uuid"
)

func linhaDeLead(numero int, nome, email, telefone string) portsin.LinhaImportacao {
	return portsin.LinhaImportacao{Numero: numero, Nome: nome, Email: email, Telefone: telefone, Origem: "feira"}
}

func TestImportarLeadsGravaValidasERejeitaDuplicadas(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)
	existente := uuid.New()
	c.leads.itens[existente] = &domainlead.Lead{ID: existente, Nome: "Já cadastrado", Email: "antigo@exemplo.com", Telefone: "(83) 98888-7777", Ativo: true, EtapaID: etapaID}
	usecase := leadusecases.NewLeadImportacaoUsecase(c.leads, c.etapas)

	relatorio, err := usecase.Importar(c.ctx, portsin.ImportarLeadsInput{
		EtapaID: etapaID,
		Linhas: []portsin.LinhaImportacao{
			linhaDeLead(2, " Construtora Águia ", "Contato@Aguia.com.br", "(83) 99999-0000"),
			linhaDeLead(3, "Loja Sem Contato", "", ""),
			linhaDeLead(4, "Águia de novo", "contato@aguia.com.br", ""),
			linhaDeLead(5, "Mesmo telefone", "", "83 99999 0000"),
			linhaDeLead(6, "Email antigo", "ANTIGO@exemplo.com", ""),
			linhaDeLead(7, "Telefone antigo", "", "83988887777"),
			linhaDeLead(8, "", "semnome@exemplo.com", ""),
			linhaDeLead(9, "Email ruim", "nao-e-email", ""),
		},
	})

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if relatorio.Total != 8 || relatorio.Importadas != 2 || relatorio.Simulacao {
		t.Errorf("relatório = %+v", relatorio)
	}
	esperados := map[int]string{
		4: "email repetido no arquivo (linha 2)",
		5: "telefone repetido no arquivo (linha 2)",
		6: "já existe um lead com este email",
		7: "já existe um lead com este telefone",
		8: "nome do lead é obrigatório",
		9: "email do lead é inválido",
	}
	if len(relatorio.Rejeitadas) != len(esperados) {
		t.Fatalf("rejeitadas = %+v", relatorio.Rejeitadas)
	}
	for _, rejeitada := range relatorio.Rejeitadas {
		if esperados[rejeitada.Linha] != rejeitada.Motivo {
			t.Errorf("linha %d: motivo %q, esperado %q", rejeitada.Linha, rejeitada.Motivo, esperados[rejeitada.Linha])
		}
	}

	if len(c.leads.itens) != 3 {
		t.Fatalf("leads gravados = %d, esperado 3", len(c.leads.itens))
	}
	for id, item := range c.leads.itens {
		if id == existente {
			continue
		}
		if !item.Ativo || item.EtapaID != etapaID || item.Origem != "feira" || item.CriadoEm.IsZero() {
			t.Errorf("lead gravado = %+v", item)
		}
		if item.Nome == "Construtora Águia" && item.Email != "contato@aguia.com.br" {
			t.Errorf("email não foi normalizado: %+v", item)
		}
	}
}

func TestImportarLeadsSimulandoNaoGrava(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)

	relatorio, err := leadusecases.NewLeadImportacaoUsecase(c.leads, c.etapas).Importar(c.ctx, portsin.ImportarLeadsInput{
		EtapaID: etapaID,
		Simular: true,
		Linhas:  []portsin.LinhaImportacao{linhaDeLead(2, "Ana", "ana@exemplo.com", ""), linhaDeLead(3, "Ana 2", "ana@exemplo.com", "")},
	})

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !relatorio.Simulacao || relatorio.Importadas != 1 || len(relatorio.Rejeitadas) != 1 {
		t.Errorf("relatório = %+v", relatorio)
	}
	if len(c.leads.itens) != 0 {
		t.Errorf("simulação gravou %d leads", len(c.leads.itens))
	}
}

func TestImportarLeadsExigeEtapaAtiva(t *testing.T) {
	c := novoCenario()
	inativa := c.novaEtapa(c.funilID, false)
	usecase := leadusecases.NewLeadImportacaoUsecase(c.leads, c.etapas)

	casos := map[string]uuid.UUID{"sem etapa": uuid.Nil, "etapa inexistente": uuid.New(), "etapa inativa": inativa}
	for nome, etapaID := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := usecase.Importar(c.ctx, portsin.ImportarLeadsInput{EtapaID: etapaID, Linhas: []portsin.LinhaImportacao{linhaDeLead(2, "Ana", "", "")}})

			if !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}
}

func TestImportarLeadsRecusaMaisDe5000Linhas(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)

	_, err := leadusecases.NewLeadImportacaoUsecase(c.leads, c.etapas).Importar(c.ctx, portsin.ImportarLeadsInput{
		EtapaID: etapaID,
		Linhas:  make([]portsin.LinhaImportacao, domain.LimiteDeLinhasDaImportacao+1),
	})

	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro = %v, esperado validação", err)
	}
}
