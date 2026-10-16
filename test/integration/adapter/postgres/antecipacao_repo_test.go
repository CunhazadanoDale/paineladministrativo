package postgres_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

var _ portsreceber.FornecedorRepository = (*postgres.FornecedorRepository)(nil)

func (c *cenarioBancoTitulos) gravacaoDoBordero(t *testing.T, liquido int64, titulos ...*domainreceber.Titulo) *portsreceber.GravacaoAntecipacao {
	t.Helper()

	versoes := make([]int, len(titulos))
	anteriores := make([]domainreceber.Status, len(titulos))
	for i, titulo := range titulos {
		versoes[i], anteriores[i] = titulo.Versao, titulo.Status
	}
	numero := "B-1"
	antecipacao, historicos, err := domainreceber.NovaAntecipacao(c.empresaID, c.solicitanteID, domainreceber.DadosAntecipacao{
		FornecedorID: c.fornecedorID, Numero: &numero, OperadoEm: diaDoTitulo(10, 14), TaxaBP: 300,
		ValorLiquidoCentavos: liquido, ContaBancariaID: c.contaID,
	}, titulos, agoraDoTitulo)
	if err != nil {
		t.Fatalf("borderô de teste inválido: %v", err)
	}

	gravacao := &portsreceber.GravacaoAntecipacao{Antecipacao: antecipacao}
	for i, titulo := range titulos {
		gravacao.Titulos = append(gravacao.Titulos, portsreceber.TituloAlterado{
			Titulo: titulo, VersaoLida: versoes[i], StatusAnterior: anteriores[i], Historico: historicos[i],
		})
	}

	return gravacao
}

func (c *cenarioBancoTitulos) novoBordero(t *testing.T, titulos ...*domainreceber.Titulo) *domainreceber.Antecipacao {
	t.Helper()

	var bruto int64
	for _, titulo := range titulos {
		bruto += titulo.SaldoCentavos()
	}
	gravacao := c.gravacaoDoBordero(t, bruto-1000, titulos...)
	if err := postgres.NewAntecipacaoRepository(c.banco).Criar(c.ctx, gravacao); err != nil {
		t.Fatalf("criar borderô falhou: %v", err)
	}

	return gravacao.Antecipacao
}

func TestAntecipacaoCriarObterEListar(t *testing.T) {
	c := novoCenarioTitulos(t)
	antecipacoes := postgres.NewAntecipacaoRepository(c.banco)
	primeiro := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	segundo := c.novoTitulo(t, "2", diaDoTitulo(11, 20))

	antecipacao := c.novoBordero(t, primeiro, segundo)
	if primeiro.Versao != 2 || segundo.Versao != 2 {
		t.Fatalf("versões em memória = %d, %d", primeiro.Versao, segundo.Versao)
	}

	obtida, err := antecipacoes.Obter(c.ctx, antecipacao.ID)
	if err != nil || obtida == nil {
		t.Fatalf("Obter = %v, %v", obtida, err)
	}
	if obtida.ValorBrutoCentavos != 300000 || obtida.ValorLiquidoCentavos != 299000 || *obtida.Numero != "B-1" || len(obtida.Titulos) != 2 ||
		!obtida.OperadoEm.Equal(diaDoTitulo(10, 14)) || obtida.Status != domainreceber.StatusAntecipacaoAtiva {
		t.Fatalf("antecipação = %+v", obtida)
	}
	if titulo := c.obterTitulo(t, primeiro.ID); titulo.Status != domainreceber.StatusAntecipado {
		t.Fatalf("título = %+v", titulo)
	}

	vigente, err := antecipacoes.ObterVigenteDoTitulo(c.ctx, segundo.ID)
	if err != nil || vigente == nil || vigente.ID != antecipacao.ID {
		t.Fatalf("vigente do título = %+v, %v", vigente, err)
	}
	if nenhuma, err := antecipacoes.ObterVigenteDoTitulo(c.ctx, uuid.New()); nenhuma != nil || err != nil {
		t.Fatalf("título sem antecipação = %v, %v", nenhuma, err)
	}

	lista, err := antecipacoes.Listar(c.ctx, portsreceber.AntecipacaoFiltro{FornecedorID: &c.fornecedorID, Status: domainreceber.StatusAntecipacaoAtiva})
	if err != nil || len(lista) != 1 || len(lista[0].Titulos) != 2 {
		t.Fatalf("lista = %+v, %v", lista, err)
	}
	if vazia, err := antecipacoes.Listar(c.ctx, portsreceber.AntecipacaoFiltro{EmpresaIDs: []uuid.UUID{}}); err != nil || len(vazia) != 0 {
		t.Fatalf("lista sem acesso = %+v, %v", vazia, err)
	}
}

