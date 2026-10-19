package postgres

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.ConciliacaoRepository = (*ConciliacaoRepository)(nil)

type ConciliacaoRepository struct {
	db *sqlx.DB
}

func NewConciliacaoRepository(db *sqlx.DB) *ConciliacaoRepository {
	return &ConciliacaoRepository{db: db}
}

type movimentoSistemaLinha struct {
	Tipo            string     `db:"tipo"`
	ID              uuid.UUID  `db:"id"`
	ContaBancariaID *uuid.UUID `db:"conta_bancaria_id"`
	EmpresaID       uuid.UUID  `db:"empresa_id"`
	Data            time.Time  `db:"data"`
	ValorCentavos   int64      `db:"valor_centavos"`
	Descricao       string     `db:"descricao"`
	OrigemID        uuid.UUID  `db:"origem_id"`
	Estornado       bool       `db:"estornado"`
	ConciliacaoID   *uuid.UUID `db:"conciliacao_id"`
}

type conciliacaoLinha struct {
	ID                  uuid.UUID  `db:"id"`
	ExtratoLancamentoID uuid.UUID  `db:"extrato_lancamento_id"`
	ContaBancariaID     uuid.UUID  `db:"conta_bancaria_id"`
	Modo                string     `db:"modo"`
	Vigente             bool       `db:"vigente"`
	DesfeitaEm          *time.Time `db:"desfeita_em"`
	DesfeitaPor         *uuid.UUID `db:"desfeita_por"`
	MotivoDesfazer      *string    `db:"motivo_desfazer"`
	UsuarioID           uuid.UUID  `db:"usuario_id"`
	CriadoEm            time.Time  `db:"criado_em"`
}

type conciliacaoItemLinha struct {
	Tipo          string    `db:"alvo_tipo"`
	AlvoID        uuid.UUID `db:"alvo_id"`
	ValorCentavos int64     `db:"valor_centavos"`
	Estornado     bool      `db:"estornado"`
}

const (
	diaDoPagamento = `((p.pago_em AT TIME ZONE 'UTC') - interval '3 hours')::date`

	descricaoDoPagamento = `COALESCE(f.nome || ' - ', '') || s.observacao`

	consultaMovimentosDaConta = `
		WITH abertura AS (
			SELECT data_corte FROM caixa_abertura WHERE conta_bancaria_id = $1
		), movimento AS (
			SELECT 'pagamento' AS tipo, p.id, p.conta_bancaria_id, s.empresa_id, ` + diaDoPagamento + ` AS data,
			       -p.valor AS valor_centavos, ` + descricaoDoPagamento + ` AS descricao, s.id AS origem_id, p.estornado
			FROM pagamento p
			JOIN solicitacao s ON s.id = p.solicitacao_id
			LEFT JOIN fornecedor f ON f.id = s.fornecedor_id
			WHERE p.conta_bancaria_id = $1
			UNION ALL
			SELECT 'recebimento', r.id, r.conta_bancaria_id, t.empresa_id, r.recebido_em,
			       r.valor_centavos + r.juros_centavos, c.nome || ' - título ' || t.numero || '/' || t.parcela, t.id, r.estornado
			FROM titulo_recebimento r
			JOIN titulo_receber t ON t.id = r.titulo_id
			JOIN cliente c ON c.id = t.cliente_id
			WHERE r.tipo = 'conta' AND r.conta_bancaria_id = $1
			UNION ALL
			SELECT 'antecipacao', a.id, a.conta_bancaria_id, a.empresa_id, a.operado_em,
			       a.valor_liquido_centavos, 'Borderô ' || COALESCE(a.numero || ' ', '') || '- ' || f.nome, a.id,
			       a.status = 'estornada'
			FROM antecipacao a
			JOIN fornecedor f ON f.id = a.fornecedor_id
			WHERE a.conta_bancaria_id = $1
		)
		SELECT m.tipo, m.id, m.conta_bancaria_id, m.empresa_id, m.data, m.valor_centavos, m.descricao, m.origem_id,
		       m.estornado, ci.conciliacao_id
		FROM movimento m
		JOIN abertura ON m.data > abertura.data_corte
		LEFT JOIN conciliacao_item ci ON ci.alvo_tipo = m.tipo AND ci.alvo_id = m.id AND ci.vigente`

	alvoEstornado = `CASE ci.alvo_tipo
			WHEN 'pagamento' THEN EXISTS (SELECT 1 FROM pagamento p WHERE p.id = ci.alvo_id AND p.estornado)
			WHEN 'recebimento' THEN EXISTS (SELECT 1 FROM titulo_recebimento r WHERE r.id = ci.alvo_id AND r.estornado)
			WHEN 'antecipacao' THEN EXISTS (SELECT 1 FROM antecipacao a WHERE a.id = ci.alvo_id AND a.status = 'estornada')
			ELSE false
		END`

	colunasConciliacao = `id, extrato_lancamento_id, conta_bancaria_id, modo, vigente, desfeita_em, desfeita_por,
		motivo_desfazer, usuario_id, criado_em`
)

