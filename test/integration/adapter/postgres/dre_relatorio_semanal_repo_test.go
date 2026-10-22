package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsdre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	"github.com/google/uuid"
)

func (c *cenarioBancoDRE) pagamentoEm(t *testing.T, solicitacaoID uuid.UUID, valor int64, pagoEm time.Time, estornado bool) {
	t.Helper()

	c.executar(t, `
		INSERT INTO pagamento (solicitacao_id, valor, pago_em, conta_bancaria_id, usuario_id, estornado)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, solicitacaoID, valor, pagoEm, c.contaA, c.usuarioID, estornado)
}

func filtroDaSemana(empresas []uuid.UUID) portsdre.FiltroSemanal {
	return portsdre.FiltroSemanal{
		Intervalo:  domaindre.SemanaDe(diaDRE(time.September, 16)).Intervalo(),
		Hoje:       diaDRE(time.September, 20),
		EmpresaIDs: empresas,
		Limite:     2,
	}
}

func TestRelatorioSemanalLeOsBlocosDaSemanaEOsVencidosDeHoje(t *testing.T) {
	c := novoCenarioDRE(t)
	relatorios := postgres.NewDRERelatorioSemanalRepository(c.banco)
	filtro := filtroDaSemana(nil)

	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.September, 15, 10), 1_000)
	c.nota(t, c.empresaA, "remessa", "saida", "autorizada", brasilia(2026, time.September, 16, 10), 400)
	c.nota(t, c.empresaA, "venda", "saida", "cancelada", brasilia(2026, time.September, 16, 10), 9_000)
	c.nota(t, c.empresaA, "venda", "saida", "autorizada", brasilia(2026, time.September, 21, 0), 8_000)
	c.nota(t, c.empresaB, "venda", "saida", "autorizada", brasilia(2026, time.September, 14, 0), 500)

	notas, err := relatorios.NotasEmitidas(c.ctx, filtro)
	if err != nil || len(notas) != 3 {
		t.Fatalf("notas = %+v, %v", notas, err)
	}
	for _, linha := range notas {
		esperado := map[string]int64{"venda": 1_000, "remessa": 400}[linha.Natureza]
		if linha.EmpresaID == c.empresaB {
			esperado = 500
		}
		if linha.Total.Quantidade != 1 || linha.Total.ValorCentavos != esperado {
			t.Errorf("notas %s da empresa %s = %+v", linha.Natureza, linha.EmpresaID, linha.Total)
		}
	}
	if soB, _ := relatorios.NotasEmitidas(c.ctx, filtroDaSemana([]uuid.UUID{c.empresaB})); len(soB) != 1 || soB[0].EmpresaID != c.empresaB {
		t.Errorf("notas da empresa B = %+v", soB)
	}

	fixa := c.categoria["Despesas fixas"]
	paga := c.conta(t, "pago", diaDRE(time.September, 1), &fixa, 2_000, "manual")
	c.pagamentoEm(t, paga, 2_100, brasilia(2026, time.September, 20, 23), false)
	estornada := c.conta(t, "aprovado", diaDRE(time.September, 1), &fixa, 1_000, "manual")
	c.pagamentoEm(t, estornada, 999, brasilia(2026, time.September, 15, 10), true)
	semCategoria := c.conta(t, "pago", diaDRE(time.September, 1), nil, 300, "manual")
	c.pagamentoEm(t, semCategoria, 300, brasilia(2026, time.September, 14, 0), false)
	foraDaSemana := c.conta(t, "pago", diaDRE(time.September, 1), &fixa, 700, "manual")
	c.pagamentoEm(t, foraDaSemana, 700, brasilia(2026, time.September, 21, 0), false)

	pagamentos, err := relatorios.PagamentosFeitos(c.ctx, filtro)
	if err != nil || len(pagamentos) != 2 {
		t.Fatalf("pagamentos = %+v, %v", pagamentos, err)
	}
	if pagamentos[0].LinhaDRE == nil || *pagamentos[0].LinhaDRE != "despesa_fixa" || pagamentos[0].Total.ValorCentavos != 2_100 {
		t.Errorf("pagamento classificado = %+v", pagamentos[0])
	}
	if pagamentos[1].LinhaDRE != nil || pagamentos[1].Total.ValorCentavos != 300 {
		t.Errorf("pagamento sem categoria = %+v", pagamentos[1])
	}

	daNota := c.titulo(t, "nota_fiscal_duplicata", "carteira", diaDRE(time.September, 1), nil, 1_000)
	c.recebimento(t, daNota, diaDRE(time.September, 15), 50, 0, false)
	c.recebimento(t, daNota, diaDRE(time.September, 16), 70, 0, true)
	c.recebimento(t, daNota, diaDRE(time.September, 21), 0, 0, false)

	recebimentos, err := relatorios.RecebimentosFeitos(c.ctx, filtro)
	if err != nil || len(recebimentos) != 1 || recebimentos[0].Tipo != "conta" || recebimentos[0].Total.Quantidade != 1 ||
		recebimentos[0].Total.ValorCentavos != 1_000 || recebimentos[0].JurosCentavos != 50 {
		t.Errorf("recebimentos = %+v, %v", recebimentos, err)
	}

	parcial := c.titulo(t, "nota_fiscal_duplicata", "carteira", diaDRE(time.September, 1), nil, 1_000)
	c.executar(t, `UPDATE titulo_receber SET status = 'em_banco', recebido_centavos = 400, conta_bancaria_id = $2 WHERE id = $1`, parcial, c.contaA)
	c.titulo(t, "nota_fiscal_duplicata", "carteira", diaDRE(time.October, 1), nil, 5_000)
	c.titulo(t, "manual", "cancelado", diaDRE(time.September, 1), &fixa, 5_000)

	titulos, err := relatorios.TitulosVencidos(c.ctx, filtro)
	if err != nil {
		t.Fatalf("TitulosVencidos = %v", err)
	}
	if titulos.Total != (domaindre.TotalSemanal{Quantidade: 2, ValorCentavos: 1_600}) || len(titulos.PorEmpresa) != 1 {
		t.Errorf("títulos vencidos = %+v", titulos)
	}
	if len(titulos.Itens) != 2 || titulos.Itens[0].ClienteNome != "João" || titulos.Itens[0].Parcela != 1 {
		t.Errorf("lista de títulos vencidos = %+v", titulos.Itens)
	}

	c.conta(t, "pendente_aprovacao", diaDRE(time.September, 1), &fixa, 100, "manual")
	comFornecedor := c.conta(t, "aprovado", diaDRE(time.August, 1), &fixa, 50, "manual")
	c.executar(t, `UPDATE solicitacao SET fornecedor_id = $2 WHERE id = $1`, comFornecedor, c.fomentoID)
	c.conta(t, "aprovado", diaDRE(time.October, 1), &fixa, 9_000, "manual")

	contas, err := relatorios.ContasVencidas(c.ctx, filtro)
	if err != nil {
		t.Fatalf("ContasVencidas = %v", err)
	}
	if contas.Total != (domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 1_150}) {
		t.Errorf("contas vencidas = %+v, esperado a estornada, a pendente e a de agosto", contas.Total)
	}
	if len(contas.Itens) != 2 || contas.Itens[0].ID != comFornecedor || contas.Itens[0].FornecedorNome == nil || *contas.Itens[0].FornecedorNome != "Fomento" {
		t.Errorf("lista limitada de contas vencidas = %+v", contas.Itens)
	}

	c.executar(t, `UPDATE produto SET estoque_minimo = 10, quantidade_atual = 5 WHERE id = $1`, c.produtoID)
	subcategoria := uuid.New()
	c.executar(t, `INSERT INTO categoria (id, nome, slug) VALUES ($1, 'Areias', 'areias')`, subcategoria)
	c.executar(t, `
		INSERT INTO produto (categoria_id, nome, slug, unidade_medida, estoque_minimo, quantidade_atual, ativo)
		VALUES ($1, 'Areia inativa', 'areia-inativa', 'm3', 10, 0, false),
		       ($1, 'Areia sem mínimo', 'areia-sem-minimo', 'm3', NULL, 0, true),
		       ($1, 'Areia fina', 'areia-fina', 'm3', 3, 3, true),
		       ($1, 'Brita', 'brita', 'm3', 2, 9, true),
		       ($1, 'Argamassa', 'argamassa', 'sc', 4, 1, true)
	`, subcategoria)

	estoque, err := relatorios.EstoqueBaixo(c.ctx, 2)
	if err != nil || estoque.Total != 3 || len(estoque.Itens) != 2 {
		t.Fatalf("estoque baixo = %+v, %v", estoque, err)
	}
	if estoque.Itens[0].Nome != "Areia fina" || estoque.Itens[1].Nome != "Argamassa" || estoque.Itens[1].QuantidadeAtual != 1 || estoque.Itens[1].EstoqueMinimo != 4 {
		t.Errorf("itens de estoque baixo = %+v", estoque.Itens)
	}
}
