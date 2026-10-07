package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func novoCargo(nome string) *domainusuarios.Cargo {
	return &domainusuarios.Cargo{
		ID:        uuid.New(),
		Nome:      nome,
		Descricao: "Cargo de teste",
		Ativo:     true,
	}
}

func novoUsuario(cargoID uuid.UUID, nome, email string) *domainusuarios.Usuario {
	agora := time.Now().UTC()

	return &domainusuarios.Usuario{
		ID:           uuid.New(),
		Nome:         nome,
		Email:        email,
		Senha:        "hash-segredo",
		CargoID:      cargoID,
		Ativo:        true,
		UltimoLogin:  agora,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}
}

func cenarioUsuarios(t *testing.T) (*postgres.UsuarioRepository, *postgres.CargoRepository, *sqlx.DB) {
	t.Helper()

	banco := helpers.BancoDoTeste(t)

	return postgres.NewUsuarioRepository(banco), postgres.NewCargoRepository(banco), banco
}

var todasAsPaginas = domain.PaginacaoFiltro{Page: 1, Size: domain.TamanhoPaginaMaximo}

func TestCargoCriaAtualizaELista(t *testing.T) {
	_, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	criado := novoCargo("Engenheiro")
	id, err := cargos.Create(ctx, criado)
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("criação devolveu id zero")
	}

	salvo, err := cargos.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("cargo criado não foi encontrado")
	}
	if salvo.Nome != "Engenheiro" || !salvo.Ativo {
		t.Errorf("cargo salvo %+v, esperado nome %q ativo", salvo, "Engenheiro")
	}

	porNome, err := cargos.GetByNome(ctx, "Engenheiro")
	if err != nil {
		t.Fatalf("busca por nome falhou: %v", err)
	}
	if porNome == nil || porNome.ID != id {
		t.Errorf("busca por nome devolveu %+v, esperado o cargo %s", porNome, id)
	}

	inexistente, err := cargos.GetByNome(ctx, "Pedreiro")
	if err != nil {
		t.Fatalf("busca por nome inexistente falhou: %v", err)
	}
	if inexistente != nil {
		t.Errorf("esperava nenhum cargo, veio %+v", inexistente)
	}

	salvo.Nome = "Engenheiro Civil"
	salvo.Descricao = "Responsável técnico"
	salvo.Ativo = false
	if err := cargos.Update(ctx, salvo); err != nil {
		t.Fatalf("atualização do cargo falhou: %v", err)
	}

	atualizado, err := cargos.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após atualização falhou: %v", err)
	}
	if atualizado == nil {
		t.Fatal("cargo sumiu depois da atualização")
	}
	if atualizado.Nome != "Engenheiro Civil" || atualizado.Descricao != "Responsável técnico" || atualizado.Ativo {
		t.Errorf("cargo atualizado %+v, esperado nome %q e inativo", atualizado, "Engenheiro Civil")
	}

	if _, err := cargos.Create(ctx, novoCargo("Arquiteto")); err != nil {
		t.Fatalf("criação do segundo cargo falhou: %v", err)
	}
	if _, err := cargos.Create(ctx, novoCargo("Mestre de obras")); err != nil {
		t.Fatalf("criação do terceiro cargo falhou: %v", err)
	}

	todos, err := cargos.List(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(todos) != 3 {
		t.Errorf("listagem devolveu %d cargos, esperado 3", len(todos))
	}

	ativos, err := cargos.ListAtivos(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem de ativos falhou: %v", err)
	}
	if len(ativos) != 2 {
		t.Fatalf("listagem de ativos devolveu %d cargos, esperado 2", len(ativos))
	}
	if ativos[0].Nome != "Arquiteto" || ativos[1].Nome != "Mestre de obras" {
		t.Errorf("ordem inesperada: %q, %q", ativos[0].Nome, ativos[1].Nome)
	}
}

func TestCargoInexistenteDevolveNil(t *testing.T) {
	_, cargos, _ := cenarioUsuarios(t)

	salvo, err := cargos.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("busca por id inexistente falhou: %v", err)
	}
	if salvo != nil {
		t.Errorf("esperava nenhum cargo, veio %+v", salvo)
	}
}

func TestCargoComUsuarioNaoPodeSerExcluido(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	if _, err := usuarios.Create(ctx, novoUsuario(cargoID, "Ana Souza", "ana@exemplo.com")); err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	err = cargos.Delete(ctx, cargoID)
	if err == nil {
		t.Fatal("exclusão do cargo em uso deveria ter falhado")
	}
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %q, esperado erro de validação", err)
	}
}

