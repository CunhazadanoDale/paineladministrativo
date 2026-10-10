package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsoutsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios    portsoutusuarios.UsuarioRepository
	cargos      portsoutusuarios.CargoRepository
	aprovadores portsoutsolicitacao.AprovadorRepository
}

func (p *permissoes) ehAdministrador(ctx context.Context, usuarioID uuid.UUID) (bool, error) {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return false, err
	}

	return cargo != nil && cargo.Administrador, nil
}

func (p *permissoes) ehFinanceiro(ctx context.Context, usuarioID uuid.UUID) (bool, error) {
	cargo, err := p.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return false, err
	}
	if cargo != nil && cargo.Financeiro {
		return true, nil
	}

	return p.ehAdministrador(ctx, usuarioID)
}

func (p *permissoes) ehAprovador(ctx context.Context, usuarioID uuid.UUID) (bool, error) {
	aprovador, err := p.aprovadores.ObterPorUsuarioID(ctx, usuarioID)
	if err != nil {
		return false, err
	}
	if aprovador != nil {
		return true, nil
	}

	return p.ehAdministrador(ctx, usuarioID)
}

func (p *permissoes) cargoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainusuarios.Cargo, error) {
	if usuarioID == uuid.Nil {
		return nil, nil
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

func podeVerSolicitacao(ctx context.Context, p *permissoes, solicitacao *domainsolicitacao.Solicitacao, usuarioID uuid.UUID) (bool, error) {
	if solicitacao.SolicitanteID == usuarioID {
		return true, nil
	}

	administrador, err := p.ehAdministrador(ctx, usuarioID)
	if err != nil {
		return false, err
	}
	if administrador {
		return true, nil
	}

	financeiro, err := p.ehFinanceiro(ctx, usuarioID)
	if err != nil {
		return false, err
	}
	if financeiro && solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao {
		return true, nil
	}

	return p.ehAprovador(ctx, usuarioID)
}