func (r *ConciliacaoRepository) ObterLancamento(ctx context.Context, id uuid.UUID) (*domaincaixa.LancamentoExtrato, error) {
	var linha extratoLancamentoLinha
	err := r.db.GetContext(ctx, &linha, consultaLancamentosComSituacao+` WHERE l.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioExtratoLancamento(&linha), nil
}

func (r *ConciliacaoRepository) LancamentosPendentes(
	ctx context.Context,
	contaBancariaID uuid.UUID,
	de, ate time.Time,
	limite int,
) ([]*domaincaixa.LancamentoExtrato, error) {
	var linhas []*extratoLancamentoLinha
	if err := r.db.SelectContext(ctx, &linhas, consultaLancamentosComSituacao+`
		JOIN caixa_abertura a ON a.conta_bancaria_id = l.conta_bancaria_id
		WHERE l.conta_bancaria_id = $1
		  AND l.data > a.data_corte
		  AND l.data >= $2::date
		  AND l.data <= $3::date
		  AND co.id IS NULL
		ORDER BY l.data ASC, l.criado_em ASC, l.sequencia ASC, l.id ASC
		LIMIT $4
	`, contaBancariaID, de.Format(formatoDataDoBanco), ate.Format(formatoDataDoBanco), limite); err != nil {
		return nil, err
	}

	lancamentos := make([]*domaincaixa.LancamentoExtrato, 0, len(linhas))
	for _, linha := range linhas {
		lancamentos = append(lancamentos, paraDominioExtratoLancamento(linha))
	}

	return lancamentos, nil
}

func (r *ConciliacaoRepository) Movimentos(ctx context.Context, filtro portsout.MovimentoFiltro) ([]*domaincaixa.MovimentoSistema, error) {
	argumentos := []any{filtro.ContaBancariaID}
	condicoes := make([]string, 0, 4)

	if filtro.De != nil {
		argumentos = append(argumentos, filtro.De.Format(formatoDataDoBanco))
		condicoes = append(condicoes, "m.data >= $"+strconv.Itoa(len(argumentos))+"::date")
	}
	if filtro.Ate != nil {
		argumentos = append(argumentos, filtro.Ate.Format(formatoDataDoBanco))
		condicoes = append(condicoes, "m.data <= $"+strconv.Itoa(len(argumentos))+"::date")
	}
	if filtro.ValorCentavos != nil {
		argumentos = append(argumentos, *filtro.ValorCentavos)
		condicoes = append(condicoes, condicao("m.valor_centavos", len(argumentos)))
	}
	if filtro.SoPendentes {
		condicoes = append(condicoes, "NOT m.estornado AND ci.conciliacao_id IS NULL")
	}

	query := consultaMovimentosDaConta
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}
	query += ` ORDER BY m.data ASC, m.tipo ASC, m.id ASC`
	if filtro.Limite > 0 {
		argumentos = append(argumentos, filtro.Limite, filtro.Deslocamento)
		query += limiteOffset(argumentos)
	}

	return r.selecionarMovimentos(ctx, query, argumentos...)
}

func (r *ConciliacaoRepository) MovimentosPorReferencia(
	ctx context.Context,
	contaBancariaID uuid.UUID,
	referencias []domaincaixa.ReferenciaMovimento,
) ([]*domaincaixa.MovimentoSistema, error) {
	if len(referencias) == 0 {
		return nil, nil
	}

	tipos := make([]string, 0, len(referencias))
	ids := make([]string, 0, len(referencias))
	for _, referencia := range referencias {
		tipos = append(tipos, referencia.Tipo.String())
		ids = append(ids, referencia.ID.String())
	}

	return r.selecionarMovimentos(ctx, consultaMovimentosDaConta+`
		WHERE (m.tipo, m.id) IN (SELECT * FROM unnest($2::text[], $3::uuid[]))
		ORDER BY m.data ASC, m.tipo ASC, m.id ASC
	`, contaBancariaID, pq.Array(tipos), pq.Array(ids))
}

func (r *ConciliacaoRepository) PagamentosSemConta(ctx context.Context, filtro portsout.SemContaFiltro) ([]*domaincaixa.MovimentoSistema, error) {
	argumentos := make([]any, 0, 5)
	condicoes := []string{"p.conta_bancaria_id IS NULL", "NOT p.estornado"}

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "s.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.De != nil {
		argumentos = append(argumentos, filtro.De.Format(formatoDataDoBanco))
		condicoes = append(condicoes, diaDoPagamento+" >= $"+strconv.Itoa(len(argumentos))+"::date")
	}
	if filtro.Ate != nil {
		argumentos = append(argumentos, filtro.Ate.Format(formatoDataDoBanco))
		condicoes = append(condicoes, diaDoPagamento+" <= $"+strconv.Itoa(len(argumentos))+"::date")
	}

	argumentos = append(argumentos, filtro.Limite, filtro.Deslocamento)
	query := `
		SELECT 'pagamento' AS tipo, p.id, p.conta_bancaria_id, s.empresa_id, ` + diaDoPagamento + ` AS data,
		       -p.valor AS valor_centavos, ` + descricaoDoPagamento + ` AS descricao, s.id AS origem_id, p.estornado,
		       NULL::uuid AS conciliacao_id
		FROM pagamento p
		JOIN solicitacao s ON s.id = p.solicitacao_id
		LEFT JOIN fornecedor f ON f.id = s.fornecedor_id
		WHERE ` + unirCondicoes(condicoes) + `
		ORDER BY data ASC, p.id ASC` + limiteOffset(argumentos)

	return r.selecionarMovimentos(ctx, query, argumentos...)
}

func (r *ConciliacaoRepository) Criar(ctx context.Context, conciliacao *domaincaixa.Conciliacao) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	var contaDoLancamento uuid.UUID
	err = tx.GetContext(ctx, &contaDoLancamento, `SELECT conta_bancaria_id FROM extrato_lancamento WHERE id = $1 FOR SHARE`,
		conciliacao.ExtratoLancamentoID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroLancamentoNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if contaDoLancamento != conciliacao.ContaBancariaID {
		return domaincaixa.ErroMovimentoAlterado()
	}

	itens := slices.Clone(conciliacao.Itens)
	slices.SortFunc(itens, func(a, b domaincaixa.ItemConciliacao) int {
		return cmp.Or(cmp.Compare(a.Tipo, b.Tipo), cmp.Compare(a.AlvoID.String(), b.AlvoID.String()))
	})
	for _, item := range itens {
		if err := travarMovimentoAtivo(ctx, tx, item, conciliacao.ContaBancariaID); err != nil {
			return err
		}
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO conciliacao (`+colunasConciliacao+`)
		VALUES (
			:id, :extrato_lancamento_id, :conta_bancaria_id, :modo, :vigente, :desfeita_em, :desfeita_por,
			:motivo_desfazer, :usuario_id, :criado_em
		)
	`, paraLinhaConciliacao(conciliacao)); err != nil {
		return traduzirConflitoDaConciliacao(err)
	}

	for _, item := range conciliacao.Itens {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO conciliacao_item (conciliacao_id, alvo_tipo, alvo_id, valor_centavos, vigente)
			VALUES ($1, $2, $3, $4, true)
		`, conciliacao.ID, item.Tipo.String(), item.AlvoID, item.ValorCentavos); err != nil {
			return traduzirConflitoDaConciliacao(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return traduzirConflitoDaConciliacao(err)
	}

	return nil
}

func (r *ConciliacaoRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincaixa.Conciliacao, error) {
	var linha conciliacaoLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasConciliacao+` FROM conciliacao WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var itens []*conciliacaoItemLinha
	if err := r.db.SelectContext(ctx, &itens, `
		SELECT ci.alvo_tipo, ci.alvo_id, ci.valor_centavos, `+alvoEstornado+` AS estornado
		FROM conciliacao_item ci
		WHERE ci.conciliacao_id = $1
		ORDER BY ci.alvo_tipo ASC, ci.alvo_id ASC
	`, id); err != nil {
		return nil, err
	}

	return paraDominioConciliacao(&linha, itens), nil
}

func (r *ConciliacaoRepository) Desfazer(ctx context.Context, conciliacao *domaincaixa.Conciliacao) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	resultado, err := tx.ExecContext(ctx, `
		UPDATE conciliacao
		SET vigente = false, desfeita_em = $1, desfeita_por = $2, motivo_desfazer = $3
		WHERE id = $4 AND vigente
	`, conciliacao.DesfeitaEm, conciliacao.DesfeitaPor, conciliacao.MotivoDesfazer, conciliacao.ID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domaincaixa.ErroConciliacaoJaDesfeita()
	}

	if _, err := tx.ExecContext(ctx, `UPDATE conciliacao_item SET vigente = false WHERE conciliacao_id = $1`, conciliacao.ID); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *ConciliacaoRepository) selecionarMovimentos(ctx context.Context, query string, argumentos ...any) ([]*domaincaixa.MovimentoSistema, error) {
	var linhas []*movimentoSistemaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	movimentos := make([]*domaincaixa.MovimentoSistema, 0, len(linhas))
	for _, linha := range linhas {
		movimento := &domaincaixa.MovimentoSistema{
			Tipo:          domaincaixa.TipoMovimentoDe(linha.Tipo),
			ID:            linha.ID,
			EmpresaID:     linha.EmpresaID,
			Data:          domaincaixa.DataDe(linha.Data),
			ValorCentavos: linha.ValorCentavos,
			Descricao:     linha.Descricao,
			OrigemID:      linha.OrigemID,
			Estornado:     linha.Estornado,
			ConciliacaoID: linha.ConciliacaoID,
		}
		if linha.ContaBancariaID != nil {
			movimento.ContaBancariaID = *linha.ContaBancariaID
		}
		movimentos = append(movimentos, movimento)
	}

	return movimentos, nil
}

