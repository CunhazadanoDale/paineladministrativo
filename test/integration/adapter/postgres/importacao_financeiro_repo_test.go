package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var _ portsimportacao.CategoriaFinanceiraRepository = (*postgres.CategoriaFinanceiraRepository)(nil)

func (c *cenarioBancoImportacoes) compraMercadoria(t *testing.T) uuid.UUID {
	t.Helper()

	categorias := postgres.NewCategoriaFinanceiraRepository(c.banco)
	classificacao, err := domainfinanceiro.NovaClassificacao("despesa", "compra_mercadoria")
	if err != nil {
		t.Fatalf("classificação de teste inválida: %v", err)
	}
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Compras"}, &classificacao, nil)
	if err != nil {
		t.Fatalf("raiz de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, raiz); err != nil {
		t.Fatalf("criação da raiz falhou: %v", err)
	}
	sub, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Importação"}, nil, raiz)
	if err != nil {
		t.Fatalf("subcategoria de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, sub); err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	return sub.ID
}

func (c *cenarioBancoImportacoes) gravarOperacao(t *testing.T, importacao *domainimportacao.Importacao, gravacao portsimportacao.Gravacao) error {
	t.Helper()

	gravacao.Importacao = importacao
	gravacao.VersaoLida = importacao.Versao
	gravacao.StatusAnterior = importacao.Status

	return c.importacoes.Gravar(c.ctx, &gravacao)
}

func contaDaParcela(t *testing.T, importacao *domainimportacao.Importacao, usuarioID uuid.UUID, parcela *domainimportacao.Parcela, categoriaID uuid.UUID) portsimportacao.ContaAPagarGerada {
	t.Helper()

	fornecedorID := importacao.FornecedorID
	solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: importacao.EmpresaID, SolicitanteID: usuarioID, FornecedorID: &fornecedorID,
		CategoriaFinanceiraID: categoriaID, Origem: domainsolicitacao.OrigemImportacaoParcela, OrigemID: parcela.ID,
		ValorCentavos: *parcela.ValorCentavos, Vencimento: *parcela.Vencimento, Observacao: "parcela",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("conta a pagar de teste inválida: %v", err)
	}
	parcela.SolicitacaoID = &solicitacao.ID

	return portsimportacao.ContaAPagarGerada{Solicitacao: solicitacao, Historico: historico}
}

func TestImportacaoFechaCambioComContaAPagarNaMesmaTransacao(t *testing.T) {
	c := novoCenarioImportacoes(t)
	categoriaID := c.compraMercadoria(t)
	importacao := c.nova(t, c.mia)
	agora := time.Now().UTC()

	parcela, err := importacao.IncluirParcela(c.usuarioID, domainimportacao.DadosParcela{
		Descricao: "Sinal", ValorMoedaCentesimos: 900000, VencimentoPrevisto: agora,
	}, agora)
	if err != nil {
		t.Fatalf("IncluirParcela: %v", err)
	}
	if err := c.gravarOperacao(t, importacao, portsimportacao.Gravacao{ParcelasNovas: []*domainimportacao.Parcela{parcela}}); err != nil {
		t.Fatalf("gravar parcela: %v", err)
	}

	cambio := int64(780_000)
	if _, err := importacao.FecharCambio(parcela.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: agora}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	conta := contaDaParcela(t, importacao, c.usuarioID, parcela, categoriaID)
	gravacao := portsimportacao.Gravacao{ParcelasAlteradas: []*domainimportacao.Parcela{parcela}, ContasAPagar: []portsimportacao.ContaAPagarGerada{conta}}
	if err := c.gravarOperacao(t, importacao, gravacao); err != nil {
		t.Fatalf("gravar câmbio: %v", err)
	}

	lida := c.obter(t, importacao.ID)
	fechada := lida.Parcela(parcela.ID)
	if lida.Versao != 3 || fechada.Status != domainimportacao.StatusParcelaCambioFechado || *fechada.ValorCentavos != 702000 ||
		fechada.Cambio.Micro() != cambio || *fechada.SolicitacaoID != conta.Solicitacao.ID ||
		*fechada.StatusContaAPagar != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("parcela lida = %+v", fechada)
	}
	solicitacao, err := postgres.NewSolicitacaoRepository(c.banco).Obter(c.ctx, conta.Solicitacao.ID)
	if err != nil || solicitacao == nil || solicitacao.Origem != domainsolicitacao.OrigemImportacaoParcela || *solicitacao.OrigemID != parcela.ID {
		t.Fatalf("solicitação gerada = %+v (%v)", solicitacao, err)
	}

	repetida := contaDaParcela(t, lida, c.usuarioID, lida.Parcela(parcela.ID), categoriaID)
	if err := c.gravarOperacao(t, lida, portsimportacao.Gravacao{ContasAPagar: []portsimportacao.ContaAPagarGerada{repetida}}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segunda conta da mesma parcela: erro = %v", err)
	}
	if lida.Versao != 3 {
		t.Errorf("versão mudou com a gravação recusada: %d", lida.Versao)
	}

	if err := lida.Cancelar("desistência", agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("domínio aceitou cancelar com conta pendente: %v", err)
	}
	motivo := "desistência"
	lida.Status = domainimportacao.StatusCancelada
	lida.CanceladaMotivo = &motivo
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: lida, VersaoLida: 3, StatusAnterior: domainimportacao.StatusAberta,
		ContasExigidasEncerradas: lida.ContasAPagarVinculadas(),
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("banco aceitou cancelar com conta pendente: %v", err)
	}
}

