package importacao_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

var agora = time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)

func dia(ano int, mes time.Month, d int) time.Time {
	return time.Date(ano, mes, d, 0, 0, 0, 0, time.UTC)
}

func ponteiro[T any](valor T) *T {
	return &valor
}

func dadosDeTeste() domainimportacao.DadosImportacao {
	return domainimportacao.DadosImportacao{
		FornecedorID: uuid.New(),
		Moeda:        domainfornecedor.MoedaDolar,
		Referencia:   ponteiro("  PI-2026-001 "),
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: uuid.New(), Quantidade: 1000, ValorMoedaCentesimos: 450000},
			{ProdutoID: uuid.New(), Quantidade: 200, ValorMoedaCentesimos: 0},
		},
	}
}

func novaImportacaoDeTeste(t *testing.T) *domainimportacao.Importacao {
	t.Helper()

	importacao, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), dadosDeTeste(), agora)
	if err != nil {
		t.Fatalf("NovaImportacao: %v", err)
	}

	return importacao
}

func embarcada(t *testing.T) *domainimportacao.Importacao {
	t.Helper()

	importacao := novaImportacaoDeTeste(t)
	if err := importacao.Embarcar(domainimportacao.EmbarqueInput{
		EmbarcadaEm: dia(2026, 10, 1), ChegadaPrevista: dia(2026, 11, 20), DocumentoEmbarque: ponteiro("BL-778"),
	}, agora); err != nil {
		t.Fatalf("Embarcar: %v", err)
	}

	return importacao
}

func TestNovaImportacaoSomaOsItensEComecaAberta(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)

	if importacao.Status != domainimportacao.StatusAberta || importacao.Versao != 1 {
		t.Errorf("status %q e versão %d", importacao.Status, importacao.Versao)
	}
	if importacao.ValorTotal.Centesimos() != 450000 {
		t.Errorf("valor total = %d, esperado 450000", importacao.ValorTotal.Centesimos())
	}
	if importacao.Referencia == nil || *importacao.Referencia != "PI-2026-001" {
		t.Errorf("referência = %v", importacao.Referencia)
	}
	for _, item := range importacao.Itens {
		if item.ImportacaoID != importacao.ID || item.ID == uuid.Nil {
			t.Errorf("item %+v sem vínculo com a importação", item)
		}
	}
}

func TestNovaImportacaoValida(t *testing.T) {
	repetido := uuid.New()
	casos := []struct {
		nome    string
		alterar func(*domainimportacao.DadosImportacao)
	}{
		{"sem fornecedor", func(d *domainimportacao.DadosImportacao) { d.FornecedorID = uuid.Nil }},
		{"moeda inválida", func(d *domainimportacao.DadosImportacao) { d.Moeda = "JPY" }},
		{"sem itens", func(d *domainimportacao.DadosImportacao) { d.Itens = nil }},
		{"item sem produto", func(d *domainimportacao.DadosImportacao) { d.Itens[0].ProdutoID = uuid.Nil }},
		{"quantidade zero", func(d *domainimportacao.DadosImportacao) { d.Itens[0].Quantidade = 0 }},
		{"quantidade acima de int32", func(d *domainimportacao.DadosImportacao) { d.Itens[0].Quantidade = math.MaxInt32 + 1 }},
		{"valor negativo", func(d *domainimportacao.DadosImportacao) { d.Itens[0].ValorMoedaCentesimos = -1 }},
		{"soma acima do limite", func(d *domainimportacao.DadosImportacao) {
			d.Itens[0].ValorMoedaCentesimos = math.MaxInt64
			d.Itens[1].ValorMoedaCentesimos = 1
		}},
		{"produto repetido", func(d *domainimportacao.DadosImportacao) {
			d.Itens[0].ProdutoID = repetido
			d.Itens[1].ProdutoID = repetido
		}},
		{"referência longa", func(d *domainimportacao.DadosImportacao) { d.Referencia = ponteiro(strings.Repeat("a", 61)) }},
		{"observações longas", func(d *domainimportacao.DadosImportacao) { d.Observacoes = strings.Repeat("a", 2001) }},
		{"chegada antes do embarque previsto", func(d *domainimportacao.DadosImportacao) {
			d.EmbarquePrevisto = ponteiro(dia(2026, 11, 10))
			d.ChegadaPrevista = ponteiro(dia(2026, 11, 9))
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeTeste()
			caso.alterar(&dados)
			if _, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), dados, agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}

	if _, err := domainimportacao.NovaImportacao(uuid.Nil, uuid.New(), dadosDeTeste(), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem empresa: erro = %v", err)
	}
	if _, err := domainimportacao.NovaImportacao(uuid.New(), uuid.Nil, dadosDeTeste(), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem usuário: erro = %v", err)
	}
}

func TestAlterarMantemOIdDoItemDoMesmoProduto(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	mantido := importacao.Itens[0]

	dados := domainimportacao.DadosImportacao{
		FornecedorID: importacao.FornecedorID,
		Moeda:        importacao.Moeda,
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: mantido.ProdutoID, Quantidade: 1200, ValorMoedaCentesimos: 540000},
			{ProdutoID: uuid.New(), Quantidade: 10, ValorMoedaCentesimos: 1000},
		},
	}
	if err := importacao.Alterar(dados, agora.Add(time.Hour)); err != nil {
		t.Fatalf("Alterar: %v", err)
	}

	if importacao.Itens[0].ID != mantido.ID || importacao.Itens[0].Quantidade != 1200 {
		t.Errorf("item mantido = %+v, esperado o id %v", importacao.Itens[0], mantido.ID)
	}
	if importacao.Itens[1].ID == mantido.ID || importacao.ValorTotal.Centesimos() != 541000 || importacao.Referencia != nil {
		t.Errorf("importação alterada = %+v", importacao)
	}
}

