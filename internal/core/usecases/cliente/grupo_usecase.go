package cliente

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
)

var _ portsin.GrupoUseCase = (*GrupoUsecaseImpl)(nil)

type GrupoUsecaseImpl struct {
	grupos portsout.GrupoRepository
	permissoes
}

func NewGrupoUsecase(
	grupos portsout.GrupoRepository,
	representantes portsout.RepresentanteRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *GrupoUsecaseImpl {
	return &GrupoUsecaseImpl{
		grupos: grupos,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
		},
	}
}

func (u *GrupoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarGrupoInput) (*domaincliente.Grupo, error) {
	if _, err := u.exigeComercial(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	grupo, err := domaincliente.NovoGrupo(input.Nome)
	if err != nil {
		return nil, err
	}

	if _, err := u.grupos.Criar(ctx, grupo); err != nil {
		return nil, traduzirGrupoRepetido(err)
	}

	return grupo, nil
}

func (u *GrupoUsecaseImpl) Obter(ctx context.Context, grupoID uuid.UUID) (*domaincliente.Grupo, error) {
	grupo, err := u.grupos.Obter(ctx, grupoID)
	if err != nil {
		return nil, err
	}
	if grupo == nil {
		return nil, domain.ErroNaoEncontrado("grupo de clientes não encontrado")
	}

	return grupo, nil
}

func (u *GrupoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarGruposInput) ([]*domaincliente.Grupo, error) {
	return u.grupos.Listar(ctx, portsout.GrupoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Busca:           input.Busca,
		Ativo:           input.Ativo,
	})
}

func (u *GrupoUsecaseImpl) Renomear(ctx context.Context, input portsin.RenomearGrupoInput) (*domaincliente.Grupo, error) {
	return u.alterar(ctx, input.GrupoID, input.UsuarioID, func(grupo *domaincliente.Grupo, agora time.Time) error {
		return grupo.Renomear(input.Nome, agora)
	})
}

func (u *GrupoUsecaseImpl) Ativar(ctx context.Context, grupoID, usuarioID uuid.UUID) (*domaincliente.Grupo, error) {
	return u.alterar(ctx, grupoID, usuarioID, func(grupo *domaincliente.Grupo, agora time.Time) error {
		grupo.Ativar(agora)
		return nil
	})
}

func (u *GrupoUsecaseImpl) Desativar(ctx context.Context, grupoID, usuarioID uuid.UUID) (*domaincliente.Grupo, error) {
	return u.alterar(ctx, grupoID, usuarioID, func(grupo *domaincliente.Grupo, agora time.Time) error {
		grupo.Desativar(agora)
		return nil
	})
}

func (u *GrupoUsecaseImpl) alterar(
	ctx context.Context,
	grupoID, usuarioID uuid.UUID,
	aplicar func(*domaincliente.Grupo, time.Time) error,
) (*domaincliente.Grupo, error) {
	if _, err := u.exigeComercial(ctx, usuarioID); err != nil {
		return nil, err
	}

	grupo, err := u.Obter(ctx, grupoID)
	if err != nil {
		return nil, err
	}

	if err := aplicar(grupo, time.Now().UTC()); err != nil {
		return nil, err
	}

	atualizado, err := u.grupos.Atualizar(ctx, grupo)
	if err != nil {
		return nil, traduzirGrupoRepetido(err)
	}
	if !atualizado {
		return nil, domain.ErroNaoEncontrado("grupo de clientes não encontrado")
	}

	return grupo, nil
}

func traduzirGrupoRepetido(err error) error {
	if errors.Is(err, domain.ErrConflito) {
		return domain.ErroConflito("já existe um grupo de clientes com este nome")
	}

	return err
}
