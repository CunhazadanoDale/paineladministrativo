package usuarios_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/usuarios"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var _ portsout.UsuarioRepository = (*memoriaUsuarios)(nil)

var todasAsPaginas = domain.PaginacaoFiltro{Page: 1, Size: domain.TamanhoPaginaMaximo}

func paginar[T any](itens []T, filtro domain.PaginacaoFiltro) []T {
	filtro = filtro.Normalizada()

	inicio := (filtro.Page - 1) * filtro.Size
	if inicio >= len(itens) {
		return nil
	}

	fim := inicio + filtro.Size
	if fim > len(itens) {
		fim = len(itens)
	}

	return itens[inicio:fim]
}

type memoriaUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func novoRepositorioUsuarios() *memoriaUsuarios {
	return &memoriaUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
}

func (m *memoriaUsuarios) Create(ctx context.Context, usuario *domainusuarios.Usuario) (uuid.UUID, error) {
	if usuario.ID == uuid.Nil {
		return uuid.Nil, errors.New("id do usuário não informado")
	}

	copia := *usuario
	m.itens[usuario.ID] = &copia

	return usuario.ID, nil
}

func (m *memoriaUsuarios) Update(ctx context.Context, usuario *domainusuarios.Usuario) error {
	if _, ok := m.itens[usuario.ID]; !ok {
		return nil
	}

	copia := *usuario
	m.itens[usuario.ID] = &copia

	return nil
}

func (m *memoriaUsuarios) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := m.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

func (m *memoriaUsuarios) GetByEmail(ctx context.Context, email string) (*domainusuarios.Usuario, error) {
	for _, usuario := range m.itens {
		if usuario.Email == email {
			copia := *usuario
			return &copia, nil
		}
	}

	return nil, nil
}

func (m *memoriaUsuarios) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	itens := m.copias()
	ordenarUsuarios(itens)

	return paginar(itens, filtro), nil
}

func (m *memoriaUsuarios) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	var itens []*domainusuarios.Usuario
	for _, usuario := range m.copias() {
		if usuario.Ativo {
			itens = append(itens, usuario)
		}
	}
	ordenarUsuarios(itens)

	return paginar(itens, filtro), nil
}

func (m *memoriaUsuarios) Search(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	termo = strings.ToLower(termo)

	var itens []*domainusuarios.Usuario
	for _, usuario := range m.copias() {
		if strings.Contains(strings.ToLower(usuario.Nome), termo) || strings.Contains(strings.ToLower(usuario.Email), termo) {
			itens = append(itens, usuario)
		}
	}
	ordenarUsuarios(itens)

	return paginar(itens, filtro), nil
}

func (m *memoriaUsuarios) UpdateUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	usuario, ok := m.itens[id]
	if !ok {
		return nil
	}

	usuario.UltimoLogin = ultimoLogin

	return nil
}

func (m *memoriaUsuarios) AtualizarSenha(ctx context.Context, id uuid.UUID, senha string, atualizadoEm time.Time) error {
	usuario, ok := m.itens[id]
	if !ok {
		return nil
	}

	usuario.Senha = senha
	usuario.VersaoSessao++
	usuario.AtualizadoEm = atualizadoEm

	return nil
}

func (m *memoriaUsuarios) EncerrarSessoes(ctx context.Context, id uuid.UUID) error {
	if usuario, ok := m.itens[id]; ok {
		usuario.VersaoSessao++
	}

	return nil
}

func (m *memoriaUsuarios) Ativar(ctx context.Context, id uuid.UUID) error {
	if usuario, ok := m.itens[id]; ok {
		usuario.Ativo = true
	}

	return nil
}

func (m *memoriaUsuarios) Desativar(ctx context.Context, id uuid.UUID) error {
	if usuario, ok := m.itens[id]; ok {
		usuario.Ativo = false
	}

	return nil
}

func (m *memoriaUsuarios) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.itens, id)

	return nil
}

