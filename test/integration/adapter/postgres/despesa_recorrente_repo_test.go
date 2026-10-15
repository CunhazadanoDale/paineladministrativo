package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

type cenarioBancoRecorrencias struct {
	*cenarioBancoClassificacao
	recorrencias *postgres.DespesaRecorrenteRepository
}

func cenarioRecorrencias(t *testing.T) *cenarioBancoRecorrencias {
	t.Helper()

	base := cenarioClassificacao(t)

	return &cenarioBancoRecorrencias{
		cenarioBancoClassificacao: base,
		recorrencias:              postgres.NewDespesaRecorrenteRepository(base.banco),
	}
}

func competenciaDe(t *testing.T, texto string) domainsolicitacao.Competencia {
	t.Helper()

	competencia, err := domainsolicitacao.NovaCompetencia(texto)
	if err != nil {
		t.Fatalf("competência de teste inválida: %v", err)
	}

	return competencia
}

func (c *cenarioBancoRecorrencias) recorrenciaDeDominio(t *testing.T, empresaID uuid.UUID, descricao, inicio string, fim *string) *domainsolicitacao.DespesaRecorrente {
	t.Helper()

	texto, err := domainsolicitacao.NovaObservacao(descricao)
	if err != nil {
		t.Fatalf("descrição de teste inválida: %v", err)
	}
	dados := domainsolicitacao.DadosDespesaRecorrente{
		FornecedorID:          &c.fornecedorID,
		CategoriaFinanceiraID: c.despesaID,
		Descricao:             texto,
		Valor:                 domainsolicitacao.ValorDe(420000),
		Estimado:              true,
		FormaPagamento:        domainsolicitacao.FormaPagamentoDebitoAutomatico,
		DiaVencimento:         domainsolicitacao.DiaVencimentoDe(31),
		VencimentoMesSeguinte: true,
		Inicio:                competenciaDe(t, inicio),
	}
	if fim != nil {
		competencia := competenciaDe(t, *fim)
		dados.Fim = &competencia
	}

	despesa, err := domainsolicitacao.NovaDespesaRecorrente(empresaID, dados)
	if err != nil {
		t.Fatalf("despesa recorrente de teste inválida: %v", err)
	}

	return despesa
}

func (c *cenarioBancoRecorrencias) novaRecorrencia(t *testing.T, empresaID uuid.UUID, descricao, inicio string, fim *string) *domainsolicitacao.DespesaRecorrente {
	t.Helper()

	despesa := c.recorrenciaDeDominio(t, empresaID, descricao, inicio, fim)
	if err := c.recorrencias.Criar(context.Background(), despesa); err != nil {
		t.Fatalf("criação da recorrência %q falhou: %v", descricao, err)
	}

	return despesa
}

