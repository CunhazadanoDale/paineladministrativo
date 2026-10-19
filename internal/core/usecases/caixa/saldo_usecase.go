package caixa

import (
	"context"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

var _ portsin.SaldoUseCase = (*SaldoUsecaseImpl)(nil)

type SaldoUsecaseImpl struct {
	saldos portsout.SaldoRepository
	permissoes
	escopoPorEmpresa
}

func NewSaldoUsecase(
	saldos portsout.SaldoRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *SaldoUsecaseImpl {
	return &SaldoUsecaseImpl{
		saldos:           saldos,
		permissoes:       permissoes{usuarios: usuarios, cargos: cargos},
		escopoPorEmpresa: escopoPorEmpresa{empresas: empresas},
	}
}

func (u *SaldoUsecaseImpl) Listar(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) ([]*domaincaixa.SaldoConta, error) {
	if err := u.exigeFinanceiro(ctx, usuarioID); err != nil {
		return nil, err
	}

	empresaIDs, err := u.empresasVisiveis(ctx, usuarioID, empresa)
	if err != nil {
		return nil, err
	}

	return u.saldos.Saldos(ctx, empresaIDs)
}
