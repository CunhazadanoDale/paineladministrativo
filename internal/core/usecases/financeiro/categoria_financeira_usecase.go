package financeiro

import (
	"context"
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/google/uuid"
)

var _ portsin.CategoriaFinanceiraUseCase = (*CategoriaFinanceiraUsecaseImpl)(nil)

type CategoriaFinanceiraUsecaseImpl struct {
	categorias portsout.CategoriaFinanceiraRepository
	permissoes
}

func NewCategoriaFinanceiraUsecase(
	categorias portsout.CategoriaFinanceiraRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *CategoriaFinanceiraUsecaseImpl {
	return &CategoriaFinanceiraUsecaseImpl{
		categorias: categorias,
		permissoes: permissoes{usuarios: usuarios, cargos: cargos},
	}
}

func (u *CategoriaFinanceiraUsecaseImpl) Criar(ctx context.Context, input portsin.CriarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	classificacao, pai, err := u.classificacaoOuPai(ctx, input)
	if err != nil {
		return nil, err
	}

	categoria, err := domainfinanceiro.NovaCategoriaFinanceira(dadosDaCategoria(input.Dados), classificacao, pai)
	if err != nil {
		return nil, err
	}

	if err := u.categorias.Criar(ctx, categoria); err != nil {
		return nil, err
	}

	return categoria, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Obter(ctx context.Context, categoriaID uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return nil, err
	}
	if categoria == nil {
		return nil, domainfinanceiro.ErroCategoriaFinanceiraNaoEncontrada()
	}

	return categoria, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Listar(ctx context.Context, input portsin.ListarCategoriasFinanceirasInput) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	filtro := portsout.CategoriaFinanceiraFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Pai:             input.Pai,
		SomenteRaiz:     input.SomenteRaiz,
		Ativa:           input.Ativa,
		Busca:           input.Busca,
	}
	if input.Natureza != "" {
		natureza, err := domainfinanceiro.NovaNatureza(input.Natureza)
		if err != nil {
			return nil, err
		}
		filtro.Natureza = &natureza
	}
	if input.LinhaDRE != "" {
		linha, err := domainfinanceiro.NovaLinhaDRE(input.LinhaDRE)
		if err != nil {
			return nil, err
		}
		filtro.LinhaDRE = &linha
	}

	return u.categorias.Listar(ctx, filtro)
}

func (u *CategoriaFinanceiraUsecaseImpl) Plano(ctx context.Context, somenteAtivas bool) ([]domainfinanceiro.RamoDoPlano, error) {
	categorias, err := u.categorias.ListarPlano(ctx, somenteAtivas)
	if err != nil {
		return nil, err
	}

	return domainfinanceiro.MontarPlano(categorias), nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, err := u.buscarParaAlterar(ctx, input.CategoriaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if err := categoria.Alterar(dadosDaCategoria(input.Dados), time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.categorias.Atualizar(ctx, categoria); err != nil {
		return nil, err
	}

	return categoria, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Ativar(ctx context.Context, input portsin.MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, err := u.buscarParaAlterar(ctx, input.CategoriaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	var pai *domainfinanceiro.CategoriaFinanceira
	if categoria.EhSubcategoria() && !categoria.Ativa {
		if pai, err = u.Obter(ctx, *categoria.CategoriaPaiID); err != nil {
			return nil, err
		}
	}

	mudou, err := categoria.Ativar(pai, time.Now().UTC())
	if err != nil || !mudou {
		return categoriaOuNada(categoria, err)
	}
	if err := u.categorias.Ativar(ctx, categoria); err != nil {
		return nil, err
	}

	return categoria, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Desativar(ctx context.Context, input portsin.MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	categoria, err := u.buscarParaAlterar(ctx, input.CategoriaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	possuiSubcategoriasAtivas := false
	if !categoria.EhSubcategoria() && categoria.Ativa {
		if possuiSubcategoriasAtivas, err = u.categorias.PossuiSubcategoriasAtivas(ctx, categoria.ID); err != nil {
			return nil, err
		}
	}

	mudou, err := categoria.Desativar(possuiSubcategoriasAtivas, time.Now().UTC())
	if err != nil || !mudou {
		return categoriaOuNada(categoria, err)
	}
	if err := u.categorias.Desativar(ctx, categoria); err != nil {
		return nil, err
	}

	return categoria, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) Remover(ctx context.Context, categoriaID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	categoria, err := u.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}

	return u.categorias.Remover(ctx, categoria.ID)
}

func (u *CategoriaFinanceiraUsecaseImpl) classificacaoOuPai(
	ctx context.Context,
	input portsin.CriarCategoriaFinanceiraInput,
) (*domainfinanceiro.Classificacao, *domainfinanceiro.CategoriaFinanceira, error) {
	informouClassificacao := input.Natureza != nil || input.LinhaDRE != nil

	if input.CategoriaPaiID != nil {
		if informouClassificacao {
			return nil, nil, domainfinanceiro.ErroSubcategoriaHerdaClassificacao()
		}

		pai, err := u.categorias.Obter(ctx, *input.CategoriaPaiID)
		if err != nil {
			return nil, nil, err
		}
		if pai == nil {
			return nil, nil, domainfinanceiro.ErroPaiDeveSerRaizAtiva()
		}

		return nil, pai, nil
	}

	if !informouClassificacao {
		return nil, nil, nil
	}

	classificacao, err := domainfinanceiro.NovaClassificacao(textoOuVazio(input.Natureza), textoOuVazio(input.LinhaDRE))
	if err != nil {
		return nil, nil, err
	}

	return &classificacao, nil, nil
}

func (u *CategoriaFinanceiraUsecaseImpl) buscarParaAlterar(ctx context.Context, categoriaID, usuarioID uuid.UUID, versao int) (*domainfinanceiro.CategoriaFinanceira, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	categoria, err := u.Obter(ctx, categoriaID)
	if err != nil {
		return nil, err
	}
	if err := categoria.ConferirVersao(versao); err != nil {
		return nil, err
	}

	return categoria, nil
}

func dadosDaCategoria(dados portsin.DadosCategoriaFinanceiraInput) domainfinanceiro.DadosCategoriaFinanceira {
	return domainfinanceiro.DadosCategoriaFinanceira{
		Nome:   dados.Nome,
		Codigo: dados.Codigo,
		Ordem:  dados.Ordem,
	}
}

func categoriaOuNada(categoria *domainfinanceiro.CategoriaFinanceira, err error) (*domainfinanceiro.CategoriaFinanceira, error) {
	if err != nil {
		return nil, err
	}

	return categoria, nil
}

func textoOuVazio(valor *string) string {
	if valor == nil {
		return ""
	}

	return *valor
}
