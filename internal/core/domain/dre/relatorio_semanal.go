package dre

import (
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	"github.com/google/uuid"
)

const (
	LimiteDasListasDoRelatorio = 50
	TipoRecebimentoNaConta     = "conta"
)

type TotalSemanal struct {
	Quantidade    int
	ValorCentavos int64
}

func (t TotalSemanal) Somar(outro TotalSemanal) TotalSemanal {
	return TotalSemanal{Quantidade: t.Quantidade + outro.Quantidade, ValorCentavos: t.ValorCentavos + outro.ValorCentavos}
}

type NotasEmitidas struct {
	EmpresaID uuid.UUID
	Natureza  string
	Total     TotalSemanal
}

type PagamentosFeitos struct {
	EmpresaID uuid.UUID
	LinhaDRE  *string
	Total     TotalSemanal
}

type RecebimentosFeitos struct {
	EmpresaID     uuid.UUID
	Tipo          string
	Total         TotalSemanal
	JurosCentavos int64
}

type VencidosPorEmpresa struct {
	EmpresaID uuid.UUID
	Total     TotalSemanal
}

type TituloVencido struct {
	ID            uuid.UUID
	EmpresaID     uuid.UUID
	ClienteNome   string
	Numero        string
	Parcela       int
	Status        string
	Vencimento    time.Time
	SaldoCentavos int64
}

type ContaVencida struct {
	ID             uuid.UUID
	EmpresaID      uuid.UUID
	FornecedorNome *string
	Observacao     string
	Status         string
	Vencimento     time.Time
	ValorCentavos  int64
}

type ProdutoEstoqueBaixo struct {
	ID              uuid.UUID
	Nome            string
	Codigo          *string
	QuantidadeAtual int
	EstoqueMinimo   int
}

type Vencidos[T any] struct {
	PorEmpresa []VencidosPorEmpresa
	Itens      []T
	Total      TotalSemanal
}

func NovosVencidos[T any](porEmpresa []VencidosPorEmpresa, itens []T) Vencidos[T] {
	vencidos := Vencidos[T]{PorEmpresa: listaNaoNula(porEmpresa), Itens: listaNaoNula(itens)}
	for _, linha := range vencidos.PorEmpresa {
		vencidos.Total = vencidos.Total.Somar(linha.Total)
	}

	return vencidos
}

type EstoqueBaixo struct {
	Total int
	Itens []ProdutoEstoqueBaixo
}

type SaldosBancarios struct {
	Contas             []*domaincaixa.SaldoConta
	SaldoTotalCentavos int64
	ContasSemAbertura  int
}

type RelatorioSemanal struct {
	Inicio          time.Time
	Fim             time.Time
	Hoje            time.Time
	Notas           []NotasEmitidas
	TotalNotasVenda TotalSemanal
	Pagamentos      []PagamentosFeitos
	TotalPagamentos TotalSemanal
	Recebimentos    []RecebimentosFeitos
	TotalRecebido   TotalSemanal
	TitulosVencidos Vencidos[TituloVencido]
	ContasVencidas  Vencidos[ContaVencida]
	EstoqueBaixo    EstoqueBaixo
	Saldos          SaldosBancarios
	Endividamento   domaindivida.Resumo
}

type ConteudoDoRelatorio struct {
	Notas           []NotasEmitidas
	Pagamentos      []PagamentosFeitos
	Recebimentos    []RecebimentosFeitos
	TitulosVencidos Vencidos[TituloVencido]
	ContasVencidas  Vencidos[ContaVencida]
	EstoqueBaixo    EstoqueBaixo
	Saldos          []*domaincaixa.SaldoConta
	Parcelas        []domaindivida.ParcelaEmAberto
}

func NovoRelatorioSemanal(semana Semana, hoje time.Time, conteudo ConteudoDoRelatorio) RelatorioSemanal {
	relatorio := RelatorioSemanal{
		Inicio:          semana.Inicio(),
		Fim:             semana.Fim(),
		Hoje:            hoje,
		Notas:           listaNaoNula(conteudo.Notas),
		Pagamentos:      listaNaoNula(conteudo.Pagamentos),
		Recebimentos:    listaNaoNula(conteudo.Recebimentos),
		TitulosVencidos: NovosVencidos(conteudo.TitulosVencidos.PorEmpresa, conteudo.TitulosVencidos.Itens),
		ContasVencidas:  NovosVencidos(conteudo.ContasVencidas.PorEmpresa, conteudo.ContasVencidas.Itens),
		EstoqueBaixo:    EstoqueBaixo{Total: conteudo.EstoqueBaixo.Total, Itens: listaNaoNula(conteudo.EstoqueBaixo.Itens)},
		Saldos:          SaldosBancarios{Contas: listaNaoNula(conteudo.Saldos)},
		Endividamento:   domaindivida.NovoResumo(conteudo.Parcelas, hoje),
	}

	for _, notas := range relatorio.Notas {
		if notas.Natureza == "venda" {
			relatorio.TotalNotasVenda = relatorio.TotalNotasVenda.Somar(notas.Total)
		}
	}
	for _, pagamentos := range relatorio.Pagamentos {
		relatorio.TotalPagamentos = relatorio.TotalPagamentos.Somar(pagamentos.Total)
	}
	for _, recebimentos := range relatorio.Recebimentos {
		if recebimentos.Tipo == TipoRecebimentoNaConta {
			relatorio.TotalRecebido = relatorio.TotalRecebido.Somar(recebimentos.Total)
		}
	}
	for _, conta := range relatorio.Saldos.Contas {
		saldo := conta.SaldoAtualCentavos()
		if saldo == nil {
			relatorio.Saldos.ContasSemAbertura++
			continue
		}
		relatorio.Saldos.SaldoTotalCentavos += *saldo
	}

	return relatorio
}

func listaNaoNula[T any](itens []T) []T {
	if itens == nil {
		return []T{}
	}

	return itens
}