func (m *memoriaUsuarios) copias() []*domainusuarios.Usuario {
	itens := make([]*domainusuarios.Usuario, 0, len(m.itens))
	for _, usuario := range m.itens {
		copia := *usuario
		itens = append(itens, &copia)
	}

	return itens
}

func ordenarUsuarios(itens []*domainusuarios.Usuario) {
	sort.Slice(itens, func(i, j int) bool { return itens[i].Nome < itens[j].Nome })
}

type cenario struct {
	usuario      *usuarios.UsuarioUsecaseImpl
	cargo        *usuarios.CargoUsecaseImpl
	repoUsuarios *memoriaUsuarios
	repoCargos   *memoriaCargos
	cargoID      uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	repoCargos := novoRepositorioCargos()
	repoUsuarios := novoRepositorioUsuarios()
	cargos := usuarios.NewCargoUsecase(repoCargos)

	cargoID, err := cargos.Create(context.Background(), "Gerente de obra", "Responsável pela obra", false, false)
	if err != nil {
		t.Fatalf("não criei o cargo do cenário: %v", err)
	}

	return &cenario{
		usuario:      usuarios.NewUsuarioUsecase(repoUsuarios, repoCargos),
		cargo:        cargos,
		repoUsuarios: repoUsuarios,
		repoCargos:   repoCargos,
		cargoID:      cargoID,
	}
}

func (c *cenario) criarUsuario(t *testing.T, nome, email, senha string) uuid.UUID {
	t.Helper()

	id, err := c.usuario.Create(context.Background(), nome, email, senha, c.cargoID)
	if err != nil {
		t.Fatalf("não criei o usuário %q: %v", email, err)
	}

	return id
}

func TestUsuarioCreateProtegeASenha(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id, err := c.usuario.Create(ctx, "  Ana Souza  ", "ANA@Exemplo.com", "segredo123", c.cargoID)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("criação devolveu id zero")
	}

	salvo, err := c.repoUsuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca no repositório falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("usuário não foi gravado")
	}
	if salvo.Senha == "segredo123" {
		t.Fatal("senha gravada sem hash")
	}
	if bcrypt.CompareHashAndPassword([]byte(salvo.Senha), []byte("segredo123")) != nil {
		t.Error("hash gravado não confere com a senha enviada")
	}
	if salvo.Nome != "Ana Souza" {
		t.Errorf("nome %q, esperado %q", salvo.Nome, "Ana Souza")
	}
	if salvo.Email != "ana@exemplo.com" {
		t.Errorf("email %q, esperado %q", salvo.Email, "ana@exemplo.com")
	}
	if salvo.CargoID != c.cargoID {
		t.Errorf("cargo %s, esperado %s", salvo.CargoID, c.cargoID)
	}
	if !salvo.Ativo {
		t.Error("usuário deveria ter sido criado ativo")
	}
	if salvo.CriadoEm.IsZero() || salvo.AtualizadoEm.IsZero() || salvo.UltimoLogin.IsZero() {
		t.Error("datas do usuário não foram preenchidas")
	}
}

func TestUsuarioCreateValidaEntrada(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	casos := []struct {
		nome    string
		email   string
		senha   string
		cargoID uuid.UUID
		trecho  string
	}{
		{"", "ana@exemplo.com", "segredo123", c.cargoID, "nome do usuário é obrigatório"},
		{"Ana Souza", "", "segredo123", c.cargoID, "email do usuário é obrigatório"},
		{"Ana Souza", "ana.exemplo.com", "segredo123", c.cargoID, "email do usuário é inválido"},
		{"Ana Souza", "ana@@exemplo.com", "segredo123", c.cargoID, "email do usuário é inválido"},
		{"Ana Souza", "ana@exemplo.com", "", c.cargoID, "senha do usuário é obrigatória"},
		{"Ana Souza", "ana@exemplo.com", "1234567", c.cargoID, "mínimo"},
		{"Ana Souza", "ana@exemplo.com", strings.Repeat("a", 73), c.cargoID, "máximo"},
		{"Ana Souza", "ana@exemplo.com", "segredo123", uuid.Nil, "cargo do usuário é obrigatório"},
		{"Ana Souza", "ana@exemplo.com", "segredo123", uuid.New(), "não encontrado"},
	}

	for _, caso := range casos {
		_, err := c.usuario.Create(ctx, caso.nome, caso.email, caso.senha, caso.cargoID)
		if err == nil {
			t.Errorf("%q: criação deveria ter falhado", caso.email)
			continue
		}
		if !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%q: erro %v, esperado erro de validação", caso.email, err)
			continue
		}
		if !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%q: mensagem %q, esperada menção a %q", caso.email, err, caso.trecho)
		}
	}
}

