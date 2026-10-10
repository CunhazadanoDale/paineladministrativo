package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoEstoque struct {
	banco      *sqlx.DB
	categorias *postgres.CategoriaRepository
	produtos   *postgres.ProdutoRepository
	movimentos *postgres.MovimentoRepository
	imagens    *postgres.ImagemRepository
	usuarioID  uuid.UUID
}

func cenarioEstoque(t *testing.T) *cenarioBancoEstoque {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	cargos := postgres.NewCargoRepository(banco)
	usuarios := postgres.NewUsuarioRepository(banco)

	cargoID, err := cargos.Criar(context.Background(), novoCargo("Almoxarife"))
	if err != nil {
		t.Fatalf("não criei o cargo do cenário: %v", err)
	}
	usuarioID, err := usuarios.Criar(context.Background(), novoUsuario(cargoID, "Ana Souza", "ana.souza@estoque.exemplo.com"))
	if err != nil {
		t.Fatalf("não criei o usuário do cenário: %v", err)
	}

	return &cenarioBancoEstoque{
		banco:      banco,
		categorias: postgres.NewCategoriaRepository(banco),
		produtos:   postgres.NewProdutoRepository(banco),
		movimentos: postgres.NewMovimentoRepository(banco),
		imagens:    postgres.NewImagemRepository(banco),
		usuarioID:  usuarioID,
	}
}

func (c *cenarioBancoEstoque) novaCategoriaRaiz(t *testing.T, nome string) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug(nome)
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	categoria, err := domainestoque.NovaCategoria(nome, slug, nil, 0, "")
	if err != nil {
		t.Fatalf("criação da categoria falhou: %v", err)
	}

	if _, err := c.categorias.Criar(context.Background(), categoria); err != nil {
		t.Fatalf("persistência da categoria falhou: %v", err)
	}

	return categoria
}

func (c *cenarioBancoEstoque) novaSubcategoria(t *testing.T, nome string, raiz *domainestoque.Categoria) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug(nome)
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	subcategoria, err := domainestoque.NovaCategoria(nome, slug, raiz, 1, "")
	if err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	if _, err := c.categorias.Criar(context.Background(), subcategoria); err != nil {
		t.Fatalf("persistência da subcategoria falhou: %v", err)
	}

	return subcategoria
}

func (c *cenarioBancoEstoque) novoProduto(t *testing.T, categoria *domainestoque.Categoria, nome string) *domainestoque.Produto {
	t.Helper()

	slug, err := domainestoque.NovoSlug(nome)
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	produto, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     categoria,
		Nome:          nome,
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	})
	if err != nil {
		t.Fatalf("criação do produto falhou: %v", err)
	}

	if _, err := c.produtos.Criar(context.Background(), produto); err != nil {
		t.Fatalf("persistência do produto falhou: %v", err)
	}

	return produto
}

func (c *cenarioBancoEstoque) movimentar(t *testing.T, produto *domainestoque.Produto, tipo domainestoque.TipoMovimento, quantidade int) {
	t.Helper()
	ctx := context.Background()

	movimento, err := produto.Movimentar(tipo, domainestoque.QuantidadeDe(quantidade), c.usuarioID, nil, "", time.Now().UTC())
	if err != nil {
		t.Fatalf("movimentação de domínio falhou: %v", err)
	}
	if err := c.produtos.Movimentar(ctx, produto, movimento); err != nil {
		t.Fatalf("persistência da movimentação falhou: %v", err)
	}
}

