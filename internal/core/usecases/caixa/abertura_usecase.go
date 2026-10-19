package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.AberturaUseCase = (*AberturaUsecaseImpl)(nil)

type AberturaUsecaseImpl struct {
	aberturas portsout.AberturaRepository
	permissoes
	escopoPorEmpresa
}

func NewAberturaUsecase(
	aberturas portsout.AberturaRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *AberturaUsecaseImpl {
	return &AberturaUsecaseImpl{
		aberturas:        aberturas,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas, contas: contas},
	}
}

func (u *AberturaUsecaseImpl) Definir(ctx context.Context, input portsin.DefinirAberturaInput) (*domaincaixa.Abertura, error) {
	if err := u.exigeFinanceiro(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, input.ContaBancariaID, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigeContaAtiva(ctx, conta); err != nil {
		return nil, err
	}

	existente, err := u.aberturas.Obter(ctx, conta.ID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	if existente == nil {
		abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
			ContaBancariaID: conta.ID,
			DataCorte:       input.DataCorte,
			SaldoCentavos:   input.SaldoCentavos,
			UsuarioID:       input.UsuarioID,
		}, agora)
		if err != nil {
			return nil, err
		}
		if err := u.aberturas.Criar(ctx, abertura); err != nil {
			return nil, err
		}

		return abertura, nil
	}

	if input.Versao == nil {
		return nil, domaincaixa.ErroAberturaJaDefinida()
	}
	if err := existente.ConferirVersao(*input.Versao); err != nil {
		return nil, err
	}
	if err := existente.Redefinir(input.DataCorte, input.SaldoCentavos, input.UsuarioID, agora); err != nil {
		return nil, err
	}
	if err := u.aberturas.Atualizar(ctx, existente); err != nil {
		return nil, err
	}

	return existente, nil
}

func (u *AberturaUsecaseImpl) Obter(ctx context.Context, contaBancariaID, usuarioID uuid.UUID) (*domaincaixa.Abertura, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	conta, err := u.contaComAcesso(ctx, contaBancariaID, usuarioID)
	if err != nil {
		return nil, err
	}

	abertura, err := u.aberturas.Obter(ctx, conta.ID)
	if err != nil {
		return nil, err
	}
	if abertura == nil {
		return nil, domaincaixa.ErroAberturaNaoEncontrada()
	}

	return abertura, nil
}
