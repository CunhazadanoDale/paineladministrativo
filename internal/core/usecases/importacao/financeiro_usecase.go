package importacao

import (
	"context"
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var _ portsin.FinanceiroUseCase = (*FinanceiroUsecaseImpl)(nil)

type FinanceiroUsecaseImpl struct {
	fornecedores portsout.FornecedorRepository
	categorias   portsout.CategoriaFinanceiraRepository
	permissoes
}

type operacaoFinanceira struct {
	acao     domainimportacao.Acao
	detalhe  string
	gravacao portsout.Gravacao
}

func NewFinanceiroUsecase(
	importacoes portsout.ImportacaoRepository,
	fornecedores portsout.FornecedorRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *FinanceiroUsecaseImpl {
	return &FinanceiroUsecaseImpl{
		fornecedores: fornecedores,
		categorias:   categorias,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			empresas:    empresas,
			importacoes: importacoes,
		},
	}
}

func (u *FinanceiroUsecaseImpl) IncluirParcela(ctx context.Context, input portsin.IncluirParcelaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			parcela, err := importacao.IncluirParcela(input.UsuarioID, dadosDaParcela(input.Dados), agora)
			if err != nil {
				return nil, err
			}

			return &operacaoFinanceira{
				acao:     domainimportacao.AcaoParcelaIncluida,
				detalhe:  descreverParcela(importacao, parcela),
				gravacao: portsout.Gravacao{ParcelasNovas: []*domainimportacao.Parcela{parcela}},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) AlterarParcela(ctx context.Context, input portsin.AlterarParcelaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			parcela, err := importacao.AlterarParcela(input.ParcelaID, dadosDaParcela(input.Dados), agora)
			if err != nil {
				return nil, err
			}

			return &operacaoFinanceira{
				acao:     domainimportacao.AcaoParcelaAlterada,
				detalhe:  descreverParcela(importacao, parcela),
				gravacao: portsout.Gravacao{ParcelasAlteradas: []*domainimportacao.Parcela{parcela}},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) RemoverParcela(ctx context.Context, input portsin.ParcelaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			parcela, err := importacao.RemoverParcela(input.ParcelaID, agora)
			if err != nil {
				return nil, err
			}

			return &operacaoFinanceira{
				acao:     domainimportacao.AcaoParcelaRemovida,
				detalhe:  descreverParcela(importacao, parcela),
				gravacao: portsout.Gravacao{ParcelasRemovidas: []uuid.UUID{parcela.ID}},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) FecharCambio(ctx context.Context, input portsin.FecharCambioInput) (*portsin.Visao, error) {
	if err := u.exigirCategoriaDeDespesa(ctx, input.CategoriaFinanceiraID); err != nil {
		return nil, err
	}

	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			parcela, err := importacao.FecharCambio(input.ParcelaID, domainimportacao.FechamentoCambioInput{
				CambioMicro: input.CambioMicro,
				Vencimento:  input.Vencimento,
			}, agora)
			if err != nil {
				return nil, err
			}

			fornecedorID := importacao.FornecedorID
			conta, err := contaAPagar(importacao, input.UsuarioID, domainsolicitacao.DadosSolicitacaoDeOrigem{
				FornecedorID:          &fornecedorID,
				CategoriaFinanceiraID: input.CategoriaFinanceiraID,
				Origem:                domainsolicitacao.OrigemImportacaoParcela,
				OrigemID:              parcela.ID,
				ValorCentavos:         *parcela.ValorCentavos,
				Vencimento:            *parcela.Vencimento,
				Observacao:            fmt.Sprintf("importação %d — %s", importacao.Numero, parcela.Descricao),
			}, agora)
			if err != nil {
				return nil, err
			}
			parcela.SolicitacaoID = &conta.Solicitacao.ID
			parcela.StatusContaAPagar = &conta.Solicitacao.Status

			return &operacaoFinanceira{
				acao: domainimportacao.AcaoParcelaCambioFechado,
				detalhe: fmt.Sprintf("%s: câmbio %s, %s", parcela.Descricao, formatarCambio(*parcela.Cambio),
					formatarReais(*parcela.ValorCentavos)),
				gravacao: portsout.Gravacao{
					ParcelasAlteradas: []*domainimportacao.Parcela{parcela},
					ContasAPagar:      []portsout.ContaAPagarGerada{conta},
				},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) ReabrirParcela(ctx context.Context, input portsin.ParcelaInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			antiga, nova, err := importacao.ReabrirParcela(input.ParcelaID, input.UsuarioID, agora)
			if err != nil {
				return nil, err
			}

			return &operacaoFinanceira{
				acao:    domainimportacao.AcaoParcelaReaberta,
				detalhe: antiga.Descricao + ": conta a pagar " + antiga.StatusContaAPagar.String(),
				gravacao: portsout.Gravacao{
					ParcelasAlteradas:        []*domainimportacao.Parcela{antiga},
					ParcelasNovas:            []*domainimportacao.Parcela{nova},
					ContasExigidasEncerradas: []uuid.UUID{*antiga.SolicitacaoID},
				},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) LancarCusto(ctx context.Context, input portsin.LancarCustoInput) (*portsin.Visao, error) {
	tipo, err := domainimportacao.NovoTipoCusto(input.Tipo)
	if err != nil {
		return nil, err
	}
	if err := u.exigirFornecedorAtivo(ctx, input.FornecedorID); err != nil {
		return nil, err
	}
	if err := u.exigirCategoriaDeDespesa(ctx, input.CategoriaFinanceiraID); err != nil {
		return nil, err
	}

	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			custo, err := importacao.LancarCusto(input.UsuarioID, domainimportacao.DadosCusto{
				Tipo:                  tipo,
				Descricao:             input.Descricao,
				FornecedorID:          input.FornecedorID,
				ValorCentavos:         input.ValorCentavos,
				Vencimento:            input.Vencimento,
				CompoeCusto:           input.CompoeCusto,
				CategoriaFinanceiraID: input.CategoriaFinanceiraID,
			}, agora)
			if err != nil {
				return nil, err
			}

			fornecedorID := custo.FornecedorID
			conta, err := contaAPagar(importacao, input.UsuarioID, domainsolicitacao.DadosSolicitacaoDeOrigem{
				FornecedorID:          &fornecedorID,
				CategoriaFinanceiraID: custo.CategoriaFinanceiraID,
				Origem:                domainsolicitacao.OrigemImportacaoCusto,
				OrigemID:              custo.ID,
				ValorCentavos:         custo.ValorCentavos,
				Vencimento:            custo.Vencimento,
				Observacao:            fmt.Sprintf("importação %d — %s", importacao.Numero, custo.Descricao),
			}, agora)
			if err != nil {
				return nil, err
			}
			custo.SolicitacaoID = conta.Solicitacao.ID
			custo.StatusContaAPagar = &conta.Solicitacao.Status

			return &operacaoFinanceira{
				acao:    domainimportacao.AcaoCustoLancado,
				detalhe: descreverCusto(custo),
				gravacao: portsout.Gravacao{
					CustosNovos:  []*domainimportacao.Custo{custo},
					ContasAPagar: []portsout.ContaAPagarGerada{conta},
				},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) CancelarCusto(ctx context.Context, input portsin.CustoInput) (*portsin.Visao, error) {
	return u.operar(ctx, input.UsuarioID, input.ImportacaoID, input.Versao,
		func(importacao *domainimportacao.Importacao, agora time.Time) (*operacaoFinanceira, error) {
			custo, err := importacao.CancelarCusto(input.CustoID, input.UsuarioID, agora)
			if err != nil {
				return nil, err
			}

			return &operacaoFinanceira{
				acao:    domainimportacao.AcaoCustoCancelado,
				detalhe: descreverCusto(custo),
				gravacao: portsout.Gravacao{
					CustosCancelados:         []*domainimportacao.Custo{custo},
					ContasExigidasEncerradas: []uuid.UUID{custo.SolicitacaoID},
				},
			}, nil
		})
}

func (u *FinanceiroUsecaseImpl) CustoPrevisto(
	ctx context.Context, importacaoID, usuarioID uuid.UUID, cambioMicro *int64,
) (*domainimportacao.CalculoCusto, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, importacaoID)
	if err != nil {
		return nil, err
	}

	return importacao.CalcularCusto(cambioMicro, nil)
}

func (u *FinanceiroUsecaseImpl) operar(
	ctx context.Context,
	usuarioID, importacaoID uuid.UUID,
	versao int,
	executar func(*domainimportacao.Importacao, time.Time) (*operacaoFinanceira, error),
) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, importacaoID)
	if err != nil {
		return nil, err
	}
	if err := importacao.ConferirVersao(versao); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	operacao, err := executar(importacao, agora)
	if err != nil {
		return nil, err
	}

	gravacao := operacao.gravacao
	gravacao.Importacao = importacao
	gravacao.VersaoLida = versao
	gravacao.StatusAnterior = importacao.Status
	gravacao.Historicos = []*domainimportacao.Historico{
		domainimportacao.NovoHistorico(importacao.ID, usuarioID, operacao.acao, nil, importacao.Status, operacao.detalhe, agora),
	}
	if err := u.importacoes.Gravar(ctx, &gravacao); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: true}, nil
}

func (u *FinanceiroUsecaseImpl) exigirFornecedorAtivo(ctx context.Context, fornecedorID uuid.UUID) error {
	if fornecedorID == uuid.Nil {
		return domain.ErroValidacao("fornecedor do custo é obrigatório")
	}

	fornecedor, err := u.fornecedores.Obter(ctx, fornecedorID)
	if err != nil {
		return err
	}
	if fornecedor == nil {
		return domain.ErroValidacao("fornecedor do custo não encontrado")
	}
	if !fornecedor.Ativo {
		return domainimportacao.ErroFornecedorInativo(fornecedor.Nome)
	}

	return nil
}

func (u *FinanceiroUsecaseImpl) exigirCategoriaDeDespesa(ctx context.Context, categoriaID uuid.UUID) error {
	if categoriaID == uuid.Nil {
		return domain.ErroValidacao("categoria financeira da conta a pagar é obrigatória")
	}

	categoria, err := u.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domain.ErroValidacao("categoria financeira não encontrada")
	}
	if !categoria.Lancavel() {
		return domain.ErroValidacao("categoria financeira deve ser uma subcategoria ativa")
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa {
		return domain.ErroValidacao("categoria financeira da conta a pagar deve ser de despesa")
	}

	return nil
}

func contaAPagar(
	importacao *domainimportacao.Importacao, usuarioID uuid.UUID, dados domainsolicitacao.DadosSolicitacaoDeOrigem, agora time.Time,
) (portsout.ContaAPagarGerada, error) {
	dados.EmpresaID = importacao.EmpresaID
	dados.SolicitanteID = usuarioID

	solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, agora)
	if err != nil {
		return portsout.ContaAPagarGerada{}, err
	}

	return portsout.ContaAPagarGerada{Solicitacao: solicitacao, Historico: historico}, nil
}

func dadosDaParcela(input portsin.DadosParcelaInput) domainimportacao.DadosParcela {
	return domainimportacao.DadosParcela{
		Descricao:            input.Descricao,
		ValorMoedaCentesimos: input.ValorMoedaCentesimos,
		VencimentoPrevisto:   input.VencimentoPrevisto,
	}
}

func descreverParcela(importacao *domainimportacao.Importacao, parcela *domainimportacao.Parcela) string {
	return fmt.Sprintf("%s: %s %s, vencimento previsto %s", parcela.Descricao, importacao.Moeda,
		formatarCentesimos(parcela.Valor.Centesimos()), parcela.VencimentoPrevisto.Format("02/01/2006"))
}

func descreverCusto(custo *domainimportacao.Custo) string {
	composicao := "fora do custo do produto"
	if custo.CompoeCusto {
		composicao = "compõe o custo do produto"
	}

	return fmt.Sprintf("%s: %s, %s", custo.Descricao, formatarReais(custo.ValorCentavos), composicao)
}

func formatarReais(centavos int64) string {
	return "R$ " + formatarCentesimos(centavos)
}

func formatarCentesimos(valor int64) string {
	return fmt.Sprintf("%d,%02d", valor/100, valor%100)
}

func formatarCambio(cambio domainimportacao.Cambio) string {
	return fmt.Sprintf("%d,%06d", cambio.Micro()/1_000_000, cambio.Micro()%1_000_000)
}
