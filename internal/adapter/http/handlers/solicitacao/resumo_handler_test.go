package solicitacao

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type resumoUseCaseFalso struct {
	resumo domainsolicitacao.ResumoContasAPagar
	input  portsin.ConsultarResumoInput
	erro   error
}

func (f *resumoUseCaseFalso) Consultar(_ context.Context, input portsin.ConsultarResumoInput) (domainsolicitacao.ResumoContasAPagar, error) {
	f.input = input
	return f.resumo, f.erro
}

func TestPendenciasRespondeNuloParaFilaForaDoPerfil(t *testing.T) {
	quatro := 4
	usecase := &solicitacaoUseCaseFalso{pendencias: domainsolicitacao.Pendencias{Aprovacao: &quatro}}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Pendencias, http.MethodGet, "/api/v1/solicitacoes/pendencias", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	var corpo map[string]map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	dados := corpo["dados"]
	if dados["aprovacao"] != float64(4) || dados["financeiro"] != nil || dados["vencidas"] != nil {
		t.Errorf("pendências = %v", dados)
	}
	if _, existe := dados["financeiro"]; !existe {
		t.Error("campo financeiro deveria vir como null, não omitido")
	}
}

func TestResumoRepassaPeriodoEEmpresaEDevolveDatasCurtas(t *testing.T) {
	empresaID := uuid.New()
	usecase := &resumoUseCaseFalso{resumo: domainsolicitacao.NovoResumoContasAPagar(
		domainsolicitacao.PeriodoResumo{De: time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC), Ate: time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC)},
		[]domainsolicitacao.LinhaResumoSemanal{{
			Semana:    time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC),
			EmpresaID: empresaID,
			APagar:    domainsolicitacao.TotalResumo{Quantidade: 2, ValorCentavos: 500},
		}},
		nil,
	)}

	resposta := executaProtegido(t, NewResumoContasAPagarHandler(usecase).Consultar, http.MethodGet,
		"/api/v1/contas-a-pagar/resumo?de=2026-10-12&ate=2026-10-25&empresa="+empresaID.String(), "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.input.De == nil || usecase.input.De.Format("2006-01-02") != "2026-10-12" || usecase.input.Ate == nil ||
		usecase.input.Empresa == nil || *usecase.input.Empresa != empresaID || usecase.input.UsuarioID == uuid.Nil {
		t.Errorf("entrada = %+v", usecase.input)
	}

	var corpo struct {
		Dados struct {
			De        string `json:"de"`
			Ate       string `json:"ate"`
			Semanas   []map[string]any
			Atrasadas []any                               `json:"atrasadas"`
			Totais    solicitacaodto.TotaisResumoResponse `json:"totais"`
		} `json:"dados"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpo.Dados.De != "2026-10-12" || corpo.Dados.Ate != "2026-10-25" || len(corpo.Dados.Semanas) != 1 || corpo.Dados.Atrasadas == nil {
		t.Errorf("resumo = %+v", corpo.Dados)
	}
	if corpo.Dados.Semanas[0]["semana"] != "2026-10-12" || corpo.Dados.Totais.APagar.ValorCentavos != 500 {
		t.Errorf("semana = %v, totais = %v", corpo.Dados.Semanas[0], corpo.Dados.Totais)
	}
}

func TestResumoComDataInvalidaResponde400(t *testing.T) {
	for _, consulta := range []string{"de=12/10/2026", "ate=2026-13-01", "empresa=x"} {
		usecase := &resumoUseCaseFalso{}

		resposta := executaProtegido(t, NewResumoContasAPagarHandler(usecase).Consultar, http.MethodGet,
			"/api/v1/contas-a-pagar/resumo?"+consulta, "", "")

		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
		if usecase.input.UsuarioID != uuid.Nil {
			t.Errorf("%s chegou ao usecase", consulta)
		}
	}
}
