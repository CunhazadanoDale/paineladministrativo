package estoque

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type reservaUseCaseFalso struct {
	erro          error
	reserva       *domainestoque.Reserva
	inputReservar portsin.ReservarEstoqueInput
	inputListar   portsin.ListarReservasInput
	liberada      uuid.UUID
}

func (f *reservaUseCaseFalso) Reservar(_ context.Context, input portsin.ReservarEstoqueInput) (*domainestoque.Reserva, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputReservar = input

	return f.reserva, nil
}

func (f *reservaUseCaseFalso) Liberar(_ context.Context, reservaID, _ uuid.UUID) (*domainestoque.Reserva, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.liberada = reservaID

	return f.reserva, nil
}

func (f *reservaUseCaseFalso) Listar(_ context.Context, input portsin.ListarReservasInput) ([]*domainestoque.Reserva, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return []*domainestoque.Reserva{f.reserva}, nil
}

var _ portsin.ReservaUseCase = (*reservaUseCaseFalso)(nil)

func novaReservaDeTeste() *domainestoque.Reserva {
	return &domainestoque.Reserva{
		ID:         uuid.New(),
		ProdutoID:  uuid.New(),
		LocalID:    uuid.New(),
		Quantidade: 3,
		Origem:     domainestoque.OrigemReservaManual,
		Status:     domainestoque.StatusReservaAtiva,
		UsuarioID:  uuid.New(),
		Observacao: "pedido do cliente",
		CriadoEm:   time.Now().UTC(),
	}
}

func TestReservarRespondeCreatedERepassaInput(t *testing.T) {
	reserva := novaReservaDeTeste()
	usecase := &reservaUseCaseFalso{reserva: reserva}
	id := uuid.New()

	resposta := executaProtegido(t, NewReservaHandler(usecase).Reservar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/reservas", id.String(),
		`{"local_id":"`+reserva.LocalID.String()+`","quantidade":3,"observacao":"pedido do cliente"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	input := usecase.inputReservar
	if input.ProdutoID != id || input.LocalID != reserva.LocalID || input.Quantidade != 3 || input.UsuarioID == uuid.Nil {
		t.Errorf("input repassado = %+v", input)
	}

	var envelope dto.Resposta[estoquedto.ReservaResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if envelope.Dados.ID != reserva.ID || envelope.Dados.Status != "ativa" || envelope.Dados.Origem != "manual" || envelope.Dados.EncerradaEm != nil {
		t.Errorf("reserva devolvida = %+v", envelope.Dados)
	}
}

func TestLiberarReservaRepassaIdEMapeiaConflito(t *testing.T) {
	reserva := novaReservaDeTeste()
	usecase := &reservaUseCaseFalso{reserva: reserva}
	caminho := "/api/v1/reservas/" + reserva.ID.String() + "/liberar"

	resposta := executaProtegido(t, NewReservaHandler(usecase).Liberar, http.MethodPost, caminho, reserva.ID.String(), "")
	if resposta.Code != http.StatusOK || usecase.liberada != reserva.ID {
		t.Errorf("liberar: status %d, reserva %s", resposta.Code, usecase.liberada)
	}

	usecase.erro = domain.ErroConflito("reserva já encerrada")
	resposta = executaProtegido(t, NewReservaHandler(usecase).Liberar, http.MethodPost, caminho, reserva.ID.String(), "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
}

func TestListarReservasRepassaFiltros(t *testing.T) {
	usecase := &reservaUseCaseFalso{reserva: novaReservaDeTeste()}
	id, localID := uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewReservaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/reservas?status=ativa&local="+localID.String(), id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	filtro := usecase.inputListar
	if filtro.ProdutoID != id || filtro.Status != "ativa" || filtro.LocalID == nil || *filtro.LocalID != localID || filtro.UsuarioID == uuid.Nil {
		t.Errorf("filtro repassado = %+v", filtro)
	}

	resposta = executaProtegido(t, NewReservaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/reservas?local=abc", id.String(), "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestMovimentarRepassaReserva(t *testing.T) {
	usecase := &movimentoUseCaseFalso{movimento: novoMovimentoDeTeste(t)}
	id, localID, reservaID := uuid.New(), uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Movimentar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/movimentos", id.String(),
		`{"local_id":"`+localID.String()+`","tipo":"saida","quantidade":3,"reserva_id":"`+reservaID.String()+`"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputMovimentar.ReservaID == nil || *usecase.inputMovimentar.ReservaID != reservaID {
		t.Errorf("reserva repassada = %v, esperado %s", usecase.inputMovimentar.ReservaID, reservaID)
	}
}
