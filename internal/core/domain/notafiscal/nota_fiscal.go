package notafiscal

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

const versaoInicial = 1

type NotaFiscal struct {
	ID                    uuid.UUID
	Chave                 Chave
	EmpresaID             uuid.UUID
	Numero                int
	Serie                 int
	EmitidaEm             time.Time
	TipoOperacao          TipoOperacao
	Finalidade            Finalidade
	Natureza              Natureza
	NaturezaOperacao      string
	EntreEmpresas         bool
	ClienteID             *uuid.UUID
	EmpresaDestinatariaID *uuid.UUID
	DestinatarioDocumento string
	DestinatarioNome      string
	CFOPPredominante      string
	ValorTotalCentavos    int64
	ValorProdutosCentavos int64
	ValorFreteCentavos    int64
	ValorDescontoCentavos int64
	NumeroPedidoCliente   *string
	Protocolo             string
	AutorizadaEm          time.Time
	Status                Status
	PedidoID              *uuid.UUID
	EntregaID             *uuid.UUID
	ArquivoID             uuid.UUID
	UsuarioID             uuid.UUID
	Versao                int
	CriadoEm              time.Time
	AtualizadoEm          time.Time
	Itens                 []*Item
	Duplicatas            []*Duplicata
}

type Item struct {
	NotaFiscalID            uuid.UUID
	NumeroItem              int
	Codigo                  string
	Descricao               string
	ProdutoID               *uuid.UUID
	NCM                     string
	CFOP                    string
	Unidade                 string
	QuantidadeDecimilesimos int64
	ValorUnitario           string
	ValorCentavos           int64
	DescontoCentavos        int64
	PedidoCliente           *string
}

type Duplicata struct {
	ID            uuid.UUID
	NotaFiscalID  uuid.UUID
	Numero        string
	Vencimento    time.Time
	ValorCentavos int64
}

type ImportacaoInput struct {
	Documento             *Documento
	EmpresaID             uuid.UUID
	ClienteID             *uuid.UUID
	EmpresaDestinatariaID *uuid.UUID
	ProdutosPorCodigo     map[string]uuid.UUID
	ArquivoID             uuid.UUID
	UsuarioID             uuid.UUID
	Agora                 time.Time
}

func NovaNotaFiscal(input ImportacaoInput) (*NotaFiscal, error) {
	documento := input.Documento
	switch {
	case documento == nil:
		return nil, domain.ErroValidacao("documento da nota fiscal não informado")
	case input.EmpresaID == uuid.Nil:
		return nil, domain.ErroValidacao("empresa emitente não informada")
	case input.ArquivoID == uuid.Nil:
		return nil, domain.ErroValidacao("arquivo XML da nota não informado")
	case input.UsuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário da importação não informado")
	case (input.ClienteID == nil) == (input.EmpresaDestinatariaID == nil):
		return nil, domain.ErroValidacao("a nota precisa de um cliente ou de uma empresa destinatária")
	case input.EmpresaDestinatariaID != nil && *input.EmpresaDestinatariaID == input.EmpresaID:
		return nil, domain.ErroValidacao("destinatário da nota é a própria empresa emitente")
	case len(documento.Itens) == 0:
		return nil, domain.ErroValidacao("nota sem itens")
	}

	entreEmpresas := input.EmpresaDestinatariaID != nil
	cfop := documento.CFOPPredominante()
	agora := input.Agora.UTC()

	nota := &NotaFiscal{
		ID:                    uuid.New(),
		Chave:                 documento.Chave,
		EmpresaID:             input.EmpresaID,
		Numero:                documento.Numero,
		Serie:                 documento.Serie,
		EmitidaEm:             documento.EmitidaEm,
		TipoOperacao:          documento.TipoOperacao,
		Finalidade:            documento.Finalidade,
		Natureza:              ClassificarNatureza(entreEmpresas, documento.Finalidade, cfop),
		NaturezaOperacao:      documento.NaturezaOperacao,
		EntreEmpresas:         entreEmpresas,
		ClienteID:             copiarID(input.ClienteID),
		EmpresaDestinatariaID: copiarID(input.EmpresaDestinatariaID),
		DestinatarioDocumento: documento.Destinatario.Documento,
		DestinatarioNome:      documento.Destinatario.Nome,
		CFOPPredominante:      cfop,
		ValorTotalCentavos:    documento.Totais.TotalCentavos,
		ValorProdutosCentavos: documento.Totais.ProdutosCentavos,
		ValorFreteCentavos:    documento.Totais.FreteCentavos,
		ValorDescontoCentavos: documento.Totais.DescontoCentavos,
		NumeroPedidoCliente:   copiarTexto(documento.NumeroPedidoCliente),
		Protocolo:             documento.Protocolo,
		AutorizadaEm:          documento.AutorizadaEm,
		Status:                StatusAutorizada,
		ArquivoID:             input.ArquivoID,
		UsuarioID:             input.UsuarioID,
		Versao:                versaoInicial,
		CriadoEm:              agora,
		AtualizadoEm:          agora,
		Itens:                 make([]*Item, 0, len(documento.Itens)),
		Duplicatas:            make([]*Duplicata, 0, len(documento.Duplicatas)),
	}

	for _, lido := range documento.Itens {
		item := &Item{
			NotaFiscalID:            nota.ID,
			NumeroItem:              lido.NumeroItem,
			Codigo:                  lido.Codigo,
			Descricao:               lido.Descricao,
			NCM:                     lido.NCM,
			CFOP:                    lido.CFOP,
			Unidade:                 lido.Unidade,
			QuantidadeDecimilesimos: lido.QuantidadeDecimilesimos,
			ValorUnitario:           lido.ValorUnitario,
			ValorCentavos:           lido.ValorCentavos,
			DescontoCentavos:        lido.DescontoCentavos,
			PedidoCliente:           copiarTexto(lido.PedidoCliente),
		}
		if produtoID, ok := input.ProdutosPorCodigo[lido.Codigo]; ok && produtoID != uuid.Nil {
			item.ProdutoID = &produtoID
		}
		nota.Itens = append(nota.Itens, item)
	}

	for _, lida := range documento.Duplicatas {
		nota.Duplicatas = append(nota.Duplicatas, &Duplicata{
			ID:            uuid.New(),
			NotaFiscalID:  nota.ID,
			Numero:        lida.Numero,
			Vencimento:    domainpedido.DataDe(lida.Vencimento),
			ValorCentavos: lida.ValorCentavos,
		})
	}

	return nota, nil
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("a nota fiscal foi alterada por outra pessoa; recarregue e tente de novo")
}

