package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
)

var _ portsin.MovimentoUseCase = (*MovimentoUsecaseImpl)(nil)

type MovimentoUsecaseImpl struct {
	produtos   portsout.ProdutoRepository
	movimentos portsout.MovimentoRepository
	permissoes
}

func NewMovimentoUsecase(
	produtos portsout.ProdutoRepository,
	movimentos portsout.MovimentoRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *MovimentoUsecaseImpl {
	return &MovimentoUsecaseImpl{
		produtos:   produtos,
		movimentos: movimentos,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
		},
	}
}

func (u *MovimentoUsecaseImpl) Movimentar(ctx context.Context, input portsin.MovimentarEstoqueInput) (*domainestoque.Movimento, error) {
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

	tipo, err := domainestoque.NovoTipoMovimento(input.Tipo)
	if err != nil {
		return nil, err
	}

	quantidade, err := domainestoque.NovaQuantidade(input.Quantidade)
	if err != nil {
		return nil, err
	}

	movimento, err := produto.Movimentar(tipo, quantidade, input.UsuarioID, input.DocumentoRef, input.Observacao, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := u.produtos.Movimentar(ctx, produto, movimento); err != nil {
		return nil, err
	}

	return movimento, nil
}

func (u *MovimentoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarMovimentosInput) ([]*domainestoque.Movimento, error) {
	filtro := portsout.MovimentoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ProdutoID:       input.ProdutoID,
	}

	if input.Tipo != "" {
		tipo, err := domainestoque.NovoTipoMovimento(input.Tipo)
		if err != nil {
			return nil, err
		}
		filtro.Tipo = tipo
	}

	return u.movimentos.Listar(ctx, filtro)
}
