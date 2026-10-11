package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

type cenarioBancoLocais struct {
	*cenarioBancoEstoque
	ctx     context.Context
	locais  *postgres.LocalRepository
	estoque *postgres.EstoqueRepository
	lotes   *postgres.LoteRepository
	mia     uuid.UUID
	red     uuid.UUID
	produto *domainestoque.Produto
}

func novoCenarioLocais(t *testing.T) *cenarioBancoLocais {
	t.Helper()

	base := cenarioEstoque(t)
	empresas := postgres.NewEmpresaRepository(base.banco)
	raiz := base.novaCategoriaRaiz(t, "Materiais")
	sub := base.novaSubcategoria(t, "Cimento", raiz)

	return &cenarioBancoLocais{
		cenarioBancoEstoque: base,
		ctx:                 context.Background(),
		locais:              postgres.NewLocalRepository(base.banco),
		estoque:             postgres.NewEstoqueRepository(base.banco),
		lotes:               postgres.NewLoteRepository(base.banco),
		mia:                 novaEmpresaNoBanco(t, empresas, "11.222.333/0001-81", "MIA"),
		red:                 novaEmpresaNoBanco(t, empresas, "33.400.004/0001-16", "Red Diamond"),
		produto:             base.novoProduto(t, sub, "Cimento CP II"),
	}
}

func novaEmpresaNoBanco(t *testing.T, empresas *postgres.EmpresaRepository, cnpj, nome string) uuid.UUID {
	t.Helper()

	documento, err := domainempresa.NovoCNPJ(cnpj)
	if err != nil {
		t.Fatalf("CNPJ de teste inválido: %v", err)
	}
	empresa, err := domainempresa.NovaEmpresa(documento, domainempresa.DadosEmpresa{
		RazaoSocial:      nome + " LTDA",
		NomeFantasia:     nome,
		UF:               domainempresa.UFDe("PB"),
		RegimeTributario: domainempresa.RegimePresumido,
	})
	if err != nil {
		t.Fatalf("empresa de teste inválida: %v", err)
	}
	if _, err := empresas.Criar(context.Background(), empresa); err != nil {
		t.Fatalf("criação da empresa falhou: %v", err)
	}

	return empresa.ID
}

func (c *cenarioBancoLocais) novoLocal(t *testing.T, empresaID uuid.UUID, nome string) *domainestoque.Local {
	t.Helper()

	local, err := domainestoque.NovoLocal(empresaID, nome)
	if err != nil {
		t.Fatalf("local de teste inválido: %v", err)
	}
	if _, err := c.locais.Criar(c.ctx, local); err != nil {
		t.Fatalf("criação do local falhou: %v", err)
	}

	return local
}

func (c *cenarioBancoLocais) carregar(t *testing.T, locais ...*domainestoque.Local) *domainestoque.EstoqueProduto {
	t.Helper()

	ids := make([]uuid.UUID, 0, len(locais))
	for _, local := range locais {
		ids = append(ids, local.ID)
	}

	estoque, err := c.estoque.Carregar(c.ctx, c.produto.ID, ids)
	if err != nil {
		t.Fatalf("carregar o estoque falhou: %v", err)
	}

	return estoque
}

func (c *cenarioBancoLocais) operacao(t *testing.T, estoque *domainestoque.EstoqueProduto, tipo domainestoque.TipoMovimento, local *domainestoque.Local, quantidade int, custo *int64) *domainestoque.OperacaoEstoque {
	t.Helper()

	input := domainestoque.MovimentoLocalInput{
		Tipo:       tipo,
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	}
	if custo != nil {
		valor := domainestoque.CustoDe(*custo)
		input.Custo = &valor
	}

	operacao, err := estoque.Movimentar(input)
	if err != nil {
		t.Fatalf("operação de domínio falhou: %v", err)
	}

	return operacao
}

func (c *cenarioBancoLocais) gravar(t *testing.T, tipo domainestoque.TipoMovimento, local *domainestoque.Local, quantidade int, custo *int64) *domainestoque.OperacaoEstoque {
	t.Helper()

	operacao := c.operacao(t, c.carregar(t, local), tipo, local, quantidade, custo)
	if err := c.estoque.Gravar(c.ctx, operacao); err != nil {
		t.Fatalf("gravar a operação falhou: %v", err)
	}

	return operacao
}

