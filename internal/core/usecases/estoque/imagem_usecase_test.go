package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func TestAnexarImagemExigeAdministrador(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	arquivo := c.novoArquivo(t, "image/png")

	_, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.visitanteID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
	})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("erro = %v, esperado erro de permissão", err)
	}
}

func TestAnexarImagemValidaProdutoEArquivo(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	arquivo := c.novoArquivo(t, "image/png")

	casos := []struct {
		nome  string
		input portsin.AnexarImagemInput
		erro  error
	}{
		{
			nome: "produto inexistente",
			input: portsin.AnexarImagemInput{
				UsuarioID: c.usuarioID,
				ProdutoID: uuid.New(),
				ArquivoID: arquivo.ID,
			},
			erro: domain.ErrNotFound,
		},
		{
			nome: "arquivo inexistente",
			input: portsin.AnexarImagemInput{
				UsuarioID: c.usuarioID,
				ProdutoID: produto.ID,
				ArquivoID: uuid.New(),
			},
			erro: domain.ErrNotFound,
		},
		{
			nome: "arquivo que não é imagem",
			input: portsin.AnexarImagemInput{
				UsuarioID: c.usuarioID,
				ProdutoID: produto.ID,
				ArquivoID: c.novoArquivo(t, "application/pdf").ID,
			},
			erro: domain.ErrValidacao,
		},
		{
			nome: "ordem negativa",
			input: portsin.AnexarImagemInput{
				UsuarioID: c.usuarioID,
				ProdutoID: produto.ID,
				ArquivoID: arquivo.ID,
				Ordem:     -1,
			},
			erro: domain.ErrValidacao,
		},
	}

	for _, caso := range casos {
		if _, err := c.imagem.Anexar(c.ctx, caso.input); !errors.Is(err, caso.erro) {
			t.Errorf("%s = %v, esperado erro %v", caso.nome, err, caso.erro)
		}
	}
}

func TestAnexarImagemComArquivoRepetidoFalha(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	arquivo := c.novoArquivo(t, "image/png")

	if _, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
	}); err != nil {
		t.Fatalf("primeira anexação falhou: %v", err)
	}

	_, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
	})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("erro = %v, esperado erro de conflito", err)
	}
}

func TestAnexarImagemPersisteOrdemETextoAlternativo(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")

	arquivo := c.novoArquivo(t, "image/webp")
	imagem, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
		Ordem:     3,
		Alt:       "  Saco de cimento  ",
	})
	if err != nil {
		t.Fatalf("anexação falhou: %v", err)
	}

	if imagem.ID == uuid.Nil {
		t.Error("imagem nasceu sem identificador")
	}
	if imagem.Ordem != 3 {
		t.Errorf("ordem = %d, esperado 3", imagem.Ordem)
	}
	if imagem.Alt != "Saco de cimento" {
		t.Errorf("texto alternativo = %q, esperado sem espaços nas bordas", imagem.Alt)
	}

	detalhada, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca do produto falhou: %v", err)
	}
	if len(detalhada.Imagens) != 1 {
		t.Fatalf("imagens do produto = %d, esperado 1", len(detalhada.Imagens))
	}
	if detalhada.Imagens[0].ID != imagem.ID || detalhada.Imagens[0].ArquivoID != arquivo.ID {
		t.Errorf("imagem = %+v, esperado a imagem anexada", detalhada.Imagens[0])
	}
}

func TestRemoverImagemConfereProdutoDono(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	outro := c.novoProduto(t, sub, "Areia Fina")
	imagem := c.novaImagem(t, produto, "image/png", 0)

	if err := c.imagem.Remover(c.ctx, c.usuarioID, outro.ID, imagem.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("produto de outra imagem = %v, esperado erro não encontrado", err)
	}

	if err := c.imagem.Remover(c.ctx, c.usuarioID, produto.ID, imagem.ID); err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}

	detalhada, err := c.produto.Obter(c.ctx, produto.ID)
	if err != nil {
		t.Fatalf("busca do produto falhou: %v", err)
	}
	if len(detalhada.Imagens) != 0 {
		t.Errorf("imagens do produto = %d, esperado 0 após a remoção", len(detalhada.Imagens))
	}
}

