package dre

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

const formatoData = "2006-01-02"

func DataDeConsulta(texto, campo string) (*time.Time, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return nil, nil
	}

	data, err := time.Parse(formatoData, texto)
	if err != nil {
		return nil, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM-DD")
	}

	return &data, nil
}

func dataOpcional(data *time.Time) *string {
	if data == nil {
		return nil
	}

	texto := data.Format(formatoData)

	return &texto
}

type TotalSemanalResponse struct {
	Quantidade    int   `json:"quantidade"`
	ValorCentavos int64 `json:"valor_centavos"`
}

func novoTotalSemanal(total domaindre.TotalSemanal) TotalSemanalResponse {
	return TotalSemanalResponse{Quantidade: total.Quantidade, ValorCentavos: total.ValorCentavos}
}

type NotasEmitidasResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	Natureza  string    `json:"natureza"`
	TotalSemanalResponse
}

type BlocoNotasResponse struct {
	TotalVenda TotalSemanalResponse    `json:"total_venda"`
	PorEmpresa []NotasEmitidasResponse `json:"por_empresa"`
}

type PagamentosFeitosResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	LinhaDRE  *string   `json:"linha_dre"`
	TotalSemanalResponse
}

type BlocoPagamentosResponse struct {
	Total      TotalSemanalResponse       `json:"total"`
	PorEmpresa []PagamentosFeitosResponse `json:"por_empresa"`
}

type RecebimentosFeitosResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	Tipo      string    `json:"tipo"`
	TotalSemanalResponse
	JurosCentavos int64 `json:"juros_centavos"`
}

type BlocoRecebimentosResponse struct {
	Total      TotalSemanalResponse         `json:"total"`
	PorEmpresa []RecebimentosFeitosResponse `json:"por_empresa"`
}

type VencidosPorEmpresaResponse struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	TotalSemanalResponse
}

type TituloVencidoResponse struct {
	ID            uuid.UUID `json:"id"`
	EmpresaID     uuid.UUID `json:"empresa_id"`
	ClienteNome   string    `json:"cliente_nome"`
	Numero        string    `json:"numero"`
	Parcela       int       `json:"parcela"`
	Status        string    `json:"status"`
	Vencimento    string    `json:"vencimento"`
	SaldoCentavos int64     `json:"saldo_centavos"`
}

type BlocoTitulosVencidosResponse struct {
	Total      TotalSemanalResponse         `json:"total"`
	PorEmpresa []VencidosPorEmpresaResponse `json:"por_empresa"`
	Itens      []TituloVencidoResponse      `json:"itens"`
}

type ContaVencidaResponse struct {
	ID             uuid.UUID `json:"id"`
	EmpresaID      uuid.UUID `json:"empresa_id"`
	FornecedorNome *string   `json:"fornecedor_nome"`
	Observacao     string    `json:"observacao"`
	Status         string    `json:"status"`
	Vencimento     string    `json:"vencimento"`
	ValorCentavos  int64     `json:"valor_centavos"`
}

type BlocoContasVencidasResponse struct {
	Total      TotalSemanalResponse         `json:"total"`
	PorEmpresa []VencidosPorEmpresaResponse `json:"por_empresa"`
	Itens      []ContaVencidaResponse       `json:"itens"`
}

type ProdutoEstoqueBaixoResponse struct {
	ProdutoID       uuid.UUID `json:"produto_id"`
	Nome            string    `json:"nome"`
	Codigo          *string   `json:"codigo"`
	QuantidadeAtual int       `json:"quantidade_atual"`
	EstoqueMinimo   int       `json:"estoque_minimo"`
}

type BlocoEstoqueBaixoResponse struct {
	Total int                           `json:"total"`
	Itens []ProdutoEstoqueBaixoResponse `json:"itens"`
}

type SaldoBancarioResponse struct {
	ContaBancariaID    uuid.UUID `json:"conta_bancaria_id"`
	EmpresaID          uuid.UUID `json:"empresa_id"`
	Apelido            string    `json:"apelido"`
	Banco              string    `json:"banco"`
	BancoNome          string    `json:"banco_nome"`
	Ativa              bool      `json:"ativa"`
	DataCorte          *string   `json:"data_corte"`
	SaldoAtualCentavos *int64    `json:"saldo_atual_centavos"`
	UltimoLancamentoEm *string   `json:"ultimo_lancamento_em"`
}

type BlocoSaldosResponse struct {
	SaldoTotalCentavos int64                   `json:"saldo_total_centavos"`
	ContasSemAbertura  int                     `json:"contas_sem_abertura"`
	Contas             []SaldoBancarioResponse `json:"contas"`
}

type RelatorioSemanalResponse struct {
	Inicio          string                       `json:"inicio"`
	Fim             string                       `json:"fim"`
	Hoje            string                       `json:"hoje"`
	NotasEmitidas   BlocoNotasResponse           `json:"notas_emitidas"`
	Pagamentos      BlocoPagamentosResponse      `json:"pagamentos"`
	Recebimentos    BlocoRecebimentosResponse    `json:"recebimentos"`
	TitulosVencidos BlocoTitulosVencidosResponse `json:"titulos_vencidos"`
	ContasVencidas  BlocoContasVencidasResponse  `json:"contas_vencidas"`
	EstoqueBaixo    BlocoEstoqueBaixoResponse    `json:"estoque_baixo"`
	SaldosBancarios BlocoSaldosResponse          `json:"saldos_bancarios"`
	Endividamento   BlocoEndividamentoResponse   `json:"endividamento"`
}