func TestCategoriaPersisteRaizESubcategoria(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	salva, err := c.categorias.Obter(ctx, raiz.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva == nil || salva.CategoriaPaiID != nil || salva.Slug.Valor() != "materiais" {
		t.Errorf("categoria = %+v, esperado raiz com slug materiais", salva)
	}

	porSlug, err := c.categorias.ObterPorSlug(ctx, "alvenaria")
	if err != nil {
		t.Fatalf("busca por slug falhou: %v", err)
	}
	if porSlug == nil || porSlug.ID != sub.ID {
		t.Errorf("categoria por slug = %+v, esperado %s", porSlug, sub.ID)
	}
	if porSlug.CategoriaPaiID == nil || *porSlug.CategoriaPaiID != raiz.ID {
		t.Errorf("pai = %v, esperado %s", porSlug.CategoriaPaiID, raiz.ID)
	}

	subcategorias, err := c.categorias.PossuiSubcategorias(ctx, raiz.ID)
	if err != nil {
		t.Fatalf("verificação de subcategorias falhou: %v", err)
	}
	if !subcategorias {
		t.Error("raiz deveria ter subcategorias")
	}

	temProdutos, err := c.categorias.PossuiProdutos(ctx, raiz.ID)
	if err != nil {
		t.Fatalf("verificação de produtos falhou: %v", err)
	}
	if temProdutos {
		t.Error("raiz sem produtos não deveria reportar produtos")
	}
}

func TestCategoriaAtualizaNomeEAtivo(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	raiz.Nome = "Materiais Gerais"
	raiz.Ativo = false
	raiz.AtualizadoEm = time.Now().UTC()

	atualizada, err := c.categorias.Atualizar(ctx, raiz)
	if err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}
	if !atualizada {
		t.Fatal("atualização não manteve nenhuma linha")
	}

	salva, err := c.categorias.Obter(ctx, raiz.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva.Nome != "Materiais Gerais" || salva.Ativo {
		t.Errorf("categoria = %+v, esperado renomeada e inativa", salva)
	}
}

func TestProdutoPersisteComPrecoESaldoInicialZerado(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	preco, err := domainestoque.NovoPreco(2590)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}
	minimo, err := domainestoque.NovoEstoqueMinimo(5)
	if err != nil {
		t.Fatalf("mínimo de teste inválido: %v", err)
	}
	slug, err := domainestoque.NovoSlug("Cimento CP II 50kg")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	produto, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     sub,
		Nome:          "  Cimento CP II 50kg  ",
		Slug:          slug,
		Descricao:     "Cimento de uso geral",
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
		Preco:         &preco,
		EstoqueMinimo: &minimo,
		Destaque:      true,
	})
	if err != nil {
		t.Fatalf("criação do produto falhou: %v", err)
	}

	if _, err := c.produtos.Criar(ctx, produto); err != nil {
		t.Fatalf("persistência falhou: %v", err)
	}

	salvo, err := c.produtos.Obter(ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("produto criado não encontrado")
	}
	if salvo.Nome != "Cimento CP II 50kg" {
		t.Errorf("nome = %q, esperado sem espaços nas bordas", salvo.Nome)
	}
	if salvo.CategoriaID != sub.ID {
		t.Errorf("categoria = %s, esperado %s", salvo.CategoriaID, sub.ID)
	}
	if salvo.Preco == nil || salvo.Preco.Centavos() != 2590 {
		t.Errorf("preço = %v, esperado 2590 centavos", salvo.Preco)
	}
	if salvo.EstoqueMinimo == nil || salvo.EstoqueMinimo.Quantidade() != 5 {
		t.Errorf("mínimo = %v, esperado 5", salvo.EstoqueMinimo)
	}
	if !salvo.Saldo.Vazio() {
		t.Errorf("saldo = %d, esperado 0", salvo.Saldo.Quantidade())
	}
	if !salvo.Destaque || !salvo.Ativo {
		t.Error("produto deveria nascer em destaque e ativo")
	}
}

func TestProdutoAceitaSobConsultaPersistindoPrecoNulo(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	produto := c.novoProduto(t, c.novaSubcategoria(t, "Alvenaria", raiz), "Areia Fina")

	salvo, err := c.produtos.ObterPorSlug(ctx, "areia-fina")
	if err != nil {
		t.Fatalf("busca por slug falhou: %v", err)
	}
	if salvo == nil || salvo.ID != produto.ID {
		t.Fatalf("produto = %+v, esperado encontrado por slug", salvo)
	}
	if salvo.Preco != nil || salvo.PrecoPromocional != nil {
		t.Error("produto sob consulta não deveria ter preço")
	}
}

func TestProdutoMovimentacaoAtualizaSaldoEGuardaConcorrencia(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	produto := c.novoProduto(t, c.novaSubcategoria(t, "Alvenaria", raiz), "Cimento CP II 50kg")

	c.movimentar(t, produto, domainestoque.TipoMovimentoEntrada, 20)

	salvo, err := c.produtos.Obter(ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Saldo.Quantidade() != 20 {
		t.Fatalf("saldo = %d, esperado 20", salvo.Saldo.Quantidade())
	}

	c.movimentar(t, produto, domainestoque.TipoMovimentoSaida, 6)

	salvo, err = c.produtos.Obter(ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Saldo.Quantidade() != 14 {
		t.Errorf("saldo = %d, esperado 14", salvo.Saldo.Quantidade())
	}

	movimentos, err := c.movimentos.Listar(ctx, portsout.MovimentoFiltro{
		ProdutoID:       &produto.ID,
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10},
	})
	if err != nil {
		t.Fatalf("listagem de movimentos falhou: %v", err)
	}
	if len(movimentos) != 2 {
		t.Fatalf("%d movimentos, esperado 2", len(movimentos))
	}
	if movimentos[0].Tipo != domainestoque.TipoMovimentoSaida || movimentos[0].SaldoApos.Quantidade() != 14 {
		t.Errorf("movimento mais recente = %+v, esperado saída com saldo 14", movimentos[0])
	}

	produto.Saldo = domainestoque.SaldoDe(100)
	movimentoConcorrente, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(5),
		c.usuarioID,
		nil,
		"",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("movimentação concorrente falhou no domínio: %v", err)
	}

	if err := c.produtos.Movimentar(ctx, produto, movimentoConcorrente); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("movimentação com saldo desatualizado = %v, esperado erro de conflito", err)
	}

	salvo, err = c.produtos.Obter(ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Saldo.Quantidade() != 14 {
		t.Errorf("saldo = %d, esperado permanecer 14 após conflito", salvo.Saldo.Quantidade())
	}
}

