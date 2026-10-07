package solicitacao_test

import (
	"context"
	"strings"
	"testing"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	"github.com/google/uuid"
)

type cenario struct {
	ctx              context.Context
	solicitacao      portsin.SolicitacaoUseCase
	arquivo          portsin.ArquivoUseCase
	aprovador        portsin.AprovadorUseCase
	usuariosRepo     *repositorioUsuarios
	cargosRepo       *repositorioCargos
	arquivosRepo     *repositorioArquivos
	solicitacoesRepo *repositorioSolicitacoes
	aprovadoresRepo  *repositorioAprovadores
	storage          *storageFalso
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	usuarios := novoRepositorioUsuarios()
	cargos := novoRepositorioCargos()
	arquivos := novoRepositorioArquivos()
	solicitacoes := novoRepositorioSolicitacoes(arquivos)
	aprovadores := novoRepositorioAprovadores()
	storage := novoStorageFalso()

	return &cenario{
		ctx:              context.Background(),
		solicitacao:      solicitacaousecases.NewSolicitacaoUsecase(solicitacoes, arquivos, aprovadores, usuarios, cargos),
		arquivo:          solicitacaousecases.NewArquivoUsecase(arquivos, solicitacoes, storage, usuarios, cargos, aprovadores),
		aprovador:        solicitacaousecases.NewAprovadorUsecase(aprovadores, usuarios),
		usuariosRepo:     usuarios,
		cargosRepo:       cargos,
		arquivosRepo:     arquivos,
		solicitacoesRepo: solicitacoes,
		aprovadoresRepo:  aprovadores,
		storage:          storage,
	}
}

func (c *cenario) novoUsuario(nome string, administrador, financeiro bool) uuid.UUID {
	cargoID := uuid.New()
	c.cargosRepo.itens[cargoID] = &domainusuarios.Cargo{
		ID:            cargoID,
		Nome:          "Cargo de " + nome,
		Ativo:         true,
		Administrador: administrador,
		Financeiro:    financeiro,
	}

	usuarioID := uuid.New()
	c.usuariosRepo.itens[usuarioID] = &domainusuarios.Usuario{
		ID:       usuarioID,
		Nome:     nome,
		Email:    nome + "@exemplo.com",
		CargoID:  cargoID,
		Ativo:    true,
		CriadoEm: time.Now().UTC(),
	}

	return usuarioID
}

func (c *cenario) designar(t *testing.T, usuarioID uuid.UUID) uuid.UUID {
	t.Helper()

	if _, err := c.aprovador.Designar(c.ctx, usuarioID); err != nil {
		t.Fatalf("designação de aprovador falhou: %v", err)
	}

	aprovador, err := c.aprovadoresRepo.ObterPorUsuarioID(c.ctx, usuarioID)
	if err != nil || aprovador == nil {
		t.Fatalf("aprovador designado não encontrado: %v", err)
	}

	return aprovador.ID
}

func (c *cenario) criarSolicitacao(t *testing.T, solicitanteID uuid.UUID, valorCentavos int64) uuid.UUID {
	t.Helper()

	id, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  solicitanteID,
		ValorCentavos:  valorCentavos,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 7),
		Observacao:     "material para a obra",
		FormaPagamento: "pix",
	})
	if err != nil {
		t.Fatalf("criação da solicitação falhou: %v", err)
	}

	return id
}

func (c *cenario) criarSolicitacaoComArquivos(t *testing.T, solicitanteID uuid.UUID, arquivoIDs []uuid.UUID) uuid.UUID {
	t.Helper()

	id, err := c.solicitacao.Criar(c.ctx, portsin.CriarSolicitacaoInput{
		SolicitanteID:  solicitanteID,
		ValorCentavos:  10000,
		PrazoPagamento: time.Now().UTC().AddDate(0, 0, 7),
		Observacao:     "material para a obra",
		FormaPagamento: "pix",
		ArquivoIDs:     arquivoIDs,
	})
	if err != nil {
		t.Fatalf("criação da solicitação com arquivos falhou: %v", err)
	}

	return id
}

func (c *cenario) enviarArquivo(t *testing.T, proprietarioID uuid.UUID, nome, contentType string) *domainsolicitacao.Arquivo {
	t.Helper()

	conteudo := "conteudo-teste"
	arquivo, err := c.arquivo.Enviar(
		c.ctx,
		proprietarioID,
		nome,
		contentType,
		int64(len(conteudo)),
		strings.NewReader(conteudo),
	)
	if err != nil {
		t.Fatalf("envio do arquivo falhou: %v", err)
	}

	return arquivo
}