func (c *cenarioBancoLocais) movimentoAnteriorAosLocais(t *testing.T, quantidade int) {
	t.Helper()

	comandos := []string{
		`ALTER TABLE produto_movimento DROP CONSTRAINT chk_produto_movimento_local`,
		`UPDATE produto SET quantidade_atual = quantidade_atual + $1, versao_estoque = versao_estoque + 1 WHERE id = $2`,
		`INSERT INTO produto_movimento (produto_id, tipo, quantidade, saldo_apos, usuario_id)
		 SELECT id, 'entrada', $1, quantidade_atual, $3 FROM produto WHERE id = $2`,
		`ALTER TABLE produto_movimento
		 ADD CONSTRAINT chk_produto_movimento_local CHECK (local_id IS NOT NULL AND saldo_local_apos IS NOT NULL) NOT VALID`,
	}
	argumentos := [][]any{nil, {quantidade, c.produto.ID}, {quantidade, c.produto.ID, c.usuarioID}, nil}

	for i, comando := range comandos {
		if _, err := c.banco.ExecContext(c.ctx, comando, argumentos[i]...); err != nil {
			t.Fatalf("simular movimento anterior aos locais falhou: %v", err)
		}
	}
}

func custoEm(centavos int64) *int64 {
	return &centavos
}

func TestLocalCriaListaAtualizaERemove(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	loja := c.novoLocal(t, c.mia, "Loja")
	daRed := c.novoLocal(t, c.red, "Galpão")

	salvo, err := c.locais.Obter(c.ctx, galpao.ID)
	if err != nil || salvo == nil || salvo.EmpresaID != c.mia || salvo.Nome != "Galpão" || !salvo.Ativo {
		t.Fatalf("local salvo %+v (%v)", salvo, err)
	}
	if ausente, err := c.locais.Obter(c.ctx, uuid.New()); err != nil || ausente != nil {
		t.Errorf("local inexistente = %+v, %v; esperado nil sem erro", ausente, err)
	}

	repetido, err := domainestoque.NovoLocal(c.mia, "GALPÃO")
	if err != nil {
		t.Fatalf("local de teste inválido: %v", err)
	}
	if _, err := c.locais.Criar(c.ctx, repetido); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("nome repetido na mesma empresa = %v, esperado conflito", err)
	}

	casos := []struct {
		nome     string
		filtro   portsout.LocalFiltro
		esperado []uuid.UUID
	}{
		{"todos", portsout.LocalFiltro{}, []uuid.UUID{galpao.ID, daRed.ID, loja.ID}},
		{"da MIA", portsout.LocalFiltro{Empresa: &c.mia}, []uuid.UUID{galpao.ID, loja.ID}},
		{"do acesso", portsout.LocalFiltro{EmpresaIDs: []uuid.UUID{c.red}}, []uuid.UUID{daRed.ID}},
		{"acesso vazio", portsout.LocalFiltro{EmpresaIDs: []uuid.UUID{}}, nil},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.filtro.PaginacaoFiltro = todasAsPaginas
			locais, err := c.locais.Listar(c.ctx, caso.filtro)
			if err != nil {
				t.Fatalf("listar falhou: %v", err)
			}
			if len(locais) != len(caso.esperado) {
				t.Fatalf("%d locais, esperado %d", len(locais), len(caso.esperado))
			}
			encontrados := make(map[uuid.UUID]bool, len(locais))
			for _, local := range locais {
				encontrados[local.ID] = true
			}
			for _, id := range caso.esperado {
				if !encontrados[id] {
					t.Errorf("local %s ausente da listagem", id)
				}
			}
			if len(locais) == 3 && locais[2].ID != loja.ID {
				t.Errorf("último local = %s, esperado a Loja pela ordem de nome", locais[2].Nome)
			}
		})
	}

	varios, err := c.locais.ObterVarios(c.ctx, []uuid.UUID{loja.ID, daRed.ID, uuid.New()})
	if err != nil || len(varios) != 2 {
		t.Errorf("ObterVarios = %d locais (%v), esperado 2", len(varios), err)
	}

	if err := loja.Renomear("Loja Centro", time.Now().UTC()); err != nil {
		t.Fatalf("renomear falhou: %v", err)
	}
	if ok, err := c.locais.Atualizar(c.ctx, loja); err != nil || !ok {
		t.Fatalf("atualizar = %v, %v", ok, err)
	}
	if salvo, _ := c.locais.Obter(c.ctx, loja.ID); salvo.Nome != "Loja Centro" {
		t.Errorf("nome salvo = %q, esperado Loja Centro", salvo.Nome)
	}

	if ok, err := c.locais.Remover(c.ctx, loja.ID); err != nil || !ok {
		t.Errorf("remover local sem vínculo = %v, %v", ok, err)
	}

	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 3, custoEm(100))
	if _, err := c.locais.Remover(c.ctx, galpao.ID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remover local com estoque = %v, esperado erro de validação", err)
	}
}

