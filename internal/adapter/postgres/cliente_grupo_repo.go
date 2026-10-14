package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.GrupoRepository = (*ClienteGrupoRepository)(nil)

type ClienteGrupoRepository struct {
	db *sqlx.DB
}

func NewClienteGrupoRepository(db *sqlx.DB) *ClienteGrupoRepository {
	return &ClienteGrupoRepository{db: db}
}

type clienteGrupoLinha struct {
	ID           uuid.UUID `db:"id"`
	Nome         string    `db:"nome"`
	NomeBusca    string    `db:"nome_busca"`
	Ativo        bool      `db:"ativo"`
	CriadoEm     time.Time `db:"criado_em"`
	AtualizadoEm time.Time `db:"atualizado_em"`
}

const colunasClienteGrupo = `id, nome, nome_busca, ativo, criado_em, atualizado_em`

func (r *ClienteGrupoRepository) Criar(ctx context.Context, grupo *domaincliente.Grupo) (uuid.UUID, error) {
	if _, err := r.db.NamedExecContext(ctx, `
		INSERT INTO cliente_grupo (id, nome, nome_busca, ativo, criado_em, atualizado_em)
		VALUES (:id, :nome, :nome_busca, :ativo, :criado_em, :atualizado_em)
	`, clienteGrupoLinha(*grupo)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return grupo.ID, nil
}

func (r *ClienteGrupoRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Grupo, error) {
	return r.obterPor(ctx, `id = $1`, id)
}

func (r *ClienteGrupoRepository) ObterPorNomeBusca(ctx context.Context, nomeBusca string) (*domaincliente.Grupo, error) {
	return r.obterPor(ctx, `nome_busca = $1`, nomeBusca)
}

func (r *ClienteGrupoRepository) obterPor(ctx context.Context, condicao string, valor any) (*domaincliente.Grupo, error) {
	var linha clienteGrupoLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasClienteGrupo+` FROM cliente_grupo WHERE `+condicao, valor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	grupo := domaincliente.Grupo(linha)

	return &grupo, nil
}

func (r *ClienteGrupoRepository) Listar(ctx context.Context, filtro portsout.GrupoFiltro) ([]*domaincliente.Grupo, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasClienteGrupo + ` FROM cliente_grupo`
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 4)

	if termo := domaincliente.NomeDeBusca(filtro.Busca); termo != "" {
		argumentos = append(argumentos, "%"+escaparBusca(termo)+"%")
		condicoes = append(condicoes, "nome_busca LIKE $"+strconv.Itoa(len(argumentos)))
	}
	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome_busca ASC, id ASC` + limiteOffset(argumentos)

	var linhas []clienteGrupoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domaincliente.Grupo, 0, len(linhas))
	for _, linha := range linhas {
		grupo := domaincliente.Grupo(linha)
		itens = append(itens, &grupo)
	}

	return itens, nil
}

func (r *ClienteGrupoRepository) Atualizar(ctx context.Context, grupo *domaincliente.Grupo) (bool, error) {
	resultado, err := r.db.ExecContext(ctx, `
		UPDATE cliente_grupo
		SET nome = $1,
		    nome_busca = $2,
		    ativo = $3,
		    atualizado_em = $4
		WHERE id = $5
	`, grupo.Nome, grupo.NomeBusca, grupo.Ativo, grupo.AtualizadoEm, grupo.ID)

	return linhasAfetadas(resultado, err)
}

func linhasAfetadas(resultado sql.Result, err error) (bool, error) {
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return linhas > 0, nil
}
