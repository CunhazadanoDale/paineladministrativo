package pedido

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

const tamanhoMaximoTransportadora = 160

type StatusEntrega string

const (
	StatusEntregaPendente  StatusEntrega = "pendente"
	StatusEntregaSaiu      StatusEntrega = "saiu"
	StatusEntregaEntregue  StatusEntrega = "entregue"
	StatusEntregaCancelada StatusEntrega = "cancelada"
)

var transicoesEntrega = map[StatusEntrega][]StatusEntrega{
	StatusEntregaPendente:  {StatusEntregaSaiu, StatusEntregaCancelada},
	StatusEntregaSaiu:      {StatusEntregaEntregue},
	StatusEntregaEntregue:  {},
	StatusEntregaCancelada: {},
}

func NovoStatusEntrega(valor string) (StatusEntrega, error) {
	status := StatusEntrega(valor)

	if _, ok := transicoesEntrega[status]; !ok {
		return "", domain.ErroValidacao("status da entrega deve ser pendente, saiu, entregue ou cancelada")
	}

	return status, nil
}

func StatusEntregaDe(valor string) StatusEntrega {
	return StatusEntrega(valor)
}

func (s StatusEntrega) String() string {
	return string(s)
}

func (s StatusEntrega) PodeTransicionarPara(proximo StatusEntrega) bool {
	for _, permitido := range transicoesEntrega[s] {
		if permitido == proximo {
			return true
		}
	}

	return false
}

type Entrega struct {
	ID             uuid.UUID
	PedidoID       uuid.UUID
	Status         StatusEntrega
	Transportadora *string
	Frete          Frete
	Previsao       *time.Time
	SaiuEm         *time.Time
	EntregueEm     *time.Time
	UsuarioID      uuid.UUID
	Versao         int
	CriadoEm       time.Time
	AtualizadoEm   time.Time
	Itens          []*EntregaItem
}

type EntregaItem struct {
	EntregaID    uuid.UUID
	PedidoItemID uuid.UUID
	LocalID      uuid.UUID
	Quantidade   int
	MovimentoID  *uuid.UUID
}

type DadosEntrega struct {
	Transportadora *string
	Frete          Frete
	Previsao       *time.Time
	Itens          []EntregaItemInput
}

type EntregaItemInput struct {
	PedidoItemID uuid.UUID
	Local        *domainestoque.Local
	Quantidade   int
}

func (p *Pedido) NovaEntrega(usuarioID uuid.UUID, dados DadosEntrega, programado map[uuid.UUID]int, agora time.Time) (*Entrega, error) {
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário da entrega não informado")
	}
	if !p.Status.EmAtendimento() {
		return nil, domain.ErroConflito("só pedido confirmado e não entregue recebe entrega")
	}
	if err := dados.Frete.Tipo().Validado(); err != nil {
		return nil, err
	}
	if len(dados.Itens) == 0 {
		return nil, domain.ErroValidacao("a entrega precisa de pelo menos um item")
	}

	transportadora, err := textoOpcional(dados.Transportadora, "transportadora", tamanhoMaximoTransportadora)
	if err != nil {
		return nil, err
	}

	entrega := &Entrega{
		ID:             uuid.New(),
		PedidoID:       p.ID,
		Status:         StatusEntregaPendente,
		Transportadora: transportadora,
		Frete:          FreteDe(dados.Frete.Tipo(), dados.Frete.Centavos()),
		Previsao:       dataOpcional(dados.Previsao),
		UsuarioID:      usuarioID,
		Versao:         versaoInicial,
		CriadoEm:       agora,
		AtualizadoEm:   agora,
		Itens:          make([]*EntregaItem, 0, len(dados.Itens)),
	}

	incluidos := make(map[uuid.UUID]bool, len(dados.Itens))
	for _, input := range dados.Itens {
		item, err := p.validarItemDaEntrega(input, programado[input.PedidoItemID])
		if err != nil {
			return nil, err
		}
		if incluidos[item.ID] {
			return nil, domain.ErroValidacao("item repetido na entrega")
		}
		incluidos[item.ID] = true

		entrega.Itens = append(entrega.Itens, &EntregaItem{
			EntregaID:    entrega.ID,
			PedidoItemID: item.ID,
			LocalID:      input.Local.ID,
			Quantidade:   input.Quantidade,
		})
	}

	return entrega, nil
}

