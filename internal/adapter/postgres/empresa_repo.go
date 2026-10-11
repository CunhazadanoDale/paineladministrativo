package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/empresa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.EmpresaRepository = (*EmpresaRepository)(nil)

type EmpresaRepository struct {
	db *sqlx.DB
}

func NewEmpresaRepository(db *sqlx.DB) *EmpresaRepository {
	return &EmpresaRepository{db: db}
}

type empresaLinha struct {
	ID               uuid.UUID `db:"id"`
	RazaoSocial      string    `db:"razao_social"`
	NomeFantasia     string    `db:"nome_fantasia"`
	CNPJ             string    `db:"cnpj"`
	UF               string    `db:"uf"`
	RegimeTributario string    `db:"regime_tributario"`
	Ativa            bool      `db:"ativa"`
	CriadoEm         time.Time `db:"criado_em"`
	AtualizadoEm     time.Time `db:"atualizado_em"`
}

type acessoLinha struct {
	Todas     bool       `db:"todas"`
	EmpresaID *uuid.UUID `db:"empresa_id"`
}

const colunasEmpresa = `
	id, razao_social, nome_fantasia, cnpj, uf, regime_tributario, ativa, criado_em, atualizado_em
`

func (e *EmpresaRepository) Criar(ctx context.Context, empresa *domainempresa.Empresa) (uuid.UUID, error) {
	query := `
		INSERT INTO empresa (id, razao_social, nome_fantasia, cnpj, uf, regime_tributario, ativa, criado_em, atualizado_em)
		VALUES (:id, :razao_social, :nome_fantasia, :cnpj, :uf, :regime_tributario, :ativa, :criado_em, :atualizado_em)
	`

	if _, err := e.db.NamedExecContext(ctx, query, paraLinhaEmpresa(empresa)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return empresa.ID, nil
}

func (e *EmpresaRepository) Obter(ctx context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return e.obterPor(ctx, `id = $1`, id)
}

func (e *EmpresaRepository) ObterPorCNPJ(ctx context.Context, cnpj domainempresa.CNPJ) (*domainempresa.Empresa, error) {
	return e.obterPor(ctx, `cnpj = $1`, cnpj.Digitos())
}

func (e *EmpresaRepository) Listar(ctx context.Context, filtro portsout.EmpresaFiltro) ([]*domainempresa.Empresa, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasEmpresa + ` FROM empresa`
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 4)

	if filtro.Ativa != nil {
		argumentos = append(argumentos, *filtro.Ativa)
		condicoes = append(condicoes, condicao("ativa", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome_fantasia ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*empresaLinha
	if err := e.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainempresa.Empresa, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioEmpresa(linha))
	}

	return itens, nil
}

func (e *EmpresaRepository) Atualizar(ctx context.Context, empresa *domainempresa.Empresa) (bool, error) {
	query := `
		UPDATE empresa
		SET razao_social = $1,
		    nome_fantasia = $2,
		    uf = $3,
		    regime_tributario = $4,
		    ativa = $5,
		    atualizado_em = $6
		WHERE id = $7
	`

	resultado, err := e.db.ExecContext(ctx, query,
		empresa.RazaoSocial,
		empresa.NomeFantasia,
		empresa.UF.String(),
		empresa.RegimeTributario.String(),
		empresa.Ativa,
		empresa.AtualizadoEm,
		empresa.ID,
	)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return linhas > 0, nil
}

func (e *EmpresaRepository) Remover(ctx context.Context, id uuid.UUID) (bool, error) {
	resultado, err := e.db.ExecContext(ctx, `DELETE FROM empresa WHERE id = $1`, id)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErro(err)
	}

	return linhas > 0, nil
}

func (e *EmpresaRepository) AcessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	query := `
		SELECT u.acesso_todas_empresas OR c.administrador AS todas, ue.empresa_id
		FROM usuario u
		JOIN cargo c ON c.id = u.cargo_id
		LEFT JOIN usuario_empresa ue ON ue.usuario_id = u.id
		WHERE u.id = $1
	`

	var linhas []acessoLinha
	if err := e.db.SelectContext(ctx, &linhas, query, usuarioID); err != nil {
		return domainempresa.Acesso{}, err
	}
	if len(linhas) == 0 {
		return domainempresa.Acesso{}, domain.ErroNaoEncontrado("usuário não encontrado")
	}
	if linhas[0].Todas {
		return domainempresa.AcessoTotal(), nil
	}

	return domainempresa.AcessoRestrito(empresasDasLinhas(linhas)), nil
}

