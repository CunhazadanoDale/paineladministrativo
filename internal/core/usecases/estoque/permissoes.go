package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsoutusuarios.UsuarioRepository
	cargos   portsoutusuarios.CargoRepository
}

func (p *permissoes) exigeAdministrador(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if cargo == nil || !cargo.Administrador {
		return domain.ErroPermissao("perfil sem permissão para gerenciar o estoque")
	}

	return nil
}

func (p *permissoes) cargoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainusuarios.Cargo, error) {
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	usuario, err := p.usuarios.GetByID(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := p.cargos.GetByID(ctx, usuario.CargoID)
	if err != nil {
		return nil, err
	}

	return cargo, nil
}