func (n *NotaFiscal) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da nota fiscal é obrigatória")
	}
	if informada != n.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (n *NotaFiscal) Item(numero int) *Item {
	for _, item := range n.Itens {
		if item.NumeroItem == numero {
			return item
		}
	}

	return nil
}

func (n *NotaFiscal) ItensSemProduto() []*Item {
	pendentes := make([]*Item, 0)
	for _, item := range n.Itens {
		if item.ProdutoID == nil {
			pendentes = append(pendentes, item)
		}
	}

	return pendentes
}

func (n *NotaFiscal) FaltaNumeroPedido(exigeNumeroPedido bool) bool {
	return exigeNumeroPedido && n.NumeroPedidoCliente == nil
}

func (n *NotaFiscal) Vincular(pedido *domainpedido.Pedido, entrega *domainpedido.Entrega, agora time.Time) error {
	if n.Status != StatusAutorizada {
		return domain.ErroConflito("nota fiscal cancelada não pode ser vinculada")
	}

	if pedido == nil {
		if entrega != nil {
			return domain.ErroValidacao("entrega exige o pedido")
		}
		n.PedidoID = nil
		n.EntregaID = nil
		n.AtualizadoEm = agora.UTC()

		return nil
	}

	switch {
	case n.EntreEmpresas:
		return domain.ErroValidacao("nota entre empresas do grupo não se vincula a pedido")
	case pedido.EmpresaID != n.EmpresaID:
		return domain.ErroValidacao("o pedido precisa ser da empresa emitente da nota")
	case n.ClienteID == nil || pedido.ClienteID != *n.ClienteID:
		return domain.ErroValidacao("o pedido precisa ser do cliente da nota")
	case pedido.Status == domainpedido.StatusProposta || pedido.Status == domainpedido.StatusCancelado:
		return domain.ErroValidacao("só pedido confirmado, em entrega ou entregue recebe nota fiscal")
	}

	if entrega != nil {
		switch {
		case entrega.PedidoID != pedido.ID:
			return domain.ErroValidacao("a entrega não pertence ao pedido")
		case entrega.Status == domainpedido.StatusEntregaCancelada:
			return domain.ErroValidacao("entrega cancelada não recebe nota fiscal")
		}
	}

	pedidoID := pedido.ID
	n.PedidoID = &pedidoID
	n.EntregaID = nil
	if entrega != nil {
		entregaID := entrega.ID
		n.EntregaID = &entregaID
	}
	n.AtualizadoEm = agora.UTC()

	return nil
}

func (n *NotaFiscal) VincularProduto(numeroItem int, produtoID uuid.UUID, agora time.Time) (*Item, error) {
	if produtoID == uuid.Nil {
		return nil, domain.ErroValidacao("produto é obrigatório")
	}

	item := n.Item(numeroItem)
	if item == nil {
		return nil, domain.ErroNaoEncontrado("item da nota fiscal não encontrado")
	}

	item.ProdutoID = &produtoID
	n.AtualizadoEm = agora.UTC()

	return item, nil
}

func copiarID(valor *uuid.UUID) *uuid.UUID {
	if valor == nil {
		return nil
	}

	copia := *valor

	return &copia
}

func copiarTexto(valor *string) *string {
	if valor == nil {
		return nil
	}

	copia := *valor

	return &copia
}
