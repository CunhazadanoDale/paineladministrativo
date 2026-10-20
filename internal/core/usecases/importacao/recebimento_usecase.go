package importacao

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var _ portsin.RecebimentoUseCase = (*RecebimentoUsecaseImpl)(nil)

type RecebimentoUsecaseImpl struct {
	estoque portsout.EstoqueRepository
	locais  portsout.LocalRepository
	permissoes
}

type encadeamento struct {
	operacoes []*domainestoque.OperacaoEstoque
}

func (e *encadeamento) aplicar(estoque *domainestoque.EstoqueProduto, operacao *domainestoque.OperacaoEstoque, err error) (*domainestoque.OperacaoEstoque, error) {
	if err != nil {
		return nil, err
	}
	if err := estoque.Aplicar(operacao); err != nil {
		return nil, err
	}
	e.operacoes = append(e.operacoes, operacao)

	return operacao, nil
}

func NewRecebimentoUsecase(
	importacoes portsout.ImportacaoRepository,
	estoque portsout.EstoqueRepository,
	locais portsout.LocalRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *RecebimentoUsecaseImpl {
	return &RecebimentoUsecaseImpl{
		estoque: estoque,
		locais:  locais,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			empresas:    empresas,
			importacoes: importacoes,
		},
	}
}

func (u *RecebimentoUsecaseImpl) DefinirAlocacoes(ctx context.Context, input portsin.DefinirAlocacoesInput) (*portsin.Visao, error) {
	perfil, err := u.exigeLeitura(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, input.ImportacaoID)
	if err != nil {
		return nil, err
	}
	if err := importacao.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	entradas := make([]domainimportacao.AlocacaoInput, 0, len(input.Alocacoes))
	ids := make([]uuid.UUID, 0, len(input.Alocacoes))
	total := 0
	for _, alocacao := range input.Alocacoes {
		entradas = append(entradas, domainimportacao.AlocacaoInput{
			ImportacaoItemID: alocacao.ImportacaoItemID,
			PedidoItemID:     alocacao.PedidoItemID,
			Quantidade:       alocacao.Quantidade,
		})
		ids = append(ids, alocacao.PedidoItemID)
		total += alocacao.Quantidade
	}

	pedidos, err := u.itensDePedido(ctx, ids, importacao.ID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	if err := importacao.DefinirAlocacoes(input.UsuarioID, entradas, pedidos, agora); err != nil {
		return nil, err
	}

	if err := u.importacoes.Gravar(ctx, &portsout.Gravacao{
		Importacao:          importacao,
		VersaoLida:          input.Versao,
		StatusAnterior:      importacao.Status,
		SubstituirAlocacoes: true,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, input.UsuarioID, domainimportacao.AcaoAlocacaoAlterada, nil, importacao.Status,
				fmt.Sprintf("%d alocações, %d unidades para pedidos", len(entradas), total), agora),
		},
	}); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: perfil.financeiro}, nil
}

