package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.DividaRepository = (*DividaRepository)(nil)

type DividaRepository struct {
	db *sqlx.DB
}

func NewDividaRepository(db *sqlx.DB) *DividaRepository {
	return &DividaRepository{db: db}
}

var guardaDaDivida = guardaDeVersao{
	tabela:        "divida",
	naoEncontrado: domaindivida.ErroDividaNaoEncontrada,
	alterado:      domaindivida.ErroVersaoDivergente,
}

type dividaLinha struct {
	ID                     uuid.UUID  `db:"id"`
	EmpresaID              uuid.UUID  `db:"empresa_id"`
	CredorTipo             string     `db:"credor_tipo"`
	FornecedorID           uuid.UUID  `db:"fornecedor_id"`
	Descricao              string     `db:"descricao"`
	Contrato               *string    `db:"contrato"`
	ValorOriginalCentavos  *int64     `db:"valor_original_centavos"`
	ContratadaEm           *time.Time `db:"contratada_em"`
	TaxaBP                 *int       `db:"taxa_bp"`
	TaxaPeriodo            *string    `db:"taxa_periodo"`
	Indexador              string     `db:"indexador"`
	SaldoInicialCentavos   int64      `db:"saldo_inicial_centavos"`
	DataSaldo              time.Time  `db:"data_saldo"`
	ContaLiberacaoID       *uuid.UUID `db:"conta_liberacao_id"`
	LiberadaEm             *time.Time `db:"liberada_em"`
	CategoriaAmortizacaoID uuid.UUID  `db:"categoria_amortizacao_id"`
	CategoriaJurosID       uuid.UUID  `db:"categoria_juros_id"`
	DividaOrigemID         *uuid.UUID `db:"divida_origem_id"`
	Status                 string     `db:"status"`
	Motivo                 *string    `db:"motivo"`
	Observacoes            string     `db:"observacoes"`
	UsuarioID              uuid.UUID  `db:"usuario_id"`
	Versao                 int        `db:"versao"`
	CriadoEm               time.Time  `db:"criado_em"`
	AtualizadoEm           time.Time  `db:"atualizado_em"`
}

type dividaParcelaLinha struct {
	ID                   uuid.UUID  `db:"id"`
	DividaID             uuid.UUID  `db:"divida_id"`
	Numero               int        `db:"numero"`
	Vencimento           time.Time  `db:"vencimento"`
	AmortizacaoCentavos  int64      `db:"amortizacao_centavos"`
	JurosCentavos        int64      `db:"juros_centavos"`
	Status               string     `db:"status"`
	GeradaEm             *time.Time `db:"gerada_em"`
	SubstitutaID         *uuid.UUID `db:"substituta_id"`
	CriadoEm             time.Time  `db:"criado_em"`
	AtualizadoEm         time.Time  `db:"atualizado_em"`
	ContaAmortizacaoID   *uuid.UUID `db:"conta_amortizacao_id"`
	ContaAmortizacaoStat *string    `db:"conta_amortizacao_status"`
	ContaJurosID         *uuid.UUID `db:"conta_juros_id"`
	ContaJurosStatus     *string    `db:"conta_juros_status"`
}

type dividaHistoricoLinha struct {
	ID             uuid.UUID `db:"id"`
	DividaID       uuid.UUID `db:"divida_id"`
	UsuarioID      uuid.UUID `db:"usuario_id"`
	Acao           string    `db:"acao"`
	StatusAnterior *string   `db:"status_anterior"`
	StatusNovo     string    `db:"status_novo"`
	Detalhe        string    `db:"detalhe"`
	CriadoEm       time.Time `db:"criado_em"`
}

type contaDaDividaLinha struct {
	ID     uuid.UUID `db:"id"`
	Status string    `db:"status"`
}

const colunasDivida = `
	id, empresa_id, credor_tipo, fornecedor_id, descricao, contrato, valor_original_centavos, contratada_em, taxa_bp,
	taxa_periodo, indexador, saldo_inicial_centavos, data_saldo, conta_liberacao_id, liberada_em, categoria_amortizacao_id,
	categoria_juros_id, divida_origem_id, status, motivo, observacoes, usuario_id, versao, criado_em, atualizado_em
`

