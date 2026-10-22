package dre

import (
	"math/big"
	"sort"

	"github.com/google/uuid"
)

const pontosBaseInteiros = 10000

type ItemDemonstrativo struct {
	Item
	ValorCentavos int64
	Quantidade    int
}

type LinhaDemonstrativo struct {
	Linha         Linha
	ValorCentavos int64
	Quantidade    int
	Itens         []ItemDemonstrativo
}

type Subtotais struct {
	ReceitaLiquidaCentavos       int64
	LucroBrutoCentavos           int64
	MargemContribuicaoCentavos   int64
	ResultadoOperacionalCentavos int64
	ResultadoCentavos            int64
}

type Indicadores struct {
	MargemContribuicaoBP    *int64
	PontoEquilibrioCentavos *int64
}

type Demonstrativo struct {
	Linhas             []LinhaDemonstrativo
	Subtotais          Subtotais
	Indicadores        Indicadores
	QuantidadeSemCusto int64
}

func (d Demonstrativo) Valor(linha Linha) int64 {
	for _, atual := range d.Linhas {
		if atual.Linha == linha {
			return atual.ValorCentavos
		}
	}

	return 0
}

type montagem struct {
	itens              map[Linha]map[string]*ItemDemonstrativo
	quantidadeSemCusto int64
}

func novaMontagem() *montagem {
	return &montagem{itens: make(map[Linha]map[string]*ItemDemonstrativo, len(ordemDasLinhas))}
}

func (m *montagem) somar(lancamento Lancamento) {
	itens, ok := m.itens[lancamento.Linha]
	if !ok {
		itens = make(map[string]*ItemDemonstrativo)
		m.itens[lancamento.Linha] = itens
	}

	chave := lancamento.Item.identificador()
	item, ok := itens[chave]
	if !ok {
		item = &ItemDemonstrativo{Item: lancamento.Item}
		itens[chave] = item
	}
	item.ValorCentavos += lancamento.ValorCentavos
	item.Quantidade += lancamento.Quantidade
}

func (m *montagem) fechar() Demonstrativo {
	demonstrativo := Demonstrativo{
		Linhas:             make([]LinhaDemonstrativo, 0, len(ordemDasLinhas)),
		QuantidadeSemCusto: m.quantidadeSemCusto,
	}

	for _, linha := range ordemDasLinhas {
		atual := LinhaDemonstrativo{Linha: linha, Itens: itensOrdenados(m.itens[linha])}
		for _, item := range atual.Itens {
			atual.ValorCentavos += item.ValorCentavos
			atual.Quantidade += item.Quantidade
		}
		demonstrativo.Linhas = append(demonstrativo.Linhas, atual)
	}

	demonstrativo.Subtotais = subtotaisDe(demonstrativo)
	demonstrativo.Indicadores = indicadoresDe(demonstrativo)

	return demonstrativo
}

func itensOrdenados(itens map[string]*ItemDemonstrativo) []ItemDemonstrativo {
	ordenados := make([]ItemDemonstrativo, 0, len(itens))
	for _, item := range itens {
		ordenados = append(ordenados, *item)
	}

	sort.Slice(ordenados, func(i, j int) bool {
		a, b := ordenados[i], ordenados[j]
		if (a.CategoriaID == nil) != (b.CategoriaID == nil) {
			return a.CategoriaID == nil
		}
		if a.Nome != b.Nome {
			return a.Nome < b.Nome
		}

		return a.identificador() < b.identificador()
	})

	return ordenados
}

func subtotaisDe(d Demonstrativo) Subtotais {
	receitaLiquida := d.Valor(LinhaReceitaBruta) + d.Valor(LinhaOutrasReceitas) + d.Valor(LinhaDeducoes)
	lucroBruto := receitaLiquida + d.Valor(LinhaCMV) + d.Valor(LinhaPerdasEstoque)
	margem := lucroBruto + d.Valor(LinhaDespesasVariaveis)
	operacional := margem + d.Valor(LinhaDespesasFixas) + d.Valor(LinhaNaoClassificado)

	return Subtotais{
		ReceitaLiquidaCentavos:       receitaLiquida,
		LucroBrutoCentavos:           lucroBruto,
		MargemContribuicaoCentavos:   margem,
		ResultadoOperacionalCentavos: operacional,
		ResultadoCentavos:            operacional + d.Valor(LinhaResultadoFinanceiro),
	}
}

func indicadoresDe(d Demonstrativo) Indicadores {
	receita := d.Subtotais.ReceitaLiquidaCentavos
	margem := d.Subtotais.MargemContribuicaoCentavos
	if receita <= 0 {
		return Indicadores{}
	}

	indicadores := Indicadores{
		MargemContribuicaoBP: dividirArredondado(
			new(big.Int).Mul(big.NewInt(margem), big.NewInt(pontosBaseInteiros)),
			big.NewInt(receita),
		),
	}
	if margem <= 0 {
		return indicadores
	}

	fixas := -d.Valor(LinhaDespesasFixas)
	if fixas < 0 {
		fixas = 0
	}
	indicadores.PontoEquilibrioCentavos = dividirArredondado(
		new(big.Int).Mul(big.NewInt(fixas), big.NewInt(receita)),
		big.NewInt(margem),
	)

	return indicadores
}

