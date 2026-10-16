package receber

import (
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type ReceberRequest struct {
	Versao           int             `json:"versao"`
	Tipo             string          `json:"tipo"`
	ContaBancariaID  *uuid.UUID      `json:"conta_bancaria_id"`
	RecebidoEm       string          `json:"recebido_em"`
	ValorCentavos    int64           `json:"valor_centavos"`
	JurosCentavos    int64           `json:"juros_centavos"`
	DescontoCentavos int64           `json:"desconto_centavos"`
	Repasse          *RepasseRequest `json:"repasse"`
}

type EstornoRecebimentoRequest struct {
	Versao      int    `json:"versao"`
	Motivo      string `json:"motivo"`
	EstornadoEm string `json:"estornado_em"`
}

type EstornoRecebimentoResponse struct {
	ID          uuid.UUID `json:"id"`
	Motivo      string    `json:"motivo"`
	EstornadoEm string    `json:"estornado_em"`
	UsuarioID   uuid.UUID `json:"usuario_id"`
	CriadoEm    time.Time `json:"criado_em"`
}

type RecebimentoResponse struct {
	ID                     uuid.UUID                   `json:"id"`
	TituloID               uuid.UUID                   `json:"titulo_id"`
	Tipo                   string                      `json:"tipo"`
	ContaBancariaID        *uuid.UUID                  `json:"conta_bancaria_id"`
	RecebidoEm             string                      `json:"recebido_em"`
	ValorCentavos          int64                       `json:"valor_centavos"`
	JurosCentavos          int64                       `json:"juros_centavos"`
	DescontoCentavos       int64                       `json:"desconto_centavos"`
	EntradaNoCaixaCentavos int64                       `json:"entrada_no_caixa_centavos"`
	StatusTituloAnterior   string                      `json:"status_titulo_anterior"`
	RegistrouEntradaPedido bool                        `json:"registrou_entrada_pedido"`
	Estornado              bool                        `json:"estornado"`
	Estorno                *EstornoRecebimentoResponse `json:"estorno"`
	Repasse                *RepasseResponse            `json:"repasse"`
	UsuarioID              uuid.UUID                   `json:"usuario_id"`
	CriadoEm               time.Time                   `json:"criado_em"`
}

type RecebimentoRegistradoResponse struct {
	Titulo      TituloResponse      `json:"titulo"`
	Recebimento RecebimentoResponse `json:"recebimento"`
}

func NovoRecebimentoResponse(item *domainreceber.Recebimento) RecebimentoResponse {
	resposta := RecebimentoResponse{
		ID:                     item.ID,
		TituloID:               item.TituloID,
		Tipo:                   item.Tipo.String(),
		ContaBancariaID:        item.ContaBancariaID,
		RecebidoEm:             formatarData(item.RecebidoEm),
		ValorCentavos:          item.ValorCentavos,
		JurosCentavos:          item.JurosCentavos,
		DescontoCentavos:       item.DescontoCentavos,
		EntradaNoCaixaCentavos: item.EntradaNoCaixaCentavos(),
		StatusTituloAnterior:   item.StatusTituloAnterior.String(),
		RegistrouEntradaPedido: item.RegistrouEntradaPedido,
		Estornado:              item.Estornado,
		UsuarioID:              item.UsuarioID,
		CriadoEm:               item.CriadoEm,
		Repasse:                novoRepasseResponse(item.Repasse),
	}
	if item.Estorno != nil {
		resposta.Estorno = &EstornoRecebimentoResponse{
			ID:          item.Estorno.ID,
			Motivo:      item.Estorno.Motivo,
			EstornadoEm: formatarData(item.Estorno.EstornadoEm),
			UsuarioID:   item.Estorno.UsuarioID,
			CriadoEm:    item.Estorno.CriadoEm,
		}
	}

	return resposta
}

func NovoRecebimentoResponses(itens []*domainreceber.Recebimento) []RecebimentoResponse {
	respostas := make([]RecebimentoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoRecebimentoResponse(item))
	}

	return respostas
}

func NovoRecebimentoRegistradoResponse(titulo *domainreceber.Titulo, recebimento *domainreceber.Recebimento) RecebimentoRegistradoResponse {
	return RecebimentoRegistradoResponse{
		Titulo:      NovoTituloResponse(titulo),
		Recebimento: NovoRecebimentoResponse(recebimento),
	}
}