func TestProdutoListarComFiltros(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	cimento := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, cimento, domainestoque.TipoMovimentoEntrada, 3)

	minimo, err := domainestoque.NovoEstoqueMinimo(10)
	if err != nil {
		t.Fatalf("mínimo de teste inválido: %v", err)
	}
	cimento.EstoqueMinimo = &minimo
	cimento.AtualizadoEm = time.Now().UTC()
	if _, err := c.produtos.Atualizar(ctx, cimento); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	argamassa := c.novoProduto(t, sub, "Argamassa AC III")
	argamassa.Destaque = true
	if _, err := c.produtos.Atualizar(ctx, argamassa); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	filtro := portsout.ProdutoFiltro{
		CategoriaID:     &sub.ID,
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10},
	}
	itens, err := c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Fatalf("%d itens, esperado 2 na subcategoria", len(itens))
	}

	filtro.Busca = "cimento"
	itens, err = c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != cimento.ID {
		t.Errorf("busca = %+v, esperado só o cimento", itens)
	}

	filtro.Busca = ""
	filtro.EstoqueBaixo = true
	itens, err = c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("filtro de estoque baixo falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != cimento.ID {
		t.Errorf("estoque baixo = %+v, esperado só o cimento", itens)
	}

	filtro.EstoqueBaixo = false
	filtro.Destaque = &[]bool{true}[0]
	itens, err = c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("filtro de destaque falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != argamassa.ID {
		t.Errorf("destaque = %+v, esperado só a argamassa", itens)
	}
}

func TestResumoConsolidaTotaisEUltimosMovimentos(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	cimento := c.novoProduto(t, sub, "Cimento CP II 50kg")
	preco, err := domainestoque.NovoPreco(2500)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}
	minimo, err := domainestoque.NovoEstoqueMinimo(10)
	if err != nil {
		t.Fatalf("mínimo de teste inválido: %v", err)
	}
	cimento.Preco = &preco
	cimento.EstoqueMinimo = &minimo
	cimento.AtualizadoEm = time.Now().UTC()
	if _, err := c.produtos.Atualizar(ctx, cimento); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}
	c.movimentar(t, cimento, domainestoque.TipoMovimentoEntrada, 8)

	areia := c.novoProduto(t, sub, "Areia Fina")
	c.movimentar(t, areia, domainestoque.TipoMovimentoEntrada, 4)

	tijolo := c.novoProduto(t, sub, "Tijolo 8 furos")
	tijolo.Ativo = false
	tijolo.AtualizadoEm = time.Now().UTC()
	if _, err := c.produtos.Atualizar(ctx, tijolo); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	resumo, err := c.produtos.Resumo(ctx)
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}
	if resumo.TotalProdutos != 3 {
		t.Errorf("total de produtos = %d, esperado 3", resumo.TotalProdutos)
	}
	if resumo.ProdutosAtivos != 2 {
		t.Errorf("produtos ativos = %d, esperado 2", resumo.ProdutosAtivos)
	}
	if resumo.ValorEstoqueCentavos != 20000 {
		t.Errorf("valor em estoque = %d, esperado 20000 centavos", resumo.ValorEstoqueCentavos)
	}
	if resumo.ProdutosEstoqueBaixo != 1 {
		t.Errorf("produtos com estoque baixo = %d, esperado 1", resumo.ProdutosEstoqueBaixo)
	}

	movimentos, err := c.movimentos.ListarResumo(ctx, 1)
	if err != nil {
		t.Fatalf("resumo de movimentos falhou: %v", err)
	}
	if len(movimentos) != 1 {
		t.Fatalf("%d movimentos, esperado 1", len(movimentos))
	}
	if movimentos[0].ProdutoID != cimento.ID && movimentos[0].ProdutoID != areia.ID {
		t.Errorf("produto = %s, esperado um dos dois movimentados", movimentos[0].ProdutoID)
	}
	if movimentos[0].ProdutoNome == "" {
		t.Error("resumo do movimento deveria trazer o nome do produto")
	}
	if movimentos[0].Tipo != domainestoque.TipoMovimentoEntrada {
		t.Errorf("tipo = %s, esperado entrada", movimentos[0].Tipo)
	}
	if movimentos[0].Quantidade.Valor() != 4 && movimentos[0].Quantidade.Valor() != 8 {
		t.Errorf("quantidade = %d, esperada 4 ou 8", movimentos[0].Quantidade.Valor())
	}
	if movimentos[0].ID == uuid.Nil || movimentos[0].CriadoEm.IsZero() {
		t.Errorf("movimento = %+v, esperado identificador e data de criação", movimentos[0])
	}
}

