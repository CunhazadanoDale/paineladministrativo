package solicitacao_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func TestEnviarArquivoValidoGuardaNoStorage(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)
	conteudo := "%PDF-1.4 conteudo do orçamento"

	arquivo, err := c.arquivo.Enviar(
		c.ctx,
		dono,
		"  orcamento final.pdf  ",
		"Application/PDF; charset=binary",
		int64(len(conteudo)),
		strings.NewReader(conteudo),
	)
	if err != nil {
		t.Fatalf("envio falhou: %v", err)
	}

	if arquivo.Nome != "orcamento final.pdf" {
		t.Errorf("nome = %q, esperado sem espaços nas bordas", arquivo.Nome)
	}
	if arquivo.ContentType != "application/pdf" {
		t.Errorf("content type = %q, esperado %q", arquivo.ContentType, "application/pdf")
	}
	if arquivo.ProprietarioID != dono {
		t.Errorf("proprietário = %s, esperado %s", arquivo.ProprietarioID, dono)
	}

	chaveEsperada := dono.String() + "/" + arquivo.ID.String() + "/" + arquivo.Nome
	if arquivo.Chave != chaveEsperada {
		t.Errorf("chave = %q, esperada %q", arquivo.Chave, chaveEsperada)
	}
	if !bytes.Equal(c.storage.conteudos[arquivo.Chave], []byte(conteudo)) {
		t.Errorf("conteúdo no storage = %q, esperado %q", c.storage.conteudos[arquivo.Chave], conteudo)
	}
	if c.storage.tipos[arquivo.Chave] != "application/pdf" {
		t.Errorf("tipo no storage = %q", c.storage.tipos[arquivo.Chave])
	}
}

func TestEnviarArquivoRejeitaTiposNaoPermitidos(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)

	for _, tipo := range []string{"", "text/plain", "application/zip", "application/octet-stream"} {
		_, err := c.arquivo.Enviar(c.ctx, dono, "arquivo.txt", tipo, 10, strings.NewReader("0123456789"))
		if !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("tipo %q = %v, esperado erro de validação", tipo, err)
		}
	}

	if len(c.storage.conteudos) != 0 {
		t.Errorf("storage guardou %d arquivos rejeitados", len(c.storage.conteudos))
	}
}

func TestEnviarArquivoRejeitaTamanhoInvalido(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)

	if _, err := c.arquivo.Enviar(c.ctx, dono, "a.pdf", "application/pdf", 0, strings.NewReader("")); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tamanho zero = %v, esperado erro de validação", err)
	}

	if _, err := c.arquivo.Enviar(
		c.ctx, dono, "a.pdf", "application/pdf",
		portsin.TamanhoMaximoArquivo+1, strings.NewReader("grande"),
	); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tamanho acima do limite = %v, esperado erro de validação", err)
	}
}

func TestEnviarArquivoRejeitaNomeInvalido(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)

	for _, nome := range []string{"", "   ", "..", "pasta/.."} {
		_, err := c.arquivo.Enviar(c.ctx, dono, nome, "application/pdf", 4, strings.NewReader("abcd"))
		if !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("nome %q = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestEnviarArquivoComFalhaNoRepositorioNaoDeixaArquivoOrfao(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)
	c.arquivosRepo.falharCriacao = true

	if _, err := c.arquivo.Enviar(c.ctx, dono, "a.pdf", "application/pdf", 4, strings.NewReader("abcd")); err == nil {
		t.Fatal("envio com falha no repositório devia devolver erro")
	}
	if len(c.storage.conteudos) != 0 {
		t.Errorf("storage ficou com %d arquivos órfãos", len(c.storage.conteudos))
	}
}

func TestBaixarArquivoDoProprietario(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)
	conteudo := "conteudo do anexo"
	arquivo, err := c.arquivo.Enviar(c.ctx, dono, "nota.pdf", "application/pdf", int64(len(conteudo)), strings.NewReader(conteudo))
	if err != nil {
		t.Fatalf("envio falhou: %v", err)
	}

	baixado, leitor, err := c.arquivo.Baixar(c.ctx, arquivo.ID, dono)
	if err != nil {
		t.Fatalf("download falhou: %v", err)
	}
	defer leitor.Close()

	dados, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("leitura falhou: %v", err)
	}
	if string(dados) != conteudo {
		t.Errorf("conteúdo = %q, esperado %q", dados, conteudo)
	}
	if baixado.ID != arquivo.ID {
		t.Errorf("arquivo = %s, esperado %s", baixado.ID, arquivo.ID)
	}
}

