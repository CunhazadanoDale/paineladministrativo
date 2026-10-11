package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type CriarLocalRequest struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	Nome      string    `json:"nome"`
}

func (r CriarLocalRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarLocalInput {
	return portsin.CriarLocalInput{
		UsuarioID: usuarioID,
		EmpresaID: r.EmpresaID,
		Nome:      r.Nome,
	}
}

type RenomearLocalRequest struct {
	Nome string `json:"nome"`
}

func (r RenomearLocalRequest) ParaInput(usuarioID, localID uuid.UUID) portsin.RenomearLocalInput {
	return portsin.RenomearLocalInput{
		UsuarioID: usuarioID,
		LocalID:   localID,
		Nome:      r.Nome,
	}
}

type LocalResponse struct {
	ID           uuid.UUID `json:"id"`
	EmpresaID    uuid.UUID `json:"empresa_id"`
	Nome         string    `json:"nome"`
	Ativo        bool      `json:"ativo"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

func NovoLocalResponse(item *domainestoque.Local) LocalResponse {
	return LocalResponse{
		ID:           item.ID,
		EmpresaID:    item.EmpresaID,
		Nome:         item.Nome,
		Ativo:        item.Ativo,
		CriadoEm:     item.CriadoEm,
		AtualizadoEm: item.AtualizadoEm,
	}
}

func NovoLocalResponses(itens []*domainestoque.Local) []LocalResponse {
	respostas := make([]LocalResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoLocalResponse(item))
	}

	return respostas
}
