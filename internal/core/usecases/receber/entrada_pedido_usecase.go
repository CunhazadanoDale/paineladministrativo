package receber

import (
	"context"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

const pontosBaseInteiros = 10000

var _ portsin.EntradaDoPedidoUseCase = (*EntradaDoPedidoUsecaseImpl)(nil)

type EntradaDoPedidoUsecaseImpl struct {
	pedidos portsout.PedidoRepository
	cobranca
	permissoes
}

func NewEntradaDoPedidoUsecase(
	titulos portsout.TituloReceberRepository,
	pedidos portsout.PedidoRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *EntradaDoPedidoUsecaseImpl {
	return &EntradaDoPedidoUsecaseImpl{
		pedidos:  pedidos,
		cobranca: cobranca{contas: contas},
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
			empresas: empresas,
			titulos:  titulos,
		},
	}
}

func (u *EntradaDoPedidoUsecaseImpl) GerarTituloDeEntrada(ctx context.Context, input portsin.GerarTituloDeEntradaInput) (*domainreceber.Titulo, bool, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, false, err
	}

	pedido, err := u.pedidos.Obter(ctx, input.PedidoID)
	if err != nil {
		return nil, false, err
	}
	if pedido == nil {
		return nil, false, domain.ErroNaoEncontrado("pedido não encontrado")
	}
	if !perfil.acesso.Permite(pedido.EmpresaID) {
		return nil, false, erroSemAcessoAEmpresa()
	}

	existente, err := u.tituloDeEntrada(ctx, pedido.ID)
	if err != nil || existente != nil {
		return existente, false, err
	}

	valor, err := valorDaEntrada(pedido)
	if err != nil {
		return nil, false, err
	}
	if input.ContaBancariaID != nil && *input.ContaBancariaID != uuid.Nil {
		if err := u.exigirContaDeCobranca(ctx, *input.ContaBancariaID, pedido.EmpresaID); err != nil {
			return nil, false, err
		}
	}

	geracao, err := domainreceber.NovaGeracao(domainreceber.OrigemPedido, pedido.EmpresaID, pedido.ClienteID, input.ContaBancariaID,
		[]domainreceber.ParcelaDeOrigem{{
			OrigemID:      pedido.ID,
			Numero:        "PED-" + strconv.Itoa(pedido.Numero),
			Parcela:       1,
			Vencimento:    input.Vencimento,
			ValorCentavos: valor,
		}}, input.UsuarioID, time.Now().UTC())
	if err != nil {
		return nil, false, err
	}

	resultado, err := u.titulos.GerarDeOrigem(ctx, geracao)
	if err != nil {
		return nil, false, err
	}

	criado := len(resultado.Gerados) == 1
	id := geracao.Titulos[0].ID
	if !criado {
		id = resultado.JaExistentes[0]
	}

	titulo, err := u.titulos.Obter(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if titulo == nil {
		return nil, false, domainreceber.ErroTituloNaoEncontrado()
	}

	return titulo, criado, nil
}

func (u *EntradaDoPedidoUsecaseImpl) tituloDeEntrada(ctx context.Context, pedidoID uuid.UUID) (*domainreceber.Titulo, error) {
	titulos, err := u.titulos.Listar(ctx, portsout.TituloFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 1},
		Origem:          domainreceber.OrigemPedido,
		OrigemID:        &pedidoID,
	})
	if err != nil || len(titulos) == 0 {
		return nil, err
	}

	return titulos[0], nil
}

func valorDaEntrada(pedido *domainpedido.Pedido) (int64, error) {
	switch {
	case pedido.Modelo != domainpedido.ModeloImportacao:
		return 0, domain.ErroValidacao("só o pedido de importação tem título de entrada")
	case !pedido.Status.EmAtendimento():
		return 0, domain.ErroConflito("o título de entrada só é gerado para pedido confirmado e não entregue")
	case pedido.EntradaPagaEm != nil:
		return 0, domain.ErroConflito("a entrada do pedido já foi registrada")
	}

	valor := pedido.ValorTotalCentavos * int64(pedido.Condicao.Entrada().PontosBase()) / pontosBaseInteiros
	if valor <= 0 {
		return 0, domain.ErroValidacao("pedido sem valor de entrada")
	}

	return valor, nil
}