func TestAntecipacaoNaoAntecipaDuasVezes(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	copia := *titulo

	c.novoBordero(t, titulo)
	err := postgres.NewAntecipacaoRepository(c.banco).Criar(c.ctx, c.gravacaoDoBordero(t, 100, &copia))
	if !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("segundo borderô do mesmo título = %v", err)
	}
}

func TestAntecipacaoConfereClienteEFomentoNaTransacao(t *testing.T) {
	c := novoCenarioTitulos(t)
	antecipacoes := postgres.NewAntecipacaoRepository(c.banco)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))

	if _, err := c.banco.Exec(`UPDATE cliente SET proibe_antecipacao = true WHERE id = $1`, c.clienteID); err != nil {
		t.Fatal(err)
	}
	copia := *titulo
	esperarErro(t, antecipacoes.Criar(c.ctx, c.gravacaoDoBordero(t, 100, &copia)), domain.ErrValidacao, "cliente não permite antecipação: Construtora Alfa")

	if _, err := c.banco.Exec(`UPDATE cliente SET proibe_antecipacao = false WHERE id = $1`, c.clienteID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.banco.Exec(`UPDATE fornecedor SET ativo = false WHERE id = $1`, c.fornecedorID); err != nil {
		t.Fatal(err)
	}
	copia = *titulo
	esperarErro(t, antecipacoes.Criar(c.ctx, c.gravacaoDoBordero(t, 100, &copia)), domain.ErrValidacao, "fomento inativo")

	if obtido := c.obterTitulo(t, titulo.ID); obtido.Status != domainreceber.StatusCarteira || obtido.Versao != 1 {
		t.Fatalf("título não muda quando o borderô falha: %+v", obtido)
	}
}

func estornoDoBordero(t *testing.T, c *cenarioBancoTitulos, antecipacao *domainreceber.Antecipacao) *portsreceber.EstornoDaAntecipacao {
	t.Helper()

	titulos := map[uuid.UUID]*domainreceber.Titulo{}
	estorno := &portsreceber.EstornoDaAntecipacao{Antecipacao: antecipacao, VersaoLida: antecipacao.Versao}
	for _, vinculo := range antecipacao.Titulos {
		titulo := c.obterTitulo(t, vinculo.TituloID)
		titulos[titulo.ID] = titulo
		estorno.Titulos = append(estorno.Titulos, portsreceber.TituloAlterado{Titulo: titulo, VersaoLida: titulo.Versao, StatusAnterior: titulo.Status})
	}
	historicos, err := antecipacao.Estornar("lançado errado", titulos, c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("estorno de teste inválido: %v", err)
	}
	for i := range estorno.Titulos {
		estorno.Titulos[i].Historico = historicos[i]
	}

	return estorno
}

