package estoque_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type cenarioEstoque struct {
	estoque *domainestoque.EstoqueProduto
	galpao  *domainestoque.Local
	loja    *domainestoque.Local
	outra   *domainestoque.Local
	usuario uuid.UUID
	inicio  time.Time
}

func novoCenarioEstoque(t *testing.T) *cenarioEstoque {
	t.Helper()

	empresaID := uuid.New()
	produto := novoProdutoDeTeste(t)
	produto.VersaoEstoque = 7

	return &cenarioEstoque{
		estoque: &domainestoque.EstoqueProduto{
			Produto: produto,
			Saldos:  map[uuid.UUID]domainestoque.SaldoLocal{},
		},
		galpao:  novoLocalDeTeste(t, empresaID, "Galpão"),
		loja:    novoLocalDeTeste(t, empresaID, "Loja"),
		outra:   novoLocalDeTeste(t, uuid.New(), "Galpão de outra empresa"),
		usuario: uuid.New(),
		inicio:  time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
	}
}

func novoLocalDeTeste(t *testing.T, empresaID uuid.UUID, nome string) *domainestoque.Local {
	t.Helper()

	local, err := domainestoque.NovoLocal(empresaID, nome)
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}

	return local
}

func (c *cenarioEstoque) comLote(local *domainestoque.Local, quantidade int, custo *int64, diasDesdeOInicio int) *domainestoque.Lote {
	lote := &domainestoque.Lote{
		ID:                uuid.New(),
		ProdutoID:         c.estoque.Produto.ID,
		LocalID:           local.ID,
		MovimentoID:       uuid.New(),
		QuantidadeInicial: quantidade,
		QuantidadeAtual:   quantidade,
		EntradaEm:         c.inicio.AddDate(0, 0, diasDesdeOInicio),
	}
	if custo != nil {
		valor := domainestoque.CustoDe(*custo)
		lote.Custo = &valor
	}

	c.estoque.Lotes = append(c.estoque.Lotes, lote)
	saldo := c.estoque.SaldoNoLocal(local.ID)
	saldo.Quantidade += quantidade
	c.estoque.Saldos[local.ID] = saldo
	c.estoque.TotalAlocado += quantidade
	c.estoque.Produto.Saldo = domainestoque.SaldoDe(c.estoque.Produto.Saldo.Quantidade() + quantidade)

	return lote
}

func (c *cenarioEstoque) movimentar(tipo domainestoque.TipoMovimento, local *domainestoque.Local, quantidade int, custo *int64) (*domainestoque.OperacaoEstoque, error) {
	input := domainestoque.MovimentoLocalInput{
		Tipo:       tipo,
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		UsuarioID:  c.usuario,
		Observacao: "teste",
		Agora:      c.inicio.AddDate(0, 1, 0),
	}
	if custo != nil {
		valor := domainestoque.CustoDe(*custo)
		input.Custo = &valor
	}

	return c.estoque.Movimentar(input)
}

func (c *cenarioEstoque) transferir(origem, destino *domainestoque.Local, quantidade int) (*domainestoque.OperacaoEstoque, error) {
	return c.estoque.Transferir(domainestoque.TransferenciaInput{
		Origem:     origem,
		Destino:    destino,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		UsuarioID:  c.usuario,
		Agora:      c.inicio.AddDate(0, 1, 0),
	})
}

func centavos(valor int64) *int64 {
	return &valor
}

func saldoDaOperacao(t *testing.T, operacao *domainestoque.OperacaoEstoque, localID uuid.UUID) domainestoque.SaldoLocal {
	t.Helper()

	for _, saldo := range operacao.Saldos {
		if saldo.LocalID == localID {
			return saldo
		}
	}
	t.Fatalf("operação sem saldo do local %s", localID)

	return domainestoque.SaldoLocal{}
}

