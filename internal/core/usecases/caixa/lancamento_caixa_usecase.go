package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.LancamentoCaixaUseCase = (*LancamentoCaixaUsecaseImpl)(nil)

type LancamentoCaixaUsecaseImpl struct {
	lancamentos  portsout.LancamentoCaixaRepository
	conciliacoes portsout.ConciliacaoRepository
	categorias   portsout.CategoriaFinanceiraRepository
	permissoes
	escopoPorEmpresa
}

func NewLancamentoCaixaUsecase(
	lancamentos portsout.LancamentoCaixaRepository,
	conciliacoes portsout.ConciliacaoRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *LancamentoCaixaUsecaseImpl {
	return &LancamentoCaixaUsecaseImpl{
		lancamentos:      lancamentos,
		conciliacoes:     conciliacoes,
		categorias:       categorias,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

func (u *LancamentoCaixaUsecaseImpl) Classificar(ctx context.Context, input portsin.ClassificarInput) (*domaincaixa.Classificacao, error) {
	linha, conta, err := u.linhaComConta(ctx, input.LancamentoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	categoria, err := u.categoria(ctx, input.CategoriaFinanceiraID)
	if err != nil {
		return nil, err
	}

	classificacao, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: linha,
		EmpresaID:  conta.EmpresaID,
		Categoria:  categoria,
		Descricao:  input.Descricao,
		UsuarioID:  input.UsuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := u.lancamentos.Classificar(ctx, classificacao); err != nil {
		return nil, err
	}

	return classificacao, nil
}

func (u *LancamentoCaixaUsecaseImpl) Transferir(ctx context.Context, input portsin.TransferirInput) (*domaincaixa.TransferenciaRegistrada, error) {
	linha, conta, err := u.linhaComConta(ctx, input.LancamentoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	contraparte, err := u.contraparte(ctx, input.ContraparteID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	categoria, err := u.categoria(ctx, input.CategoriaFinanceiraID)
	if err != nil {
		return nil, err
	}

	registrada, err := domaincaixa.NovaTransferencia(domaincaixa.DadosTransferencia{
		Lancamento:  linha,
		Conta:       conta,
		Contraparte: contraparte,
		Categoria:   categoria,
		Descricao:   input.Descricao,
		UsuarioID:   input.UsuarioID,
		Agora:       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := u.lancamentos.Transferir(ctx, registrada); err != nil {
		return nil, err
	}

	return registrada, nil
}

func (u *LancamentoCaixaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarCaixaInput) ([]*domaincaixa.LancamentoCaixa, error) {
	filtro, err := u.filtroDaListagem(ctx, input)
	if err != nil {
		return nil, err
	}

	return u.lancamentos.Listar(ctx, filtro)
}

func (u *LancamentoCaixaUsecaseImpl) ListarTransferencias(ctx context.Context, input portsin.ListarCaixaInput) ([]*domaincaixa.Transferencia, error) {
	filtro, err := u.filtroDaListagem(ctx, input)
	if err != nil {
		return nil, err
	}

	return u.lancamentos.ListarTransferencias(ctx, filtro)
}

func (u *LancamentoCaixaUsecaseImpl) linhaComConta(
	ctx context.Context,
	lancamentoID, usuarioID uuid.UUID,
) (*domaincaixa.LancamentoExtrato, *domainfinanceiro.ContaBancaria, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, nil, err
	}

	linha, err := u.conciliacoes.ObterLancamento(ctx, lancamentoID)
	if err != nil {
		return nil, nil, err
	}
	if linha == nil {
		return nil, nil, domaincaixa.ErroLancamentoNaoEncontrado()
	}

	conta, err := u.contaComAcesso(ctx, linha.ContaBancariaID, usuarioID)
	if err != nil {
		return nil, nil, err
	}

	return linha, conta, nil
}

func (u *LancamentoCaixaUsecaseImpl) contraparte(ctx context.Context, contaID, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	if contaID == uuid.Nil {
		return nil, domain.ErroValidacao("conta da outra ponta é obrigatória")
	}

	conta, err := u.contas.Obter(ctx, contaID)
	if err != nil {
		return nil, err
	}
	if conta == nil {
		return nil, domain.ErroValidacao("conta da outra ponta não encontrada")
	}
	if err := u.exigeAcesso(ctx, usuarioID, conta.EmpresaID); err != nil {
		return nil, err
	}

	return conta, nil
}

func (u *LancamentoCaixaUsecaseImpl) categoria(ctx context.Context, categoriaID uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	if categoriaID == uuid.Nil {
		return nil, domain.ErroValidacao("categoria financeira é obrigatória")
	}

	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return nil, err
	}
	if categoria == nil {
		return nil, domain.ErroValidacao("categoria financeira não encontrada")
	}

	return categoria, nil
}

func (u *LancamentoCaixaUsecaseImpl) filtroDaListagem(ctx context.Context, input portsin.ListarCaixaInput) (portsout.LancamentoCaixaFiltro, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return portsout.LancamentoCaixaFiltro{}, err
	}

	filtro := portsout.LancamentoCaixaFiltro{}
	if input.ContaBancariaID != nil {
		conta, err := u.contaComAcesso(ctx, *input.ContaBancariaID, input.UsuarioID)
		if err != nil {
			return portsout.LancamentoCaixaFiltro{}, err
		}
		filtro.ContaBancariaID = &conta.ID
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return portsout.LancamentoCaixaFiltro{}, err
	}
	filtro.EmpresaIDs = empresaIDs

	if filtro.De, filtro.Ate, err = intervaloDeDatas(input.De, input.Ate); err != nil {
		return portsout.LancamentoCaixaFiltro{}, err
	}

	paginacao := input.Filtro.Normalizada()
	filtro.Limite = paginacao.Size
	filtro.Deslocamento = (paginacao.Page - 1) * paginacao.Size

	return filtro, nil
}