func TestAntecipacaoEstornar(t *testing.T) {
	c := novoCenarioTitulos(t)
	antecipacoes := postgres.NewAntecipacaoRepository(c.banco)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	if err := titulo.EnviarAoBanco(c.contaID, agoraDoTitulo); err != nil {
		t.Fatal(err)
	}
	if err := c.titulos.Atualizar(c.ctx, &portsreceber.Alteracao{Titulo: titulo, VersaoLida: 1, StatusAnterior: domainreceber.StatusCarteira}); err != nil {
		t.Fatal(err)
	}
	criada := c.novoBordero(t, titulo)
	antecipacao, _ := antecipacoes.Obter(c.ctx, criada.ID)

	if err := antecipacoes.Estornar(c.ctx, estornoDoBordero(t, c, antecipacao)); err != nil {
		t.Fatalf("Estornar falhou: %v", err)
	}
	estornada, _ := antecipacoes.Obter(c.ctx, antecipacao.ID)
	if estornada.Status != domainreceber.StatusAntecipacaoEstornada || *estornada.EstornoMotivo != "lançado errado" || estornada.Versao != 2 || estornada.Titulos[0].Vigente {
		t.Fatalf("antecipação estornada = %+v", estornada)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.Status != domainreceber.StatusEmBanco {
		t.Fatalf("título volta para em_banco: %+v", obtido)
	}

	novo := c.novoBordero(t, c.obterTitulo(t, titulo.ID))
	if novo.ID == antecipacao.ID {
		t.Fatal("o título estornado pode ir para outro borderô")
	}
}

func (c *cenarioBancoTitulos) gravacaoDaRecompra(t *testing.T, titulo *domainreceber.Titulo, antecipacao *domainreceber.Antecipacao) *portsreceber.GravacaoRecompra {
	t.Helper()

	versao, anterior := titulo.Versao, titulo.Status
	recompra, historico, err := titulo.Recomprar(antecipacao.Vinculo(titulo.ID), domainreceber.DadosRecompra{
		RecompradoEm: diaDoTitulo(10, 15), EncargosCentavos: 3000, Vencimento: diaDoTitulo(10, 20),
		CategoriaFinanceiraID: c.despesaID, Motivo: "cliente não pagou",
	}, c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("recompra de teste inválida: %v", err)
	}
	solicitacao, historicoDaConta, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: c.empresaID, SolicitanteID: c.solicitanteID, FornecedorID: &antecipacao.FornecedorID,
		CategoriaFinanceiraID: c.despesaID, Origem: domainsolicitacao.OrigemRecompra, OrigemID: recompra.ID,
		ValorCentavos: recompra.ValorCentavos, Vencimento: recompra.Vencimento, Observacao: recompra.Observacao(titulo),
	}, agoraDoTitulo)
	if err != nil {
		t.Fatalf("conta a pagar de teste inválida: %v", err)
	}

	return &portsreceber.GravacaoRecompra{
		Titulo:      portsreceber.TituloAlterado{Titulo: titulo, VersaoLida: versao, StatusAnterior: anterior, Historico: historico},
		Recompra:    recompra,
		ContaAPagar: portsreceber.ContaAPagarGerada{Solicitacao: solicitacao, Historico: historicoDaConta},
	}
}

