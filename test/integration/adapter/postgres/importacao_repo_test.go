package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var (
	_ portsimportacao.FornecedorRepository = (*postgres.FornecedorRepository)(nil)
	_ portsimportacao.ProdutoRepository    = (*postgres.ProdutoRepository)(nil)
	_ portsimportacao.EmpresaRepository    = (*postgres.EmpresaRepository)(nil)
)

type cenarioBancoImportacoes struct {
	*cenarioBancoLocais
	importacoes  *postgres.ImportacaoRepository
	fornecedorID uuid.UUID
	segundo      *domainestoque.Produto
}

func novoCenarioImportacoes(t *testing.T) *cenarioBancoImportacoes {
	t.Helper()

	base := novoCenarioLocais(t)
	raiz := base.novaCategoriaRaiz(t, "Acabamento")
	sub := base.novaSubcategoria(t, "Porcelanato", raiz)

	identificacao := "91440600MA4UX"
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisDe("CN"), domaincliente.Documento{}, domainfornecedor.DadosFornecedor{
		Nome: "Foshan Ceramics", MoedaPadrao: domainfornecedor.MoedaYuanChines, IdentificacaoEstrangeira: &identificacao,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}
	if _, err := postgres.NewFornecedorRepository(base.banco).Criar(base.ctx, fornecedor); err != nil {
		t.Fatalf("criação do fornecedor falhou: %v", err)
	}

	return &cenarioBancoImportacoes{
		cenarioBancoLocais: base,
		importacoes:        postgres.NewImportacaoRepository(base.banco),
		fornecedorID:       fornecedor.ID,
		segundo:            base.novoProduto(t, sub, "Porcelanato 60x60"),
	}
}

func (c *cenarioBancoImportacoes) dados() domainimportacao.DadosImportacao {
	return domainimportacao.DadosImportacao{
		FornecedorID: c.fornecedorID,
		Moeda:        domainfornecedor.MoedaYuanChines,
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: c.produto.ID, Quantidade: 800, ValorMoedaCentesimos: 2400000},
			{ProdutoID: c.segundo.ID, Quantidade: 300, ValorMoedaCentesimos: 600000},
		},
	}
}

func (c *cenarioBancoImportacoes) nova(t *testing.T, empresaID uuid.UUID) *domainimportacao.Importacao {
	t.Helper()

	importacao, err := domainimportacao.NovaImportacao(empresaID, c.usuarioID, c.dados(), time.Now().UTC())
	if err != nil {
		t.Fatalf("importação de teste inválida: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao:        importacao,
		Novo:              true,
		FornecedorTrocado: true,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, c.usuarioID, domainimportacao.AcaoCriada, nil, importacao.Status, "", importacao.CriadoEm),
		},
	}); err != nil {
		t.Fatalf("gravar a importação falhou: %v", err)
	}

	return importacao
}

func (c *cenarioBancoImportacoes) obter(t *testing.T, id uuid.UUID) *domainimportacao.Importacao {
	t.Helper()

	importacao, err := c.importacoes.Obter(c.ctx, id)
	if err != nil || importacao == nil {
		t.Fatalf("obter a importação = %+v, %v", importacao, err)
	}

	return importacao
}

func TestImportacaoNumeraPorEmpresaEObtemComItens(t *testing.T) {
	c := novoCenarioImportacoes(t)

	primeira := c.nova(t, c.mia)
	segunda := c.nova(t, c.mia)
	daRed := c.nova(t, c.red)
	if primeira.Numero != 1 || segunda.Numero != 2 || daRed.Numero != 1 {
		t.Errorf("números = %d, %d e %d", primeira.Numero, segunda.Numero, daRed.Numero)
	}

	lida := c.obter(t, primeira.ID)
	if lida.Moeda != domainfornecedor.MoedaYuanChines || lida.ValorTotal.Centesimos() != 3000000 || lida.Status != domainimportacao.StatusAberta ||
		lida.Versao != 1 || len(lida.Itens) != 2 || lida.Itens[0].ProdutoID != c.produto.ID || lida.Itens[0].Valor.Centesimos() != 2400000 {
		t.Errorf("importação lida = %+v", lida)
	}

	if inexistente, err := c.importacoes.Obter(c.ctx, uuid.New()); err != nil || inexistente != nil {
		t.Errorf("inexistente = %+v, %v", inexistente, err)
	}
	historico, err := c.importacoes.ListarHistorico(c.ctx, primeira.ID)
	if err != nil || len(historico) != 1 || historico[0].Acao != domainimportacao.AcaoCriada || historico[0].StatusAnterior != nil {
		t.Errorf("histórico = %+v, %v", historico, err)
	}
}