func TestUsuarioCreateRejeitaEmailDuplicado(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	_, err := c.usuario.Create(ctx, "Ana Lima", "ANA@exemplo.com", "outrasenha1", c.cargoID)
	if !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("erro %v, esperado erro de validação", err)
	}
	if !strings.Contains(err.Error(), "já cadastrado") {
		t.Errorf("mensagem %q, esperada menção a email já cadastrado", err)
	}
}

func TestUsuarioGetByEmail(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	encontrado, err := c.usuario.GetByEmail(ctx, "  ANA@Exemplo.COM  ")
	if err != nil {
		t.Fatalf("busca por email falhou: %v", err)
	}
	if encontrado == nil || encontrado.Nome != "Ana Souza" {
		t.Errorf("busca devolveu %+v, esperado Ana Souza", encontrado)
	}

	if _, err := c.usuario.GetByEmail(ctx, "outra@exemplo.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}

	if _, err := c.usuario.GetByEmail(ctx, "sem-arroba"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
}

func TestUsuarioAuthenticate(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	autenticado, err := c.usuario.Authenticate(ctx, "ANA@Exemplo.com", "segredo123")
	if err != nil {
		t.Fatalf("autenticação falhou: %v", err)
	}
	if autenticado == nil || autenticado.Nome != "Ana Souza" {
		t.Fatalf("autenticação devolveu %+v, esperado Ana Souza", autenticado)
	}

	_, errSenha := c.usuario.Authenticate(ctx, "ana@exemplo.com", "senhaerrada")
	if !errors.Is(errSenha, domain.ErrNotFound) {
		t.Errorf("senha errada devolveu %v, esperado registro não encontrado", errSenha)
	}

	_, errEmail := c.usuario.Authenticate(ctx, "outra@exemplo.com", "segredo123")
	if !errors.Is(errEmail, domain.ErrNotFound) {
		t.Errorf("email inexistente devolveu %v, esperado registro não encontrado", errEmail)
	}
	if errEmail.Error() != errSenha.Error() {
		t.Errorf("mensagens diferentes: %q e %q", errEmail, errSenha)
	}

	if _, err := c.usuario.Authenticate(ctx, "sem-arroba", "segredo123"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
}

func TestUsuarioAuthenticateBloqueiaDesativado(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")
	if err := c.usuario.Desativar(ctx, id); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	_, err := c.usuario.Authenticate(ctx, "ana@exemplo.com", "segredo123")
	if !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("erro %v, esperado erro de validação", err)
	}
	if !strings.Contains(err.Error(), "desativado") {
		t.Errorf("mensagem %q, esperada menção a usuário desativado", err)
	}
}

func TestUsuarioUpdateSenha(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	if err := c.usuario.UpdateSenha(ctx, id, "senhaNova123"); err != nil {
		t.Fatalf("troca de senha falhou: %v", err)
	}

	salvo, err := c.repoUsuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca no repositório falhou: %v", err)
	}
	if salvo.Senha == "senhaNova123" || salvo.Senha == "segredo123" {
		t.Fatal("nova senha gravada sem hash")
	}
	if bcrypt.CompareHashAndPassword([]byte(salvo.Senha), []byte("senhaNova123")) != nil {
		t.Error("hash gravado não confere com a nova senha")
	}

	if _, err := c.usuario.Authenticate(ctx, "ana@exemplo.com", "senhaNova123"); err != nil {
		t.Errorf("nova senha não autenticou: %v", err)
	}
	if _, err := c.usuario.Authenticate(ctx, "ana@exemplo.com", "segredo123"); err == nil {
		t.Error("senha antiga ainda autentica")
	}

	if err := c.usuario.UpdateSenha(ctx, id, "curta"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.usuario.UpdateSenha(ctx, uuid.New(), "senhaNova123"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
}

func TestUsuarioUpdateSenhaEncerraSessoes(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	if err := c.usuario.UpdateSenha(ctx, id, "senhaNova123"); err != nil {
		t.Fatalf("troca de senha falhou: %v", err)
	}

	salvo, err := c.repoUsuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca no repositório falhou: %v", err)
	}
	if salvo.VersaoSessao != 1 {
		t.Errorf("versão da sessão %d, esperada 1", salvo.VersaoSessao)
	}
}

func TestUsuarioTrocarSenhaPropriaExigeSenhaAtual(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	casos := []struct {
		nome       string
		senhaAtual string
		novaSenha  string
	}{
		{"senha atual vazia", "", "senhaNova123"},
		{"senha atual incorreta", "errada123", "senhaNova123"},
		{"nova senha curta", "segredo123", "curta"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := c.usuario.TrocarSenhaPropria(ctx, id, caso.senhaAtual, caso.novaSenha); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := c.usuario.Authenticate(ctx, "ana@exemplo.com", "segredo123"); err != nil {
		t.Errorf("senha original deixou de autenticar: %v", err)
	}

	if err := c.usuario.TrocarSenhaPropria(ctx, id, "segredo123", "senhaNova123"); err != nil {
		t.Fatalf("troca de senha falhou: %v", err)
	}
	if _, err := c.usuario.Authenticate(ctx, "ana@exemplo.com", "senhaNova123"); err != nil {
		t.Errorf("nova senha não autenticou: %v", err)
	}

	salvo, err := c.repoUsuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca no repositório falhou: %v", err)
	}
	if salvo.VersaoSessao != 1 {
		t.Errorf("versão da sessão %d, esperada 1", salvo.VersaoSessao)
	}
}

func TestUsuarioEncerrarSessoes(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	if err := c.usuario.EncerrarSessoes(ctx, id); err != nil {
		t.Fatalf("encerramento falhou: %v", err)
	}

	salvo, err := c.repoUsuarios.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca no repositório falhou: %v", err)
	}
	if salvo.VersaoSessao != 1 {
		t.Errorf("versão da sessão %d, esperada 1", salvo.VersaoSessao)
	}
	if err := c.usuario.EncerrarSessoes(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
}

func TestUsuarioUpdatePreservaSenhaEUltimoLogin(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")

	login := time.Date(2026, time.October, 5, 10, 30, 0, 0, time.UTC)
	if err := c.usuario.UpdateUltimoLogin(ctx, id, login); err != nil {
		t.Fatalf("atualização do último login falhou: %v", err)
	}

	usuario, err := c.usuario.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}

	hashAntes := usuario.Senha
	usuario.Nome = "  Ana Souza Lima  "
	usuario.Email = "ANA.LIMA@exemplo.com"
	usuario.Senha = "senha que não deveria ser gravada"
	usuario.UltimoLogin = time.Time{}

	if err := c.usuario.Update(ctx, usuario); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	atualizado, err := c.usuario.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca após atualização falhou: %v", err)
	}
	if atualizado.Nome != "Ana Souza Lima" {
		t.Errorf("nome %q, esperado %q", atualizado.Nome, "Ana Souza Lima")
	}
	if atualizado.Email != "ana.lima@exemplo.com" {
		t.Errorf("email %q, esperado %q", atualizado.Email, "ana.lima@exemplo.com")
	}
	if atualizado.Senha != hashAntes {
		t.Error("atualização sobrescreveu a senha do usuário")
	}
	if !atualizado.UltimoLogin.Equal(login) {
		t.Errorf("último login %s, esperado %s", atualizado.UltimoLogin, login)
	}
}

func TestUsuarioUpdateValida(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	id := c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")
	outroID := c.criarUsuario(t, "Bruno Lima", "bruno@exemplo.com", "segredo123")

	usuario, err := c.usuario.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}

	if err := c.usuario.Update(ctx, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	semID := *usuario
	semID.ID = uuid.Nil
	if err := c.usuario.Update(ctx, &semID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	nomeVazio := *usuario
	nomeVazio.Nome = "   "
	if err := c.usuario.Update(ctx, &nomeVazio); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	emailInvalido := *usuario
	emailInvalido.Email = "ana.exemplo.com"
	if err := c.usuario.Update(ctx, &emailInvalido); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	semCargo := *usuario
	semCargo.CargoID = uuid.Nil
	if err := c.usuario.Update(ctx, &semCargo); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	inexistente := *usuario
	inexistente.ID = uuid.New()
	if err := c.usuario.Update(ctx, &inexistente); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}

	cargoInexistente := *usuario
	cargoInexistente.CargoID = uuid.New()
	if err := c.usuario.Update(ctx, &cargoInexistente); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}

	emailRepetido := *usuario
	emailRepetido.Email = "bruno@exemplo.com"
	if err := c.usuario.Update(ctx, &emailRepetido); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if outroID == uuid.Nil {
		t.Fatal("segundo usuário não foi criado")
	}
}