func (c *cenarioBancoRecorrencias) gerar(t *testing.T, despesa *domainsolicitacao.DespesaRecorrente, competencia string) (*domainsolicitacao.Solicitacao, bool, error) {
	t.Helper()

	solicitacao, err := despesa.GerarSolicitacao(c.terceiroID, competenciaDe(t, competencia), time.Now().UTC())
	if err != nil {
		t.Fatalf("geração no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, nil, solicitacao.Status, domainsolicitacao.EventoCriada, "gerada pela recorrência",
	)

	criada, err := c.solicitacoes.GerarDeRecorrencia(context.Background(), solicitacao, historico)

	return solicitacao, criada, err
}

func textoPara(valor string) *string {
	return &valor
}

func TestDespesaRecorrenteCriadaPersisteTodosOsCampos(t *testing.T) {
	c := cenarioRecorrencias(t)
	despesa := c.novaRecorrencia(t, c.empresaID, "Aluguel do galpão", "2026-01", textoPara("2026-12"))

	gravada, err := c.recorrencias.Obter(context.Background(), despesa.ID)
	if err != nil || gravada == nil {
		t.Fatalf("obter = %v, %v", gravada, err)
	}
	if gravada.EmpresaID != c.empresaID || *gravada.FornecedorID != c.fornecedorID || gravada.CategoriaFinanceiraID != c.despesaID {
		t.Errorf("vínculos = %+v", gravada)
	}
	if gravada.Descricao.Texto() != "Aluguel do galpão" || gravada.Valor.Centavos() != 420000 || !gravada.Estimado ||
		gravada.FormaPagamento != domainsolicitacao.FormaPagamentoDebitoAutomatico {
		t.Errorf("dados = %+v", gravada)
	}
	if gravada.DiaVencimento.Dia() != 31 || !gravada.VencimentoMesSeguinte || gravada.Inicio.String() != "2026-01" ||
		gravada.Fim == nil || gravada.Fim.String() != "2026-12" || !gravada.Ativa || gravada.Versao != 1 {
		t.Errorf("vigência = %+v", gravada)
	}
}

func TestDespesaRecorrenteCriarRecusaEmpresaInativaECategoriaDeReceita(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()
	inativa := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Inativa")
	if _, err := c.banco.ExecContext(ctx, `UPDATE empresa SET ativa = false WHERE id = $1`, inativa); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	if err := c.recorrencias.Criar(ctx, c.recorrenciaDeDominio(t, inativa, "Energia", "2026-01", nil)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("empresa inativa = %v, esperado erro de validação", err)
	}

	receita := c.recorrenciaDeDominio(t, c.empresaID, "Energia", "2026-01", nil)
	receita.CategoriaFinanceiraID = c.receitaID
	if err := c.recorrencias.Criar(ctx, receita); err == nil || err.Error() != domainsolicitacao.ErroCategoriaFinanceiraNaoEhDespesa().Error() {
		t.Errorf("categoria de receita = %v", err)
	}
}

func TestDespesaRecorrenteAtualizarAtivarEDesativarComVersao(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()
	despesa := c.novaRecorrencia(t, c.empresaID, "Internet", "2026-01", nil)

	dados := domainsolicitacao.DadosDespesaRecorrente{
		CategoriaFinanceiraID: c.despesaID,
		Descricao:             domainsolicitacao.ObservacaoDe("Internet fibra"),
		Valor:                 domainsolicitacao.ValorDe(19990),
		FormaPagamento:        domainsolicitacao.FormaPagamentoBoleto,
		DiaVencimento:         domainsolicitacao.DiaVencimentoDe(5),
		Inicio:                competenciaDe(t, "2026-02"),
	}
	if err := despesa.Alterar(dados, time.Now().UTC()); err != nil {
		t.Fatalf("alteração no domínio falhou: %v", err)
	}
	velha := *despesa
	if err := c.recorrencias.Atualizar(ctx, despesa); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}
	if despesa.Versao != 2 {
		t.Errorf("versão = %d, esperado 2", despesa.Versao)
	}
	if err := c.recorrencias.Atualizar(ctx, &velha); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	gravada, _ := c.recorrencias.Obter(ctx, despesa.ID)
	if gravada.Descricao.Texto() != "Internet fibra" || gravada.FornecedorID != nil || gravada.Inicio.String() != "2026-02" || gravada.Estimado {
		t.Errorf("gravada = %+v", gravada)
	}

	despesa.Desativar(time.Now().UTC())
	if err := c.recorrencias.Desativar(ctx, despesa); err != nil || despesa.Versao != 3 {
		t.Fatalf("desativação = %v, versão %d", err, despesa.Versao)
	}

	if _, err := c.banco.ExecContext(ctx, `UPDATE empresa SET ativa = false WHERE id = $1`, c.empresaID); err != nil {
		t.Fatalf("desativação da empresa falhou: %v", err)
	}
	despesa.Ativar(time.Now().UTC())
	if err := c.recorrencias.Ativar(ctx, despesa); err == nil || err.Error() != domainsolicitacao.ErroEmpresaInativaNaRecorrencia().Error() {
		t.Errorf("reativar com empresa inativa = %v", err)
	}
}

func TestDespesaRecorrenteListarEListarVigentes(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()
	outra := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Red Diamond")

	aluguel := c.novaRecorrencia(t, c.empresaID, "Aluguel", "2026-01", nil)
	energia := c.novaRecorrencia(t, c.empresaID, "energia", "2026-05", nil)
	encerrada := c.novaRecorrencia(t, c.empresaID, "Contador", "2025-01", textoPara("2026-02"))
	daOutra := c.novaRecorrencia(t, outra, "Água", "2026-01", nil)
	inativa := c.novaRecorrencia(t, c.empresaID, "Seguro", "2026-01", nil)
	inativa.Desativar(time.Now().UTC())
	if err := c.recorrencias.Desativar(ctx, inativa); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	ativas := true
	listadas, err := c.recorrencias.Listar(ctx, portsout.DespesaRecorrenteFiltro{
		PaginacaoFiltro: todasAsPaginas, EmpresaIDs: []uuid.UUID{c.empresaID}, Ativa: &ativas,
	})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(listadas) != 3 || listadas[0].ID != aluguel.ID || listadas[1].ID != encerrada.ID || listadas[2].ID != energia.ID {
		t.Errorf("listadas = %d, esperado aluguel, contador e energia nessa ordem", len(listadas))
	}

	vigentes, err := c.recorrencias.ListarVigentes(ctx, competenciaDe(t, "2026-03"), nil)
	if err != nil {
		t.Fatalf("vigentes falhou: %v", err)
	}
	ids := map[uuid.UUID]bool{}
	for _, despesa := range vigentes {
		ids[despesa.ID] = true
	}
	if len(vigentes) != 2 || !ids[aluguel.ID] || !ids[daOutra.ID] {
		t.Errorf("vigentes em 2026-03 = %d, esperado aluguel e água", len(vigentes))
	}

	soDaMIA, err := c.recorrencias.ListarVigentes(ctx, competenciaDe(t, "2026-05"), []uuid.UUID{c.empresaID})
	if err != nil || len(soDaMIA) != 2 {
		t.Errorf("vigentes da MIA em 2026-05 = %d, %v; esperado aluguel e energia", len(soDaMIA), err)
	}

	if _, err := c.banco.ExecContext(ctx, `UPDATE empresa SET ativa = false WHERE id = $1`, outra); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	semInativa, err := c.recorrencias.ListarVigentes(ctx, competenciaDe(t, "2026-03"), nil)
	if err != nil || len(semInativa) != 1 || semInativa[0].ID != aluguel.ID {
		t.Errorf("vigentes sem a empresa inativa = %d, %v", len(semInativa), err)
	}
}

