package comissao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios       portsout.UsuarioRepository
	cargos         portsout.CargoRepository
	representantes portsout.RepresentanteRepository
	empresas       portsout.EmpresaRepository
}

type perfilDoUsuario struct {
	usuarioID       uuid.UUID
	comercial       bool
	financeiro      bool
	representanteID *uuid.UUID
	acesso          domainempresa.Acesso
}

type escopo struct {
	empresa         *uuid.UUID
	empresaIDs      []uuid.UUID
	representanteID *uuid.UUID
}

func (p perfilDoUsuario) restritoAoRepresentante() bool {
	return !p.financeiro && p.representanteID != nil
}

func (p perfilDoUsuario) escopo(empresa, representanteID *uuid.UUID) (escopo, error) {
	if empresa != nil && !p.acesso.Permite(*empresa) {
		return escopo{}, erroSemAcessoAEmpresa()
	}

	resultado := escopo{empresa: empresa, representanteID: representanteID}
	if !p.acesso.Todas() {
		resultado.empresaIDs = p.acesso.EmpresaIDs()
	}
	if p.restritoAoRepresentante() {
		if representanteID != nil && *representanteID != *p.representanteID {
			return escopo{}, erroDeOutroRepresentante()
		}
		resultado.representanteID = p.representanteID
	}

	return resultado, nil
}

func (p perfilDoUsuario) conferirPedido(pedido *domainpedido.Pedido) error {
	if !p.acesso.Permite(pedido.EmpresaID) {
		return erroSemAcessoAEmpresa()
	}
	if p.restritoAoRepresentante() && (pedido.RepresentanteID == nil || *pedido.RepresentanteID != *p.representanteID) {
		return erroDeOutroRepresentante()
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
	administrador := false
	if cargo != nil {
		administrador = cargo.Administrador
		perfil.comercial = cargo.Comercial || cargo.Administrador
		perfil.financeiro = cargo.Financeiro || cargo.Administrador
	}

	if perfil.acesso, err = p.empresas.AcessoDoUsuario(ctx, usuarioID); err != nil {
		return perfilDoUsuario{}, err
	}
	if administrador {
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

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.financeiro {
		return perfil, domain.ErroPermissao("perfil sem permissão para apurar comissões")
	}

	return perfil, nil
}

func (p *permissoes) exigeLeituraDeComissoes(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.financeiro && perfil.representanteID == nil {
		return perfil, domain.ErroPermissao("perfil sem permissão para ver comissões")
	}

	return perfil, nil
}

func (p *permissoes) exigeLeituraDoPercentual(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.comercial && !perfil.financeiro && perfil.representanteID == nil {
		return perfil, domain.ErroPermissao("perfil sem permissão para ver a comissão do pedido")
	}

	return perfil, nil
}

func (p *permissoes) exigeDefinicaoDoPercentual(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.comercial {
		return perfil, domain.ErroPermissao("perfil sem permissão para definir a comissão do pedido")
	}
	if perfil.representanteID != nil {
		return perfil, domaincomissao.ErroRepresentanteNaoDefineComissao()
	}

	return perfil, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}

func erroDeOutroRepresentante() error {
	return domain.ErroPermissao("comissão de outro representante")
}