func NovoRelatorioSemanalResponse(relatorio domaindre.RelatorioSemanal) RelatorioSemanalResponse {
	resposta := RelatorioSemanalResponse{
		Inicio: relatorio.Inicio.Format(formatoData),
		Fim:    relatorio.Fim.Format(formatoData),
		Hoje:   relatorio.Hoje.Format(formatoData),
		NotasEmitidas: BlocoNotasResponse{
			TotalVenda: novoTotalSemanal(relatorio.TotalNotasVenda),
			PorEmpresa: make([]NotasEmitidasResponse, 0, len(relatorio.Notas)),
		},
		Pagamentos: BlocoPagamentosResponse{
			Total:      novoTotalSemanal(relatorio.TotalPagamentos),
			PorEmpresa: make([]PagamentosFeitosResponse, 0, len(relatorio.Pagamentos)),
		},
		Recebimentos: BlocoRecebimentosResponse{
			Total:      novoTotalSemanal(relatorio.TotalRecebido),
			PorEmpresa: make([]RecebimentosFeitosResponse, 0, len(relatorio.Recebimentos)),
		},
		TitulosVencidos: BlocoTitulosVencidosResponse{
			Total:      novoTotalSemanal(relatorio.TitulosVencidos.Total),
			PorEmpresa: novosVencidosPorEmpresa(relatorio.TitulosVencidos.PorEmpresa),
			Itens:      make([]TituloVencidoResponse, 0, len(relatorio.TitulosVencidos.Itens)),
		},
		ContasVencidas: BlocoContasVencidasResponse{
			Total:      novoTotalSemanal(relatorio.ContasVencidas.Total),
			PorEmpresa: novosVencidosPorEmpresa(relatorio.ContasVencidas.PorEmpresa),
			Itens:      make([]ContaVencidaResponse, 0, len(relatorio.ContasVencidas.Itens)),
		},
		EstoqueBaixo: BlocoEstoqueBaixoResponse{
			Total: relatorio.EstoqueBaixo.Total,
			Itens: make([]ProdutoEstoqueBaixoResponse, 0, len(relatorio.EstoqueBaixo.Itens)),
		},
		SaldosBancarios: BlocoSaldosResponse{
			SaldoTotalCentavos: relatorio.Saldos.SaldoTotalCentavos,
			ContasSemAbertura:  relatorio.Saldos.ContasSemAbertura,
			Contas:             novosSaldosBancarios(relatorio.Saldos.Contas),
		},
		Endividamento: novoEndividamento(relatorio.Endividamento),
	}

	for _, notas := range relatorio.Notas {
		resposta.NotasEmitidas.PorEmpresa = append(resposta.NotasEmitidas.PorEmpresa, NotasEmitidasResponse{
			EmpresaID: notas.EmpresaID, Natureza: notas.Natureza, TotalSemanalResponse: novoTotalSemanal(notas.Total),
		})
	}
	for _, pagamentos := range relatorio.Pagamentos {
		resposta.Pagamentos.PorEmpresa = append(resposta.Pagamentos.PorEmpresa, PagamentosFeitosResponse{
			EmpresaID: pagamentos.EmpresaID, LinhaDRE: pagamentos.LinhaDRE, TotalSemanalResponse: novoTotalSemanal(pagamentos.Total),
		})
	}
	for _, recebimentos := range relatorio.Recebimentos {
		resposta.Recebimentos.PorEmpresa = append(resposta.Recebimentos.PorEmpresa, RecebimentosFeitosResponse{
			EmpresaID: recebimentos.EmpresaID, Tipo: recebimentos.Tipo,
			TotalSemanalResponse: novoTotalSemanal(recebimentos.Total), JurosCentavos: recebimentos.JurosCentavos,
		})
	}
	for _, titulo := range relatorio.TitulosVencidos.Itens {
		resposta.TitulosVencidos.Itens = append(resposta.TitulosVencidos.Itens, TituloVencidoResponse{
			ID: titulo.ID, EmpresaID: titulo.EmpresaID, ClienteNome: titulo.ClienteNome, Numero: titulo.Numero,
			Parcela: titulo.Parcela, Status: titulo.Status, Vencimento: titulo.Vencimento.Format(formatoData),
			SaldoCentavos: titulo.SaldoCentavos,
		})
	}
	for _, conta := range relatorio.ContasVencidas.Itens {
		resposta.ContasVencidas.Itens = append(resposta.ContasVencidas.Itens, ContaVencidaResponse{
			ID: conta.ID, EmpresaID: conta.EmpresaID, FornecedorNome: conta.FornecedorNome, Observacao: conta.Observacao,
			Status: conta.Status, Vencimento: conta.Vencimento.Format(formatoData), ValorCentavos: conta.ValorCentavos,
		})
	}
	for _, produto := range relatorio.EstoqueBaixo.Itens {
		resposta.EstoqueBaixo.Itens = append(resposta.EstoqueBaixo.Itens, ProdutoEstoqueBaixoResponse{
			ProdutoID: produto.ID, Nome: produto.Nome, Codigo: produto.Codigo,
			QuantidadeAtual: produto.QuantidadeAtual, EstoqueMinimo: produto.EstoqueMinimo,
		})
	}

	return resposta
}

