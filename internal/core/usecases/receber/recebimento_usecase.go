package receber

import (
	"context"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

var _ portsin.RecebimentoUseCase = (*RecebimentoUsecaseImpl)(nil)

type RecebimentoUsecaseImpl struct {
	pedidos portsout.PedidoRepository
	cobranca
	permissoes
}

func NewRecebimentoUsecase(
	titulos portsout.TituloReceberRepository,
	pedidos portsout.PedidoRepository,
	contas portsout.ContaBancariaRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *RecebimentoUsecaseImpl {
	return &RecebimentoUsecaseImpl{
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

func (u *RecebimentoUsecaseImpl) Receber(ctx context.Context, input portsin.ReceberInput) (*domainreceber.Titulo, *domainreceber.Recebimento, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, nil, err
	}
	if err := titulo.PodeReceber(); err != nil {
		return nil, nil, err
	}

	versaoLida, anterior := titulo.Versao, titulo.Status
	agora := time.Now().UTC()
	recebimento, historico, err := titulo.Receber(domainreceber.DadosRecebimento{
		Tipo:             input.Tipo,
		ContaBancariaID:  input.ContaBancariaID,
		RecebidoEm:       input.RecebidoEm,
		ValorCentavos:    input.ValorCentavos,
		JurosCentavos:    input.JurosCentavos,
		DescontoCentavos: input.DescontoCentavos,
	}, input.UsuarioID, agora)
	if err != nil {
		return nil, nil, err
	}
	if recebimento.ContaBancariaID != nil {
		if err := u.exigirContaDeCobranca(ctx, *recebimento.ContaBancariaID, titulo.EmpresaID); err != nil {
			return nil, nil, err
		}
	}

	entrada, nota, err := u.entradaDoPedido(ctx, titulo, recebimento, input.UsuarioID, agora)
	if err != nil {
		return nil, nil, err
	}
	if nota != "" {
		historico.Descricao += "; " + nota
	}

	if err := u.titulos.Receber(ctx, &portsout.Baixa{
		Titulo:         titulo,
		VersaoLida:     versaoLida,
		StatusAnterior: anterior,
		Recebimento:    recebimento,
		Historico:      historico,
		Entrada:        entrada,
	}); err != nil {
		return nil, nil, err
	}

	return titulo, recebimento, nil
}

func (u *RecebimentoUsecaseImpl) ListarRecebimentos(ctx context.Context, tituloID, usuarioID uuid.UUID) ([]*domainreceber.Recebimento, error) {
	perfil, err := u.exigeFinanceiro(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if _, err := u.tituloVisivel(ctx, perfil, tituloID); err != nil {
		return nil, err
	}

	return u.titulos.ListarRecebimentos(ctx, tituloID)
}

func (u *RecebimentoUsecaseImpl) EstornarRecebimento(ctx context.Context, input portsin.EstornarRecebimentoInput) (*domainreceber.Titulo, error) {
	titulo, err := u.tituloParaAlterar(ctx, input.UsuarioID, input.TituloID, input.Versao)
	if err != nil {
		return nil, err
	}

	recebimentos, err := u.titulos.ListarRecebimentos(ctx, titulo.ID)
	if err != nil {
		return nil, err
	}
	var recebimento *domainreceber.Recebimento
	for _, item := range recebimentos {
		if item.ID == input.RecebimentoID {
			recebimento = item
		}
	}

	versaoLida, anterior := titulo.Versao, titulo.Status
	_, historico, err := titulo.EstornarRecebimento(recebimento, input.Motivo, input.EstornadoEm, input.UsuarioID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := u.titulos.EstornarRecebimento(ctx, &portsout.EstornoDaBaixa{
		Titulo:         titulo,
		VersaoLida:     versaoLida,
		StatusAnterior: anterior,
		Recebimento:    recebimento,
		Historico:      historico,
	}); err != nil {
		return nil, err
	}

	return titulo, nil
}

func (u *RecebimentoUsecaseImpl) entradaDoPedido(
	ctx context.Context,
	titulo *domainreceber.Titulo,
	recebimento *domainreceber.Recebimento,
	usuarioID uuid.UUID,
	agora time.Time,
) (*portsout.EntradaDoPedido, string, error) {
	if titulo.Origem != domainreceber.OrigemPedido || titulo.Status != domainreceber.StatusRecebido || titulo.OrigemID == nil {
		return nil, "", nil
	}

	pedido, err := u.pedidos.Obter(ctx, *titulo.OrigemID)
	if err != nil {
		return nil, "", err
	}
	if pedido == nil {
		return nil, "", domain.ErroNaoEncontrado("pedido de origem do título não encontrado")
	}
	if pedido.EntradaPagaEm != nil {
		return nil, "a entrada do pedido já estava registrada; o pedido não mudou", nil
	}

	pago, err := u.valorRecebido(ctx, titulo.ID)
	if err != nil {
		return nil, "", err
	}

	versaoLida, anterior := pedido.Versao, pedido.Status
	err = pedido.RegistrarEntrada(recebimento.RecebidoEm, pago+recebimento.ValorCentavos, agora)
	if errors.Is(err, domain.ErrConflito) {
		return nil, "pedido " + pedido.Status.String() + ": a entrada não foi registrada", nil
	}
	if err != nil {
		return nil, "", err
	}

	recebimento.RegistrouEntradaPedido = true
	historico := domainpedido.NovoHistorico(pedido.ID, usuarioID, domainpedido.AcaoEntradaRegistrada, &anterior, pedido.Status,
		"pela baixa do título "+titulo.Numero, agora)

	return &portsout.EntradaDoPedido{
		Pedido:         pedido,
		VersaoLida:     versaoLida,
		StatusAnterior: anterior,
		Historico:      historico,
	}, "entrada paga registrada no pedido", nil
}

func (u *RecebimentoUsecaseImpl) valorRecebido(ctx context.Context, tituloID uuid.UUID) (int64, error) {
	recebimentos, err := u.titulos.ListarRecebimentos(ctx, tituloID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, item := range recebimentos {
		if !item.Estornado {
			total += item.ValorCentavos
		}
	}

	return total, nil
}
