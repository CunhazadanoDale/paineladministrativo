package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func TestCriarProdutoExigeAdministrador(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.visitanteID,
		CategoriaID:   sub.ID,
		Nome:          "Cimento CP II 50kg",
		UnidadeMedida: "sc",
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("criação por visitante = %v, esperado erro de permissão", err)
	}
}

func TestCriarProdutoExigeSubcategoria(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   raiz.ID,
		Nome:          "Cimento CP II 50kg",
		UnidadeMedida: "sc",
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("produto em categoria raiz = %v, esperado erro de validação", err)
	}
}

func TestCriarProdutoValidaCategoriaInexistente(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   uuid.New(),
		Nome:          "Cimento CP II 50kg",
		UnidadeMedida: "sc",
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("categoria inexistente = %v, esperado erro de não encontrado", err)
	}
}

func TestCriarProdutoValidaUnidadeDeMedida(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   sub.ID,
		Nome:          "Cimento CP II 50kg",
		UnidadeMedida: "balde",
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("unidade inválida = %v, esperado erro de validação", err)
	}
}

func TestCriarProdutoCompletoPersisteTodosOsCampos(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	preco := int64(2590)
	promocional := int64(2200)
	minimo := 5
	peso := 50.5
	codigo := "CIM-0001"

	id, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:                c.usuarioID,
		CategoriaID:              sub.ID,
		Nome:                     "  Cimento CP II 50kg  ",
		Descricao:                "Cimento de uso geral",
		Codigo:                   &codigo,
		UnidadeMedida:            "sc",
		PrecoCentavos:            &preco,
		PrecoPromocionalCentavos: &promocional,
		EstoqueMinimo:            &minimo,
		PesoKg:                   &peso,
		Destaque:                 true,
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	produto, err := c.produto.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if produto.Nome != "Cimento CP II 50kg" {
		t.Errorf("nome = %q, esperado sem espaços nas bordas", produto.Nome)
	}
	if produto.Slug.Valor() != "cimento-cp-ii-50kg" {
		t.Errorf("slug = %q", produto.Slug.Valor())
	}
	if produto.CategoriaID != sub.ID {
		t.Errorf("categoria = %s, esperado %s", produto.CategoriaID, sub.ID)
	}
	if produto.Preco == nil || produto.Preco.Centavos() != 2590 {
		t.Errorf("preço = %v, esperado 2590 centavos", produto.Preco)
	}
	if produto.PrecoPromocional == nil || produto.PrecoPromocional.Centavos() != 2200 {
		t.Errorf("promocional = %v, esperado 2200 centavos", produto.PrecoPromocional)
	}
	if produto.EstoqueMinimo == nil || produto.EstoqueMinimo.Quantidade() != 5 {
		t.Errorf("mínimo = %v, esperado 5", produto.EstoqueMinimo)
	}
	if produto.Peso == nil || produto.Peso.Quilos() != 50.5 {
		t.Errorf("peso = %v, esperado 50.5kg", produto.Peso)
	}
	if produto.Codigo == nil || *produto.Codigo != "CIM-0001" {
		t.Errorf("código = %v, esperado CIM-0001", produto.Codigo)
	}
	if !produto.Destaque || !produto.Ativo {
		t.Error("produto deveria nascer em destaque e ativo")
	}
	if !produto.Saldo.Vazio() {
		t.Errorf("saldo = %d, esperado 0", produto.Saldo.Quantidade())
	}
}

func TestCriarProdutoComNomeRepetidoFalha(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	c.novoProduto(t, sub, "Cimento CP II 50kg")

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   sub.ID,
		Nome:          "cimento cp ii 50kg",
		UnidadeMedida: "sc",
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("nome repetido = %v, esperado erro de conflito", err)
	}
}

func TestCriarProdutoValidaPrecoPromocional(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	cheio := int64(1000)
	promocional := int64(1500)

	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:                c.usuarioID,
		CategoriaID:              sub.ID,
		Nome:                     "Cimento CP II 50kg",
		UnidadeMedida:            "sc",
		PrecoCentavos:            &cheio,
		PrecoPromocionalCentavos: &promocional,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("promocional maior que o cheio = %v, esperado erro de validação", err)
	}
}

