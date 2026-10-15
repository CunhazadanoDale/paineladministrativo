package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func (c *cenario) solicitacaoAprovada(t *testing.T) uuid.UUID {
	t.Helper()

	solicitante := c.novoUsuario("Solicitante "+uuid.NewString()[:8], false, false)
	id := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, id)

	return id
}

func (c *cenario) pagarSolicitacao(t *testing.T, id, financeiro uuid.UUID, conta *uuid.UUID) {
	t.Helper()

	if err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID: id, UsuarioID: financeiro, ValorCentavos: 9900, ContaBancariaID: conta,
		PagoEm: time.Now().UTC().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
}

func TestRegistrarPagamentoGravaContaEQuemRegistrou(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	id := c.solicitacaoAprovada(t)
	conta := c.prepararPagamento(t, id)

	c.pagarSolicitacao(t, id, financeiro, conta)

	pagamento, err := c.solicitacao.ObterPagamento(c.ctx, id, financeiro)
	if err != nil {
		t.Fatalf("consulta do pagamento falhou: %v", err)
	}
	if *pagamento.ContaBancariaID != *conta || *pagamento.UsuarioID != financeiro || pagamento.Estornado {
		t.Errorf("pagamento = %+v", pagamento)
	}
	if evento := c.ultimoEvento(t, id, financeiro); evento.Evento != domainsolicitacao.EventoPaga {
		t.Errorf("último evento = %q, esperado paga", evento.Evento)
	}
}

func TestRegistrarPagamentoRecusaContaInvalidaESolicitacaoSemCategoria(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	outraEmpresa := c.novaEmpresa(true)

	semCategoria := c.solicitacaoAprovada(t)
	contaSemCategoria := c.novaConta(c.empresaID, true)

	empresaInativa := c.novaEmpresa(true)
	solicitante := c.novoUsuario("Ana", false, false)
	daInativa := c.criarSolicitacaoNaEmpresa(t, solicitante, empresaInativa)
	c.aprovarSolicitacao(t, daInativa)
	contaDaInativa := c.prepararPagamento(t, daInativa)
	c.empresasRepo.itens[empresaInativa].Ativa = false

	id := c.solicitacaoAprovada(t)
	c.prepararPagamento(t, id)
	inexistente := uuid.New()
	daOutra := c.novaConta(outraEmpresa, true)
	inativa := c.novaConta(c.empresaID, false)

	casos := map[string]struct {
		solicitacaoID uuid.UUID
		conta         *uuid.UUID
		mensagem      string
	}{
		"sem conta":         {id, nil, "conta bancária do pagamento é obrigatória"},
		"conta inexistente": {id, &inexistente, "conta bancária não encontrada"},
		"de outra empresa":  {id, &daOutra, "conta bancária deve ser da empresa da solicitação"},
		"conta inativa":     {id, &inativa, "conta bancária inativa não recebe pagamento"},
		"empresa inativa":   {daInativa, contaDaInativa, "empresa inativa não recebe pagamento"},
		"sem categoria":     {semCategoria, &contaSemCategoria, "solicitação sem categoria financeira; classifique antes de pagar"},
	}

	for nome, caso := range casos {
		err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
			SolicitacaoID: caso.solicitacaoID, UsuarioID: financeiro, ValorCentavos: 9900, ContaBancariaID: caso.conta,
		})
		if err == nil || err.Error() != domain.ErroValidacao(caso.mensagem).Error() {
			t.Errorf("%s = %v, esperado %q", nome, err, caso.mensagem)
		}
	}
}

