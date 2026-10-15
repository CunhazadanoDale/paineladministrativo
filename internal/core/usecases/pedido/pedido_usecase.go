package pedido

import (
	"context"
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

var _ portsin.PedidoUseCase = (*PedidoUsecaseImpl)(nil)

type PedidoUsecaseImpl struct {
	clientes portsout.ClienteRepository
	obras    portsout.ObraRepository
	produtos portsout.ProdutoRepository
	entregas portsout.EntregaRepository
	estoqueDoPedido
	permissoes
}

func NewPedidoUsecase(
	pedidos portsout.PedidoRepository,
	entregas portsout.EntregaRepository,
	clientes portsout.ClienteRepository,
	obras portsout.ObraRepository,
	representantes portsout.RepresentanteRepository,
	produtos portsout.ProdutoRepository,
	empresas portsout.EmpresaRepository,
	estoque portsout.EstoqueRepository,
	reservas portsout.ReservaRepository,
	locais portsout.LocalRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *PedidoUsecaseImpl {
	return &PedidoUsecaseImpl{
		clientes: clientes,
		obras:    obras,
		produtos: produtos,
		entregas: entregas,
		estoqueDoPedido: estoqueDoPedido{
			estoque:  estoque,
			reservas: reservas,
			locais:   locais,
		},
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
			pedidos:        pedidos,
		},
	}
}

func (u *PedidoUsecaseImpl) Criar(ctx context.Context, input portsin.CriarPedidoInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := u.exigirEmpresaAtiva(ctx, perfil, input.EmpresaID); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, perfil, nil, input.Dados)
	if err != nil {
		return nil, err
	}

	pedido, err := domainpedido.NovoPedido(input.EmpresaID, input.UsuarioID, dados)
	if err != nil {
		return nil, err
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido: pedido,
		Novo:   true,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoCriado, nil, pedido.Status, "", pedido.CriadoEm),
		},
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) Obter(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.Pedido, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	return u.pedidoVisivel(ctx, perfil, pedidoID)
}

func (u *PedidoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarPedidosInput) ([]*domainpedido.Pedido, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa, input.RepresentanteID)
	if err != nil {
		return nil, err
	}

	filtro := portsout.PedidoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.Empresa,
		EmpresaIDs:      escopo.EmpresaIDs,
		ClienteID:       input.ClienteID,
		RepresentanteID: escopo.RepresentanteID,
		Numero:          input.Numero,
	}
	if input.Status != "" {
		if filtro.Status, err = domainpedido.NovoStatus(input.Status); err != nil {
			return nil, err
		}
	}
	if input.Modelo != "" {
		if filtro.Modelo, err = domainpedido.NovoModelo(input.Modelo); err != nil {
			return nil, err
		}
	}

	return u.pedidos.Listar(ctx, filtro)
}

