package estoque

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.LocalUseCase = (*LocalUsecaseImpl)(nil)

type LocalUsecaseImpl struct {
	locais  portsout.LocalRepository
	estoque portsout.EstoqueRepository
	permissoes
}

func NewLocalUsecase(
	locais portsout.LocalRepository,
	estoque portsout.EstoqueRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *LocalUsecaseImpl {
	return &LocalUsecaseImpl{
		locais:  locais,
		estoque: estoque,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
		},
	}
}

func (u *LocalUsecaseImpl) Criar(ctx context.Context, input portsin.CriarLocalInput) (uuid.UUID, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return uuid.Nil, err
	}
	if err := u.validarEmpresa(ctx, input.UsuarioID, input.EmpresaID); err != nil {
		return uuid.Nil, err
	}

	local, err := domainestoque.NovoLocal(input.EmpresaID, input.Nome)
	if err != nil {
		return uuid.Nil, err
	}

	id, err := u.locais.Criar(ctx, local)
	if errors.Is(err, domain.ErrConflito) {
		return uuid.Nil, erroNomeDeLocalRepetido()
	}

	return id, err
}

func (u *LocalUsecaseImpl) Obter(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error) {
	return u.buscarComAcesso(ctx, localID, usuarioID)
}

func (u *LocalUsecaseImpl) Listar(ctx context.Context, input portsin.ListarLocaisInput) ([]*domainestoque.Local, error) {
	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.locais.Listar(ctx, portsout.LocalFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EmpresaIDs:      empresaIDs,
		Ativo:           input.Ativo,
	})
}

func (u *LocalUsecaseImpl) Renomear(ctx context.Context, input portsin.RenomearLocalInput) (*domainestoque.Local, error) {
	local, err := u.buscarParaAlterar(ctx, input.LocalID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	if err := local.Renomear(input.Nome, time.Now().UTC()); err != nil {
		return nil, err
	}

	if err := u.atualizar(ctx, local); err != nil {
		return nil, err
	}

	return local, nil
}

func (u *LocalUsecaseImpl) Ativar(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error) {
	local, err := u.buscarParaAlterar(ctx, localID, usuarioID)
	if err != nil {
		return nil, err
	}

	local.Ativar(time.Now().UTC())
	if err := u.atualizar(ctx, local); err != nil {
		return nil, err
	}

	return local, nil
}

func (u *LocalUsecaseImpl) Desativar(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error) {
	local, err := u.buscarParaAlterar(ctx, localID, usuarioID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	desativado, err := u.locais.DesativarSemSaldo(ctx, local.ID, agora)
	if err != nil {
		return nil, err
	}
	if err := local.Desativar(!desativado, agora); err != nil {
		return nil, err
	}

	return local, nil
}

func (u *LocalUsecaseImpl) Remover(ctx context.Context, localID, usuarioID uuid.UUID) error {
	local, err := u.buscarParaAlterar(ctx, localID, usuarioID)
	if err != nil {
		return err
	}

	removido, err := u.locais.Remover(ctx, local.ID)
	if err != nil {
		return err
	}
	if !removido {
		return erroLocalNaoEncontrado()
	}

	return nil
}

func (u *LocalUsecaseImpl) ListarSaldos(ctx context.Context, input portsin.ListarSaldosDoLocalInput) ([]*domainestoque.SaldoLocalDetalhe, error) {
	local, err := u.buscarComAcesso(ctx, input.LocalID, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	return u.estoque.SaldosDoLocal(ctx, portsout.SaldoDoLocalFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		LocalID:         local.ID,
		ComSaldo:        input.ComSaldo,
	})
}

func (u *LocalUsecaseImpl) validarEmpresa(ctx context.Context, usuarioID, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa do local não informada")
	}

	acesso, err := u.acessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domain.ErroValidacao("empresa do local não encontrada")
	}
	if !empresa.Ativa {
		return domain.ErroValidacao("empresa inativa não recebe local novo")
	}

	return nil
}

func (u *LocalUsecaseImpl) buscarParaAlterar(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error) {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return nil, err
	}

	return u.buscarComAcesso(ctx, localID, usuarioID)
}

func (u *LocalUsecaseImpl) buscarComAcesso(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error) {
	local, err := u.locais.Obter(ctx, localID)
	if err != nil {
		return nil, err
	}
	if local == nil {
		return nil, erroLocalNaoEncontrado()
	}
	if err := u.exigeAcessoAosLocais(ctx, usuarioID, local); err != nil {
		return nil, err
	}

	return local, nil
}

func (u *LocalUsecaseImpl) atualizar(ctx context.Context, local *domainestoque.Local) error {
	atualizado, err := u.locais.Atualizar(ctx, local)
	if errors.Is(err, domain.ErrConflito) {
		return erroNomeDeLocalRepetido()
	}
	if err != nil {
		return err
	}
	if !atualizado {
		return erroLocalNaoEncontrado()
	}

	return nil
}

func erroLocalNaoEncontrado() error {
	return domain.ErroNaoEncontrado("local de estoque não encontrado")
}

func erroNomeDeLocalRepetido() error {
	return domain.ErroConflito("já existe um local com este nome na empresa")
}
