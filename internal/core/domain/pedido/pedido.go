package pedido

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	versaoInicial                    = 1
	tamanhoMaximoNumeroPedidoCliente = 60
	tamanhoMaximoObservacoes         = 2000
	tamanhoMaximoMotivoCancelamento  = 1000
)

type Pedido struct {
	ID                     uuid.UUID
	EmpresaID              uuid.UUID
	Numero                 int
	ClienteID              uuid.UUID
	ObraID                 *uuid.UUID
	RepresentanteID        *uuid.UUID
	Modelo                 Modelo
	Condicao               Condicao
	PrazoLimite            *time.Time
	NumeroPedidoCliente    *string
	Observacoes            string
	Status                 Status
	ValorTotalCentavos     int64
	EntradaPagaEm          *time.Time
	EntradaPagaCentavos    *int64
	CanceladoMotivo        *string
	ValorADevolverCentavos *int64
	UsuarioID              uuid.UUID
	Versao                 int
	CriadoEm               time.Time
	AtualizadoEm           time.Time
	Itens                  []*Item
}

type DadosPedido struct {
	ClienteID           uuid.UUID
	ObraID              *uuid.UUID
	RepresentanteID     *uuid.UUID
	Modelo              Modelo
	PrazoLimite         *time.Time
	NumeroPedidoCliente *string
	Observacoes         string
	Itens               []ItemInput
}

type CancelamentoInput struct {
	Motivo                 string
	ValorADevolverCentavos *int64
	Entregas               []*Entrega
}

func NovoPedido(empresaID, usuarioID uuid.UUID, dados DadosPedido) (*Pedido, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa do pedido é obrigatória")
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário do pedido não informado")
	}

	agora := time.Now().UTC()
	pedido := &Pedido{
		ID:        uuid.New(),
		EmpresaID: empresaID,
		Status:    StatusProposta,
		UsuarioID: usuarioID,
		Versao:    versaoInicial,
		CriadoEm:  agora,
	}

	if err := pedido.aplicarDados(dados, agora); err != nil {
		return nil, err
	}

	return pedido, nil
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("o pedido foi alterado por outra pessoa; recarregue e tente de novo")
}

func (p *Pedido) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão do pedido é obrigatória")
	}
	if informada != p.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (p *Pedido) Alterar(dados DadosPedido, agora time.Time) error {
	if p.Status != StatusProposta {
		return domain.ErroConflito("só a proposta pode ser alterada; cancele o pedido e crie outro")
	}

	return p.aplicarDados(dados, agora)
}

func (p *Pedido) Confirmar(agora time.Time) error {
	if err := p.transicionarPara(StatusConfirmado); err != nil {
		return err
	}

	p.AtualizadoEm = agora

	return nil
}

func (p *Pedido) RegistrarEntrada(pagaEm time.Time, centavos int64, agora time.Time) error {
	switch {
	case p.Modelo != ModeloImportacao:
		return domain.ErroValidacao("só o pedido de importação tem entrada")
	case !p.Status.EmAtendimento():
		return domain.ErroConflito("a entrada só é registrada em pedido confirmado e não entregue")
	case p.EntradaPagaEm != nil:
		return domain.ErroConflito("a entrada do pedido já foi registrada")
	case centavos <= 0:
		return domain.ErroValidacao("valor da entrada deve ser maior que zero")
	case centavos > p.ValorTotalCentavos:
		return domain.ErroValidacao("valor da entrada não pode passar do valor do pedido")
	}

	data, err := dataDoFato(pagaEm, agora, feminino("data do pagamento da entrada"))
	if err != nil {
		return err
	}

	p.EntradaPagaEm = &data
	p.EntradaPagaCentavos = &centavos
	p.PrazoLimite = p.Condicao.PrazoLimiteDaEntrada(data)
	p.AtualizadoEm = agora

	return nil
}

func (p *Pedido) Cancelar(input CancelamentoInput, agora time.Time) ([]*Entrega, error) {
	motivo, err := textoObrigatorio(input.Motivo, masculino("motivo do cancelamento"), tamanhoMaximoMotivoCancelamento)
	if err != nil {
		return nil, err
	}
	if !p.Status.PodeTransicionarPara(StatusCancelado) {
		return nil, erroTransicao(p.Status, StatusCancelado)
	}

	valorADevolver, err := p.valorADevolver(input.ValorADevolverCentavos)
	if err != nil {
		return nil, err
	}

	pendentes := make([]*Entrega, 0, len(input.Entregas))
	for _, entrega := range input.Entregas {
		if entrega.PedidoID != p.ID {
			return nil, domain.ErroValidacao("entrega de outro pedido")
		}
		switch entrega.Status {
		case StatusEntregaSaiu:
			return nil, domain.ErroConflito("há entrega em trânsito; conclua a entrega antes de cancelar o pedido")
		case StatusEntregaPendente:
			pendentes = append(pendentes, entrega)
		}
	}

	for _, entrega := range pendentes {
		if err := entrega.Cancelar(agora); err != nil {
			return nil, err
		}
	}

	p.Status = StatusCancelado
	p.CanceladoMotivo = &motivo
	p.ValorADevolverCentavos = valorADevolver
	p.AtualizadoEm = agora

	return pendentes, nil
}

