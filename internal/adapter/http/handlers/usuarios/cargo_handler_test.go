package usuarios

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type dtoRespostaCargo = dto.Resposta[usuariosdto.CargoResponse]

type dtoRespostaCargos = dto.Paginado[usuariosdto.CargoResponse]

type fakeCargoUseCase struct {
	cargos map[uuid.UUID]*domainusuarios.Cargo
}

func novoFakeCargos() *fakeCargoUseCase {
	return &fakeCargoUseCase{cargos: map[uuid.UUID]*domainusuarios.Cargo{}}
}

func (f *fakeCargoUseCase) Create(_ context.Context, nome, descricao string, administrador bool) (uuid.UUID, error) {
	if nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do cargo é obrigatório")
	}

	for _, cargo := range f.cargos {
		if cargo.Nome == nome {
			return uuid.Nil, domain.ErroValidacao("já existe um cargo com esse nome")
		}
	}

	id := uuid.New()
	f.cargos[id] = &domainusuarios.Cargo{
		ID:            id,
		Nome:          nome,
		Descricao:     descricao,
		Ativo:         true,
		Administrador: administrador,
	}

	return id, nil
}

func (f *fakeCargoUseCase) Update(_ context.Context, cargo *domainusuarios.Cargo) error {
	if _, ok := f.cargos[cargo.ID]; !ok {
		return domain.ErroNaoEncontrado("cargo não encontrado")
	}

	copia := *cargo
	f.cargos[cargo.ID] = &copia

	return nil
}

func (f *fakeCargoUseCase) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := f.cargos[id]
	if !ok {
		return nil, domain.ErroNaoEncontrado("cargo não encontrado")
	}

	copia := *cargo

	return &copia, nil
}

func (f *fakeCargoUseCase) GetByNome(_ context.Context, nome string) (*domainusuarios.Cargo, error) {
	for _, cargo := range f.cargos {
		if cargo.Nome == nome {
			copia := *cargo

			return &copia, nil
		}
	}

	return nil, domain.ErroNaoEncontrado("cargo não encontrado")
}

func (f *fakeCargoUseCase) List(_ context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	var itens []*domainusuarios.Cargo
	for _, cargo := range f.cargos {
		copia := *cargo
		itens = append(itens, &copia)
	}
	ordenarCargos(itens)

	return paginar(itens, filtro), nil
}

func (f *fakeCargoUseCase) ListAtivos(_ context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	var itens []*domainusuarios.Cargo
	for _, cargo := range f.cargos {
		if !cargo.Ativo {
			continue
		}

		copia := *cargo
		itens = append(itens, &copia)
	}
	ordenarCargos(itens)

	return paginar(itens, filtro), nil
}

func ordenarCargos(itens []*domainusuarios.Cargo) {
	sort.Slice(itens, func(i, j int) bool {
		return itens[i].Nome < itens[j].Nome
	})
}

func (f *fakeCargoUseCase) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.cargos[id]; !ok {
		return domain.ErroNaoEncontrado("cargo não encontrado")
	}

	delete(f.cargos, id)

	return nil
}

func semearCargo(t *testing.T, fake *fakeCargoUseCase, nome, descricao string) uuid.UUID {
	t.Helper()

	id, err := fake.Create(context.Background(), nome, descricao, false)
	if err != nil {
		t.Fatalf("falha ao semear cargo: %v", err)
	}

	return id
}

