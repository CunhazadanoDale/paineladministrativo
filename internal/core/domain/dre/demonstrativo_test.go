package dre_test

import (
	"testing"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

var (
	empresaA = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	empresaB = uuid.MustParse("00000000-0000-0000-0000-00000000000b")
)

func lancamento(m domaindre.Mes, empresa uuid.UUID, linha domaindre.Linha, item domaindre.Item, valor int64) domaindre.Lancamento {
	return domaindre.Lancamento{Mes: m, EmpresaID: empresa, Linha: linha, Item: item, ValorCentavos: valor, Quantidade: 1}
}

func periodoDeUmMes(t *testing.T, texto string) domaindre.PeriodoMensal {
	t.Helper()
	m := mes(t, texto)

	return domaindre.PeriodoMensalDe(m, m)
}

func TestLinhaDaCategoriaMapeiaOPlanoDeContas(t *testing.T) {
	casos := map[string]domaindre.Linha{
		"receita_operacional":  domaindre.LinhaOutrasReceitas,
		"deducao_venda":        domaindre.LinhaDeducoes,
		"despesa_variavel":     domaindre.LinhaDespesasVariaveis,
		"despesa_fixa":         domaindre.LinhaDespesasFixas,
		"resultado_financeiro": domaindre.LinhaResultadoFinanceiro,
		"fora_dre":             domaindre.LinhaForaDRE,
	}
	for linhaDRE, esperada := range casos {
		if linha, ok := domaindre.LinhaDaCategoria(linhaDRE); !ok || linha != esperada {
			t.Errorf("LinhaDaCategoria(%q) = %s, %v; esperado %s", linhaDRE, linha, ok, esperada)
		}
	}
	for _, foraDoResultado := range []string{"compra_mercadoria", "entre_contas", "outra"} {
		if _, ok := domaindre.LinhaDaCategoria(foraDoResultado); ok {
			t.Errorf("LinhaDaCategoria(%q) deveria ficar fora da DRE", foraDoResultado)
		}
	}
	if len(domaindre.LinhasDoPlanoDeContas()) != len(casos) {
		t.Errorf("LinhasDoPlanoDeContas deveria listar as %d linhas mapeadas", len(casos))
	}
}

func TestDemonstrativoCalculaSubtotaisComOSinalDeCadaLinha(t *testing.T) {
	m := mes(t, "2026-09")
	aluguel := domaindre.ItemDaCategoria(uuid.New(), "Aluguel")
	lancamentos := []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemFixo(domaindre.ItemNFVenda), 100_000_00),
		lancamento(m, empresaA, domaindre.LinhaOutrasReceitas, domaindre.ItemDaCategoria(uuid.New(), "Serviços"), 5_000_00),
		lancamento(m, empresaA, domaindre.LinhaDeducoes, domaindre.ItemFixo(domaindre.ItemNFDevolucao), -10_000_00),
		lancamento(m, empresaA, domaindre.LinhaCMV, domaindre.ItemFixo(domaindre.ItemCMV), -40_000_00),
		lancamento(m, empresaA, domaindre.LinhaPerdasEstoque, domaindre.ItemFixo(domaindre.ItemPerdasEstoque), -1_000_00),
		lancamento(m, empresaA, domaindre.LinhaDespesasVariaveis, domaindre.ItemDaCategoria(uuid.New(), "Frete"), -4_000_00),
		lancamento(m, empresaA, domaindre.LinhaDespesasFixas, aluguel, -20_000_00),
		lancamento(m, empresaA, domaindre.LinhaNaoClassificado, domaindre.ItemFixo(domaindre.ItemSemCategoria), -2_000_00),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemFixo(domaindre.ItemJurosRecebidos), 500_00),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemFixo(domaindre.ItemEncargosAntecipacao), -1_500_00),
		lancamento(m, empresaA, domaindre.LinhaForaDRE, domaindre.ItemDaCategoria(uuid.New(), "Empréstimos"), 300_000_00),
	}

	resumo := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, lancamentos, nil)
	demonstrativo := resumo.Meses[0].Empresas[0].Demonstrativo

	if len(demonstrativo.Linhas) != len(domaindre.Linhas()) {
		t.Fatalf("linhas = %d, esperado todas as %d", len(demonstrativo.Linhas), len(domaindre.Linhas()))
	}
	for i, linha := range domaindre.Linhas() {
		if demonstrativo.Linhas[i].Linha != linha {
			t.Errorf("linha %d = %s, esperado %s", i, demonstrativo.Linhas[i].Linha, linha)
		}
	}

	subtotais := demonstrativo.Subtotais
	if subtotais.ReceitaLiquidaCentavos != 95_000_00 {
		t.Errorf("receita líquida = %d, esperado 9500000", subtotais.ReceitaLiquidaCentavos)
	}
	if subtotais.LucroBrutoCentavos != 54_000_00 {
		t.Errorf("lucro bruto = %d, esperado 5400000", subtotais.LucroBrutoCentavos)
	}
	if subtotais.MargemContribuicaoCentavos != 50_000_00 {
		t.Errorf("margem de contribuição = %d, esperado 5000000", subtotais.MargemContribuicaoCentavos)
	}
	if subtotais.ResultadoOperacionalCentavos != 28_000_00 {
		t.Errorf("resultado operacional = %d, esperado 2800000", subtotais.ResultadoOperacionalCentavos)
	}
	if subtotais.ResultadoCentavos != 27_000_00 {
		t.Errorf("resultado = %d, esperado 2700000 (fora_dre não entra)", subtotais.ResultadoCentavos)
	}
	if demonstrativo.Valor(domaindre.LinhaForaDRE) != 300_000_00 {
		t.Errorf("fora_dre = %d, esperado 30000000 à parte", demonstrativo.Valor(domaindre.LinhaForaDRE))
	}

	indicadores := demonstrativo.Indicadores
	if indicadores.MargemContribuicaoBP == nil || *indicadores.MargemContribuicaoBP != 5263 {
		t.Errorf("margem de contribuição = %v, esperado 5263 bp", indicadores.MargemContribuicaoBP)
	}
	if indicadores.PontoEquilibrioCentavos == nil || *indicadores.PontoEquilibrioCentavos != 38_000_00 {
		t.Errorf("ponto de equilíbrio = %v, esperado 3800000", indicadores.PontoEquilibrioCentavos)
	}
}

