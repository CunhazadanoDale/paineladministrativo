package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type saldoAbertoLinha struct {
	RepresentanteID   uuid.UUID `db:"representante_id"`
	ComissoesCentavos int64     `db:"comissoes_centavos"`
	AjustesCentavos   int64     `db:"ajustes_centavos"`
}

type fechamentoLinha struct {
	ID                    uuid.UUID `db:"id"`
	EmpresaID             uuid.UUID `db:"empresa_id"`
	RepresentanteID       uuid.UUID `db:"representante_id"`
	Competencia           time.Time `db:"competencia"`
	ComissoesCentavos     int64     `db:"comissoes_centavos"`
	AjustesCentavos       int64     `db:"ajustes_centavos"`
	TotalCentavos         int64     `db:"total_centavos"`
	Vencimento            time.Time `db:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `db:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID `db:"solicitacao_id"`
	UsuarioID             uuid.UUID `db:"usuario_id"`
	CriadoEm              time.Time `db:"criado_em"`
}

type fechamentoComContaLinha struct {
	fechamentoLinha
	StatusContaAPagar string `db:"status_conta_a_pagar"`
}

type valorTravadoLinha struct {
	ID            uuid.UUID `db:"id"`
	ValorCentavos int64     `db:"valor_centavos"`
}

const colunasFechamento = `
	id, empresa_id, representante_id, competencia, comissoes_centavos, ajustes_centavos, total_centavos,
	vencimento, categoria_financeira_id, solicitacao_id, usuario_id, criado_em
`

const selecaoFechamento = `
	SELECT f.id, f.empresa_id, f.representante_id, f.competencia, f.comissoes_centavos, f.ajustes_centavos,
		f.total_centavos, f.vencimento, f.categoria_financeira_id, f.solicitacao_id, f.usuario_id, f.criado_em,
		s.status AS status_conta_a_pagar
	FROM comissao_fechamento f
	JOIN solicitacao s ON s.id = f.solicitacao_id
`

