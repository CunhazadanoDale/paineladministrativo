package importacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

func TestCriarNumeraPorEmpresaEUsaAMoedaDoFornecedor(t *testing.T) {
	c := novoCenario(t)

	primeira := c.criar(t, c.financeiro, c.mia, c.dados())
	segunda := c.criar(t, c.administrador, c.mia, c.dados())
	daRed := c.criar(t, c.financeiroRed, c.red, c.dados())

	if primeira.Numero != 1 || segunda.Numero != 2 || daRed.Numero != 1 {
		t.Errorf("números = %d, %d e %d; esperado 1, 2 e 1", primeira.Numero, segunda.Numero, daRed.Numero)
	}
	if primeira.Moeda != domainfornecedor.MoedaYuanChines || primeira.ValorTotal.Centesimos() != 3000000 {
		t.Errorf("moeda %q e total %d", primeira.Moeda, primeira.ValorTotal.Centesimos())
	}
	if primeira.Status != domainimportacao.StatusAberta || primeira.UsuarioID != c.financeiro {
		t.Errorf("importação criada com status %q e usuário %v", primeira.Status, primeira.UsuarioID)
	}

	dados := c.dados()
	dados.Moeda = "usd"
	if emDolar := c.criar(t, c.financeiro, c.mia, dados); emDolar.Moeda != domainfornecedor.MoedaDolar {
		t.Errorf("moeda informada = %q, esperado USD", emDolar.Moeda)
	}

	historico, err := c.importacoes.ListarHistorico(c.ctx, primeira.ID, c.comercial)
	if err != nil || len(historico) != 1 || historico[0].Acao != domainimportacao.AcaoCriada {
		t.Errorf("histórico = %+v (%v)", historico, err)
	}
}

func TestCriarValida(t *testing.T) {
	c := novoCenario(t)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		empresa   uuid.UUID
		alterar   func(*portsin.DadosImportacaoInput)
		sentinela error
	}{
		{"comercial sem perfil financeiro", c.comercial, c.mia, nil, domain.ErrPermissao},
		{"operador", c.operador, c.mia, nil, domain.ErrPermissao},
		{"empresa fora do acesso", c.financeiro, c.red, nil, domain.ErrPermissao},
		{"empresa inativa", c.financeiro, c.inativa, nil, domain.ErrValidacao},
		{"sem empresa", c.financeiro, uuid.Nil, nil, domain.ErrValidacao},
		{"sem fornecedor", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.FornecedorID = uuid.Nil }, domain.ErrValidacao},
		{"fornecedor inexistente", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.FornecedorID = uuid.New() }, domain.ErrValidacao},
		{"fornecedor inativo", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.FornecedorID = c.fornecedorInativo.ID }, domain.ErrValidacao},
		{"moeda inválida", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.Moeda = "JPY" }, domain.ErrValidacao},
		{"produto inativo", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.Itens[0].ProdutoID = c.produtoInativo.ID }, domain.ErrValidacao},
		{"produto inexistente", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.Itens[0].ProdutoID = uuid.New() }, domain.ErrValidacao},
		{"sem itens", c.financeiro, c.mia, func(d *portsin.DadosImportacaoInput) { d.Itens = nil }, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := c.dados()
			if caso.alterar != nil {
				caso.alterar(&dados)
			}
			_, err := c.importacoes.Criar(c.ctx, portsin.CriarImportacaoInput{UsuarioID: caso.usuario, EmpresaID: caso.empresa, Dados: dados})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro = %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	if len(c.m.importacoes) != 0 {
		t.Errorf("%d importações gravadas com erro", len(c.m.importacoes))
	}
}

func TestComercialLeSemValoresEFinanceiroComValores(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())

	visao, err := c.importacoes.Obter(c.ctx, importacao.ID, c.comercial)
	if err != nil || visao.ComValores || len(visao.Importacao.Itens) != 2 {
		t.Errorf("comercial: visão = %+v (%v)", visao, err)
	}
	visao, err = c.importacoes.Obter(c.ctx, importacao.ID, c.financeiro)
	if err != nil || !visao.ComValores {
		t.Errorf("financeiro: visão = %+v (%v)", visao, err)
	}
	lista, err := c.importacoes.Listar(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.comercial})
	if err != nil || lista.ComValores || len(lista.Importacoes) != 1 {
		t.Errorf("lista do comercial = %+v (%v)", lista, err)
	}

	if _, err := c.importacoes.Obter(c.ctx, importacao.ID, c.operador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("operador: erro = %v", err)
	}
	if _, err := c.importacoes.Obter(c.ctx, uuid.New(), c.financeiro); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("inexistente: erro = %v", err)
	}
}