func TestUsuarioOperacoesComIdInexistente(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()
	desconhecido := uuid.New()

	if _, err := c.usuario.GetByID(ctx, desconhecido); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
	if err := c.usuario.Ativar(ctx, desconhecido); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
	if err := c.usuario.Desativar(ctx, desconhecido); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
	if err := c.usuario.Delete(ctx, desconhecido); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}
	if err := c.usuario.UpdateUltimoLogin(ctx, desconhecido, time.Now().UTC()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro %v, esperado registro não encontrado", err)
	}

	if _, err := c.usuario.GetByID(ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.usuario.Ativar(ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.usuario.Delete(ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.usuario.UpdateUltimoLogin(ctx, uuid.Nil, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if err := c.usuario.UpdateUltimoLogin(ctx, c.cargoID, time.Time{}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
}

func TestUsuarioListasEBusca(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	c.criarUsuario(t, "Ana Souza", "ana@exemplo.com", "segredo123")
	c.criarUsuario(t, "Bruno Lima", "bruno@exemplo.com", "segredo123")
	diego := c.criarUsuario(t, "Diego Ramos", "diego@exemplo.com", "segredo123")

	todos, err := c.usuario.List(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("listagem devolveu %d usuários, esperado 3", len(todos))
	}
	if todos[0].Nome != "Ana Souza" || todos[1].Nome != "Bruno Lima" || todos[2].Nome != "Diego Ramos" {
		t.Errorf("ordem inesperada: %q, %q, %q", todos[0].Nome, todos[1].Nome, todos[2].Nome)
	}

	if err := c.usuario.Desativar(ctx, diego); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	ativos, err := c.usuario.ListAtivos(ctx, todasAsPaginas)
	if err != nil {
		t.Fatalf("listagem de ativos falhou: %v", err)
	}
	if len(ativos) != 2 {
		t.Fatalf("listagem de ativos devolveu %d usuários, esperado 2", len(ativos))
	}
	for _, item := range ativos {
		if item.ID == diego {
			t.Error("usuário inativo apareceu na listagem de ativos")
		}
	}

	encontrados, err := c.usuario.Search(ctx, "  BRUNO  ", todasAsPaginas)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(encontrados) != 1 || encontrados[0].Nome != "Bruno Lima" {
		t.Errorf("busca devolveu %+v, esperado Bruno Lima", encontrados)
	}

	porEmail, err := c.usuario.Search(ctx, "ANA@exemplo", todasAsPaginas)
	if err != nil {
		t.Fatalf("busca por email falhou: %v", err)
	}
	if len(porEmail) != 1 || porEmail[0].Email != "ana@exemplo.com" {
		t.Errorf("busca devolveu %+v, esperado ana@exemplo.com", porEmail)
	}
}