func TestImportacaoRecusaEmpresaEFornecedorInativos(t *testing.T) {
	c := novoCenarioImportacoes(t)

	if _, err := c.banco.Exec(`UPDATE fornecedor SET ativo = false WHERE id = $1`, c.fornecedorID); err != nil {
		t.Fatalf("desativar fornecedor: %v", err)
	}
	importacao, err := domainimportacao.NovaImportacao(c.mia, c.usuarioID, c.dados(), time.Now().UTC())
	if err != nil {
		t.Fatalf("importação de teste inválida: %v", err)
	}
	gravacao := &portsimportacao.Gravacao{Importacao: importacao, Novo: true, FornecedorTrocado: true}
	if err := c.importacoes.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("fornecedor inativo: erro = %v", err)
	}

	if _, err := c.banco.Exec(`UPDATE fornecedor SET ativo = true WHERE id = $1`, c.fornecedorID); err != nil {
		t.Fatalf("ativar fornecedor: %v", err)
	}
	if _, err := c.banco.Exec(`UPDATE empresa SET ativa = false WHERE id = $1`, c.mia); err != nil {
		t.Fatalf("desativar empresa: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("empresa inativa: erro = %v", err)
	}

	var total int
	if err := c.banco.Get(&total, `SELECT count(*) FROM importacao`); err != nil || total != 0 {
		t.Errorf("importações gravadas = %d, %v", total, err)
	}
}

func TestImportacaoAlteraItensComGuardaDeVersao(t *testing.T) {
	c := novoCenarioImportacoes(t)
	importacao := c.nova(t, c.mia)
	porcelanato := importacao.Itens[0]

	dados := c.dados()
	dados.Itens = []domainimportacao.ItemInput{{ProdutoID: c.produto.ID, Quantidade: 900, ValorMoedaCentesimos: 2700000}}
	if err := importacao.Alterar(dados, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	gravacao := &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: 1, StatusAnterior: domainimportacao.StatusAberta, SubstituirItens: true,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, c.usuarioID, domainimportacao.AcaoAlterada, nil, importacao.Status, "", time.Now().UTC()),
		},
	}
	if err := c.importacoes.Gravar(c.ctx, gravacao); err != nil {
		t.Fatalf("gravar alteração: %v", err)
	}
	if importacao.Versao != 2 {
		t.Errorf("versão depois de gravar = %d", importacao.Versao)
	}

	lida := c.obter(t, importacao.ID)
	if lida.Versao != 2 || len(lida.Itens) != 1 || lida.Itens[0].ID != porcelanato.ID || lida.Itens[0].Quantidade != 900 ||
		lida.ValorTotal.Centesimos() != 2700000 {
		t.Errorf("importação depois da alteração = %+v", lida)
	}

	if err := c.importacoes.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão antiga: erro = %v", err)
	}
	outra := c.obter(t, importacao.ID)
	outra.ID = uuid.New()
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: outra, VersaoLida: 2, StatusAnterior: domainimportacao.StatusAberta,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("inexistente: erro = %v", err)
	}
}

