package solicitacao

import (
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ArquivoResponse struct {
	ID             uuid.UUID `json:"id"`
	ProprietarioID uuid.UUID `json:"proprietario_id"`
	Nome           string    `json:"nome"`
	ContentType    string    `json:"content_type"`
	Tamanho        int64     `json:"tamanho"`
	CriadoEm       time.Time `json:"criado_em"`
}

func NovoArquivoResponse(item *domainsolicitacao.Arquivo) ArquivoResponse {
	return ArquivoResponse{
		ID:             item.ID,
		ProprietarioID: item.ProprietarioID,
		Nome:           item.Nome,
		ContentType:    item.ContentType,
		Tamanho:        item.Tamanho,
		CriadoEm:       item.CriadoEm,
	}
}

func NovoArquivoResponses(itens []*domainsolicitacao.Arquivo) []ArquivoResponse {
	respostas := make([]ArquivoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoArquivoResponse(item))
	}

	return respostas
}
