package dre

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	"github.com/google/uuid"
)

type permissoes struct {
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
	empresas portsout.EmpresaRepository
}

func (p *permissoes) exigeFinanceiro(ctx context.Context, usuarioID uuid.UUID) error {
	if usuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário não informado")
	}

	usuario, err := p.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return err
	}
	if usuario == nil {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := p.cargos.Obter(ctx, usuario.CargoID)
	if err != nil {
		return err
	}
	if cargo == nil || (!cargo.Financeiro && !cargo.Administrador) {
		return domain.ErroPermissao("perfil sem permissão para a DRE e os painéis")
	}

	return nil
}

type escopo struct {
	empresaIDs       []uuid.UUID
	empresasExigidas []uuid.UUID
}

func (p *permissoes) escopoDoUsuario(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) (escopo, error) {
	if err := p.exigeFinanceiro(ctx, usuarioID); err != nil {
		return escopo{}, err
	}

	acesso, err := p.empresas.AcessoDoUsuario(ctx, usuarioID)
	if err != nil {
		return escopo{}, err
	}
	if empresa == nil {
		return escopo{empresaIDs: acesso.EmpresaIDs()}, nil
	}
	if !acesso.Permite(*empresa) {
		return escopo{}, domain.ErroPermissao("perfil sem acesso a esta empresa")
	}

	return escopo{empresaIDs: []uuid.UUID{*empresa}, empresasExigidas: []uuid.UUID{*empresa}}, nil
}