func novosVencidosPorEmpresa(linhas []domaindre.VencidosPorEmpresa) []VencidosPorEmpresaResponse {
	respostas := make([]VencidosPorEmpresaResponse, 0, len(linhas))
	for _, linha := range linhas {
		respostas = append(respostas, VencidosPorEmpresaResponse{EmpresaID: linha.EmpresaID, TotalSemanalResponse: novoTotalSemanal(linha.Total)})
	}

	return respostas
}

func novosSaldosBancarios(contas []*domaincaixa.SaldoConta) []SaldoBancarioResponse {
	respostas := make([]SaldoBancarioResponse, 0, len(contas))
	for _, conta := range contas {
		respostas = append(respostas, SaldoBancarioResponse{
			ContaBancariaID:    conta.ContaBancariaID,
			EmpresaID:          conta.EmpresaID,
			Apelido:            conta.Apelido,
			Banco:              conta.Banco.String(),
			BancoNome:          conta.Banco.Nome(),
			Ativa:              conta.Ativa,
			DataCorte:          dataOpcional(conta.DataCorte),
			SaldoAtualCentavos: conta.SaldoAtualCentavos(),
			UltimoLancamentoEm: dataOpcional(conta.UltimoLancamentoEm),
		})
	}

	return respostas
}

type TotaisDividaResponse struct {
	AmortizacaoCentavos int64 `json:"amortizacao_centavos"`
	JurosCentavos       int64 `json:"juros_centavos"`
	Parcelas            int   `json:"parcelas"`
}

type SaldoDoCredorResponse struct {
	CredorTipo           string `json:"credor_tipo"`
	SaldoDevedorCentavos int64  `json:"saldo_devedor_centavos"`
	Dividas              int    `json:"dividas"`
}

type SaldoDaEmpresaResponse struct {
	EmpresaID            uuid.UUID `json:"empresa_id"`
	SaldoDevedorCentavos int64     `json:"saldo_devedor_centavos"`
	Dividas              int       `json:"dividas"`
}

type BlocoEndividamentoResponse struct {
	SaldoDevedorCentavos int64                    `json:"saldo_devedor_centavos"`
	Dividas              int                      `json:"dividas"`
	PorCredor            []SaldoDoCredorResponse  `json:"por_credor"`
	PorEmpresa           []SaldoDaEmpresaResponse `json:"por_empresa"`
	Vencido              TotaisDividaResponse     `json:"vencido"`
	AVencer              TotaisDividaResponse     `json:"a_vencer"`
	AVencer30            TotaisDividaResponse     `json:"a_vencer_30"`
	AVencer90            TotaisDividaResponse     `json:"a_vencer_90"`
	AVencer365           TotaisDividaResponse     `json:"a_vencer_365"`
}

func novoEndividamento(resumo domaindivida.Resumo) BlocoEndividamentoResponse {
	resposta := BlocoEndividamentoResponse{
		SaldoDevedorCentavos: resumo.SaldoDevedorCentavos,
		Dividas:              resumo.Dividas,
		PorCredor:            make([]SaldoDoCredorResponse, 0, len(resumo.PorCredor)),
		PorEmpresa:           make([]SaldoDaEmpresaResponse, 0, len(resumo.PorEmpresa)),
		Vencido:              novosTotaisDivida(resumo.Vencido),
		AVencer:              novosTotaisDivida(resumo.AVencer),
		AVencer30:            novosTotaisDivida(resumo.AVencer30),
		AVencer90:            novosTotaisDivida(resumo.AVencer90),
		AVencer365:           novosTotaisDivida(resumo.AVencer365),
	}

	for _, credor := range resumo.PorCredor {
		resposta.PorCredor = append(resposta.PorCredor, SaldoDoCredorResponse{
			CredorTipo: credor.CredorTipo.String(), SaldoDevedorCentavos: credor.SaldoDevedorCentavos, Dividas: credor.Dividas,
		})
	}
	for _, empresa := range resumo.PorEmpresa {
		resposta.PorEmpresa = append(resposta.PorEmpresa, SaldoDaEmpresaResponse{
			EmpresaID: empresa.EmpresaID, SaldoDevedorCentavos: empresa.SaldoDevedorCentavos, Dividas: empresa.Dividas,
		})
	}

	return resposta
}

func novosTotaisDivida(totais domaindivida.Totais) TotaisDividaResponse {
	return TotaisDividaResponse{
		AmortizacaoCentavos: totais.AmortizacaoCentavos,
		JurosCentavos:       totais.JurosCentavos,
		Parcelas:            totais.Parcelas,
	}
}
