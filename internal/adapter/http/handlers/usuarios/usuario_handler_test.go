package usuarios

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type dtoRespostaUsuario = dto.Resposta[usuariosdto.UsuarioResponse]

type dtoRespostaUsuarios = dto.Resposta[[]usuariosdto.UsuarioResponse]

type fakeUsuarioUseCase struct {
	usuarios      map[uuid.UUID]*domainusuarios.Usuario
	ultimoLoginID uuid.UUID
	ultimoLogin   time.Time
}

func novoFakeUsuarios() *fakeUsuarioUseCase {
	return &fakeUsuarioUseCase{usuarios: map[uuid.UUID]*domainusuarios.Usuario{}}
}

func (f *fakeUsuarioUseCase) Create(_ context.Context, nome, email, senha string, cargoID uuid.UUID) (uuid.UUID, error) {
	if nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do usuário é obrigatório")
	}
	if email == "" {
		return uuid.Nil, domain.ErroValidacao("email do usuário é obrigatório")
	}
	if senha == "" {
		return uuid.Nil, domain.ErroValidacao("senha do usuário é obrigatória")
	}
	if cargoID == uuid.Nil {
		return uuid.Nil, domain.ErroValidacao("cargo do usuário é obrigatório")
	}

	id := uuid.New()
	f.usuarios[id] = &domainusuarios.Usuario{
		ID:           id,
		Nome:         nome,
		Email:        email,
		Senha:        senha,
		Ativo:        true,
		CargoID:      cargoID,
		CriadoEm:     time.Now().UTC(),
		AtualizadoEm: time.Now().UTC(),
	}

	return id, nil
}

func (f *fakeUsuarioUseCase) Update(_ context.Context, usuario *domainusuarios.Usuario) error {
	atual, ok := f.usuarios[usuario.ID]
	if !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	copia := *usuario
	copia.Senha = atual.Senha
	copia.CriadoEm = atual.CriadoEm
	copia.UltimoLogin = atual.UltimoLogin
	f.usuarios[usuario.ID] = &copia

	return nil
}

func (f *fakeUsuarioUseCase) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := f.usuarios[id]
	if !ok {
		return nil, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	copia := *usuario

	return &copia, nil
}

func (f *fakeUsuarioUseCase) GetByEmail(_ context.Context, email string) (*domainusuarios.Usuario, error) {
	for _, usuario := range f.usuarios {
		if usuario.Email == email {
			copia := *usuario

			return &copia, nil
		}
	}

	return nil, domain.ErroNaoEncontrado("usuário não encontrado")
}

func (f *fakeUsuarioUseCase) List(_ context.Context) ([]*domainusuarios.Usuario, error) {
	var itens []*domainusuarios.Usuario
	for _, usuario := range f.usuarios {
		copia := *usuario
		itens = append(itens, &copia)
	}

	return itens, nil
}

func (f *fakeUsuarioUseCase) ListAtivos(_ context.Context) ([]*domainusuarios.Usuario, error) {
	var itens []*domainusuarios.Usuario
	for _, usuario := range f.usuarios {
		if !usuario.Ativo {
			continue
		}

		copia := *usuario
		itens = append(itens, &copia)
	}

	return itens, nil
}

func (f *fakeUsuarioUseCase) Search(_ context.Context, termo string) ([]*domainusuarios.Usuario, error) {
	pesquisa := strings.ToLower(termo)

	var itens []*domainusuarios.Usuario
	for _, usuario := range f.usuarios {
		if !strings.Contains(strings.ToLower(usuario.Nome), pesquisa) && !strings.Contains(strings.ToLower(usuario.Email), pesquisa) {
			continue
		}

		copia := *usuario
		itens = append(itens, &copia)
	}

	return itens, nil
}

func (f *fakeUsuarioUseCase) Authenticate(_ context.Context, email, senha string) (*domainusuarios.Usuario, error) {
	for _, usuario := range f.usuarios {
		if usuario.Email != email {
			continue
		}
		if usuario.Senha != senha {
			return nil, domain.ErroNaoEncontrado("email ou senha inválidos")
		}

		copia := *usuario

		return &copia, nil
	}

	return nil, domain.ErroNaoEncontrado("email ou senha inválidos")
}