func TestProdutoAtualizarPreservaSaldo(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	produto := c.novoProduto(t, c.novaSubcategoria(t, "Alvenaria", raiz), "Cimento CP II 50kg")
	c.movimentar(t, produto, domainestoque.TipoMovimentoEntrada, 8)

	preco, err := domainestoque.NovoPreco(3200)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}

	produto.Nome = "Cimento CP II 50kg (saco)"
	produto.Preco = &preco
	produto.AtualizadoEm = time.Now().UTC()

	atualizado, err := c.produtos.Atualizar(ctx, produto)
	if err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}
	if !atualizado {
		t.Fatal("atualização não manteve nenhuma linha")
	}

	salvo, err := c.produtos.Obter(ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Nome != "Cimento CP II 50kg (saco)" {
		t.Errorf("nome = %q", salvo.Nome)
	}
	if salvo.Preco == nil || salvo.Preco.Centavos() != 3200 {
		t.Errorf("preço = %v, esperado 3200", salvo.Preco)
	}
	if salvo.Saldo.Quantidade() != 8 {
		t.Errorf("saldo = %d, esperado 8 (preservado)", salvo.Saldo.Quantidade())
	}
}

func (c *cenarioBancoEstoque) novoArquivo(t *testing.T) *domainsolicitacao.Arquivo {
	t.Helper()

	arquivo := &domainsolicitacao.Arquivo{
		ID:             uuid.New(),
		ProprietarioID: c.usuarioID,
		Nome:           "foto-do-produto",
		Chave:          "estoque/" + uuid.NewString(),
		ContentType:    "image/png",
		Tamanho:        1024,
		CriadoEm:       time.Now().UTC(),
	}

	if _, err := postgres.NewArquivoRepository(c.banco).Criar(context.Background(), arquivo); err != nil {
		t.Fatalf("persistência do arquivo falhou: %v", err)
	}

	return arquivo
}

func (c *cenarioBancoEstoque) novaImagem(t *testing.T, produto *domainestoque.Produto, arquivo *domainsolicitacao.Arquivo, ordem int, alt string) *domainestoque.Imagem {
	t.Helper()

	imagem, err := domainestoque.NovaImagem(domainestoque.ImagemInput{
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
		Ordem:     ordem,
		Alt:       alt,
	})
	if err != nil {
		t.Fatalf("criação da imagem falhou: %v", err)
	}

	if _, err := c.imagens.Criar(context.Background(), imagem); err != nil {
		t.Fatalf("persistência da imagem falhou: %v", err)
	}

	return imagem
}

