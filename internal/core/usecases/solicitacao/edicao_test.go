package solicitacao_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func entradaEdicao(id, usuarioID uuid.UUID) portsin.EditarSolicitacaoInput {
	return portsin.EditarSolicitacaoInput{
		SolicitacaoID:  id,
		UsuarioID:      usuarioID,
		ValorCentavos:  25000,
		PrazoPagamento: time.Date(2099, time.April, 15, 0, 0, 0, 0, time.UTC),
		Observacao:     "conta de energia",
		FormaPagamento: "debito_automatico",
		Estimado:       true,
		Versao:         1,
	}
}

func (c *cenario) ultimoEvento(t *testing.T, id, usuarioID uuid.UUID) *domainsolicitacao.Historico {
	t.Helper()

	historico, err := c.solicitacao.ListarHistorico(c.ctx, id, usuarioID)
	if err != nil || len(historico) == 0 {
		t.Fatalf("histórico = %v, %v", historico, err)
	}

	return historico[0]
}

func TestEditarSolicitacaoPendentePeloSolicitante(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	id := c.criarSolicitacao(t, solicitante, 10000)

	entrada := entradaEdicao(id, solicitante)
	entrada.CategoriaFinanceiraID = &categoriaID
	if err := c.solicitacao.Editar(c.ctx, entrada); err != nil {
		t.Fatalf("edição falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Versao != 2 || salva.Valor.Centavos() != 25000 || salva.Prazo.String() != "2099-04-15" || !salva.Estimado {
		t.Errorf("salva = versão %d, valor %d, vencimento %s, estimado %v", salva.Versao, salva.Valor.Centavos(), salva.Prazo, salva.Estimado)
	}
	if salva.Competencia.String() != "2099-04" || salva.FormaPagamento != domainsolicitacao.FormaPagamentoDebitoAutomatico {
		t.Errorf("competência = %s, forma = %q", salva.Competencia, salva.FormaPagamento)
	}
	if salva.CategoriaFinanceiraID == nil || *salva.CategoriaFinanceiraID != categoriaID {
		t.Errorf("categoria = %v", salva.CategoriaFinanceiraID)
	}

	evento := c.ultimoEvento(t, id, solicitante)
	if evento.Evento != domainsolicitacao.EventoEditada || evento.ParaStatus != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("último registro = %+v, esperado edição sem troca de status", evento)
	}
	if !strings.HasPrefix(evento.Descricao, "solicitação editada: valor, vencimento, observação") {
		t.Errorf("descrição = %q", evento.Descricao)
	}
}

func TestEditarSolicitacaoSemMudancaNaoGrava(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	id := c.criarSolicitacao(t, solicitante, 10000)
	original := c.solicitacoesRepo.itens[id]

	entrada := entradaEdicao(id, solicitante)
	entrada.ValorCentavos = original.Valor.Centavos()
	entrada.PrazoPagamento = original.Prazo.Data()
	entrada.Observacao = original.Observacao.Texto()
	entrada.FormaPagamento = original.FormaPagamento.String()
	entrada.Estimado = false

	if err := c.solicitacao.Editar(c.ctx, entrada); err != nil {
		t.Fatalf("edição sem mudança falhou: %v", err)
	}
	if salva := c.solicitacoesRepo.itens[id]; salva.Versao != 1 {
		t.Errorf("versão = %d, esperado 1", salva.Versao)
	}
	if evento := c.ultimoEvento(t, id, solicitante); evento.Evento != domainsolicitacao.EventoCriada {
		t.Errorf("edição sem mudança gravou histórico: %+v", evento)
	}
}

func TestEditarSolicitacaoRecusaPerfilVersaoStatusEDados(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	administrador := c.novoUsuario("Admin", true, false)
	terceiro := c.novoUsuario("Bia", false, true)
	receita := c.novaCategoria(domainfinanceiro.NaturezaReceita, true, true)

	pendente := c.criarSolicitacao(t, solicitante, 10000)
	aprovada := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, aprovada)

	casos := map[string]struct {
		ajustar  func(*portsin.EditarSolicitacaoInput)
		esperado error
	}{
		"terceiro, mesmo do financeiro": {func(e *portsin.EditarSolicitacaoInput) { e.UsuarioID = terceiro }, domain.ErrPermissao},
		"inexistente":                   {func(e *portsin.EditarSolicitacaoInput) { e.SolicitacaoID = uuid.New() }, domain.ErrNotFound},
		"versão ausente":                {func(e *portsin.EditarSolicitacaoInput) { e.Versao = 0 }, domain.ErrValidacao},
		"versão divergente":             {func(e *portsin.EditarSolicitacaoInput) { e.Versao = 5 }, domain.ErrConflito},
		"aprovada":                      {func(e *portsin.EditarSolicitacaoInput) { e.SolicitacaoID = aprovada; e.Versao = 2 }, domain.ErrConflito},
		"valor zerado":                  {func(e *portsin.EditarSolicitacaoInput) { e.ValorCentavos = 0 }, domain.ErrValidacao},
		"vencimento passado":            {func(e *portsin.EditarSolicitacaoInput) { e.PrazoPagamento = time.Now().AddDate(0, 0, -1) }, domain.ErrValidacao},
		"observação vazia":              {func(e *portsin.EditarSolicitacaoInput) { e.Observacao = " " }, domain.ErrValidacao},
		"forma inválida":                {func(e *portsin.EditarSolicitacaoInput) { e.FormaPagamento = "dinheiro" }, domain.ErrValidacao},
		"competência inválida":          {func(e *portsin.EditarSolicitacaoInput) { e.Competencia = "abril" }, domain.ErrValidacao},
		"categoria de receita":          {func(e *portsin.EditarSolicitacaoInput) { e.CategoriaFinanceiraID = &receita }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		entrada := entradaEdicao(pendente, solicitante)
		caso.ajustar(&entrada)

		if err := c.solicitacao.Editar(c.ctx, entrada); !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}

	if salva := c.solicitacoesRepo.itens[pendente]; salva.Versao != 1 {
		t.Errorf("tentativas recusadas alteraram a solicitação: versão %d", salva.Versao)
	}

	if err := c.solicitacao.Editar(c.ctx, entradaEdicao(pendente, administrador)); err != nil {
		t.Errorf("administrador = %v, esperado edição permitida", err)
	}
}

