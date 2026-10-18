package notafiscal

import (
	"time"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/google/uuid"
)

type ImportarEventoRequest struct {
	ArquivoID uuid.UUID `json:"arquivo_id"`
}

func (r ImportarEventoRequest) ParaInput(usuarioID uuid.UUID) portsin.ImportarEventoInput {
	return portsin.ImportarEventoInput{UsuarioID: usuarioID, ArquivoID: r.ArquivoID}
}

type EventoResponse struct {
	ID            uuid.UUID `json:"id"`
	NotaFiscalID  uuid.UUID `json:"nota_fiscal_id"`
	Tipo          string    `json:"tipo"`
	Sequencia     int       `json:"sequencia"`
	OcorridoEm    time.Time `json:"ocorrido_em"`
	Protocolo     string    `json:"protocolo"`
	Justificativa string    `json:"justificativa"`
	ArquivoID     uuid.UUID `json:"arquivo_id"`
	UsuarioID     uuid.UUID `json:"usuario_id"`
	CriadoEm      time.Time `json:"criado_em"`
}

type ImportacaoEventoResponse struct {
	Evento            EventoResponse     `json:"evento"`
	NotaFiscal        NotaFiscalResponse `json:"nota_fiscal"`
	Avisos            []string           `json:"avisos"`
	TitulosCancelados []uuid.UUID        `json:"titulos_cancelados"`
	TitulosMarcados   []uuid.UUID        `json:"titulos_marcados"`
}

func NovoEventoResponse(evento *domainnotafiscal.Evento) EventoResponse {
	return EventoResponse{
		ID:            evento.ID,
		NotaFiscalID:  evento.NotaFiscalID,
		Tipo:          evento.Tipo.String(),
		Sequencia:     evento.Sequencia,
		OcorridoEm:    evento.OcorridoEm,
		Protocolo:     evento.Protocolo,
		Justificativa: evento.Justificativa,
		ArquivoID:     evento.ArquivoID,
		UsuarioID:     evento.UsuarioID,
		CriadoEm:      evento.CriadoEm,
	}
}

func NovosEventosResponse(eventos []*domainnotafiscal.Evento) []EventoResponse {
	respostas := make([]EventoResponse, 0, len(eventos))
	for _, evento := range eventos {
		respostas = append(respostas, NovoEventoResponse(evento))
	}

	return respostas
}

func NovaImportacaoEventoResponse(importacao *portsin.ImportacaoEvento) ImportacaoEventoResponse {
	avisos := importacao.Avisos
	if avisos == nil {
		avisos = []string{}
	}

	return ImportacaoEventoResponse{
		Evento:            NovoEventoResponse(importacao.Evento),
		NotaFiscal:        NovaNotaFiscalResponse(importacao.Nota),
		Avisos:            avisos,
		TitulosCancelados: idsOuVazio(importacao.TitulosCancelados),
		TitulosMarcados:   idsOuVazio(importacao.TitulosMarcados),
	}
}