func TestOperacaoGravaSaldoLoteEMovimentoNaMesmaTransacao(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")

	inicial := c.carregar(t, galpao)
	if inicial.Produto.VersaoEstoque != 0 || inicial.TotalAlocado != 0 || len(inicial.Saldos) != 0 || len(inicial.Lotes) != 0 {
		t.Fatalf("estoque inicial = %+v", inicial)
	}

	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 10, custoEm(100))
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 5, custoEm(200))
	saida := c.gravar(t, domainestoque.TipoMovimentoSaida, galpao, 12, nil)

	estoque := c.carregar(t, galpao)
	if estoque.Produto.VersaoEstoque != 3 || estoque.Produto.Saldo.Quantidade() != 3 || estoque.TotalAlocado != 3 {
		t.Errorf("produto com versão %d, saldo %d e alocado %d; esperado 3, 3 e 3",
			estoque.Produto.VersaoEstoque, estoque.Produto.Saldo.Quantidade(), estoque.TotalAlocado)
	}
	if saldo := estoque.SaldoNoLocal(galpao.ID); saldo.Quantidade != 3 || saldo.Reservado != 0 {
		t.Errorf("saldo do galpão = %+v, esperado 3", saldo)
	}
	if len(estoque.Lotes) != 1 || estoque.Lotes[0].QuantidadeAtual != 3 || estoque.Lotes[0].Custo.Centavos() != 200 {
		t.Errorf("lotes com saldo = %+v, esperado só o de custo 200 com 3", estoque.Lotes)
	}

	var consumos []struct {
		Quantidade int   `db:"quantidade"`
		Custo      int64 `db:"custo_unitario_centavos"`
	}
	if err := c.banco.SelectContext(c.ctx, &consumos, `
		SELECT ml.quantidade, ml.custo_unitario_centavos
		FROM produto_movimento_lote ml
		JOIN produto_lote l ON l.id = ml.lote_id
		WHERE ml.movimento_id = $1
		ORDER BY l.entrada_em
	`, saida.Movimento.ID); err != nil {
		t.Fatalf("consulta dos consumos falhou: %v", err)
	}
	if len(consumos) != 2 || consumos[0].Quantidade != 10 || consumos[0].Custo != 100 || consumos[1].Quantidade != 2 || consumos[1].Custo != 200 {
		t.Errorf("consumos gravados = %+v, esperado 10×100 e 2×200", consumos)
	}

	movimentos, err := c.movimentos.Listar(c.ctx, portsout.MovimentoFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: &c.produto.ID})
	if err != nil || len(movimentos) != 3 {
		t.Fatalf("movimentos = %d (%v), esperado 3", len(movimentos), err)
	}
	ultimo := movimentos[0]
	if ultimo.Tipo != domainestoque.TipoMovimentoSaida || ultimo.LocalID == nil || *ultimo.LocalID != galpao.ID ||
		ultimo.SaldoLocalApos == nil || ultimo.SaldoLocalApos.Quantidade() != 3 || ultimo.Custo != nil {
		t.Errorf("último movimento gravado = %+v", ultimo)
	}
	if entrada := movimentos[2]; entrada.Custo == nil || entrada.Custo.Centavos() != 100 {
		t.Errorf("primeira entrada com custo %v, esperado 100", entrada.Custo)
	}
}