func TestImportacaoGuardaEmbarqueDesembaracoECancelamento(t *testing.T) {
	c := novoCenarioImportacoes(t)
	importacao := c.nova(t, c.mia)
	agora := time.Now().UTC()
	hoje := domainimportacao.Hoje(agora)
	documento := "BL-778"

	if err := importacao.Embarcar(domainimportacao.EmbarqueInput{
		EmbarcadaEm: hoje.AddDate(0, 0, -20), ChegadaPrevista: hoje.AddDate(0, 0, 10), DocumentoEmbarque: &documento,
	}, agora); err != nil {
		t.Fatalf("Embarcar: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: 1, StatusAnterior: domainimportacao.StatusAberta,
	}); err != nil {
		t.Fatalf("gravar embarque: %v", err)
	}

	if err := importacao.Desembaracar(domainimportacao.DesembaracoInput{
		ChegadaEm: hoje.AddDate(0, 0, -2), DesembaracadaEm: hoje, Declaracao: "26/0001234-5",
	}, agora); err != nil {
		t.Fatalf("Desembaracar: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: 2, StatusAnterior: domainimportacao.StatusEmbarcada,
	}); err != nil {
		t.Fatalf("gravar desembaraço: %v", err)
	}

	lida := c.obter(t, importacao.ID)
	if lida.Status != domainimportacao.StatusDesembaracada || !lida.EmbarcadaEm.Equal(hoje.AddDate(0, 0, -20)) ||
		!lida.ChegadaPrevista.Equal(hoje.AddDate(0, 0, 10)) || !lida.DesembaracadaEm.Equal(hoje) ||
		*lida.DocumentoEmbarque != documento || *lida.Declaracao != "26/0001234-5" {
		t.Errorf("importação desembaraçada = %+v", lida)
	}

	if err := lida.Cancelar("container avariado", agora); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: lida, VersaoLida: 3, StatusAnterior: domainimportacao.StatusEmbarcada,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("status anterior divergente: erro = %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: lida, VersaoLida: 3, StatusAnterior: domainimportacao.StatusDesembaracada,
	}); err != nil {
		t.Fatalf("gravar cancelamento: %v", err)
	}
	if cancelada := c.obter(t, importacao.ID); cancelada.Status != domainimportacao.StatusCancelada || *cancelada.CanceladaMotivo != "container avariado" {
		t.Errorf("importação cancelada = %+v", cancelada)
	}
}

func TestListarImportacoesFiltra(t *testing.T) {
	c := novoCenarioImportacoes(t)
	aberta := c.nova(t, c.mia)
	embarcada := c.nova(t, c.mia)
	daRed := c.nova(t, c.red)

	agora := time.Now().UTC()
	hoje := domainimportacao.Hoje(agora)
	if err := embarcada.Embarcar(domainimportacao.EmbarqueInput{EmbarcadaEm: hoje, ChegadaPrevista: hoje.AddDate(0, 0, 30)}, agora); err != nil {
		t.Fatalf("Embarcar: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: embarcada, VersaoLida: 1, StatusAnterior: domainimportacao.StatusAberta,
	}); err != nil {
		t.Fatalf("gravar embarque: %v", err)
	}

	ids := func(filtro portsimportacao.ImportacaoFiltro) []uuid.UUID {
		t.Helper()
		importacoes, err := c.importacoes.Listar(c.ctx, filtro)
		if err != nil {
			t.Fatalf("Listar: %v", err)
		}
		resultado := make([]uuid.UUID, 0, len(importacoes))
		for _, importacao := range importacoes {
			resultado = append(resultado, importacao.ID)
		}
		return resultado
	}
	iguais := func(nome string, obtidos []uuid.UUID, esperados ...uuid.UUID) {
		t.Helper()
		if len(obtidos) != len(esperados) {
			t.Errorf("%s: %v, esperado %v", nome, obtidos, esperados)
			return
		}
		for i := range esperados {
			if obtidos[i] != esperados[i] {
				t.Errorf("%s: %v, esperado %v", nome, obtidos, esperados)
				return
			}
		}
	}

	iguais("todas", ids(portsimportacao.ImportacaoFiltro{}), daRed.ID, embarcada.ID, aberta.ID)
	iguais("empresa", ids(portsimportacao.ImportacaoFiltro{Empresa: &c.mia}), embarcada.ID, aberta.ID)
	iguais("empresas do acesso", ids(portsimportacao.ImportacaoFiltro{EmpresaIDs: []uuid.UUID{c.red}}), daRed.ID)
	iguais("status", ids(portsimportacao.ImportacaoFiltro{Status: domainimportacao.StatusEmbarcada}), embarcada.ID)
	iguais("fornecedor", ids(portsimportacao.ImportacaoFiltro{FornecedorID: &c.fornecedorID, Empresa: &c.red}), daRed.ID)
	numero := 1
	iguais("número", ids(portsimportacao.ImportacaoFiltro{Numero: &numero, Empresa: &c.mia}), aberta.ID)
	limite := hoje.AddDate(0, 0, 30)
	iguais("chegada até", ids(portsimportacao.ImportacaoFiltro{ChegadaAte: &limite}), embarcada.ID)
	antes := hoje.AddDate(0, 0, 29)
	iguais("chegada antes", ids(portsimportacao.ImportacaoFiltro{ChegadaAte: &antes}))
	iguais("paginação", ids(portsimportacao.ImportacaoFiltro{PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 2}}), aberta.ID)
}