func TestEntradaCriaLoteComCustoNoLocal(t *testing.T) {
	c := novoCenarioEstoque(t)

	operacao, err := c.movimentar(domainestoque.TipoMovimentoEntrada, c.galpao, 10, centavos(2590))
	if err != nil {
		t.Fatalf("entrada falhou: %v", err)
	}

	if operacao.VersaoLida != 7 || operacao.ProdutoID != c.estoque.Produto.ID {
		t.Errorf("operação com versão %d e produto %s", operacao.VersaoLida, operacao.ProdutoID)
	}
	if operacao.Saldo.Quantidade() != 10 {
		t.Errorf("saldo total = %d, esperado 10", operacao.Saldo.Quantidade())
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Quantidade != 10 {
		t.Errorf("saldo do galpão = %d, esperado 10", saldo.Quantidade)
	}

	movimento := operacao.Movimento
	if movimento.LocalID == nil || *movimento.LocalID != c.galpao.ID || movimento.LocalDestinoID != nil {
		t.Errorf("movimento com local %v e destino %v", movimento.LocalID, movimento.LocalDestinoID)
	}
	if movimento.SaldoLocalApos == nil || movimento.SaldoLocalApos.Quantidade() != 10 || movimento.SaldoApos.Quantidade() != 10 {
		t.Errorf("movimento com saldo local %v e total %d", movimento.SaldoLocalApos, movimento.SaldoApos.Quantidade())
	}
	if movimento.Custo == nil || movimento.Custo.Centavos() != 2590 || len(movimento.Consumos) != 0 {
		t.Errorf("movimento com custo %v e %d consumos", movimento.Custo, len(movimento.Consumos))
	}

	if len(operacao.LotesCriados) != 1 {
		t.Fatalf("entrada criou %d lotes, esperado 1", len(operacao.LotesCriados))
	}
	lote := operacao.LotesCriados[0]
	if lote.LocalID != c.galpao.ID || lote.MovimentoID != movimento.ID || lote.LoteOrigemID != nil {
		t.Errorf("lote com local %s, movimento %s e origem %v", lote.LocalID, lote.MovimentoID, lote.LoteOrigemID)
	}
	if lote.QuantidadeInicial != 10 || lote.QuantidadeAtual != 10 || lote.Custo.Centavos() != 2590 {
		t.Errorf("lote com quantidades %d/%d e custo %v", lote.QuantidadeInicial, lote.QuantidadeAtual, lote.Custo)
	}
	if !lote.EntradaEm.Equal(movimento.CriadoEm) {
		t.Errorf("lote com entrada em %s, esperado %s", lote.EntradaEm, movimento.CriadoEm)
	}
}

func TestCustoPorTipoDeMovimento(t *testing.T) {
	casos := []struct {
		nome       string
		tipo       domainestoque.TipoMovimento
		quantidade int
		custo      *int64
		sentinela  error
	}{
		{"entrada sem custo", domainestoque.TipoMovimentoEntrada, 5, nil, domain.ErrValidacao},
		{"entrada com custo zero", domainestoque.TipoMovimentoEntrada, 5, centavos(0), nil},
		{"devolução sem custo", domainestoque.TipoMovimentoDevolucao, 5, nil, nil},
		{"devolução com custo", domainestoque.TipoMovimentoDevolucao, 5, centavos(100), nil},
		{"ajuste positivo sem custo", domainestoque.TipoMovimentoAjuste, 5, nil, nil},
		{"implantação sem custo", domainestoque.TipoMovimentoImplantacao, 5, nil, nil},
		{"saída com custo", domainestoque.TipoMovimentoSaida, 1, centavos(100), domain.ErrValidacao},
		{"perda com custo", domainestoque.TipoMovimentoPerda, 1, centavos(100), domain.ErrValidacao},
		{"ajuste negativo com custo", domainestoque.TipoMovimentoAjuste, -1, centavos(100), domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := novoCenarioEstoque(t)
			c.comLote(c.galpao, 3, centavos(50), 0)
			c.estoque.Produto.Saldo = domainestoque.SaldoDe(c.estoque.Produto.Saldo.Quantidade() + 10)

			operacao, err := c.movimentar(caso.tipo, c.galpao, caso.quantidade, caso.custo)
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.sentinela != nil {
				return
			}
			if len(operacao.LotesCriados) == 1 {
				lote := operacao.LotesCriados[0]
				if (caso.custo == nil) != lote.CustoDesconhecido() {
					t.Errorf("lote com custo desconhecido = %v, custo informado %v", lote.CustoDesconhecido(), caso.custo)
				}
			}
		})
	}
}

