package comissao

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	"github.com/google/uuid"
)

type ajusteUseCaseFalso struct {
	portsin.AjusteUseCase
	input portsin.CriarAjusteInput
}

func (f *ajusteUseCaseFalso) Criar(_ context.Context, input portsin.CriarAjusteInput) (*domaincomissao.Ajuste, error) {
	f.input = input

	competencia, err := domaincomissao.NovaCompetencia(input.Dados.Competencia)
	if err != nil {
		return nil, err
	}

	return &domaincomissao.Ajuste{
		ID:              uuid.New(),
		EmpresaID:       input.Dados.EmpresaID,
		RepresentanteID: input.Dados.RepresentanteID,
		Competencia:     competencia,
		ValorCentavos:   input.Dados.ValorCentavos,
		Motivo:          input.Dados.Motivo,
		UsuarioID:       input.UsuarioID,
	}, nil
}

func TestCriarAjusteLeOCorpoERespondeCriado(t *testing.T) {
	usecase := &ajusteUseCaseFalso{}
	handler := NewAjusteHandler(usecase)
	empresaID, representanteID := uuid.New(), uuid.New()

	corpo := `{"empresa_id":"` + empresaID.String() + `","representante_id":"` + representanteID.String() +
		`","competencia":"2026-10","valor_centavos":-375,"motivo":"devolução"}`
	registrador := executaProtegido(t, handler.Criar, http.MethodPost, "/api/v1/comissoes/ajustes", "", corpo)
	if registrador.Code != http.StatusCreated || !strings.Contains(registrador.Body.String(), `"valor_centavos":-375`) ||
		!strings.Contains(registrador.Body.String(), `"competencia":"2026-10"`) {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.input.Dados.EmpresaID != empresaID || usecase.input.Dados.RepresentanteID != representanteID || usecase.input.Dados.PedidoID != nil {
		t.Fatalf("input = %+v", usecase.input)
	}

	desconhecido := executaProtegido(t, handler.Criar, http.MethodPost, "/api/v1/comissoes/ajustes", "", `{"valor":1}`)
	apoioteste.VerificarEnvelopeDeErro(t, desconhecido, http.StatusBadRequest)
}
