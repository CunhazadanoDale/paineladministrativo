package usuarios

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type UsuarioUseCase interface {
	Criar(ctx context.Context, nome string, email string, senha string, cargoID uuid.UUID) (uuid.UUID, error)
	Atualizar(ctx context.Context, usuario *usuarios.Usuario) error
	Obter(ctx context.Context, id uuid.UUID) (*usuarios.Usuario, error)
	ObterPorEmail(ctx context.Context, email string) (*usuarios.Usuario, error)
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	Buscar(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	Autenticar(ctx context.Context, email string, senha string) (*usuarios.Usuario, error)
	EhAdministrador(ctx context.Context, usuario *usuarios.Usuario) (bool, error)
	TemAcessoComercial(ctx context.Context, usuario *usuarios.Usuario) (bool, error)
	AtualizarSenha(ctx context.Context, id uuid.UUID, novaSenha string) error
	TrocarSenhaPropria(ctx context.Context, id uuid.UUID, senhaAtual string, novaSenha string) error
	EncerrarSessoes(ctx context.Context, id uuid.UUID) error
	AtualizarUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error
	Ativar(ctx context.Context, id uuid.UUID) error
	Desativar(ctx context.Context, id uuid.UUID) error
	Remover(ctx context.Context, id uuid.UUID) error
}
