package pedido

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios       portsout.UsuarioRepository
	cargos         portsout.CargoRepository
	representantes portsout.RepresentanteRepository
	empresas       portsout.EmpresaRepository
	pedidos        portsout.PedidoRepository
}

type perfilDoUsuario struct {
	usuarioID       uuid.UUID
	administrador   bool
	comercial       bool
	financeiro      bool
	representanteID *uuid.UUID
	acesso          domainempresa.Acesso
}

func (p perfilDoUsuario) restritoAoRepresentante() bool {
	return p.representanteID != nil
}

func (p perfilDoUsuario) doRepresentante(pedido *domainpedido.Pedido) bool {
	return pedido.RepresentanteID != nil && *pedido.RepresentanteID == *p.representanteID
}

func (p perfilDoUsuario) conferirVisivel(pedido *domainpedido.Pedido) error {
	if !p.acesso.Permite(pedido.EmpresaID) {
		return erroSemAcessoAEmpresa()
	}
	if p.restritoAoRepresentante() && !p.doRepresentante(pedido) {
		return domain.ErroPermissao("pedido de outro representante")
	}

	return nil
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

	perfil := perfilDoUsuario{usuarioID: usuarioID}
	if cargo != nil {
		perfil.administrador = cargo.Administrador
		perfil.comercial = cargo.Comercial || cargo.Administrador
		perfil.financeiro = cargo.Financeiro || cargo.Administrador
	}

	if perfil.acesso, err = p.empresas.AcessoDoUsuario(ctx, usuarioID); err != nil {
		return perfilDoUsuario{}, err
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
		return perfil, domain.ErroPermissao("perfil sem permissão para gerenciar pedidos")
	}

	return perfil, nil
}

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.financeiro {
		return perfil, domain.ErroPermissao("perfil sem permissão para registrar a entrada do pedido")
	}

	return perfil, nil
}

func (p *permissoes) pedidoVisivel(ctx context.Context, perfil perfilDoUsuario, pedidoID uuid.UUID) (*domainpedido.Pedido, error) {
	pedido, err := p.pedidos.Obter(ctx, pedidoID)
	if err != nil {
		return nil, err
	}
	if pedido == nil {
		return nil, domain.ErroNaoEncontrado("pedido não encontrado")
	}
	if err := perfil.conferirVisivel(pedido); err != nil {
		return nil, err
	}

	return pedido, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
