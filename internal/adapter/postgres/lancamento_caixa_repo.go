package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.LancamentoCaixaRepository = (*LancamentoCaixaRepository)(nil)

type LancamentoCaixaRepository struct {
	db *sqlx.DB
}

func NewLancamentoCaixaRepository(db *sqlx.DB) *LancamentoCaixaRepository {
	return &LancamentoCaixaRepository{db: db}
}

type lancamentoCaixaLinha struct {
	ID                    uuid.UUID `db:"id"`
	EmpresaID             uuid.UUID `db:"empresa_id"`
	ContaBancariaID       uuid.UUID `db:"conta_bancaria_id"`
	Data                  time.Time `db:"data"`
	ValorCentavos         int64     `db:"valor_centavos"`
	CategoriaFinanceiraID uuid.UUID `db:"categoria_financeira_id"`
	Competencia           time.Time `db:"competencia"`
	Descricao             string    `db:"descricao"`
	Estornado             bool      `db:"estornado"`
	UsuarioID             uuid.UUID `db:"usuario_id"`
	CriadoEm              time.Time `db:"criado_em"`
}

type transferenciaContaLinha struct {
	ID                    uuid.UUID `db:"id"`
	ContaOrigemID         uuid.UUID `db:"conta_origem_id"`
	ContaDestinoID        uuid.UUID `db:"conta_destino_id"`
	Data                  time.Time `db:"data"`
	ValorCentavos         int64     `db:"valor_centavos"`
	EntreEmpresas         bool      `db:"entre_empresas"`
	CategoriaFinanceiraID uuid.UUID `db:"categoria_financeira_id"`
	Descricao             string    `db:"descricao"`
	Estornado             bool      `db:"estornado"`
	UsuarioID             uuid.UUID `db:"usuario_id"`
	CriadoEm              time.Time `db:"criado_em"`
}

const (
	colunasLancamentoCaixa = `id, empresa_id, conta_bancaria_id, data, valor_centavos, categoria_financeira_id, competencia,
		descricao, estornado, usuario_id, criado_em`
	colunasTransferenciaConta = `id, conta_origem_id, conta_destino_id, data, valor_centavos, entre_empresas,
		categoria_financeira_id, descricao, estornado, usuario_id, criado_em`
)

func (r *LancamentoCaixaRepository) Classificar(ctx context.Context, classificacao *domaincaixa.Classificacao) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := travarLinhaDaConciliacao(ctx, tx, classificacao.Conciliacao); err != nil {
		return err
	}

	lancamento := classificacao.Lancamento
	natureza := domainfinanceiro.NaturezaDespesa
	if lancamento.ValorCentavos > 0 {
		natureza = domainfinanceiro.NaturezaReceita
	}
	if err := exigirCategoriaDoCaixa(ctx, tx, lancamento.CategoriaFinanceiraID, natureza); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO lancamento_caixa (`+colunasLancamentoCaixa+`)
		VALUES (
			:id, :empresa_id, :conta_bancaria_id, :data, :valor_centavos, :categoria_financeira_id, :competencia,
			:descricao, :estornado, :usuario_id, :criado_em
		)
	`, paraLinhaLancamentoCaixa(lancamento)); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := inserirConciliacao(ctx, tx, classificacao.Conciliacao); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return traduzirConflitoDaConciliacao(err)
	}

	return nil
}

func (r *LancamentoCaixaRepository) Transferir(ctx context.Context, registrada *domaincaixa.TransferenciaRegistrada) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	conciliacao := registrada.Conciliacao
	if err := travarLinhaDaConciliacao(ctx, tx, conciliacao); err != nil {
		return err
	}

	transferencia := registrada.Transferencia
	contraparte := transferencia.ContaOrigemID
	if contraparte == conciliacao.ContaBancariaID {
		contraparte = transferencia.ContaDestinoID
	}
	if err := exigirContraparteAtiva(ctx, tx, contraparte); err != nil {
		return err
	}
	if err := exigirCategoriaDoCaixa(ctx, tx, transferencia.CategoriaFinanceiraID, domainfinanceiro.NaturezaTransferencia); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO transferencia_conta (`+colunasTransferenciaConta+`)
		VALUES (
			:id, :conta_origem_id, :conta_destino_id, :data, :valor_centavos, :entre_empresas,
			:categoria_financeira_id, :descricao, :estornado, :usuario_id, :criado_em
		)
	`, paraLinhaTransferenciaConta(transferencia)); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := inserirConciliacao(ctx, tx, conciliacao); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return traduzirConflitoDaConciliacao(err)
	}

	return nil
}

