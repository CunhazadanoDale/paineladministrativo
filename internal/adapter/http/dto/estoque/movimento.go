package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type MovimentarEstoqueRequest struct {
	LocalID               uuid.UUID  `json:"local_id"`
	Tipo                  string     `json:"tipo"`
	Quantidade            int        `json:"quantidade"`
	CustoUnitarioCentavos *int64     `json:"custo_unitario_centavos"`
	ReservaID             *uuid.UUID `json:"reserva_id"`
	DocumentoRef          *string    `json:"documento_ref"`
	Observacao            string     `json:"observacao"`
}

func (r MovimentarEstoqueRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.MovimentarEstoqueInput {
	return portsin.MovimentarEstoqueInput{
		UsuarioID:     usuarioID,
		ProdutoID:     produtoID,
		LocalID:       r.LocalID,
		Tipo:          r.Tipo,
		Quantidade:    r.Quantidade,
		CustoCentavos: r.CustoUnitarioCentavos,
		ReservaID:     r.ReservaID,
		DocumentoRef:  r.DocumentoRef,
		Observacao:    r.Observacao,
	}
}

type TransferirEstoqueRequest struct {
	LocalOrigemID  uuid.UUID `json:"local_origem_id"`
	LocalDestinoID uuid.UUID `json:"local_destino_id"`
	Quantidade     int       `json:"quantidade"`
	DocumentoRef   *string   `json:"documento_ref"`
	Observacao     string    `json:"observacao"`
}

func (r TransferirEstoqueRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.TransferirEstoqueInput {
	return portsin.TransferirEstoqueInput{
		UsuarioID:      usuarioID,
		ProdutoID:      produtoID,
		LocalOrigemID:  r.LocalOrigemID,
		LocalDestinoID: r.LocalDestinoID,
		Quantidade:     r.Quantidade,
		DocumentoRef:   r.DocumentoRef,
		Observacao:     r.Observacao,
	}
}

type MovimentoResponse struct {
	ID             uuid.UUID  `json:"id"`
	ProdutoID      uuid.UUID  `json:"produto_id"`
	Tipo           string     `json:"tipo"`
	Quantidade     int        `json:"quantidade"`
	SaldoApos      int        `json:"saldo_apos"`
	LocalID        *uuid.UUID `json:"local_id"`
	LocalDestinoID *uuid.UUID `json:"local_destino_id"`
	SaldoLocalApos *int       `json:"saldo_local_apos"`
	ReservaID      *uuid.UUID `json:"reserva_id"`
	UsuarioID      uuid.UUID  `json:"usuario_id"`
	DocumentoRef   *string    `json:"documento_ref"`
	Observacao     string     `json:"observacao"`
	CriadoEm       time.Time  `json:"criado_em"`
}

func NovoMovimentoResponse(item *domainestoque.Movimento) MovimentoResponse {
	resposta := MovimentoResponse{
		ID:             item.ID,
		ProdutoID:      item.ProdutoID,
		Tipo:           item.Tipo.String(),
		Quantidade:     item.Quantidade.Valor(),
		SaldoApos:      item.SaldoApos.Quantidade(),
		LocalID:        item.LocalID,
		LocalDestinoID: item.LocalDestinoID,
		ReservaID:      item.ReservaID,
		UsuarioID:      item.UsuarioID,
		DocumentoRef:   item.DocumentoRef,
		Observacao:     item.Observacao,
		CriadoEm:       item.CriadoEm,
	}
	if item.SaldoLocalApos != nil {
		saldo := item.SaldoLocalApos.Quantidade()
		resposta.SaldoLocalApos = &saldo
	}

	return resposta
}

func NovoMovimentoResponses(itens []*domainestoque.Movimento) []MovimentoResponse {
	respostas := make([]MovimentoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoMovimentoResponse(item))
	}

	return respostas
}
