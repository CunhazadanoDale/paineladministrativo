//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
)

func TestSolicitanteEditaPendenteEFinanceiroAdiaAprovada(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "edicao")

	criada := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	caminho := "/api/v1/solicitacoes/" + criada.ID.String()

	vencimentoEditado := time.Now().UTC().AddDate(0, 0, 10)
	edicao := map[string]any{
		"valor_centavos":  64000,
		"prazo_pagamento": vencimentoEditado.Format("2006-01-02"),
		"observacao":      "Compra de cimento e areia",
		"forma_pagamento": "transferencia",
		"estimado":        true,
		"versao":          1,
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho, edicao, tokenAprovador),
		http.StatusForbidden, "perfil sem permissão para editar esta solicitação")

	respostaEdicao := enviaComToken(t, servidor, http.MethodPut, caminho, edicao, tokenSolicitante)
	conferirStatus(t, respostaEdicao, http.StatusOK)
	editada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaEdicao).Dados
	if editada.Versao != 2 || editada.ValorCentavos != 64000 || editada.FormaPagamento != "transferencia" || !editada.Estimado {
		t.Errorf("editada = versão %d, valor %d, forma %q, estimado %v", editada.Versao, editada.ValorCentavos, editada.FormaPagamento, editada.Estimado)
	}
	if editada.Competencia != vencimentoEditado.Format("2006-01") || editada.Status != "pendente_aprovacao" {
		t.Errorf("competência = %q, status = %q", editada.Competencia, editada.Status)
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho, edicao, tokenSolicitante),
		http.StatusConflict, "solicitação alterada por outra operação, recarregue e tente novamente")

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", map[string]any{"versao": 1}, tokenAprovador),
		http.StatusConflict, "solicitação alterada por outra operação, recarregue e tente novamente")

	respostaAprovacao := enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", map[string]any{"versao": 2}, tokenAprovador)
	conferirStatus(t, respostaAprovacao, http.StatusOK)

	edicao["versao"] = 3
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho, edicao, tokenSolicitante),
		http.StatusConflict, "só é possível editar solicitação pendente de aprovação")

	novoVencimento := vencimentoEditado.AddDate(0, 0, 20)
	adiamento := map[string]any{
		"vencimento": novoVencimento.Format("2006-01-02"),
		"motivo":     "fornecedor aceitou o novo prazo",
		"versao":     3,
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/adiar", adiamento, tokenSolicitante),
		http.StatusForbidden, "perfil sem permissão para adiar solicitação")

	cedo := map[string]any{"vencimento": time.Now().UTC().AddDate(0, 0, 2).Format("2006-01-02"), "motivo": "x", "versao": 3}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/adiar", cedo, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: novo vencimento deve ser posterior ao atual")

	respostaAdiamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/adiar", adiamento, tokenFinanceiro)
	conferirStatus(t, respostaAdiamento, http.StatusOK)
	adiada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaAdiamento).Dados
	if adiada.Versao != 4 || adiada.PrazoPagamento.Format("2006-01-02") != novoVencimento.Format("2006-01-02") {
		t.Errorf("adiada = versão %d, vencimento %s", adiada.Versao, adiada.PrazoPagamento)
	}
	if adiada.Competencia != editada.Competencia || adiada.Status != "aprovado" {
		t.Errorf("adiamento mudou competência ou status: %q, %q", adiada.Competencia, adiada.Status)
	}

	respostaHistorico := enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, respostaHistorico, http.StatusOK)
	historico := decodificarEnvelope[[]solicitacaodto.HistoricoResponse](t, respostaHistorico).Dados
	eventos := make([]string, 0, len(historico))
	for _, registro := range historico {
		eventos = append(eventos, registro.Evento)
	}
	if strings.Join(eventos, ",") != "adiada,aprovada,editada,criada" {
		t.Fatalf("eventos = %v", eventos)
	}
	esperada := "vencimento adiado de " + vencimentoEditado.Format("2006-01-02") + " para " + novoVencimento.Format("2006-01-02") +
		": fornecedor aceitou o novo prazo"
	if historico[0].Descricao != esperada {
		t.Errorf("descrição = %q, esperado %q", historico[0].Descricao, esperada)
	}
	if !strings.HasPrefix(historico[2].Descricao, "solicitação editada: valor, vencimento, observação, forma de pagamento") {
		t.Errorf("descrição da edição = %q", historico[2].Descricao)
	}
}