func (r *LancamentoCaixaRepository) Listar(ctx context.Context, filtro portsout.LancamentoCaixaFiltro) ([]*domaincaixa.LancamentoCaixa, error) {
	argumentos := make([]any, 0, 6)
	condicoes := make([]string, 0, 4)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ContaBancariaID != nil {
		argumentos = append(argumentos, *filtro.ContaBancariaID)
		condicoes = append(condicoes, condicao("conta_bancaria_id", len(argumentos)))
	}
	argumentos, condicoes = condicoesDePeriodo("data", filtro, argumentos, condicoes)

	query := `SELECT ` + colunasLancamentoCaixa + ` FROM lancamento_caixa`
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Limite, filtro.Deslocamento)
	query += ` ORDER BY data ASC, criado_em ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*lancamentoCaixaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	lancamentos := make([]*domaincaixa.LancamentoCaixa, 0, len(linhas))
	for _, linha := range linhas {
		lancamentos = append(lancamentos, paraDominioLancamentoCaixa(linha))
	}

	return lancamentos, nil
}

func (r *LancamentoCaixaRepository) ListarTransferencias(ctx context.Context, filtro portsout.LancamentoCaixaFiltro) ([]*domaincaixa.Transferencia, error) {
	argumentos := make([]any, 0, 6)
	condicoes := make([]string, 0, 4)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		posicao := strconv.Itoa(len(argumentos))
		condicoes = append(condicoes, "(origem.empresa_id = ANY($"+posicao+"::uuid[]) OR destino.empresa_id = ANY($"+posicao+"::uuid[]))")
	}
	if filtro.ContaBancariaID != nil {
		argumentos = append(argumentos, *filtro.ContaBancariaID)
		posicao := strconv.Itoa(len(argumentos))
		condicoes = append(condicoes, "(t.conta_origem_id = $"+posicao+" OR t.conta_destino_id = $"+posicao+")")
	}
	argumentos, condicoes = condicoesDePeriodo("t.data", filtro, argumentos, condicoes)

	query := `
		SELECT t.id, t.conta_origem_id, t.conta_destino_id, t.data, t.valor_centavos, t.entre_empresas,
		       t.categoria_financeira_id, t.descricao, t.estornado, t.usuario_id, t.criado_em
		FROM transferencia_conta t
		JOIN conta_bancaria origem ON origem.id = t.conta_origem_id
		JOIN conta_bancaria destino ON destino.id = t.conta_destino_id`
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Limite, filtro.Deslocamento)
	query += ` ORDER BY t.data ASC, t.criado_em ASC, t.id ASC` + limiteOffset(argumentos)

	var linhas []*transferenciaContaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	transferencias := make([]*domaincaixa.Transferencia, 0, len(linhas))
	for _, linha := range linhas {
		transferencias = append(transferencias, paraDominioTransferenciaConta(linha))
	}

	return transferencias, nil
}

func condicoesDePeriodo(coluna string, filtro portsout.LancamentoCaixaFiltro, argumentos []any, condicoes []string) ([]any, []string) {
	if filtro.De != nil {
		argumentos = append(argumentos, filtro.De.Format(formatoDataDoBanco))
		condicoes = append(condicoes, coluna+" >= $"+strconv.Itoa(len(argumentos))+"::date")
	}
	if filtro.Ate != nil {
		argumentos = append(argumentos, filtro.Ate.Format(formatoDataDoBanco))
		condicoes = append(condicoes, coluna+" <= $"+strconv.Itoa(len(argumentos))+"::date")
	}

	return argumentos, condicoes
}

func exigirCategoriaDoCaixa(ctx context.Context, tx *sqlx.Tx, categoriaID uuid.UUID, natureza domainfinanceiro.Natureza) error {
	var categoria categoriaParaLancamento
	err := tx.GetContext(ctx, &categoria, `
		SELECT ativa, categoria_pai_id IS NOT NULL AS subcategoria, natureza
		FROM categoria_financeira
		WHERE id = $1
		FOR SHARE
	`, categoriaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErroValidacao("categoria financeira não encontrada")
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !categoria.Ativa || !categoria.Subcategoria {
		return domain.ErroValidacao("categoria financeira precisa ser uma subcategoria ativa")
	}
	if categoria.Natureza != natureza.String() {
		return domain.ErroValidacao("categoria financeira precisa ser de " + natureza.String())
	}

	return nil
}

func exigirContraparteAtiva(ctx context.Context, tx *sqlx.Tx, contaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM conta_bancaria WHERE id = $1 FOR SHARE`, contaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErroValidacao("conta da outra ponta não encontrada")
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domain.ErroValidacao("conta da outra ponta está inativa")
	}

	return nil
}

