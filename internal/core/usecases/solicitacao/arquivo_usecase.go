package solicitacao

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

const tamanhoMaximoNomeArquivo = 255

var _ portsin.ArquivoUseCase = (*ArquivoUsecaseImpl)(nil)

type ArquivoUsecaseImpl struct {
	repo         portsout.ArquivoRepository
	solicitacoes portsout.SolicitacaoRepository
	storage      portsout.Storage
	permissoes
}

func NewArquivoUsecase(
	arquivos portsout.ArquivoRepository,
	solicitacoes portsout.SolicitacaoRepository,
	storage portsout.Storage,
	usuarios portsoutusuarios.UsuarioRepository,
	cargos portsoutusuarios.CargoRepository,
	aprovadores portsout.AprovadorRepository,
) *ArquivoUsecaseImpl {
	return &ArquivoUsecaseImpl{
		repo:         arquivos,
		solicitacoes: solicitacoes,
		storage:      storage,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			aprovadores: aprovadores,
		},
	}
}

func (u *ArquivoUsecaseImpl) Enviar(
	ctx context.Context,
	proprietarioID uuid.UUID,
	nome, contentType string,
	tamanho int64,
	conteudo io.Reader,
) (*domainsolicitacao.Arquivo, error) {
	if proprietarioID == uuid.Nil {
		return nil, domain.ErroValidacao("proprietário do arquivo não informado")
	}
	if conteudo == nil {
		return nil, domain.ErroValidacao("conteúdo do arquivo não informado")
	}
	if tamanho <= 0 {
		return nil, domain.ErroValidacao("arquivo sem conteúdo")
	}
	if tamanho > portsin.TamanhoMaximoArquivo {
		return nil, domain.ErroValidacao("arquivo deve ter no máximo 10MB")
	}

	nomeLimpo, err := normalizarNomeArquivo(nome)
	if err != nil {
		return nil, err
	}

	tipo, err := normalizarContentType(contentType)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	chave := fmt.Sprintf("%s/%s/%s", proprietarioID, id, nomeLimpo)

	if err := u.storage.Enviar(ctx, chave, conteudo, tipo); err != nil {
		return nil, err
	}

	arquivo := &domainsolicitacao.Arquivo{
		ID:             id,
		ProprietarioID: proprietarioID,
		Nome:           nomeLimpo,
		Chave:          chave,
		ContentType:    tipo,
		Tamanho:        tamanho,
		CriadoEm:       time.Now().UTC(),
	}

	criadoID, err := u.repo.Criar(ctx, arquivo)
	if err != nil {
		u.descartarArquivo(context.WithoutCancel(ctx), chave)
		return nil, err
	}
	arquivo.ID = criadoID

	return arquivo, nil
}

func (u *ArquivoUsecaseImpl) Baixar(ctx context.Context, id, usuarioID uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	arquivo, err := u.buscar(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if arquivo.ProprietarioID != usuarioID {
		podeVer, err := u.podeVisualizar(ctx, arquivo, usuarioID)
		if err != nil {
			return nil, nil, err
		}
		if !podeVer {
			return nil, nil, domain.ErroPermissao("perfil sem permissão para baixar este arquivo")
		}
	}

	conteudo, err := u.storage.Baixar(ctx, arquivo.Chave)
	if err != nil {
		return nil, nil, err
	}

	return arquivo, conteudo, nil
}

func (u *ArquivoUsecaseImpl) ListarPorProprietario(ctx context.Context, proprietarioID uuid.UUID, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	if proprietarioID == uuid.Nil {
		return nil, domain.ErroValidacao("proprietário dos arquivos não informado")
	}

	return u.repo.ListarPorProprietario(ctx, proprietarioID, filtro.Normalizada())
}

func (u *ArquivoUsecaseImpl) Remover(ctx context.Context, id, usuarioID uuid.UUID) error {
	arquivo, err := u.buscar(ctx, id)
	if err != nil {
		return err
	}

	if arquivo.ProprietarioID != usuarioID {
		administrador, err := u.ehAdministrador(ctx, usuarioID)
		if err != nil {
			return err
		}
		if !administrador {
			return domain.ErroPermissao("perfil sem permissão para remover este arquivo")
		}
	}

	vinculado, err := u.repo.VinculadoASolicitacao(ctx, id)
	if err != nil {
		return err
	}
	if vinculado {
		return domain.ErroConflito("arquivo vinculado a uma solicitação não pode ser removido")
	}

	if err := u.repo.Remover(ctx, id); err != nil {
		return err
	}

	return u.storage.Remover(context.WithoutCancel(ctx), arquivo.Chave)
}

func (u *ArquivoUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do arquivo não informado")
	}

	arquivo, err := u.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if arquivo == nil {
		return nil, domain.ErrNotFound
	}

	return arquivo, nil
}

func (u *ArquivoUsecaseImpl) podeVisualizar(ctx context.Context, arquivo *domainsolicitacao.Arquivo, usuarioID uuid.UUID) (bool, error) {
	administrador, err := u.ehAdministrador(ctx, usuarioID)
	if err != nil {
		return false, err
	}
	if administrador {
		return true, nil
	}

	solicitacaoID, err := u.repo.SolicitacaoDoArquivo(ctx, arquivo.ID)
	if err != nil {
		return false, err
	}
	if solicitacaoID == nil {
		return false, nil
	}

	solicitacao, err := u.solicitacoes.Obter(ctx, *solicitacaoID)
	if err != nil {
		return false, err
	}
	if solicitacao == nil {
		return false, nil
	}

	return podeVerSolicitacao(ctx, &u.permissoes, solicitacao, usuarioID)
}

func (u *ArquivoUsecaseImpl) descartarArquivo(ctx context.Context, chave string) {
	_ = u.storage.Remover(ctx, chave)
}

func normalizarNomeArquivo(nome string) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", domain.ErroValidacao("nome do arquivo não informado")
	}
	if strings.ContainsRune(nome, 0) {
		return "", domain.ErroValidacao("nome do arquivo inválido")
	}

	ultimo := strings.LastIndexAny(nome, `/\`) + 1
	nome = nome[ultimo:]

	if nome == "" || nome == "." || nome == ".." {
		return "", domain.ErroValidacao("nome do arquivo inválido")
	}
	if utf8.RuneCountInString(nome) > tamanhoMaximoNomeArquivo {
		return "", domain.ErroValidacao("nome do arquivo deve ter no máximo 255 caracteres")
	}

	return nome, nil
}

func normalizarContentType(contentType string) (string, error) {
	tipo := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	switch tipo {
	case "application/pdf", "image/png", "image/jpeg", "image/webp":
		return tipo, nil
	case "":
		return "", domain.ErroValidacao("tipo do arquivo não informado")
	default:
		return "", domain.ErroValidacao("tipo de arquivo não permitido: use PDF, PNG, JPEG ou WebP")
	}
}
