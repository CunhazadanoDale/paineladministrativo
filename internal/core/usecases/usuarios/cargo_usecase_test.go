package usuarios_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var _ portsout.CargoRepository = (*memoriaCargos)(nil)

type memoriaCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func novoRepositorioCargos() *memoriaCargos {
	return &memoriaCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
}

func (m *memoriaCargos) Create(ctx context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	if cargo.ID == uuid.Nil {
		return uuid.Nil, errors.New("id do cargo não informado")
	}

	copia := *cargo
	m.itens[cargo.ID] = &copia

	return cargo.ID, nil
}

func (m *memoriaCargos) Update(ctx context.Context, cargo *domainusuarios.Cargo) error {
	if _, ok := m.itens[cargo.ID]; !ok {
		return nil
	}

	copia := *cargo
	m.itens[cargo.ID] = &copia

	return nil
}

func (m *memoriaCargos) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := m.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

func (m *memoriaCargos) GetByNome(ctx context.Context, nome string) (*domainusuarios.Cargo, error) {
	for _, cargo := range m.itens {
		if cargo.Nome == nome {
			copia := *cargo
			return &copia, nil
		}
	}

	return nil, nil
}

func (m *memoriaCargos) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	itens := m.copias()
	ordenarCargos(itens)

	return paginar(itens, filtro), nil
}

func (m *memoriaCargos) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	var itens []*domainusuarios.Cargo
	for _, cargo := range m.copias() {
		if cargo.Ativo {
			itens = append(itens, cargo)
		}
	}
	ordenarCargos(itens)

	return paginar(itens, filtro), nil
}

func (m *memoriaCargos) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.itens, id)

	return nil
}

func (m *memoriaCargos) copias() []*domainusuarios.Cargo {
	itens := make([]*domainusuarios.Cargo, 0, len(m.itens))
	for _, cargo := range m.itens {
		copia := *cargo
		itens = append(itens, &copia)
	}

	return itens
}

func ordenarCargos(itens []*domainusuarios.Cargo) {
	sort.Slice(itens, func(i, j int) bool { return itens[i].Nome < itens[j].Nome })
}

func TestCargoCreate(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id, err := c.cargo.Create(ctx, "  Engenheiro  ", "  Responsável técnico  ", true, false)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("criação devolveu id zero")
	}

	salvo, err := c.cargo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("cargo criado não foi encontrado")
	}
	if salvo.Nome != "Engenheiro" || salvo.Descricao != "Responsável técnico" {
		t.Errorf("cargo salvo %+v, esperado Engenheiro <Responsável técnico>", salvo)
	}
	if !salvo.Ativo {
		t.Error("cargo deveria ter sido criado ativo")
	}
	if !salvo.Administrador {
		t.Error("cargo deveria ter sido criado como administrador")
	}

	if _, err := c.cargo.Create(ctx, "   ", "sem nome", false, false); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	if _, err := c.cargo.Create(ctx, "Engenheiro", "outro", false, false); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
}

func TestCargoGet(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	porNome, err := c.cargo.GetByNome(ctx, "  Gerente de obra  ")
	if err != nil {
		t.Fatalf("busca por nome falhou: %v", err)
	}
	if porNome == nil || porNome.ID != c.cargoID {
		t.Errorf("busca por nome devolveu %+v, esperado o cargo %s", porNome, c.cargoID)
	}

	if _, err := c.cargo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
	if _, err := c.cargo.GetByID(ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if _, err := c.cargo.GetByNome(ctx, "  "); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if _, err := c.cargo.GetByNome(ctx, "Pedreiro"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
}

func TestCargoUpdate(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	engenheiroID, err := c.cargo.Create(ctx, "Engenheiro", "", false, false)
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}
	if _, err := c.cargo.Create(ctx, "Arquiteto", "", false, false); err != nil {
		t.Fatalf("criação do segundo cargo falhou: %v", err)
	}

	salvo, err := c.cargo.GetByID(ctx, engenheiroID)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("cargo criado não foi encontrado")
	}

	if err := c.cargo.Update(ctx, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	semID := *salvo
	semID.ID = uuid.Nil
	if err := c.cargo.Update(ctx, &semID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	inexistente := *salvo
	inexistente.ID = uuid.New()
	if err := c.cargo.Update(ctx, &inexistente); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}

	nomeVazio := *salvo
	nomeVazio.Nome = "   "
	if err := c.cargo.Update(ctx, &nomeVazio); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	repetido := *salvo
	repetido.Nome = "Arquiteto"
	if err := c.cargo.Update(ctx, &repetido); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	atualizar := *salvo
	atualizar.Nome = "  Engenheiro Civil  "
	atualizar.Descricao = "  Responsável técnico  "
	atualizar.Ativo = false
	if err := c.cargo.Update(ctx, &atualizar); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	atualizado, err := c.cargo.GetByID(ctx, engenheiroID)
	if err != nil {
		t.Fatalf("busca após atualização falhou: %v", err)
	}
	if atualizado == nil {
		t.Fatal("cargo sumiu depois da atualização")
	}
	if atualizado.Nome != "Engenheiro Civil" || atualizado.Descricao != "Responsável técnico" {
		t.Errorf("cargo atualizado %+v, esperado Engenheiro Civil <Responsável técnico>", atualizado)
	}
	if atualizado.Ativo {
		t.Error("cargo deveria ter sido atualizado como inativo")
	}
}

func TestCargoDelete(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	if err := c.cargo.Delete(ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.cargo.Delete(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}

	if err := c.cargo.Delete(ctx, c.cargoID); err != nil {
		t.Fatalf("exclusão falhou: %v", err)
	}

	if _, err := c.cargo.GetByID(ctx, c.cargoID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
}

func TestCargoListas(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	if _, err := c.cargo.Create(ctx, "Engenheiro", "", false, false); err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}
	arquitetoID, err := c.cargo.Create(ctx, "Arquiteto", "", false, false)
	if err != nil {
		t.Fatalf("criação do segundo cargo falhou: %v", err)
	}

	todos, err := c.cargo.List(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("listagem devolveu %d cargos, esperado 3", len(todos))
	}
	if todos[0].Nome != "Arquiteto" || todos[1].Nome != "Engenheiro" || todos[2].Nome != "Gerente de obra" {
		t.Errorf("ordem inesperada: %q, %q, %q", todos[0].Nome, todos[1].Nome, todos[2].Nome)
	}

	arquiteto, err := c.cargo.GetByID(ctx, arquitetoID)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	arquiteto.Ativo = false
	if err := c.cargo.Update(ctx, arquiteto); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	ativos, err := c.cargo.ListAtivos(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem de ativos falhou: %v", err)
	}
	if len(ativos) != 2 {
		t.Fatalf("listagem de ativos devolveu %d cargos, esperado 2", len(ativos))
	}
	for _, item := range ativos {
		if item.ID == arquitetoID {
			t.Error("cargo inativo apareceu na listagem de ativos")
		}
	}
}
