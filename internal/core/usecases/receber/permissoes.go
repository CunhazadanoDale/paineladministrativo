package receber

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
	empresas portsout.EmpresaRepository
	titulos  portsout.TituloReceberRepository
}

type perfilDoUsuario struct {
	usuarioID uuid.UUID
	acesso    domainempresa.Acesso
}

type escopo struct {
	empresa    *uuid.UUID
	empresaIDs []uuid.UUID
}

func (p perfilDoUsuario) escopo(empresa *uuid.UUID) (escopo, error) {
	if empresa != nil && !p.acesso.Permite(*empresa) {
		return escopo{}, erroSemAcessoAEmpresa()
	}

	resultado := escopo{empresa: empresa}
	if !p.acesso.Todas() {
		resultado.empresaIDs = p.acesso.EmpresaIDs()
	}

	return resultado, nil
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
		return perfilDoUsuario{}, domain.ErroPermissao("perfil sem permissão para contas a receber")
	}

	acesso, err := p.empresas.AcessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return perfilDoUsuario{}, err
	}

	return perfilDoUsuario{usuarioID: usuarioID, acesso: acesso}, nil
}

func (p *permissoes) tituloVisivel(ctx context.Context, perfil perfilDoUsuario, tituloID uuid.UUID) (*domainreceber.Titulo, error) {
	titulo, err := p.titulos.Obter(ctx, tituloID)
	if err != nil {
		return nil, err
	}
	if titulo == nil {
		return nil, domainreceber.ErroTituloNaoEncontrado()
	}
	if !perfil.acesso.Permite(titulo.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	return titulo, nil
}

func (p *permissoes) tituloParaAlterar(ctx context.Context, usuarioID, tituloID uuid.UUID, versao int) (*domainreceber.Titulo, error) {
	perfil, err := p.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	titulo, err := p.tituloVisivel(ctx, perfil, tituloID)
	if err != nil {
		return nil, err
	}
	if err := titulo.ConferirVersao(versao); err != nil {
		return nil, err
	}

	return titulo, nil
}

func (p *permissoes) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domainreceber.ErroEmpresaObrigatoria()
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := p.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domainreceber.ErroEmpresaNaoEncontrada()
	}
	if !empresa.Ativa {
		return domainreceber.ErroEmpresaInativa()
	}

	return nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
