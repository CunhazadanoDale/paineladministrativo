package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.CategoriaFinanceiraRepository = (*CategoriaFinanceiraRepository)(nil)

type CategoriaFinanceiraRepository struct {
	db     *sqlx.DB
	guarda guardaDeVersao
}

func NewCategoriaFinanceiraRepository(db *sqlx.DB) *CategoriaFinanceiraRepository {
	return &CategoriaFinanceiraRepository{
		db: db,
		guarda: guardaDeVersao{
			tabela:        "categoria_financeira",
			naoEncontrado: domainfinanceiro.ErroCategoriaFinanceiraNaoEncontrada,
			alterado:      domainfinanceiro.ErroCategoriaFinanceiraAlterada,
			conflito:      conflitoDaCategoriaFinanceira,
		},
	}
}

type categoriaFinanceiraLinha struct {
	ID             uuid.UUID  `db:"id"`
	Nome           string     `db:"nome"`
	CategoriaPaiID *uuid.UUID `db:"categoria_pai_id"`
	Natureza       string     `db:"natureza"`
	LinhaDRE       string     `db:"linha_dre"`
	Codigo         *string    `db:"codigo"`
	Ordem          int        `db:"ordem"`
	Ativa          bool       `db:"ativa"`
	Versao         int        `db:"versao"`
	CriadoEm       time.Time  `db:"criado_em"`
	AtualizadoEm   time.Time  `db:"atualizado_em"`
}

const colunasCategoriaFinanceira = `id, nome, categoria_pai_id, natureza, linha_dre, codigo, ordem, ativa, versao,
	criado_em, atualizado_em`

const ordemCategoriaFinanceira = ` ORDER BY ordem ASC, lower(nome) ASC, id ASC`

func (r *CategoriaFinanceiraRepository) Criar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if categoria.EhSubcategoria() {
		var pai categoriaFinanceiraLinha
		err := tx.GetContext(ctx, &pai, `
			SELECT `+colunasCategoriaFinanceira+`
			FROM categoria_financeira
			WHERE id = $1
			  AND categoria_pai_id IS NULL
			  AND ativa
			FOR SHARE
		`, *categoria.CategoriaPaiID)
		if errors.Is(err, sql.ErrNoRows) {
			return domainfinanceiro.ErroPaiDeveSerRaizAtiva()
		}
		if err != nil {
			return tratarErroDeGravacao(err)
		}

		categoria.Natureza = domainfinanceiro.NaturezaDe(pai.Natureza)
		categoria.LinhaDRE = domainfinanceiro.LinhaDREDe(pai.LinhaDRE)
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO categoria_financeira (`+colunasCategoriaFinanceira+`)
		VALUES (
			:id, :nome, :categoria_pai_id, :natureza, :linha_dre, :codigo, :ordem, :ativa, :versao,
			:criado_em, :atualizado_em
		)
	`, paraLinhaCategoriaFinanceira(categoria)); err != nil {
		return r.guarda.traduzir(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *CategoriaFinanceiraRepository) Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	var linha categoriaFinanceiraLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasCategoriaFinanceira+` FROM categoria_financeira WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioCategoriaFinanceira(&linha), nil
}

func (r *CategoriaFinanceiraRepository) Listar(ctx context.Context, filtro portsout.CategoriaFinanceiraFiltro) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasCategoriaFinanceira + ` FROM categoria_financeira`
	condicoes := make([]string, 0, 6)
	argumentos := make([]any, 0, 8)

	if filtro.Pai != nil {
		argumentos = append(argumentos, *filtro.Pai)
		condicoes = append(condicoes, condicao("categoria_pai_id", len(argumentos)))
	}
	if filtro.SomenteRaiz {
		condicoes = append(condicoes, "categoria_pai_id IS NULL")
	}
	if filtro.Natureza != nil {
		argumentos = append(argumentos, filtro.Natureza.String())
		condicoes = append(condicoes, condicao("natureza", len(argumentos)))
	}
	if filtro.LinhaDRE != nil {
		argumentos = append(argumentos, filtro.LinhaDRE.String())
		condicoes = append(condicoes, condicao("linha_dre", len(argumentos)))
	}
	if filtro.Ativa != nil {
		argumentos = append(argumentos, *filtro.Ativa)
		condicoes = append(condicoes, condicao("ativa", len(argumentos)))
	}
	if busca := strings.TrimSpace(filtro.Busca); busca != "" {
		argumentos = append(argumentos, "%"+escaparBusca(busca)+"%", escaparBusca(busca)+"%")
		condicoes = append(condicoes, "(nome ILIKE $"+strconv.Itoa(len(argumentos)-1)+
			" OR codigo LIKE $"+strconv.Itoa(len(argumentos))+")")
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ordemCategoriaFinanceira + limiteOffset(argumentos)

	return r.selecionar(ctx, query, argumentos...)
}

func (r *CategoriaFinanceiraRepository) ListarPlano(ctx context.Context, somenteAtivas bool) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	query := `SELECT ` + colunasCategoriaFinanceira + ` FROM categoria_financeira`
	if somenteAtivas {
		query += ` WHERE ativa`
	}

	return r.selecionar(ctx, query+ordemCategoriaFinanceira)
}