func TestAdiarSolicitacaoAprovadaPeloFinanceiro(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Fábio", false, true)
	id := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, id)
	vencimentoAnterior := c.solicitacoesRepo.itens[id].Prazo.String()
	competencia := c.solicitacoesRepo.itens[id].Competencia

	novoVencimento := time.Now().UTC().AddDate(0, 1, 0)
	err := c.solicitacao.Adiar(c.ctx, portsin.AdiarSolicitacaoInput{
		SolicitacaoID: id,
		UsuarioID:     financeiro,
		Vencimento:    novoVencimento,
		Motivo:        "fornecedor aceitou o novo prazo",
		Versao:        2,
	})
	if err != nil {
		t.Fatalf("adiamento falhou: %v", err)
	}

	salva := c.solicitacoesRepo.itens[id]
	if salva.Prazo.String() != novoVencimento.Format("2006-01-02") || salva.Versao != 3 || salva.Competencia != competencia {
		t.Errorf("salva = vencimento %s, versão %d, competência %s", salva.Prazo, salva.Versao, salva.Competencia)
	}

	evento := c.ultimoEvento(t, id, financeiro)
	esperada := "vencimento adiado de " + vencimentoAnterior + " para " + novoVencimento.Format("2006-01-02") + ": fornecedor aceitou o novo prazo"
	if evento.Evento != domainsolicitacao.EventoAdiada || evento.Descricao != esperada || evento.UsuarioID != financeiro {
		t.Errorf("último registro = %+v, esperado %q", evento, esperada)
	}
}

func TestAdiarSolicitacaoRecusaPerfilAcessoVersaoStatusEDados(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Fábio", false, true)
	outraEmpresa := c.novaEmpresa(true)
	financeiroDeOutra := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroDeOutra, outraEmpresa)

	aprovada := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, aprovada)
	pendente := c.criarSolicitacao(t, solicitante, 10000)

	valida := func() portsin.AdiarSolicitacaoInput {
		return portsin.AdiarSolicitacaoInput{
			SolicitacaoID: aprovada,
			UsuarioID:     financeiro,
			Vencimento:    time.Now().UTC().AddDate(0, 1, 0),
			Motivo:        "acordo com o fornecedor",
			Versao:        2,
		}
	}

	casos := map[string]struct {
		ajustar  func(*portsin.AdiarSolicitacaoInput)
		esperado error
	}{
		"solicitante":            {func(e *portsin.AdiarSolicitacaoInput) { e.UsuarioID = solicitante }, domain.ErrPermissao},
		"empresa fora do acesso": {func(e *portsin.AdiarSolicitacaoInput) { e.UsuarioID = financeiroDeOutra }, domain.ErrPermissao},
		"versão divergente":      {func(e *portsin.AdiarSolicitacaoInput) { e.Versao = 1 }, domain.ErrConflito},
		"pendente":               {func(e *portsin.AdiarSolicitacaoInput) { e.SolicitacaoID = pendente; e.Versao = 1 }, domain.ErrConflito},
		"sem motivo":             {func(e *portsin.AdiarSolicitacaoInput) { e.Motivo = "" }, domain.ErrValidacao},
		"sem vencimento":         {func(e *portsin.AdiarSolicitacaoInput) { e.Vencimento = time.Time{} }, domain.ErrValidacao},
		"antes do atual":         {func(e *portsin.AdiarSolicitacaoInput) { e.Vencimento = time.Now().UTC().AddDate(0, 0, 1) }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		entrada := valida()
		caso.ajustar(&entrada)

		if err := c.solicitacao.Adiar(c.ctx, entrada); !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}

	if salva := c.solicitacoesRepo.itens[aprovada]; salva.Versao != 2 {
		t.Errorf("tentativas recusadas alteraram a solicitação: versão %d", salva.Versao)
	}
}