func TestTransferenciaPreservaCustoEIdadeNoDestino(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	loja := c.novoLocal(t, c.mia, "Loja")
	entrada := c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 8, custoEm(150))

	operacao, err := c.carregar(t, galpao, loja).Transferir(domainestoque.TransferenciaInput{
		Origem:     galpao,
		Destino:    loja,
		Quantidade: domainestoque.QuantidadeDe(5),
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("transferência de domínio falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, operacao); err != nil {
		t.Fatalf("gravar a transferência falhou: %v", err)
	}

	estoque := c.carregar(t, galpao, loja)
	if estoque.Produto.Saldo.Quantidade() != 8 || estoque.SaldoNoLocal(galpao.ID).Quantidade != 3 || estoque.SaldoNoLocal(loja.ID).Quantidade != 5 {
		t.Errorf("saldos depois da transferência: total %d, galpão %d, loja %d",
			estoque.Produto.Saldo.Quantidade(), estoque.SaldoNoLocal(galpao.ID).Quantidade, estoque.SaldoNoLocal(loja.ID).Quantidade)
	}

	lotesDaLoja, err := c.lotes.Listar(c.ctx, portsout.LoteFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: c.produto.ID, LocalID: &loja.ID})
	if err != nil || len(lotesDaLoja) != 1 {
		t.Fatalf("lotes da loja = %d (%v), esperado 1", len(lotesDaLoja), err)
	}
	lotesDoGalpao, err := c.lotes.Listar(c.ctx, portsout.LoteFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: c.produto.ID, LocalID: &galpao.ID})
	if err != nil || len(lotesDoGalpao) != 1 || lotesDoGalpao[0].ID != entrada.LotesCriados[0].ID {
		t.Fatalf("lotes do galpão = %+v (%v), esperado o da entrada", lotesDoGalpao, err)
	}
	lote := lotesDaLoja[0]
	origem := lotesDoGalpao[0]
	if lote.LoteOrigemID == nil || *lote.LoteOrigemID != origem.ID || lote.Custo.Centavos() != 150 || !lote.EntradaEm.Equal(origem.EntradaEm) {
		t.Errorf("lote da loja = %+v, esperado herdar custo e entrada do lote %s", lote, origem.ID)
	}

	movimentosDaLoja, err := c.movimentos.Listar(c.ctx, portsout.MovimentoFiltro{PaginacaoFiltro: todasAsPaginas, LocalID: &loja.ID})
	if err != nil || len(movimentosDaLoja) != 1 || movimentosDaLoja[0].LocalDestinoID == nil || *movimentosDaLoja[0].LocalDestinoID != loja.ID {
		t.Errorf("movimentos da loja = %+v (%v), esperado a transferência", movimentosDaLoja, err)
	}
}

func TestOperacaoCalculadaSobreEstadoVelhoRecebeConflito(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 10, custoEm(100))

	estado := c.carregar(t, galpao)
	primeira := c.operacao(t, estado, domainestoque.TipoMovimentoSaida, galpao, 6, nil)
	segunda := c.operacao(t, estado, domainestoque.TipoMovimentoSaida, galpao, 6, nil)

	if err := c.estoque.Gravar(c.ctx, primeira); err != nil {
		t.Fatalf("primeira operação falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, segunda); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("segunda operação sobre o mesmo estado = %v, esperado conflito", err)
	}

	depois := c.carregar(t, galpao)
	if depois.Produto.Saldo.Quantidade() != 4 || depois.SaldoNoLocal(galpao.ID).Quantidade != 4 || depois.Produto.VersaoEstoque != 2 {
		t.Errorf("estado depois do conflito: total %d, local %d, versão %d; esperado 4, 4 e 2",
			depois.Produto.Saldo.Quantidade(), depois.SaldoNoLocal(galpao.ID).Quantidade, depois.Produto.VersaoEstoque)
	}

	var movimentos int
	if err := c.banco.GetContext(c.ctx, &movimentos, `SELECT COUNT(*) FROM produto_movimento WHERE produto_id = $1`, c.produto.ID); err != nil || movimentos != 2 {
		t.Errorf("movimentos gravados = %d (%v), esperado 2: a operação em conflito não pode deixar rastro", movimentos, err)
	}
}

