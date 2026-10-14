package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios       portsout.UsuarioRepository
	cargos         portsout.CargoRepository
	representantes portsout.RepresentanteRepository
}

type perfilDoUsuario struct {
	administrador   bool
	comercial       bool
	representanteID *uuid.UUID
}

func (p perfilDoUsuario) restritoAoRepresentante() bool {
	return p.representanteID != nil
}

func (p perfilDoUsuario) podeVer(cliente *domaincliente.Cliente) bool {
	if !p.restritoAoRepresentante() {
		return true
	}

	return cliente.RepresentanteID != nil && *cliente.RepresentanteID == *p.representanteID
}

func (p *permissoes) perfil(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	if usuarioID == uuid.Nil {
		return perfilDoUsuario{}, domain.ErroValidacao("usuário não informado")
	}

	usuario, err := p.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return perfilDoUsuario{}, err
	}
	if usuario == nil {
		return perfilDoUsuario{}, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := p.cargos.Obter(ctx, usuario.CargoID)
	if err != nil {
		return perfilDoUsuario{}, err
	}

	perfil := perfilDoUsuario{}
	if cargo != nil {
		perfil.administrador = cargo.Administrador
		perfil.comercial = cargo.Comercial || cargo.Administrador
	}
	if perfil.administrador {
		return perfil, nil
	}

	representante, err := p.representantes.ObterPorUsuario(ctx, usuarioID)
	if err != nil {
		return perfilDoUsuario{}, err
	}
	if representante != nil {
		perfil.representanteID = &representante.ID
	}

	return perfil, nil
}

func (p *permissoes) exigeComercial(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.comercial {
		return perfil, domain.ErroPermissao("perfil sem permissão para gerenciar clientes")
	}

	return perfil, nil
}

func (p *permissoes) exigeAdministrador(ctx context.Context, usuarioID uuid.UUID) error {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !perfil.administrador {
		return domain.ErroPermissao("perfil sem permissão para esta operação")
	}

	return nil
}

func erroClienteDeOutroRepresentante() error {
	return domain.ErroPermissao("cliente de outro representante")
}