func TestDemonstrativoAgrupaItensPorCategoriaEPoeOsFixosPrimeiro(t *testing.T) {
	m := mes(t, "2026-09")
	energia := uuid.New()
	lancamentos := []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemDaCategoria(energia, "Tarifas"), -100),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemDaCategoria(energia, "Tarifas"), -50),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemDaCategoria(uuid.New(), "Rendimentos"), 30),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemFixo(domaindre.ItemJurosRecebidos), 20),
	}

	resumo := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, lancamentos, nil)
	financeiro := resumo.Meses[0].Consolidado.Linhas[8]

	if financeiro.ValorCentavos != -100 || financeiro.Quantidade != 4 {
		t.Errorf("resultado financeiro = %d (%d), esperado -100 (4)", financeiro.ValorCentavos, financeiro.Quantidade)
	}
	if len(financeiro.Itens) != 3 {
		t.Fatalf("itens = %d, esperado 3", len(financeiro.Itens))
	}
	if financeiro.Itens[0].Chave != domaindre.ItemJurosRecebidos || financeiro.Itens[0].Nome != "Juros recebidos" {
		t.Errorf("primeiro item = %+v, esperado juros recebidos", financeiro.Itens[0])
	}
	if financeiro.Itens[1].Nome != "Rendimentos" || financeiro.Itens[2].Nome != "Tarifas" {
		t.Errorf("categorias = %s, %s; esperado em ordem alfabética", financeiro.Itens[1].Nome, financeiro.Itens[2].Nome)
	}
	if financeiro.Itens[2].ValorCentavos != -150 || *financeiro.Itens[2].CategoriaID != energia {
		t.Errorf("tarifas = %+v, esperado -150 somado na mesma categoria", financeiro.Itens[2])
	}
}

func TestIndicadoresFicamNulosSemReceitaOuSemMargem(t *testing.T) {
	m := mes(t, "2026-09")

	semReceita := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaDespesasFixas, domaindre.ItemDaCategoria(uuid.New(), "Aluguel"), -1000),
	}, nil)
	indicadores := semReceita.Meses[0].Consolidado.Indicadores
	if indicadores.MargemContribuicaoBP != nil || indicadores.PontoEquilibrioCentavos != nil {
		t.Errorf("sem receita os indicadores deveriam ser nulos: %+v", indicadores)
	}

	margemNegativa := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemFixo(domaindre.ItemNFVenda), 1000),
		lancamento(m, empresaA, domaindre.LinhaCMV, domaindre.ItemFixo(domaindre.ItemCMV), -1500),
	}, nil)
	indicadores = margemNegativa.Meses[0].Consolidado.Indicadores
	if indicadores.MargemContribuicaoBP == nil || *indicadores.MargemContribuicaoBP != -5000 {
		t.Errorf("margem = %v, esperado -5000 bp", indicadores.MargemContribuicaoBP)
	}
	if indicadores.PontoEquilibrioCentavos != nil {
		t.Errorf("com margem negativa o ponto de equilíbrio deveria ser nulo")
	}

	semFixas := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemFixo(domaindre.ItemNFVenda), 1000),
	}, nil)
	indicadores = semFixas.Meses[0].Consolidado.Indicadores
	if indicadores.PontoEquilibrioCentavos == nil || *indicadores.PontoEquilibrioCentavos != 0 {
		t.Errorf("sem despesas fixas o ponto de equilíbrio deveria ser zero: %v", indicadores.PontoEquilibrioCentavos)
	}
}