func TestSaidaConsomeLotesPorFIFO(t *testing.T) {
	c := novoCenarioEstoque(t)
	recente := c.comLote(c.galpao, 10, centavos(300), 10)
	antigo := c.comLote(c.galpao, 4, centavos(100), 0)
	c.comLote(c.loja, 50, centavos(1), -5)
	intermediario := c.comLote(c.galpao, 5, nil, 5)

	operacao, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 12, nil)
	if err != nil {
		t.Fatalf("saída falhou: %v", err)
	}

	esperado := []struct {
		loteID     uuid.UUID
		quantidade int
		custo      *int64
	}{
		{antigo.ID, 4, centavos(100)},
		{intermediario.ID, 5, nil},
		{recente.ID, 3, centavos(300)},
	}

	consumos := operacao.Movimento.Consumos
	if len(consumos) != len(esperado) {
		t.Fatalf("%d consumos, esperado %d", len(consumos), len(esperado))
	}
	for i, consumo := range consumos {
		if consumo.LoteID != esperado[i].loteID || consumo.Quantidade != esperado[i].quantidade {
			t.Errorf("consumo %d = lote %s × %d, esperado lote %s × %d", i, consumo.LoteID, consumo.Quantidade, esperado[i].loteID, esperado[i].quantidade)
		}
		if (consumo.Custo == nil) != (esperado[i].custo == nil) || (consumo.Custo != nil && consumo.Custo.Centavos() != *esperado[i].custo) {
			t.Errorf("consumo %d com custo %v, esperado %v", i, consumo.Custo, esperado[i].custo)
		}
	}

	if operacao.Saldo.Quantidade() != 57 {
		t.Errorf("saldo total = %d, esperado 57", operacao.Saldo.Quantidade())
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Quantidade != 7 {
		t.Errorf("saldo do galpão = %d, esperado 7", saldo.Quantidade)
	}
	if len(operacao.Saldos) != 1 || len(operacao.LotesCriados) != 0 {
		t.Errorf("saída alterou %d saldos e criou %d lotes, esperado 1 e 0", len(operacao.Saldos), len(operacao.LotesCriados))
	}
	if recente.QuantidadeAtual != 10 {
		t.Error("o cálculo da operação não deve alterar os lotes carregados")
	}
}

func TestFIFODesempataPeloID(t *testing.T) {
	c := novoCenarioEstoque(t)
	primeiro := c.comLote(c.galpao, 2, centavos(100), 0)
	segundo := c.comLote(c.galpao, 2, centavos(200), 0)
	primeiro.ID = uuid.MustParse("00000000-0000-4000-8000-000000000002")
	segundo.ID = uuid.MustParse("00000000-0000-4000-8000-000000000001")

	operacao, err := c.movimentar(domainestoque.TipoMovimentoPerda, c.galpao, 1, nil)
	if err != nil {
		t.Fatalf("perda falhou: %v", err)
	}
	if consumos := operacao.Movimento.Consumos; len(consumos) != 1 || consumos[0].LoteID != segundo.ID {
		t.Errorf("consumos %+v, esperado o lote de menor id", consumos)
	}
}

func TestSaidaRespeitaODisponivelDoLocal(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)
	c.comLote(c.loja, 20, centavos(100), 0)
	saldo := c.estoque.Saldos[c.galpao.ID]
	saldo.Reservado = 4
	c.estoque.Saldos[c.galpao.ID] = saldo

	for _, tipo := range []domainestoque.TipoMovimento{domainestoque.TipoMovimentoSaida, domainestoque.TipoMovimentoPerda} {
		if _, err := c.movimentar(tipo, c.galpao, 7, nil); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s acima do disponível = %v, esperado erro de validação", tipo, err)
		}
	}
	if _, err := c.movimentar(domainestoque.TipoMovimentoAjuste, c.galpao, -7, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("ajuste negativo acima do disponível = %v, esperado erro de validação", err)
	}

	operacao, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 6, nil)
	if err != nil {
		t.Fatalf("saída dentro do disponível falhou: %v", err)
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Quantidade != 4 || saldo.Reservado != 4 {
		t.Errorf("saldo do galpão = %d com %d reservado, esperado 4 e 4", saldo.Quantidade, saldo.Reservado)
	}
}

