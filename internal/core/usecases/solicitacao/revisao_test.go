package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

func TestEditarSolicitacaoVencidaSemMudarOVencimento(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	id := c.criarSolicitacao(t, solicitante, 10000)
	vencido := time.Now().UTC().AddDate(0, 0, -5)
	c.solicitacoesRepo.itens[id].Prazo = domainsolicitacao.PrazoPagamentoDe(vencido)

	entrada := entradaEdicao(id, solicitante)
	entrada.PrazoPagamento = time.Date(vencido.Year(), vencido.Month(), vencido.Day(), 0, 0, 0, 0, time.UTC)
	if err := c.solicitacao.Editar(c.ctx, entrada); err != nil {
		t.Fatalf("editar vencida sem mudar o vencimento = %v, esperado permitido", err)
	}
	if salva := c.solicitacoesRepo.itens[id]; salva.Valor.Centavos() != 25000 || salva.Prazo.String() != vencido.Format("2006-01-02") {
		t.Errorf("salva = valor %d, vencimento %s", salva.Valor.Centavos(), salva.Prazo)
	}

	entrada.Versao = 2
	entrada.PrazoPagamento = time.Now().UTC().AddDate(0, 0, -1)
	if err := c.solicitacao.Editar(c.ctx, entrada); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("mudar para outro dia no passado = %v, esperado erro de validação", err)
	}
}

func TestEditarAprovadaRespondeConflitoAntesDeValidarOCorpo(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	id := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, id)

	entrada := entradaEdicao(id, solicitante)
	entrada.Versao = 2
	entrada.ValorCentavos = 0
	if err := c.solicitacao.Editar(c.ctx, entrada); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("editar aprovada com corpo inválido = %v, esperado conflito", err)
	}
}

func TestRegistrarPagamentoConfereStatusECategoriaAntesDaConta(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)

	pendente := c.criarSolicitacao(t, c.novoUsuario("Ana", false, false), 10000)
	err := c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{SolicitacaoID: pendente, UsuarioID: financeiro, ValorCentavos: 100})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pagar pendente sem conta = %v, esperado conflito", err)
	}

	aprovada := c.solicitacaoAprovada(t)
	conta := c.prepararPagamento(t, aprovada)
	categoria := c.categoriasRepo.itens[*c.solicitacoesRepo.itens[aprovada].CategoriaFinanceiraID]
	categoria.Ativa = false
	err = c.solicitacao.RegistrarPagamento(c.ctx, portsin.RegistrarPagamentoInput{
		SolicitacaoID: aprovada, UsuarioID: financeiro, ValorCentavos: 100, ContaBancariaID: conta,
	})
	if err == nil || err.Error() != domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel().Error() {
		t.Errorf("pagar com categoria desativada = %v", err)
	}
}

func TestAprovarNaVersaoRecusaVersaoAntiga(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	id := c.criarSolicitacao(t, solicitante, 10000)

	if err := c.solicitacao.Editar(c.ctx, entradaEdicao(id, solicitante)); err != nil {
		t.Fatalf("edição falhou: %v", err)
	}

	if err := c.solicitacao.AprovarNaVersao(c.ctx, id, aprovador, 1); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("aprovar a versão vista antes da edição = %v, esperado conflito", err)
	}
	if salva := c.solicitacoesRepo.itens[id]; salva.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("status = %q, esperado pendente", salva.Status)
	}
	if err := c.solicitacao.AprovarNaVersao(c.ctx, id, aprovador, 2); err != nil {
		t.Errorf("aprovar a versão atual = %v", err)
	}
}

func TestGerarDeNovoDepoisDeDesativarACategoriaContaComoJaExistente(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))

	primeira, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2026-03"})
	if err != nil || len(primeira.Geradas) != 1 {
		t.Fatalf("primeira geração = %+v, %v", primeira, err)
	}

	c.categoriasRepo.itens[categoriaID].Ativa = false
	segunda, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2026-03"})
	if err != nil || segunda.JaExistentes != 1 || len(segunda.Recusadas) != 0 {
		t.Errorf("segunda geração = %+v, %v; esperado uma já existente e nenhuma recusada", segunda, err)
	}
}

func TestEditarSolicitacaoGeradaNaoMudaACompetencia(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	dados := c.dadosRecorrencia(categoriaID)
	dados.VencimentoMesSeguinte = true
	c.criarRecorrencia(t, financeiro, c.empresaID, dados)

	resultado, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2099-10"})
	if err != nil || len(resultado.Geradas) != 1 {
		t.Fatalf("geração = %+v, %v", resultado, err)
	}
	id := resultado.Geradas[0]
	gerada := c.solicitacoesRepo.itens[id]

	entrada := entradaEdicao(id, financeiro)
	entrada.PrazoPagamento = gerada.Prazo.Data()
	entrada.CategoriaFinanceiraID = &categoriaID
	if err := c.solicitacao.Editar(c.ctx, entrada); err != nil {
		t.Fatalf("edição da gerada falhou: %v", err)
	}
	if salva := c.solicitacoesRepo.itens[id]; salva.Competencia.String() != "2099-10" {
		t.Errorf("competência = %s, esperado 2099-10 mantida", salva.Competencia)
	}

	repetida, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2099-10"})
	if err != nil || len(repetida.Geradas) != 0 || repetida.JaExistentes != 1 {
		t.Errorf("regerar depois da edição = %+v, %v; esperado nenhuma duplicata", repetida, err)
	}
}