func paraLinhaLancamentoCaixa(lancamento *domaincaixa.LancamentoCaixa) *lancamentoCaixaLinha {
	return &lancamentoCaixaLinha{
		ID:                    lancamento.ID,
		EmpresaID:             lancamento.EmpresaID,
		ContaBancariaID:       lancamento.ContaBancariaID,
		Data:                  lancamento.Data,
		ValorCentavos:         lancamento.ValorCentavos,
		CategoriaFinanceiraID: lancamento.CategoriaFinanceiraID,
		Competencia:           lancamento.Competencia,
		Descricao:             lancamento.Descricao,
		Estornado:             lancamento.Estornado,
		UsuarioID:             lancamento.UsuarioID,
		CriadoEm:              lancamento.CriadoEm,
	}
}

func paraDominioLancamentoCaixa(linha *lancamentoCaixaLinha) *domaincaixa.LancamentoCaixa {
	return &domaincaixa.LancamentoCaixa{
		ID:                    linha.ID,
		EmpresaID:             linha.EmpresaID,
		ContaBancariaID:       linha.ContaBancariaID,
		Data:                  domaincaixa.DataDe(linha.Data),
		ValorCentavos:         linha.ValorCentavos,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		Competencia:           domaincaixa.DataDe(linha.Competencia),
		Descricao:             linha.Descricao,
		Estornado:             linha.Estornado,
		UsuarioID:             linha.UsuarioID,
		CriadoEm:              linha.CriadoEm,
	}
}

func paraLinhaTransferenciaConta(transferencia *domaincaixa.Transferencia) *transferenciaContaLinha {
	return &transferenciaContaLinha{
		ID:                    transferencia.ID,
		ContaOrigemID:         transferencia.ContaOrigemID,
		ContaDestinoID:        transferencia.ContaDestinoID,
		Data:                  transferencia.Data,
		ValorCentavos:         transferencia.ValorCentavos,
		EntreEmpresas:         transferencia.EntreEmpresas,
		CategoriaFinanceiraID: transferencia.CategoriaFinanceiraID,
		Descricao:             transferencia.Descricao,
		Estornado:             transferencia.Estornado,
		UsuarioID:             transferencia.UsuarioID,
		CriadoEm:              transferencia.CriadoEm,
	}
}

func paraDominioTransferenciaConta(linha *transferenciaContaLinha) *domaincaixa.Transferencia {
	return &domaincaixa.Transferencia{
		ID:                    linha.ID,
		ContaOrigemID:         linha.ContaOrigemID,
		ContaDestinoID:        linha.ContaDestinoID,
		Data:                  domaincaixa.DataDe(linha.Data),
		ValorCentavos:         linha.ValorCentavos,
		EntreEmpresas:         linha.EntreEmpresas,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		Descricao:             linha.Descricao,
		Estornado:             linha.Estornado,
		UsuarioID:             linha.UsuarioID,
		CriadoEm:              linha.CriadoEm,
	}
}
