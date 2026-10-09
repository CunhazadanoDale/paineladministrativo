package estoque

import (
	"context"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	portsoutsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var _ portsin.ImagemUseCase = (*ImagemUsecaseImpl)(nil)

const prefixoContentTypeImagem = "image/"

type ImagemUsecaseImpl struct {
	imagens  portsout.ImagemRepository
	produtos portsout.ProdutoRepository
	arquivos portsoutsolicitacao.ArquivoRepository
	permissoes
}

func NewImagemUsecase(
	imagens portsout.ImagemRepository,
	produtos portsout.ProdutoRepository,
	arquivos portsoutsolicitacao.ArquivoRepository,
	usuarios portsoutusuarios.UsuarioRepository,
	cargos portsoutusuarios.CargoRepository,
) *ImagemUsecaseImpl {
	return &ImagemUsecaseImpl{
		imagens:  imagens,
		produtos: produtos,
		arquivos: arquivos,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
		},
	}
}

func (u *ImagemUsecaseImpl) Anexar(ctx context.Context, input portsin.AnexarImagemInput) (*domainestoque.Imagem, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	produto, err := u.produtos.Obter(ctx, input.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, domain.ErroNaoEncontrado("produto não encontrado")
	}

	arquivo, err := u.arquivos.Obter(ctx, input.ArquivoID)
	if err != nil {
		return nil, err
	}
	if arquivo == nil {
		return nil, domain.ErroNaoEncontrado("arquivo não encontrado")
	}
	if !strings.HasPrefix(arquivo.ContentType, prefixoContentTypeImagem) {
		return nil, domain.ErroValidacao("arquivo deve ser uma imagem")
	}

	anexadas, err := u.imagens.ListarPorProduto(ctx, produto.ID)
	if err != nil {
		return nil, err
	}
	for _, anexada := range anexadas {
		if anexada.ArquivoID == arquivo.ID {
			return nil, domain.ErroConflito("arquivo já anexado a este produto")
		}
	}

	imagem, err := domainestoque.NovaImagem(domainestoque.ImagemInput{
		ProdutoID: produto.ID,
		ArquivoID: arquivo.ID,
		Ordem:     input.Ordem,
		Alt:       input.Alt,
	})
	if err != nil {
		return nil, err
	}

	if _, err := u.imagens.Criar(ctx, imagem); err != nil {
		return nil, err
	}

	return imagem, nil
}

func (u *ImagemUsecaseImpl) Remover(ctx context.Context, usuarioID, produtoID, imagemID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	imagem, err := u.imagens.Obter(ctx, imagemID)
	if err != nil {
		return err
	}
	if imagem == nil || imagem.ProdutoID != produtoID {
		return domain.ErroNaoEncontrado("imagem não encontrada")
	}

	_, err = u.imagens.Remover(ctx, imagem.ID)

	return err
}

func (u *ImagemUsecaseImpl) ObterPublica(ctx context.Context, imagemID uuid.UUID) (*domainestoque.Imagem, error) {
	imagem, err := u.imagens.Obter(ctx, imagemID)
	if err != nil {
		return nil, err
	}
	if imagem == nil {
		return nil, domain.ErroNaoEncontrado("imagem não encontrada")
	}

	produto, err := u.produtos.Obter(ctx, imagem.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil || !produto.Ativo {
		return nil, domain.ErroNaoEncontrado("imagem não encontrada")
	}

	return imagem, nil
}