func (p *Pedido) RegistrarSaida(entrega *Entrega, saiuEm, agora time.Time) error {
	if entrega == nil || entrega.PedidoID != p.ID {
		return domain.ErroValidacao("entrega de outro pedido")
	}
	if !p.Status.EmAtendimento() {
		return domain.ErroConflito("o pedido não está em atendimento")
	}

	entregues := make(map[uuid.UUID]int, len(entrega.Itens))
	for _, itemEntrega := range entrega.Itens {
		item := p.Item(itemEntrega.PedidoItemID)
		if item == nil {
			return domain.ErroValidacao("item da entrega não pertence ao pedido")
		}
		entregues[item.ID] += itemEntrega.Quantidade
		if entregues[item.ID] > item.Restante() {
			return domain.ErroValidacao("quantidade da entrega maior que o restante do item")
		}
	}

	if err := entrega.sair(saiuEm, agora); err != nil {
		return err
	}

	completo := true
	for _, item := range p.Itens {
		item.QuantidadeEntregue += entregues[item.ID]
		completo = completo && item.Completo()
	}

	p.Status = StatusParcialmenteEntregue
	if completo {
		p.Status = StatusEntregue
	}
	p.AtualizadoEm = agora

	return nil
}

func (p *Pedido) Item(id uuid.UUID) *Item {
	for _, item := range p.Itens {
		if item.ID == id {
			return item
		}
	}

	return nil
}

func (p *Pedido) valorADevolver(informado *int64) (*int64, error) {
	if p.EntradaPagaCentavos == nil {
		if informado != nil && *informado != 0 {
			return nil, domain.ErroValidacao("pedido sem entrada paga não tem valor a devolver")
		}

		return nil, nil
	}

	valor := *p.EntradaPagaCentavos
	if informado != nil {
		valor = *informado
	}
	if valor < 0 || valor > *p.EntradaPagaCentavos {
		return nil, domain.ErroValidacao("valor a devolver deve estar entre zero e o valor da entrada paga")
	}

	return &valor, nil
}

func (p *Pedido) transicionarPara(proximo Status) error {
	if !p.Status.PodeTransicionarPara(proximo) {
		return erroTransicao(p.Status, proximo)
	}

	p.Status = proximo

	return nil
}

func (p *Pedido) aplicarDados(dados DadosPedido, agora time.Time) error {
	validados, err := validarDados(dados)
	if err != nil {
		return err
	}

	itens := make([]*Item, 0, len(dados.Itens))
	produtos := make(map[uuid.UUID]bool, len(dados.Itens))
	var total int64
	for _, input := range dados.Itens {
		if produtos[input.ProdutoID] {
			return domain.ErroValidacao("produto repetido no pedido; some as quantidades em um item")
		}
		produtos[input.ProdutoID] = true

		item, err := novoItem(p.ID, input, agora)
		if err != nil {
			return err
		}
		subtotal, err := item.Subtotal()
		if err != nil {
			return err
		}
		if total, err = somarCentavos(total, subtotal); err != nil {
			return err
		}
		itens = append(itens, item)
	}

	p.ClienteID = validados.ClienteID
	p.ObraID = validados.ObraID
	p.RepresentanteID = validados.RepresentanteID
	p.Modelo = validados.Modelo
	p.Condicao = CondicaoDoModelo(validados.Modelo)
	p.PrazoLimite = validados.PrazoLimite
	p.NumeroPedidoCliente = validados.NumeroPedidoCliente
	p.Observacoes = validados.Observacoes
	p.Itens = itens
	p.ValorTotalCentavos = total
	p.AtualizadoEm = agora

	return nil
}

func validarDados(dados DadosPedido) (DadosPedido, error) {
	if dados.ClienteID == uuid.Nil {
		return DadosPedido{}, domain.ErroValidacao("cliente do pedido é obrigatório")
	}
	if err := dados.Modelo.Validado(); err != nil {
		return DadosPedido{}, err
	}
	if len(dados.Itens) == 0 {
		return DadosPedido{}, domain.ErroValidacao("o pedido precisa de pelo menos um item")
	}

	prazoLimite := dataOpcional(dados.PrazoLimite)
	if prazoLimite != nil && dados.Modelo == ModeloImportacao {
		return DadosPedido{}, domain.ErroValidacao("o prazo-limite da importação é calculado pela data da entrada paga")
	}

	numeroPedidoCliente, err := textoOpcional(dados.NumeroPedidoCliente, "número do pedido do cliente", tamanhoMaximoNumeroPedidoCliente)
	if err != nil {
		return DadosPedido{}, err
	}
	observacoes, err := textoLimitado(dados.Observacoes, "observações", tamanhoMaximoObservacoes)
	if err != nil {
		return DadosPedido{}, err
	}

	return DadosPedido{
		ClienteID:           dados.ClienteID,
		ObraID:              idOpcional(dados.ObraID),
		RepresentanteID:     idOpcional(dados.RepresentanteID),
		Modelo:              dados.Modelo,
		PrazoLimite:         prazoLimite,
		NumeroPedidoCliente: numeroPedidoCliente,
		Observacoes:         observacoes,
	}, nil
}