func TestUsuarioCriaEBuscaPorIDEEEmail(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	criado := novoUsuario(cargoID, "Ana Souza", "ana@exemplo.com")
	id, err := usuarios.Create(ctx, criado)
	if err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("criação devolveu id zero")
	}

	porID, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if porID == nil {
		t.Fatal("usuário criado não foi encontrado")
	}
	if porID.Nome != "Ana Souza" || porID.Email != "ana@exemplo.com" {
		t.Errorf("usuário salvo %+v, esperado Ana Souza <ana@exemplo.com>", porID)
	}
	if porID.Senha != "hash-segredo" {
		t.Errorf("senha salva %q, esperada %q", porID.Senha, "hash-segredo")
	}
	if porID.CargoID != cargoID {
		t.Errorf("cargo do usuário %s, esperado %s", porID.CargoID, cargoID)
	}
	if !porID.Ativo {
		t.Error("usuário deveria ter sido criado ativo")
	}

	porEmail, err := usuarios.GetByEmail(ctx, "ana@exemplo.com")
	if err != nil {
		t.Fatalf("busca por email falhou: %v", err)
	}
	if porEmail == nil || porEmail.ID != id {
		t.Errorf("busca por email devolveu %+v, esperado o usuário %s", porEmail, id)
	}

	semRegistro, err := usuarios.GetByEmail(ctx, "nao@exemplo.com")
	if err != nil {
		t.Fatalf("busca por email inexistente falhou: %v", err)
	}
	if semRegistro != nil {
		t.Errorf("esperava nenhum usuário, veio %+v", semRegistro)
	}
}

func TestUsuarioInexistenteDevolveNil(t *testing.T) {
	usuarios, _, _ := cenarioUsuarios(t)

	salvo, err := usuarios.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("busca por id inexistente falhou: %v", err)
	}
	if salvo != nil {
		t.Errorf("esperava nenhum usuário, veio %+v", salvo)
	}
}

func TestUsuarioAtualiza(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}
	outroCargoID, err := cargos.Create(ctx, novoCargo("Almoxarife"))
	if err != nil {
		t.Fatalf("criação do segundo cargo falhou: %v", err)
	}

	id, err := usuarios.Create(ctx, novoUsuario(cargoID, "Ana Souza", "ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	salvo, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("usuário criado não foi encontrado")
	}

	salvo.Nome = "Ana Souza Lima"
	salvo.Email = "ana.lima@exemplo.com"
	salvo.Senha = "novo-hash"
	salvo.CargoID = outroCargoID
	salvo.Ativo = false
	salvo.AtualizadoEm = time.Now().UTC().Add(time.Hour)
	if err := usuarios.Update(ctx, salvo); err != nil {
		t.Fatalf("atualização do usuário falhou: %v", err)
	}

	atualizado, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após atualização falhou: %v", err)
	}
	if atualizado == nil {
		t.Fatal("usuário sumiu depois da atualização")
	}
	if atualizado.Nome != "Ana Souza Lima" || atualizado.Email != "ana.lima@exemplo.com" {
		t.Errorf("usuário atualizado %+v, esperado Ana Souza Lima <ana.lima@exemplo.com>", atualizado)
	}
	if atualizado.Senha != "novo-hash" {
		t.Errorf("senha atualizada %q, esperada %q", atualizado.Senha, "novo-hash")
	}
	if atualizado.CargoID != outroCargoID {
		t.Errorf("cargo do usuário %s, esperado %s", atualizado.CargoID, outroCargoID)
	}
	if atualizado.Ativo {
		t.Error("usuário deveria ter sido atualizado como inativo")
	}
	if !atualizado.AtualizadoEm.After(time.Now().UTC()) {
		t.Errorf("atualizado_em %s, esperado o valor enviado na atualização", atualizado.AtualizadoEm)
	}
}

