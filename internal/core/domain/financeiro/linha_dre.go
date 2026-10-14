package financeiro

import (
	"slices"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type LinhaDRE string

const (
	LinhaReceitaOperacional  LinhaDRE = "receita_operacional"
	LinhaDeducaoVenda        LinhaDRE = "deducao_venda"
	LinhaCompraMercadoria    LinhaDRE = "compra_mercadoria"
	LinhaDespesaVariavel     LinhaDRE = "despesa_variavel"
	LinhaDespesaFixa         LinhaDRE = "despesa_fixa"
	LinhaResultadoFinanceiro LinhaDRE = "resultado_financeiro"
	LinhaForaDRE             LinhaDRE = "fora_dre"
	LinhaEntreContas         LinhaDRE = "entre_contas"
)

var naturezasPorLinha = map[LinhaDRE][]Natureza{
	LinhaReceitaOperacional:  {NaturezaReceita},
	LinhaDeducaoVenda:        {NaturezaDespesa},
	LinhaCompraMercadoria:    {NaturezaDespesa},
	LinhaDespesaVariavel:     {NaturezaDespesa},
	LinhaDespesaFixa:         {NaturezaDespesa},
	LinhaResultadoFinanceiro: {NaturezaReceita, NaturezaDespesa},
	LinhaForaDRE:             {NaturezaReceita, NaturezaDespesa},
	LinhaEntreContas:         {NaturezaTransferencia},
}

func NovaLinhaDRE(valor string) (LinhaDRE, error) {
	linha := LinhaDRE(strings.ToLower(strings.TrimSpace(valor)))
	if err := linha.Validado(); err != nil {
		return "", err
	}

	return linha, nil
}

func LinhaDREDe(valor string) LinhaDRE {
	return LinhaDRE(valor)
}

func (l LinhaDRE) String() string {
	return string(l)
}

func (l LinhaDRE) Validado() error {
	if _, ok := naturezasPorLinha[l]; !ok {
		return domain.ErroValidacao("linha da DRE deve ser receita_operacional, deducao_venda, compra_mercadoria, " +
			"despesa_variavel, despesa_fixa, resultado_financeiro, fora_dre ou entre_contas")
	}

	return nil
}

func (l LinhaDRE) AceitaNatureza(natureza Natureza) bool {
	return slices.Contains(naturezasPorLinha[l], natureza)
}

type Classificacao struct {
	natureza Natureza
	linha    LinhaDRE
}

func NovaClassificacao(natureza, linha string) (Classificacao, error) {
	naturezaValida, err := NovaNatureza(natureza)
	if err != nil {
		return Classificacao{}, err
	}

	linhaValida, err := NovaLinhaDRE(linha)
	if err != nil {
		return Classificacao{}, err
	}

	if !linhaValida.AceitaNatureza(naturezaValida) {
		return Classificacao{}, domain.ErroValidacao("linha da DRE " + linhaValida.String() + " não aceita natureza " + naturezaValida.String())
	}

	return Classificacao{natureza: naturezaValida, linha: linhaValida}, nil
}

func (c Classificacao) Natureza() Natureza {
	return c.natureza
}

func (c Classificacao) LinhaDRE() LinhaDRE {
	return c.linha
}