func travarMovimentoAtivo(ctx context.Context, tx *sqlx.Tx, item domaincaixa.ItemConciliacao, contaBancariaID uuid.UUID) error {
	var query string
	switch item.Tipo {
	case domaincaixa.MovimentoPagamento:
		query = `SELECT estornado FROM pagamento WHERE id = $1 AND conta_bancaria_id = $2 FOR SHARE`
	case domaincaixa.MovimentoRecebimento:
		query = `SELECT estornado FROM titulo_recebimento WHERE id = $1 AND conta_bancaria_id = $2 AND tipo = 'conta' FOR SHARE`
	case domaincaixa.MovimentoAntecipacao:
		query = `SELECT status = 'estornada' FROM antecipacao WHERE id = $1 AND conta_bancaria_id = $2 FOR SHARE`
	default:
		return domaincaixa.ErroMovimentoAlterado()
	}

	var estornado bool
	err := tx.GetContext(ctx, &estornado, query, item.AlvoID, contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroMovimentoAlterado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if estornado {
		return domaincaixa.ErroMovimentoAlterado()
	}

	return nil
}

func traduzirConflitoDaConciliacao(err error) error {
	var erroPostgres *pq.Error
	if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
		switch erroPostgres.Constraint {
		case "uq_conciliacao_lancamento_vigente":
			return domaincaixa.ErroLancamentoJaConciliado()
		case "uq_conciliacao_item_alvo_vigente":
			return domaincaixa.ErroMovimentoJaConciliado()
		}
	}

	return tratarErroDeGravacao(err)
}

