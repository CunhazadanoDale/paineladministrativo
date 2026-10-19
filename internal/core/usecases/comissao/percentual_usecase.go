package comissao

import (
	"context"
	"time"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

var _ portsin.PercentualUseCase = (*PercentualUsecaseImpl)(nil)

type PercentualUsecaseImpl struct {
	comissoes portsout.ComissaoRepository
	pedidos   portsout.PedidoRepository
	permissoes
}

func NewPercentualUsecase(
	comissoes portsout.ComissaoRepository,
	pedidos portsout.PedidoRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *PercentualUsecaseImpl {
	return &PercentualUsecaseImpl{
		comissoes: comissoes,
		pedidos:   pedidos,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
		},
	}
}

func (u *PercentualUsecaseImpl) Obter(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domaincomissao.PercentualPedido, error) {
	perfil, err := u.exigeLeituraDoPercentual(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoComRepresentante(ctx, perfil, pedidoID)
	if err != nil {
		return nil, err
	}

	atual, err := u.comissoes.ObterPercentual(ctx, pedido.ID)
	if err != nil {
		return nil, err
	}
	if atual.VigentePara(*pedido.RepresentanteID) {
		return atual, nil
	}

	representante, err := u.representantes.Obter(ctx, *pedido.RepresentanteID)
	if err != nil {
		return nil, err
	}

	return domaincomissao.PercentualSugerido(pedido.ID, pedido.Modelo, representante, usuarioID, time.Now().UTC())
}

func (u *PercentualUsecaseImpl) Definir(ctx context.Context, input portsin.DefinirPercentualInput) (*domaincomissao.PercentualPedido, error) {
	perfil, err := u.exigeDefinicaoDoPercentual(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoComRepresentante(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if pedido.Status == domainpedido.StatusCancelado {
		return nil, domaincomissao.ErroPedidoCancelado()
	}

	atual, err := u.comissoes.ObterPercentual(ctx, pedido.ID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	if !atual.VigentePara(*pedido.RepresentanteID) {
		novo, err := domaincomissao.NovoPercentualManual(pedido.ID, *pedido.RepresentanteID, input.UsuarioID, input.PontosBase, agora)
		if err != nil {
			return nil, err
		}
		if err := u.comissoes.DefinirPercentual(ctx, novo, nil); err != nil {
			return nil, err
		}

		return novo, nil
	}

	versaoLida := 0
	if input.Versao != nil {
		versaoLida = *input.Versao
	}
	if err := atual.ConferirVersao(versaoLida); err != nil {
		return nil, err
	}
	if err := atual.Redefinir(input.PontosBase, input.UsuarioID, agora); err != nil {
		return nil, err
	}
	if err := u.comissoes.DefinirPercentual(ctx, atual, &versaoLida); err != nil {
		return nil, err
	}

	return atual, nil
}

func (u *PercentualUsecaseImpl) pedidoComRepresentante(ctx context.Context, perfil perfilDoUsuario, pedidoID uuid.UUID) (*domainpedido.Pedido, error) {
	pedido, err := u.pedidos.Obter(ctx, pedidoID)
	if err != nil {
		return nil, err
	}
	if pedido == nil {
		return nil, domaincomissao.ErroPedidoNaoEncontrado()
	}
	if err := perfil.conferirPedido(pedido); err != nil {
		return nil, err
	}
	if pedido.RepresentanteID == nil {
		return nil, domaincomissao.ErroPedidoSemRepresentante()
	}

	return pedido, nil
}
