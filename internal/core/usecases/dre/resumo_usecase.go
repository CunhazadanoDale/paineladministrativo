package dre

import (
	"context"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
)

var _ portsin.ResumoDREUseCase = (*ResumoDREUsecaseImpl)(nil)

type ResumoDREUsecaseImpl struct {
	resumos portsout.ResumoRepository
	permissoes
}

func NewResumoDREUsecase(
	resumos portsout.ResumoRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *ResumoDREUsecaseImpl {
	return &ResumoDREUsecaseImpl{
		resumos:    resumos,
		permissoes: permissoes{usuarios: usuarios, cargos: cargos, empresas: empresas},
	}
}

func (u *ResumoDREUsecaseImpl) Consultar(ctx context.Context, input portsin.ConsultarPeriodoInput) (domaindre.ResumoDRE, error) {
	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return domaindre.ResumoDRE{}, err
	}

	periodo, err := domaindre.NovoPeriodoDaDRE(input.De, input.Ate, domaindre.Hoje(time.Now()))
	if err != nil {
		return domaindre.ResumoDRE{}, err
	}

	filtro := portsout.FiltroResumo{Intervalo: periodo.Intervalo(), EmpresaIDs: escopo.empresaIDs}
	lancamentos, err := u.resumos.Lancamentos(ctx, filtro)
	if err != nil {
		return domaindre.ResumoDRE{}, err
	}
	semCusto, err := u.resumos.ConsumosSemCusto(ctx, filtro)
	if err != nil {
		return domaindre.ResumoDRE{}, err
	}

	return domaindre.NovoResumoDRE(periodo, escopo.empresasExigidas, lancamentos, semCusto), nil
}
