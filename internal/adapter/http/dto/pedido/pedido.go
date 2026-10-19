package pedido

import (
	"time"

	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

type ItemRequest struct {
	ProdutoID             uuid.UUID `json:"produto_id"`
	Quantidade            int       `json:"quantidade"`
	PrecoUnitarioCentavos int64     `json:"preco_unitario_centavos"`
}

type DadosPedidoRequest struct {
	ClienteID             uuid.UUID     `json:"cliente_id"`
	ObraID                *uuid.UUID    `json:"obra_id"`
	RepresentanteID       *uuid.UUID    `json:"representante_id"`
	Modelo                string        `json:"modelo"`
	EntradaBP             *int          `json:"entrada_bp"`
	PrazoBoletoDias       *int          `json:"prazo_boleto_dias"`
	PrazoEntregaDias      *int          `json:"prazo_entrega_dias"`
	Validade              *string       `json:"validade"`
	FreteTipo             string        `json:"frete_tipo"`
	FreteEstimadoCentavos *int64        `json:"frete_estimado_centavos"`
	PrazoLimite           *string       `json:"prazo_limite"`
	NumeroPedidoCliente   *string       `json:"numero_pedido_cliente"`
	Observacoes           string        `json:"observacoes"`
	Itens                 []ItemRequest `json:"itens"`
}

func (r DadosPedidoRequest) ParaDados() (portsin.DadosPedidoInput, error) {
	prazo, err := paraDataOpcional(r.PrazoLimite, "prazo-limite")
	if err != nil {
		return portsin.DadosPedidoInput{}, err
	}
	validade, err := paraDataOpcional(r.Validade, "validade")
	if err != nil {
		return portsin.DadosPedidoInput{}, err
	}

	itens := make([]portsin.ItemInput, 0, len(r.Itens))
	for _, item := range r.Itens {
		itens = append(itens, portsin.ItemInput{
			ProdutoID:             item.ProdutoID,
			Quantidade:            item.Quantidade,
			PrecoUnitarioCentavos: item.PrecoUnitarioCentavos,
		})
	}

	return portsin.DadosPedidoInput{
		ClienteID:             r.ClienteID,
		ObraID:                r.ObraID,
		RepresentanteID:       r.RepresentanteID,
		Modelo:                r.Modelo,
		EntradaBP:             r.EntradaBP,
		PrazoBoletoDias:       r.PrazoBoletoDias,
		PrazoEntregaDias:      r.PrazoEntregaDias,
		Validade:              validade,
		FreteTipo:             r.FreteTipo,
		FreteEstimadoCentavos: r.FreteEstimadoCentavos,
		PrazoLimite:           prazo,
		NumeroPedidoCliente:   r.NumeroPedidoCliente,
		Observacoes:           r.Observacoes,
		Itens:                 itens,
	}, nil
}

type CriarPedidoRequest struct {
	DadosPedidoRequest
	EmpresaID uuid.UUID `json:"empresa_id"`
}

type AlterarPedidoRequest struct {
	DadosPedidoRequest
	Versao int `json:"versao"`
}

type VersaoRequest struct {
	Versao int `json:"versao"`
}

type EntradaRequest struct {
	Versao        int    `json:"versao"`
	PagoEm        string `json:"pago_em"`
	ValorCentavos int64  `json:"valor_centavos"`
}

type CancelarPedidoRequest struct {
	Versao                 int    `json:"versao"`
	Motivo                 string `json:"motivo"`
	ValorADevolverCentavos *int64 `json:"valor_a_devolver_centavos"`
}

type CondicaoResponse struct {
	EntradaBP        int  `json:"entrada_bp"`
	PrazoBoletoDias  *int `json:"prazo_boleto_dias"`
	PrazoEntregaDias *int `json:"prazo_entrega_dias"`
}

type ItemResponse struct {
	ID                    uuid.UUID `json:"id"`
	ProdutoID             uuid.UUID `json:"produto_id"`
	Quantidade            int       `json:"quantidade"`
	PrecoUnitarioCentavos int64     `json:"preco_unitario_centavos"`
	PrecoTabelaCentavos   *int64    `json:"preco_tabela_centavos"`
	DescontoCentavos      *int64    `json:"desconto_unitario_centavos"`
	SubtotalCentavos      int64     `json:"subtotal_centavos"`
	QuantidadeEntregue    int       `json:"quantidade_entregue"`
	Restante              int       `json:"restante"`
	Reservado             *int      `json:"reservado,omitempty"`
	FaltaCobrir           *int      `json:"falta_cobrir,omitempty"`
}

type PedidoResponse struct {
	ID                     uuid.UUID        `json:"id"`
	EmpresaID              uuid.UUID        `json:"empresa_id"`
	Numero                 int              `json:"numero"`
	ClienteID              uuid.UUID        `json:"cliente_id"`
	ObraID                 *uuid.UUID       `json:"obra_id"`
	RepresentanteID        *uuid.UUID       `json:"representante_id"`
	Modelo                 string           `json:"modelo"`
	Condicao               CondicaoResponse `json:"condicao"`
	Validade               *string          `json:"validade"`
	Vencida                bool             `json:"vencida"`
	Revisao                int              `json:"revisao"`
	FreteTipo              *string          `json:"frete_tipo"`
	FreteEstimadoCentavos  *int64           `json:"frete_estimado_centavos"`
	PrazoLimite            *time.Time       `json:"prazo_limite"`
	NumeroPedidoCliente    *string          `json:"numero_pedido_cliente"`
	Observacoes            string           `json:"observacoes"`
	Status                 string           `json:"status"`
	ValorTotalCentavos     int64            `json:"valor_total_centavos"`
	EntradaPagaEm          *time.Time       `json:"entrada_paga_em"`
	EntradaPagaCentavos    *int64           `json:"entrada_paga_centavos"`
	CanceladoMotivo        *string          `json:"cancelado_motivo"`
	ValorADevolverCentavos *int64           `json:"valor_a_devolver_centavos"`
	UsuarioID              uuid.UUID        `json:"usuario_id"`
	Versao                 int              `json:"versao"`
	CriadoEm               time.Time        `json:"criado_em"`
	AtualizadoEm           time.Time        `json:"atualizado_em"`
	Itens                  []ItemResponse   `json:"itens"`
}

type PedidoDetalheResponse struct {
	PedidoResponse
	Reservas []estoquedto.ReservaResponse `json:"reservas"`
}

type HistoricoResponse struct {
	ID             uuid.UUID `json:"id"`
	UsuarioID      uuid.UUID `json:"usuario_id"`
	Acao           string    `json:"acao"`
	StatusAnterior *string   `json:"status_anterior"`
	StatusNovo     string    `json:"status_novo"`
	Detalhe        string    `json:"detalhe"`
	CriadoEm       time.Time `json:"criado_em"`
}

func NovoPedidoResponse(item *domainpedido.Pedido) PedidoResponse {
	resposta := PedidoResponse{
		ID:              item.ID,
		EmpresaID:       item.EmpresaID,
		Numero:          item.Numero,
		ClienteID:       item.ClienteID,
		ObraID:          item.ObraID,
		RepresentanteID: item.RepresentanteID,
		Modelo:          item.Modelo.String(),
		Condicao: CondicaoResponse{
			EntradaBP:        item.Condicao.Entrada().PontosBase(),
			PrazoBoletoDias:  item.Condicao.PrazoBoletoDias(),
			PrazoEntregaDias: item.Condicao.PrazoEntregaDias(),
		},
		Vencida:                item.Vencida(time.Now().UTC()),
		Revisao:                item.Revisao,
		PrazoLimite:            item.PrazoLimite,
		NumeroPedidoCliente:    item.NumeroPedidoCliente,
		Observacoes:            item.Observacoes,
		Status:                 item.Status.String(),
		ValorTotalCentavos:     item.ValorTotalCentavos,
		EntradaPagaEm:          item.EntradaPagaEm,
		EntradaPagaCentavos:    item.EntradaPagaCentavos,
		CanceladoMotivo:        item.CanceladoMotivo,
		ValorADevolverCentavos: item.ValorADevolverCentavos,
		UsuarioID:              item.UsuarioID,
		Versao:                 item.Versao,
		CriadoEm:               item.CriadoEm,
		AtualizadoEm:           item.AtualizadoEm,
		Itens:                  make([]ItemResponse, 0, len(item.Itens)),
	}
	if item.Validade != nil {
		validade := item.Validade.Format(formatoData)
		resposta.Validade = &validade
	}
	if item.Frete != nil {
		tipo := item.Frete.Tipo().String()
		resposta.FreteTipo = &tipo
		resposta.FreteEstimadoCentavos = item.Frete.Centavos()
	}
	for _, pedidoItem := range item.Itens {
		subtotal, _ := pedidoItem.Subtotal()
		itemResposta := ItemResponse{
			ID:                    pedidoItem.ID,
			ProdutoID:             pedidoItem.ProdutoID,
			Quantidade:            pedidoItem.Quantidade,
			PrecoUnitarioCentavos: pedidoItem.PrecoUnitario.Centavos(),
			DescontoCentavos:      pedidoItem.DescontoUnitarioCentavos(),
			SubtotalCentavos:      subtotal,
			QuantidadeEntregue:    pedidoItem.QuantidadeEntregue,
			Restante:              pedidoItem.Restante(),
		}
		if pedidoItem.PrecoTabela != nil {
			precoTabela := pedidoItem.PrecoTabela.Centavos()
			itemResposta.PrecoTabelaCentavos = &precoTabela
		}
		resposta.Itens = append(resposta.Itens, itemResposta)
	}

	return resposta
}

func NovoPedidoResponses(itens []*domainpedido.Pedido) []PedidoResponse {
	respostas := make([]PedidoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoPedidoResponse(item))
	}

	return respostas
}

