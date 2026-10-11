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
	bruno := c.novoUsuario("Bruno", false, false)
	outra := c.novaEmpresa(true)
	inativa := c.novaEmpresa(false)
	c.restringirAcesso(ana, c.empresaID, inativa)

	casos := []struct {
		nome        string
		solicitante uuid.UUID
		empresaID   uuid.UUID
		sentinela   error
	}{
		{"sem empresa", ana, uuid.Nil, domain.ErrValidacao},
		{"empresa inexistente com acesso a todas", bruno, uuid.New(), domain.ErrValidacao},
		{"empresa inexistente com acesso restrito", ana, uuid.New(), domain.ErrPermissao},
		{"empresa fora do acesso", ana, outra, domain.ErrPermissao},
		{"empresa inativa", ana, inativa, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
				SolicitanteID:  caso.solicitante,
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
	proprias, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: ana, Empresa: &segunda})
	if err != nil || len(proprias) != 1 {
		t.Errorf("solicitante filtrando as próprias de empresa sem acesso = %d, %v; esperado a própria", len(proprias), err)
	}

	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	c.restringirAcesso(aprovador, c.empresaID)
	if _, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{
		UsuarioID: aprovador,
		Escopo:    portsin.EscopoAprovacao,
		Empresa:   &segunda,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("fila filtrada por empresa fora do acesso = %v, esperado erro de permissão", err)
	}
}

func TestDetalhesDaSolicitacaoRespeitamOAcessoAEmpresa(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)
	ana := c.novoUsuario("Ana", false, false)
	administrador := c.novoUsuario("Diretoria", true, false)
	financeiro := c.novoUsuario("Caio", false, true)
	c.restringirAcesso(financeiro, c.empresaID)

	anexo := c.enviarArquivo(t, ana, "nota.pdf", "application/pdf")
	id, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  ana,
		EmpresaID:      segunda,
		ValorCentavos:  10000,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 7),
		Observacao:     "material para a obra",
		FormaPagamento: "pix",
		ArquivoIDs:     []uuid.UUID{anexo.ID},
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	if err := c.solicitacao.Aprovar(c.ctx, id, administrador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID: id,
		UsuarioID:     administrador,
		ValorCentavos: 10000,
	}); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	consultas := []struct {
		nome     string
		executar func() error
	}{
		{"histórico", func() error { _, err := c.solicitacao.ListarHistorico(c.ctx, id, financeiro); return err }},
		{"anexos", func() error { _, err := c.solicitacao.ListarArquivos(c.ctx, id, financeiro); return err }},
		{"pagamento", func() error { _, err := c.solicitacao.ObterPagamento(c.ctx, id, financeiro); return err }},
		{"download do anexo", func() error {
			_, conteudo, err := c.arquivo.Baixar(c.ctx, anexo.ID, financeiro)
			if conteudo != nil {
				_ = conteudo.Close()
			}
			return err
		}},
	}

	for _, consulta := range consultas {
		t.Run(consulta.nome, func(t *testing.T) {
			if err := consulta.executar(); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("erro %v, esperado erro de permissão", err)
			}
		})
	}

	c.restringirAcesso(financeiro, c.empresaID, segunda)
	for _, consulta := range consultas {
		if err := consulta.executar(); err != nil {
			t.Errorf("%s com acesso à empresa falhou: %v", consulta.nome, err)
		}
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

func TestEquipesDeEmpresasDiferentesNaoSeEnxergam(t *testing.T) {
	c := novoCenario(t)
	segunda := c.novaEmpresa(true)

	ana := c.novoUsuario("Ana", false, false)
	bia := c.novoUsuario("Bia", false, false)
	c.restringirAcesso(ana, c.empresaID)
	c.restringirAcesso(bia, segunda)

	aprovadorDaPrimeira := c.novoUsuario("Bruno", false, false)
	aprovadorDaSegunda := c.novoUsuario("Beto", false, false)
	c.designar(t, aprovadorDaPrimeira)
	c.designar(t, aprovadorDaSegunda)
	c.restringirAcesso(aprovadorDaPrimeira, c.empresaID)
	c.restringirAcesso(aprovadorDaSegunda, segunda)

	daAna := c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	daBia := c.criarSolicitacaoNaEmpresa(t, bia, segunda)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		propria   uuid.UUID
		alheia    uuid.UUID
		aprovador bool
	}{
		{"solicitante da primeira", ana, daAna, daBia, false},
		{"solicitante da segunda", bia, daBia, daAna, false},
		{"aprovador da primeira", aprovadorDaPrimeira, daAna, daBia, true},
		{"aprovador da segunda", aprovadorDaSegunda, daBia, daAna, true},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			escopo := portsin.EscopoMinhas
			if caso.aprovador {
				escopo = portsin.EscopoAprovacao
			}

			lista, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: caso.usuario, Escopo: escopo})
			if err != nil {
				t.Fatalf("listagem falhou: %v", err)
			}
			if len(lista) != 1 || lista[0].ID != caso.propria {
				t.Errorf("listagem com %d solicitações, esperado só a da própria empresa", len(lista))
			}

			if _, err := c.solicitacao.Obter(c.ctx, caso.alheia, caso.usuario); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("consulta da outra empresa = %v, esperado erro de permissão", err)
			}
			if err := c.solicitacao.Aprovar(c.ctx, caso.alheia, caso.usuario); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("aprovação da outra empresa = %v, esperado erro de permissão", err)
			}
		})
	}
}

func TestFilasComAcessoVazioNaoDevolvemNada(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	administrador := c.novoUsuario("Diretoria", true, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	financeiro := c.novoUsuario("Caio", false, true)
	c.designar(t, aprovador)
	c.restringirAcesso(aprovador)
	c.restringirAcesso(financeiro)

	c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	aprovada := c.criarSolicitacaoNaEmpresa(t, ana, c.empresaID)
	if err := c.solicitacao.Aprovar(c.ctx, aprovada, administrador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	casos := []struct {
		nome    string
		usuario uuid.UUID
		escopo  string
	}{
		{"aprovação", aprovador, portsin.EscopoAprovacao},
		{"financeiro", financeiro, portsin.EscopoFinanceiro},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			fila, err := c.solicitacao.Listar(c.ctx, portsin.ListarSolicitacoesInput{UsuarioID: caso.usuario, Escopo: caso.escopo})
			if err != nil {
				t.Fatalf("fila falhou: %v", err)
			}
			if len(fila) != 0 {
				t.Errorf("fila com %d solicitações, esperado vazia para acesso sem empresas", len(fila))
			}
		})
	}
}