func (e *EmpresaRepository) VinculosDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainempresa.Vinculos, error) {
	query := `
		SELECT u.acesso_todas_empresas AS todas, ue.empresa_id
		FROM usuario u
		LEFT JOIN usuario_empresa ue ON ue.usuario_id = u.id
		WHERE u.id = $1
		ORDER BY ue.empresa_id ASC
	`

	var linhas []acessoLinha
	if err := e.db.SelectContext(ctx, &linhas, query, usuarioID); err != nil {
		return nil, err
	}
	if len(linhas) == 0 {
		return nil, nil
	}

	return &domainempresa.Vinculos{
		AcessoTodas: linhas[0].Todas,
		EmpresaIDs:  empresasDasLinhas(linhas),
	}, nil
}

func (e *EmpresaRepository) DefinirVinculos(ctx context.Context, usuarioID uuid.UUID, vinculos domainempresa.Vinculos) error {
	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	resultado, err := tx.ExecContext(ctx, `UPDATE usuario SET acesso_todas_empresas = $1 WHERE id = $2`, vinculos.AcessoTodas, usuarioID)
	if err != nil {
		_ = tx.Rollback()
		return tratarErroDeGravacao(err)
	}

	afetados, err := resultado.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return tratarErroDeGravacao(err)
	}
	if afetados == 0 {
		_ = tx.Rollback()
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM usuario_empresa WHERE usuario_id = $1`, usuarioID); err != nil {
		_ = tx.Rollback()
		return tratarErroDeGravacao(err)
	}

	for _, empresaID := range vinculos.EmpresaIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO usuario_empresa (usuario_id, empresa_id, criado_em)
			VALUES ($1, $2, NOW())
			ON CONFLICT DO NOTHING
		`, usuarioID, empresaID); err != nil {
			_ = tx.Rollback()
			return tratarErroDeGravacao(err)
		}
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (e *EmpresaRepository) obterPor(ctx context.Context, criterio string, valor any) (*domainempresa.Empresa, error) {
	query := `SELECT ` + colunasEmpresa + ` FROM empresa WHERE ` + criterio

	var linha empresaLinha
	if err := e.db.GetContext(ctx, &linha, query, valor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioEmpresa(&linha), nil
}

func empresasDasLinhas(linhas []acessoLinha) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(linhas))
	for _, linha := range linhas {
		if linha.EmpresaID != nil {
			ids = append(ids, *linha.EmpresaID)
		}
	}

	return ids
}

func textosDeIDs(ids []uuid.UUID) []string {
	textos := make([]string, 0, len(ids))
	for _, id := range ids {
		textos = append(textos, id.String())
	}

	return textos
}

func paraLinhaEmpresa(empresa *domainempresa.Empresa) *empresaLinha {
	return &empresaLinha{
		ID:               empresa.ID,
		RazaoSocial:      empresa.RazaoSocial,
		NomeFantasia:     empresa.NomeFantasia,
		CNPJ:             empresa.CNPJ.Digitos(),
		UF:               empresa.UF.String(),
		RegimeTributario: empresa.RegimeTributario.String(),
		Ativa:            empresa.Ativa,
		CriadoEm:         empresa.CriadoEm,
		AtualizadoEm:     empresa.AtualizadoEm,
	}
}

func paraDominioEmpresa(linha *empresaLinha) *domainempresa.Empresa {
	return &domainempresa.Empresa{
		ID:               linha.ID,
		RazaoSocial:      linha.RazaoSocial,
		NomeFantasia:     linha.NomeFantasia,
		CNPJ:             domainempresa.CNPJDe(linha.CNPJ),
		UF:               domainempresa.UFDe(linha.UF),
		RegimeTributario: domainempresa.RegimeTributarioDe(linha.RegimeTributario),
		Ativa:            linha.Ativa,
		CriadoEm:         linha.CriadoEm,
		AtualizadoEm:     linha.AtualizadoEm,
	}
}
