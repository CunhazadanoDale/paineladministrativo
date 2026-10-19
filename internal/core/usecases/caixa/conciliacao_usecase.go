package caixa

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

const (
	limiteDeMovimentosDaConta = 5000
	maximoDeDiasDaAutomatica  = 92
)

var _ portsin.ConciliacaoUseCase = (*ConciliacaoUsecaseImpl)(nil)

type ConciliacaoUsecaseImpl struct {
	conciliacoes portsout.ConciliacaoRepository
	permissoes
	escopoPorEmpresa
}

func NewConciliacaoUsecase(
	conciliacoes portsout.ConciliacaoRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ConciliacaoUsecaseImpl {
	return &ConciliacaoUsecaseImpl{
		conciliacoes:     conciliacoes,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

func (u *ConciliacaoUsecaseImpl) Sugestoes(ctx context.Context, lancamentoID, usuarioID uuid.UUID) ([]*domaincaixa.MovimentoSistema, error) {
	lancamento, err := u.lancamentoComAcesso(ctx, lancamentoID, usuarioID)
	if err != nil {
		return nil, err
	}

	de := lancamento.Data.AddDate(0, 0, -domaincaixa.JanelaDeConciliacaoDias)
	ate := lancamento.Data.AddDate(0, 0, domaincaixa.JanelaDeConciliacaoDias)
	valor := lancamento.ValorCentavos
	candidatos, err := u.conciliacoes.Movimentos(ctx, portsout.MovimentoFiltro{
		ContaBancariaID: lancamento.ContaBancariaID,
		De:              &de,
		Ate:             &ate,
		ValorCentavos:   &valor,
		SoPendentes:     true,
		Limite:          limiteDeMovimentosDaConta,
	})
	if err != nil {
		return nil, err
	}

	return domaincaixa.OrdenarSugestoes(lancamento, candidatos), nil
}

func (u *ConciliacaoUsecaseImpl) Conciliar(ctx context.Context, input portsin.ConciliarInput) (*domaincaixa.Conciliacao, error) {
	lancamento, err := u.lancamentoComAcesso(ctx, input.LancamentoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	referencias, err := referenciasDosMovimentos(input.Movimentos)
	if err != nil {
		return nil, err
	}

	movimentos, err := u.conciliacoes.MovimentosPorReferencia(ctx, lancamento.ContaBancariaID, referencias)
	if err != nil {
		return nil, err
	}
	if len(movimentos) != len(referencias) {
		return nil, domain.ErroValidacao("movimento não encontrado nesta conta ou anterior ao saldo inicial")
	}

	conciliacao, err := domaincaixa.NovaConciliacao(domaincaixa.DadosConciliacao{
		Lancamento: lancamento,
		Movimentos: movimentos,
		Modo:       domaincaixa.ConciliacaoManual,
		UsuarioID:  input.UsuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := u.conciliacoes.Criar(ctx, conciliacao); err != nil {
		return nil, err
	}

	return conciliacao, nil
}

func (u *ConciliacaoUsecaseImpl) ConciliarAutomatico(ctx context.Context, input portsin.ConciliarAutomaticoInput) (*portsin.ResultadoAutomatico, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, input.ContaBancariaID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	if input.De.IsZero() || input.Ate.IsZero() {
		return nil, domain.ErroValidacao("informe o período (de e ate) da conciliação automática")
	}
	de, ate := domaincaixa.DataDe(input.De), domaincaixa.DataDe(input.Ate)
	if ate.Before(de) {
		return nil, domain.ErroValidacao("data final anterior à inicial")
	}
	if ate.Sub(de) > maximoDeDiasDaAutomatica*24*time.Hour {
		return nil, domain.ErroValidacao("a conciliação automática aceita no máximo 92 dias por vez")
	}

	lancamentos, err := u.conciliacoes.LancamentosPendentes(ctx, conta.ID, de, ate, portsin.LimiteDaConciliacaoAutomatica+1)
	if err != nil {
		return nil, err
	}

	resultado := &portsin.ResultadoAutomatico{}
	if len(lancamentos) > portsin.LimiteDaConciliacaoAutomatica {
		lancamentos = lancamentos[:portsin.LimiteDaConciliacaoAutomatica]
		resultado.RestamLinhas = true
	}

	movimentos, err := u.conciliacoes.Movimentos(ctx, portsout.MovimentoFiltro{
		ContaBancariaID: conta.ID,
		De:              &de,
		Ate:             &ate,
		SoPendentes:     true,
		Limite:          limiteDeMovimentosDaConta,
	})
	if err != nil {
		return nil, err
	}

	pares := domaincaixa.ParesExatos(lancamentos, movimentos)
	resultado.Ambiguos = pares.Ambiguos
	resultado.SemPar = pares.SemPar

	for _, par := range pares.Pares {
		conciliacao, err := domaincaixa.NovaConciliacao(domaincaixa.DadosConciliacao{
			Lancamento: par.Lancamento,
			Movimentos: []*domaincaixa.MovimentoSistema{par.Movimento},
			Modo:       domaincaixa.ConciliacaoAutomatica,
			UsuarioID:  input.UsuarioID,
			Agora:      time.Now().UTC(),
		})
		if err == nil {
			err = u.conciliacoes.Criar(ctx, conciliacao)
		}
		if err != nil {
			if !errors.Is(err, domain.ErrConflito) && !errors.Is(err, domain.ErrValidacao) {
				return nil, err
			}
			resultado.Falhas++
			continue
		}
		resultado.Conciliadas++
	}

	return resultado, nil
}

func (u *ConciliacaoUsecaseImpl) Obter(ctx context.Context, conciliacaoID, usuarioID uuid.UUID) (*domaincaixa.Conciliacao, error) {
	return u.conciliacaoComAcesso(ctx, conciliacaoID, usuarioID)
}

func (u *ConciliacaoUsecaseImpl) Desfazer(ctx context.Context, input portsin.DesfazerConciliacaoInput) (*domaincaixa.Conciliacao, error) {
	conciliacao, err := u.conciliacaoComAcesso(ctx, input.ConciliacaoID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	if err := conciliacao.Desfazer(input.UsuarioID, input.Motivo, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := u.conciliacoes.Desfazer(ctx, conciliacao); err != nil {
		return nil, err
	}

	return conciliacao, nil
}

func (u *ConciliacaoUsecaseImpl) Pendencias(ctx context.Context, input portsin.PendenciasInput) ([]*domaincaixa.MovimentoSistema, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, input.ContaBancariaID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	de, ate, err := intervaloDeDatas(input.De, input.Ate)
	if err != nil {
		return nil, err
	}

	paginacao := input.Filtro.Normalizada()

	return u.conciliacoes.Movimentos(ctx, portsout.MovimentoFiltro{
		ContaBancariaID: conta.ID,
		De:              de,
		Ate:             ate,
		SoPendentes:     true,
		Limite:          paginacao.Size,
		Deslocamento:    (paginacao.Page - 1) * paginacao.Size,
	})
}

func (u *ConciliacaoUsecaseImpl) PagamentosSemConta(ctx context.Context, input portsin.PagamentosSemContaInput) ([]*domaincaixa.MovimentoSistema, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	de, ate, err := intervaloDeDatas(input.De, input.Ate)
	if err != nil {
		return nil, err
	}

	paginacao := input.Filtro.Normalizada()

	return u.conciliacoes.PagamentosSemConta(ctx, portsout.SemContaFiltro{
		EmpresaIDs:   empresaIDs,
		De:           de,
		Ate:          ate,
		Limite:       paginacao.Size,
		Deslocamento: (paginacao.Page - 1) * paginacao.Size,
	})
}

func (u *ConciliacaoUsecaseImpl) lancamentoComAcesso(ctx context.Context, lancamentoID, usuarioID uuid.UUID) (*domaincaixa.LancamentoExtrato, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	lancamento, err := u.conciliacoes.ObterLancamento(ctx, lancamentoID)
	if err != nil {
		return nil, err
	}
	if lancamento == nil {
		return nil, domaincaixa.ErroLancamentoNaoEncontrado()
	}
	if _, err := u.contaComAcesso(ctx, lancamento.ContaBancariaID, usuarioID); err != nil {
		return nil, err
	}

	return lancamento, nil
}

func (u *ConciliacaoUsecaseImpl) conciliacaoComAcesso(ctx context.Context, conciliacaoID, usuarioID uuid.UUID) (*domaincaixa.Conciliacao, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	conciliacao, err := u.conciliacoes.Obter(ctx, conciliacaoID)
	if err != nil {
		return nil, err
	}
	if conciliacao == nil {
		return nil, domaincaixa.ErroConciliacaoNaoEncontrada()
	}
	if _, err := u.contaComAcesso(ctx, conciliacao.ContaBancariaID, usuarioID); err != nil {
		return nil, err
	}

	return conciliacao, nil
}

func referenciasDosMovimentos(entradas []portsin.ReferenciaInput) ([]domaincaixa.ReferenciaMovimento, error) {
	if len(entradas) == 0 {
		return nil, domain.ErroValidacao("informe ao menos um movimento para conciliar")
	}
	if len(entradas) > domaincaixa.MaximoDeMovimentosPorConciliacao {
		return nil, domain.ErroValidacao("uma conciliação aceita no máximo 50 movimentos")
	}

	referencias := make([]domaincaixa.ReferenciaMovimento, 0, len(entradas))
	vistas := make(map[domaincaixa.ReferenciaMovimento]struct{}, len(entradas))
	for _, entrada := range entradas {
		tipo, err := domaincaixa.NovoTipoMovimento(entrada.Tipo)
		if err != nil {
			return nil, err
		}
		if entrada.ID == uuid.Nil {
			return nil, domain.ErroValidacao("movimento sem id")
		}

		referencia := domaincaixa.ReferenciaMovimento{Tipo: tipo, ID: entrada.ID}
		if _, repetida := vistas[referencia]; repetida {
			return nil, domain.ErroValidacao("movimento repetido na conciliação")
		}
		vistas[referencia] = struct{}{}
		referencias = append(referencias, referencia)
	}

	return referencias, nil
}
