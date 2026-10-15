package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func (c *cenario) dadosRecorrencia(categoriaID uuid.UUID) portsin.DadosDespesaRecorrenteInput {
	return portsin.DadosDespesaRecorrenteInput{
		CategoriaFinanceiraID: categoriaID,
		Descricao:             "aluguel do galpão",
		ValorCentavos:         420000,
		FormaPagamento:        "transferencia",
		DiaVencimento:         10,
		Inicio:                "2026-01",
	}
}

func (c *cenario) criarRecorrencia(t *testing.T, usuarioID, empresaID uuid.UUID, dados portsin.DadosDespesaRecorrenteInput) *domainsolicitacao.DespesaRecorrente {
	t.Helper()

	despesa, err := c.recorrencia.Criar(c.ctx, portsin.CriarDespesaRecorrenteInput{UsuarioID: usuarioID, EmpresaID: empresaID, Dados: dados})
	if err != nil {
		t.Fatalf("criação da recorrência falhou: %v", err)
	}

	return despesa
}

func TestCriarDespesaRecorrentePeloFinanceiro(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	fornecedorID := c.novoFornecedor(true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)

	dados := c.dadosRecorrencia(categoriaID)
	dados.FornecedorID = &fornecedorID
	dados.Fim = "2026-12"
	dados.Estimado = true
	dados.VencimentoMesSeguinte = true

	despesa := c.criarRecorrencia(t, financeiro, c.empresaID, dados)

	salva := c.recorrenciasRepo.itens[despesa.ID]
	if salva == nil || salva.EmpresaID != c.empresaID || *salva.FornecedorID != fornecedorID || salva.CategoriaFinanceiraID != categoriaID {
		t.Fatalf("salva = %+v", salva)
	}
	if salva.Inicio.String() != "2026-01" || salva.Fim.String() != "2026-12" || !salva.Estimado || !salva.VencimentoMesSeguinte ||
		salva.DiaVencimento.Dia() != 10 || salva.FormaPagamento != domainsolicitacao.FormaPagamentoTransferencia {
		t.Errorf("salva = %+v", salva)
	}
}

func TestCriarDespesaRecorrenteRecusaPerfilAcessoEmpresaEDados(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	comum := c.novoUsuario("Ana", false, false)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	receita := c.novaCategoria(domainfinanceiro.NaturezaReceita, true, true)
	inativa := c.novaEmpresa(false)
	outra := c.novaEmpresa(true)
	financeiroRestrito := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroRestrito, c.empresaID)

	casos := map[string]struct {
		usuarioID uuid.UUID
		empresaID uuid.UUID
		ajustar   func(*portsin.DadosDespesaRecorrenteInput)
		esperado  error
	}{
		"sem perfil financeiro":  {comum, c.empresaID, func(*portsin.DadosDespesaRecorrenteInput) {}, domain.ErrPermissao},
		"empresa fora do acesso": {financeiroRestrito, outra, func(*portsin.DadosDespesaRecorrenteInput) {}, domain.ErrPermissao},
		"empresa inativa":        {financeiro, inativa, func(*portsin.DadosDespesaRecorrenteInput) {}, domain.ErrValidacao},
		"categoria de receita":   {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.CategoriaFinanceiraID = receita }, domain.ErrValidacao},
		"sem categoria":          {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.CategoriaFinanceiraID = uuid.Nil }, domain.ErrValidacao},
		"sem início":             {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.Inicio = "" }, domain.ErrValidacao},
		"fim antes do início":    {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.Fim = "2025-12" }, domain.ErrValidacao},
		"dia inválido":           {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.DiaVencimento = 0 }, domain.ErrValidacao},
		"valor zerado":           {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.ValorCentavos = 0 }, domain.ErrValidacao},
		"sem descrição":          {financeiro, c.empresaID, func(d *portsin.DadosDespesaRecorrenteInput) { d.Descricao = "" }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		dados := c.dadosRecorrencia(categoriaID)
		caso.ajustar(&dados)

		_, err := c.recorrencia.Criar(c.ctx, portsin.CriarDespesaRecorrenteInput{UsuarioID: caso.usuarioID, EmpresaID: caso.empresaID, Dados: dados})
		if !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}

	if len(c.recorrenciasRepo.itens) != 0 {
		t.Errorf("%d recorrências gravadas, esperado nenhuma", len(c.recorrenciasRepo.itens))
	}
}

