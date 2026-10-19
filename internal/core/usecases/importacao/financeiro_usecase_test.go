package importacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

func (c *cenario) incluirParcela(t *testing.T, importacao *domainimportacao.Importacao, centesimos int64) *domainimportacao.Importacao {
	t.Helper()

	visao, err := c.financas.IncluirParcela(c.ctx, portsin.IncluirParcelaInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		Dados: portsin.DadosParcelaInput{Descricao: "Sinal", ValorMoedaCentesimos: centesimos, VencimentoPrevisto: time.Now()},
	})
	if err != nil {
		t.Fatalf("IncluirParcela: %v", err)
	}

	return visao.Importacao
}

func (c *cenario) fecharCambio(t *testing.T, importacao *domainimportacao.Importacao, parcelaID uuid.UUID, cambio int64) *domainimportacao.Importacao {
	t.Helper()

	visao, err := c.financas.FecharCambio(c.ctx, portsin.FecharCambioInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, ParcelaID: parcelaID, Versao: importacao.Versao,
		CambioMicro: &cambio, Vencimento: time.Now().AddDate(0, 0, 3), CategoriaFinanceiraID: c.compraMercadoria,
	})
	if err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}

	return visao.Importacao
}

func TestFecharCambioGeraContaAPagarDaParcela(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	parcela := importacao.Parcelas[0]

	importacao = c.fecharCambio(t, importacao, parcela.ID, 780_000)
	fechada := importacao.Parcela(parcela.ID)
	if fechada.Status != domainimportacao.StatusParcelaCambioFechado || *fechada.ValorCentavos != 702000 || fechada.SolicitacaoID == nil {
		t.Fatalf("parcela fechada = %+v", fechada)
	}
	if importacao.Versao != 3 {
		t.Errorf("versão = %d, esperado 3", importacao.Versao)
	}

	conta := c.m.contas[*fechada.SolicitacaoID]
	if conta == nil || conta.Origem != domainsolicitacao.OrigemImportacaoParcela || *conta.OrigemID != parcela.ID ||
		conta.Valor.Centavos() != 702000 || *conta.FornecedorID != c.fornecedorChines.ID || conta.EmpresaID != c.mia ||
		*conta.CategoriaFinanceiraID != c.compraMercadoria || conta.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("conta a pagar = %+v", conta)
	}

	historico, _ := c.importacoes.ListarHistorico(c.ctx, importacao.ID, c.financeiro)
	if ultimo := historico[len(historico)-1]; ultimo.Acao != domainimportacao.AcaoParcelaCambioFechado || ultimo.Detalhe != "Sinal: câmbio 0,780000, R$ 7020,00" {
		t.Errorf("histórico = %+v", ultimo)
	}
}

func TestFecharCambioValidaCategoria(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	cambio := int64(780_000)

	for nome, categoria := range map[string]uuid.UUID{"sem categoria": uuid.Nil, "receita": c.receita, "inexistente": uuid.New()} {
		t.Run(nome, func(t *testing.T) {
			_, err := c.financas.FecharCambio(c.ctx, portsin.FecharCambioInput{
				UsuarioID: c.financeiro, ImportacaoID: importacao.ID, ParcelaID: importacao.Parcelas[0].ID, Versao: importacao.Versao,
				CambioMicro: &cambio, Vencimento: time.Now(), CategoriaFinanceiraID: categoria,
			})
			if !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}
}

func TestParcelasExigemFinanceiroVersaoEAcesso(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	dados := portsin.DadosParcelaInput{Descricao: "Sinal", ValorMoedaCentesimos: 100, VencimentoPrevisto: time.Now()}

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		versao    int
		sentinela error
	}{
		{"comercial", c.comercial, 1, domain.ErrPermissao},
		{"outra empresa", c.financeiroRed, 1, domain.ErrPermissao},
		{"versão antiga", c.financeiro, 9, domain.ErrConflito},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.financas.IncluirParcela(c.ctx, portsin.IncluirParcelaInput{
				UsuarioID: caso.usuario, ImportacaoID: importacao.ID, Versao: caso.versao, Dados: dados,
			})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro = %v, esperado %v", err, caso.sentinela)
			}
		})
	}
}

