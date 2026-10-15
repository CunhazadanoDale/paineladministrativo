package solicitacao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

var _ portsin.DespesaRecorrenteUseCase = (*DespesaRecorrenteUsecaseImpl)(nil)

type DespesaRecorrenteUsecaseImpl struct {
	recorrencias portsout.DespesaRecorrenteRepository
	solicitacoes portsout.SolicitacaoRepository
	fornecedores portsout.FornecedorRepository
	categorias   portsout.CategoriaFinanceiraRepository
	permissoes
}

func NewDespesaRecorrenteUsecase(
	recorrencias portsout.DespesaRecorrenteRepository,
	solicitacoes portsout.SolicitacaoRepository,
	fornecedores portsout.FornecedorRepository,
	categorias portsout.CategoriaFinanceiraRepository,
	aprovadores portsout.AprovadorRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *DespesaRecorrenteUsecaseImpl {
	return &DespesaRecorrenteUsecaseImpl{
		recorrencias: recorrencias,
		solicitacoes: solicitacoes,
		fornecedores: fornecedores,
		categorias:   categorias,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			aprovadores: aprovadores,
			empresas:    empresas,
		},
	}
}

func (u *DespesaRecorrenteUsecaseImpl) Criar(ctx context.Context, input portsin.CriarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	if err := u.exigeFinanceiroNaRecorrencia(ctx, input.UsuarioID); err != nil {
		return nil, err
	}
	if err := u.validarEmpresaDaRecorrencia(ctx, input.UsuarioID, input.EmpresaID); err != nil {
		return nil, err
	}

	dados, err := u.dadosDaRecorrencia(ctx, input.Dados)
	if err != nil {
		return nil, err
	}

	despesa, err := domainsolicitacao.NovaDespesaRecorrente(input.EmpresaID, dados)
	if err != nil {
		return nil, err
	}
	if err := u.recorrencias.Criar(ctx, despesa); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (u *DespesaRecorrenteUsecaseImpl) Obter(ctx context.Context, despesaID, usuarioID uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error) {
	if err := u.exigeFinanceiroNaRecorrencia(ctx, usuarioID); err != nil {
		return nil, err
	}

	return u.buscarComAcesso(ctx, despesaID, usuarioID)
}

func (u *DespesaRecorrenteUsecaseImpl) Listar(ctx context.Context, input portsin.ListarDespesasRecorrentesInput) ([]*domainsolicitacao.DespesaRecorrente, error) {
	if err := u.exigeFinanceiroNaRecorrencia(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.recorrencias.Listar(ctx, portsout.DespesaRecorrenteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EmpresaIDs:      empresaIDs,
		Ativa:           input.Ativa,
		Fornecedor:      input.Fornecedor,
	})
}

func (u *DespesaRecorrenteUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	despesa, err := u.buscarParaAlterar(ctx, input.DespesaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	dados, err := u.dadosDaRecorrencia(ctx, input.Dados)
	if err != nil {
		return nil, err
	}
	if err := despesa.Alterar(dados, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.recorrencias.Atualizar(ctx, despesa); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (u *DespesaRecorrenteUsecaseImpl) Ativar(ctx context.Context, input portsin.MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	despesa, err := u.buscarParaAlterar(ctx, input.DespesaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if !despesa.Ativar(time.Now().UTC()) {
		return despesa, nil
	}
	if err := u.recorrencias.Ativar(ctx, despesa); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (u *DespesaRecorrenteUsecaseImpl) Desativar(ctx context.Context, input portsin.MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	despesa, err := u.buscarParaAlterar(ctx, input.DespesaID, input.UsuarioID, input.Versao)
	if err != nil {
		return nil, err
	}

	if !despesa.Desativar(time.Now().UTC()) {
		return despesa, nil
	}
	if err := u.recorrencias.Desativar(ctx, despesa); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (u *DespesaRecorrenteUsecaseImpl) Remover(ctx context.Context, despesaID, usuarioID uuid.UUID) error {
	administrador, err := u.ehAdministrador(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !administrador {
		return domain.ErroPermissao("perfil sem permissão para excluir despesa recorrente")
	}

	if _, err := u.buscarComAcesso(ctx, despesaID, usuarioID); err != nil {
		return err
	}

	return u.recorrencias.Remover(ctx, despesaID)
}

func (u *DespesaRecorrenteUsecaseImpl) Gerar(ctx context.Context, input portsin.GerarDespesasRecorrentesInput) (portsin.ResultadoGeracao, error) {
	if err := u.exigeFinanceiroNaRecorrencia(ctx, input.UsuarioID); err != nil {
		return portsin.ResultadoGeracao{}, err
	}
	if strings.TrimSpace(input.Competencia) == "" {
		return portsin.ResultadoGeracao{}, domain.ErroValidacao("competência da geração é obrigatória")
	}
	competencia, err := domainsolicitacao.NovaCompetencia(input.Competencia)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}

	despesas, err := u.recorrencias.ListarVigentes(ctx, competencia, empresaIDs)
	if err != nil {
		return portsin.ResultadoGeracao{}, err
	}

	resultado := portsin.ResultadoGeracao{
		Competencia: competencia.String(),
		Geradas:     []uuid.UUID{},
		Recusadas:   []portsin.RecorrenciaRecusada{},
	}
	for _, despesa := range despesas {
		id, criada, err := u.gerarUma(ctx, despesa, input.UsuarioID, competencia)
		switch {
		case errors.Is(err, domain.ErrValidacao):
			resultado.Recusadas = append(resultado.Recusadas, portsin.RecorrenciaRecusada{
				DespesaRecorrenteID: despesa.ID,
				Motivo:              strings.TrimPrefix(err.Error(), domain.ErrValidacao.Error()+": "),
			})
		case err != nil:
			return portsin.ResultadoGeracao{}, err
		case criada:
			resultado.Geradas = append(resultado.Geradas, id)
		default:
			resultado.JaExistentes++
		}
	}

	return resultado, nil
}

func (u *DespesaRecorrenteUsecaseImpl) gerarUma(
	ctx context.Context,
	despesa *domainsolicitacao.DespesaRecorrente,
	usuarioID uuid.UUID,
	competencia domainsolicitacao.Competencia,
) (uuid.UUID, bool, error) {
	jaGeradas, err := u.solicitacoes.Contar(ctx, portsout.SolicitacaoFiltro{
		Origem:      domainsolicitacao.OrigemRecorrencia,
		OrigemID:    &despesa.ID,
		Competencia: &competencia,
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	if jaGeradas > 0 {
		return uuid.Nil, false, nil
	}

	solicitacao, err := despesa.GerarSolicitacao(usuarioID, competencia, time.Now().UTC())
	if err != nil {
		return uuid.Nil, false, err
	}

	classificacao := domainsolicitacao.Classificacao{
		FornecedorID:          solicitacao.FornecedorID,
		CategoriaFinanceiraID: solicitacao.CategoriaFinanceiraID,
		Competencia:           &competencia,
	}
	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, classificacao); err != nil {
		return uuid.Nil, false, err
	}

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, usuarioID, nil, solicitacao.Status, domainsolicitacao.EventoCriada,
		"solicitação gerada pela despesa recorrente na competência "+competencia.String(),
	)
	criada, err := u.solicitacoes.GerarDeRecorrencia(ctx, solicitacao, historico)
	if err != nil {
		return uuid.Nil, false, err
	}

	return solicitacao.ID, criada, nil
}

func (u *DespesaRecorrenteUsecaseImpl) dadosDaRecorrencia(ctx context.Context, input portsin.DadosDespesaRecorrenteInput) (domainsolicitacao.DadosDespesaRecorrente, error) {
	descricao, err := domainsolicitacao.NovaObservacao(input.Descricao)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}
	valor, err := domainsolicitacao.NovoValor(input.ValorCentavos)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}
	forma, err := domainsolicitacao.NovaFormaPagamento(input.FormaPagamento)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}
	dia, err := domainsolicitacao.NovoDiaVencimento(input.DiaVencimento)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}
	if strings.TrimSpace(input.Inicio) == "" {
		return domainsolicitacao.DadosDespesaRecorrente{}, domain.ErroValidacao("início da despesa recorrente é obrigatório")
	}
	inicio, err := domainsolicitacao.NovaCompetencia(input.Inicio)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}
	fim, err := competenciaInformada(input.Fim)
	if err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}

	categoriaID := input.CategoriaFinanceiraID
	if categoriaID == uuid.Nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, domain.ErroValidacao("categoria financeira é obrigatória na despesa recorrente")
	}
	if err := validarClassificacao(ctx, u.fornecedores, u.categorias, domainsolicitacao.Classificacao{
		FornecedorID:          input.FornecedorID,
		CategoriaFinanceiraID: &categoriaID,
	}); err != nil {
		return domainsolicitacao.DadosDespesaRecorrente{}, err
	}

	return domainsolicitacao.DadosDespesaRecorrente{
		FornecedorID:          input.FornecedorID,
		CategoriaFinanceiraID: categoriaID,
		Descricao:             descricao,
		Valor:                 valor,
		Estimado:              input.Estimado,
		FormaPagamento:        forma,
		DiaVencimento:         dia,
		VencimentoMesSeguinte: input.VencimentoMesSeguinte,
		Inicio:                inicio,
		Fim:                   fim,
	}, nil
}