func TestCargoHandlerCriar(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	registrador := executarRequisicao(handler.Criar, http.MethodPost, "/api/v1/cargos", "", `{"nome":"Gerente","descricao":"Gerente de obra"}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusCreated, registrador.Body.String())
	}

	var resposta dtoRespostaCargo
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Nome != "Gerente" {
		t.Errorf("nome %q, esperado %q", resposta.Dados.Nome, "Gerente")
	}
	if !resposta.Dados.Ativo {
		t.Error("cargo criado deveria estar ativo")
	}

	registrador = executarRequisicao(handler.Criar, http.MethodPost, "/api/v1/cargos", "", `{"nome":"Gerente"}`)

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)

	registrador = executarRequisicao(handler.Criar, http.MethodPost, "/api/v1/cargos", "", `{}`)

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestCargoHandlerListarComFiltro(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	semearCargo(t, fake, "Gerente", "Gerente de obra")
	inativo := semearCargo(t, fake, "Pedreiro", "Mão de obra")
	fake.cargos[inativo].Ativo = false

	casos := []struct {
		nome      string
		caminho   string
		esperados int
		tamanho   int
	}{
		{"sem filtro", "/api/v1/cargos", 2, 20},
		{"somente ativos", "/api/v1/cargos?ativos=true", 1, 20},
		{"tamanho cortado", "/api/v1/cargos?tamanho=1", 1, 1},
	}

	for _, caso := range casos {
		registrador := executarRequisicao(handler.Listar, http.MethodGet, caso.caminho, "", "")

		if registrador.Code != http.StatusOK {
			t.Errorf("%s: status %d, esperado %d", caso.nome, registrador.Code, http.StatusOK)
			continue
		}

		var resposta dtoRespostaCargos
		if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("%s: corpo não é um envelope válido: %v", caso.nome, err)
		}
		if len(resposta.Dados) != caso.esperados {
			t.Errorf("%s: %d cargos, esperados %d", caso.nome, len(resposta.Dados), caso.esperados)
		}
		if resposta.Tamanho != caso.tamanho {
			t.Errorf("%s: tamanho %d, esperado %d", caso.nome, resposta.Tamanho, caso.tamanho)
		}
	}
}

func TestCargoHandlerObter(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	id := semearCargo(t, fake, "Gerente", "Gerente de obra")

	registrador := executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/cargos/"+id.String(), id.String(), "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}

	var resposta dtoRespostaCargo
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Descricao != "Gerente de obra" {
		t.Errorf("descrição %q, esperada %q", resposta.Dados.Descricao, "Gerente de obra")
	}

	desconhecido := uuid.NewString()
	registrador = executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/cargos/"+desconhecido, desconhecido, "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)

	registrador = executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/cargos/abc", "abc", "")

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestCargoHandlerObterPorNome(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	semearCargo(t, fake, "Gerente", "Gerente de obra")

	registrador := executarRequisicao(handler.ObterPorNome, http.MethodGet, "/api/v1/cargos/busca?nome=Gerente", "", "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}

	registrador = executarRequisicao(handler.ObterPorNome, http.MethodGet, "/api/v1/cargos/busca", "", "")

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)

	registrador = executarRequisicao(handler.ObterPorNome, http.MethodGet, "/api/v1/cargos/busca?nome=Diretor", "", "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}

func TestCargoHandlerAtualizar(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	id := semearCargo(t, fake, "Gerente", "Gerente de obra")
	registrador := executarRequisicao(handler.Atualizar, http.MethodPut, "/api/v1/cargos/"+id.String(), id.String(), `{"nome":"Diretor","descricao":"Diretor de obra","ativo":true}`)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}

	var resposta dtoRespostaCargo
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Nome != "Diretor" {
		t.Errorf("nome %q, esperado %q", resposta.Dados.Nome, "Diretor")
	}
}

func TestCargoHandlerRemover(t *testing.T) {
	fake := novoFakeCargos()
	handler := NewCargoHandler(fake)

	id := semearCargo(t, fake, "Gerente", "Gerente de obra")

	registrador := executarRequisicao(handler.Remover, http.MethodDelete, "/api/v1/cargos/"+id.String(), id.String(), "")

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}
	if _, ok := fake.cargos[id]; ok {
		t.Error("cargo não foi removido")
	}

	desconhecido := uuid.NewString()
	registrador = executarRequisicao(handler.Remover, http.MethodDelete, "/api/v1/cargos/"+desconhecido, desconhecido, "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}