func NovoPedidoDetalheResponse(item *domainpedido.Pedido, reservas []*domainestoque.Reserva) PedidoDetalheResponse {
	reservadoPorItem := make(map[uuid.UUID]int, len(item.Itens))
	for _, reserva := range reservas {
		if reserva.OrigemID != nil {
			reservadoPorItem[*reserva.OrigemID] += reserva.Quantidade
		}
	}

	detalhe := PedidoDetalheResponse{
		PedidoResponse: NovoPedidoResponse(item),
		Reservas:       estoquedto.NovaReservaResponses(reservas),
	}
	for indice, pedidoItem := range item.Itens {
		reservado := reservadoPorItem[pedidoItem.ID]
		falta := pedidoItem.FaltaCobrir(reservado)
		detalhe.Itens[indice].Reservado = &reservado
		detalhe.Itens[indice].FaltaCobrir = &falta
	}

	return detalhe
}

func NovoHistoricoResponses(itens []*domainpedido.Historico) []HistoricoResponse {
	respostas := make([]HistoricoResponse, 0, len(itens))
	for _, item := range itens {
		resposta := HistoricoResponse{
			ID:         item.ID,
			UsuarioID:  item.UsuarioID,
			Acao:       item.Acao.String(),
			StatusNovo: item.StatusNovo.String(),
			Detalhe:    item.Detalhe,
			CriadoEm:   item.CriadoEm,
		}
		if item.StatusAnterior != nil {
			anterior := item.StatusAnterior.String()
			resposta.StatusAnterior = &anterior
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}
