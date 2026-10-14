package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ContaBancariaRepository = (*ContaBancariaRepository)(nil)

type ContaBancariaRepository struct {
	db     *sqlx.DB
	guarda guardaDeVersao
}

func NewContaBancariaRepository(db *sqlx.DB) *ContaBancariaRepository {
	return &ContaBancariaRepository{
		db: db,
		guarda: guardaDeVersao{
			tabela:        "conta_bancaria",
			naoEncontrado: domainfinanceiro.ErroContaBancariaNaoEncontrada,
			alterado:      domainfinanceiro.ErroContaBancariaAlterada,
			conflito:      conflitoDaContaBancaria,
		},
	}
}

type contaBancariaLinha struct {
	ID            uuid.UUID `db:"id"`
	EmpresaID     uuid.UUID `db:"empresa_id"`
	Banco         string    `db:"banco"`
	Tipo          string    `db:"tipo"`
	Agencia       string    `db:"agencia"`
	AgenciaDigito *string   `db:"agencia_digito"`
	Conta         string    `db:"conta"`
	ContaDigito   *string   `db:"conta_digito"`
	Apelido       string    `db:"apelido"`
	Ativa         bool      `db:"ativa"`
	Versao        int       `db:"versao"`
	CriadoEm      time.Time `db:"criado_em"`
	AtualizadoEm  time.Time `db:"atualizado_em"`
}

const colunasContaBancaria = `id, empresa_id, banco, tipo, agencia, agencia_digito, conta, conta_digito, apelido,
	ativa, versao, criado_em, atualizado_em`

func (r *ContaBancariaRepository) Criar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaConta(ctx, tx, conta.EmpresaID); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO conta_bancaria (`+colunasContaBancaria+`)
		VALUES (
			:id, :empresa_id, :banco, :tipo, :agencia, :agencia_digito, :conta, :conta_digito, :apelido,
			:ativa, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaContaBancaria(conta)); err != nil {
		return r.guarda.traduzir(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *ContaBancariaRepository) Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	var linha contaBancariaLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasContaBancaria+` FROM conta_bancaria WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioContaBancaria(&linha), nil
}

func (r *ContaBancariaRepository) Listar(ctx context.Context, filtro portsout.ContaBancariaFiltro) ([]*domainfinanceiro.ContaBancaria, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasContaBancaria + ` FROM conta_bancaria`
	condicoes := make([]string, 0, 4)
	argumentos := make([]any, 0, 6)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.Banco != nil {
		argumentos = append(argumentos, filtro.Banco.String())
		condicoes = append(condicoes, condicao("banco", len(argumentos)))
	}
	if filtro.Tipo != nil {
		argumentos = append(argumentos, filtro.Tipo.String())
		condicoes = append(condicoes, condicao("tipo", len(argumentos)))
	}
	if filtro.Ativa != nil {
		argumentos = append(argumentos, *filtro.Ativa)
		condicoes = append(condicoes, condicao("ativa", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY lower(apelido) ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*contaBancariaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	contas := make([]*domainfinanceiro.ContaBancaria, 0, len(linhas))
	for _, linha := range linhas {
		contas = append(contas, paraDominioContaBancaria(linha))
	}

	return contas, nil
}

func (r *ContaBancariaRepository) Atualizar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error {
	versao, err := r.guarda.atualizar(ctx, r.db, conta.ID, `
		UPDATE conta_bancaria
		SET apelido = $1,
		    atualizado_em = $2,
		    versao = versao + 1
		WHERE id = $3
		  AND versao = $4
		RETURNING versao
	`, conta.Apelido, conta.AtualizadoEm, conta.ID, conta.Versao)
	if err != nil {
		return err
	}

	conta.Versao = versao

	return nil
}

func (r *ContaBancariaRepository) Ativar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaConta(ctx, tx, conta.EmpresaID); err != nil {
		return err
	}

	versao, err := r.trocarAtiva(ctx, tx, conta, true)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	conta.Versao = versao

	return nil
}

func (r *ContaBancariaRepository) Desativar(ctx context.Context, conta *domainfinanceiro.ContaBancaria) error {
	versao, err := r.trocarAtiva(ctx, r.db, conta, false)
	if err != nil {
		return err
	}

	conta.Versao = versao

	return nil
}

func (r *ContaBancariaRepository) Remover(ctx context.Context, id uuid.UUID) error {
	return r.guarda.remover(ctx, r.db, id)
}

func (r *ContaBancariaRepository) trocarAtiva(ctx context.Context, consulta sqlx.QueryerContext, conta *domainfinanceiro.ContaBancaria, ativa bool) (int, error) {
	return r.guarda.atualizar(ctx, consulta, conta.ID, `
		UPDATE conta_bancaria
		SET ativa = $1,
		    atualizado_em = $2,
		    versao = versao + 1
		WHERE id = $3
		  AND versao = $4
		RETURNING versao
	`, ativa, conta.AtualizadoEm, conta.ID, conta.Versao)
}

func exigirEmpresaAtivaParaConta(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainfinanceiro.ErroEmpresaDaContaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domainfinanceiro.ErroEmpresaInativa()
	}

	return nil
}

func conflitoDaContaBancaria(restricao string) error {
	switch restricao {
	case "uq_conta_bancaria_numero":
		return domainfinanceiro.ErroContaDuplicada()
	case "uq_conta_bancaria_empresa_apelido":
		return domainfinanceiro.ErroApelidoRepetido()
	default:
		return nil
	}
}

func paraLinhaContaBancaria(conta *domainfinanceiro.ContaBancaria) contaBancariaLinha {
	return contaBancariaLinha{
		ID:            conta.ID,
		EmpresaID:     conta.EmpresaID,
		Banco:         conta.Banco.String(),
		Tipo:          conta.Tipo.String(),
		Agencia:       conta.Agencia.Numero(),
		AgenciaDigito: conta.Agencia.Digito(),
		Conta:         conta.Conta.Numero(),
		ContaDigito:   conta.Conta.Digito(),
		Apelido:       conta.Apelido,
		Ativa:         conta.Ativa,
		Versao:        conta.Versao,
		CriadoEm:      conta.CriadoEm,
		AtualizadoEm:  conta.AtualizadoEm,
	}
}

func paraDominioContaBancaria(linha *contaBancariaLinha) *domainfinanceiro.ContaBancaria {
	return &domainfinanceiro.ContaBancaria{
		ID:           linha.ID,
		EmpresaID:    linha.EmpresaID,
		Banco:        domainfinanceiro.BancoDe(linha.Banco),
		Tipo:         domainfinanceiro.TipoContaDe(linha.Tipo),
		Agencia:      domainfinanceiro.NumeroBancarioDe(linha.Agencia, linha.AgenciaDigito),
		Conta:        domainfinanceiro.NumeroBancarioDe(linha.Conta, linha.ContaDigito),
		Apelido:      linha.Apelido,
		Ativa:        linha.Ativa,
		Versao:       linha.Versao,
		CriadoEm:     linha.CriadoEm,
		AtualizadoEm: linha.AtualizadoEm,
	}
}
