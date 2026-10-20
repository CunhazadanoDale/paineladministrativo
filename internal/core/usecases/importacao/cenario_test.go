package importacao_test

import (
	"context"
	"testing"
	"time"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	importacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/importacao"
	"github.com/google/uuid"
)

type cenario struct {
	ctx         context.Context
	m           *memoria
	importacoes portsin.ImportacaoUseCase
	financas    portsin.FinanceiroUseCase
	recebimento portsin.RecebimentoUseCase

	galpao    *domainestoque.Local
	galpaoRed *domainestoque.Local
	fechado   *domainestoque.Local

	compraMercadoria uuid.UUID
	receita          uuid.UUID

	mia     uuid.UUID
	red     uuid.UUID
	inativa uuid.UUID

	administrador uuid.UUID
	financeiro    uuid.UUID
	financeiroRed uuid.UUID
	comercial     uuid.UUID
	operador      uuid.UUID

	fornecedorChines  *domainfornecedor.Fornecedor
	fornecedorEuropeu *domainfornecedor.Fornecedor
	fornecedorInativo *domainfornecedor.Fornecedor
	porcelanato       *domainestoque.Produto
	revestimento      *domainestoque.Produto
	produtoInativo    *domainestoque.Produto
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	m := novaMemoria()
	c := &cenario{
		ctx: context.Background(),
		m:   m,
		importacoes: importacaousecases.NewImportacaoUsecase(
			repoImportacoes{m}, repoFornecedores{m}, repoProdutos{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		financas: importacaousecases.NewFinanceiroUsecase(
			repoImportacoes{m}, repoFornecedores{m}, repoCategorias{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		recebimento: importacaousecases.NewRecebimentoUsecase(
			repoImportacoes{m}, repoEstoque{m}, repoLocais{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
	}

	c.mia = c.novaEmpresa(true)
	c.red = c.novaEmpresa(true)
	c.inativa = c.novaEmpresa(false)

	acessoMia := domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa})
	c.administrador = c.novoUsuario(true, false, false, domainempresa.AcessoTotal())
	c.financeiro = c.novoUsuario(false, false, true, acessoMia)
	c.financeiroRed = c.novoUsuario(false, false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.comercial = c.novoUsuario(false, true, false, acessoMia)
	c.operador = c.novoUsuario(false, false, false, acessoMia)

	c.fornecedorChines = c.novoFornecedor(domainfornecedor.MoedaYuanChines, true)
	c.fornecedorEuropeu = c.novoFornecedor(domainfornecedor.MoedaEuro, true)
	c.fornecedorInativo = c.novoFornecedor(domainfornecedor.MoedaDolar, false)
	c.porcelanato = c.novoProduto("Porcelanato 120x120", true)
	c.revestimento = c.novoProduto("Revestimento 30x60", true)
	c.produtoInativo = c.novoProduto("Pastilha", false)
	c.compraMercadoria = c.novaCategoria(domainfinanceiro.NaturezaDespesa)
	c.receita = c.novaCategoria(domainfinanceiro.NaturezaReceita)
	c.galpao = c.novoLocal(c.mia, true)
	c.galpaoRed = c.novoLocal(c.red, true)
	c.fechado = c.novoLocal(c.mia, false)

	return c
}

func (c *cenario) novaEmpresa(ativa bool) uuid.UUID {
	empresa := &domainempresa.Empresa{ID: uuid.New(), NomeFantasia: "Empresa", Ativa: ativa}
	c.m.empresas[empresa.ID] = empresa

	return empresa.ID
}

func (c *cenario) novoUsuario(administrador, comercial, financeiro bool, acesso domainempresa.Acesso) uuid.UUID {
	cargo := &domainusuarios.Cargo{
		ID: uuid.New(), Nome: "Cargo", Ativo: true,
		Administrador: administrador, Comercial: comercial, Financeiro: financeiro,
	}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = cargo
	c.m.usuarios[usuario.ID] = usuario
	c.m.acessos[usuario.ID] = acesso

	return usuario.ID
}

func (c *cenario) novoFornecedor(moeda domainfornecedor.Moeda, ativo bool) *domainfornecedor.Fornecedor {
	fornecedor := &domainfornecedor.Fornecedor{ID: uuid.New(), Nome: "Foshan Ceramics", MoedaPadrao: moeda, Ativo: ativo}
	c.m.fornecedores[fornecedor.ID] = fornecedor

	return fornecedor
}

func (c *cenario) novoProduto(nome string, ativo bool) *domainestoque.Produto {
	produto := &domainestoque.Produto{ID: uuid.New(), Nome: nome, Ativo: ativo, VersaoEstoque: 1}
	c.m.produtos[produto.ID] = produto
	c.m.estoques[produto.ID] = &domainestoque.EstoqueProduto{Produto: copiar(produto), Saldos: map[uuid.UUID]domainestoque.SaldoLocal{}}

	return produto
}

func (c *cenario) dados() portsin.DadosImportacaoInput {
	return portsin.DadosImportacaoInput{
		FornecedorID: c.fornecedorChines.ID,
		Itens: []portsin.ItemInput{
			{ProdutoID: c.porcelanato.ID, Quantidade: 800, ValorMoedaCentesimos: 2400000},
			{ProdutoID: c.revestimento.ID, Quantidade: 300, ValorMoedaCentesimos: 600000},
		},
	}
}

func (c *cenario) criar(t *testing.T, usuarioID, empresaID uuid.UUID, dados portsin.DadosImportacaoInput) *domainimportacao.Importacao {
	t.Helper()

	visao, err := c.importacoes.Criar(c.ctx, portsin.CriarImportacaoInput{UsuarioID: usuarioID, EmpresaID: empresaID, Dados: dados})
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	return visao.Importacao
}

func (c *cenario) embarcar(t *testing.T, importacao *domainimportacao.Importacao) *domainimportacao.Importacao {
	t.Helper()

	hoje := domainimportacao.Hoje(time.Now())
	visao, err := c.importacoes.Embarcar(c.ctx, portsin.EmbarcarInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		EmbarcadaEm: hoje.AddDate(0, 0, -5), ChegadaPrevista: hoje.AddDate(0, 0, 40),
	})
	if err != nil {
		t.Fatalf("Embarcar falhou: %v", err)
	}

	return visao.Importacao
}

func (c *cenario) novaCategoria(natureza domainfinanceiro.Natureza) uuid.UUID {
	pai := uuid.New()
	categoria := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), CategoriaPaiID: &pai, Natureza: natureza, Ativa: true}
	c.m.categorias[categoria.ID] = categoria

	return categoria.ID
}

func (c *cenario) novoLocal(empresaID uuid.UUID, ativo bool) *domainestoque.Local {
	local := &domainestoque.Local{ID: uuid.New(), EmpresaID: empresaID, Nome: "Galpão", Ativo: ativo}
	c.m.locais[local.ID] = local

	return local
}

func (c *cenario) novoItemDePedido(empresaID uuid.UUID, produto *domainestoque.Produto, quantidade int, prazo *time.Time) *domainimportacao.PedidoItemAlocavel {
	item := &domainimportacao.PedidoItemAlocavel{
		PedidoItemID: uuid.New(), PedidoID: uuid.New(), PedidoNumero: len(c.m.pedidoItens) + 1, EmpresaID: empresaID,
		Modelo: domainpedido.ModeloImportacao, StatusPedido: domainpedido.StatusConfirmado, PrazoLimite: prazo,
		ProdutoID: produto.ID, Quantidade: quantidade,
	}
	c.m.pedidoItens[item.PedidoItemID] = item

	return item
}

func (c *cenario) desembaracar(t *testing.T, importacao *domainimportacao.Importacao) *domainimportacao.Importacao {
	t.Helper()

	embarcada := c.embarcar(t, importacao)
	hoje := domainimportacao.Hoje(time.Now())
	visao, err := c.importacoes.Desembaracar(c.ctx, portsin.DesembaracarInput{
		UsuarioID: c.financeiro, ImportacaoID: embarcada.ID, Versao: embarcada.Versao,
		ChegadaEm: hoje.AddDate(0, 0, -2), DesembaracadaEm: hoje.AddDate(0, 0, -1), Declaracao: "26/0001234-5",
	})
	if err != nil {
		t.Fatalf("Desembaracar falhou: %v", err)
	}

	return visao.Importacao
}