func (r *DividaRepository) Gravar(ctx context.Context, gravacao *portsout.Gravacao) error {
	if gravacao == nil || gravacao.Divida == nil {
		return domain.ErroValidacao("gravação de dívida sem dívida")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	divida := gravacao.Divida
	versao := divida.Versao
	if gravacao.Novo {
		if err := inserirDivida(ctx, tx, divida); err != nil {
			return err
		}
	} else if versao, err = atualizarDivida(ctx, tx, gravacao); err != nil {
		return err
	}
	if gravacao.Substituta != nil {
		if err := inserirDivida(ctx, tx, gravacao.Substituta); err != nil {
			return err
		}
	}

	if err := conferirContasDaDivida(ctx, tx, gravacao); err != nil {
		return err
	}
	if err := cancelarContasDaDivida(ctx, tx, gravacao.ContasCanceladas); err != nil {
		return err
	}
	for _, parcela := range gravacao.ParcelasAlteradas {
		if err := atualizarParcelaDaDivida(ctx, tx, parcela); err != nil {
			return err
		}
	}
	for _, parcela := range gravacao.ParcelasNovas {
		if err := inserirParcelaDaDivida(ctx, tx, parcela); err != nil {
			return err
		}
	}
	for _, conta := range gravacao.ContasAPagar {
		criada, err := gerarSolicitacaoDeOrigem(ctx, tx, conta.Solicitacao, conta.Historico)
		if err != nil {
			return err
		}
		if !criada {
			return domain.ErroConflito("a conta a pagar desta parcela já foi gerada; recarregue a dívida")
		}
	}
	for _, historico := range gravacao.Historicos {
		if err := inserirHistoricoDaDivida(ctx, tx, historico); err != nil {
			return err
		}
	}

	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}
	divida.Versao = versao

	return nil
}

func (r *DividaRepository) Obter(ctx context.Context, id uuid.UUID) (*domaindivida.Divida, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var linha dividaLinha
	if err := tx.GetContext(ctx, &linha, `SELECT `+colunasDivida+` FROM divida WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	divida := paraDominioDivida(&linha)
	if err := carregarParcelasDasDividas(ctx, tx, []*domaindivida.Divida{divida}); err != nil {
		return nil, err
	}

	return divida, tx.Commit()
}

func carregarParcelasDasDividas(ctx context.Context, tx *sqlx.Tx, dividas []*domaindivida.Divida) error {
	if len(dividas) == 0 {
		return nil
	}

	porID := make(map[uuid.UUID]*domaindivida.Divida, len(dividas))
	ids := make([]uuid.UUID, 0, len(dividas))
	for _, divida := range dividas {
		divida.Parcelas = []*domaindivida.Parcela{}
		porID[divida.ID] = divida
		ids = append(ids, divida.ID)
	}

	var parcelas []*dividaParcelaLinha
	if err := tx.SelectContext(ctx, &parcelas, `
		SELECT p.id, p.divida_id, p.numero, p.vencimento, p.amortizacao_centavos, p.juros_centavos, p.status, p.gerada_em,
		       p.substituta_id, p.criado_em, p.atualizado_em,
		       sa.id AS conta_amortizacao_id, sa.status AS conta_amortizacao_status,
		       sj.id AS conta_juros_id, sj.status AS conta_juros_status
		FROM divida_parcela p
		LEFT JOIN solicitacao sa ON sa.origem = 'divida_amortizacao' AND sa.origem_id = p.id
		LEFT JOIN solicitacao sj ON sj.origem = 'divida_juros' AND sj.origem_id = p.id
		WHERE p.divida_id = ANY($1::uuid[])
		ORDER BY p.divida_id, p.numero ASC, p.criado_em ASC, p.id ASC
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return err
	}

	for _, parcela := range parcelas {
		divida := porID[parcela.DividaID]
		divida.Parcelas = append(divida.Parcelas, paraDominioParcelaDaDivida(parcela))
	}

	return nil
}