func TestImagemRepoAnexaListaOrdenadaERemove(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	outro := c.novoProduto(t, sub, "Areia Fina")

	arquivoA := c.novoArquivo(t)
	arquivoB := c.novoArquivo(t)

	ultima := c.novaImagem(t, produto, arquivoA, 5, "Capa")
	primeira := c.novaImagem(t, produto, arquivoB, 1, "Verso")

	listagem, err := c.imagens.ListarPorProduto(ctx, produto.ID)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(listagem) != 2 {
		t.Fatalf("imagens = %d, esperado 2", len(listagem))
	}
	if listagem[0].ID != primeira.ID || listagem[1].ID != ultima.ID {
		t.Errorf("ordem = [%s %s], esperado por ordem crescente", listagem[0].ID, listagem[1].ID)
	}
	if listagem[0].Alt != "Verso" || listagem[1].Alt != "Capa" {
		t.Errorf("textos alternativos = [%q %q]", listagem[0].Alt, listagem[1].Alt)
	}

	outras, err := c.imagens.ListarPorProdutos(ctx, []uuid.UUID{produto.ID, uuid.New()})
	if err != nil {
		t.Fatalf("listagem por produtos falhou: %v", err)
	}
	if len(outras) != 2 {
		t.Errorf("imagens dos produtos = %d, esperado 2", len(outras))
	}

	vazias, err := c.imagens.ListarPorProdutos(ctx, nil)
	if err != nil {
		t.Fatalf("listagem sem produtos falhou: %v", err)
	}
	if len(vazias) != 0 {
		t.Errorf("imagens sem produtos = %d, esperado 0", len(vazias))
	}

	obtida, err := c.imagens.Obter(ctx, primeira.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if obtida.ProdutoID != produto.ID || obtida.ArquivoID != arquivoB.ID {
		t.Errorf("imagem = %+v, esperado produto e arquivo persistidos", obtida)
	}

	naoEncontrada, err := c.imagens.Obter(ctx, uuid.New())
	if err != nil {
		t.Fatalf("busca por id inexistente falhou: %v", err)
	}
	if naoEncontrada != nil {
		t.Error("imagem inexistente não deveria ser encontrada")
	}

	if _, err := c.imagens.Criar(ctx, &domainestoque.Imagem{
		ID:        uuid.New(),
		ProdutoID: produto.ID,
		ArquivoID: arquivoA.ID,
		Ordem:     9,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("arquivo repetido no mesmo produto = %v, esperado conflito", err)
	}

	produtos, err := c.imagens.ListarPorProduto(ctx, outro.ID)
	if err != nil {
		t.Fatalf("listagem do outro produto falhou: %v", err)
	}
	if len(produtos) != 0 {
		t.Errorf("produto sem imagens = %d, esperado 0", len(produtos))
	}

	removida, err := c.imagens.Remover(ctx, ultima.ID)
	if err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}
	if !removida {
		t.Fatal("remoção não apagou nenhuma linha")
	}

	removidaNovamente, err := c.imagens.Remover(ctx, ultima.ID)
	if err != nil {
		t.Fatalf("segunda remoção falhou: %v", err)
	}
	if removidaNovamente {
		t.Error("segunda remoção não deveria apagar nada")
	}

	sobrando, err := c.imagens.ListarPorProduto(ctx, produto.ID)
	if err != nil {
		t.Fatalf("listagem final falhou: %v", err)
	}
	if len(sobrando) != 1 || sobrando[0].ID != primeira.ID {
		t.Errorf("imagens restantes = %d, esperado só a primeira", len(sobrando))
	}
}

func TestProdutoListarPorCategoriaRaizIncluiSubcategoriasESemSaldo(t *testing.T) {
	c := cenarioEstoque(t)
	ctx := context.Background()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)
	outraRaiz := c.novaCategoriaRaiz(t, "Hidráulica")
	subOutra := c.novaSubcategoria(t, "Tubos", outraRaiz)

	naRaiz := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, naRaiz, domainestoque.TipoMovimentoEntrada, 4)

	naRaizSemSaldo := c.novoProduto(t, sub, "Areia Fina")

	naOutra := c.novoProduto(t, subOutra, "Tubo PVC 100mm")
	c.movimentar(t, naOutra, domainestoque.TipoMovimentoEntrada, 4)

	filtro := portsout.ProdutoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 100},
		CategoriaID:     &raiz.ID,
	}

	itens, err := c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("listagem por raiz falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Fatalf("produtos na raiz = %d, esperado 2 (expansão das subcategorias)", len(itens))
	}
	for _, item := range itens {
		if item.ID == naOutra.ID {
			t.Error("produto de outra árvore não deveria aparecer")
		}
	}

	filtro.ComSaldo = true
	comSaldo, err := c.produtos.Listar(ctx, filtro)
	if err != nil {
		t.Fatalf("listagem com saldo falhou: %v", err)
	}
	if len(comSaldo) != 1 || comSaldo[0].ID != naRaiz.ID {
		t.Errorf("produtos com saldo = %d, esperado só o cimento", len(comSaldo))
	}

	subFiltro := portsout.ProdutoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 100},
		CategoriaID:     &naRaizSemSaldo.CategoriaID,
		ComSaldo:        true,
	}
	semSaldo, err := c.produtos.Listar(ctx, subFiltro)
	if err != nil {
		t.Fatalf("listagem da subcategoria falhou: %v", err)
	}
	if len(semSaldo) != 1 || semSaldo[0].ID != naRaiz.ID {
		t.Errorf("produtos da subcategoria com saldo = %d, esperado só o cimento", len(semSaldo))
	}
}
