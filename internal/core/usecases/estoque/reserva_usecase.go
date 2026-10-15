package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.ReservaUseCase = (*ReservaUsecaseImpl)(nil)

type ReservaUsecaseImpl struct {
	produtos portsout.ProdutoRepository
	reservas portsout.ReservaRepository
	operacoesDeEstoque
}

func NewReservaUsecase(
	produtos portsout.ProdutoRepository,
	estoque portsout.EstoqueRepository,
	locais portsout.LocalRepository,
	reservas portsout.ReservaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
	empresas portsout.EmpresaRepository,
) *ReservaUsecaseImpl {
	return &ReservaUsecaseImpl{
		produtos: produtos,
		reservas: reservas,
		operacoesDeEstoque: operacoesDeEstoque{
			estoque: estoque,
			locais:  locais,
			permissoes: permissoes{
				usuarios: usuarios,
				cargos:   cargos,
				empresas: empresas,
			},
		},
	}
}

func (u *ReservaUsecaseImpl) Reservar(ctx context.Context, input portsin.ReservarEstoqueInput) (*domainestoque.Reserva, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return nil, err
	}

	quantidade, err := domainestoque.NovaQuantidade(input.Quantidade)
	if err != nil {
		return nil, err
	}

	local, err := u.localInformado(ctx, input.LocalID, "local da reserva não encontrado")
	if err != nil {
		return nil, err
	}
	if err := u.exigeAcessoAosLocais(ctx, input.UsuarioID, local); err != nil {
		return nil, err
	}

	estoque, err := u.carregar(ctx, input.ProdutoID, local.ID)
	if err != nil {
		return nil, err
	}

	operacao, err := estoque.Reservar(domainestoque.ReservaInput{
		Local:      local,
		Quantidade: quantidade,
		Origem:     domainestoque.OrigemReservaManual,
		UsuarioID:  input.UsuarioID,
		Observacao: input.Observacao,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.estoque.Gravar(ctx, operacao); err != nil {
		return nil, err
	}

	return operacao.ReservaCriada, nil
}

func (u *ReservaUsecaseImpl) Liberar(ctx context.Context, reservaID, usuarioID uuid.UUID) (*domainestoque.Reserva, error) {
	if err := u.exigeAdministrador(ctx, usuarioID); err != nil {
		return nil, err
	}

	reserva, err := u.reservas.Obter(ctx, reservaID)
	if err != nil {
		return nil, err
	}
	if reserva == nil {
		return nil, domain.ErroNaoEncontrado("reserva não encontrada")
	}
	if reserva.Origem == domainestoque.OrigemReservaPedidoItem {
		return nil, erroReservaDePedido()
	}

	local, err := u.locais.Obter(ctx, reserva.LocalID)
	if err != nil {
		return nil, err
	}
	if local == nil {
		return nil, domain.ErroNaoEncontrado("local da reserva não encontrado")
	}
	if err := u.exigeAcessoAosLocais(ctx, usuarioID, local); err != nil {
		return nil, err
	}

	estoque, err := u.carregar(ctx, reserva.ProdutoID, local.ID)
	if err != nil {
		return nil, err
	}

	operacao, err := estoque.LiberarReserva(reserva, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := u.estoque.Gravar(ctx, operacao); err != nil {
		return nil, err
	}

	return operacao.ReservaEncerrada, nil
}

func (u *ReservaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarReservasInput) ([]*domainestoque.Reserva, error) {
	empresaIDs, err := u.empresasVisiveis(ctx, input.UsuarioID, nil)
	if err != nil {
		return nil, err
	}

	produto, err := u.produtos.Obter(ctx, input.ProdutoID)
	if err != nil {
		return nil, err
	}
	if produto == nil {
		return nil, erroProdutoNaoEncontrado()
	}

	filtro := portsout.ReservaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		ProdutoID:       produto.ID,
		EmpresaIDs:      empresaIDs,
	}

	if input.LocalID != nil {
		local, err := u.localDoFiltro(ctx, input.UsuarioID, *input.LocalID)
		if err != nil {
			return nil, err
		}
		filtro.LocalID = &local.ID
	}

	if input.Status != "" {
		status, err := domainestoque.NovoStatusReserva(input.Status)
		if err != nil {
			return nil, err
		}
		filtro.Status = status
	}

	return u.reservas.Listar(ctx, filtro)
}
