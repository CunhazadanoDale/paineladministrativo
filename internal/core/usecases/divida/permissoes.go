package divida

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
	empresas portsout.EmpresaRepository
	dividas  portsout.DividaRepository
}

type perfilDoUsuario struct {
	usuarioID uuid.UUID
	acesso    domainempresa.Acesso
}

type escopoDeEmpresas struct {
	empresa    *uuid.UUID
	empresaIDs []uuid.UUID
}

func (p perfilDoUsuario) escopo(empresa *uuid.UUID) (escopoDeEmpresas, error) {
	if empresa != nil && !p.acesso.Permite(*empresa) {
		return escopoDeEmpresas{}, erroSemAcessoAEmpresa()
	}

	escopo := escopoDeEmpresas{empresa: empresa}
	if !p.acesso.Todas() {
		escopo.empresaIDs = p.acesso.EmpresaIDs()
	}

	return escopo, nil
}

func (e escopoDeEmpresas) lista() []uuid.UUID {
	if e.empresa != nil {
		return []uuid.UUID{*e.empresa}
	}

	return e.empresaIDs
}

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
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
	if cargo == nil || (!cargo.Financeiro && !cargo.Administrador) {
		return perfilDoUsuario{}, domain.ErroPermissao("perfil sem permissão para gerenciar dívidas")
	}

	perfil := perfilDoUsuario{usuarioID: usuarioID}
	if perfil.acesso, err = p.empresas.AcessoDoUsuario(ctx, usuarioID); err != nil {
		return perfilDoUsuario{}, err
	}

	return perfil, nil
}

func (p *permissoes) dividaVisivel(ctx context.Context, perfil perfilDoUsuario, dividaID uuid.UUID) (*domaindivida.Divida, error) {
	divida, err := p.dividas.Obter(ctx, dividaID)
	if err != nil {
		return nil, err
	}
	if divida == nil {
		return nil, domaindivida.ErroDividaNaoEncontrada()
	}
	if !perfil.acesso.Permite(divida.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	return divida, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