func TestAlterarERemoverParcela(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	parcelaID := importacao.Parcelas[0].ID

	visao, err := c.financas.AlterarParcela(c.ctx, portsin.AlterarParcelaInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, ParcelaID: parcelaID, Versao: 2,
		Dados: portsin.DadosParcelaInput{Descricao: "Sinal 30%", ValorMoedaCentesimos: 900001, VencimentoPrevisto: time.Now()},
	})
	if err != nil || visao.Importacao.Parcela(parcelaID).Valor.Centesimos() != 900001 {
		t.Fatalf("AlterarParcela = %+v (%v)", visao, err)
	}
	if c.m.gravacoes[len(c.m.gravacoes)-1].ParcelasAlteradas[0].ID != parcelaID {
		t.Errorf("gravação da alteração = %+v", c.m.gravacoes[len(c.m.gravacoes)-1])
	}

	visao, err = c.financas.RemoverParcela(c.ctx, portsin.ParcelaInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, ParcelaID: parcelaID, Versao: 3,
	})
	if err != nil || len(visao.Importacao.Parcelas) != 0 || c.m.gravacoes[len(c.m.gravacoes)-1].ParcelasRemovidas[0] != parcelaID {
		t.Errorf("RemoverParcela = %+v (%v)", visao, err)
	}
}

func TestReabrirParcelaDepoisDeCancelarAConta(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	parcelaID := importacao.Parcelas[0].ID
	importacao = c.fecharCambio(t, importacao, parcelaID, 780_000)
	contaID := *importacao.Parcela(parcelaID).SolicitacaoID

	reabrir := portsin.ParcelaInput{UsuarioID: c.financeiro, ImportacaoID: importacao.ID, ParcelaID: parcelaID, Versao: importacao.Versao}
	if _, err := c.financas.ReabrirParcela(c.ctx, reabrir); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir com conta pendente: erro = %v", err)
	}

	c.m.contas[contaID].Status = domainsolicitacao.StatusCancelado
	visao, err := c.financas.ReabrirParcela(c.ctx, reabrir)
	if err != nil {
		t.Fatalf("ReabrirParcela: %v", err)
	}
	if len(visao.Importacao.Parcelas) != 2 || visao.Importacao.Parcela(parcelaID).Status != domainimportacao.StatusParcelaCancelada ||
		visao.Importacao.Parcelas[1].Status != domainimportacao.StatusParcelaAberta {
		t.Errorf("parcelas depois de reabrir = %+v", visao.Importacao.Parcelas)
	}
	gravacao := c.m.gravacoes[len(c.m.gravacoes)-1]
	if len(gravacao.ContasExigidasEncerradas) != 1 || gravacao.ContasExigidasEncerradas[0] != contaID ||
		len(gravacao.ParcelasNovas) != 1 || len(gravacao.ParcelasAlteradas) != 1 {
		t.Errorf("gravação da reabertura = %+v", gravacao)
	}

	nova := c.fecharCambio(t, visao.Importacao, visao.Importacao.Parcelas[1].ID, 800_000)
	if *nova.Parcelas[1].SolicitacaoID == contaID || len(c.m.contas) != 2 {
		t.Errorf("nova conta a pagar = %+v", nova.Parcelas[1])
	}
}

