package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type MovimentarEstoqueRequest struct {
	Tipo         string  `json:"tipo"`
	Quantidade   int     `json:"quantidade"`
	DocumentoRef *string `json:"documento_ref"`
	Observacao   string  `json:"observacao"`
}

func (r MovimentarEstoqueRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.MovimentarEstoqueInput {
	return portsin.MovimentarEstoqueInput{
		UsuarioID:    usuarioID,
		ProdutoID:    produtoID,
		Tipo:         r.Tipo,
		Quantidade:   r.Quantidade,
		DocumentoRef: r.DocumentoRef,
		Observacao:   r.Observacao,
	}
}

type MovimentoResponse struct {
	ID           uuid.UUID `json:"id"`
	ProdutoID    uuid.UUID `json:"produto_id"`
	Tipo         string    `json:"tipo"`
	Quantidade   int       `json:"quantidade"`
	SaldoApos    int       `json:"saldo_apos"`
	UsuarioID    uuid.UUID `json:"usuario_id"`
	DocumentoRef *string   `json:"documento_ref"`
	Observacao   string    `json:"observacao"`
	CriadoEm     time.Time `json:"criado_em"`
}

func NovoMovimentoResponse(item *domainestoque.Movimento) MovimentoResponse {
	return MovimentoResponse{
		ID:           item.ID,
		ProdutoID:    item.ProdutoID,
		Tipo:         item.Tipo.String(),
		Quantidade:   item.Quantidade.Valor(),
		SaldoApos:    item.SaldoApos.Quantidade(),
		UsuarioID:    item.UsuarioID,
		DocumentoRef: item.DocumentoRef,
		Observacao:   item.Observacao,
		CriadoEm:     item.CriadoEm,
	}
}

func NovoMovimentoResponses(itens []*domainestoque.Movimento) []MovimentoResponse {
	respostas := make([]MovimentoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoMovimentoResponse(item))
	}

	return respostas
}

type SaldoResponse struct {
	ProdutoID uuid.UUID `json:"produto_id"`
	Saldo     int       `json:"saldo"`
}

func NovoSaldoResponse(produto *domainestoque.Produto) SaldoResponse {
	return SaldoResponse{
		ProdutoID: produto.ID,
		Saldo:     produto.Saldo.Quantidade(),
	}
}
