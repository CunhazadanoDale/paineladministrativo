package usuarios

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	tamanhoMinimoSenha = 8
	tamanhoMaximoSenha = 72
)

var _ portsin.UsuarioUseCase = (*UsuarioUsecaseImpl)(nil)

type UsuarioUsecaseImpl struct {
	repo   portsout.UsuarioRepository
	cargos portsout.CargoRepository
}

func NewUsuarioUsecase(repo portsout.UsuarioRepository, cargos portsout.CargoRepository) *UsuarioUsecaseImpl {
	return &UsuarioUsecaseImpl{repo: repo, cargos: cargos}
}

func (u *UsuarioUsecaseImpl) Ativar(ctx context.Context, id uuid.UUID) error {
	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.Ativar(ctx, id)
}

func (u *UsuarioUsecaseImpl) Authenticate(ctx context.Context, email string, senha string) (*domainusuarios.Usuario, error) {
	email = normalizarEmail(email)

	if err := validarEmail(email); err != nil {
		return nil, err
	}
	if senha == "" {
		return nil, domain.ErroValidacao("senha do usuário é obrigatória")
	}

	usuario, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErroNaoEncontrado("email ou senha inválidos")
	}
	if bcrypt.CompareHashAndPassword([]byte(usuario.Senha), []byte(senha)) != nil {
		return nil, domain.ErroNaoEncontrado("email ou senha inválidos")
	}
	if !usuario.Ativo {
		return nil, domain.ErroValidacao("usuário desativado")
	}

	return usuario, nil
}

func (u *UsuarioUsecaseImpl) Create(ctx context.Context, nome string, email string, senha string, cargoID uuid.UUID) (uuid.UUID, error) {
	nome = strings.TrimSpace(nome)
	email = normalizarEmail(email)

	if nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do usuário é obrigatório")
	}
	if err := validarEmail(email); err != nil {
		return uuid.Nil, err
	}
	if err := validarSenha(senha); err != nil {
		return uuid.Nil, err
	}
	if cargoID == uuid.Nil {
		return uuid.Nil, domain.ErroValidacao("cargo do usuário é obrigatório")
	}

	cargo, err := u.cargos.GetByID(ctx, cargoID)
	if err != nil {
		return uuid.Nil, err
	}
	if cargo == nil {
		return uuid.Nil, domain.ErroValidacao("cargo do usuário não encontrado")
	}

	cadastrado, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, err
	}
	if cadastrado != nil {
		return uuid.Nil, domain.ErroValidacao("email já cadastrado")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}

	agora := time.Now().UTC()
	usuario := &domainusuarios.Usuario{
		ID:           uuid.New(),
		Nome:         nome,
		Email:        email,
		Senha:        string(hash),
		CargoID:      cargoID,
		Ativo:        true,
		UltimoLogin:  agora,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}

	return u.repo.Create(ctx, usuario)
}

func (u *UsuarioUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.Delete(ctx, id)
}

func (u *UsuarioUsecaseImpl) Desativar(ctx context.Context, id uuid.UUID) error {
	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.Desativar(ctx, id)
}

func (u *UsuarioUsecaseImpl) GetByEmail(ctx context.Context, email string) (*domainusuarios.Usuario, error) {
	email = normalizarEmail(email)

	if err := validarEmail(email); err != nil {
		return nil, err
	}

	usuario, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErrNotFound
	}

	return usuario, nil
}

func (u *UsuarioUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.buscar(ctx, id)
}

func (u *UsuarioUsecaseImpl) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.List(ctx, filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.ListAtivos(ctx, filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) Search(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.Search(ctx, strings.TrimSpace(termo), filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) Update(ctx context.Context, usuario *domainusuarios.Usuario) error {
	if usuario == nil || usuario.ID == uuid.Nil {
		return domain.ErroValidacao("usuário inválido")
	}

	usuario.Nome = strings.TrimSpace(usuario.Nome)
	usuario.Email = normalizarEmail(usuario.Email)

	if usuario.Nome == "" {
		return domain.ErroValidacao("nome do usuário é obrigatório")
	}
	if err := validarEmail(usuario.Email); err != nil {
		return err
	}
	if usuario.CargoID == uuid.Nil {
		return domain.ErroValidacao("cargo do usuário é obrigatório")
	}

	atual, err := u.buscar(ctx, usuario.ID)
	if err != nil {
		return err
	}

	if usuario.Email != atual.Email {
		cadastrado, err := u.repo.GetByEmail(ctx, usuario.Email)
		if err != nil {
			return err
		}
		if cadastrado != nil {
			return domain.ErroValidacao("email já cadastrado")
		}
	}

	if usuario.CargoID != atual.CargoID {
		cargo, err := u.cargos.GetByID(ctx, usuario.CargoID)
		if err != nil {
			return err
		}
		if cargo == nil {
			return domain.ErroValidacao("cargo do usuário não encontrado")
		}
	}

	usuario.Senha = atual.Senha
	usuario.UltimoLogin = atual.UltimoLogin
	usuario.CriadoEm = atual.CriadoEm
	usuario.AtualizadoEm = time.Now().UTC()

	return u.repo.Update(ctx, usuario)
}

func (u *UsuarioUsecaseImpl) UpdateSenha(ctx context.Context, id uuid.UUID, novaSenha string) error {
	if err := validarSenha(novaSenha); err != nil {
		return err
	}

	usuario, err := u.buscar(ctx, id)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	usuario.Senha = string(hash)
	usuario.AtualizadoEm = time.Now().UTC()

	return u.repo.Update(ctx, usuario)
}

func (u *UsuarioUsecaseImpl) UpdateUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	if ultimoLogin.IsZero() {
		return domain.ErroValidacao("data do último login inválida")
	}

	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.UpdateUltimoLogin(ctx, id, ultimoLogin)
}

func (u *UsuarioUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do usuário não informado")
	}

	usuario, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErrNotFound
	}

	return usuario, nil
}

func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validarEmail(email string) error {
	if email == "" {
		return domain.ErroValidacao("email do usuário é obrigatório")
	}

	local, dominio, achou := strings.Cut(email, "@")
	if !achou || local == "" || dominio == "" || strings.Contains(dominio, "@") || strings.ContainsAny(email, " \t") {
		return domain.ErroValidacao("email do usuário é inválido")
	}

	return nil
}

func validarSenha(senha string) error {
	switch {
	case senha == "":
		return domain.ErroValidacao("senha do usuário é obrigatória")
	case len(senha) < tamanhoMinimoSenha:
		return domain.ErroValidacao("senha do usuário deve ter no mínimo 8 caracteres")
	case len(senha) > tamanhoMaximoSenha:
		return domain.ErroValidacao("senha do usuário deve ter no máximo 72 caracteres")
	}

	return nil
}
