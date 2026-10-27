package tabelapreco

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
}

func (u *permissoes) cargoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainusuarios.Cargo, error) {
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	usuario, err := u.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := u.cargos.Obter(ctx, usuario.CargoID)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return &domainusuarios.Cargo{}, nil
	}

	return cargo, nil
}

func (u *permissoes) exigeAdministrador(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := u.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !cargo.Administrador {
		return domain.ErroPermissao("perfil sem permissão para alterar a tabela de preços")
	}

	return nil
}

func (u *permissoes) exigeLeitura(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := u.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !cargo.Administrador && !cargo.Comercial && !cargo.Financeiro {
		return domain.ErroPermissao("perfil sem permissão para ver a tabela de preços")
	}

	return nil
}
