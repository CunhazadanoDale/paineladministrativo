package solicitacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

var _ portsin.ResumoContasAPagarUseCase = (*ResumoContasAPagarUsecaseImpl)(nil)

type ResumoContasAPagarUsecaseImpl struct {
	resumos portsout.ResumoContasAPagarRepository
	permissoes
}

func NewResumoContasAPagarUsecase(
	resumos portsout.ResumoContasAPagarRepository,
	aprovadores portsout.AprovadorRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *ResumoContasAPagarUsecaseImpl {
	return &ResumoContasAPagarUsecaseImpl{
		resumos: resumos,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			aprovadores: aprovadores,
			empresas:    empresas,
		},
	}
}

func (u *ResumoContasAPagarUsecaseImpl) Consultar(ctx context.Context, input portsin.ConsultarResumoInput) (domainsolicitacao.ResumoContasAPagar, error) {
	financeiro, err := u.ehFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return domainsolicitacao.ResumoContasAPagar{}, err
	}
	if !financeiro {
		return domainsolicitacao.ResumoContasAPagar{}, domain.ErroPermissao("perfil sem permissão para o resumo de contas a pagar")
	}

	agora := time.Now().UTC()
	periodo, err := domainsolicitacao.NovoPeriodoResumo(input.De, input.Ate, agora)
	if err != nil {
		return domainsolicitacao.ResumoContasAPagar{}, err
	}

	acesso, err := u.empresas.AcessoDoUsuario(ctx, input.UsuarioID)
	if err != nil {
		return domainsolicitacao.ResumoContasAPagar{}, err
	}
	empresaIDs := acesso.EmpresaIDs()
	if input.Empresa != nil {
		if !acesso.Permite(*input.Empresa) {
			return domainsolicitacao.ResumoContasAPagar{}, domain.ErroPermissao("perfil sem acesso a esta empresa")
		}
		empresaIDs = []uuid.UUID{*input.Empresa}
	}

	filtro := portsout.FiltroResumo{
		De:         periodo.De,
		Ate:        periodo.Ate,
		Hoje:       inicioDoDia(agora),
		EmpresaIDs: empresaIDs,
	}

	semanas, err := u.resumos.ResumoSemanal(ctx, filtro)
	if err != nil {
		return domainsolicitacao.ResumoContasAPagar{}, err
	}
	atrasadas, err := u.resumos.Atrasadas(ctx, filtro)
	if err != nil {
		return domainsolicitacao.ResumoContasAPagar{}, err
	}

	return domainsolicitacao.NovoResumoContasAPagar(periodo, semanas, atrasadas), nil
}