func TestLancarECancelarCusto(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	lancamento := portsin.LancarCustoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 1, Tipo: "frete_internacional",
		FornecedorID: c.fornecedorEuropeu.ID, ValorCentavos: 1250000, Vencimento: time.Now().AddDate(0, 0, 10),
		CompoeCusto: true, CategoriaFinanceiraID: c.compraMercadoria,
	}

	invalidos := map[string]func(*portsin.LancarCustoInput){
		"tipo inválido":        func(i *portsin.LancarCustoInput) { i.Tipo = "multa" },
		"fornecedor inativo":   func(i *portsin.LancarCustoInput) { i.FornecedorID = c.fornecedorInativo.ID },
		"sem fornecedor":       func(i *portsin.LancarCustoInput) { i.FornecedorID = uuid.Nil },
		"categoria de receita": func(i *portsin.LancarCustoInput) { i.CategoriaFinanceiraID = c.receita },
		"valor zero":           func(i *portsin.LancarCustoInput) { i.ValorCentavos = 0 },
	}
	for nome, alterar := range invalidos {
		t.Run(nome, func(t *testing.T) {
			entrada := lancamento
			alterar(&entrada)
			if _, err := c.financas.LancarCusto(c.ctx, entrada); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}

	visao, err := c.financas.LancarCusto(c.ctx, lancamento)
	if err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}
	custo := visao.Importacao.Custos[0]
	conta := c.m.contas[custo.SolicitacaoID]
	if custo.Descricao != "Frete internacional" || !custo.CompoeCusto || conta == nil ||
		conta.Origem != domainsolicitacao.OrigemImportacaoCusto || *conta.OrigemID != custo.ID || *conta.FornecedorID != c.fornecedorEuropeu.ID ||
		conta.Valor.Centavos() != 1250000 {
		t.Fatalf("custo %+v, conta %+v", custo, conta)
	}

	cancelar := portsin.CustoInput{UsuarioID: c.financeiro, ImportacaoID: importacao.ID, CustoID: custo.ID, Versao: 2}
	if _, err := c.financas.CancelarCusto(c.ctx, cancelar); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar com conta pendente: erro = %v", err)
	}
	c.m.contas[custo.SolicitacaoID].Status = domainsolicitacao.StatusRejeitado
	visao, err = c.financas.CancelarCusto(c.ctx, cancelar)
	if err != nil || visao.Importacao.Custos[0].Ativo() || visao.Importacao.Custos[0].CanceladoPor == nil {
		t.Errorf("CancelarCusto = %+v (%v)", visao, err)
	}
}

func TestCancelarImportacaoExigeContasEncerradas(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	importacao = c.fecharCambio(t, importacao, importacao.Parcelas[0].ID, 780_000)

	cancelar := portsin.CancelarImportacaoInput{UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao, Motivo: "desistência"}
	if _, err := c.importacoes.Cancelar(c.ctx, cancelar); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar com conta pendente: erro = %v", err)
	}

	c.m.contas[*importacao.Parcelas[0].SolicitacaoID].Status = domainsolicitacao.StatusCancelado
	visao, err := c.importacoes.Cancelar(c.ctx, cancelar)
	if err != nil || visao.Importacao.Status != domainimportacao.StatusCancelada {
		t.Fatalf("Cancelar = %+v (%v)", visao, err)
	}
	if gravacao := c.m.gravacoes[len(c.m.gravacoes)-1]; len(gravacao.ContasExigidasEncerradas) != 1 {
		t.Errorf("gravação do cancelamento = %+v", gravacao)
	}
}

func TestCustoPrevisto(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())

	if _, err := c.financas.CustoPrevisto(c.ctx, importacao.ID, c.financeiro, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem câmbio: erro = %v", err)
	}
	cambio := int64(780_000)
	calculo, err := c.financas.CustoPrevisto(c.ctx, importacao.ID, c.financeiro, &cambio)
	if err != nil || calculo.ValorMercadoriaCentavos != 2340000 || len(calculo.Itens) != 2 || *calculo.Itens[0].CustoUnitarioCentavos != 2340 {
		t.Errorf("custo previsto = %+v (%v)", calculo, err)
	}
	if _, err := c.financas.CustoPrevisto(c.ctx, importacao.ID, c.comercial, &cambio); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial: erro = %v", err)
	}
}