func TestAlterarDepoisDoEmbarque(t *testing.T) {
	importacao := embarcada(t)
	dados := dadosDeTeste()
	dados.FornecedorID = importacao.FornecedorID
	dados.ChegadaPrevista = ponteiro(dia(2026, 12, 5))

	if err := importacao.Alterar(dados, agora); err != nil {
		t.Fatalf("Alterar embarcada: %v", err)
	}
	if !importacao.ChegadaPrevista.Equal(dia(2026, 12, 5)) {
		t.Errorf("chegada prevista = %v", importacao.ChegadaPrevista)
	}

	outroFornecedor := dados
	outroFornecedor.FornecedorID = uuid.New()
	if err := importacao.Alterar(outroFornecedor, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("trocar fornecedor embarcada: erro = %v", err)
	}
	outraMoeda := dados
	outraMoeda.Moeda = domainfornecedor.MoedaEuro
	if err := importacao.Alterar(outraMoeda, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("trocar moeda embarcada: erro = %v", err)
	}
	semPrevisao := dados
	semPrevisao.ChegadaPrevista = nil
	if err := importacao.Alterar(semPrevisao, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tirar chegada prevista: erro = %v", err)
	}
	antesDoEmbarque := dados
	antesDoEmbarque.ChegadaPrevista = ponteiro(dia(2026, 9, 30))
	if err := importacao.Alterar(antesDoEmbarque, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("chegada antes do embarque: erro = %v", err)
	}
}