func TestRemoverImagemExigeAdministrador(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	imagem := c.novaImagem(t, produto, "image/png", 0)

	if err := c.imagem.Remover(c.ctx, c.visitanteID, produto.ID, imagem.ID); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("erro = %v, esperado erro de permissão", err)
	}
}

func TestObterImagemPublicaExigeProdutoAtivo(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, produto, "entrada", 10)
	imagem := c.novaImagem(t, produto, "image/png", 0)

	publica, err := c.imagem.ObterPublica(c.ctx, imagem.ID)
	if err != nil {
		t.Fatalf("busca pública falhou: %v", err)
	}
	if publica.ID != imagem.ID {
		t.Errorf("imagem = %s, esperado %s", publica.ID, imagem.ID)
	}

	if err := c.produto.AlternarAtivo(c.ctx, portsin.AlternarAtivoProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		Ativo:     false,
	}); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	if _, err := c.imagem.ObterPublica(c.ctx, imagem.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("imagem de produto inativo = %v, esperado erro não encontrado", err)
	}

	if _, err := c.imagem.ObterPublica(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("imagem inexistente = %v, esperado erro não encontrado", err)
	}
}

func TestObterImagemPublicaSegueAsRegrasDaVitrine(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	imagem := c.novaImagem(t, produto, "image/png", 0)

	if _, err := c.imagem.ObterPublica(c.ctx, imagem.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("imagem de produto sem saldo = %v, esperado não encontrado", err)
	}

	c.movimentar(t, produto, "entrada", 10)
	if _, err := c.imagem.ObterPublica(c.ctx, imagem.ID); err != nil {
		t.Fatalf("imagem de produto visível falhou: %v", err)
	}

	c.categoriasRepo.itens[raiz.ID].Ativo = false
	if _, err := c.imagem.ObterPublica(c.ctx, imagem.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("imagem de produto com categoria pai inativa = %v, esperado não encontrado", err)
	}
}

func TestAnexarImagemRecusaArquivoDeSolicitacao(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	arquivo := c.novoArquivo(t, "image/png")
	c.arquivosRepo.vinculados[arquivo.ID] = true

	if _, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("anexo de solicitação como imagem = %v, esperado erro de validação", err)
	}
}

func TestRemoverImagemDescartaArquivoSemOutroUso(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	cimento := c.novoProduto(t, sub, "Cimento CP II 50kg")
	areia := c.novoProduto(t, sub, "Areia média")
	compartilhada := c.novaImagem(t, cimento, "image/png", 0)

	if _, err := c.imagem.Anexar(c.ctx, portsin.AnexarImagemInput{
		UsuarioID: c.usuarioID,
		ProdutoID: areia.ID,
		ArquivoID: compartilhada.ArquivoID,
	}); err != nil {
		t.Fatalf("segundo vínculo falhou: %v", err)
	}

	if err := c.imagem.Remover(c.ctx, c.usuarioID, cimento.ID, compartilhada.ID); err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}
	if _, existe := c.arquivosRepo.itens[compartilhada.ArquivoID]; !existe || len(c.storage.removidas) != 0 {
		t.Fatal("arquivo ainda usado por outro produto foi descartado")
	}

	imagensDaAreia, err := c.imagensRepo.ListarPorProduto(c.ctx, areia.ID)
	if err != nil || len(imagensDaAreia) != 1 {
		t.Fatalf("imagens da areia = %v, %v", imagensDaAreia, err)
	}
	if err := c.imagem.Remover(c.ctx, c.usuarioID, areia.ID, imagensDaAreia[0].ID); err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}
	if _, existe := c.arquivosRepo.itens[compartilhada.ArquivoID]; existe {
		t.Error("arquivo sem uso continuou cadastrado")
	}
	if len(c.storage.removidas) != 1 {
		t.Errorf("%d remoções no storage, esperado 1", len(c.storage.removidas))
	}
}