func TestLocalInativoNaoRecebeEntradaENaoDesativaComSaldo(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 2, custoEm(100))

	if ok, err := c.locais.DesativarSemSaldo(c.ctx, galpao.ID, time.Now().UTC()); err != nil || ok {
		t.Fatalf("desativar com saldo = %v, %v; esperado recusado", ok, err)
	}

	pendente := c.operacao(t, c.carregar(t, galpao), domainestoque.TipoMovimentoEntrada, galpao, 1, custoEm(100))
	c.gravar(t, domainestoque.TipoMovimentoSaida, galpao, 2, nil)

	if ok, err := c.locais.DesativarSemSaldo(c.ctx, galpao.ID, time.Now().UTC()); err != nil || !ok {
		t.Fatalf("desativar sem saldo = %v, %v; esperado desativado", ok, err)
	}
	if salvo, _ := c.locais.Obter(c.ctx, galpao.ID); salvo.Ativo {
		t.Error("local deveria estar inativo")
	}

	pendente.VersaoLida = 2
	if err := c.estoque.Gravar(c.ctx, pendente); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("entrada calculada antes da desativação = %v, esperado erro de validação", err)
	}
	if ok, err := c.locais.DesativarSemSaldo(c.ctx, uuid.New(), time.Now().UTC()); err != nil || ok {
		t.Errorf("desativar local inexistente = %v, %v; esperado recusado sem erro", ok, err)
	}
}

func TestConsultasDeSaldoELotesRespeitamEmpresasELocais(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	loja := c.novoLocal(t, c.mia, "Loja")
	daRed := c.novoLocal(t, c.red, "Depósito")
	c.movimentoAnteriorAosLocais(t, 4)
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 3, custoEm(100))
	c.gravar(t, domainestoque.TipoMovimentoEntrada, daRed, 7, custoEm(300))
	c.gravar(t, domainestoque.TipoMovimentoEntrada, loja, 1, custoEm(50))
	c.gravar(t, domainestoque.TipoMovimentoSaida, loja, 1, nil)

	todos, err := c.estoque.SaldosDoProduto(c.ctx, c.produto.ID, nil)
	if err != nil || len(todos) != 3 {
		t.Fatalf("saldos do produto = %d (%v), esperado 3", len(todos), err)
	}
	if todos[0].LocalNome != "Depósito" || todos[0].EmpresaID != c.red || todos[0].ProdutoNome != "Cimento CP II" {
		t.Errorf("primeiro saldo = %+v, esperado o Depósito da Red Diamond", todos[0])
	}

	daMia, err := c.estoque.SaldosDoProduto(c.ctx, c.produto.ID, []uuid.UUID{c.mia})
	if err != nil || len(daMia) != 2 {
		t.Errorf("saldos da MIA = %d (%v), esperado 2", len(daMia), err)
	}
	nenhum, err := c.estoque.SaldosDoProduto(c.ctx, c.produto.ID, []uuid.UUID{})
	if err != nil || len(nenhum) != 0 {
		t.Errorf("saldos com acesso vazio = %d (%v), esperado 0", len(nenhum), err)
	}

	if total, err := c.estoque.TotalAlocado(c.ctx, c.produto.ID); err != nil || total != 10 {
		t.Errorf("total alocado = %d (%v), esperado 10", total, err)
	}
	if estoque := c.carregar(t, galpao); estoque.NaoAlocado() != 4 {
		t.Errorf("não alocado = %d, esperado o saldo antigo de 4", estoque.NaoAlocado())
	}

	daLoja, err := c.estoque.SaldosDoLocal(c.ctx, portsout.SaldoDoLocalFiltro{PaginacaoFiltro: todasAsPaginas, LocalID: loja.ID})
	if err != nil || len(daLoja) != 1 || daLoja[0].Quantidade != 0 {
		t.Errorf("saldos da loja = %+v (%v), esperado a linha zerada", daLoja, err)
	}
	comSaldo, err := c.estoque.SaldosDoLocal(c.ctx, portsout.SaldoDoLocalFiltro{PaginacaoFiltro: todasAsPaginas, LocalID: loja.ID, ComSaldo: true})
	if err != nil || len(comSaldo) != 0 {
		t.Errorf("saldos da loja com saldo = %d (%v), esperado 0", len(comSaldo), err)
	}

	lotesDaMia, err := c.lotes.Listar(c.ctx, portsout.LoteFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: c.produto.ID, EmpresaIDs: []uuid.UUID{c.mia}})
	if err != nil || len(lotesDaMia) != 2 {
		t.Errorf("lotes da MIA = %d (%v), esperado 2", len(lotesDaMia), err)
	}
	lotesComSaldo, err := c.lotes.Listar(c.ctx, portsout.LoteFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: c.produto.ID, EmpresaIDs: []uuid.UUID{c.mia}, ComSaldo: true})
	if err != nil || len(lotesComSaldo) != 1 || lotesComSaldo[0].LocalID != galpao.ID {
		t.Errorf("lotes da MIA com saldo = %+v (%v), esperado só o do galpão", lotesComSaldo, err)
	}

	daRedMovimentos, err := c.movimentos.Listar(c.ctx, portsout.MovimentoFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: &c.produto.ID, EmpresaIDs: []uuid.UUID{c.red}})
	if err != nil || len(daRedMovimentos) != 2 {
		t.Errorf("movimentos visíveis para a Red = %d (%v), esperado a entrada dela e o movimento antigo sem local", len(daRedMovimentos), err)
	}
	semAcesso, err := c.movimentos.ListarResumo(c.ctx, 10, []uuid.UUID{})
	if err != nil || len(semAcesso) != 1 || semAcesso[0].Tipo != domainestoque.TipoMovimentoEntrada {
		t.Errorf("resumo com acesso vazio = %+v (%v), esperado só o movimento antigo", semAcesso, err)
	}
}