func TestCriarProdutoAceitaSobConsulta(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	produto := c.novoProduto(t, sub, "Areia Fina")

	if produto.Preco != nil || produto.PrecoPromocional != nil {
		t.Error("produto sob consulta não deveria ter preço")
	}
}

func TestAlterarProdutoPreservaSaldoESlug(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, produto, "entrada", 8)

	preco := int64(3200)
	err := c.produto.Alterar(c.ctx, portsin.AlterarProdutoInput{
		UsuarioID:     c.usuarioID,
		ProdutoID:     produto.ID,
		CategoriaID:   sub.ID,
		Nome:          "Cimento CP II 50kg (saco)",
		UnidadeMedida: "sc",
		PrecoCentavos: &preco,
	})
	if err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}

	salvo, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Nome != "Cimento CP II 50kg (saco)" {
		t.Errorf("nome = %q", salvo.Nome)
	}
	if salvo.Slug.Valor() != "cimento-cp-ii-50kg" {
		t.Errorf("slug = %q, esperado manter o slug original", salvo.Slug.Valor())
	}
	if salvo.Saldo.Quantidade() != 8 {
		t.Errorf("saldo = %d, esperado 8 (preservado)", salvo.Saldo.Quantidade())
	}
	if salvo.Preco == nil || salvo.Preco.Centavos() != 3200 {
		t.Errorf("preço = %v, esperado 3200", salvo.Preco)
	}
}

func TestAlterarProdutoExigeAdministrador(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	err := c.produto.Alterar(c.ctx, portsin.AlterarProdutoInput{
		UsuarioID:     c.visitanteID,
		ProdutoID:     produto.ID,
		CategoriaID:   sub.ID,
		Nome:          "Outro Nome",
		UnidadeMedida: "sc",
	})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("alteração por visitante = %v, esperado erro de permissão", err)
	}
}

func TestAlternarAtivoEDestaqueDoProduto(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	if err := c.produto.AlternarAtivo(c.ctx, portsin.AlternarAtivoProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		Ativo:     false,
	}); err != nil {
		t.Fatalf("alternar ativo falhou: %v", err)
	}
	if err := c.produto.AlternarDestaque(c.ctx, portsin.AlternarDestaqueProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		Destaque:  true,
	}); err != nil {
		t.Fatalf("alternar destaque falhou: %v", err)
	}

	salvo, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Ativo {
		t.Error("produto deveria estar inativo")
	}
	if !salvo.Destaque {
		t.Error("produto deveria estar em destaque")
	}
}

