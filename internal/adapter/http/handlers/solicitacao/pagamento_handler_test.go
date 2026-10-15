package solicitacao

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func TestRegistrarPagamentoRepassaContaBancaria(t *testing.T) {
	contaID := uuid.New()
	pagamento := &domainsolicitacao.Pagamento{ID: uuid.New(), ContaBancariaID: &contaID, Valor: domainsolicitacao.ValorDe(9900)}
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste(), pagamento: pagamento}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).RegistrarPagamento, http.MethodPost,
		"/api/v1/solicitacoes/"+id.String()+"/pagamento", id.String(),
		`{"conta_bancaria_id": "`+contaID.String()+`", "valor_centavos": 9900}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputPagamento.ContaBancariaID == nil || *usecase.inputPagamento.ContaBancariaID != contaID {
		t.Errorf("conta repassada = %v, esperado %s", usecase.inputPagamento.ContaBancariaID, contaID)
	}

	var corpo dto.Resposta[solicitacaodto.PagamentoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpo.Dados.ContaBancariaID == nil || *corpo.Dados.ContaBancariaID != contaID || corpo.Dados.Estornado || corpo.Dados.Estorno != nil {
		t.Errorf("resposta = %+v", corpo.Dados)
	}
}

func TestEstornarPagamentoRepassaCorpoERespondeASolicitacao(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).EstornarPagamento, http.MethodPost,
		"/api/v1/solicitacoes/"+id.String()+"/pagamento/estorno", id.String(),
		`{"motivo": "pago em duplicidade", "estornado_em": "2026-10-15T14:00:00Z", "versao": 3}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	entrada := usecase.inputEstorno
	if entrada.SolicitacaoID != id || entrada.UsuarioID == uuid.Nil || entrada.Versao != 3 || entrada.Motivo != "pago em duplicidade" {
		t.Errorf("entrada = %+v", entrada)
	}
	if !entrada.EstornadoEm.Equal(time.Date(2026, time.October, 15, 14, 0, 0, 0, time.UTC)) {
		t.Errorf("data = %s", entrada.EstornadoEm)
	}
}

func TestEstornarPagamentoComDataInvalidaResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).EstornarPagamento, http.MethodPost,
		"/api/v1/solicitacoes/"+id+"/pagamento/estorno", id, `{"motivo": "x", "estornado_em": "15/10/2026", "versao": 1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "erro de validação: data do estorno deve estar no formato AAAA-MM-DDTHH:MM:SSZ" {
		t.Errorf("mensagem = %q", apoioteste.MensagemDoEnvelope(t, resposta))
	}
	if usecase.inputEstorno.SolicitacaoID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com data inválida")
	}
}

func TestListarPagamentosDevolveOEstorno(t *testing.T) {
	usuarioID := uuid.New()
	estornado := &domainsolicitacao.Pagamento{
		ID:        uuid.New(),
		Valor:     domainsolicitacao.ValorDe(9900),
		Estornado: true,
		Estorno: &domainsolicitacao.Estorno{
			ID: uuid.New(), Motivo: domainsolicitacao.MotivoEstornoDe("pago em duplicidade"), UsuarioID: usuarioID,
		},
	}
	vigente := &domainsolicitacao.Pagamento{ID: uuid.New(), Valor: domainsolicitacao.ValorDe(9900)}
	usecase := &solicitacaoUseCaseFalso{pagamentos: []*domainsolicitacao.Pagamento{estornado, vigente}}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).ListarPagamentos, http.MethodGet,
		"/api/v1/solicitacoes/"+id+"/pagamentos", id, "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	var corpo dto.Resposta[[]solicitacaodto.PagamentoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if len(corpo.Dados) != 2 || !corpo.Dados[0].Estornado || corpo.Dados[0].Estorno == nil || corpo.Dados[1].Estorno != nil {
		t.Fatalf("pagamentos = %+v", corpo.Dados)
	}
	if corpo.Dados[0].Estorno.Motivo != "pago em duplicidade" || corpo.Dados[0].Estorno.UsuarioID != usuarioID {
		t.Errorf("estorno = %+v", corpo.Dados[0].Estorno)
	}
}