func (f *fakeUsuarioUseCase) UpdateSenha(_ context.Context, id uuid.UUID, novaSenha string) error {
	if novaSenha == "" {
		return domain.ErroValidacao("senha do usuário é obrigatória")
	}

	usuario, ok := f.usuarios[id]
	if !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	usuario.Senha = novaSenha

	return nil
}

func (f *fakeUsuarioUseCase) UpdateUltimoLogin(_ context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	if _, ok := f.usuarios[id]; !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	f.ultimoLoginID = id
	f.ultimoLogin = ultimoLogin

	return nil
}

func (f *fakeUsuarioUseCase) Ativar(_ context.Context, id uuid.UUID) error {
	return f.alterarAtivo(id, true)
}

func (f *fakeUsuarioUseCase) Desativar(_ context.Context, id uuid.UUID) error {
	return f.alterarAtivo(id, false)
}

func (f *fakeUsuarioUseCase) alterarAtivo(id uuid.UUID, ativo bool) error {
	usuario, ok := f.usuarios[id]
	if !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	usuario.Ativo = ativo

	return nil
}

func (f *fakeUsuarioUseCase) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.usuarios[id]; !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	delete(f.usuarios, id)

	return nil
}

func semearUsuario(t *testing.T, fake *fakeUsuarioUseCase, nome, email, senha string) uuid.UUID {
	t.Helper()

	id, err := fake.Create(context.Background(), nome, email, senha, uuid.New())
	if err != nil {
		t.Fatalf("falha ao semear usuário: %v", err)
	}

	return id
}

func TestUsuarioHandlerCriar(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	cargoID := uuid.NewString()
	corpo := `{"nome":"Ana Souza","email":"ana@exemplo.com","senha":"segredo123","cargo_id":"` + cargoID + `"}`
	registrador := executarRequisicao(handler.Criar, http.MethodPost, "/api/v1/usuarios", "", corpo)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusCreated, registrador.Body.String())
	}

	var resposta dtoRespostaUsuario
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Nome != "Ana Souza" {
		t.Errorf("nome %q, esperado %q", resposta.Dados.Nome, "Ana Souza")
	}
	if resposta.Dados.CargoID.String() != cargoID {
		t.Errorf("cargo %q, esperado %q", resposta.Dados.CargoID, cargoID)
	}
	if !resposta.Dados.Ativo {
		t.Error("usuário criado deveria estar ativo")
	}
	if strings.Contains(registrador.Body.String(), "senha") {
		t.Error("resposta expôs a senha")
	}
}

func TestUsuarioHandlerCriarRejeitaCorpoInvalido(t *testing.T) {
	handler := NewUsuarioHandler(novoFakeUsuarios())

	casos := []struct {
		nome  string
		corpo string
	}{
		{"campo desconhecido", `{"nome":"Ana Souza","senha":"segredo123"}`},
		{"senha vazia", `{"nome":"Ana Souza","email":"ana@exemplo.com","senha":""}`},
	}

	for _, caso := range casos {
		registrador := executarRequisicao(handler.Criar, http.MethodPost, "/api/v1/usuarios", "", caso.corpo)

		if registrador.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, esperado %d", caso.nome, registrador.Code, http.StatusBadRequest)
			continue
		}

		verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	}
}

func TestUsuarioHandlerObter(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")

	registrador := executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/usuarios/"+id.String(), id.String(), "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}

	var resposta dtoRespostaUsuario
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Email != "ana@exemplo.com" {
		t.Errorf("email %q, esperado %q", resposta.Dados.Email, "ana@exemplo.com")
	}

	desconhecido := uuid.NewString()
	registrador = executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/usuarios/"+desconhecido, desconhecido, "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)

	registrador = executarRequisicao(handler.Obter, http.MethodGet, "/api/v1/usuarios/abc", "abc", "")

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestUsuarioHandlerListarComFiltros(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")
	semearUsuario(t, fake, "Bruno Lima", "bruno@exemplo.com", "segredo123")
	inativo := semearUsuario(t, fake, "Carla Dias", "carla@exemplo.com", "segredo123")
	fake.usuarios[inativo].Ativo = false

	casos := []struct {
		nome      string
		caminho   string
		esperados int
	}{
		{"sem filtro", "/api/v1/usuarios", 3},
		{"somente ativos", "/api/v1/usuarios?ativos=true", 2},
		{"busca por texto", "/api/v1/usuarios?q=bruno", 1},
	}

	for _, caso := range casos {
		registrador := executarRequisicao(handler.Listar, http.MethodGet, caso.caminho, "", "")

		if registrador.Code != http.StatusOK {
			t.Errorf("%s: status %d, esperado %d", caso.nome, registrador.Code, http.StatusOK)
			continue
		}

		var resposta dtoRespostaUsuarios
		if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("%s: corpo não é um envelope válido: %v", caso.nome, err)
		}
		if len(resposta.Dados) != caso.esperados {
			t.Errorf("%s: %d usuários, esperados %d", caso.nome, len(resposta.Dados), caso.esperados)
		}
	}
}

