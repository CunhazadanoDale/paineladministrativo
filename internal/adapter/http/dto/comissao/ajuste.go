package comissao

import (
	"time"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type CriarAjusteRequest struct {
	EmpresaID       uuid.UUID  `json:"empresa_id"`
	RepresentanteID uuid.UUID  `json:"representante_id"`
	PedidoID        *uuid.UUID `json:"pedido_id"`
	Competencia     string     `json:"competencia"`
	ValorCentavos   int64      `json:"valor_centavos"`
	Motivo          string     `json:"motivo"`
}

func (r CriarAjusteRequest) ParaDados() domaincomissao.DadosAjuste {
	return domaincomissao.DadosAjuste{
		EmpresaID:       r.EmpresaID,
		RepresentanteID: r.RepresentanteID,
		PedidoID:        r.PedidoID,
		Competencia:     r.Competencia,
		ValorCentavos:   r.ValorCentavos,
		Motivo:          r.Motivo,
	}
}

type AjusteResponse struct {
	ID              uuid.UUID  `json:"id"`
	EmpresaID       uuid.UUID  `json:"empresa_id"`
	RepresentanteID uuid.UUID  `json:"representante_id"`
	PedidoID        *uuid.UUID `json:"pedido_id"`
	Competencia     string     `json:"competencia"`
	ValorCentavos   int64      `json:"valor_centavos"`
	Motivo          string     `json:"motivo"`
	FechamentoID    *uuid.UUID `json:"fechamento_id"`
	UsuarioID       uuid.UUID  `json:"usuario_id"`
	CriadoEm        time.Time  `json:"criado_em"`
}

func NovoAjusteResponse(item *domaincomissao.Ajuste) AjusteResponse {
	return AjusteResponse{
		ID:              item.ID,
		EmpresaID:       item.EmpresaID,
		RepresentanteID: item.RepresentanteID,
		PedidoID:        item.PedidoID,
		Competencia:     item.Competencia.String(),
		ValorCentavos:   item.ValorCentavos,
		Motivo:          item.Motivo,
		FechamentoID:    item.FechamentoID,
		UsuarioID:       item.UsuarioID,
		CriadoEm:        item.CriadoEm,
	}
}

func NovoAjusteResponses(itens []*domaincomissao.Ajuste) []AjusteResponse {
	respostas := make([]AjusteResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoAjusteResponse(item))
	}

	return respostas
}
