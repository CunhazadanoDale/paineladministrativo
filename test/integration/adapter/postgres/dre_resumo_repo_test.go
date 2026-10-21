package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsdre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoDRE struct {
	ctx       context.Context
	banco     *sqlx.DB
	resumos   *postgres.DREResumoRepository
	usuarioID uuid.UUID
	empresaA  uuid.UUID
	empresaB  uuid.UUID
	clienteID uuid.UUID
	produtoID uuid.UUID
	localA    uuid.UUID
	localA2   uuid.UUID
	contaA    uuid.UUID
	fomentoID uuid.UUID
	sequencia int
	categoria map[string]uuid.UUID
}

func brasilia(ano int, mes time.Month, dia, hora int) time.Time {
	return time.Date(ano, mes, dia, hora, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
}

func diaDRE(mes time.Month, dia int) time.Time {
	return time.Date(2026, mes, dia, 0, 0, 0, 0, time.UTC)
}

func novoCenarioDRE(t *testing.T) *cenarioBancoDRE {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	c := &cenarioBancoDRE{
		ctx:       context.Background(),
		banco:     banco,
		resumos:   postgres.NewDREResumoRepository(banco),
		usuarioID: uuid.New(),
		empresaA:  uuid.New(),
		empresaB:  uuid.New(),
		clienteID: uuid.New(),
		produtoID: uuid.New(),
		localA:    uuid.New(),
		localA2:   uuid.New(),
		contaA:    uuid.New(),
		fomentoID: uuid.New(),
		categoria: make(map[string]uuid.UUID),
	}
	cargoID, categoriaEstoque, subcategoriaEstoque := uuid.New(), uuid.New(), uuid.New()

	c.executar(t, `INSERT INTO cargo (id, nome) VALUES ($1, 'Financeiro')`, cargoID)
	c.executar(t, `INSERT INTO usuario (id, nome, email, senha, cargo_id) VALUES ($1, 'Ana', 'ana@exemplo.com', 'x', $2)`, c.usuarioID, cargoID)
	c.executar(t, `
		INSERT INTO empresa (id, razao_social, nome_fantasia, cnpj, uf, regime_tributario)
		VALUES ($1, 'MIA LTDA', 'MIA', '11222333000181', 'PB', 'presumido'),
		       ($2, 'Gigante LTDA', 'Gigante', '11444777000161', 'PB', 'presumido')
	`, c.empresaA, c.empresaB)
	c.executar(t, `
		INSERT INTO cliente (id, documento, tipo_pessoa, nome, nome_busca, tipo, uf, cidade)
		VALUES ($1, '12345678909', 'pf', 'João', 'joao', 'pf', 'PB', 'João Pessoa')
	`, c.clienteID)
	c.executar(t, `INSERT INTO categoria (id, nome, slug) VALUES ($1, 'Cimentos', 'cimentos')`, categoriaEstoque)
	c.executar(t, `INSERT INTO categoria (id, nome, slug, categoria_pai_id) VALUES ($1, 'CP II', 'cp-ii', $2)`, subcategoriaEstoque, categoriaEstoque)
	c.executar(t, `
		INSERT INTO produto (id, categoria_id, nome, slug, unidade_medida) VALUES ($1, $2, 'Cimento', 'cimento', 'un')
	`, c.produtoID, subcategoriaEstoque)
	c.executar(t, `
		INSERT INTO estoque_local (id, empresa_id, nome) VALUES ($1, $3, 'Depósito'), ($2, $3, 'Loja')
	`, c.localA, c.localA2, c.empresaA)
	c.executar(t, `
		INSERT INTO conta_bancaria (id, empresa_id, banco, tipo, agencia, conta, apelido)
		VALUES ($1, $2, '748', 'corrente', '0101', '12345', 'Sicredi')
	`, c.contaA, c.empresaA)
	c.executar(t, `INSERT INTO fornecedor (id, pais, nome, nome_busca) VALUES ($1, 'US', 'Fomento', 'fomento')`, c.fomentoID)

	for _, categoria := range []struct{ nome, natureza, linha string }{
		{"Despesas fixas", "despesa", "despesa_fixa"},
		{"Financeiro", "despesa", "resultado_financeiro"},
		{"Mercadorias", "despesa", "compra_mercadoria"},
		{"Outras receitas", "receita", "receita_operacional"},
		{"Captação", "receita", "fora_dre"},
		{"Transferências", "transferencia", "entre_contas"},
	} {
		raiz, sub := uuid.New(), uuid.New()
		c.executar(t, `
			INSERT INTO categoria_financeira (id, nome, natureza, linha_dre) VALUES ($1, $2, $3, $4)
		`, raiz, categoria.nome, categoria.natureza, categoria.linha)
		c.executar(t, `
			INSERT INTO categoria_financeira (id, nome, categoria_pai_id, natureza, linha_dre) VALUES ($1, $2, $3, $4, $5)
		`, sub, "Sub de "+categoria.nome, raiz, categoria.natureza, categoria.linha)
		c.categoria[categoria.nome] = sub
		c.categoria["raiz "+categoria.nome] = raiz
	}

	return c
}

func (c *cenarioBancoDRE) executar(t *testing.T, query string, argumentos ...any) {
	t.Helper()

	if _, err := c.banco.ExecContext(c.ctx, query, argumentos...); err != nil {
		t.Fatalf("fixture falhou: %v\n%s", err, query)
	}
}

func (c *cenarioBancoDRE) proximo() int {
	c.sequencia++

	return c.sequencia
}

func (c *cenarioBancoDRE) nota(t *testing.T, empresaID uuid.UUID, natureza, tipoOperacao, status string, emitidaEm time.Time, valor int64) {
	t.Helper()

	numero := c.proximo()
	arquivoID := uuid.New()
	c.executar(t, `
		INSERT INTO arquivo (id, proprietario_id, nome, chave, content_type, tamanho)
		VALUES ($1, $2, 'nota.xml', $3, 'application/xml', 1)
	`, arquivoID, c.usuarioID, fmt.Sprintf("nota-%d", numero))

	finalidade := "normal"
	if natureza == "devolucao" {
		finalidade = "devolucao"
	}
	entreEmpresas := natureza == "transferencia"
	var clienteID, destinatariaID *uuid.UUID
	if entreEmpresas {
		destinatariaID = &c.empresaB
	} else {
		clienteID = &c.clienteID
	}

	c.executar(t, `
		INSERT INTO nota_fiscal (
			chave, empresa_id, numero, serie, emitida_em, tipo_operacao, finalidade, natureza, natureza_operacao,
			entre_empresas, cliente_id, empresa_destinataria_id, destinatario_documento, destinatario_nome,
			cfop_predominante, valor_total_centavos, valor_produtos_centavos, valor_frete_centavos,
			valor_desconto_centavos, protocolo, autorizada_em, status, arquivo_id, usuario_id
		) VALUES (
			$1, $2, $3, 1, $4, $5, $6, $7, 'Operação', $8, $9, $10, '12345678909', 'João',
			'5102', $11, $11, 0, 0, $12, $4, $13, $14, $15
		)
	`, fmt.Sprintf("%044d", numero), empresaID, numero, emitidaEm, tipoOperacao, finalidade, natureza,
		entreEmpresas, clienteID, destinatariaID, valor, fmt.Sprintf("%015d", numero), status, arquivoID, c.usuarioID)
}

func (c *cenarioBancoDRE) movimento(t *testing.T, tipo string, quantidade int, localID uuid.UUID, destinoID *uuid.UUID, criadoEm time.Time) uuid.UUID {
	t.Helper()

	id := uuid.New()
	c.executar(t, `
		INSERT INTO produto_movimento (id, produto_id, tipo, quantidade, saldo_apos, usuario_id, local_id, local_destino_id, saldo_local_apos, criado_em)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, 0, $8)
	`, id, c.produtoID, tipo, quantidade, c.usuarioID, localID, destinoID, criadoEm)

	return id
}

func (c *cenarioBancoDRE) lote(t *testing.T, custo *int64) uuid.UUID {
	t.Helper()

	entrada := c.movimento(t, "entrada", 50, c.localA, nil, brasilia(2026, time.January, 5, 10))
	id := uuid.New()
	c.executar(t, `
		INSERT INTO produto_lote (id, produto_id, local_id, movimento_id, custo_unitario_centavos, quantidade_inicial, quantidade_atual, entrada_em)
		VALUES ($1, $2, $3, $4, $5, 50, 50, $6)
	`, id, c.produtoID, c.localA, entrada, custo, brasilia(2026, time.January, 5, 10))

	return id
}

func (c *cenarioBancoDRE) consumir(t *testing.T, movimentoID, loteID uuid.UUID, quantidade int, custo *int64) {
	t.Helper()

	c.executar(t, `
		INSERT INTO produto_movimento_lote (movimento_id, lote_id, quantidade, custo_unitario_centavos) VALUES ($1, $2, $3, $4)
	`, movimentoID, loteID, quantidade, custo)
}

func (c *cenarioBancoDRE) conta(t *testing.T, status string, competencia time.Time, categoria *uuid.UUID, valor int64, origem string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	var origemID *uuid.UUID
	if origem != "manual" {
		outro := uuid.New()
		origemID = &outro
	}
	var motivo *string
	if status == "rejeitado" {
		texto := "fora do orçamento"
		motivo = &texto
	}
	c.executar(t, `
		INSERT INTO solicitacao (
			id, solicitante_id, valor_estimado, prazo_pagamento, observacao, forma_pagamento, status, rejeitacao_motivo,
			empresa_id, competencia, categoria_financeira_id, origem, origem_id
		) VALUES ($1, $2, $3, $4, 'Conta', 'pix', $5, $6, $7, $4, $8, $9, $10)
	`, id, c.usuarioID, valor, competencia, status, motivo, c.empresaA, categoria, origem, origemID)

	return id
}

func (c *cenarioBancoDRE) pagamento(t *testing.T, solicitacaoID uuid.UUID, valor int64, estornado bool) {
	t.Helper()

	c.executar(t, `
		INSERT INTO pagamento (solicitacao_id, valor, pago_em, conta_bancaria_id, usuario_id, estornado)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, solicitacaoID, valor, brasilia(2026, time.September, 20, 10), c.contaA, c.usuarioID, estornado)
}

func (c *cenarioBancoDRE) titulo(t *testing.T, origem, status string, competencia time.Time, categoria *uuid.UUID, valor int64) uuid.UUID {
	t.Helper()

	id := uuid.New()
	var origemID *uuid.UUID
	if origem != "manual" {
		outro := uuid.New()
		origemID = &outro
	}
	var motivo *string
	if status == "cancelado" {
		texto := "lançado errado"
		motivo = &texto
	}
	c.executar(t, `
		INSERT INTO titulo_receber (
			id, empresa_id, cliente_id, origem, origem_id, numero, parcela, vencimento, valor_centavos,
			categoria_financeira_id, competencia, status, cancelado_motivo, usuario_id
		) VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, $9, $7, $10, $11, $12)
	`, id, c.empresaA, c.clienteID, origem, origemID, fmt.Sprintf("T-%d", c.proximo()), competencia, valor,
		categoria, status, motivo, c.usuarioID)

	return id
}

func (c *cenarioBancoDRE) recebimento(t *testing.T, tituloID uuid.UUID, recebidoEm time.Time, juros, desconto int64, estornado bool) {
	t.Helper()

	c.executar(t, `
		INSERT INTO titulo_recebimento (
			titulo_id, tipo, conta_bancaria_id, recebido_em, valor_centavos, juros_centavos, desconto_centavos,
			status_titulo_anterior, estornado, usuario_id
		) VALUES ($1, 'conta', $2, $3, 1000, $4, $5, 'carteira', $6, $7)
	`, tituloID, c.contaA, recebidoEm, juros, desconto, estornado, c.usuarioID)
}

func (c *cenarioBancoDRE) antecipacao(t *testing.T, operadoEm time.Time, bruto, liquido int64, estornada bool) {
	t.Helper()

	status := "ativa"
	var motivo *string
	var estornadoEm *time.Time
	if estornada {
		status = "estornada"
		texto := "lançada errada"
		agora := time.Now()
		motivo, estornadoEm = &texto, &agora
	}
	c.executar(t, `
		INSERT INTO antecipacao (
			empresa_id, fornecedor_id, operado_em, valor_bruto_centavos, valor_liquido_centavos, conta_bancaria_id,
			status, estorno_motivo, estornado_em, usuario_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, c.empresaA, c.fomentoID, operadoEm, bruto, liquido, c.contaA, status, motivo, estornadoEm, c.usuarioID)
}

func (c *cenarioBancoDRE) lancamentoCaixa(t *testing.T, competencia time.Time, categoria uuid.UUID, valor int64, estornado bool) {
	t.Helper()

	c.executar(t, `
		INSERT INTO lancamento_caixa (
			empresa_id, conta_bancaria_id, data, valor_centavos, categoria_financeira_id, competencia, descricao,
			estornado, usuario_id
		) VALUES ($1, $2, $3, $4, $5, $3, 'Linha do extrato', $6, $7)
	`, c.empresaA, c.contaA, competencia, valor, categoria, estornado, c.usuarioID)
}

func custoDRE(centavos int64) *int64 {
	return &centavos
}

func (c *cenarioBancoDRE) filtro(empresas []uuid.UUID) portsdre.FiltroResumo {
	return portsdre.FiltroResumo{
		Intervalo:  domaindre.Intervalo{Inicio: diaDRE(time.September, 1), Fim: diaDRE(time.November, 1)},
		EmpresaIDs: empresas,
	}
}

type totalDRE struct {
	valor      int64
	quantidade int
}

func somaDRE(lancamentos []domaindre.Lancamento, mes string, empresaID uuid.UUID, linha domaindre.Linha, item string) totalDRE {
	var total totalDRE
	for _, lancamento := range lancamentos {
		identificador := lancamento.Item.Chave
		if lancamento.Item.CategoriaID != nil {
			identificador = lancamento.Item.CategoriaID.String()
		}
		if lancamento.Mes.String() == mes && lancamento.EmpresaID == empresaID && lancamento.Linha == linha && identificador == item {
			total.valor += lancamento.ValorCentavos
			total.quantidade += lancamento.Quantidade
		}
	}

	return total
}

func TestDREResumoLeCadaFonteUmaVez(t *testing.T) {
	c := novoCenarioDRE(t)

	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.September, 30, 23), 100_000)
	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.October, 1, 1), 50_000)
	c.nota(t, c.empresaA, "venda", "saida", "cancelada", brasilia(2026, time.September, 10, 10), 7_000)
	c.nota(t, c.empresaA, "transferencia", "saida", "autorizada", brasilia(2026, time.September, 10, 10), 8_000)
	c.nota(t, c.empresaA, "devolucao", "entrada", "autorizada", brasilia(2026, time.September, 12, 10), 10_000)
	c.nota(t, c.empresaA, "devolucao", "saida", "autorizada", brasilia(2026, time.September, 12, 10), 9_000)
	c.nota(t, c.empresaA, "remessa", "saida", "autorizada", brasilia(2026, time.September, 12, 10), 6_000)
	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.August, 31, 23), 5_000)
	c.nota(t, c.empresaB, "venda", "saida", "autorizada", brasilia(2026, time.September, 15, 10), 30_000)

	comCusto, semCusto := c.lote(t, custoDRE(100)), c.lote(t, nil)
	saida := c.movimento(t, "saida", -5, c.localA, nil, brasilia(2026, time.September, 30, 22))
	c.consumir(t, saida, comCusto, 3, custoDRE(100))
	c.consumir(t, saida, semCusto, 2, nil)
	perda := c.movimento(t, "perda", -1, c.localA, nil, brasilia(2026, time.September, 3, 10))
	c.consumir(t, perda, comCusto, 1, custoDRE(100))
	ajuste := c.movimento(t, "ajuste", -1, c.localA, nil, brasilia(2026, time.September, 4, 10))
	c.consumir(t, ajuste, comCusto, 1, custoDRE(100))
	transferencia := c.movimento(t, "transferencia", 4, c.localA, &c.localA2, brasilia(2026, time.September, 5, 10))
	c.consumir(t, transferencia, comCusto, 4, custoDRE(100))
	outubro := c.movimento(t, "saida", -1, c.localA, nil, brasilia(2026, time.October, 1, 0))
	c.consumir(t, outubro, comCusto, 1, custoDRE(100))

	fixa, financeira := c.categoria["Despesas fixas"], c.categoria["Financeiro"]
	mercadoria := c.categoria["Mercadorias"]
	paga := c.conta(t, "pago", diaDRE(time.September, 1), &fixa, 2_000, "manual")
	c.pagamento(t, paga, 2_100, false)
	reaberta := c.conta(t, "aprovado", diaDRE(time.September, 1), &fixa, 1_000, "manual")
	c.pagamento(t, reaberta, 999, true)
	c.conta(t, "pendente_aprovacao", diaDRE(time.September, 1), &fixa, 500, "recorrencia")
	c.conta(t, "cancelado", diaDRE(time.September, 1), &fixa, 7_777, "manual")
	c.conta(t, "rejeitado", diaDRE(time.September, 1), &fixa, 8_888, "manual")
	c.conta(t, "aprovado", diaDRE(time.September, 1), &financeira, 300, "manual")
	c.conta(t, "aprovado", diaDRE(time.September, 1), &mercadoria, 99_999, "importacao_parcela")
	c.conta(t, "aprovado", diaDRE(time.September, 1), nil, 400, "manual")
	c.conta(t, "aprovado", diaDRE(time.October, 1), &fixa, 1_000, "manual")

	servicos := c.categoria["Outras receitas"]
	c.titulo(t, "manual", "carteira", diaDRE(time.September, 1), &servicos, 700)
	c.titulo(t, "manual", "cancelado", diaDRE(time.September, 1), &servicos, 7_000)
	daNota := c.titulo(t, "nota_fiscal_duplicata", "carteira", diaDRE(time.September, 1), nil, 100_000)
	c.recebimento(t, daNota, diaDRE(time.September, 15), 50, 20, false)
	c.recebimento(t, daNota, diaDRE(time.September, 16), 999, 999, true)
	c.recebimento(t, daNota, diaDRE(time.September, 17), 0, 0, false)
	c.recebimento(t, daNota, diaDRE(time.October, 2), 10, 0, false)

	c.antecipacao(t, diaDRE(time.September, 10), 10_000, 9_500, false)
	c.antecipacao(t, diaDRE(time.September, 11), 10_000, 9_000, true)

	c.lancamentoCaixa(t, diaDRE(time.September, 1), financeira, -150, false)
	c.lancamentoCaixa(t, diaDRE(time.September, 1), c.categoria["Captação"], 5_000, false)
	c.lancamentoCaixa(t, diaDRE(time.September, 1), financeira, -999, true)
	c.lancamentoCaixa(t, diaDRE(time.September, 1), c.categoria["Transferências"], -777, false)

	lancamentos, err := c.resumos.Lancamentos(c.ctx, c.filtro(nil))
	if err != nil {
		t.Fatalf("Lancamentos = %v", err)
	}

	esperados := []struct {
		mes     string
		empresa uuid.UUID
		linha   domaindre.Linha
		item    string
		total   totalDRE
	}{
		{"2026-09", c.empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemNFVenda, totalDRE{100_000, 1}},
		{"2026-10", c.empresaA, domaindre.LinhaReceitaBruta, domaindre.ItemNFVenda, totalDRE{50_000, 1}},
		{"2026-09", c.empresaB, domaindre.LinhaReceitaBruta, domaindre.ItemNFVenda, totalDRE{30_000, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaDeducoes, domaindre.ItemNFDevolucao, totalDRE{-10_000, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaCMV, domaindre.ItemCMV, totalDRE{-300, 1}},
		{"2026-10", c.empresaA, domaindre.LinhaCMV, domaindre.ItemCMV, totalDRE{-100, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaPerdasEstoque, domaindre.ItemPerdasEstoque, totalDRE{-200, 2}},
		{"2026-09", c.empresaA, domaindre.LinhaDespesasFixas, c.categoria["raiz Despesas fixas"].String(), totalDRE{-3_600, 3}},
		{"2026-10", c.empresaA, domaindre.LinhaDespesasFixas, c.categoria["raiz Despesas fixas"].String(), totalDRE{-1_000, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaNaoClassificado, domaindre.ItemSemCategoria, totalDRE{-400, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaResultadoFinanceiro, c.categoria["raiz Financeiro"].String(), totalDRE{-450, 2}},
		{"2026-09", c.empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemJurosRecebidos, totalDRE{50, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemDescontosConcedidos, totalDRE{-20, 1}},
		{"2026-10", c.empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemJurosRecebidos, totalDRE{10, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaResultadoFinanceiro, domaindre.ItemEncargosAntecipacao, totalDRE{-500, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaOutrasReceitas, c.categoria["raiz Outras receitas"].String(), totalDRE{700, 1}},
		{"2026-09", c.empresaA, domaindre.LinhaForaDRE, c.categoria["raiz Captação"].String(), totalDRE{5_000, 1}},
	}

	var somaEsperada int64
	for _, esperado := range esperados {
		obtido := somaDRE(lancamentos, esperado.mes, esperado.empresa, esperado.linha, esperado.item)
		if obtido != esperado.total {
			t.Errorf("%s %s %s: obtido %+v, esperado %+v", esperado.mes, esperado.linha, esperado.item, obtido, esperado.total)
		}
		somaEsperada += esperado.total.valor
	}

	var somaObtida int64
	for _, lancamento := range lancamentos {
		somaObtida += lancamento.ValorCentavos
		if lancamento.Item.CategoriaID != nil && lancamento.Item.Nome == "" {
			t.Errorf("item de categoria sem nome: %+v", lancamento)
		}
	}
	if somaObtida != somaEsperada {
		t.Errorf("soma de todos os lançamentos = %d, esperado %d: alguma exclusão vazou", somaObtida, somaEsperada)
	}

	consumos, err := c.resumos.ConsumosSemCusto(c.ctx, c.filtro(nil))
	if err != nil || len(consumos) != 1 || consumos[0].Quantidade != 2 || consumos[0].Mes.String() != "2026-09" || consumos[0].EmpresaID != c.empresaA {
		t.Errorf("consumos sem custo = %+v, %v", consumos, err)
	}

	faturamento, err := c.resumos.Faturamento(c.ctx, c.filtro(nil))
	if err != nil || len(faturamento) != 3 {
		t.Fatalf("faturamento = %+v, %v", faturamento, err)
	}
	for _, linha := range faturamento {
		if linha.Mes.String() == "2026-09" && linha.EmpresaID == c.empresaA && (linha.Quantidade != 1 || linha.ValorCentavos != 100_000) {
			t.Errorf("faturamento de setembro da empresa A = %+v", linha)
		}
	}
}

func TestDREResumoRespeitaOEscopoDeEmpresas(t *testing.T) {
	c := novoCenarioDRE(t)
	fixa := c.categoria["Despesas fixas"]

	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.September, 10, 10), 100)
	c.nota(t, c.empresaB, "venda", "saida", "autorizada", brasilia(2026, time.September, 10, 10), 200)
	c.conta(t, "aprovado", diaDRE(time.September, 1), &fixa, 300, "manual")

	soB, err := c.resumos.Lancamentos(c.ctx, c.filtro([]uuid.UUID{c.empresaB}))
	if err != nil || len(soB) != 1 || soB[0].EmpresaID != c.empresaB || soB[0].ValorCentavos != 200 {
		t.Errorf("lançamentos da empresa B = %+v, %v", soB, err)
	}

	nenhuma, err := c.resumos.Lancamentos(c.ctx, c.filtro([]uuid.UUID{}))
	if err != nil || len(nenhuma) != 0 {
		t.Errorf("sem empresas no acesso = %+v, %v", nenhuma, err)
	}

	faturamento, err := c.resumos.Faturamento(c.ctx, c.filtro([]uuid.UUID{c.empresaA}))
	if err != nil || len(faturamento) != 1 || faturamento[0].ValorCentavos != 100 {
		t.Errorf("faturamento da empresa A = %+v, %v", faturamento, err)
	}
}
