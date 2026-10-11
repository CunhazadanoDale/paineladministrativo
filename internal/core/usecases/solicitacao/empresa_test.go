package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func TestCriarSolicitacaoExigeEmpresaAtivaEAcessivel(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	outra := c.novaEmpresa(true)
	inativa := c.novaEmpresa(false)
	c.restringirAcesso(ana, c.empresaID, inativa)

	casos := []struct {
		nome      string
		empresaID uuid.UUID
		sentinela error
	}{
		{"sem empresa", uuid.Nil, domain.ErrValidacao},
		{"empresa inexistente", uuid.New(), domain.ErrValidacao},
		{"empresa fora do acesso", outra, domain.ErrPermissao},
		{"empresa inativa", inativa, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
				SolicitanteID:  ana,
				EmpresaID:      caso.empresaID,
				ValorCentavos:  10000,
				PrazoPagamento: time.Now().UTC().AddDate(0, 0, 7),
				Observacao:     "material para a obra",
				FormaPagamento: "pix",
			})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	id := c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	salva, err := c.solicitacao.Obter(c.ctx, id, ana)
	if err != nil || salva.EmpresaID != c.empresaID {
		t.Errorf("solicitação salva %+v (%v), esperado na empresa do acesso", salva, err)
	}
}

func TestAprovadorSoEnxergaEDecideEmpresasDoSeuAcesso(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)
	ana := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	c.restringirAcesso(aprovador, c.empresaID)

	daPrimeira := c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	daSegunda := c.criarSolicitacaoNaEmpresa(t, ana, segunda)

	fila, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: aprovador, Escopo: portsin.EscopoAprovacao})
	if err != nil {
		t.Fatalf("fila de aprovação falhou: %v", err)
	}
	if len(fila) != 1 || fila[0].ID != daPrimeira {
		t.Errorf("fila com %d solicitações, esperado só a da empresa do acesso", len(fila))
	}

	if _, err := c.solicitacao.Obter(c.ctx, daSegunda, aprovador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("consulta fora do acesso = %v, esperado erro de permissão", err)
	}
	if err := c.solicitacao.Aprovar(c.ctx, daSegunda, aprovador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("aprovação fora do acesso = %v, esperado erro de permissão", err)
	}
	if err := c.solicitacao.Rejeitar(c.ctx, daSegunda, aprovador, "fora do orçamento"); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("rejeição fora do acesso = %v, esperado erro de permissão", err)
	}
	if err := c.solicitacao.Aprovar(c.ctx, daPrimeira, aprovador); err != nil {
		t.Errorf("aprovação dentro do acesso falhou: %v", err)
	}
}

func TestFinanceiroSoEnxergaEPagaEmpresasDoSeuAcesso(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)
	ana := c.novoUsuario("Ana", false, false)
	administrador := c.novoUsuario("Diretoria", true, false)
	financeiro := c.novoUsuario("Caio", false, true)
	c.restringirAcesso(financeiro, segunda)

	daPrimeira := c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	daSegunda := c.criarSolicitacaoNaEmpresa(t, ana, segunda)
	for _, id := range []uuid.UUID{daPrimeira, daSegunda} {
		if err := c.solicitacao.Aprovar(c.ctx, id, administrador); err != nil {
			t.Fatalf("aprovação falhou: %v", err)
		}
	}

	fila, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: financeiro, Escopo: portsin.EscopoFinanceiro})
	if err != nil {
		t.Fatalf("fila do financeiro falhou: %v", err)
	}
	if len(fila) != 1 || fila[0].ID != daSegunda {
		t.Errorf("fila com %d solicitações, esperado só a da empresa do acesso", len(fila))
	}

	pagamento := func(id uuid.UUID) portsin.RegistrarPagamentoInput {
		return portsin.RegistrarPagamentoInput{SolicitacaoID: id, UsuarioID: financeiro, ValorCentavos: 10000}
	}
	if err := c.solicitacao.RegistrarPagamento(c.ctx, pagamento(daPrimeira)); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("pagamento fora do acesso = %v, esperado erro de permissão", err)
	}
	if err := c.solicitacao.RegistrarPagamento(c.ctx, pagamento(daSegunda)); err != nil {
		t.Errorf("pagamento dentro do acesso falhou: %v", err)
	}
}

func TestFiltroDeEmpresaNaListagem(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)
	ana := c.novoUsuario("Ana", false, false)

	c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	daSegunda := c.criarSolicitacaoNaEmpresa(t, ana, segunda)

	minhas, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: ana, Empresa: &segunda})
	if err != nil {
		t.Fatalf("listagem filtrada falhou: %v", err)
	}
	if len(minhas) != 1 || minhas[0].ID != daSegunda {
		t.Errorf("listagem filtrada com %d solicitações, esperado só a da segunda empresa", len(minhas))
	}

	c.restringirAcesso(ana, c.empresaID)
	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: ana, Empresa: &segunda}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("filtro por empresa fora do acesso = %v, esperado erro de permissão", err)
	}
}

func TestSolicitanteContinuaVendoAPropriaSolicitacaoSemAcessoAEmpresa(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)
	ana := c.novoUsuario("Ana", false, false)
	id := c.criarSolicitacaoNaEmpresa(t, ana, segunda)

	c.restringirAcesso(ana, c.empresaID)

	if _, err := c.solicitacao.Obter(c.ctx, id, ana); err != nil {
		t.Errorf("solicitante perdeu a própria solicitação: %v", err)
	}
	if err := c.solicitacao.Cancelar(c.ctx, id, ana); err != nil {
		t.Errorf("solicitante não cancelou a própria solicitação: %v", err)
	}
}