func TestUsuarioHandlerAtualizar(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")
	cargoID := uuid.NewString()
	corpo := `{"nome":"Ana Alterada","email":"ana@exemplo.com","cargo_id":"` + cargoID + `","ativo":true}`
	registrador := executarRequisicao(handler.Atualizar, http.MethodPut, "/api/v1/usuarios/"+id.String(), id.String(), corpo)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}

	var resposta dtoRespostaUsuario
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Nome != "Ana Alterada" {
		t.Errorf("nome %q, esperado %q", resposta.Dados.Nome, "Ana Alterada")
	}
	if fake.usuarios[id].Senha != "segredo123" {
		t.Error("atualização não deveria alterar a senha")
	}
}

func TestUsuarioHandlerRemover(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")

	registrador := executarRequisicao(handler.Remover, http.MethodDelete, "/api/v1/usuarios/"+id.String(), id.String(), "")

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}

	desconhecido := uuid.NewString()
	registrador = executarRequisicao(handler.Remover, http.MethodDelete, "/api/v1/usuarios/"+desconhecido, desconhecido, "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}

func TestUsuarioHandlerTrocarSenha(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")

	registrador := executarRequisicao(handler.TrocarSenha, http.MethodPost, "/api/v1/usuarios/"+id.String()+"/senha", id.String(), `{"nova_senha":"nova12345"}`)

	if registrador.Code != http.StatusNoContent {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusNoContent, registrador.Body.String())
	}
	if fake.usuarios[id].Senha != "nova12345" {
		t.Errorf("senha %q, esperada %q", fake.usuarios[id].Senha, "nova12345")
	}

	registrador = executarRequisicao(handler.TrocarSenha, http.MethodPost, "/api/v1/usuarios/"+id.String()+"/senha", id.String(), `{"nova_senha":""}`)

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestUsuarioHandlerAtivarEDesativar(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")

	registrador := executarRequisicao(handler.Ativar, http.MethodPatch, "/api/v1/usuarios/"+id.String()+"/ativar", id.String(), "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}
	if !fake.usuarios[id].Ativo {
		t.Error("usuário deveria estar ativo")
	}

	registrador = executarRequisicao(handler.Desativar, http.MethodPatch, "/api/v1/usuarios/"+id.String()+"/desativar", id.String(), "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}
	if fake.usuarios[id].Ativo {
		t.Error("usuário deveria estar inativo")
	}

	desconhecido := uuid.NewString()
	registrador = executarRequisicao(handler.Ativar, http.MethodPatch, "/api/v1/usuarios/"+desconhecido+"/ativar", desconhecido, "")

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}

func TestUsuarioHandlerAutenticar(t *testing.T) {
	fake := novoFakeUsuarios()
	handler := NewUsuarioHandler(fake)

	id := semearUsuario(t, fake, "Ana Souza", "ana@exemplo.com", "segredo123")

	registrador := executarRequisicao(handler.Autenticar, http.MethodPost, "/api/v1/usuarios/autenticar", "", `{"email":"ana@exemplo.com","senha":"segredo123"}`)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}
	if fake.ultimoLoginID != id {
		t.Errorf("último login registrado para %q, esperado %q", fake.ultimoLoginID, id)
	}
	if fake.ultimoLogin.IsZero() {
		t.Error("último login não foi registrado")
	}

	var resposta dtoRespostaUsuario
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.UltimoLogin.IsZero() {
		t.Error("resposta não retornou o último login")
	}
	if strings.Contains(registrador.Body.String(), "senha") {
		t.Error("resposta expôs a senha")
	}

	registrador = executarRequisicao(handler.Autenticar, http.MethodPost, "/api/v1/usuarios/autenticar", "", `{"email":"ana@exemplo.com","senha":"errada123"}`)

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}
	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}
