package dre

import (
	"context"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
)

var _ portsin.FaturamentoUseCase = (*FaturamentoUsecaseImpl)(nil)

type FaturamentoUsecaseImpl struct {
	resumos portsout.ResumoRepository
	permissoes
}

func NewFaturamentoUsecase(
	resumos portsout.ResumoRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *FaturamentoUsecaseImpl {
	return &FaturamentoUsecaseImpl{
		resumos:    resumos,
		permissoes: permissoes{usuarios: usuarios, cargos: cargos, empresas: empresas},
	}
}

func (u *FaturamentoUsecaseImpl) Consultar(ctx context.Context, input portsin.ConsultarPeriodoInput) (domaindre.ResumoFaturamento, error) {
	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return domaindre.ResumoFaturamento{}, err
	}

	periodo, err := domaindre.NovoPeriodoDoFaturamento(input.De, input.Ate, domaindre.Hoje(time.Now()))
	if err != nil {
		return domaindre.ResumoFaturamento{}, err
	}

	faturamentos, err := u.resumos.Faturamento(ctx, portsout.FiltroResumo{
		Intervalo:  periodo.Intervalo(),
		EmpresaIDs: escopo.empresaIDs,
	})
	if err != nil {
		return domaindre.ResumoFaturamento{}, err
	}

	return domaindre.NovoResumoFaturamento(periodo, escopo.empresasExigidas, faturamentos), nil
}