func TestEmbarcar(t *testing.T) {
	importacao := embarcada(t)

	if importacao.Status != domainimportacao.StatusEmbarcada || !importacao.EmbarcadaEm.Equal(dia(2026, 10, 1)) ||
		!importacao.ChegadaPrevista.Equal(dia(2026, 11, 20)) || *importacao.DocumentoEmbarque != "BL-778" {
		t.Errorf("importação embarcada = %+v", importacao)
	}
	if err := importacao.Embarcar(domainimportacao.EmbarqueInput{EmbarcadaEm: dia(2026, 10, 1), ChegadaPrevista: dia(2026, 11, 20)}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("embarcar de novo: erro = %v", err)
	}

	casos := []struct {
		nome  string
		input domainimportacao.EmbarqueInput
	}{
		{"sem data", domainimportacao.EmbarqueInput{ChegadaPrevista: dia(2026, 11, 20)}},
		{"no futuro", domainimportacao.EmbarqueInput{EmbarcadaEm: dia(2026, 10, 15), ChegadaPrevista: dia(2026, 11, 20)}},
		{"sem chegada prevista", domainimportacao.EmbarqueInput{EmbarcadaEm: dia(2026, 10, 1)}},
		{"chegada antes do embarque", domainimportacao.EmbarqueInput{EmbarcadaEm: dia(2026, 10, 1), ChegadaPrevista: dia(2026, 9, 30)}},
		{"documento longo", domainimportacao.EmbarqueInput{
			EmbarcadaEm: dia(2026, 10, 1), ChegadaPrevista: dia(2026, 11, 20), DocumentoEmbarque: ponteiro(strings.Repeat("b", 61)),
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := novaImportacaoDeTeste(t).Embarcar(caso.input, agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}
}

func TestEmbarcarHojeEmBrasilia(t *testing.T) {
	noiteEmBrasilia := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)

	err := novaImportacaoDeTeste(t).Embarcar(domainimportacao.EmbarqueInput{
		EmbarcadaEm: dia(2026, 10, 15), ChegadaPrevista: dia(2026, 11, 20),
	}, noiteEmBrasilia)
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("embarque no dia 15 às 22h do dia 14 em Brasília: erro = %v", err)
	}
}

func TestDesembaracar(t *testing.T) {
	if err := novaImportacaoDeTeste(t).Desembaracar(domainimportacao.DesembaracoInput{
		ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 12), Declaracao: "26/1234567-8",
	}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desembaraçar aberta: erro = %v", err)
	}

	casos := []struct {
		nome  string
		input domainimportacao.DesembaracoInput
	}{
		{"chegada antes do embarque", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 9, 30), DesembaracadaEm: dia(2026, 10, 12), Declaracao: "DI"}},
		{"chegada no futuro", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 10, 20), DesembaracadaEm: dia(2026, 10, 20), Declaracao: "DI"}},
		{"desembaraço antes da chegada", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 9), Declaracao: "DI"}},
		{"sem desembaraço", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 10, 10), Declaracao: "DI"}},
		{"sem declaração", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 12), Declaracao: "  "}},
		{"declaração longa", domainimportacao.DesembaracoInput{ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 12), Declaracao: strings.Repeat("9", 31)}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := embarcada(t).Desembaracar(caso.input, agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v, esperado validação", err)
			}
		})
	}

	importacao := embarcada(t)
	if err := importacao.Desembaracar(domainimportacao.DesembaracoInput{
		ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 12), Declaracao: " 26/1234567-8 ",
	}, agora); err != nil {
		t.Fatalf("Desembaracar: %v", err)
	}
	if importacao.Status != domainimportacao.StatusDesembaracada || *importacao.Declaracao != "26/1234567-8" ||
		!importacao.ChegadaEm.Equal(dia(2026, 10, 10)) || !importacao.DesembaracadaEm.Equal(dia(2026, 10, 12)) {
		t.Errorf("importação desembaraçada = %+v", importacao)
	}
	if err := importacao.Alterar(dadosDeTeste(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar desembaraçada: erro = %v", err)
	}
}

func TestCancelar(t *testing.T) {
	importacao := embarcada(t)

	if err := importacao.Cancelar("  ", agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem motivo: erro = %v", err)
	}
	if err := importacao.Cancelar(strings.Repeat("m", 1001), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("motivo longo: erro = %v", err)
	}
	if err := importacao.Cancelar(" fornecedor desistiu ", agora); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if importacao.Status != domainimportacao.StatusCancelada || *importacao.CanceladaMotivo != "fornecedor desistiu" {
		t.Errorf("importação cancelada = %+v", importacao)
	}
	if err := importacao.Cancelar("de novo", agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar de novo: erro = %v", err)
	}
	if err := importacao.Alterar(dadosDeTeste(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar cancelada: erro = %v", err)
	}
}

func TestConferirVersao(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)

	if err := importacao.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão zero: erro = %v", err)
	}
	if err := importacao.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente: erro = %v", err)
	}
	if err := importacao.ConferirVersao(1); err != nil {
		t.Errorf("versão correta: erro = %v", err)
	}
}

func TestStatus(t *testing.T) {
	if status, err := domainimportacao.NovoStatus(" Embarcada "); err != nil || status != domainimportacao.StatusEmbarcada {
		t.Errorf("NovoStatus = %q (%v)", status, err)
	}
	if _, err := domainimportacao.NovoStatus("em_transito"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status inválido: erro = %v", err)
	}
	if !domainimportacao.StatusRecebida.Final() || !domainimportacao.StatusCancelada.Final() || domainimportacao.StatusDesembaracada.Final() {
		t.Error("status finais errados")
	}
	if !domainimportacao.StatusEmbarcada.Editavel() || domainimportacao.StatusDesembaracada.Editavel() {
		t.Error("status editáveis errados")
	}
}

func TestHistoricoLimitaODetalhe(t *testing.T) {
	historico := domainimportacao.NovoHistorico(uuid.New(), uuid.New(), domainimportacao.AcaoCancelada, nil,
		domainimportacao.StatusCancelada, strings.Repeat("ç", 1200), agora)

	if len([]rune(historico.Detalhe)) != 1000 || historico.ID == uuid.Nil {
		t.Errorf("histórico = %d runas, id %v", len([]rune(historico.Detalhe)), historico.ID)
	}
}