func TestDespesasRecorrentesIsoladasPorEmpresa(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	outra := c.novaEmpresa(true)
	financeiroRestrito := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroRestrito, c.empresaID)

	daPrimeira := c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))
	daOutra := c.criarRecorrencia(t, financeiro, outra, c.dadosRecorrencia(categoriaID))

	listadas, err := c.recorrencia.Listar(c.ctx, portsin.ListarDespesasRecorrentesInput{
		UsuarioID: financeiroRestrito, Filtro: domain.PaginacaoFiltro{Page: 1, Size: 50},
	})
	if err != nil || len(listadas) != 1 || listadas[0].ID != daPrimeira.ID {
		t.Errorf("listadas = %d, %v; esperado só a da empresa do acesso", len(listadas), err)
	}
	if _, err := c.recorrencia.Listar(c.ctx, portsin.ListarDespesasRecorrentesInput{UsuarioID: financeiroRestrito, Empresa: &outra}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("filtro por empresa fora do acesso = %v", err)
	}
	if _, err := c.recorrencia.Obter(c.ctx, daOutra.ID, financeiroRestrito); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("obter da outra empresa = %v", err)
	}
	if _, err := c.recorrencia.Desativar(c.ctx, portsin.MudarSituacaoDespesaRecorrenteInput{
		UsuarioID: financeiroRestrito, DespesaID: daOutra.ID, Versao: 1,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("desativar da outra empresa = %v", err)
	}
}

func TestAlterarAtivarEDesativarDespesaRecorrenteComVersao(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	despesa := c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))

	dados := c.dadosRecorrencia(categoriaID)
	dados.Descricao = "aluguel reajustado"
	dados.ValorCentavos = 450000

	if _, err := c.recorrencia.Alterar(c.ctx, portsin.AlterarDespesaRecorrenteInput{
		UsuarioID: financeiro, DespesaID: despesa.ID, Dados: dados, Versao: 2,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v", err)
	}

	alterada, err := c.recorrencia.Alterar(c.ctx, portsin.AlterarDespesaRecorrenteInput{
		UsuarioID: financeiro, DespesaID: despesa.ID, Dados: dados, Versao: 1,
	})
	if err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}
	if alterada.Versao != 2 || alterada.Valor.Centavos() != 450000 || alterada.Descricao.Texto() != "aluguel reajustado" {
		t.Errorf("alterada = %+v", alterada)
	}

	desativada, err := c.recorrencia.Desativar(c.ctx, portsin.MudarSituacaoDespesaRecorrenteInput{UsuarioID: financeiro, DespesaID: despesa.ID, Versao: 2})
	if err != nil || desativada.Ativa || desativada.Versao != 3 {
		t.Fatalf("desativada = %+v, %v", desativada, err)
	}
	deNovo, err := c.recorrencia.Desativar(c.ctx, portsin.MudarSituacaoDespesaRecorrenteInput{UsuarioID: financeiro, DespesaID: despesa.ID, Versao: 3})
	if err != nil || deNovo.Versao != 3 {
		t.Errorf("desativar desativada = versão %d, %v; esperado sem mudança", deNovo.Versao, err)
	}
	ativada, err := c.recorrencia.Ativar(c.ctx, portsin.MudarSituacaoDespesaRecorrenteInput{UsuarioID: financeiro, DespesaID: despesa.ID, Versao: 3})
	if err != nil || !ativada.Ativa || ativada.Versao != 4 {
		t.Errorf("ativada = %+v, %v", ativada, err)
	}
}