func paraLinhaConciliacao(conciliacao *domaincaixa.Conciliacao) *conciliacaoLinha {
	return &conciliacaoLinha{
		ID:                  conciliacao.ID,
		ExtratoLancamentoID: conciliacao.ExtratoLancamentoID,
		ContaBancariaID:     conciliacao.ContaBancariaID,
		Modo:                conciliacao.Modo.String(),
		Vigente:             conciliacao.Vigente,
		DesfeitaEm:          conciliacao.DesfeitaEm,
		DesfeitaPor:         conciliacao.DesfeitaPor,
		MotivoDesfazer:      conciliacao.MotivoDesfazer,
		UsuarioID:           conciliacao.UsuarioID,
		CriadoEm:            conciliacao.CriadoEm,
	}
}

func paraDominioConciliacao(linha *conciliacaoLinha, itens []*conciliacaoItemLinha) *domaincaixa.Conciliacao {
	conciliacao := &domaincaixa.Conciliacao{
		ID:                  linha.ID,
		ExtratoLancamentoID: linha.ExtratoLancamentoID,
		ContaBancariaID:     linha.ContaBancariaID,
		Modo:                domaincaixa.ModoConciliacaoDe(linha.Modo),
		Vigente:             linha.Vigente,
		Itens:               make([]domaincaixa.ItemConciliacao, 0, len(itens)),
		UsuarioID:           linha.UsuarioID,
		CriadoEm:            linha.CriadoEm,
		DesfeitaEm:          linha.DesfeitaEm,
		DesfeitaPor:         linha.DesfeitaPor,
		MotivoDesfazer:      linha.MotivoDesfazer,
	}
	for _, item := range itens {
		conciliacao.Itens = append(conciliacao.Itens, domaincaixa.ItemConciliacao{
			Tipo:          domaincaixa.TipoMovimentoDe(item.Tipo),
			AlvoID:        item.AlvoID,
			ValorCentavos: item.ValorCentavos,
			Estornado:     item.Estornado,
		})
	}

	return conciliacao
}