func TestBaixarArquivoDeTerceiroRecebeProibido(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)
	estranho := c.novoUsuario("Bruno", false, false)
	arquivo := c.enviarArquivo(t, dono, "nota.pdf", "application/pdf")

	if _, _, err := c.arquivo.Baixar(c.ctx, arquivo.ID, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("download por terceiro = %v, esperado erro de permissão", err)
	}
}

func TestBaixarArquivoDeSolicitacaoVisivelPermitidoParaAprovador(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)

	arquivo := c.enviarArquivo(t, solicitante, "nota.pdf", "application/pdf")
	c.criarSolicitacaoComArquivos(t, solicitante, []uuid.UUID{arquivo.ID})

	_, leitor, err := c.arquivo.Baixar(c.ctx, arquivo.ID, aprovador)
	if err != nil {
		t.Fatalf("download pelo aprovador falhou: %v", err)
	}
	_ = leitor.Close()
}

func TestBaixarArquivoInexistenteRetornaNaoEncontrado(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)

	if _, _, err := c.arquivo.Baixar(c.ctx, uuid.New(), dono); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("download inexistente = %v, esperado não encontrado", err)
	}
}

func TestRemoverArquivoVinculadoRecebeConflito(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	arquivo := c.enviarArquivo(t, solicitante, "nota.pdf", "application/pdf")
	c.criarSolicitacaoComArquivos(t, solicitante, []uuid.UUID{arquivo.ID})

	if err := c.arquivo.Remover(c.ctx, arquivo.ID, solicitante); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("remoção de vinculado = %v, esperado erro de conflito", err)
	}
}

func TestRemoverArquivoDeTerceiroRecebeProibido(t *testing.T) {
	c := novoCenario(t)
	dono := c.novoUsuario("Ana", false, false)
	estranho := c.novoUsuario("Bruno", false, false)
	arquivo := c.enviarArquivo(t, dono, "nota.pdf", "application/pdf")

	if err := c.arquivo.Remover(c.ctx, arquivo.ID, estranho); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("remoção por terceiro = %v, esperado erro de permissão", err)
	}

	admin := c.novoUsuario("Carla", true, false)
	if err := c.arquivo.Remover(c.ctx, arquivo.ID, admin); err != nil {
		t.Errorf("remoção pelo administrador = %v, esperado permitido", err)
	}
	if len(c.storage.conteudos) != 0 {
		t.Errorf("storage ficou com %d arquivos após remoção", len(c.storage.conteudos))
	}
}

func TestListarArquivosDoProprietario(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	bruno := c.novoUsuario("Bruno", false, false)
	c.enviarArquivo(t, ana, "primeiro.pdf", "application/pdf")
	c.enviarArquivo(t, ana, "segundo.pdf", "application/pdf")
	c.enviarArquivo(t, bruno, "do-bruno.pdf", "application/pdf")

	itens, err := c.arquivo.ListarPorProprietario(c.ctx, ana, domain.PaginacaoFiltro{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Errorf("%d arquivos, esperado 2", len(itens))
	}
	for _, arquivo := range itens {
		if arquivo.ProprietarioID != ana {
			t.Errorf("arquivo %s de outro proprietário", arquivo.ID)
		}
	}
}