func condicaoDeParcelaEmAberto(vencidaAntes string) string {
	condicao := `EXISTS (
		SELECT 1 FROM divida_parcela p
		WHERE p.divida_id = d.id AND p.status <> 'cancelada'
		  AND (
		    (p.amortizacao_centavos > 0 AND NOT EXISTS (
		      SELECT 1 FROM solicitacao s WHERE s.origem = 'divida_amortizacao' AND s.origem_id = p.id AND s.status = 'pago'))
		    OR (p.juros_centavos > 0 AND NOT EXISTS (
		      SELECT 1 FROM solicitacao s WHERE s.origem = 'divida_juros' AND s.origem_id = p.id AND s.status = 'pago'))
		  )`
	if vencidaAntes != "" {
		condicao += ` AND p.vencimento < ` + vencidaAntes + `::date`
	}

	return condicao + `)`
}

func (r *DividaRepository) Listar(ctx context.Context, filtro portsout.DividaFiltro) ([]*domaindivida.Divida, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 5)
	argumentos := make([]any, 0, 7)
	adicionar := func(coluna string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, condicao(coluna, len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("empresa_id", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.FornecedorID != nil {
		adicionar("fornecedor_id", *filtro.FornecedorID)
	}
	if filtro.CredorTipo != "" {
		adicionar("credor_tipo", filtro.CredorTipo.String())
	}
	if filtro.Status != "" {
		adicionar("status", filtro.Status.String())
	}
	if filtro.Quitada != nil {
		quitada := `(d.status = 'ativa' AND NOT ` + condicaoDeParcelaEmAberto("") + `)`
		if !*filtro.Quitada {
			quitada = `NOT ` + quitada
		}
		condicoes = append(condicoes, quitada)
	}
	if filtro.Vencidas != nil {
		argumentos = append(argumentos, filtro.Hoje)
		vencidas := `(d.status = 'ativa' AND ` + condicaoDeParcelaEmAberto("$"+strconv.Itoa(len(argumentos))) + `)`
		if !*filtro.Vencidas {
			vencidas = `NOT ` + vencidas
		}
		condicoes = append(condicoes, vencidas)
	}

	query := `SELECT ` + colunasDivida + ` FROM divida d`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var linhas []*dividaLinha
	if err := tx.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	dividas := make([]*domaindivida.Divida, 0, len(linhas))
	for _, linha := range linhas {
		dividas = append(dividas, paraDominioDivida(linha))
	}
	if err := carregarParcelasDasDividas(ctx, tx, dividas); err != nil {
		return nil, err
	}

	return dividas, tx.Commit()
}

func (r *DividaRepository) ListarHistorico(ctx context.Context, dividaID uuid.UUID) ([]*domaindivida.Historico, error) {
	var linhas []*dividaHistoricoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT id, divida_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em
		FROM divida_historico
		WHERE divida_id = $1
		ORDER BY criado_em ASC, id ASC
	`, dividaID); err != nil {
		return nil, err
	}

	historicos := make([]*domaindivida.Historico, 0, len(linhas))
	for _, linha := range linhas {
		historico := &domaindivida.Historico{
			ID:         linha.ID,
			DividaID:   linha.DividaID,
			UsuarioID:  linha.UsuarioID,
			Acao:       domaindivida.AcaoDe(linha.Acao),
			StatusNovo: domaindivida.StatusDe(linha.StatusNovo),
			Detalhe:    linha.Detalhe,
			CriadoEm:   linha.CriadoEm,
		}
		if linha.StatusAnterior != nil {
			anterior := domaindivida.StatusDe(*linha.StatusAnterior)
			historico.StatusAnterior = &anterior
		}
		historicos = append(historicos, historico)
	}

	return historicos, nil
}

func (r *DividaRepository) ComParcelasAGerar(ctx context.Context, limite time.Time, empresaIDs []uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT d.id
		FROM divida d
		JOIN divida_parcela p ON p.divida_id = d.id
		WHERE d.status = 'ativa' AND p.status = 'prevista' AND p.vencimento <= $1::date
	`
	argumentos := []any{limite}
	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` AND d.empresa_id = ANY($2::uuid[])`
	}
	query += ` ORDER BY d.id`

	var ids []uuid.UUID
	if err := r.db.SelectContext(ctx, &ids, query, argumentos...); err != nil {
		return nil, err
	}

	return ids, nil
}

func inserirDivida(ctx context.Context, tx *sqlx.Tx, divida *domaindivida.Divida) error {
	if err := exigirEmpresaDaDividaAtiva(ctx, tx, divida.EmpresaID); err != nil {
		return err
	}
	if err := exigirCredorDaDividaAtivo(ctx, tx, divida.FornecedorID); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO divida (`+colunasDivida+`)
		VALUES (
			:id, :empresa_id, :credor_tipo, :fornecedor_id, :descricao, :contrato, :valor_original_centavos, :contratada_em,
			:taxa_bp, :taxa_periodo, :indexador, :saldo_inicial_centavos, :data_saldo, :conta_liberacao_id, :liberada_em,
			:categoria_amortizacao_id, :categoria_juros_id, :divida_origem_id, :status, :motivo, :observacoes, :usuario_id,
			:versao, :criado_em, :atualizado_em
		)
	`, paraLinhaDivida(divida)); err != nil {
		return tratarErroDeGravacao(err)
	}

	for _, parcela := range divida.Parcelas {
		if err := inserirParcelaDaDivida(ctx, tx, parcela); err != nil {
			return err
		}
	}

	return nil
}

func inserirParcelaDaDivida(ctx context.Context, tx *sqlx.Tx, parcela *domaindivida.Parcela) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO divida_parcela (
			id, divida_id, numero, vencimento, amortizacao_centavos, juros_centavos, status, gerada_em, substituta_id,
			criado_em, atualizado_em
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		parcela.ID, parcela.DividaID, parcela.Numero, parcela.Vencimento, parcela.AmortizacaoCentavos, parcela.JurosCentavos,
		parcela.Status.String(), parcela.GeradaEm, parcela.SubstitutaID, parcela.CriadoEm, parcela.AtualizadoEm,
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func atualizarDivida(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) (int, error) {
	linha := paraLinhaDivida(gravacao.Divida)

	return guardaDaDivida.atualizar(ctx, tx, linha.ID, `
		UPDATE divida
		SET credor_tipo = $1,
		    descricao = $2,
		    contrato = $3,
		    valor_original_centavos = $4,
		    contratada_em = $5,
		    taxa_bp = $6,
		    taxa_periodo = $7,
		    indexador = $8,
		    conta_liberacao_id = $9,
		    liberada_em = $10,
		    categoria_amortizacao_id = $11,
		    categoria_juros_id = $12,
		    status = $13,
		    motivo = $14,
		    observacoes = $15,
		    atualizado_em = $16,
		    versao = versao + 1
		WHERE id = $17
		  AND versao = $18
		  AND status = $19
		RETURNING versao
	`,
		linha.CredorTipo, linha.Descricao, linha.Contrato, linha.ValorOriginalCentavos, linha.ContratadaEm, linha.TaxaBP,
		linha.TaxaPeriodo, linha.Indexador, linha.ContaLiberacaoID, linha.LiberadaEm, linha.CategoriaAmortizacaoID,
		linha.CategoriaJurosID, linha.Status, linha.Motivo, linha.Observacoes, linha.AtualizadoEm,
		linha.ID, gravacao.VersaoLida, gravacao.StatusAnterior.String(),
	)
}

func atualizarParcelaDaDivida(ctx context.Context, tx *sqlx.Tx, parcela *domaindivida.Parcela) error {
	resultado, err := tx.ExecContext(ctx, `
		UPDATE divida_parcela
		SET vencimento = $1,
		    juros_centavos = $2,
		    status = $3,
		    gerada_em = $4,
		    substituta_id = $5,
		    atualizado_em = $6
		WHERE id = $7 AND divida_id = $8
	`,
		parcela.Vencimento, parcela.JurosCentavos, parcela.Status.String(), parcela.GeradaEm, parcela.SubstitutaID,
		parcela.AtualizadoEm, parcela.ID, parcela.DividaID,
	)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas != 1 {
		return domain.ErroConflito("parcela da dívida não encontrada; recarregue a dívida")
	}

	return nil
}

func conferirContasDaDivida(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) error {
	if !gravacao.ConferirContas && len(gravacao.ContasCanceladas) == 0 {
		return nil
	}

	esperados := make(map[uuid.UUID]domainsolicitacao.Status)
	for _, parcela := range gravacao.Divida.Parcelas {
		for _, conta := range parcela.Contas() {
			esperados[conta.SolicitacaoID] = conta.Status
		}
	}
	for _, conta := range gravacao.ContasCanceladas {
		esperados[conta.SolicitacaoID] = conta.StatusAnterior
	}

	var atuais []contaDaDividaLinha
	if err := tx.SelectContext(ctx, &atuais, `
		SELECT s.id, s.status
		FROM solicitacao s
		JOIN divida_parcela p ON p.id = s.origem_id
		WHERE p.divida_id = $1 AND s.origem IN ('divida_amortizacao', 'divida_juros')
		ORDER BY s.id
		FOR UPDATE OF s
	`, gravacao.Divida.ID); err != nil {
		return tratarErroDeGravacao(err)
	}
	if len(atuais) != len(esperados) {
		return domaindivida.ErroContaAPagarAlterada()
	}
	for _, atual := range atuais {
		if esperado, ok := esperados[atual.ID]; !ok || esperado.String() != atual.Status {
			return domaindivida.ErroContaAPagarAlterada()
		}
	}

	return nil
}

func cancelarContasDaDivida(ctx context.Context, tx *sqlx.Tx, contas []portsout.ContaCancelada) error {
	for _, conta := range contas {
		agora := conta.Historico.CriadoEm
		resultado, err := tx.ExecContext(ctx, `
			UPDATE solicitacao
			SET status = 'cancelado',
			    cancelado_em = $1,
			    atualizado_em = $1,
			    versao = versao + 1
			WHERE id = $2 AND status = $3 AND origem IN ('divida_amortizacao', 'divida_juros')
		`, agora, conta.SolicitacaoID, conta.StatusAnterior.String())
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		linhas, err := resultado.RowsAffected()
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		if linhas != 1 {
			return domaindivida.ErroContaAPagarAlterada()
		}
		if err := inserirHistorico(ctx, tx, conta.Historico); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	return nil
}

func exigirEmpresaDaDividaAtiva(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaindivida.ErroEmpresaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domaindivida.ErroEmpresaInativa()
	}

	return nil
}

func exigirCredorDaDividaAtivo(ctx context.Context, tx *sqlx.Tx, fornecedorID uuid.UUID) error {
	var ativo bool
	err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM fornecedor WHERE id = $1 FOR SHARE`, fornecedorID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaindivida.ErroFornecedorNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativo {
		return domaindivida.ErroFornecedorInativo("")
	}

	return nil
}

func inserirHistoricoDaDivida(ctx context.Context, tx *sqlx.Tx, historico *domaindivida.Historico) error {
	var anterior *string
	if historico.StatusAnterior != nil {
		valor := historico.StatusAnterior.String()
		anterior = &valor
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO divida_historico (id, divida_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		historico.ID, historico.DividaID, historico.UsuarioID, historico.Acao.String(),
		anterior, historico.StatusNovo.String(), historico.Detalhe, historico.CriadoEm,
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func paraLinhaDivida(divida *domaindivida.Divida) dividaLinha {
	linha := dividaLinha{
		ID:                     divida.ID,
		EmpresaID:              divida.EmpresaID,
		CredorTipo:             divida.CredorTipo.String(),
		FornecedorID:           divida.FornecedorID,
		Descricao:              divida.Descricao,
		Contrato:               divida.Contrato,
		ValorOriginalCentavos:  divida.ValorOriginalCentavos,
		ContratadaEm:           divida.ContratadaEm,
		Indexador:              divida.Indexador.String(),
		SaldoInicialCentavos:   divida.SaldoInicialCentavos,
		DataSaldo:              divida.DataSaldo,
		ContaLiberacaoID:       divida.ContaLiberacaoID,
		LiberadaEm:             divida.LiberadaEm,
		CategoriaAmortizacaoID: divida.CategoriaAmortizacaoID,
		CategoriaJurosID:       divida.CategoriaJurosID,
		DividaOrigemID:         divida.DividaOrigemID,
		Status:                 divida.Status.String(),
		Motivo:                 divida.Motivo,
		Observacoes:            divida.Observacoes,
		UsuarioID:              divida.UsuarioID,
		Versao:                 divida.Versao,
		CriadoEm:               divida.CriadoEm,
		AtualizadoEm:           divida.AtualizadoEm,
	}
	if divida.Taxa != nil {
		pontosBase := divida.Taxa.PontosBase()
		periodo := divida.Taxa.Periodo().String()
		linha.TaxaBP = &pontosBase
		linha.TaxaPeriodo = &periodo
	}

	return linha
}

func paraDominioDivida(linha *dividaLinha) *domaindivida.Divida {
	divida := &domaindivida.Divida{
		ID:                     linha.ID,
		EmpresaID:              linha.EmpresaID,
		CredorTipo:             domaindivida.TipoCredorDe(linha.CredorTipo),
		FornecedorID:           linha.FornecedorID,
		Descricao:              linha.Descricao,
		Contrato:               linha.Contrato,
		ValorOriginalCentavos:  linha.ValorOriginalCentavos,
		ContratadaEm:           dataDaDivida(linha.ContratadaEm),
		Indexador:              domaindivida.IndexadorDe(linha.Indexador),
		SaldoInicialCentavos:   linha.SaldoInicialCentavos,
		DataSaldo:              domaindivida.DataDe(linha.DataSaldo),
		ContaLiberacaoID:       linha.ContaLiberacaoID,
		LiberadaEm:             dataDaDivida(linha.LiberadaEm),
		CategoriaAmortizacaoID: linha.CategoriaAmortizacaoID,
		CategoriaJurosID:       linha.CategoriaJurosID,
		DividaOrigemID:         linha.DividaOrigemID,
		Status:                 domaindivida.StatusDe(linha.Status),
		Motivo:                 linha.Motivo,
		Observacoes:            linha.Observacoes,
		UsuarioID:              linha.UsuarioID,
		Versao:                 linha.Versao,
		CriadoEm:               linha.CriadoEm,
		AtualizadoEm:           linha.AtualizadoEm,
	}
	if linha.TaxaBP != nil && linha.TaxaPeriodo != nil {
		taxa := domaindivida.TaxaDe(*linha.TaxaBP, *linha.TaxaPeriodo)
		divida.Taxa = &taxa
	}

	return divida
}

func paraDominioParcelaDaDivida(linha *dividaParcelaLinha) *domaindivida.Parcela {
	parcela := &domaindivida.Parcela{
		ID:                  linha.ID,
		DividaID:            linha.DividaID,
		Numero:              linha.Numero,
		Vencimento:          domaindivida.DataDe(linha.Vencimento),
		AmortizacaoCentavos: linha.AmortizacaoCentavos,
		JurosCentavos:       linha.JurosCentavos,
		Status:              domaindivida.StatusParcelaDe(linha.Status),
		GeradaEm:            linha.GeradaEm,
		SubstitutaID:        linha.SubstitutaID,
		CriadoEm:            linha.CriadoEm,
		AtualizadoEm:        linha.AtualizadoEm,
	}
	if linha.ContaAmortizacaoID != nil && linha.ContaAmortizacaoStat != nil {
		parcela.ContaAmortizacao = &domaindivida.Conta{
			SolicitacaoID: *linha.ContaAmortizacaoID, Status: domainsolicitacao.StatusDe(*linha.ContaAmortizacaoStat),
		}
	}
	if linha.ContaJurosID != nil && linha.ContaJurosStatus != nil {
		parcela.ContaJuros = &domaindivida.Conta{
			SolicitacaoID: *linha.ContaJurosID, Status: domainsolicitacao.StatusDe(*linha.ContaJurosStatus),
		}
	}

	return parcela
}

func dataDaDivida(valor *time.Time) *time.Time {
	if valor == nil {
		return nil
	}

	data := domaindivida.DataDe(*valor)

	return &data
}