func (u *PedidoUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarPedidoInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if err := pedido.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	dados, err := u.dadosValidados(ctx, perfil, pedido, input.Dados)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	if err := pedido.Alterar(dados, agora); err != nil {
		return nil, err
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:          pedido,
		VersaoLida:      input.Versao,
		StatusAnterior:  domainpedido.StatusProposta,
		SubstituirItens: true,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoAlterado, nil, pedido.Status, "", agora),
		},
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) Confirmar(ctx context.Context, input portsin.TransicaoPedidoInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if err := pedido.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	anterior := pedido.Status
	if err := pedido.Confirmar(agora); err != nil {
		return nil, err
	}

	operacoes, reservado, err := u.reservarFaltantes(ctx, pedido, input.UsuarioID, agora)
	if err != nil {
		return nil, err
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:           pedido,
		VersaoLida:       input.Versao,
		StatusAnterior:   anterior,
		OperacoesEstoque: operacoes,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoConfirmado, &anterior, pedido.Status,
				detalheDaReserva(reservado, pedido), agora),
		},
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) ReservarPendencias(ctx context.Context, input portsin.TransicaoPedidoInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if err := pedido.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}
	if !pedido.Status.EmAtendimento() {
		return nil, domain.ErroConflito("só pedido confirmado e não entregue reserva pendências")
	}

	agora := time.Now().UTC()
	operacoes, reservado, err := u.reservarFaltantes(ctx, pedido, input.UsuarioID, agora)
	if err != nil {
		return nil, err
	}
	if reservado == 0 {
		return pedido, nil
	}

	pedido.AtualizadoEm = agora
	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:           pedido,
		VersaoLida:       input.Versao,
		StatusAnterior:   pedido.Status,
		OperacoesEstoque: operacoes,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoPendenciasReservadas, nil, pedido.Status,
				detalheDaReserva(reservado, pedido), agora),
		},
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) RegistrarEntrada(ctx context.Context, input portsin.RegistrarEntradaInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if err := pedido.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	if err := pedido.RegistrarEntrada(input.PagaEm, input.ValorCentavos, agora); err != nil {
		return nil, err
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:         pedido,
		VersaoLida:     input.Versao,
		StatusAnterior: pedido.Status,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoEntradaRegistrada, nil, pedido.Status,
				fmt.Sprintf("entrada de %d centavos paga em %s", input.ValorCentavos, pedido.EntradaPagaEm.Format(time.DateOnly)), agora),
		},
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) Cancelar(ctx context.Context, input portsin.CancelarPedidoInput) (*domainpedido.Pedido, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}
	if err := pedido.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}
	if perfil.restritoAoRepresentante() && pedido.Status != domainpedido.StatusProposta {
		return nil, domain.ErroPermissao("o representante só cancela a própria proposta")
	}

	entregas, err := u.entregas.ListarDoPedido(ctx, pedido.ID)
	if err != nil {
		return nil, err
	}
	versoes := make(map[uuid.UUID]int, len(entregas))
	for _, entrega := range entregas {
		versoes[entrega.ID] = entrega.Versao
	}

	agora := time.Now().UTC()
	anterior := pedido.Status
	canceladas, err := pedido.Cancelar(domainpedido.CancelamentoInput{
		Motivo:                 input.Motivo,
		ValorADevolverCentavos: input.ValorADevolverCentavos,
		Entregas:               entregas,
	}, agora)
	if err != nil {
		return nil, err
	}

	operacoes, err := u.liberarReservas(ctx, pedido, agora)
	if err != nil {
		return nil, err
	}

	alteradas := make([]portsout.EntregaAlterada, 0, len(canceladas))
	historicos := []*domainpedido.Historico{
		domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoCancelado, &anterior, pedido.Status, *pedido.CanceladoMotivo, agora),
	}
	for _, entrega := range canceladas {
		alteradas = append(alteradas, portsout.EntregaAlterada{
			Entrega:        entrega,
			VersaoLida:     versoes[entrega.ID],
			StatusAnterior: domainpedido.StatusEntregaPendente,
		})
		historicos = append(historicos, domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoEntregaCancelada,
			nil, pedido.Status, "entrega cancelada com o pedido", agora))
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:            pedido,
		VersaoLida:        input.Versao,
		StatusAnterior:    anterior,
		EntregasAlteradas: alteradas,
		OperacoesEstoque:  operacoes,
		Historicos:        historicos,
	}); err != nil {
		return nil, err
	}

	return pedido, nil
}

func (u *PedidoUsecaseImpl) ObterComReservas(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.Pedido, []*domainestoque.Reserva, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, pedidoID)
	if err != nil {
		return nil, nil, err
	}

	porItem, err := u.reservasPorItem(ctx, pedido)
	if err != nil {
		return nil, nil, err
	}

	reservas := make([]*domainestoque.Reserva, 0)
	for _, item := range pedido.Itens {
		reservas = append(reservas, porItem[item.ID]...)
	}

	return pedido, reservas, nil
}

func (u *PedidoUsecaseImpl) ListarHistorico(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]*domainpedido.Historico, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if _, err := u.pedidoVisivel(ctx, perfil, pedidoID); err != nil {
		return nil, err
	}

	return u.pedidos.ListarHistorico(ctx, pedidoID)
}

func (u *PedidoUsecaseImpl) exigirEmpresaAtiva(ctx context.Context, perfil perfilDoUsuario, empresaID uuid.UUID) error {
	if empresaID == uuid.Nil {
		return domain.ErroValidacao("empresa do pedido é obrigatória")
	}
	if !perfil.acesso.Permite(empresaID) {
		return erroSemAcessoAEmpresa()
	}

	empresa, err := u.empresas.Obter(ctx, empresaID)
	if err != nil {
		return err
	}
	if empresa == nil {
		return domain.ErroValidacao("empresa do pedido não encontrada")
	}
	if !empresa.Ativa {
		return domain.ErroValidacao("empresa inativa não recebe pedido novo")
	}

	return nil
}