func TestGerarDeRecorrenciaEhIdempotentePorCompetencia(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()
	despesa := c.novaRecorrencia(t, c.empresaID, "Aluguel", "2026-01", nil)

	primeira, criada, err := c.gerar(t, despesa, "2026-03")
	if err != nil || !criada {
		t.Fatalf("primeira geração = %v, %v", criada, err)
	}
	if _, criadaDeNovo, err := c.gerar(t, despesa, "2026-03"); err != nil || criadaDeNovo {
		t.Errorf("segunda geração da mesma competência = %v, %v; esperado ignorada", criadaDeNovo, err)
	}
	if _, criadaAbril, err := c.gerar(t, despesa, "2026-04"); err != nil || !criadaAbril {
		t.Errorf("geração de outra competência = %v, %v", criadaAbril, err)
	}

	gravada := c.obter(t, primeira.ID)
	if gravada.Origem != domainsolicitacao.OrigemRecorrencia || *gravada.OrigemID != despesa.ID || gravada.Competencia.String() != "2026-03" {
		t.Errorf("origem = %q/%v, competência = %s", gravada.Origem, gravada.OrigemID, gravada.Competencia)
	}
	if gravada.Prazo.String() != "2026-04-30" || gravada.Status != domainsolicitacao.StatusPendenteAprovacao || !gravada.Estimado {
		t.Errorf("vencimento = %s, status = %q, estimado = %v", gravada.Prazo, gravada.Status, gravada.Estimado)
	}

	manuais, err := c.solicitacoes.Listar(ctx, portsout.SolicitacaoFiltro{PaginacaoFiltro: todasAsPaginas, Origem: domainsolicitacao.OrigemRecorrencia})
	if err != nil || len(manuais) != 2 {
		t.Errorf("solicitações da recorrência = %d, %v; esperado 2", len(manuais), err)
	}
	historico, err := c.solicitacoes.ListarHistorico(ctx, primeira.ID)
	if err != nil || len(historico) != 1 {
		t.Errorf("histórico = %d, %v; esperado 1", len(historico), err)
	}
}

func TestGerarDeRecorrenciaRecusaDespesaInativaOuCategoriaDesativada(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()

	inativa := c.novaRecorrencia(t, c.empresaID, "Seguro", "2026-01", nil)
	solicitacao, err := inativa.GerarSolicitacao(c.terceiroID, competenciaDe(t, "2026-03"), time.Now().UTC())
	if err != nil {
		t.Fatalf("geração no domínio falhou: %v", err)
	}
	inativa.Desativar(time.Now().UTC())
	if err := c.recorrencias.Desativar(ctx, inativa); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(solicitacao.ID, c.terceiroID, nil, solicitacao.Status, domainsolicitacao.EventoCriada, "x")
	if _, err := c.solicitacoes.GerarDeRecorrencia(ctx, solicitacao, historico); err == nil ||
		err.Error() != domainsolicitacao.ErroDespesaRecorrenteIndisponivel().Error() {
		t.Errorf("recorrência desativada = %v", err)
	}

	ativa := c.novaRecorrencia(t, c.empresaID, "Aluguel", "2026-01", nil)
	if _, err := c.banco.ExecContext(ctx, `UPDATE categoria_financeira SET ativa = false WHERE id = $1`, c.despesaID); err != nil {
		t.Fatalf("desativação da categoria falhou: %v", err)
	}
	if _, _, err := c.gerar(t, ativa, "2026-03"); err == nil || err.Error() != domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel().Error() {
		t.Errorf("categoria desativada = %v", err)
	}
}

func TestRemoverDespesaRecorrenteSoSemSolicitacaoGerada(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()

	livre := c.novaRecorrencia(t, c.empresaID, "Revista", "2026-01", nil)
	if err := c.recorrencias.Remover(ctx, livre.ID); err != nil {
		t.Errorf("remoção sem gerações falhou: %v", err)
	}
	if err := c.recorrencias.Remover(ctx, livre.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("remover de novo = %v, esperado não encontrado", err)
	}

	usada := c.novaRecorrencia(t, c.empresaID, "Aluguel", "2026-01", nil)
	if _, criada, err := c.gerar(t, usada, "2026-03"); err != nil || !criada {
		t.Fatalf("geração = %v, %v", criada, err)
	}
	if err := c.recorrencias.Remover(ctx, usada.ID); err == nil || err.Error() != domainsolicitacao.ErroDespesaRecorrenteComSolicitacoes().Error() {
		t.Errorf("remover com gerações = %v", err)
	}
}
