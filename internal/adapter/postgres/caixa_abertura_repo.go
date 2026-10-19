package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.AberturaRepository = (*CaixaAberturaRepository)(nil)

type CaixaAberturaRepository struct {
	db *sqlx.DB
}

func NewCaixaAberturaRepository(db *sqlx.DB) *CaixaAberturaRepository {
	return &CaixaAberturaRepository{db: db}
}

type caixaAberturaLinha struct {
	ContaBancariaID uuid.UUID `db:"conta_bancaria_id"`
	DataCorte       time.Time `db:"data_corte"`
	SaldoCentavos   int64     `db:"saldo_centavos"`
	UsuarioID       uuid.UUID `db:"usuario_id"`
	Versao          int       `db:"versao"`
	CriadoEm        time.Time `db:"criado_em"`
	AtualizadoEm    time.Time `db:"atualizado_em"`
}

const colunasCaixaAbertura = `conta_bancaria_id, data_corte, saldo_centavos, usuario_id, versao, criado_em, atualizado_em`

func (r *CaixaAberturaRepository) Obter(ctx context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Abertura, error) {
	var linha caixaAberturaLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasCaixaAbertura+` FROM caixa_abertura WHERE conta_bancaria_id = $1`, contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioCaixaAbertura(&linha), nil
}

func (r *CaixaAberturaRepository) Criar(ctx context.Context, abertura *domaincaixa.Abertura) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirContaAtivaNoCaixa(ctx, tx, abertura.ContaBancariaID); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO caixa_abertura (`+colunasCaixaAbertura+`)
		VALUES (:conta_bancaria_id, :data_corte, :saldo_centavos, :usuario_id, :versao, :criado_em, :atualizado_em)
	`, paraLinhaCaixaAbertura(abertura)); err != nil {
		var erroPostgres *pq.Error
		if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
			return domaincaixa.ErroAberturaJaDefinida()
		}
		return tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *CaixaAberturaRepository) Atualizar(ctx context.Context, abertura *domaincaixa.Abertura) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirContaAtivaNoCaixa(ctx, tx, abertura.ContaBancariaID); err != nil {
		return err
	}

	var comFechamento bool
	if err := tx.GetContext(ctx, &comFechamento, `
		SELECT EXISTS (SELECT 1 FROM caixa_fechamento WHERE conta_bancaria_id = $1 AND vigente)
	`, abertura.ContaBancariaID); err != nil {
		return tratarErroDeGravacao(err)
	}
	if comFechamento {
		return domaincaixa.ErroAberturaComFechamento()
	}

	var versao int
	err = tx.GetContext(ctx, &versao, `
		UPDATE caixa_abertura
		SET data_corte = $1,
		    saldo_centavos = $2,
		    usuario_id = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE conta_bancaria_id = $5
		  AND versao = $6
		RETURNING versao
	`, abertura.DataCorte, abertura.SaldoCentavos, abertura.UsuarioID, abertura.AtualizadoEm, abertura.ContaBancariaID, abertura.Versao)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroAberturaAlterada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	abertura.Versao = versao

	return nil
}

func exigirContaAtivaNoCaixa(ctx context.Context, tx *sqlx.Tx, contaBancariaID uuid.UUID) error {
	var situacao struct {
		ContaAtiva   bool `db:"conta_ativa"`
		EmpresaAtiva bool `db:"empresa_ativa"`
	}
	err := tx.GetContext(ctx, &situacao, `
		SELECT c.ativa AS conta_ativa, e.ativa AS empresa_ativa
		FROM conta_bancaria c
		JOIN empresa e ON e.id = c.empresa_id
		WHERE c.id = $1
		FOR SHARE OF c, e
	`, contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroContaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !situacao.ContaAtiva {
		return domaincaixa.ErroContaInativa()
	}
	if !situacao.EmpresaAtiva {
		return domaincaixa.ErroEmpresaInativa()
	}

	return nil
}

func paraLinhaCaixaAbertura(abertura *domaincaixa.Abertura) *caixaAberturaLinha {
	return &caixaAberturaLinha{
		ContaBancariaID: abertura.ContaBancariaID,
		DataCorte:       abertura.DataCorte,
		SaldoCentavos:   abertura.SaldoCentavos,
		UsuarioID:       abertura.UsuarioID,
		Versao:          abertura.Versao,
		CriadoEm:        abertura.CriadoEm,
		AtualizadoEm:    abertura.AtualizadoEm,
	}
}

func paraDominioCaixaAbertura(linha *caixaAberturaLinha) *domaincaixa.Abertura {
	return &domaincaixa.Abertura{
		ContaBancariaID: linha.ContaBancariaID,
		DataCorte:       domaincaixa.DataDe(linha.DataCorte),
		SaldoCentavos:   linha.SaldoCentavos,
		UsuarioID:       linha.UsuarioID,
		Versao:          linha.Versao,
		CriadoEm:        linha.CriadoEm,
		AtualizadoEm:    linha.AtualizadoEm,
	}
}
