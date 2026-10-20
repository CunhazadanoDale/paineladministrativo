package importacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios    portsout.UsuarioRepository
	cargos      portsout.CargoRepository
	empresas    portsout.EmpresaRepository
	importacoes portsout.ImportacaoRepository
}

type perfilDoUsuario struct {
	usuarioID  uuid.UUID
	comercial  bool
	financeiro bool
	acesso     domainempresa.Acesso
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
		perfil.comercial = cargo.Comercial || cargo.Administrador
		perfil.financeiro = cargo.Financeiro || cargo.Administrador
	}

	if perfil.acesso, err = p.empresas.AcessoDoUsuario(ctx, usuarioID); err != nil {
		return perfilDoUsuario{}, err
	}

	return perfil, nil
}

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.financeiro {
		return perfil, domain.ErroPermissao("perfil sem permissão para gerenciar importações")
	}

	return perfil, nil
}

func (p *permissoes) exigeLeitura(ctx context.Context, usuarioID uuid.UUID) (perfilDoUsuario, error) {
	perfil, err := p.perfil(ctx, usuarioID)
	if err != nil {
		return perfil, err
	}
	if !perfil.comercial && !perfil.financeiro {
		return perfil, domain.ErroPermissao("perfil sem permissão para consultar importações")
	}

	return perfil, nil
}

func (p *permissoes) importacaoVisivel(ctx context.Context, perfil perfilDoUsuario, importacaoID uuid.UUID) (*domainimportacao.Importacao, error) {
	importacao, err := p.importacoes.Obter(ctx, importacaoID)
	if err != nil {
		return nil, err
	}
	if importacao == nil {
		return nil, domain.ErroNaoEncontrado("importação não encontrada")
	}
	if !perfil.acesso.Permite(importacao.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	return importacao, nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