func TestImportacaoReabreParcelaECancelaCusto(t *testing.T) {
	c := novoCenarioImportacoes(t)
	categoriaID := c.compraMercadoria(t)
	importacao := c.nova(t, c.mia)
	agora := time.Now().UTC()

	parcela, _ := importacao.IncluirParcela(c.usuarioID, domainimportacao.DadosParcela{Descricao: "Sinal", ValorMoedaCentesimos: 900000, VencimentoPrevisto: agora}, agora)
	cambio := int64(780_000)
	if _, err := importacao.FecharCambio(parcela.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: agora}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	conta := contaDaParcela(t, importacao, c.usuarioID, parcela, categoriaID)
	if err := c.gravarOperacao(t, importacao, portsimportacao.Gravacao{
		ParcelasNovas: []*domainimportacao.Parcela{parcela}, ContasAPagar: []portsimportacao.ContaAPagarGerada{conta},
	}); err != nil {
		t.Fatalf("gravar parcela fechada: %v", err)
	}

	custo, err := importacao.LancarCusto(c.usuarioID, domainimportacao.DadosCusto{
		Tipo: domainimportacao.TipoCustoFreteInternacional, FornecedorID: c.fornecedorID, ValorCentavos: 1250000,
		Vencimento: agora, CompoeCusto: true, CategoriaFinanceiraID: categoriaID,
	}, agora)
	if err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}
	fornecedorID := custo.FornecedorID
	solicitacaoDoCusto, historicoDoCusto, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: importacao.EmpresaID, SolicitanteID: c.usuarioID, FornecedorID: &fornecedorID, CategoriaFinanceiraID: categoriaID,
		Origem: domainsolicitacao.OrigemImportacaoCusto, OrigemID: custo.ID, ValorCentavos: custo.ValorCentavos, Vencimento: custo.Vencimento,
		Observacao: "frete",
	}, agora)
	if err != nil {
		t.Fatalf("conta do custo: %v", err)
	}
	custo.SolicitacaoID = solicitacaoDoCusto.ID
	if err := c.gravarOperacao(t, importacao, portsimportacao.Gravacao{
		CustosNovos:  []*domainimportacao.Custo{custo},
		ContasAPagar: []portsimportacao.ContaAPagarGerada{{Solicitacao: solicitacaoDoCusto, Historico: historicoDoCusto}},
	}); err != nil {
		t.Fatalf("gravar custo: %v", err)
	}

	lida := c.obter(t, importacao.ID)
	if len(lida.Custos) != 1 || lida.Custos[0].Descricao != "Frete internacional" || !lida.Custos[0].CompoeCusto ||
		*lida.Custos[0].StatusContaAPagar != domainsolicitacao.StatusPendenteAprovacao {
		t.Fatalf("custos lidos = %+v", lida.Custos)
	}

	antigaForcada := *lida.Parcela(parcela.ID)
	antigaForcada.Status = domainimportacao.StatusParcelaCancelada
	if err := c.gravarOperacao(t, lida, portsimportacao.Gravacao{
		ParcelasAlteradas:        []*domainimportacao.Parcela{&antigaForcada},
		ContasExigidasEncerradas: []uuid.UUID{conta.Solicitacao.ID},
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir com conta pendente no banco: erro = %v", err)
	}

	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'cancelado', cancelado_em = now() WHERE id = ANY($1::uuid[])`,
		"{"+conta.Solicitacao.ID.String()+","+solicitacaoDoCusto.ID.String()+"}"); err != nil {
		t.Fatalf("cancelar contas: %v", err)
	}
	lida = c.obter(t, importacao.ID)
	antiga, nova, err := lida.ReabrirParcela(parcela.ID, c.usuarioID, agora)
	if err != nil {
		t.Fatalf("ReabrirParcela: %v", err)
	}
	cancelado, err := lida.CancelarCusto(custo.ID, c.usuarioID, agora)
	if err != nil {
		t.Fatalf("CancelarCusto: %v", err)
	}
	if err := c.gravarOperacao(t, lida, portsimportacao.Gravacao{
		ParcelasAlteradas:        []*domainimportacao.Parcela{antiga},
		ParcelasNovas:            []*domainimportacao.Parcela{nova},
		CustosCancelados:         []*domainimportacao.Custo{cancelado},
		ContasExigidasEncerradas: []uuid.UUID{conta.Solicitacao.ID, solicitacaoDoCusto.ID},
	}); err != nil {
		t.Fatalf("gravar reabertura: %v", err)
	}

	final := c.obter(t, importacao.ID)
	if len(final.Parcelas) != 2 || final.Parcela(parcela.ID).Status != domainimportacao.StatusParcelaCancelada ||
		final.Parcela(nova.ID).Status != domainimportacao.StatusParcelaAberta || final.Custos[0].Ativo() {
		t.Errorf("importação depois de reabrir = parcelas %+v, custos %+v", final.Parcelas, final.Custos)
	}

	removida, err := final.RemoverParcela(nova.ID, agora)
	if err != nil {
		t.Fatalf("RemoverParcela: %v", err)
	}
	if err := c.gravarOperacao(t, final, portsimportacao.Gravacao{ParcelasRemovidas: []uuid.UUID{removida.ID}}); err != nil {
		t.Fatalf("gravar remoção: %v", err)
	}
	if len(c.obter(t, importacao.ID).Parcelas) != 1 {
		t.Error("parcela aberta não foi removida")
	}
}