func (u *RecebimentoUsecaseImpl) Receber(ctx context.Context, input portsin.ReceberInput) (*portsin.Visao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	importacao, err := u.importacaoVisivel(ctx, perfil, input.ImportacaoID)
	if err != nil {
		return nil, err
	}
	if err := importacao.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	local, err := u.localDoRecebimento(ctx, importacao, input.LocalID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(importacao.Alocacoes))
	for _, alocacao := range importacao.Alocacoes {
		ids = append(ids, alocacao.PedidoItemID)
	}
	pedidos, err := u.itensDePedido(ctx, ids, importacao.ID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	anterior := importacao.Status
	calculo, err := importacao.Receber(domainimportacao.RecebimentoInput{
		RecebidaEm:  input.RecebidaEm,
		LocalID:     local.ID,
		CambioMicro: input.CambioMicro,
		Recebidas:   input.Recebidas,
	}, agora)
	if err != nil {
		return nil, err
	}

	cadeia := &encadeamento{}
	estoques, err := u.darEntrada(ctx, importacao, local, input.UsuarioID, cadeia, agora)
	if err != nil {
		return nil, err
	}

	reservadoPorPedido := make(map[uuid.UUID]int)
	pedidosAtendidos := make([]uuid.UUID, 0)
	observacao := "importação " + strconv.Itoa(importacao.Numero)
	for _, planejada := range importacao.PlanejarReservas(pedidos) {
		estoque := estoques[planejada.ProdutoID]
		pedidoItemID := planejada.Alocacao.PedidoItemID
		operacao, err := estoque.Reservar(domainestoque.ReservaInput{
			Local:      local,
			Quantidade: domainestoque.QuantidadeDe(planejada.Quantidade),
			Origem:     domainestoque.OrigemReservaPedidoItem,
			OrigemID:   &pedidoItemID,
			UsuarioID:  input.UsuarioID,
			Observacao: observacao,
			Agora:      agora,
		})
		operacao, err = cadeia.aplicar(estoque, operacao, err)
		if err != nil {
			return nil, err
		}
		reservaID := operacao.ReservaCriada.ID
		planejada.Alocacao.ReservaID = &reservaID
		if reservadoPorPedido[planejada.PedidoID] == 0 {
			pedidosAtendidos = append(pedidosAtendidos, planejada.PedidoID)
		}
		reservadoPorPedido[planejada.PedidoID] += planejada.Quantidade
	}

	historicosDePedido := make([]*domainpedido.Historico, 0, len(pedidosAtendidos))
	for _, pedidoID := range pedidosAtendidos {
		status := statusDoPedido(pedidos, pedidoID)
		historicosDePedido = append(historicosDePedido, domainpedido.NovoHistorico(pedidoID, input.UsuarioID,
			domainpedido.AcaoPendenciasReservadas, nil, status,
			fmt.Sprintf("reservadas %d unidades pelo recebimento da %s", reservadoPorPedido[pedidoID], observacao), agora))
	}

	if err := u.importacoes.Gravar(ctx, &portsout.Gravacao{
		Importacao:         importacao,
		VersaoLida:         input.Versao,
		StatusAnterior:     anterior,
		Recebimento:        true,
		OperacoesEstoque:   cadeia.operacoes,
		PedidosAtendidos:   pedidosAtendidos,
		HistoricosDePedido: historicosDePedido,
		Historicos: []*domainimportacao.Historico{
			domainimportacao.NovoHistorico(importacao.ID, input.UsuarioID, domainimportacao.AcaoRecebida, &anterior, importacao.Status,
				fmt.Sprintf("recebida em %s no local %s; mercadoria R$ %s, custos R$ %s; %d unidades reservadas para pedidos",
					importacao.RecebidaEm.Format("02/01/2006"), local.Nome, formatarCentesimos(calculo.ValorMercadoriaCentavos),
					formatarCentesimos(calculo.CustosCentavos), somaDe(reservadoPorPedido)), agora),
		},
	}); err != nil {
		return nil, err
	}

	return &portsin.Visao{Importacao: importacao, ComValores: true}, nil
}

func (u *RecebimentoUsecaseImpl) darEntrada(
	ctx context.Context, importacao *domainimportacao.Importacao, local *domainestoque.Local, usuarioID uuid.UUID,
	cadeia *encadeamento, agora time.Time,
) (map[uuid.UUID]*domainestoque.EstoqueProduto, error) {
	documento := "IMP-" + strconv.Itoa(importacao.Numero)
	estoques := make(map[uuid.UUID]*domainestoque.EstoqueProduto, len(importacao.Itens))
	for _, item := range importacao.Itens {
		if *item.QuantidadeRecebida == 0 {
			continue
		}

		estoque, err := u.estoque.Carregar(ctx, item.ProdutoID, []uuid.UUID{local.ID})
		if err != nil {
			return nil, err
		}
		if estoque == nil {
			return nil, domain.ErroValidacao("produto da importação não encontrado")
		}
		estoques[item.ProdutoID] = estoque

		custo := domainestoque.CustoDe(*item.CustoUnitarioCentavos)
		operacao, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
			Tipo:         domainestoque.TipoMovimentoEntrada,
			Local:        local,
			Quantidade:   domainestoque.QuantidadeDe(*item.QuantidadeRecebida),
			Custo:        &custo,
			UsuarioID:    usuarioID,
			DocumentoRef: &documento,
			Observacao:   "recebimento da importação " + strconv.Itoa(importacao.Numero),
			Agora:        agora,
		})
		operacao, err = cadeia.aplicar(estoque, operacao, err)
		if err != nil {
			return nil, err
		}
		movimentoID := operacao.Movimento.ID
		item.MovimentoID = &movimentoID
	}

	return estoques, nil
}

func (u *RecebimentoUsecaseImpl) localDoRecebimento(ctx context.Context, importacao *domainimportacao.Importacao, localID uuid.UUID) (*domainestoque.Local, error) {
	if localID == uuid.Nil {
		return nil, domain.ErroValidacao("local do recebimento é obrigatório")
	}

	local, err := u.locais.Obter(ctx, localID)
	if err != nil {
		return nil, err
	}
	if local == nil {
		return nil, domain.ErroValidacao("local do recebimento não encontrado")
	}
	if local.EmpresaID != importacao.EmpresaID {
		return nil, domain.ErroValidacao("o local do recebimento precisa ser da empresa importadora")
	}
	if !local.Ativo {
		return nil, domain.ErroValidacao("local do recebimento inativo")
	}

	return local, nil
}

func (u *RecebimentoUsecaseImpl) itensDePedido(
	ctx context.Context, ids []uuid.UUID, importacaoID uuid.UUID,
) (map[uuid.UUID]*domainimportacao.PedidoItemAlocavel, error) {
	pedidos := make(map[uuid.UUID]*domainimportacao.PedidoItemAlocavel, len(ids))
	if len(ids) == 0 {
		return pedidos, nil
	}

	itens, err := u.importacoes.ItensDePedido(ctx, ids, importacaoID)
	if err != nil {
		return nil, err
	}
	for _, item := range itens {
		pedidos[item.PedidoItemID] = item
	}

	return pedidos, nil
}

func statusDoPedido(pedidos map[uuid.UUID]*domainimportacao.PedidoItemAlocavel, pedidoID uuid.UUID) domainpedido.Status {
	for _, item := range pedidos {
		if item.PedidoID == pedidoID {
			return item.StatusPedido
		}
	}

	return ""
}

func somaDe(valores map[uuid.UUID]int) int {
	total := 0
	for _, valor := range valores {
		total += valor
	}

	return total
}