func TestEstornarPagamentoReabreParaPagarDeNovo(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	id := c.solicitacaoAprovada(t)
	conta := c.prepararPagamento(t, id)
	c.pagarSolicitacao(t, id, financeiro, conta)

	err := c.solicitacao.EstornarPagamento(c.ctx, portsin.EstornarPagamentoInput{
		SolicitacaoID: id,
		UsuarioID:     financeiro,
		Motivo:        "pago em duplicidade",
		Versao:        3,
	})
	if err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}

	salva := c.solicitacoesRepo.itens[id]
	if salva.Status != domainsolicitacao.StatusAprovado || salva.Versao != 4 {
		t.Errorf("salva = status %q, versão %d", salva.Status, salva.Versao)
	}
	if _, err := c.solicitacao.ObterPagamento(c.ctx, id, financeiro); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pagamento vigente depois do estorno = %v, esperado não encontrado", err)
	}

	evento := c.ultimoEvento(t, id, financeiro)
	if evento.Evento != domainsolicitacao.EventoEstornada || evento.Descricao != "pagamento estornado: pago em duplicidade" ||
		*evento.DeStatus != domainsolicitacao.StatusPago || evento.ParaStatus != domainsolicitacao.StatusAprovado {
		t.Errorf("último registro = %+v", evento)
	}

	c.pagarSolicitacao(t, id, financeiro, conta)

	pagamentos, err := c.solicitacao.ListarPagamentos(c.ctx, id, financeiro)
	if err != nil || len(pagamentos) != 2 {
		t.Fatalf("pagamentos = %+v, %v; esperado dois", pagamentos, err)
	}
	if !pagamentos[0].Estornado || pagamentos[0].Estorno == nil || pagamentos[0].Estorno.Motivo.Texto() != "pago em duplicidade" {
		t.Errorf("primeiro = %+v, esperado estornado", pagamentos[0])
	}
	if pagamentos[1].Estornado {
		t.Errorf("segundo = %+v, esperado vigente", pagamentos[1])
	}
}

func TestEstornarPagamentoRecusaPerfilAcessoVersaoStatusEDados(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	solicitante := c.novoUsuario("Ana", false, false)
	outraEmpresa := c.novaEmpresa(true)
	financeiroDeOutra := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroDeOutra, outraEmpresa)

	paga := c.solicitacaoAprovada(t)
	c.pagarSolicitacao(t, paga, financeiro, c.prepararPagamento(t, paga))
	aprovada := c.solicitacaoAprovada(t)

	valida := func() portsin.EstornarPagamentoInput {
		return portsin.EstornarPagamentoInput{SolicitacaoID: paga, UsuarioID: financeiro, Motivo: "erro de lançamento", Versao: 3}
	}

	casos := map[string]struct {
		ajustar  func(*portsin.EstornarPagamentoInput)
		esperado error
	}{
		"sem perfil financeiro":   {func(e *portsin.EstornarPagamentoInput) { e.UsuarioID = solicitante }, domain.ErrPermissao},
		"empresa fora do acesso":  {func(e *portsin.EstornarPagamentoInput) { e.UsuarioID = financeiroDeOutra }, domain.ErrPermissao},
		"versão ausente":          {func(e *portsin.EstornarPagamentoInput) { e.Versao = 0 }, domain.ErrValidacao},
		"versão divergente":       {func(e *portsin.EstornarPagamentoInput) { e.Versao = 2 }, domain.ErrConflito},
		"não paga":                {func(e *portsin.EstornarPagamentoInput) { e.SolicitacaoID = aprovada; e.Versao = 2 }, domain.ErrConflito},
		"sem motivo":              {func(e *portsin.EstornarPagamentoInput) { e.Motivo = " " }, domain.ErrValidacao},
		"data no futuro":          {func(e *portsin.EstornarPagamentoInput) { e.EstornadoEm = time.Now().Add(time.Hour) }, domain.ErrValidacao},
		"data antes do pagamento": {func(e *portsin.EstornarPagamentoInput) { e.EstornadoEm = time.Now().AddDate(0, 0, -2) }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		entrada := valida()
		caso.ajustar(&entrada)

		if err := c.solicitacao.EstornarPagamento(c.ctx, entrada); !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}

	if salva := c.solicitacoesRepo.itens[paga]; salva.Status != domainsolicitacao.StatusPago || salva.Versao != 3 {
		t.Errorf("tentativas recusadas alteraram a solicitação: status %q, versão %d", salva.Status, salva.Versao)
	}
}
