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

const mensagemUltimoAdministrador = "o sistema precisa de pelo menos um administrador ativo"

const hashFicticio = "$2a$10$oj4O7jHgLO6q7cx8igQd2OVrpG9VYkPpkeT7RT7Hq82wnq9A3O.3C"

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

func (u *UsuarioUsecaseImpl) Autenticar(ctx context.Context, email string, senha string) (*domainusuarios.Usuario, error) {
	email = normalizarEmail(email)

	if err := validarEmail(email); err != nil {
		return nil, err
	}
	if senha == "" {
		return nil, domain.ErroValidacao("senha do usuário é obrigatória")
	}

	usuario, err := u.repo.ObterPorEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		_ = bcrypt.CompareHashAndPassword([]byte(hashFicticio), []byte(senha))
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

func (u *UsuarioUsecaseImpl) EhAdministrador(ctx context.Context, usuario *domainusuarios.Usuario) (bool, error) {
	cargo, err := u.cargoDoUsuario(ctx, usuario)
	if err != nil {
		return false, err
	}

	return cargo != nil && cargo.Administrador, nil
}

func (u *UsuarioUsecaseImpl) TemAcessoComercial(ctx context.Context, usuario *domainusuarios.Usuario) (bool, error) {
	cargo, err := u.cargoDoUsuario(ctx, usuario)
	if err != nil {
		return false, err
	}

	return cargo != nil && (cargo.Comercial || cargo.Administrador), nil
}

func (u *UsuarioUsecaseImpl) cargoDoUsuario(ctx context.Context, usuario *domainusuarios.Usuario) (*domainusuarios.Cargo, error) {
	if usuario == nil || usuario.CargoID == uuid.Nil {
		return nil, nil
	}

	return u.cargos.Obter(ctx, usuario.CargoID)
}

func (u *UsuarioUsecaseImpl) Criar(ctx context.Context, nome string, email string, senha string, cargoID uuid.UUID) (uuid.UUID, error) {
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

	cargo, err := u.cargos.Obter(ctx, cargoID)
	if err != nil {
		return uuid.Nil, err
	}
	if cargo == nil {
		return uuid.Nil, domain.ErroValidacao("cargo do usuário não encontrado")
	}

	cadastrado, err := u.repo.ObterPorEmail(ctx, email)
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

	return u.repo.Criar(ctx, usuario)
}

func (u *UsuarioUsecaseImpl) Remover(ctx context.Context, id uuid.UUID) error {
	atual, err := u.buscar(ctx, id)
	if err != nil {
		return err
	}
	if err := u.garantirOutroAdministrador(ctx, atual); err != nil {
		return err
	}

	return u.repo.Remover(ctx, id)
}

func (u *UsuarioUsecaseImpl) Desativar(ctx context.Context, id uuid.UUID) error {
	atual, err := u.buscar(ctx, id)
	if err != nil {
		return err
	}
	if err := u.garantirOutroAdministrador(ctx, atual); err != nil {
		return err
	}

	return u.repo.Desativar(ctx, id)
}

func (u *UsuarioUsecaseImpl) ObterPorEmail(ctx context.Context, email string) (*domainusuarios.Usuario, error) {
	email = normalizarEmail(email)

	if err := validarEmail(email); err != nil {
		return nil, err
	}

	usuario, err := u.repo.ObterPorEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, domain.ErrNotFound
	}

	return usuario, nil
}

func (u *UsuarioUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.buscar(ctx, id)
}

func (u *UsuarioUsecaseImpl) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.Listar(ctx, filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.ListarAtivos(ctx, filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) Buscar(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return u.repo.Buscar(ctx, strings.TrimSpace(termo), filtro.Normalizada())
}

func (u *UsuarioUsecaseImpl) Atualizar(ctx context.Context, usuario *domainusuarios.Usuario) error {
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
		cadastrado, err := u.repo.ObterPorEmail(ctx, usuario.Email)
		if err != nil {
			return err
		}
		if cadastrado != nil {
			return domain.ErroValidacao("email já cadastrado")
		}
	}

	continuaAdministrador := usuario.Ativo
	if usuario.CargoID != atual.CargoID {
		cargo, err := u.cargos.Obter(ctx, usuario.CargoID)
		if err != nil {
			return err
		}
		if cargo == nil {
			return domain.ErroValidacao("cargo do usuário não encontrado")
		}
		continuaAdministrador = continuaAdministrador && cargo.Administrador
	} else if continuaAdministrador {
		continuaAdministrador, err = u.EhAdministrador(ctx, atual)
		if err != nil {
			return err
		}
	}
	if !continuaAdministrador {
		if err := u.garantirOutroAdministrador(ctx, atual); err != nil {
			return err
		}
	}

	usuario.Senha = atual.Senha
	usuario.UltimoLogin = atual.UltimoLogin
	usuario.CriadoEm = atual.CriadoEm
	usuario.AtualizadoEm = time.Now().UTC()

	return u.repo.Atualizar(ctx, usuario)
}

func (u *UsuarioUsecaseImpl) AtualizarSenha(ctx context.Context, id uuid.UUID, novaSenha string) error {
	if err := validarSenha(novaSenha); err != nil {
		return err
	}

	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.gravarSenha(ctx, id, novaSenha)
}

func (u *UsuarioUsecaseImpl) TrocarSenhaPropria(ctx context.Context, id uuid.UUID, senhaAtual string, novaSenha string) error {
	if senhaAtual == "" {
		return domain.ErroValidacao("senha atual é obrigatória")
	}
	if err := validarSenha(novaSenha); err != nil {
		return err
	}

	usuario, err := u.buscar(ctx, id)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(usuario.Senha), []byte(senhaAtual)) != nil {
		return domain.ErroValidacao("senha atual incorreta")
	}

	return u.gravarSenha(ctx, id, novaSenha)
}

func (u *UsuarioUsecaseImpl) EncerrarSessoes(ctx context.Context, id uuid.UUID) error {
	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.EncerrarSessoes(ctx, id)
}

func (u *UsuarioUsecaseImpl) gravarSenha(ctx context.Context, id uuid.UUID, novaSenha string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return u.repo.AtualizarSenha(ctx, id, string(hash), time.Now().UTC())
}

func (u *UsuarioUsecaseImpl) AtualizarUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	if ultimoLogin.IsZero() {
		return domain.ErroValidacao("data do último login inválida")
	}

	if _, err := u.buscar(ctx, id); err != nil {
		return err
	}

	return u.repo.AtualizarUltimoLogin(ctx, id, ultimoLogin)
}

func (u *UsuarioUsecaseImpl) garantirOutroAdministrador(ctx context.Context, usuario *domainusuarios.Usuario) error {
	if !usuario.Ativo {
		return nil
	}

	administrador, err := u.EhAdministrador(ctx, usuario)
	if err != nil || !administrador {
		return err
	}

	total, err := u.repo.ContarAdministradoresAtivos(ctx)
	if err != nil {
		return err
	}
	if total <= 1 {
		return domain.ErroConflito(mensagemUltimoAdministrador)
	}

	return nil
}

func (u *UsuarioUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do usuário não informado")
	}

	usuario, err := u.repo.Obter(ctx, id)
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