func TestRemoverDespesaRecorrenteSoPeloAdministrador(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	administrador := c.novoUsuario("Admin", true, false)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	despesa := c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))

	if err := c.recorrencia.Remover(c.ctx, despesa.ID, financeiro); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("financeiro removendo = %v, esperado erro de permissão", err)
	}
	if err := c.recorrencia.Remover(c.ctx, despesa.ID, administrador); err != nil {
		t.Errorf("administrador removendo = %v", err)
	}
	if err := c.recorrencia.Remover(c.ctx, despesa.ID, administrador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("remover de novo = %v, esperado não encontrado", err)
	}
}

func TestGerarDespesasRecorrentesDaCompetencia(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	fornecedorInativo := c.novoFornecedor(true)

	aluguel := c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))
	futura := c.dadosRecorrencia(categoriaID)
	futura.Inicio = "2026-08"
	c.criarRecorrencia(t, financeiro, c.empresaID, futura)
	comFornecedor := c.dadosRecorrencia(categoriaID)
	comFornecedor.FornecedorID = &fornecedorInativo
	recusada := c.criarRecorrencia(t, financeiro, c.empresaID, comFornecedor)
	c.fornecedoresRepo.itens[fornecedorInativo].Ativo = false

	resultado, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2026-03"})
	if err != nil {
		t.Fatalf("geração falhou: %v", err)
	}
	if resultado.Competencia != "2026-03" || len(resultado.Geradas) != 1 || resultado.JaExistentes != 0 || len(resultado.Recusadas) != 1 {
		t.Fatalf("resultado = %+v", resultado)
	}
	if resultado.Recusadas[0].DespesaRecorrenteID != recusada.ID || resultado.Recusadas[0].Motivo != "fornecedor inativo não recebe conta a pagar" {
		t.Errorf("recusada = %+v", resultado.Recusadas[0])
	}

	gerada := c.solicitacoesRepo.itens[resultado.Geradas[0]]
	if gerada.Origem != domainsolicitacao.OrigemRecorrencia || *gerada.OrigemID != aluguel.ID || gerada.SolicitanteID != financeiro ||
		gerada.Status != domainsolicitacao.StatusPendenteAprovacao || gerada.Prazo.String() != "2026-03-10" {
		t.Errorf("gerada = %+v", gerada)
	}
	if evento := c.ultimoEvento(t, gerada.ID, financeiro); evento.Evento != domainsolicitacao.EventoCriada ||
		evento.Descricao != "solicitação gerada pela despesa recorrente na competência 2026-03" {
		t.Errorf("histórico = %+v", evento)
	}

	repetida, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: "2026-03"})
	if err != nil || len(repetida.Geradas) != 0 || repetida.JaExistentes != 1 {
		t.Errorf("segunda geração = %+v, %v; esperado uma já existente", repetida, err)
	}
}

func TestGerarDespesasRecorrentesRespeitaPerfilEAcesso(t *testing.T) {
	c := novoCenario(t)
	financeiro := c.novoUsuario("Fábio", false, true)
	comum := c.novoUsuario("Ana", false, false)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	outra := c.novaEmpresa(true)
	financeiroRestrito := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroRestrito, c.empresaID)

	c.criarRecorrencia(t, financeiro, c.empresaID, c.dadosRecorrencia(categoriaID))
	c.criarRecorrencia(t, financeiro, outra, c.dadosRecorrencia(categoriaID))

	if _, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: comum, Competencia: "2026-03"}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("sem perfil = %v", err)
	}
	for _, competencia := range []string{"", "03/2026"} {
		if _, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiro, Competencia: competencia}); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("competência %q = %v", competencia, err)
		}
	}
	if _, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{
		UsuarioID: financeiroRestrito, Competencia: "2026-03", Empresa: &outra,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v", err)
	}

	resultado, err := c.recorrencia.Gerar(c.ctx, portsin.GerarDespesasRecorrentesInput{UsuarioID: financeiroRestrito, Competencia: "2026-03"})
	if err != nil || len(resultado.Geradas) != 1 {
		t.Fatalf("geração restrita = %+v, %v; esperado só a da empresa do acesso", resultado, err)
	}
	if gerada := c.solicitacoesRepo.itens[resultado.Geradas[0]]; gerada.EmpresaID != c.empresaID {
		t.Errorf("gerada para a empresa %s, esperado %s", gerada.EmpresaID, c.empresaID)
	}
}
