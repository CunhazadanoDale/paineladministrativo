package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
)

var _ portsin.ResumoUseCase = (*ResumoUsecaseImpl)(nil)

const quantidadeMovimentosResumo = 5

type ResumoUsecaseImpl struct {
	produtos   portsout.ProdutoRepository
	movimentos portsout.MovimentoRepository
	permissoes
}

func NewResumoUsecase(
	produtos portsout.ProdutoRepository,
	movimentos portsout.MovimentoRepository,
	empresas portsout.EmpresaRepository,
) *ResumoUsecaseImpl {
	return &ResumoUsecaseImpl{
		produtos:   produtos,
		movimentos: movimentos,
		permissoes: permissoes{empresas: empresas},
	}
}

func (u *ResumoUsecaseImpl) Resumo(ctx context.Context, input portsin.ResumoInput) (*domainestoque.ResumoEstoque, error) {
	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	produtos, err := u.produtos.Resumo(ctx)
	if err != nil {
		return nil, err
	}

	movimentos, err := u.movimentos.ListarResumo(ctx, quantidadeMovimentosResumo, empresaIDs)
	if err != nil {
		return nil, err
	}

	resumo := domainestoque.ResumoDe(produtos, movimentos)

	return &resumo, nil
}