func (r *ComissaoRepository) SaldosAbertos(ctx context.Context, filtro portsout.SaldoFiltro) ([]domaincomissao.SaldoAberto, error) {
	argumentos := []any{filtro.EmpresaID, filtro.Ate.Inicio()}
	condicaoRepresentante := ""
	if filtro.RepresentanteID != nil {
		argumentos = append(argumentos, *filtro.RepresentanteID)
		condicaoRepresentante = " AND representante_id = $3"
	}

	var linhas []*saldoAbertoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT representante_id,
			COALESCE(SUM(comissoes), 0)::bigint AS comissoes_centavos,
			COALESCE(SUM(ajustes), 0)::bigint AS ajustes_centavos
		FROM (
			SELECT representante_id, valor_centavos AS comissoes, 0::bigint AS ajustes
			FROM comissao
			WHERE empresa_id = $1 AND competencia <= $2 AND fechamento_id IS NULL`+condicaoRepresentante+`
			UNION ALL
			SELECT representante_id, 0::bigint, valor_centavos
			FROM comissao_ajuste
			WHERE empresa_id = $1 AND competencia <= $2 AND fechamento_id IS NULL`+condicaoRepresentante+`
		) abertos
		GROUP BY representante_id
		ORDER BY representante_id
	`, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	saldos := make([]domaincomissao.SaldoAberto, 0, len(linhas))
	for _, linha := range linhas {
		saldos = append(saldos, domaincomissao.SaldoAberto{
			RepresentanteID:   linha.RepresentanteID,
			ComissoesCentavos: linha.ComissoesCentavos,
			AjustesCentavos:   linha.AjustesCentavos,
		})
	}

	return saldos, nil
}

func (r *ComissaoRepository) Fechar(ctx context.Context, gravacao *portsout.GravacaoFechamento) error {
	if gravacao == nil || gravacao.Fechamento == nil || gravacao.Solicitacao == nil || gravacao.Historico == nil {
		return domain.ErroValidacao("fechamento de comissão incompleto")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	fechamento := gravacao.Fechamento
	comissoes, err := travarAbertos(ctx, tx, "comissao", fechamento)
	if err != nil {
		return err
	}
	ajustes, err := travarAbertos(ctx, tx, "comissao_ajuste", fechamento)
	if err != nil {
		return err
	}
	if somaDosValores(comissoes) != fechamento.ComissoesCentavos || somaDosValores(ajustes) != fechamento.AjustesCentavos {
		return domaincomissao.ErroSaldoMudouNoFechamento()
	}

	criada, err := gerarSolicitacaoDeOrigem(ctx, tx, gravacao.Solicitacao, gravacao.Historico)
	if err != nil {
		return err
	}
	if !criada {
		return domaincomissao.ErroContaAPagarJaGerada()
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO comissao_fechamento (`+colunasFechamento+`) VALUES (
			:id, :empresa_id, :representante_id, :competencia, :comissoes_centavos, :ajustes_centavos, :total_centavos,
			:vencimento, :categoria_financeira_id, :solicitacao_id, :usuario_id, :criado_em
		)
	`, paraLinhaFechamento(fechamento)); err != nil {
		return traduzirErroDoFechamento(err)
	}

	if err := marcarFechados(ctx, tx, "comissao", fechamento.ID, comissoes); err != nil {
		return err
	}
	if err := marcarFechados(ctx, tx, "comissao_ajuste", fechamento.ID, ajustes); err != nil {
		return err
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (r *ComissaoRepository) ObterFechamento(ctx context.Context, id uuid.UUID) (*domaincomissao.Fechamento, error) {
	var linha fechamentoComContaLinha
	err := r.db.GetContext(ctx, &linha, selecaoFechamento+` WHERE f.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	return paraDominioFechamento(&linha), nil
}

func (r *ComissaoRepository) ListarFechamentos(ctx context.Context, filtro portsout.FechamentoFiltro) ([]*domaincomissao.Fechamento, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 4)
	argumentos := make([]any, 0, 6)
	adicionar := func(expressao string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, expressao+" $"+strconv.Itoa(len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("f.empresa_id =", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "f.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.RepresentanteID != nil {
		adicionar("f.representante_id =", *filtro.RepresentanteID)
	}
	if filtro.Competencia != nil {
		adicionar("f.competencia =", filtro.Competencia.Inicio())
	}

	query := selecaoFechamento
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY f.competencia DESC, f.criado_em DESC, f.id DESC` + limiteOffset(argumentos)

	var linhas []*fechamentoComContaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	fechamentos := make([]*domaincomissao.Fechamento, 0, len(linhas))
	for _, linha := range linhas {
		fechamentos = append(fechamentos, paraDominioFechamento(linha))
	}

	return fechamentos, nil
}

func (r *ComissaoRepository) ItensDoFechamento(ctx context.Context, fechamentoID uuid.UUID) ([]*domaincomissao.Comissao, []*domaincomissao.Ajuste, error) {
	var linhasComissao []*comissaoLinha
	if err := r.db.SelectContext(ctx, &linhasComissao, `
		SELECT `+colunasComissao+` FROM comissao WHERE fechamento_id = $1 ORDER BY competencia, criado_em, id
	`, fechamentoID); err != nil {
		return nil, nil, tratarErroDeConsulta(err)
	}

	var linhasAjuste []*ajusteLinha
	if err := r.db.SelectContext(ctx, &linhasAjuste, `
		SELECT `+colunasAjuste+` FROM comissao_ajuste WHERE fechamento_id = $1 ORDER BY competencia, criado_em, id
	`, fechamentoID); err != nil {
		return nil, nil, tratarErroDeConsulta(err)
	}

	comissoes := make([]*domaincomissao.Comissao, 0, len(linhasComissao))
	for _, linha := range linhasComissao {
		comissoes = append(comissoes, paraDominioComissao(linha))
	}

	return comissoes, paraDominioAjustes(linhasAjuste), nil
}

func travarAbertos(ctx context.Context, tx *sqlx.Tx, tabela string, fechamento *domaincomissao.Fechamento) ([]valorTravadoLinha, error) {
	var linhas []valorTravadoLinha
	if err := tx.SelectContext(ctx, &linhas, `
		SELECT id, valor_centavos
		FROM `+tabela+`
		WHERE empresa_id = $1 AND representante_id = $2 AND competencia <= $3 AND fechamento_id IS NULL
		ORDER BY id
		FOR UPDATE
	`, fechamento.EmpresaID, fechamento.RepresentanteID, fechamento.Competencia.Inicio()); err != nil {
		return nil, tratarErroDeGravacao(err)
	}

	return linhas, nil
}

func marcarFechados(ctx context.Context, tx *sqlx.Tx, tabela string, fechamentoID uuid.UUID, linhas []valorTravadoLinha) error {
	if len(linhas) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(linhas))
	for _, linha := range linhas {
		ids = append(ids, linha.ID)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE `+tabela+` SET fechamento_id = $1 WHERE id = ANY($2::uuid[])`,
		fechamentoID, pq.Array(textosDeIDs(ids))); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func somaDosValores(linhas []valorTravadoLinha) int64 {
	var soma int64
	for _, linha := range linhas {
		soma += linha.ValorCentavos
	}

	return soma
}

func traduzirErroDoFechamento(err error) error {
	var erroPostgres *pq.Error
	if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation && erroPostgres.Constraint == "uq_comissao_fechamento" {
		return domaincomissao.ErroFechamentoRepetido()
	}

	return tratarErroDeGravacao(err)
}

func paraLinhaFechamento(fechamento *domaincomissao.Fechamento) *fechamentoLinha {
	return &fechamentoLinha{
		ID:                    fechamento.ID,
		EmpresaID:             fechamento.EmpresaID,
		RepresentanteID:       fechamento.RepresentanteID,
		Competencia:           fechamento.Competencia.Inicio(),
		ComissoesCentavos:     fechamento.ComissoesCentavos,
		AjustesCentavos:       fechamento.AjustesCentavos,
		TotalCentavos:         fechamento.TotalCentavos,
		Vencimento:            fechamento.Vencimento,
		CategoriaFinanceiraID: fechamento.CategoriaFinanceiraID,
		SolicitacaoID:         fechamento.SolicitacaoID,
		UsuarioID:             fechamento.UsuarioID,
		CriadoEm:              fechamento.CriadoEm,
	}
}

func paraDominioFechamento(linha *fechamentoComContaLinha) *domaincomissao.Fechamento {
	return &domaincomissao.Fechamento{
		ID:                    linha.ID,
		EmpresaID:             linha.EmpresaID,
		RepresentanteID:       linha.RepresentanteID,
		Competencia:           domaincomissao.CompetenciaDe(linha.Competencia),
		ComissoesCentavos:     linha.ComissoesCentavos,
		AjustesCentavos:       linha.AjustesCentavos,
		TotalCentavos:         linha.TotalCentavos,
		Vencimento:            linha.Vencimento,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		SolicitacaoID:         linha.SolicitacaoID,
		StatusContaAPagar:     linha.StatusContaAPagar,
		UsuarioID:             linha.UsuarioID,
		CriadoEm:              linha.CriadoEm,
	}
}