func (r *CategoriaFinanceiraRepository) PossuiSubcategoriasAtivas(ctx context.Context, id uuid.UUID) (bool, error) {
	return existeRegistro(ctx, r.db,
		`SELECT EXISTS (SELECT 1 FROM categoria_financeira WHERE categoria_pai_id = $1 AND ativa)`, id)
}

func (r *CategoriaFinanceiraRepository) Atualizar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	versao, err := r.guarda.atualizar(ctx, r.db, categoria.ID, `
		UPDATE categoria_financeira
		SET nome = $1,
		    codigo = $2,
		    ordem = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE id = $5
		  AND versao = $6
		RETURNING versao
	`, categoria.Nome, categoria.Codigo, categoria.Ordem, categoria.AtualizadoEm, categoria.ID, categoria.Versao)
	if err != nil {
		return err
	}

	categoria.Versao = versao

	return nil
}

func (r *CategoriaFinanceiraRepository) Ativar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if categoria.EhSubcategoria() {
		var paiAtivo bool
		err := tx.GetContext(ctx, &paiAtivo,
			`SELECT ativa FROM categoria_financeira WHERE id = $1 FOR SHARE`, *categoria.CategoriaPaiID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return tratarErroDeGravacao(err)
		}
		if !paiAtivo {
			return domainfinanceiro.ErroPaiInativo()
		}
	}

	versao, err := r.trocarAtiva(ctx, tx, categoria, true)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	categoria.Versao = versao

	return nil
}

func (r *CategoriaFinanceiraRepository) Desativar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.guarda.conferirVersaoTravada(ctx, tx, categoria.ID, categoria.Versao); err != nil {
		return err
	}

	if !categoria.EhSubcategoria() {
		var possuiAtivas bool
		if err := tx.GetContext(ctx, &possuiAtivas,
			`SELECT EXISTS (SELECT 1 FROM categoria_financeira WHERE categoria_pai_id = $1 AND ativa)`, categoria.ID,
		); err != nil {
			return tratarErroDeGravacao(err)
		}
		if possuiAtivas {
			return domainfinanceiro.ErroSubcategoriasAtivas()
		}
	}

	versao, err := r.trocarAtiva(ctx, tx, categoria, false)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	categoria.Versao = versao

	return nil
}

func (r *CategoriaFinanceiraRepository) Remover(ctx context.Context, id uuid.UUID) error {
	return r.guarda.remover(ctx, r.db, id)
}

func (r *CategoriaFinanceiraRepository) trocarAtiva(ctx context.Context, tx *sqlx.Tx, categoria *domainfinanceiro.CategoriaFinanceira, ativa bool) (int, error) {
	return r.guarda.atualizar(ctx, tx, categoria.ID, `
		UPDATE categoria_financeira
		SET ativa = $1,
		    atualizado_em = $2,
		    versao = versao + 1
		WHERE id = $3
		  AND versao = $4
		RETURNING versao
	`, ativa, categoria.AtualizadoEm, categoria.ID, categoria.Versao)
}

func (r *CategoriaFinanceiraRepository) selecionar(ctx context.Context, query string, argumentos ...any) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	var linhas []*categoriaFinanceiraLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	categorias := make([]*domainfinanceiro.CategoriaFinanceira, 0, len(linhas))
	for _, linha := range linhas {
		categorias = append(categorias, paraDominioCategoriaFinanceira(linha))
	}

	return categorias, nil
}

func conflitoDaCategoriaFinanceira(restricao string) error {
	switch restricao {
	case "uq_categoria_financeira_raiz_nome", "uq_categoria_financeira_sub_nome":
		return domainfinanceiro.ErroNomeDeCategoriaRepetido()
	case "uq_categoria_financeira_codigo":
		return domainfinanceiro.ErroCodigoDeCategoriaRepetido()
	default:
		return nil
	}
}

func paraLinhaCategoriaFinanceira(categoria *domainfinanceiro.CategoriaFinanceira) categoriaFinanceiraLinha {
	return categoriaFinanceiraLinha{
		ID:             categoria.ID,
		Nome:           categoria.Nome,
		CategoriaPaiID: categoria.CategoriaPaiID,
		Natureza:       categoria.Natureza.String(),
		LinhaDRE:       categoria.LinhaDRE.String(),
		Codigo:         categoria.Codigo,
		Ordem:          categoria.Ordem,
		Ativa:          categoria.Ativa,
		Versao:         categoria.Versao,
		CriadoEm:       categoria.CriadoEm,
		AtualizadoEm:   categoria.AtualizadoEm,
	}
}

func paraDominioCategoriaFinanceira(linha *categoriaFinanceiraLinha) *domainfinanceiro.CategoriaFinanceira {
	return &domainfinanceiro.CategoriaFinanceira{
		ID:             linha.ID,
		Nome:           linha.Nome,
		CategoriaPaiID: linha.CategoriaPaiID,
		Natureza:       domainfinanceiro.NaturezaDe(linha.Natureza),
		LinhaDRE:       domainfinanceiro.LinhaDREDe(linha.LinhaDRE),
		Codigo:         linha.Codigo,
		Ordem:          linha.Ordem,
		Ativa:          linha.Ativa,
		Versao:         linha.Versao,
		CriadoEm:       linha.CriadoEm,
		AtualizadoEm:   linha.AtualizadoEm,
	}
}
