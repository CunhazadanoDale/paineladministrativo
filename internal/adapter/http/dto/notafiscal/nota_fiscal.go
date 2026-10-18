package notafiscal

import (
	"fmt"
	"time"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/google/uuid"
)

const escalaDaQuantidade = 10000

type ImportarRequest struct {
	ArquivoID uuid.UUID  `json:"arquivo_id"`
	PedidoID  *uuid.UUID `json:"pedido_id"`
	EntregaID *uuid.UUID `json:"entrega_id"`
}

func (r ImportarRequest) ParaInput(usuarioID uuid.UUID) portsin.ImportarInput {
	return portsin.ImportarInput{UsuarioID: usuarioID, ArquivoID: r.ArquivoID, PedidoID: r.PedidoID, EntregaID: r.EntregaID}
}

type VinculoRequest struct {
	Versao    int        `json:"versao"`
	PedidoID  *uuid.UUID `json:"pedido_id"`
	EntregaID *uuid.UUID `json:"entrega_id"`
}

func (r VinculoRequest) ParaInput(usuarioID, notaFiscalID uuid.UUID) portsin.VincularInput {
	return portsin.VincularInput{
		UsuarioID: usuarioID, NotaFiscalID: notaFiscalID, Versao: r.Versao, PedidoID: r.PedidoID, EntregaID: r.EntregaID,
	}
}

type ProdutoDoItemRequest struct {
	Versao    int       `json:"versao"`
	ProdutoID uuid.UUID `json:"produto_id"`
}

func (r ProdutoDoItemRequest) ParaInput(usuarioID, notaFiscalID uuid.UUID, numeroItem int) portsin.VincularProdutoInput {
	return portsin.VincularProdutoInput{
		UsuarioID: usuarioID, NotaFiscalID: notaFiscalID, NumeroItem: numeroItem, Versao: r.Versao, ProdutoID: r.ProdutoID,
	}
}

type NotaFiscalResponse struct {
	ID                    uuid.UUID  `json:"id"`
	Chave                 string     `json:"chave"`
	EmpresaID             uuid.UUID  `json:"empresa_id"`
	Numero                int        `json:"numero"`
	Serie                 int        `json:"serie"`
	EmitidaEm             time.Time  `json:"emitida_em"`
	TipoOperacao          string     `json:"tipo_operacao"`
	Finalidade            string     `json:"finalidade"`
	Natureza              string     `json:"natureza"`
	NaturezaOperacao      string     `json:"natureza_operacao"`
	EntreEmpresas         bool       `json:"entre_empresas"`
	ClienteID             *uuid.UUID `json:"cliente_id"`
	EmpresaDestinatariaID *uuid.UUID `json:"empresa_destinataria_id"`
	DestinatarioDocumento string     `json:"destinatario_documento"`
	DestinatarioNome      string     `json:"destinatario_nome"`
	CFOPPredominante      string     `json:"cfop_predominante"`
	ValorTotalCentavos    int64      `json:"valor_total_centavos"`
	ValorProdutosCentavos int64      `json:"valor_produtos_centavos"`
	ValorFreteCentavos    int64      `json:"valor_frete_centavos"`
	ValorDescontoCentavos int64      `json:"valor_desconto_centavos"`
	NumeroPedidoCliente   *string    `json:"numero_pedido_cliente"`
	Protocolo             string     `json:"protocolo"`
	AutorizadaEm          time.Time  `json:"autorizada_em"`
	Status                string     `json:"status"`
	PedidoID              *uuid.UUID `json:"pedido_id"`
	EntregaID             *uuid.UUID `json:"entrega_id"`
	ArquivoID             uuid.UUID  `json:"arquivo_id"`
	UsuarioID             uuid.UUID  `json:"usuario_id"`
	Versao                int        `json:"versao"`
	CriadoEm              time.Time  `json:"criado_em"`
	AtualizadoEm          time.Time  `json:"atualizado_em"`
}

type ItemResponse struct {
	NumeroItem              int        `json:"numero_item"`
	Codigo                  string     `json:"codigo"`
	Descricao               string     `json:"descricao"`
	ProdutoID               *uuid.UUID `json:"produto_id"`
	NCM                     string     `json:"ncm"`
	CFOP                    string     `json:"cfop"`
	Unidade                 string     `json:"unidade"`
	Quantidade              string     `json:"quantidade"`
	QuantidadeDecimilesimos int64      `json:"quantidade_decimilesimos"`
	ValorUnitario           string     `json:"valor_unitario"`
	ValorCentavos           int64      `json:"valor_centavos"`
	DescontoCentavos        int64      `json:"desconto_centavos"`
	PedidoCliente           *string    `json:"pedido_cliente"`
}