func dividirArredondado(dividendo, divisor *big.Int) *int64 {
	quociente, resto := new(big.Int).QuoRem(dividendo, divisor, new(big.Int))
	dobro := new(big.Int).Mul(new(big.Int).Abs(resto), big.NewInt(2))
	if dobro.Cmp(divisor) >= 0 {
		quociente.Add(quociente, big.NewInt(int64(dividendo.Sign())))
	}
	if !quociente.IsInt64() {
		return nil
	}

	valor := quociente.Int64()

	return &valor
}

type DemonstrativoEmpresa struct {
	EmpresaID uuid.UUID
	Demonstrativo
}

type MesDRE struct {
	Mes         Mes
	Empresas    []DemonstrativoEmpresa
	Consolidado Demonstrativo
}

type TotalDRE struct {
	Empresas    []DemonstrativoEmpresa
	Consolidado Demonstrativo
}

type ResumoDRE struct {
	De    Mes
	Ate   Mes
	Meses []MesDRE
	Total TotalDRE
}

type grupoDeMontagens struct {
	empresas    map[uuid.UUID]*montagem
	consolidado *montagem
}

func novoGrupoDeMontagens(empresas []uuid.UUID) *grupoDeMontagens {
	grupo := &grupoDeMontagens{
		empresas:    make(map[uuid.UUID]*montagem, len(empresas)),
		consolidado: novaMontagem(),
	}
	for _, empresaID := range empresas {
		grupo.empresas[empresaID] = novaMontagem()
	}

	return grupo
}

func (g *grupoDeMontagens) somar(lancamento Lancamento) {
	g.empresas[lancamento.EmpresaID].somar(lancamento)
	g.consolidado.somar(lancamento)
}

func (g *grupoDeMontagens) somarSemCusto(consumo ConsumoSemCusto) {
	g.empresas[consumo.EmpresaID].quantidadeSemCusto += consumo.Quantidade
	g.consolidado.quantidadeSemCusto += consumo.Quantidade
}

func (g *grupoDeMontagens) fechar(empresas []uuid.UUID) ([]DemonstrativoEmpresa, Demonstrativo) {
	demonstrativos := make([]DemonstrativoEmpresa, 0, len(empresas))
	for _, empresaID := range empresas {
		demonstrativos = append(demonstrativos, DemonstrativoEmpresa{
			EmpresaID:     empresaID,
			Demonstrativo: g.empresas[empresaID].fechar(),
		})
	}

	return demonstrativos, g.consolidado.fechar()
}

func NovoResumoDRE(periodo PeriodoMensal, empresasExigidas []uuid.UUID, lancamentos []Lancamento, semCusto []ConsumoSemCusto) ResumoDRE {
	meses := periodo.Meses()
	dentro := make(map[Mes]bool, len(meses))
	for _, mes := range meses {
		dentro[mes] = true
	}

	presentes := append([]uuid.UUID{}, empresasExigidas...)
	for _, lancamento := range lancamentos {
		if dentro[lancamento.Mes] {
			presentes = append(presentes, lancamento.EmpresaID)
		}
	}
	for _, consumo := range semCusto {
		if dentro[consumo.Mes] {
			presentes = append(presentes, consumo.EmpresaID)
		}
	}
	empresas := empresasOrdenadas(presentes)

	porMes := make(map[Mes]*grupoDeMontagens, len(meses))
	for _, mes := range meses {
		porMes[mes] = novoGrupoDeMontagens(empresas)
	}
	total := novoGrupoDeMontagens(empresas)

	for _, lancamento := range lancamentos {
		if grupo, ok := porMes[lancamento.Mes]; ok {
			grupo.somar(lancamento)
			total.somar(lancamento)
		}
	}
	for _, consumo := range semCusto {
		if grupo, ok := porMes[consumo.Mes]; ok {
			grupo.somarSemCusto(consumo)
			total.somarSemCusto(consumo)
		}
	}

	resumo := ResumoDRE{De: periodo.De(), Ate: periodo.Ate(), Meses: make([]MesDRE, 0, len(meses))}
	for _, mes := range meses {
		demonstrativos, consolidado := porMes[mes].fechar(empresas)
		resumo.Meses = append(resumo.Meses, MesDRE{Mes: mes, Empresas: demonstrativos, Consolidado: consolidado})
	}
	resumo.Total.Empresas, resumo.Total.Consolidado = total.fechar(empresas)

	return resumo
}

func empresasOrdenadas(ids []uuid.UUID) []uuid.UUID {
	vistas := make(map[uuid.UUID]bool, len(ids))
	unicas := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !vistas[id] {
			vistas[id] = true
			unicas = append(unicas, id)
		}
	}

	sort.Slice(unicas, func(i, j int) bool { return unicas[i].String() < unicas[j].String() })

	return unicas
}