func TestIsolamentoEntreEmpresas(t *testing.T) {
	c := novoCenario(t)
	daMia := c.criar(t, c.financeiro, c.mia, c.dados())
	daRed := c.criar(t, c.financeiroRed, c.red, c.dados())

	if _, err := c.importacoes.Obter(c.ctx, daRed.ID, c.financeiro); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("obter de outra empresa: erro = %v", err)
	}
	if _, err := c.importacoes.ListarHistorico(c.ctx, daRed.ID, c.comercial); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("histórico de outra empresa: erro = %v", err)
	}
	if _, err := c.importacoes.Cancelar(c.ctx, portsin.CancelarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: daRed.ID, Versao: 1, Motivo: "teste",
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("cancelar de outra empresa: erro = %v", err)
	}

	lista, err := c.importacoes.Listar(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiro})
	if err != nil || len(lista.Importacoes) != 1 || lista.Importacoes[0].ID != daMia.ID {
		t.Errorf("lista do financeiro da MIA = %+v (%v)", lista, err)
	}
	if _, err := c.importacoes.Listar(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiro, Empresa: &c.red}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("filtro de empresa fora do acesso: erro = %v", err)
	}
	todas, err := c.importacoes.Listar(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.administrador})
	if err != nil || len(todas.Importacoes) != 2 {
		t.Errorf("lista do administrador = %+v (%v)", todas, err)
	}
}

func TestListarFiltra(t *testing.T) {
	c := novoCenario(t)
	aberta := c.criar(t, c.financeiro, c.mia, c.dados())
	embarcada := c.embarcar(t, c.criar(t, c.financeiro, c.mia, c.dados()))
	europeia := c.dados()
	europeia.FornecedorID = c.fornecedorEuropeu.ID
	daEuropa := c.criar(t, c.financeiro, c.mia, europeia)

	casos := []struct {
		nome     string
		input    portsin.ListarImportacoesInput
		esperado []uuid.UUID
	}{
		{"status", portsin.ListarImportacoesInput{Status: "embarcada"}, []uuid.UUID{embarcada.ID}},
		{"fornecedor", portsin.ListarImportacoesInput{FornecedorID: &c.fornecedorEuropeu.ID}, []uuid.UUID{daEuropa.ID}},
		{"número", portsin.ListarImportacoesInput{Numero: &aberta.Numero}, []uuid.UUID{aberta.ID}},
		{"chegada até", portsin.ListarImportacoesInput{ChegadaAte: embarcada.ChegadaPrevista}, []uuid.UUID{embarcada.ID}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.input.UsuarioID = c.financeiro
			lista, err := c.importacoes.Listar(c.ctx, caso.input)
			if err != nil || len(lista.Importacoes) != len(caso.esperado) || lista.Importacoes[0].ID != caso.esperado[0] {
				t.Errorf("lista = %+v (%v), esperado %v", lista, err, caso.esperado)
			}
		})
	}

	if _, err := c.importacoes.Listar(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiro, Status: "em_transito"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status inválido: erro = %v", err)
	}
}

