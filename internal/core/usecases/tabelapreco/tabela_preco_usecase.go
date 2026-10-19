package tabelapreco

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	"github.com/google/uuid"
)

var _ portsin.TabelaPrecoUseCase = (*TabelaPrecoUsecaseImpl)(nil)

type TabelaPrecoUsecaseImpl struct {
	itens    portsout.TabelaPrecoRepository
	produtos portsout.ProdutoRepository
	usuarios portsout.UsuarioRepository
	cargos   portsout.CargoRepository
}

func NewTabelaPrecoUsecase(
	itens portsout.TabelaPrecoRepository,
	produtos portsout.ProdutoRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *TabelaPrecoUsecaseImpl {
	return &TabelaPrecoUsecaseImpl{itens: itens, produtos: produtos, usuarios: usuarios, cargos: cargos}
}

func (u *TabelaPrecoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarItemInput) (*domaintabelapreco.Item, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	item, err := domaintabelapreco.NovoItem(input.Dados, input.UsuarioID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	produto, err := u.produtos.Obter(ctx, item.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, domaintabelapreco.ErroProdutoNaoEncontrado()
	}
	if !produto.Ativo {
		return nil, domaintabelapreco.ErroProdutoInativo()
	}

	if err := u.itens.Criar(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (u *TabelaPrecoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarItensInput) ([]*domaintabelapreco.Item, error) {
	if err := u.exigeLeitura(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	filtro := portsout.ItemFiltro{PaginacaoFiltro: input.Filtro.Normalizada(), ProdutoID: input.ProdutoID}
	if strings.TrimSpace(input.Modelo) != "" {
		modelo, err := domainpedido.NovoModelo(input.Modelo)
		if err != nil {
			return nil, err
		}
		filtro.Modelo = modelo
	}

	return u.itens.Listar(ctx, filtro)
}

func (u *TabelaPrecoUsecaseImpl) Vigentes(ctx context.Context, input portsin.VigentesInput) ([]*domaintabelapreco.PrecoVigente, error) {
	if err := u.exigeLeitura(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	modelo, err := domainpedido.NovoModelo(input.Modelo)
	if err != nil {
		return nil, err
	}

	data := domainpedido.Hoje(time.Now().UTC())
	if input.Data != nil && !input.Data.IsZero() {
		data = domainpedido.DataDe(*input.Data)
	}

	return u.itens.Vigentes(ctx, portsout.VigenteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Modelo:          modelo,
		Data:            data,
		ProdutoID:       input.ProdutoID,
		Busca:           strings.TrimSpace(input.Busca),
	})
}

func (u *TabelaPrecoUsecaseImpl) Remover(ctx context.Context, itemID, usuarioID uuid.UUID) error {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return err
	}

	return u.itens.RemoverFuturo(ctx, itemID, domainpedido.Hoje(time.Now().UTC()))
}

func (u *TabelaPrecoUsecaseImpl) cargoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainusuarios.Cargo, error) {
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	usuario, err := u.usuarios.Obter(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	cargo, err := u.cargos.Obter(ctx, usuario.CargoID)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return &domainusuarios.Cargo{}, nil
	}

	return cargo, nil
}

func (u *TabelaPrecoUsecaseImpl) exigeAdministrador(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := u.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !cargo.Administrador {
		return domain.ErroPermissao("perfil sem permissão para alterar a tabela de preços")
	}

	return nil
}

func (u *TabelaPrecoUsecaseImpl) exigeLeitura(ctx context.Context, usuarioID uuid.UUID) error {
	cargo, err := u.cargoDoUsuario(ctx, usuarioID)
	if err != nil {
		return err
	}
	if !cargo.Administrador && !cargo.Comercial && !cargo.Financeiro {
		return domain.ErroPermissao("perfil sem permissão para ver a tabela de preços")
	}

	return nil
}