type DuplicataResponse struct {
	ID            uuid.UUID `json:"id"`
	Numero        string    `json:"numero"`
	Vencimento    time.Time `json:"vencimento"`
	ValorCentavos int64     `json:"valor_centavos"`
}

type NotaFiscalDetalheResponse struct {
	NotaFiscalResponse
	Itens      []ItemResponse      `json:"itens"`
	Duplicatas []DuplicataResponse `json:"duplicatas"`
}

type ImportacaoResponse struct {
	NotaFiscalDetalheResponse
	Avisos         []string    `json:"avisos"`
	TitulosGerados []uuid.UUID `json:"titulos_gerados"`
}

type SincronizacaoTitulosResponse struct {
	Gerados      []uuid.UUID `json:"gerados"`
	JaExistentes []uuid.UUID `json:"ja_existentes"`
	Cancelados   []uuid.UUID `json:"cancelados"`
	Marcados     []uuid.UUID `json:"marcados"`
	Ignorados    int         `json:"ignorados"`
}

type ItemPendenteResponse struct {
	NotaFiscalID uuid.UUID    `json:"nota_fiscal_id"`
	EmpresaID    uuid.UUID    `json:"empresa_id"`
	Chave        string       `json:"chave"`
	NumeroNota   int          `json:"numero_nota"`
	Serie        int          `json:"serie"`
	EmitidaEm    time.Time    `json:"emitida_em"`
	Item         ItemResponse `json:"item"`
}

type EntregaSemNotaResponse struct {
	EntregaID      uuid.UUID  `json:"entrega_id"`
	PedidoID       uuid.UUID  `json:"pedido_id"`
	EmpresaID      uuid.UUID  `json:"empresa_id"`
	PedidoNumero   int        `json:"pedido_numero"`
	ClienteID      uuid.UUID  `json:"cliente_id"`
	Status         string     `json:"status"`
	SaiuEm         *time.Time `json:"saiu_em"`
	EntregueEm     *time.Time `json:"entregue_em"`
	Transportadora *string    `json:"transportadora"`
}

func NovaNotaFiscalResponse(nota *domainnotafiscal.NotaFiscal) NotaFiscalResponse {
	return NotaFiscalResponse{
		ID:                    nota.ID,
		Chave:                 nota.Chave.Valor(),
		EmpresaID:             nota.EmpresaID,
		Numero:                nota.Numero,
		Serie:                 nota.Serie,
		EmitidaEm:             nota.EmitidaEm,
		TipoOperacao:          nota.TipoOperacao.String(),
		Finalidade:            nota.Finalidade.String(),
		Natureza:              nota.Natureza.String(),
		NaturezaOperacao:      nota.NaturezaOperacao,
		EntreEmpresas:         nota.EntreEmpresas,
		ClienteID:             nota.ClienteID,
		EmpresaDestinatariaID: nota.EmpresaDestinatariaID,
		DestinatarioDocumento: nota.DestinatarioDocumento,
		DestinatarioNome:      nota.DestinatarioNome,
		CFOPPredominante:      nota.CFOPPredominante,
		ValorTotalCentavos:    nota.ValorTotalCentavos,
		ValorProdutosCentavos: nota.ValorProdutosCentavos,
		ValorFreteCentavos:    nota.ValorFreteCentavos,
		ValorDescontoCentavos: nota.ValorDescontoCentavos,
		NumeroPedidoCliente:   nota.NumeroPedidoCliente,
		Protocolo:             nota.Protocolo,
		AutorizadaEm:          nota.AutorizadaEm,
		Status:                nota.Status.String(),
		PedidoID:              nota.PedidoID,
		EntregaID:             nota.EntregaID,
		ArquivoID:             nota.ArquivoID,
		UsuarioID:             nota.UsuarioID,
		Versao:                nota.Versao,
		CriadoEm:              nota.CriadoEm,
		AtualizadoEm:          nota.AtualizadoEm,
	}
}

func NovaNotaFiscalResponses(notas []*domainnotafiscal.NotaFiscal) []NotaFiscalResponse {
	respostas := make([]NotaFiscalResponse, 0, len(notas))
	for _, nota := range notas {
		respostas = append(respostas, NovaNotaFiscalResponse(nota))
	}

	return respostas
}

