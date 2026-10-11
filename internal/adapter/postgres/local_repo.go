package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.LocalRepository = (*LocalRepository)(nil)

type LocalRepository struct {
	db *sqlx.DB
}

func NewLocalRepository(db *sqlx.DB) *LocalRepository {
	return &LocalRepository{db: db}
}

type localLinha struct {
	ID           uuid.UUID `db:"id"`
	EmpresaID    uuid.UUID `db:"empresa_id"`
	Nome         string    `db:"nome"`
	Ativo        bool      `db:"ativo"`
	CriadoEm     time.Time `db:"criado_em"`
	AtualizadoEm time.Time `db:"atualizado_em"`
}

const colunasLocal = `id, empresa_id, nome, ativo, criado_em, atualizado_em`

func (l *LocalRepository) Criar(ctx context.Context, local *domainestoque.Local) (uuid.UUID, error) {
	query := `
		INSERT INTO estoque_local (id, empresa_id, nome, ativo, criado_em, atualizado_em)
		VALUES (:id, :empresa_id, :nome, :ativo, :criado_em, :atualizado_em)
	`

	if _, err := l.db.NamedExecContext(ctx, query, paraLinhaLocal(local)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return local.ID, nil
}

func (l *LocalRepository) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Local, error) {
	var linha localLinha
	if err := l.db.GetContext(ctx, &linha, `SELECT `+colunasLocal+` FROM estoque_local WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioLocal(&linha), nil
}

func (l *LocalRepository) ObterVarios(ctx context.Context, ids []uuid.UUID) ([]*domainestoque.Local, error) {
	var linhas []*localLinha
	if err := l.db.SelectContext(ctx, &linhas,
		`SELECT `+colunasLocal+` FROM estoque_local WHERE id = ANY($1::uuid[]) ORDER BY nome ASC, id ASC`,
		pq.Array(textosDeIDs(ids)),
	); err != nil {
		return nil, err
	}

	return paraDominioLocais(linhas), nil
}

func (l *LocalRepository) Listar(ctx context.Context, filtro portsout.LocalFiltro) ([]*domainestoque.Local, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasLocal + ` FROM estoque_local`
	condicoes := make([]string, 0, 3)
	argumentos := make([]any, 0, 5)

	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes = append(condicoes, condicao("empresa_id", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*localLinha
	if err := l.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	return paraDominioLocais(linhas), nil
}

func (l *LocalRepository) Atualizar(ctx context.Context, local *domainestoque.Local) (bool, error) {
	resultado, err := l.db.ExecContext(ctx, `
		UPDATE estoque_local
		SET nome = $1,
		    ativo = $2,
		    atualizado_em = $3
		WHERE id = $4
	`, local.Nome, local.Ativo, local.AtualizadoEm, local.ID)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return linhas > 0, nil
}

func (l *LocalRepository) DesativarSemSaldo(ctx context.Context, id uuid.UUID, agora time.Time) (bool, error) {
	tx, err := l.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	var encontrado uuid.UUID
	if err := tx.GetContext(ctx, &encontrado, `SELECT id FROM estoque_local WHERE id = $1 FOR UPDATE`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, tratarErroDeGravacao(err)
	}

	var possuiSaldo bool
	if err := tx.GetContext(ctx, &possuiSaldo,
		`SELECT EXISTS (SELECT 1 FROM produto_saldo_local WHERE local_id = $1 AND quantidade > 0)`, id,
	); err != nil {
		return false, tratarErroDeGravacao(err)
	}
	if possuiSaldo {
		return false, nil
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE estoque_local SET ativo = FALSE, atualizado_em = $1 WHERE id = $2`, agora, id,
	); err != nil {
		return false, tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return true, nil
}

func (l *LocalRepository) Remover(ctx context.Context, id uuid.UUID) (bool, error) {
	resultado, err := l.db.ExecContext(ctx, `DELETE FROM estoque_local WHERE id = $1`, id)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErro(err)
	}

	return linhas > 0, nil
}

func paraLinhaLocal(local *domainestoque.Local) localLinha {
	return localLinha{
		ID:           local.ID,
		EmpresaID:    local.EmpresaID,
		Nome:         local.Nome,
		Ativo:        local.Ativo,
		CriadoEm:     local.CriadoEm,
		AtualizadoEm: local.AtualizadoEm,
	}
}

func paraDominioLocal(linha *localLinha) *domainestoque.Local {
	return &domainestoque.Local{
		ID:           linha.ID,
		EmpresaID:    linha.EmpresaID,
		Nome:         linha.Nome,
		Ativo:        linha.Ativo,
		CriadoEm:     linha.CriadoEm,
		AtualizadoEm: linha.AtualizadoEm,
	}
}

func paraDominioLocais(linhas []*localLinha) []*domainestoque.Local {
	itens := make([]*domainestoque.Local, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioLocal(linha))
	}

	return itens
}