func (u *DespesaRecorrenteUsecaseImpl) exigeFinanceiroNaRecorrencia(ctx context.Context, usuarioID uuid.UUID) error {
	financeiro, err := u.ehFinanceiro(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !financeiro {
		return domain.ErroPermissao("perfil sem permissão para despesas recorrentes")
	}

	return nil
}

func (u *DespesaRecorrenteUsecaseImpl) validarEmpresaDaRecorrencia(ctx context.Context, usuarioID, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa da despesa recorrente não informada")
	}
	if err := u.exigeAcessoAEmpresa(ctx, usuarioID, empresaID); err != nil {
		return err
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainsolicitacao.ErroEmpresaDaRecorrenciaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainsolicitacao.ErroEmpresaInativaNaRecorrencia()
	}

	return nil
}

func (u *DespesaRecorrenteUsecaseImpl) empresasVisiveis(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) ([]uuid.UUID, error) {
	acesso, err := u.empresas.AcessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if empresa == nil {
		return acesso.EmpresaIDs(), nil
	}
	if !acesso.Permite(*empresa) {
		return nil, domain.ErroPermissao("perfil sem acesso a esta empresa")
	}

	return []uuid.UUID{*empresa}, nil
}

func (u *DespesaRecorrenteUsecaseImpl) buscarParaAlterar(ctx context.Context, despesaID, usuarioID uuid.UUID, versao int) (*domainsolicitacao.DespesaRecorrente, error) {
	if err := u.exigeFinanceiroNaRecorrencia(ctx, usuarioID); err != nil {
		return nil, err
	}

	despesa, err := u.buscarComAcesso(ctx, despesaID, usuarioID)
	if err != nil {
		return nil, err
	}
	if err := despesa.ConferirVersao(versao); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (u *DespesaRecorrenteUsecaseImpl) buscarComAcesso(ctx context.Context, despesaID, usuarioID uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error) {
	despesa, err := u.recorrencias.Obter(ctx, despesaID)
	if err != nil {
		return nil, err
	}
	if despesa == nil {
		return nil, domainsolicitacao.ErroDespesaRecorrenteNaoEncontrada()
	}
	if err := u.exigeAcessoAEmpresa(ctx, usuarioID, despesa.EmpresaID); err != nil {
		return nil, err
	}

	return despesa, nil
}