func TestEntradaExigeLocalAtivo(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 5, centavos(100), 0)
	if err := c.galpao.Desativar(false, c.inicio); err != nil {
		t.Fatalf("desativar falhou: %v", err)
	}

	if _, err := c.movimentar(domainestoque.TipoMovimentoEntrada, c.galpao, 1, centavos(100)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("entrada em local inativo = %v, esperado erro de validação", err)
	}
	if _, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 5, nil); err != nil {
		t.Errorf("saída de local inativo = %v, esperado sucesso para esvaziá-lo", err)
	}
}

func TestImplantacaoUsaOSaldoNaoAlocado(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 5, centavos(100), 0)
	c.estoque.Produto.Saldo = domainestoque.SaldoDe(15)

	if c.estoque.NaoAlocado() != 10 {
		t.Fatalf("não alocado = %d, esperado 10", c.estoque.NaoAlocado())
	}
	if _, err := c.movimentar(domainestoque.TipoMovimentoImplantacao, c.loja, 11, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("implantação acima do não alocado = %v, esperado erro de validação", err)
	}

	operacao, err := c.movimentar(domainestoque.TipoMovimentoImplantacao, c.loja, 10, centavos(80))
	if err != nil {
		t.Fatalf("implantação falhou: %v", err)
	}
	if operacao.Saldo.Quantidade() != 15 || operacao.Movimento.SaldoApos.Quantidade() != 15 {
		t.Errorf("implantação mudou o total para %d, esperado 15", operacao.Saldo.Quantidade())
	}
	if saldo := saldoDaOperacao(t, operacao, c.loja.ID); saldo.Quantidade != 10 {
		t.Errorf("saldo da loja = %d, esperado 10", saldo.Quantidade)
	}
	if len(operacao.LotesCriados) != 1 || operacao.LotesCriados[0].Custo.Centavos() != 80 {
		t.Errorf("implantação criou %+v, esperado um lote de custo 80", operacao.LotesCriados)
	}
}

func TestMovimentoLocalValidaEntrada(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 5, centavos(100), 0)

	casos := []struct {
		nome  string
		input domainestoque.MovimentoLocalInput
	}{
		{"sem local", domainestoque.MovimentoLocalInput{Tipo: domainestoque.TipoMovimentoSaida, Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario}},
		{"sem usuário", domainestoque.MovimentoLocalInput{Tipo: domainestoque.TipoMovimentoSaida, Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(1)}},
		{"quantidade zero", domainestoque.MovimentoLocalInput{Tipo: domainestoque.TipoMovimentoSaida, Local: c.galpao, UsuarioID: c.usuario}},
		{"tipo inválido", domainestoque.MovimentoLocalInput{Tipo: "furto", Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario}},
		{"transferência pelo movimento", domainestoque.MovimentoLocalInput{Tipo: domainestoque.TipoMovimentoTransferencia, Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := c.estoque.Movimentar(caso.input); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}
}

func TestLotesQueNaoCobremOSaldoSaoConflito(t *testing.T) {
	c := novoCenarioEstoque(t)
	lote := c.comLote(c.galpao, 5, centavos(100), 0)
	lote.QuantidadeAtual = 2

	if _, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 3, nil); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("saída com lotes inconsistentes = %v, esperado conflito", err)
	}
}

