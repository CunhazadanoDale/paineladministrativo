package dre

import "github.com/google/uuid"

type Linha string

const (
	LinhaReceitaBruta        Linha = "receita_bruta"
	LinhaOutrasReceitas      Linha = "outras_receitas"
	LinhaDeducoes            Linha = "deducoes"
	LinhaCMV                 Linha = "cmv"
	LinhaPerdasEstoque       Linha = "perdas_estoque"
	LinhaDespesasVariaveis   Linha = "despesas_variaveis"
	LinhaDespesasFixas       Linha = "despesas_fixas"
	LinhaNaoClassificado     Linha = "nao_classificado"
	LinhaResultadoFinanceiro Linha = "resultado_financeiro"
	LinhaForaDRE             Linha = "fora_dre"
)

var ordemDasLinhas = []Linha{
	LinhaReceitaBruta,
	LinhaOutrasReceitas,
	LinhaDeducoes,
	LinhaCMV,
	LinhaPerdasEstoque,
	LinhaDespesasVariaveis,
	LinhaDespesasFixas,
	LinhaNaoClassificado,
	LinhaResultadoFinanceiro,
	LinhaForaDRE,
}

var linhasDoPlanoDeContas = map[string]Linha{
	"receita_operacional":  LinhaOutrasReceitas,
	"deducao_venda":        LinhaDeducoes,
	"despesa_variavel":     LinhaDespesasVariaveis,
	"despesa_fixa":         LinhaDespesasFixas,
	"resultado_financeiro": LinhaResultadoFinanceiro,
	"fora_dre":             LinhaForaDRE,
}

func Linhas() []Linha {
	linhas := make([]Linha, len(ordemDasLinhas))
	copy(linhas, ordemDasLinhas)

	return linhas
}

func LinhaDaCategoria(linhaDRE string) (Linha, bool) {
	linha, ok := linhasDoPlanoDeContas[linhaDRE]

	return linha, ok
}

func LinhasDoPlanoDeContas() []string {
	return []string{"receita_operacional", "deducao_venda", "despesa_variavel", "despesa_fixa", "resultado_financeiro", "fora_dre"}
}

func (l Linha) String() string {
	return string(l)
}

const ChaveCategoria = "categoria"

const (
	ItemNFVenda             = "nf_venda"
	ItemNFDevolucao         = "nf_devolucao"
	ItemCMV                 = "cmv"
	ItemPerdasEstoque       = "perdas_estoque"
	ItemJurosRecebidos      = "juros_recebidos"
	ItemDescontosConcedidos = "descontos_concedidos"
	ItemEncargosAntecipacao = "encargos_antecipacao"
	ItemSemCategoria        = "sem_categoria"
)

var nomesDosItensFixos = map[string]string{
	ItemNFVenda:             "Notas fiscais de venda",
	ItemNFDevolucao:         "Notas fiscais de devolução",
	ItemCMV:                 "Custo da mercadoria vendida",
	ItemPerdasEstoque:       "Perdas e ajustes de estoque",
	ItemJurosRecebidos:      "Juros recebidos",
	ItemDescontosConcedidos: "Descontos concedidos",
	ItemEncargosAntecipacao: "Encargos de antecipação",
	ItemSemCategoria:        "Contas a pagar sem categoria",
}

type Item struct {
	Chave       string
	CategoriaID *uuid.UUID
	Nome        string
}

func ItemFixo(chave string) Item {
	return Item{Chave: chave, Nome: nomesDosItensFixos[chave]}
}

func ItemDaCategoria(categoriaID uuid.UUID, nome string) Item {
	id := categoriaID

	return Item{Chave: ChaveCategoria, CategoriaID: &id, Nome: nome}
}

func (i Item) identificador() string {
	if i.CategoriaID != nil {
		return i.CategoriaID.String()
	}

	return i.Chave
}

type Lancamento struct {
	Mes           Mes
	EmpresaID     uuid.UUID
	Linha         Linha
	Item          Item
	ValorCentavos int64
	Quantidade    int
}

type ConsumoSemCusto struct {
	Mes        Mes
	EmpresaID  uuid.UUID
	Quantidade int64
}