func (u *PedidoUsecaseImpl) dadosValidados(
	ctx context.Context, perfil perfilDoUsuario, atual *domainpedido.Pedido, input portsin.DadosPedidoInput,
) (domainpedido.DadosPedido, error) {
	modelo, err := domainpedido.NovoModelo(input.Modelo)
	if err != nil {
		return domainpedido.DadosPedido{}, err
	}

	cliente, err := u.clientes.Obter(ctx, input.ClienteID)
	if err != nil {
		return domainpedido.DadosPedido{}, err
	}
	if cliente == nil {
		return domainpedido.DadosPedido{}, domain.ErroValidacao("cliente do pedido não encontrado")
	}
	clienteTrocado := atual == nil || atual.ClienteID != cliente.ID
	if clienteTrocado && !cliente.Ativo {
		return domainpedido.DadosPedido{}, domain.ErroValidacao("cliente inativo não recebe pedido novo")
	}
	clienteDeOutro := cliente.RepresentanteID == nil || perfil.representanteID == nil || *cliente.RepresentanteID != *perfil.representanteID
	if clienteTrocado && perfil.restritoAoRepresentante() && clienteDeOutro {
		return domainpedido.DadosPedido{}, domain.ErroPermissao("cliente de outro representante")
	}

	obraID, err := u.obraValidada(ctx, atual, cliente.ID, input.ObraID)
	if err != nil {
		return domainpedido.DadosPedido{}, err
	}

	representanteID := input.RepresentanteID
	if representanteID != nil && *representanteID == uuid.Nil {
		representanteID = nil
	}
	if representanteID == nil && atual == nil {
		representanteID = cliente.RepresentanteID
	}
	if perfil.restritoAoRepresentante() {
		if input.RepresentanteID != nil && *input.RepresentanteID != uuid.Nil && *input.RepresentanteID != *perfil.representanteID {
			return domainpedido.DadosPedido{}, domain.ErroPermissao("o representante só registra pedidos em seu nome")
		}
		representanteID = perfil.representanteID
	}
	if err := u.exigirRepresentanteAtivo(ctx, atual, representanteID); err != nil {
		return domainpedido.DadosPedido{}, err
	}

	itens, err := u.itensValidados(ctx, atual, input.Itens)
	if err != nil {
		return domainpedido.DadosPedido{}, err
	}

	return domainpedido.DadosPedido{
		ClienteID:           cliente.ID,
		ObraID:              obraID,
		RepresentanteID:     representanteID,
		Modelo:              modelo,
		PrazoLimite:         input.PrazoLimite,
		NumeroPedidoCliente: input.NumeroPedidoCliente,
		Observacoes:         input.Observacoes,
		Itens:               itens,
	}, nil
}

func (u *PedidoUsecaseImpl) obraValidada(ctx context.Context, atual *domainpedido.Pedido, clienteID uuid.UUID, obraID *uuid.UUID) (*uuid.UUID, error) {
	if obraID == nil || *obraID == uuid.Nil {
		return nil, nil
	}

	obra, err := u.obras.Obter(ctx, *obraID)
	if err != nil {
		return nil, err
	}
	if obra == nil || obra.ClienteID != clienteID {
		return nil, domain.ErroValidacao("obra não encontrada para o cliente do pedido")
	}
	mantida := atual != nil && atual.ObraID != nil && *atual.ObraID == obra.ID
	if !mantida && !obra.Ativo {
		return nil, domain.ErroValidacao("obra inativa não recebe pedido novo")
	}

	return &obra.ID, nil
}

func (u *PedidoUsecaseImpl) exigirRepresentanteAtivo(ctx context.Context, atual *domainpedido.Pedido, representanteID *uuid.UUID) error {
	if representanteID == nil {
		return nil
	}
	if atual != nil && atual.RepresentanteID != nil && *atual.RepresentanteID == *representanteID {
		return nil
	}

	representante, err := u.representantes.Obter(ctx, *representanteID)
	if err != nil {
		return err
	}
	if representante == nil {
		return domain.ErroValidacao("representante do pedido não encontrado")
	}
	if !representante.Ativo {
		return domain.ErroValidacao("representante inativo não recebe pedido novo")
	}

	return nil
}

func (u *PedidoUsecaseImpl) itensValidados(ctx context.Context, atual *domainpedido.Pedido, itens []portsin.ItemInput) ([]domainpedido.ItemInput, error) {
	jaNoPedido := make(map[uuid.UUID]bool)
	if atual != nil {
		for _, item := range atual.Itens {
			jaNoPedido[item.ProdutoID] = true
		}
	}

	validados := make([]domainpedido.ItemInput, 0, len(itens))
	for _, item := range itens {
		if item.ProdutoID != uuid.Nil && !jaNoPedido[item.ProdutoID] {
			produto, err := u.produtos.Obter(ctx, item.ProdutoID)
			if err != nil {
				return nil, err
			}
			if produto == nil {
				return nil, domain.ErroValidacao("produto do pedido não encontrado")
			}
			if !produto.Ativo {
				return nil, domain.ErroValidacao(fmt.Sprintf("produto %q está inativo e não entra em pedido novo", produto.Nome))
			}
		}

		validados = append(validados, domainpedido.ItemInput{
			ProdutoID:             item.ProdutoID,
			Quantidade:            item.Quantidade,
			PrecoUnitarioCentavos: item.PrecoUnitarioCentavos,
		})
	}

	return validados, nil
}

func detalheDaReserva(reservado int, pedido *domainpedido.Pedido) string {
	restante := 0
	for _, item := range pedido.Itens {
		restante += item.Restante()
	}

	return fmt.Sprintf("reservadas %d unidades; restante a entregar %d", reservado, restante)
}