func TestListarProdutosComFiltros(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	cimento := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, cimento, "entrada", 3)

	argamassa := c.novoProduto(t, sub, "Argamassa AC III")
	if err := c.produto.AlternarDestaque(c.ctx, portsin.AlternarDestaqueProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: argamassa.ID,
		Destaque:  true,
	}); err != nil {
		t.Fatalf("alternar destaque falhou: %v", err)
	}

	itens, err := c.produto.Listar(c.ctx, portsin.ListarProdutosInput{Busca: "cimento"})
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != cimento.ID {
		t.Errorf("busca = %+v, esperado só o cimento", itens)
	}

	itens, err = c.produto.Listar(c.ctx, portsin.ListarProdutosInput{EstoqueBaixo: true})
	if err != nil {
		t.Fatalf("filtro de estoque baixo falhou: %v", err)
	}
	if len(itens) != 0 {
		t.Errorf("estoque baixo = %+v, esperado vazio sem estoque mínimo definido", itens)
	}

	destaque := true
	itens, err = c.produto.Listar(c.ctx, portsin.ListarProdutosInput{Destaque: &destaque})
	if err != nil {
		t.Fatalf("filtro de destaque falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].ID != argamassa.ID {
		t.Errorf("destaque = %+v, esperado só a argamassa", itens)
	}

	categoriaID := raiz.ID
	itens, err = c.produto.Listar(c.ctx, portsin.ListarProdutosInput{CategoriaID: &categoriaID})
	if err != nil {
		t.Fatalf("filtro de categoria falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Errorf("filtro na raiz = %d produtos, esperado 2 (expande as subcategorias)", len(itens))
	}

	subcategoriaID := sub.ID
	itens, err = c.produto.Listar(c.ctx, portsin.ListarProdutosInput{CategoriaID: &subcategoriaID})
	if err != nil {
		t.Fatalf("filtro de subcategoria falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Errorf("filtro na subcategoria = %d produtos, esperado 2", len(itens))
	}
}

func TestMovimentarEstoqueAtualizaSaldoDoProduto(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	c.movimentar(t, produto, "entrada", 20)
	c.movimentar(t, produto, "saida", 6)

	salvo, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Saldo.Quantidade() != 14 {
		t.Errorf("saldo = %d, esperado 14", salvo.Saldo.Quantidade())
	}
}

func TestMovimentarEstoqueBloqueiaSaldoNegativo(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	c.movimentar(t, produto, "entrada", 5)

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  produto.ID,
		Tipo:       "saida",
		Quantidade: 10,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída maior que o saldo = %v, esperado erro de validação", err)
	}

	salvo, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salvo.Saldo.Quantidade() != 5 {
		t.Errorf("saldo = %d, esperado permanecer 5", salvo.Saldo.Quantidade())
	}
}

func TestMovimentarEstoqueExigeAdministrador(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.visitanteID,
		ProdutoID:  produto.ID,
		Tipo:       "entrada",
		Quantidade: 10,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("movimentação por visitante = %v, esperado erro de permissão", err)
	}
}

func TestMovimentarEstoqueValidaTipoEQuantidade(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  produto.ID,
		Tipo:       "roubo",
		Quantidade: 1,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tipo inválido = %v, esperado erro de validação", err)
	}

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  produto.ID,
		Tipo:       "entrada",
		Quantidade: 0,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("quantidade zero = %v, esperado erro de validação", err)
	}
}

func TestMovimentarEstoqueValidaProdutoInexistente(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  uuid.New(),
		Tipo:       "entrada",
		Quantidade: 1,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("produto inexistente = %v, esperado erro de não encontrado", err)
	}
}

func TestListarMovimentosRegistraHistoricoCompleto(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	c.movimentar(t, produto, "entrada", 20)
	c.movimentar(t, produto, "saida", 6)

	produtoID := produto.ID
	movimentos, err := c.movimento.Listar(c.ctx, portsin.ListarMovimentosInput{ProdutoID: &produtoID})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(movimentos) != 2 {
		t.Fatalf("%d movimentos, esperado 2", len(movimentos))
	}

	porTipo := map[domainestoque.TipoMovimento]*domainestoque.Movimento{}
	for _, movimento := range movimentos {
		porTipo[movimento.Tipo] = movimento
	}

	entrada, ok := porTipo[domainestoque.TipoMovimentoEntrada]
	if !ok || entrada.SaldoApos.Quantidade() != 20 {
		t.Errorf("entrada = %+v, esperado saldo 20", entrada)
	}
	saida, ok := porTipo[domainestoque.TipoMovimentoSaida]
	if !ok || saida.SaldoApos.Quantidade() != 14 {
		t.Errorf("saída = %+v, esperado saldo 14", saida)
	}
	if saida.Quantidade.Valor() != 6 {
		t.Errorf("quantidade da saída = %d, esperado 6", saida.Quantidade.Valor())
	}

	saidas, err := c.movimento.Listar(c.ctx, portsin.ListarMovimentosInput{Tipo: "saida"})
	if err != nil {
		t.Fatalf("listagem por tipo falhou: %v", err)
	}
	if len(saidas) != 1 {
		t.Errorf("%d saídas, esperado 1", len(saidas))
	}
}
