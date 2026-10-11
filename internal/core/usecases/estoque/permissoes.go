package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
	empresas portsout.EmpresaRepository
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

func (p *permissoes) exigeAdministradorOuFinanceiro(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if cargo == nil || (!cargo.Administrador && !cargo.Financeiro) {
		return domain.ErroPermissao("perfil sem permissão para consultar o custo do estoque")
	}

	return nil
}

func (p *permissoes) acessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	if usuarioID == uuid.Nil {
		return domainempresa.Acesso{}, domain.ErroValidacao("usuário não informado")
	}

	return p.empresas.AcessoDoUsuario(ctx, usuarioID)
}

func (p *permissoes) empresasVisiveis(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) ([]uuid.UUID, error) {
	acesso, err := p.acessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if empresa == nil {
		return acesso.EmpresaIDs(), nil
	}
	if !acesso.Permite(*empresa) {
		return nil, erroSemAcessoAEmpresa()
	}

	return []uuid.UUID{*empresa}, nil
}

func (p *permissoes) exigeAcessoAosLocais(ctx context.Context, usuarioID uuid.UUID, locais ...*domainestoque.Local) error {
	acesso, err := p.acessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	for _, local := range locais {
		if !acesso.Permite(local.EmpresaID) {
			return erroSemAcessoAEmpresa()
		}
	}

	return nil
}

func (p *permissoes) cargoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainusuarios.Cargo, error) {
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	usuario, err := p.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := p.cargos.Obter(ctx, usuario.CargoID)
	if err != nil {
		return nil, err
	}

	return cargo, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
