package notafiscal

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios       portsout.UsuarioRepository
	cargos         portsout.CargoRepository
	representantes portsout.RepresentanteRepository
	empresas       portsout.EmpresaRepository
	clientes       portsout.ClienteRepository
	notas          portsout.NotaFiscalRepository
}

type perfilDoUsuario struct {
	usuarioID       uuid.UUID
	administrador   bool
	financeiro      bool
	representanteID *uuid.UUID
	acesso          domainempresa.Acesso
}

type escopoDaConsulta struct {
	empresa         *uuid.UUID
	empresaIDs      []uuid.UUID
	representanteID *uuid.UUID
}

func (p perfilDoUsuario) restritoAoRepresentante() bool {
	return p.representanteID != nil
}

func (p perfilDoUsuario) escopo(empresa *uuid.UUID) (escopoDaConsulta, error) {
	if empresa != nil && !p.acesso.Permite(*empresa) {
		return escopoDaConsulta{}, erroSemAcessoAEmpresa()
	}

	escopo := escopoDaConsulta{empresa: empresa, representanteID: p.representanteID}
	if !p.acesso.Todas() {
		escopo.empresaIDs = p.acesso.EmpresaIDs()
	}

	return escopo, nil
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

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.financeiro {
		return perfil, domain.ErroPermissao("perfil sem permissão para gerenciar notas fiscais")
	}

	return perfil, nil
}

func (p *permissoes) conferirVisivel(ctx context.Context, perfil perfilDoUsuario, nota *domainnotafiscal.NotaFiscal) error {
	if !perfil.acesso.Permite(nota.EmpresaID) {
		return erroSemAcessoAEmpresa()
	}
	if !perfil.restritoAoRepresentante() {
		return nil
	}
	if nota.ClienteID == nil {
		return erroNotaDeOutroRepresentante()
	}

	cliente, err := p.clientes.Obter(ctx, *nota.ClienteID)
	if err != nil {
		return err
	}
	if cliente == nil || cliente.RepresentanteID == nil || *cliente.RepresentanteID != *perfil.representanteID {
		return erroNotaDeOutroRepresentante()
	}

	return nil
}

func (p *permissoes) notaVisivel(ctx context.Context, perfil perfilDoUsuario, notaFiscalID uuid.UUID) (*domainnotafiscal.NotaFiscal, error) {
	nota, err := p.notas.Obter(ctx, notaFiscalID)
	if err != nil {
		return nil, err
	}
	if nota == nil {
		return nil, domain.ErroNaoEncontrado("nota fiscal não encontrada")
	}
	if err := p.conferirVisivel(ctx, perfil, nota); err != nil {
		return nil, err
	}

	return nota, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}

func erroNotaDeOutroRepresentante() error {
	return domain.ErroPermissao("nota fiscal de cliente de outro representante")
}