func TestRecompraGeraContaAPagarNaMesmaTransacao(t *testing.T) {
	c := novoCenarioTitulos(t)
	antecipacoes := postgres.NewAntecipacaoRepository(c.banco)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	antecipacao := c.novoBordero(t, titulo)

	gravacao := c.gravacaoDaRecompra(t, c.obterTitulo(t, titulo.ID), antecipacao)
	if err := antecipacoes.Recomprar(c.ctx, gravacao); err != nil {
		t.Fatalf("Recomprar falhou: %v", err)
	}

	solicitacao, err := c.solicitacoes.Obter(c.ctx, gravacao.ContaAPagar.Solicitacao.ID)
	if err != nil || solicitacao == nil {
		t.Fatalf("conta a pagar = %v, %v", solicitacao, err)
	}
	if solicitacao.Origem != domainsolicitacao.OrigemRecompra || *solicitacao.OrigemID != gravacao.Recompra.ID ||
		solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao || solicitacao.Valor.Centavos() != 153000 ||
		*solicitacao.FornecedorID != c.fornecedorID || *solicitacao.CategoriaFinanceiraID != c.despesaID {
		t.Fatalf("solicitação = %+v", solicitacao)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.Status != domainreceber.StatusCarteira {
		t.Fatalf("título recomprado = %+v", obtido)
	}
	if vigente, _ := antecipacoes.ObterVigenteDoTitulo(c.ctx, titulo.ID); vigente != nil {
		t.Fatalf("o título recomprado sai do borderô: %+v", vigente)
	}
	if gravacao.Recompra.SolicitacaoID != solicitacao.ID {
		t.Fatalf("recompra em memória sem a conta: %+v", gravacao.Recompra)
	}

	obtida, _ := antecipacoes.Obter(c.ctx, antecipacao.ID)
	esperarErro(t, antecipacoes.Estornar(c.ctx, &portsreceber.EstornoDaAntecipacao{Antecipacao: obtida, VersaoLida: obtida.Versao}),
		domain.ErrConflito, "recebido ou recomprado")
}

func TestRecompraComCategoriaDeReceitaDesfazTudo(t *testing.T) {
	c := novoCenarioTitulos(t)
	antecipacoes := postgres.NewAntecipacaoRepository(c.banco)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	antecipacao := c.novoBordero(t, titulo)

	gravacao := c.gravacaoDaRecompra(t, c.obterTitulo(t, titulo.ID), antecipacao)
	gravacao.ContaAPagar.Solicitacao.CategoriaFinanceiraID = &c.receitaID
	if err := antecipacoes.Recomprar(c.ctx, gravacao); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("recompra com categoria de receita = %v", err)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.Status != domainreceber.StatusAntecipado {
		t.Fatalf("o título continua antecipado: %+v", obtido)
	}
	if vigente, _ := antecipacoes.ObterVigenteDoTitulo(c.ctx, titulo.ID); vigente == nil {
		t.Fatal("o vínculo continua vigente")
	}
}

func TestReceberAntecipadoEmContaPropriaGeraORepasse(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "1", diaDoTitulo(11, 10))
	antecipacao := c.novoBordero(t, titulo)
	titulo = c.obterTitulo(t, titulo.ID)

	versao, anterior := titulo.Versao, titulo.Status
	recebimento, historico, err := titulo.Receber(domainreceber.DadosRecebimento{
		ContaBancariaID: &c.contaID, RecebidoEm: diaDoTitulo(10, 15), ValorCentavos: 150000,
		Repasse: &domainreceber.DadosRepasse{Vencimento: diaDoTitulo(10, 18), CategoriaFinanceiraID: c.despesaID},
	}, c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatal(err)
	}
	recebimento.Repasse.AntecipacaoID = antecipacao.ID
	solicitacao, historicoDaConta, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: c.empresaID, SolicitanteID: c.solicitanteID, FornecedorID: &c.fornecedorID, CategoriaFinanceiraID: c.despesaID,
		Origem: domainsolicitacao.OrigemRepasseFomento, OrigemID: recebimento.Repasse.ID, ValorCentavos: 150000,
		Vencimento: diaDoTitulo(10, 18), Observacao: recebimento.Repasse.Observacao(titulo),
	}, agoraDoTitulo)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.titulos.Receber(c.ctx, &portsreceber.Baixa{
		Titulo: titulo, VersaoLida: versao, StatusAnterior: anterior, Recebimento: recebimento, Historico: historico,
		ContaDoRepasse: &portsreceber.ContaAPagarGerada{Solicitacao: solicitacao, Historico: historicoDaConta},
	}); err != nil {
		t.Fatalf("Receber com repasse falhou: %v", err)
	}

	recebimentos, _ := c.titulos.ListarRecebimentos(c.ctx, titulo.ID)
	if len(recebimentos) != 1 || recebimentos[0].Repasse == nil || recebimentos[0].Repasse.SolicitacaoID != solicitacao.ID ||
		recebimentos[0].Repasse.AntecipacaoID != antecipacao.ID || recebimentos[0].Repasse.ValorCentavos != 150000 {
		t.Fatalf("recebimentos = %+v", recebimentos)
	}
	gravada, _ := c.solicitacoes.Obter(c.ctx, solicitacao.ID)
	if gravada == nil || gravada.Origem != domainsolicitacao.OrigemRepasseFomento {
		t.Fatalf("conta do repasse = %+v", gravada)
	}

	recebimentos[0].Repasse = nil
	esperarErro(t, c.estornar(t, c.obterTitulo(t, titulo.ID), recebimentos[0]), domain.ErrConflito, "já foi estornado")
}

func TestSolicitacaoGerarDeOrigemEIdempotente(t *testing.T) {
	c := novoCenarioTitulos(t)
	dados := domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: c.empresaID, SolicitanteID: c.solicitanteID, FornecedorID: &c.fornecedorID, CategoriaFinanceiraID: c.despesaID,
		Origem: domainsolicitacao.OrigemRecompra, OrigemID: uuid.New(), ValorCentavos: 1000,
		Vencimento: diaDoTitulo(10, 20), Observacao: "recompra",
	}
	primeira, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, agoraDoTitulo)
	if err != nil {
		t.Fatal(err)
	}
	if criada, err := c.solicitacoes.GerarDeOrigem(c.ctx, primeira, historico); err != nil || !criada {
		t.Fatalf("primeira geração = %v, %v", criada, err)
	}

	segunda, historico, _ := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, agoraDoTitulo)
	if criada, err := c.solicitacoes.GerarDeOrigem(c.ctx, segunda, historico); err != nil || criada {
		t.Fatalf("segunda geração da mesma origem = %v, %v", criada, err)
	}

	manual := *primeira
	manual.Origem = domainsolicitacao.OrigemManual
	if _, err := c.solicitacoes.GerarDeOrigem(c.ctx, &manual, historico); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("origem manual = %v", err)
	}
}