func TestUsuarioListasEBusca(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	ids := make([]uuid.UUID, 0, 4)
	for _, item := range []struct {
		nome  string
		email string
	}{
		{"Ana Souza", "ana@exemplo.com"},
		{"Bruno Lima", "bruno@exemplo.com"},
		{"Carla Dias", "carla@exemplo.com"},
		{"Diego Ramos", "diego@exemplo.com"},
	} {
		id, err := usuarios.Create(ctx, novoUsuario(cargoID, item.nome, item.email))
		if err != nil {
			t.Fatalf("criação do usuário %q falhou: %v", item.nome, err)
		}
		ids = append(ids, id)
	}

	if err := usuarios.Desativar(ctx, ids[3]); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	todos, err := usuarios.List(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(todos) != 4 {
		t.Fatalf("listagem devolveu %d usuários, esperado 4", len(todos))
	}
	if todos[0].Nome != "Ana Souza" || todos[3].Nome != "Diego Ramos" {
		t.Errorf("ordem inesperada: %q, %q", todos[0].Nome, todos[3].Nome)
	}

	ativos, err := usuarios.ListAtivos(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem de ativos falhou: %v", err)
	}
	if len(ativos) != 3 {
		t.Fatalf("listagem de ativos devolveu %d usuários, esperado 3", len(ativos))
	}
	for _, item := range ativos {
		if item.ID == ids[3] {
			t.Error("usuário inativo apareceu na listagem de ativos")
		}
	}

	porNome, err := usuarios.Search(ctx, "SILVA", todasAsPaginas)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(porNome) != 0 {
		t.Errorf("busca por nome devolveu %d usuários, esperado 0", len(porNome))
	}

	porNome, err = usuarios.Search(ctx, "BRUNO", todasAsPaginas)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(porNome) != 1 || porNome[0].ID != ids[1] {
		t.Errorf("busca por nome devolveu %+v, esperado o usuário %s", porNome, ids[1])
	}

	porEmail, err := usuarios.Search(ctx, "carla@exemplo", todasAsPaginas)
	if err != nil {
		t.Fatalf("busca por email falhou: %v", err)
	}
	if len(porEmail) != 1 || porEmail[0].ID != ids[2] {
		t.Errorf("busca por email devolveu %+v, esperado o usuário %s", porEmail, ids[2])
	}
}

func TestUsuarioListaPaginada(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	for _, item := range []struct{ nome, email string }{
		{"Ana Souza", "ana@exemplo.com"},
		{"Bruno Lima", "bruno@exemplo.com"},
		{"Carla Dias", "carla@exemplo.com"},
	} {
		if _, err := usuarios.Create(ctx, novoUsuario(cargoID, item.nome, item.email)); err != nil {
			t.Fatalf("criação do usuário %q falhou: %v", item.nome, err)
		}
	}

	primeira, err := usuarios.List(ctx, domain.PaginacaoFiltro{Page: 1, Size: 2})
	if err != nil {
		t.Fatalf("primeira página falhou: %v", err)
	}
	if len(primeira) != 2 || primeira[0].Nome != "Ana Souza" || primeira[1].Nome != "Bruno Lima" {
		t.Errorf("primeira página devolveu %+v, esperado Ana Souza e Bruno Lima", nomes(primeira))
	}

	segunda, err := usuarios.List(ctx, domain.PaginacaoFiltro{Page: 2, Size: 2})
	if err != nil {
		t.Fatalf("segunda página falhou: %v", err)
	}
	if len(segunda) != 1 || segunda[0].Nome != "Carla Dias" {
		t.Errorf("segunda página devolveu %+v, esperado Carla Dias", nomes(segunda))
	}

	terceira, err := usuarios.List(ctx, domain.PaginacaoFiltro{Page: 3, Size: 2})
	if err != nil {
		t.Fatalf("terceira página falhou: %v", err)
	}
	if len(terceira) != 0 {
		t.Errorf("terceira página devolveu %d usuários, esperado 0", len(terceira))
	}
}

func nomes(itens []*domainusuarios.Usuario) []string {
	resultado := make([]string, 0, len(itens))
	for _, item := range itens {
		resultado = append(resultado, item.Nome)
	}

	return resultado
}

func TestUsuarioAtivaDesativaEAtualizaUltimoLogin(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	id, err := usuarios.Create(ctx, novoUsuario(cargoID, "Ana Souza", "ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	if err := usuarios.Desativar(ctx, id); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	desativado, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após desativação falhou: %v", err)
	}
	if desativado == nil || desativado.Ativo {
		t.Errorf("usuário %+v, esperado inativo", desativado)
	}

	if err := usuarios.Ativar(ctx, id); err != nil {
		t.Fatalf("ativação falhou: %v", err)
	}
	ativado, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após ativação falhou: %v", err)
	}
	if ativado == nil || !ativado.Ativo {
		t.Errorf("usuário %+v, esperado ativo", ativado)
	}

	login := time.Date(2026, time.October, 5, 10, 30, 0, 0, time.UTC)
	if err := usuarios.UpdateUltimoLogin(ctx, id, login); err != nil {
		t.Fatalf("atualização do último login falhou: %v", err)
	}
	comLogin, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após atualizar o último login falhou: %v", err)
	}
	if comLogin == nil {
		t.Fatal("usuário sumiu depois de atualizar o último login")
	}
	if !comLogin.UltimoLogin.Equal(login) {
		t.Errorf("último login %s, esperado %s", comLogin.UltimoLogin, login)
	}
}

func TestUsuarioDelete(t *testing.T) {
	usuarios, cargos, _ := cenarioUsuarios(t)
	ctx := context.Background()

	cargoID, err := cargos.Create(ctx, novoCargo("Gerente de obra"))
	if err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	id, err := usuarios.Create(ctx, novoUsuario(cargoID, "Ana Souza", "ana@exemplo.com"))
	if err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	if err := usuarios.Delete(ctx, id); err != nil {
		t.Fatalf("exclusão falhou: %v", err)
	}

	excluido, err := usuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após exclusão falhou: %v", err)
	}
	if excluido != nil {
		t.Errorf("usuário ainda existe depois da exclusão: %+v", excluido)
	}
}
