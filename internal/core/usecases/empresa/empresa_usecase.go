package empresa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/empresa"
	"github.com/google/uuid"
)

var _ portsin.EmpresaUseCase = (*EmpresaUsecaseImpl)(nil)

type EmpresaUsecaseImpl struct {
	repo portsout.EmpresaRepository
	permissoes
}

func NewEmpresaUsecase(
	repo portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *EmpresaUsecaseImpl {
	return &EmpresaUsecaseImpl{
		repo: repo,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
		},
	}
}

func (u *EmpresaUsecaseImpl) Criar(ctx context.Context, input portsin.CriarEmpresaInput) (uuid.UUID, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return uuid.Nil, err
	}

	cnpj, err := domainempresa.NovoCNPJ(input.CNPJ)
	if err != nil {
		return uuid.Nil, err
	}

	dados, err := dadosDaEmpresa(input.RazaoSocial, input.NomeFantasia, input.UF, input.RegimeTributario)
	if err != nil {
		return uuid.Nil, err
	}

	existente, err := u.repo.ObterPorCNPJ(ctx, cnpj)
	if err != nil {
		return uuid.Nil, err
	}
	if existente != nil {
		return uuid.Nil, domain.ErroConflito("já existe uma empresa com este cnpj")
	}

	empresa, err := domainempresa.NovaEmpresa(cnpj, dados)
	if err != nil {
		return uuid.Nil, err
	}

	return u.repo.Criar(ctx, empresa)
}

func (u *EmpresaUsecaseImpl) Obter(ctx context.Context, empresaID, usuarioID uuid.UUID) (*domainempresa.Empresa, error) {
	empresa, err := u.buscar(ctx, empresaID)
	if err != nil {
		return nil, err
	}

	acesso, err := u.repo.AcessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if !acesso.Permite(empresa.ID) {
		return nil, domain.ErroPermissao("perfil sem acesso a esta empresa")
	}

	return empresa, nil
}

func (u *EmpresaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarEmpresasInput) ([]*domainempresa.Empresa, error) {
	acesso, err := u.repo.AcessoDoUsuario(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	return u.repo.Listar(ctx, portsout.EmpresaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Ativa:           input.Ativa,
		EmpresaIDs:      acesso.EmpresaIDs(),
	})
}

func (u *EmpresaUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarEmpresaInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	empresa, err := u.buscar(ctx, input.EmpresaID)
	if err != nil {
		return err
	}

	dados, err := dadosDaEmpresa(input.RazaoSocial, input.NomeFantasia, input.UF, input.RegimeTributario)
	if err != nil {
		return err
	}

	if err := empresa.Alterar(dados, time.Now().UTC()); err != nil {
		return err
	}

	return u.gravar(ctx, empresa)
}

func (u *EmpresaUsecaseImpl) AlternarAtiva(ctx context.Context, input portsin.AlternarAtivaEmpresaInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	empresa, err := u.buscar(ctx, input.EmpresaID)
	if err != nil {
		return err
	}

	agora := time.Now().UTC()
	if input.Ativa {
		empresa.Ativar(agora)
	} else {
		empresa.Desativar(agora)
	}

	return u.gravar(ctx, empresa)
}

func (u *EmpresaUsecaseImpl) Remover(ctx context.Context, empresaID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	removida, err := u.repo.Remover(ctx, empresaID)
	if err != nil {
		return err
	}
	if !removida {
		return domain.ErroNaoEncontrado("empresa não encontrada")
	}

	return nil
}

func (u *EmpresaUsecaseImpl) ObterVinculos(ctx context.Context, alvoID, usuarioID uuid.UUID) (*domainempresa.Vinculos, error) {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return nil, err
	}

	vinculos, err := u.repo.VinculosDoUsuario(ctx, alvoID)
	if err != nil {
		return nil, err
	}
	if vinculos == nil {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	return vinculos, nil
}

func (u *EmpresaUsecaseImpl) DefinirVinculos(ctx context.Context, input portsin.DefinirVinculosInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	empresaIDs := make([]uuid.UUID, 0, len(input.EmpresaIDs))
	vistas := make(map[uuid.UUID]struct{}, len(input.EmpresaIDs))
	for _, empresaID := range input.EmpresaIDs {
		if _, repetida := vistas[empresaID]; repetida {
			continue
		}
		vistas[empresaID] = struct{}{}

		empresa, err := u.repo.Obter(ctx, empresaID)
		if err != nil {
			return err
		}
		if empresa == nil {
			return domain.ErroValidacao("empresa " + empresaID.String() + " não encontrada")
		}

		empresaIDs = append(empresaIDs, empresaID)
	}

	return u.repo.DefinirVinculos(ctx, input.AlvoID, domainempresa.Vinculos{
		AcessoTodas: input.AcessoTodas,
		EmpresaIDs:  empresaIDs,
	})
}

func (u *EmpresaUsecaseImpl) buscar(ctx context.Context, empresaID uuid.UUID) (*domainempresa.Empresa, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa não informada")
	}

	empresa, err := u.repo.Obter(ctx, empresaID)
	if err != nil {
		return nil, err
	}
	if empresa == nil {
		return nil, domain.ErroNaoEncontrado("empresa não encontrada")
	}

	return empresa, nil
}

func (u *EmpresaUsecaseImpl) gravar(ctx context.Context, empresa *domainempresa.Empresa) error {
	alterada, err := u.repo.Atualizar(ctx, empresa)
	if err != nil {
		return err
	}
	if !alterada {
		return domain.ErroNaoEncontrado("empresa não encontrada")
	}

	return nil
}

func dadosDaEmpresa(razaoSocial, nomeFantasia, uf, regimeTributario string) (domainempresa.DadosEmpresa, error) {
	estado, err := domainempresa.NovaUF(uf)
	if err != nil {
		return domainempresa.DadosEmpresa{}, err
	}

	regime, err := domainempresa.NovoRegimeTributario(regimeTributario)
	if err != nil {
		return domainempresa.DadosEmpresa{}, err
	}

	return domainempresa.DadosEmpresa{
		RazaoSocial:      razaoSocial,
		NomeFantasia:     nomeFantasia,
		UF:               estado,
		RegimeTributario: regime,
	}, nil
}