func TestTransferenciaLevaCustoEIdadeDosLotes(t *testing.T) {
	c := novoCenarioEstoque(t)
	antigo := c.comLote(c.galpao, 3, centavos(100), 0)
	recente := c.comLote(c.galpao, 10, nil, 4)
	c.comLote(c.loja, 2, centavos(500), 1)

	operacao, err := c.transferir(c.galpao, c.loja, 5)
	if err != nil {
		t.Fatalf("transferência falhou: %v", err)
	}

	if operacao.Saldo.Quantidade() != 15 || operacao.Movimento.SaldoApos.Quantidade() != 15 {
		t.Errorf("transferência mudou o total para %d, esperado 15", operacao.Saldo.Quantidade())
	}
	if origem := saldoDaOperacao(t, operacao, c.galpao.ID); origem.Quantidade != 8 {
		t.Errorf("saldo da origem = %d, esperado 8", origem.Quantidade)
	}
	if destino := saldoDaOperacao(t, operacao, c.loja.ID); destino.Quantidade != 7 {
		t.Errorf("saldo do destino = %d, esperado 7", destino.Quantidade)
	}

	movimento := operacao.Movimento
	if movimento.Tipo != domainestoque.TipoMovimentoTransferencia || *movimento.LocalID != c.galpao.ID || *movimento.LocalDestinoID != c.loja.ID {
		t.Errorf("movimento %s de %v para %v", movimento.Tipo, movimento.LocalID, movimento.LocalDestinoID)
	}
	if movimento.SaldoLocalApos.Quantidade() != 8 || len(movimento.Consumos) != 2 {
		t.Errorf("movimento com saldo de origem %d e %d consumos", movimento.SaldoLocalApos.Quantidade(), len(movimento.Consumos))
	}

	if len(operacao.LotesCriados) != 2 {
		t.Fatalf("transferência criou %d lotes, esperado 2", len(operacao.LotesCriados))
	}
	esperado := []struct {
		origem     *domainestoque.Lote
		quantidade int
	}{{antigo, 3}, {recente, 2}}
	for i, lote := range operacao.LotesCriados {
		if lote.LocalID != c.loja.ID || lote.MovimentoID != movimento.ID {
			t.Errorf("lote %d no local %s pelo movimento %s", i, lote.LocalID, lote.MovimentoID)
		}
		if lote.LoteOrigemID == nil || *lote.LoteOrigemID != esperado[i].origem.ID || lote.QuantidadeInicial != esperado[i].quantidade {
			t.Errorf("lote %d com origem %v e quantidade %d", i, lote.LoteOrigemID, lote.QuantidadeInicial)
		}
		if !lote.EntradaEm.Equal(esperado[i].origem.EntradaEm) {
			t.Errorf("lote %d com entrada em %s, esperado a do lote de origem %s", i, lote.EntradaEm, esperado[i].origem.EntradaEm)
		}
		if (lote.Custo == nil) != (esperado[i].origem.Custo == nil) {
			t.Errorf("lote %d com custo %v, esperado o do lote de origem %v", i, lote.Custo, esperado[i].origem.Custo)
		}
	}
}

func TestTransferenciaValidaLocaisEDisponivel(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 5, centavos(100), 0)
	saldo := c.estoque.Saldos[c.galpao.ID]
	saldo.Reservado = 2
	c.estoque.Saldos[c.galpao.ID] = saldo
	inativo := novoLocalDeTeste(t, c.galpao.EmpresaID, "Depósito antigo")
	if err := inativo.Desativar(false, c.inicio); err != nil {
		t.Fatalf("desativar falhou: %v", err)
	}

	casos := []struct {
		nome       string
		origem     *domainestoque.Local
		destino    *domainestoque.Local
		quantidade int
	}{
		{"sem destino", c.galpao, nil, 1},
		{"mesmo local", c.galpao, c.galpao, 1},
		{"outra empresa", c.galpao, c.outra, 1},
		{"destino inativo", c.galpao, inativo, 1},
		{"quantidade negativa", c.galpao, c.loja, -1},
		{"acima do disponível", c.galpao, c.loja, 4},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := c.transferir(caso.origem, caso.destino, caso.quantidade); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := c.transferir(c.galpao, c.loja, 3); err != nil {
		t.Errorf("transferência dentro do disponível = %v, esperado sucesso", err)
	}
}

func TestEfeitoDosTiposNovos(t *testing.T) {
	cinco := domainestoque.QuantidadeDe(5)

	if efeito := domainestoque.TipoMovimentoImplantacao.Efeito(cinco); efeito != 0 {
		t.Errorf("efeito total da implantação = %d, esperado 0", efeito)
	}
	if efeito := domainestoque.TipoMovimentoImplantacao.EfeitoNoLocal(cinco); efeito != 5 {
		t.Errorf("efeito no local da implantação = %d, esperado 5", efeito)
	}
	if efeito := domainestoque.TipoMovimentoTransferencia.Efeito(cinco); efeito != 0 {
		t.Errorf("efeito total da transferência = %d, esperado 0", efeito)
	}
	for _, valor := range []string{"implantacao", "TRANSFERENCIA"} {
		if _, err := domainestoque.NovoTipoMovimento(valor); err != nil {
			t.Errorf("NovoTipoMovimento(%q) = %v, esperado sucesso", valor, err)
		}
	}
}