func TestAlterar(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	porcelanatoID := importacao.Itens[0].ID

	dados := c.dados()
	dados.Itens[0].Quantidade = 900
	dados.Itens = dados.Itens[:1]
	referencia := "PI-778"
	dados.Referencia = &referencia
	visao, err := c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 1, Dados: dados,
	})
	if err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	alterada := visao.Importacao
	if alterada.Versao != 2 || len(alterada.Itens) != 1 || alterada.Itens[0].ID != porcelanatoID || alterada.Itens[0].Quantidade != 900 ||
		alterada.Moeda != domainfornecedor.MoedaYuanChines || *alterada.Referencia != "PI-778" {
		t.Errorf("importação alterada = %+v", alterada)
	}
	if ultima := c.m.gravacoes[len(c.m.gravacoes)-1]; !ultima.SubstituirItens || ultima.FornecedorTrocado {
		t.Errorf("gravação da alteração = %+v", ultima)
	}

	if _, err := c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 1, Dados: dados,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão antiga: erro = %v", err)
	}
	if _, err := c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.comercial, ImportacaoID: importacao.ID, Versao: 2, Dados: dados,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial alterando: erro = %v", err)
	}

	dados.FornecedorID = c.fornecedorEuropeu.ID
	visao, err = c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 2, Dados: dados,
	})
	if err != nil || visao.Importacao.Moeda != domainfornecedor.MoedaEuro || !c.m.gravacoes[len(c.m.gravacoes)-1].FornecedorTrocado {
		t.Errorf("troca de fornecedor: %+v (%v)", visao, err)
	}
}

func TestAlterarMantemFornecedorQueFicouInativo(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	c.m.fornecedores[c.fornecedorChines.ID].Ativo = false
	c.m.produtos[c.porcelanato.ID].Ativo = false

	if _, err := c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 1, Dados: c.dados(),
	}); err != nil {
		t.Errorf("alterar com fornecedor e produto já gravados e agora inativos: %v", err)
	}
}

func TestFluxoAteODesembaraco(t *testing.T) {
	c := novoCenario(t)
	importacao := c.embarcar(t, c.criar(t, c.financeiro, c.mia, c.dados()))
	hoje := domainimportacao.Hoje(time.Now())

	if importacao.Status != domainimportacao.StatusEmbarcada || importacao.Versao != 2 {
		t.Errorf("embarcada = %+v", importacao)
	}

	visao, err := c.importacoes.Desembaracar(c.ctx, portsin.DesembaracarInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 2,
		ChegadaEm: hoje.AddDate(0, 0, -2), DesembaracadaEm: hoje, Declaracao: "26/0001234-5",
	})
	if err != nil {
		t.Fatalf("Desembaracar: %v", err)
	}
	if visao.Importacao.Status != domainimportacao.StatusDesembaracada || visao.Importacao.Versao != 3 {
		t.Errorf("desembaraçada = %+v", visao.Importacao)
	}

	historico, err := c.importacoes.ListarHistorico(c.ctx, importacao.ID, c.financeiro)
	if err != nil || len(historico) != 3 {
		t.Fatalf("histórico = %+v (%v)", historico, err)
	}
	if historico[1].Acao != domainimportacao.AcaoEmbarcada || historico[1].StatusAnterior == nil ||
		*historico[1].StatusAnterior != domainimportacao.StatusAberta || historico[2].Detalhe != "declaração 26/0001234-5" {
		t.Errorf("histórico = %+v", historico)
	}

	if _, err := c.importacoes.Embarcar(c.ctx, portsin.EmbarcarInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 3, EmbarcadaEm: hoje, ChegadaPrevista: hoje,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("embarcar desembaraçada: erro = %v", err)
	}
	if _, err := c.importacoes.Desembaracar(c.ctx, portsin.DesembaracarInput{
		UsuarioID: c.comercial, ImportacaoID: importacao.ID, Versao: 3,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial desembaraçando: erro = %v", err)
	}
}

func TestCancelar(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())

	visao, err := c.importacoes.Cancelar(c.ctx, portsin.CancelarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 1, Motivo: "pedido ao fornecedor desfeito",
	})
	if err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if visao.Importacao.Status != domainimportacao.StatusCancelada || *visao.Importacao.CanceladaMotivo != "pedido ao fornecedor desfeito" {
		t.Errorf("cancelada = %+v", visao.Importacao)
	}

	if _, err := c.importacoes.Cancelar(c.ctx, portsin.CancelarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 2, Motivo: "de novo",
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar cancelada: erro = %v", err)
	}
	if _, err := c.importacoes.Alterar(c.ctx, portsin.AlterarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: 2, Dados: c.dados(),
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar cancelada: erro = %v", err)
	}
}