func NovaNotaFiscalDetalheResponse(nota *domainnotafiscal.NotaFiscal) NotaFiscalDetalheResponse {
	detalhe := NotaFiscalDetalheResponse{
		NotaFiscalResponse: NovaNotaFiscalResponse(nota),
		Itens:              make([]ItemResponse, 0, len(nota.Itens)),
		Duplicatas:         make([]DuplicataResponse, 0, len(nota.Duplicatas)),
	}
	for _, item := range nota.Itens {
		detalhe.Itens = append(detalhe.Itens, novoItemResponse(item))
	}
	for _, duplicata := range nota.Duplicatas {
		detalhe.Duplicatas = append(detalhe.Duplicatas, DuplicataResponse{
			ID:            duplicata.ID,
			Numero:        duplicata.Numero,
			Vencimento:    duplicata.Vencimento,
			ValorCentavos: duplicata.ValorCentavos,
		})
	}

	return detalhe
}

func NovaImportacaoResponse(importacao *portsin.Importacao) ImportacaoResponse {
	avisos := importacao.Avisos
	if avisos == nil {
		avisos = []string{}
	}

	return ImportacaoResponse{
		NotaFiscalDetalheResponse: NovaNotaFiscalDetalheResponse(importacao.Nota),
		Avisos:                    avisos,
		TitulosGerados:            idsOuVazio(importacao.TitulosGerados),
	}
}

func NovaSincronizacaoTitulosResponse(sincronizacao *portsin.SincronizacaoTitulos) SincronizacaoTitulosResponse {
	return SincronizacaoTitulosResponse{
		Gerados:      idsOuVazio(sincronizacao.Gerados),
		JaExistentes: idsOuVazio(sincronizacao.JaExistentes),
		Cancelados:   idsOuVazio(sincronizacao.Cancelados),
		Marcados:     idsOuVazio(sincronizacao.Marcados),
		Ignorados:    sincronizacao.Ignorados,
	}
}

func idsOuVazio(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}

	return ids
}

func NovosItensPendentesResponse(pendentes []*domainnotafiscal.ItemPendente) []ItemPendenteResponse {
	respostas := make([]ItemPendenteResponse, 0, len(pendentes))
	for _, pendente := range pendentes {
		respostas = append(respostas, ItemPendenteResponse{
			NotaFiscalID: pendente.NotaFiscalID,
			EmpresaID:    pendente.EmpresaID,
			Chave:        pendente.Chave.Valor(),
			NumeroNota:   pendente.NumeroNota,
			Serie:        pendente.Serie,
			EmitidaEm:    pendente.EmitidaEm,
			Item:         novoItemResponse(pendente.Item),
		})
	}

	return respostas
}

func NovasEntregasSemNotaResponse(entregas []*domainnotafiscal.EntregaSemNota) []EntregaSemNotaResponse {
	respostas := make([]EntregaSemNotaResponse, 0, len(entregas))
	for _, entrega := range entregas {
		respostas = append(respostas, EntregaSemNotaResponse{
			EntregaID:      entrega.EntregaID,
			PedidoID:       entrega.PedidoID,
			EmpresaID:      entrega.EmpresaID,
			PedidoNumero:   entrega.PedidoNumero,
			ClienteID:      entrega.ClienteID,
			Status:         entrega.Status,
			SaiuEm:         entrega.SaiuEm,
			EntregueEm:     entrega.EntregueEm,
			Transportadora: entrega.Transportadora,
		})
	}

	return respostas
}

func novoItemResponse(item *domainnotafiscal.Item) ItemResponse {
	return ItemResponse{
		NumeroItem:              item.NumeroItem,
		Codigo:                  item.Codigo,
		Descricao:               item.Descricao,
		ProdutoID:               item.ProdutoID,
		NCM:                     item.NCM,
		CFOP:                    item.CFOP,
		Unidade:                 item.Unidade,
		Quantidade:              quantidadeFormatada(item.QuantidadeDecimilesimos),
		QuantidadeDecimilesimos: item.QuantidadeDecimilesimos,
		ValorUnitario:           item.ValorUnitario,
		ValorCentavos:           item.ValorCentavos,
		DescontoCentavos:        item.DescontoCentavos,
		PedidoCliente:           item.PedidoCliente,
	}
}

func quantidadeFormatada(decimilesimos int64) string {
	return fmt.Sprintf("%d.%04d", decimilesimos/escalaDaQuantidade, decimilesimos%escalaDaQuantidade)
}
