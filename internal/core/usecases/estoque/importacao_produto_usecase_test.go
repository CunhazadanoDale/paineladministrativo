package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
)

func (c *cenario) importacaoDeProdutos() *estoqueusecases.ImportacaoProdutoUsecaseImpl {
	return estoqueusecases.NewImportacaoProdutoUsecase(c.produtosRepo, c.categoriasRepo, c.usuariosRepo, c.cargosRepo)
}

func linhaDeProduto(numero int, nome, codigo, categoria, preco string) portsin.LinhaImportacaoProduto {
	return portsin.LinhaImportacaoProduto{Numero: numero, Nome: nome, Codigo: codigo, Categoria: categoria, UnidadeMedida: "sc", Preco: preco}
}

func TestImportarProdutosGravaValidosERejeitaInvalidos(t *testing.T) {
	c := novoCenario(t)
	cimentos := c.novaSubcategoria(t, "Cimentos", c.novaCategoriaRaiz(t, "Materiais"))
	codigoAntigo := "ANT-1"
	if _, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{UsuarioID: c.usuarioID, CategoriaID: cimentos.ID, Nome: "Cimento antigo", Codigo: &codigoAntigo, UnidadeMedida: "sc"}); err != nil {
		t.Fatalf("criação do produto existente falhou: %v", err)
	}

	completo := linhaDeProduto(2, "Cimento CP II 50kg", "CIM-50", "cimentos", "R$ 1.239,90")
	completo.PrecoPromocional = "1.199,90"
	completo.EstoqueMinimo = "20"
	completo.PesoKg = "50,5"
	completo.Descricao = "Saco"
	completo.Destaque = "sim"
	linhas := []portsin.LinhaImportacaoProduto{
		completo,
		linhaDeProduto(3, "Argamassa AC-I", "", "Cimentos", ""),
		linhaDeProduto(4, "cimento cp ii 50KG", "", "Cimentos", ""),
		linhaDeProduto(5, "Outro cimento", "CIM-50", "Cimentos", ""),
		linhaDeProduto(6, "Cimento novo", "ANT-1", "Cimentos", ""),
		linhaDeProduto(7, "Cimento Antigo", "", "Cimentos", ""),
		linhaDeProduto(8, "Areia", "", "Agregados", ""),
		linhaDeProduto(9, "Brita", "", "Materiais", ""),
		linhaDeProduto(10, "Cal", "", "Cimentos", "dez reais"),
		linhaDeProduto(11, "", "", "Cimentos", ""),
	}
	semUnidade := linhaDeProduto(12, "Gesso", "", "Cimentos", "")
	semUnidade.UnidadeMedida = "saco"
	linhas = append(linhas, semUnidade)

	relatorio, err := c.importacaoDeProdutos().Importar(c.ctx, portsin.ImportarProdutosInput{UsuarioID: c.usuarioID, Linhas: linhas})

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if relatorio.Total != 11 || relatorio.Importadas != 2 {
		t.Errorf("relatório = %+v", relatorio)
	}
	esperados := map[int]string{
		4:  "nome repetido no arquivo (linha 2)",
		5:  "código repetido no arquivo (linha 2)",
		6:  "já existe um produto com este código",
		7:  "já existe um produto com este nome",
		8:  "categoria Agregados não encontrada",
		10: "preco deve ser um valor em reais, como 1234,56",
		11: "nome do produto é obrigatório",
	}
	motivos := map[int]string{}
	for _, rejeitada := range relatorio.Rejeitadas {
		motivos[rejeitada.Linha] = rejeitada.Motivo
	}
	for linha, motivo := range esperados {
		if motivos[linha] != motivo {
			t.Errorf("linha %d: motivo %q, esperado %q", linha, motivos[linha], motivo)
		}
	}
	if motivos[9] == "" || motivos[12] == "" || len(motivos) != 9 {
		t.Errorf("motivos = %v", motivos)
	}

	var gravado *domainestoque.Produto
	for _, produto := range c.produtosRepo.itens {
		if produto.Nome == "Cimento CP II 50kg" {
			gravado = produto
		}
	}
	if gravado == nil {
		t.Fatal("produto completo não foi gravado")
	}
	if gravado.Preco == nil || gravado.Preco.Centavos() != 123990 || gravado.PrecoPromocional == nil || gravado.PrecoPromocional.Centavos() != 119990 ||
		gravado.Codigo == nil || *gravado.Codigo != "CIM-50" || !gravado.Destaque || gravado.Peso == nil || gravado.EstoqueMinimo == nil {
		t.Errorf("produto gravado = %+v", gravado)
	}
}

func TestImportarProdutosSimulandoNaoGrava(t *testing.T) {
	c := novoCenario(t)
	c.novaSubcategoria(t, "Cimentos", c.novaCategoriaRaiz(t, "Materiais"))

	relatorio, err := c.importacaoDeProdutos().Importar(c.ctx, portsin.ImportarProdutosInput{
		UsuarioID: c.usuarioID,
		Simular:   true,
		Linhas:    []portsin.LinhaImportacaoProduto{linhaDeProduto(2, "Cimento", "", "Cimentos", "10")},
	})

	if err != nil || !relatorio.Simulacao || relatorio.Importadas != 1 {
		t.Fatalf("relatório = %+v, erro = %v", relatorio, err)
	}
	if len(c.produtosRepo.itens) != 0 {
		t.Errorf("simulação gravou %d produtos", len(c.produtosRepo.itens))
	}
}

func TestImportarProdutosExigeAdministrador(t *testing.T) {
	c := novoCenario(t)

	_, err := c.importacaoDeProdutos().Importar(c.ctx, portsin.ImportarProdutosInput{UsuarioID: c.visitanteID})

	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("erro = %v, esperado permissão", err)
	}
}