func TestSaidasSimultaneasMantemSaldoIgualASomaDosLotes(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 5, custoEm(100))
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 5, custoEm(200))

	const concorrentes = 8
	var (
		espera    sync.WaitGroup
		trava     sync.Mutex
		gravadas  int
		conflitos int
		outros    []error
	)
	for range concorrentes {
		espera.Add(1)
		go func() {
			defer espera.Done()

			estoque, err := c.estoque.Carregar(c.ctx, c.produto.ID, []uuid.UUID{galpao.ID})
			if err == nil {
				var operacao *domainestoque.OperacaoEstoque
				operacao, err = estoque.Movimentar(domainestoque.MovimentoLocalInput{
					Tipo:       domainestoque.TipoMovimentoSaida,
					Local:      galpao,
					Quantidade: domainestoque.QuantidadeDe(2),
					UsuarioID:  c.usuarioID,
					Agora:      time.Now().UTC(),
				})
				if err == nil {
					err = c.estoque.Gravar(c.ctx, operacao)
				}
			}

			trava.Lock()
			defer trava.Unlock()
			switch {
			case err == nil:
				gravadas++
			case errors.Is(err, domain.ErrConflito), errors.Is(err, domain.ErrValidacao):
				conflitos++
			default:
				outros = append(outros, err)
			}
		}()
	}
	espera.Wait()

	if len(outros) > 0 {
		t.Fatalf("erros inesperados: %v", outros)
	}
	if gravadas == 0 || gravadas+conflitos != concorrentes {
		t.Fatalf("%d gravadas e %d recusadas de %d", gravadas, conflitos, concorrentes)
	}

	var somaDosLotes int
	if err := c.banco.GetContext(c.ctx, &somaDosLotes,
		`SELECT COALESCE(SUM(quantidade_atual), 0) FROM produto_lote WHERE produto_id = $1`, c.produto.ID,
	); err != nil {
		t.Fatalf("soma dos lotes falhou: %v", err)
	}

	estoque := c.carregar(t, galpao)
	esperado := 10 - 2*gravadas
	if estoque.Produto.Saldo.Quantidade() != esperado || estoque.SaldoNoLocal(galpao.ID).Quantidade != esperado || somaDosLotes != esperado {
		t.Errorf("total %d, local %d e lotes %d; esperado %d nos três",
			estoque.Produto.Saldo.Quantidade(), estoque.SaldoNoLocal(galpao.ID).Quantidade, somaDosLotes, esperado)
	}
}

func TestBancoRecusaMovimentoNovoSemLocal(t *testing.T) {
	c := novoCenarioLocais(t)

	_, err := c.banco.ExecContext(c.ctx, `
		INSERT INTO produto_movimento (produto_id, tipo, quantidade, saldo_apos, usuario_id)
		VALUES ($1, 'entrada', 1, 1, $2)
	`, c.produto.ID, c.usuarioID)
	if err == nil {
		t.Fatal("movimento novo sem local foi aceito pelo banco")
	}
}
