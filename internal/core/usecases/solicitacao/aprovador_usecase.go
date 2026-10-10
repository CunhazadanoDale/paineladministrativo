package solicitacao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var _ portsin.AprovadorUseCase = (*AprovadorUsecaseImpl)(nil)

type AprovadorUsecaseImpl struct {
	repo     portsout.AprovadorRepository
	usuarios portsoutusuarios.UsuarioRepository
}

func NewAprovadorUsecase(aprovadores portsout.AprovadorRepository, usuarios portsoutusuarios.UsuarioRepository) *AprovadorUsecaseImpl {
	return &AprovadorUsecaseImpl{
		repo:     aprovadores,
		usuarios: usuarios,
	}
}

func (u *AprovadorUsecaseImpl) Designar(ctx context.Context, usuarioID uuid.UUID) (uuid.UUID, error) {
	if usuarioID == uuid.Nil {
		return uuid.Nil, domain.ErroValidacao("usuário do aprovador não informado")
	}

	usuario, err := u.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return uuid.Nil, err
	}
	if usuario == nil {
		return uuid.Nil, domain.ErrNotFound
	}
	if !usuario.Ativo {
		return uuid.Nil, domain.ErroValidacao("usuário inativo não pode ser designado como aprovador")
	}

	existente, err := u.repo.ObterPorUsuarioID(ctx, usuarioID)
	if err != nil {
		return uuid.Nil, err
	}
	if existente != nil {
		return uuid.Nil, domain.ErroConflito("usuário já está designado como aprovador")
	}

	aprovador := &domainsolicitacao.Aprovador{
		ID:        uuid.New(),
		UsuarioID: usuarioID,
		CriadoEm:  time.Now().UTC(),
	}

	return u.repo.Criar(ctx, aprovador)
}

func (u *AprovadorUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do aprovador não informado")
	}

	aprovador, err := u.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if aprovador == nil {
		return nil, domain.ErrNotFound
	}

	return aprovador, nil
}

func (u *AprovadorUsecaseImpl) Remover(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErroValidacao("id do aprovador não informado")
	}

	aprovador, err := u.repo.Obter(ctx, id)
	if err != nil {
		return err
	}
	if aprovador == nil {
		return domain.ErrNotFound
	}

	return u.repo.Remover(ctx, id)
}

func (u *AprovadorUsecaseImpl) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error) {
	return u.repo.Listar(ctx, filtro.Normalizada())
}

func (u *AprovadorUsecaseImpl) EhDesignado(ctx context.Context, usuarioID uuid.UUID) (bool, error) {
	if usuarioID == uuid.Nil {
		return false, domain.ErroValidacao("usuário do aprovador não informado")
	}

	aprovador, err := u.repo.ObterPorUsuarioID(ctx, usuarioID)
	if err != nil {
		return false, err
	}

	return aprovador != nil, nil
}
