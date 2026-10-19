package caixa

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
}

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if cargo == nil || (!cargo.Financeiro && !cargo.Administrador) {
		return domain.ErroPermissao("perfil sem permissão para o financeiro")
	}

	return nil
}

func (p *permissoes) exigeAdministrador(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if cargo == nil || !cargo.Administrador {
		return domain.ErroPermissao("perfil sem permissão para esta operação")
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

	return p.cargos.Obter(ctx, usuario.CargoID)
}

type escopoPorEmpresa struct {
	empresas portsout.EmpresaRepository
	contas   portsout.ContaBancariaRepository
}

func (e *escopoPorEmpresa) acesso(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return e.empresas.AcessoDoUsuario(ctx, usuarioID)
}

func (e *escopoPorEmpresa) exigeAcesso(ctx context.Context, usuarioID, empresaID uuid.UUID) error {
	acesso, err := e.acesso(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	return nil
}

func (e *escopoPorEmpresa) empresasVisiveis(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) ([]uuid.UUID, error) {
	acesso, err := e.acesso(ctx, usuarioID)
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

func (e *escopoPorEmpresa) contaComAcesso(ctx context.Context, contaID, usuarioID uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	if contaID == uuid.Nil {
		return nil, domain.ErroValidacao("conta bancária é obrigatória")
	}

	conta, err := e.contas.Obter(ctx, contaID)
	if err != nil {
		return nil, err
	}
	if conta == nil {
		return nil, domaincaixa.ErroContaNaoEncontrada()
	}
	if err := e.exigeAcesso(ctx, usuarioID, conta.EmpresaID); err != nil {
		return nil, err
	}

	return conta, nil
}

func (e *escopoPorEmpresa) exigeContaAtiva(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error {
	if !conta.Ativa {
		return domaincaixa.ErroContaInativa()
	}

	empresa, err := e.empresas.Obter(ctx, conta.EmpresaID)
	if err != nil {
		return err
	}
	if empresa == nil || !empresa.Ativa {
		return domaincaixa.ErroEmpresaInativa()
	}

	return nil
}

func erroSemAcessoAEmpresa() error {
	return domain.ErroPermissao("perfil sem acesso a esta empresa")
}
