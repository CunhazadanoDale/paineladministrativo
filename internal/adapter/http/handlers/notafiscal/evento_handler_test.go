package notafiscal

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	notafiscaldto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/google/uuid"
)

type eventoUseCaseFalso struct {
	erro          error
	nota          *domainnotafiscal.NotaFiscal
	evento        *domainnotafiscal.Evento
	inputImportar portsin.ImportarEventoInput
	notaListada   uuid.UUID
}

func (f *eventoUseCaseFalso) Importar(_ context.Context, input portsin.ImportarEventoInput) (*portsin.ImportacaoEvento, error) {
	f.inputImportar = input
	return &portsin.ImportacaoEvento{Evento: f.evento, Nota: f.nota}, f.erro
}

func (f *eventoUseCaseFalso) Listar(_ context.Context, notaFiscalID, _ uuid.UUID) ([]*domainnotafiscal.Evento, error) {
	f.notaListada = notaFiscalID
	return []*domainnotafiscal.Evento{f.evento}, f.erro
}

var _ portsin.EventoUseCase = (*eventoUseCaseFalso)(nil)

func TestImportarEventoDevolveEventoENota(t *testing.T) {
	nota := notaDeTeste()
	nota.Status = domainnotafiscal.StatusCancelada
	evento := &domainnotafiscal.Evento{ID: uuid.New(), NotaFiscalID: nota.ID, Tipo: domainnotafiscal.TipoEventoCancelamento, Sequencia: 1}
	usecase := &eventoUseCaseFalso{nota: nota, evento: evento}
	arquivo := uuid.New()

	resposta := executaProtegido(t, NewEventoHandler(usecase).Importar, http.MethodPost, "/api/v1/notas-fiscais/eventos", nil,
		`{"arquivo_id":"`+arquivo.String()+`"}`)

	if resposta.Code != http.StatusCreated || usecase.inputImportar.ArquivoID != arquivo {
		t.Fatalf("status %d input %+v: %s", resposta.Code, usecase.inputImportar, resposta.Body.String())
	}
	var envelope dto.Resposta[notafiscaldto.ImportacaoEventoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if envelope.Dados.Evento.Tipo != "cancelamento" || envelope.Dados.NotaFiscal.Status != "cancelada" {
		t.Errorf("resposta = %+v", envelope.Dados)
	}
}

func TestListarEventosEErros(t *testing.T) {
	evento := &domainnotafiscal.Evento{ID: uuid.New(), Tipo: domainnotafiscal.TipoEventoCartaCorrecao, Sequencia: 2}
	usecase := &eventoUseCaseFalso{evento: evento}
	id := uuid.New()

	resposta := executaProtegido(t, NewEventoHandler(usecase).Listar, http.MethodGet, "/api/v1/notas-fiscais/"+id.String()+"/eventos",
		map[string]string{"id": id.String()}, "")
	var envelope dto.Resposta[[]notafiscaldto.EventoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil || resposta.Code != http.StatusOK ||
		usecase.notaListada != id || len(envelope.Dados) != 1 || envelope.Dados[0].Tipo != "carta_correcao" {
		t.Errorf("eventos = %d %s", resposta.Code, resposta.Body.String())
	}

	conflito := &eventoUseCaseFalso{erro: domain.ErroConflito("nota fiscal já cancelada")}
	resposta = executaProtegido(t, NewEventoHandler(conflito).Importar, http.MethodPost, "/api/v1/notas-fiscais/eventos", nil,
		`{"arquivo_id":"`+uuid.NewString()+`"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
}
