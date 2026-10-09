package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type AnexarImagemRequest struct {
	ArquivoID uuid.UUID `json:"arquivo_id"`
	Ordem     int       `json:"ordem"`
	Alt       string    `json:"alt"`
}

func (r AnexarImagemRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.AnexarImagemInput {
	return portsin.AnexarImagemInput{
		UsuarioID: usuarioID,
		ProdutoID: produtoID,
		ArquivoID: r.ArquivoID,
		Ordem:     r.Ordem,
		Alt:       r.Alt,
	}
}

type ImagemResponse struct {
	ID        uuid.UUID `json:"id"`
	ProdutoID uuid.UUID `json:"produto_id"`
	ArquivoID uuid.UUID `json:"arquivo_id"`
	Ordem     int       `json:"ordem"`
	Alt       string    `json:"alt"`
	CriadoEm  time.Time `json:"criado_em"`
}

func NovaImagemResponse(item *domainestoque.Imagem) ImagemResponse {
	return ImagemResponse{
		ID:        item.ID,
		ProdutoID: item.ProdutoID,
		ArquivoID: item.ArquivoID,
		Ordem:     item.Ordem,
		Alt:       item.Alt,
		CriadoEm:  item.CriadoEm,
	}
}

func NovaImagemResponses(itens []*domainestoque.Imagem) []ImagemResponse {
	respostas := make([]ImagemResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaImagemResponse(item))
	}

	return respostas
}