func (p *Pedido) validarItemDaEntrega(input EntregaItemInput, programado int) (*Item, error) {
	item := p.Item(input.PedidoItemID)
	switch {
	case item == nil:
		return nil, domain.ErroValidacao("item da entrega não pertence ao pedido")
	case input.Local == nil:
		return nil, domain.ErroValidacao("local de saída do item é obrigatório")
	case input.Local.EmpresaID != p.EmpresaID:
		return nil, domain.ErroValidacao("o local de saída precisa ser da empresa do pedido")
	case input.Quantidade <= 0:
		return nil, domain.ErroValidacao("quantidade do item na entrega deve ser maior que zero")
	case input.Quantidade > item.Restante()-programado:
		return nil, domain.ErroValidacao(fmt.Sprintf(
			"quantidade maior que o restante do item ainda não programado (%d)", max(item.Restante()-programado, 0),
		))
	}

	return item, nil
}

func ErroVersaoEntregaDivergente() error {
	return domain.ErroConflito("a entrega foi alterada por outra pessoa; recarregue e tente de novo")
}

func (e *Entrega) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da entrega é obrigatória")
	}
	if informada != e.Versao {
		return ErroVersaoEntregaDivergente()
	}

	return nil
}

func (e *Entrega) Concluir(entregueEm, agora time.Time) error {
	if !e.Status.PodeTransicionarPara(StatusEntregaEntregue) {
		return erroTransicaoEntrega(e.Status, StatusEntregaEntregue)
	}

	data, err := dataDoFato(entregueEm, agora, feminino("data da entrega"))
	if err != nil {
		return err
	}
	if e.SaiuEm != nil && data.Before(*e.SaiuEm) {
		return domain.ErroValidacao("data da entrega não pode ser anterior à data de saída")
	}

	e.Status = StatusEntregaEntregue
	e.EntregueEm = &data
	e.AtualizadoEm = agora

	return nil
}

func (e *Entrega) Cancelar(agora time.Time) error {
	if !e.Status.PodeTransicionarPara(StatusEntregaCancelada) {
		return erroTransicaoEntrega(e.Status, StatusEntregaCancelada)
	}

	e.Status = StatusEntregaCancelada
	e.AtualizadoEm = agora

	return nil
}

func (e *Entrega) QuantidadePorItem() map[uuid.UUID]int {
	quantidades := make(map[uuid.UUID]int, len(e.Itens))
	for _, item := range e.Itens {
		quantidades[item.PedidoItemID] += item.Quantidade
	}

	return quantidades
}

func (e *Entrega) sair(saiuEm, agora time.Time) error {
	if !e.Status.PodeTransicionarPara(StatusEntregaSaiu) {
		return erroTransicaoEntrega(e.Status, StatusEntregaSaiu)
	}

	data, err := dataDoFato(saiuEm, agora, feminino("data de saída"))
	if err != nil {
		return err
	}

	e.Status = StatusEntregaSaiu
	e.SaiuEm = &data
	e.AtualizadoEm = agora

	return nil
}

func ProgramadoPorItem(entregas []*Entrega) map[uuid.UUID]int {
	programado := make(map[uuid.UUID]int)
	for _, entrega := range entregas {
		if entrega.Status != StatusEntregaPendente {
			continue
		}
		for itemID, quantidade := range entrega.QuantidadePorItem() {
			programado[itemID] += quantidade
		}
	}

	return programado
}

func erroTransicaoEntrega(atual, proximo StatusEntrega) error {
	return domain.ErroConflito(fmt.Sprintf("não é possível mudar a entrega de %q para %q", atual, proximo))
}
