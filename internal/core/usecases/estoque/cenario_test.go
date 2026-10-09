package estoque_test

import (
	"context"
	"testing"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
	"github.com/google/uuid"
)

type cenario struct {
	ctx            context.Context
	categoria      portsin.CategoriaUseCase
	produto        portsin.ProdutoUseCase
	movimento      portsin.MovimentoUseCase
	resumo         portsin.ResumoUseCase
	imagem         portsin.ImagemUseCase
	categoriasRepo *repositorioCategorias
	produtosRepo   *repositorioProdutos
	imagensRepo    *repositorioImagens
	arquivosRepo   *repositorioArquivos
	usuarioID      uuid.UUID
	visitanteID    uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	usuarios := novoRepositorioUsuarios()
	cargos := novoRepositorioCargos()
	categorias := novoRepositorioCategorias()
	produtos := novoRepositorioProdutos(categorias)
	movimentos := novoRepositorioMovimentos(produtos)
	imagens := novoRepositorioImagens()
	arquivos := novoRepositorioArquivos()

	adminID := criarUsuario(t, usuarios, cargos, "Ana Souza", "ana.souza@exemplo.com", true)
	visitanteID := criarUsuario(t, usuarios, cargos, "Bruno Lima", "bruno.lima@exemplo.com", false)

	return &cenario{
		ctx:            context.Background(),
		categoria:      estoqueusecases.NewCategoriaUsecase(categorias, usuarios, cargos),
		produto:        estoqueusecases.NewProdutoUsecase(produtos, categorias, imagens, usuarios, cargos),
		movimento:      estoqueusecases.NewMovimentoUsecase(produtos, movimentos, usuarios, cargos),
		resumo:         estoqueusecases.NewResumoUsecase(produtos, movimentos),
		imagem:         estoqueusecases.NewImagemUsecase(imagens, produtos, arquivos, usuarios, cargos),
		categoriasRepo: categorias,
		produtosRepo:   produtos,
		imagensRepo:    imagens,
		arquivosRepo:   arquivos,
		usuarioID:      adminID,
		visitanteID:    visitanteID,
	}
}

func criarUsuario(t *testing.T, usuarios *repositorioUsuarios, cargos *repositorioCargos, nome, email string, administrador bool) uuid.UUID {
	t.Helper()

	cargoID, err := cargos.Create(context.Background(), &domainusuarios.Cargo{
		ID:            uuid.New(),
		Nome:          "Cargo " + nome,
		Ativo:         true,
		Administrador: administrador,
	})
	if err != nil {
		t.Fatalf("não criei o cargo do cenário: %v", err)
	}

	usuarioID, err := usuarios.Create(context.Background(), &domainusuarios.Usuario{
		ID:           uuid.New(),
		Nome:         nome,
		Email:        email,
		Senha:        "hash-segredo",
		CargoID:      cargoID,
		Ativo:        true,
		CriadoEm:     time.Now().UTC(),
		AtualizadoEm: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("não criei o usuário do cenário: %v", err)
	}

	return usuarioID
}

func (c *cenario) novaCategoriaRaiz(t *testing.T, nome string) *domainestoque.Categoria {
	t.Helper()

	id, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID: c.usuarioID,
		Nome:      nome,
	})
	if err != nil {
		t.Fatalf("criação da categoria falhou: %v", err)
	}

	categoria, err := c.categoria.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("busca da categoria falhou: %v", err)
	}

	return categoria
}

func (c *cenario) novaSubcategoria(t *testing.T, nome string, raiz *domainestoque.Categoria) *domainestoque.Categoria {
	t.Helper()

	paiID := raiz.ID
	id, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID:      c.usuarioID,
		Nome:           nome,
		CategoriaPaiID: &paiID,
	})
	if err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	subcategoria, err := c.categoria.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("busca da subcategoria falhou: %v", err)
	}

	return subcategoria
}

func (c *cenario) novoProduto(t *testing.T, categoria *domainestoque.Categoria, nome string) *domainestoque.Produto {
	t.Helper()

	id, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   categoria.ID,
		Nome:          nome,
		UnidadeMedida: "sc",
	})
	if err != nil {
		t.Fatalf("criação do produto falhou: %v", err)
	}

	produto, err := c.produto.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("busca do produto falhou: %v", err)
	}

	return produto
}

func (c *cenario) movimentar(t *testing.T, produto *domainestoque.Produto, tipo string, quantidade int) {
	t.Helper()

	if _, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  produto.ID,
		Tipo:       tipo,
		Quantidade: quantidade,
	}); err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}
}

func (c *cenario) novoArquivo(t *testing.T, contentType string) *domainsolicitacao.Arquivo {
	t.Helper()

	arquivo := &domainsolicitacao.Arquivo{
		ID:             uuid.New(),
		ProprietarioID: c.usuarioID,
		Nome:           "foto-do-produto",
		Chave:          "estoque/" + uuid.NewString(),
		ContentType:    contentType,
		Tamanho:        2048,
		CriadoEm:       time.Now().UTC(),
	}

	if _, err := c.arquivosRepo.Criar(c.ctx, arquivo); err != nil {
		t.Fatalf("criação do arquivo falhou: %v", err)
	}

	return arquivo
}

func (c *cenario) novaImagem(t *testing.T, produto *domainestoque.Produto, contentType string, ordem int) *domainestoque.Imagem {
	t.Helper()

	arquivo := c.novoArquivo(t, contentType)

	imagem, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
		Ordem:     ordem,
		Alt:       produto.Nome,
	})
	if err != nil {
		t.Fatalf("anexação da imagem falhou: %v", err)
	}

	return imagem
}