func TestPontoEquilibrioIgnoraFinanceiroENaoClassificado(t *testing.T) {
	m := mes(t, "2026-09")

	resumo := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, []domaindre.Lancamento{
		lancamento(m, empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemFixo(domaindre.ItemNFVenda), 3000),
		lancamento(m, empresaA, domaindre.LinhaCMV, domaindre.ItemFixo(domaindre.ItemCMV), -2000),
		lancamento(m, empresaA, domaindre.LinhaDespesasFixas, domaindre.ItemDaCategoria(uuid.New(), "Aluguel"), -100),
		lancamento(m, empresaA, domaindre.LinhaNaoClassificado, domaindre.ItemFixo(domaindre.ItemSemCategoria), -900),
		lancamento(m, empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemFixo(domaindre.ItemEncargosAntecipacao), -900),
	}, nil)

	pontoEquilibrio := resumo.Meses[0].Consolidado.Indicadores.PontoEquilibrioCentavos
	if pontoEquilibrio == nil || *pontoEquilibrio != 300 {
		t.Errorf("ponto de equilíbrio = %v, esperado 300", pontoEquilibrio)
	}
}

func TestResumoDREPreencheTodosOsMesesEEmpresasEConsolida(t *testing.T) {
	setembro, outubro := mes(t, "2026-09"), mes(t, "2026-10")
	periodo := domaindre.PeriodoMensalDe(setembro, outubro)
	venda := domaindre.ItemFixo(domaindre.ItemNFVenda)
	lancamentos := []domaindre.Lancamento{
		lancamento(setembro, empresaB, domaindre.LinhaReceitaBruta, venda, 700),
		lancamento(outubro, empresaA, domaindre.LinhaReceitaBruta, venda, 300),
		lancamento(mes(t, "2026-11"), empresaA, domaindre.LinhaReceitaBruta, venda, 9999),
	}
	semCusto := []domaindre.ConsumoSemCusto{
		{Mes: outubro, EmpresaID: empresaA, Quantidade: 4},
		{Mes: mes(t, "2025-01"), EmpresaID: empresaB, Quantidade: 99},
	}
	empresaC := uuid.MustParse("00000000-0000-0000-0000-00000000000c")

	resumo := domaindre.NovoResumoDRE(periodo, []uuid.UUID{empresaC}, lancamentos, semCusto)

	if resumo.De != setembro || resumo.Ate != outubro || len(resumo.Meses) != 2 {
		t.Fatalf("resumo = %s a %s com %d meses", resumo.De, resumo.Ate, len(resumo.Meses))
	}
	for _, linhaMes := range resumo.Meses {
		if len(linhaMes.Empresas) != 3 {
			t.Fatalf("%s: empresas = %d, esperado 3", linhaMes.Mes, len(linhaMes.Empresas))
		}
		if linhaMes.Empresas[0].EmpresaID != empresaA || linhaMes.Empresas[1].EmpresaID != empresaB || linhaMes.Empresas[2].EmpresaID != empresaC {
			t.Errorf("%s: empresas fora de ordem", linhaMes.Mes)
		}
	}

	if resumo.Meses[0].Consolidado.Valor(domaindre.LinhaReceitaBruta) != 700 {
		t.Errorf("setembro consolidado = %d, esperado 700", resumo.Meses[0].Consolidado.Valor(domaindre.LinhaReceitaBruta))
	}
	if resumo.Meses[0].Empresas[0].Valor(domaindre.LinhaReceitaBruta) != 0 {
		t.Errorf("empresa A em setembro deveria estar zerada")
	}
	if resumo.Meses[1].Consolidado.QuantidadeSemCusto != 4 || resumo.Meses[1].Empresas[0].QuantidadeSemCusto != 4 {
		t.Errorf("quantidade sem custo de outubro = %d, esperado 4", resumo.Meses[1].Consolidado.QuantidadeSemCusto)
	}

	if resumo.Total.Consolidado.Valor(domaindre.LinhaReceitaBruta) != 1000 {
		t.Errorf("total = %d, esperado 1000 sem o mês fora do período", resumo.Total.Consolidado.Valor(domaindre.LinhaReceitaBruta))
	}
	if resumo.Total.Consolidado.QuantidadeSemCusto != 4 {
		t.Errorf("total sem custo = %d, esperado 4", resumo.Total.Consolidado.QuantidadeSemCusto)
	}
	if resumo.Total.Empresas[1].Valor(domaindre.LinhaReceitaBruta) != 700 {
		t.Errorf("total da empresa B = %d, esperado 700", resumo.Total.Empresas[1].Valor(domaindre.LinhaReceitaBruta))
	}
}

func TestResumoDRESemDadosDevolveListasVazias(t *testing.T) {
	resumo := domaindre.NovoResumoDRE(periodoDeUmMes(t, "2026-09"), nil, nil, nil)

	if resumo.Meses[0].Empresas == nil || len(resumo.Meses[0].Empresas) != 0 {
		t.Errorf("empresas deveria ser lista vazia")
	}
	if resumo.Total.Empresas == nil {
		t.Errorf("total por empresa deveria ser lista vazia")
	}
	for _, linha := range resumo.Meses[0].Consolidado.Linhas {
		if linha.Itens == nil || linha.ValorCentavos != 0 {
			t.Errorf("linha %s deveria vir zerada com itens vazios", linha.Linha)
		}
	}
}
