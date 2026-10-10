package usuarios

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type UsuarioRepository interface {
	Criar(ctx context.Context, usuario *usuarios.Usuario) (uuid.UUID, error)
	Atualizar(ctx context.Context, usuario *usuarios.Usuario) error
	Obter(ctx context.Context, id uuid.UUID) (*usuarios.Usuario, error)
	ObterPorEmail(ctx context.Context, email string) (*usuarios.Usuario, error)
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	Buscar(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	AtualizarUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error
	AtualizarSenha(ctx context.Context, id uuid.UUID, senha string, atualizadoEm time.Time) error
	EncerrarSessoes(ctx context.Context, id uuid.UUID) error
	ContarAdministradoresAtivos(ctx context.Context) (int, error)
	ContarAtivosPorCargo(ctx context.Context, cargoID uuid.UUID) (int, error)
	Ativar(ctx context.Context, id uuid.UUID) error
	Desativar(ctx context.Context, id uuid.UUID) error
	Remover(ctx context.Context, id uuid.UUID) error
}
