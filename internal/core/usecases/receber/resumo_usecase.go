package receber

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
)

var _ portsin.ResumoContasAReceberUseCase = (*ResumoContasAReceberUsecaseImpl)(nil)

type ResumoContasAReceberUsecaseImpl struct {
	resumos portsout.ResumoContasAReceberRepository
	permissoes
}

func NewResumoContasAReceberUsecase(
	resumos portsout.ResumoContasAReceberRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ResumoContasAReceberUsecaseImpl {
	return &ResumoContasAReceberUsecaseImpl{
		resumos: resumos,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
		},
	}
}

func (u *ResumoContasAReceberUsecaseImpl) Consultar(ctx context.Context, input portsin.ConsultarResumoInput) (domainreceber.ResumoContasAReceber, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return domainreceber.ResumoContasAReceber{}, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return domainreceber.ResumoContasAReceber{}, err
	}
	empresaIDs := escopo.empresaIDs
	if escopo.empresa != nil {
		empresaIDs = append(empresaIDs[:0:0], *escopo.empresa)
	}

	agora := time.Now().UTC()
	periodo, err := domainreceber.NovoPeriodoResumo(input.De, input.Ate, agora)
	if err != nil {
		return domainreceber.ResumoContasAReceber{}, err
	}

	filtro := portsout.FiltroResumo{De: periodo.De, Ate: periodo.Ate, Hoje: domainreceber.Hoje(agora), EmpresaIDs: empresaIDs}
	semanas, err := u.resumos.ResumoSemanal(ctx, filtro)
	if err != nil {
		return domainreceber.ResumoContasAReceber{}, err
	}
	atrasados, err := u.resumos.Atrasados(ctx, filtro)
	if err != nil {
		return domainreceber.ResumoContasAReceber{}, err
	}

	return domainreceber.NovoResumoContasAReceber(periodo, semanas, atrasados), nil
}
