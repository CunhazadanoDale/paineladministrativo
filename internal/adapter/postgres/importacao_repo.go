package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ImportacaoRepository = (*ImportacaoRepository)(nil)

type ImportacaoRepository struct {
	db *sqlx.DB
}

func NewImportacaoRepository(db *sqlx.DB) *ImportacaoRepository {
	return &ImportacaoRepository{db: db}
}

var guardaDaImportacao = guardaDeVersao{
	tabela:        "importacao",
	naoEncontrado: func() error { return domain.ErroNaoEncontrado("importação não encontrada") },
	alterado:      domainimportacao.ErroVersaoDivergente,
}

type importacaoLinha struct {
	ID                        uuid.UUID  `db:"id"`
	EmpresaID                 uuid.UUID  `db:"empresa_id"`
	Numero                    int        `db:"numero"`
	FornecedorID              uuid.UUID  `db:"fornecedor_id"`
	Referencia                *string    `db:"referencia"`
	Moeda                     string     `db:"moeda"`
	Status                    string     `db:"status"`
	ValorTotalMoedaCentesimos int64      `db:"valor_total_moeda_centesimos"`
	EmbarquePrevisto          *time.Time `db:"embarque_previsto"`
	EmbarcadaEm               *time.Time `db:"embarcada_em"`
	ChegadaPrevista           *time.Time `db:"chegada_prevista"`
	ChegadaEm                 *time.Time `db:"chegada_em"`
	DesembaracadaEm           *time.Time `db:"desembaracada_em"`
	DocumentoEmbarque         *string    `db:"documento_embarque"`
	Declaracao                *string    `db:"declaracao"`
	Observacoes               string     `db:"observacoes"`
	CanceladaMotivo           *string    `db:"cancelada_motivo"`
	UsuarioID                 uuid.UUID  `db:"usuario_id"`
	Versao                    int        `db:"versao"`
	CriadoEm                  time.Time  `db:"criado_em"`
	AtualizadoEm              time.Time  `db:"atualizado_em"`
}

type importacaoItemLinha struct {
	ID                   uuid.UUID `db:"id"`
	ImportacaoID         uuid.UUID `db:"importacao_id"`
	ProdutoID            uuid.UUID `db:"produto_id"`
	Quantidade           int       `db:"quantidade"`
	ValorMoedaCentesimos int64     `db:"valor_moeda_centesimos"`
	CriadoEm             time.Time `db:"criado_em"`
}

type importacaoHistoricoLinha struct {
	ID             uuid.UUID `db:"id"`
	ImportacaoID   uuid.UUID `db:"importacao_id"`
	UsuarioID      uuid.UUID `db:"usuario_id"`
	Acao           string    `db:"acao"`
	StatusAnterior *string   `db:"status_anterior"`
	StatusNovo     string    `db:"status_novo"`
	Detalhe        string    `db:"detalhe"`
	CriadoEm       time.Time `db:"criado_em"`
}

const colunasImportacao = `
	id, empresa_id, numero, fornecedor_id, referencia, moeda, status, valor_total_moeda_centesimos,
	embarque_previsto, embarcada_em, chegada_prevista, chegada_em, desembaracada_em, documento_embarque,
	declaracao, observacoes, cancelada_motivo, usuario_id, versao, criado_em, atualizado_em
`

const colunasImportacaoItem = `id, importacao_id, produto_id, quantidade, valor_moeda_centesimos, criado_em`

func (r *ImportacaoRepository) Gravar(ctx context.Context, gravacao *portsout.Gravacao) error {
	if gravacao == nil || gravacao.Importacao == nil {
		return domain.ErroValidacao("gravação de importação sem importação")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	importacao := gravacao.Importacao
	if gravacao.FornecedorTrocado {
		if err := exigirFornecedorDaImportacaoAtivo(ctx, tx, importacao.FornecedorID); err != nil {
			return err
		}
	}

	numero := importacao.Numero
	versao := importacao.Versao
	if gravacao.Novo {
		if numero, err = inserirImportacao(ctx, tx, importacao); err != nil {
			return err
		}
	} else if versao, err = atualizarImportacao(ctx, tx, gravacao); err != nil {
		return err
	}

	if gravacao.Novo || gravacao.SubstituirItens {
		if err := gravarItensDaImportacao(ctx, tx, importacao); err != nil {
			return err
		}
	}
	for _, historico := range gravacao.Historicos {
		if err := inserirHistoricoDaImportacao(ctx, tx, historico); err != nil {
			return err
		}
	}

	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	importacao.Numero = numero
	importacao.Versao = versao

	return nil
}

func (r *ImportacaoRepository) Obter(ctx context.Context, id uuid.UUID) (*domainimportacao.Importacao, error) {
	var linha importacaoLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasImportacao+` FROM importacao WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var itens []*importacaoItemLinha
	if err := r.db.SelectContext(ctx, &itens,
		`SELECT `+colunasImportacaoItem+` FROM importacao_item WHERE importacao_id = $1 ORDER BY criado_em ASC, id ASC`, id,
	); err != nil {
		return nil, err
	}

	importacao := paraDominioImportacao(&linha)
	importacao.Itens = make([]*domainimportacao.Item, 0, len(itens))
	for _, item := range itens {
		importacao.Itens = append(importacao.Itens, &domainimportacao.Item{
			ID:           item.ID,
			ImportacaoID: item.ImportacaoID,
			ProdutoID:    item.ProdutoID,
			Quantidade:   item.Quantidade,
			Valor:        domainimportacao.ValorMoedaDe(item.ValorMoedaCentesimos),
			CriadoEm:     item.CriadoEm,
		})
	}

	return importacao, nil
}

func (r *ImportacaoRepository) Listar(ctx context.Context, filtro portsout.ImportacaoFiltro) ([]*domainimportacao.Importacao, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 6)
	argumentos := make([]any, 0, 8)
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
	if filtro.Status != "" {
		adicionar("status", filtro.Status.String())
	}
	if filtro.Numero != nil {
		adicionar("numero", *filtro.Numero)
	}
	if filtro.ChegadaAte != nil {
		argumentos = append(argumentos, *filtro.ChegadaAte)
		condicoes = append(condicoes, "chegada_prevista <= $"+strconv.Itoa(len(argumentos)))
	}

	query := `SELECT ` + colunasImportacao + ` FROM importacao`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*importacaoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	importacoes := make([]*domainimportacao.Importacao, 0, len(linhas))
	for _, linha := range linhas {
		importacoes = append(importacoes, paraDominioImportacao(linha))
	}

	return importacoes, nil
}

func (r *ImportacaoRepository) ListarHistorico(ctx context.Context, importacaoID uuid.UUID) ([]*domainimportacao.Historico, error) {
	var linhas []*importacaoHistoricoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT id, importacao_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em
		FROM importacao_historico
		WHERE importacao_id = $1
		ORDER BY criado_em ASC, id ASC
	`, importacaoID); err != nil {
		return nil, err
	}

	historicos := make([]*domainimportacao.Historico, 0, len(linhas))
	for _, linha := range linhas {
		historico := &domainimportacao.Historico{
			ID:           linha.ID,
			ImportacaoID: linha.ImportacaoID,
			UsuarioID:    linha.UsuarioID,
			Acao:         domainimportacao.AcaoDe(linha.Acao),
			StatusNovo:   domainimportacao.StatusDe(linha.StatusNovo),
			Detalhe:      linha.Detalhe,
			CriadoEm:     linha.CriadoEm,
		}
		if linha.StatusAnterior != nil {
			anterior := domainimportacao.StatusDe(*linha.StatusAnterior)
			historico.StatusAnterior = &anterior
		}
		historicos = append(historicos, historico)
	}

	return historicos, nil
}

func exigirEmpresaDaImportacaoAtiva(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainimportacao.ErroEmpresaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domainimportacao.ErroEmpresaInativa()
	}

	return nil
}

func exigirFornecedorDaImportacaoAtivo(ctx context.Context, tx *sqlx.Tx, fornecedorID uuid.UUID) error {
	var ativo bool
	err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM fornecedor WHERE id = $1 FOR SHARE`, fornecedorID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainimportacao.ErroFornecedorNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativo {
		return domainimportacao.ErroFornecedorInativo("")
	}

	return nil
}

func inserirImportacao(ctx context.Context, tx *sqlx.Tx, importacao *domainimportacao.Importacao) (int, error) {
	if err := exigirEmpresaDaImportacaoAtiva(ctx, tx, importacao.EmpresaID); err != nil {
		return 0, err
	}

	var numero int
	if err := tx.GetContext(ctx, &numero, `
		INSERT INTO importacao_sequencia (empresa_id, ultimo_numero)
		VALUES ($1, 1)
		ON CONFLICT (empresa_id) DO UPDATE
		SET ultimo_numero = importacao_sequencia.ultimo_numero + 1
		RETURNING ultimo_numero
	`, importacao.EmpresaID); err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	linha := paraLinhaImportacao(importacao)
	linha.Numero = numero
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO importacao (`+colunasImportacao+`)
		VALUES (
			:id, :empresa_id, :numero, :fornecedor_id, :referencia, :moeda, :status, :valor_total_moeda_centesimos,
			:embarque_previsto, :embarcada_em, :chegada_prevista, :chegada_em, :desembaracada_em, :documento_embarque,
			:declaracao, :observacoes, :cancelada_motivo, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, linha); err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	return numero, nil
}

func atualizarImportacao(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) (int, error) {
	linha := paraLinhaImportacao(gravacao.Importacao)

	return guardaDaImportacao.atualizar(ctx, tx, linha.ID, `
		UPDATE importacao
		SET fornecedor_id = $1,
		    referencia = $2,
		    moeda = $3,
		    status = $4,
		    valor_total_moeda_centesimos = $5,
		    embarque_previsto = $6,
		    embarcada_em = $7,
		    chegada_prevista = $8,
		    chegada_em = $9,
		    desembaracada_em = $10,
		    documento_embarque = $11,
		    declaracao = $12,
		    observacoes = $13,
		    cancelada_motivo = $14,
		    atualizado_em = $15,
		    versao = versao + 1
		WHERE id = $16
		  AND versao = $17
		  AND status = $18
		RETURNING versao
	`,
		linha.FornecedorID, linha.Referencia, linha.Moeda, linha.Status, linha.ValorTotalMoedaCentesimos,
		linha.EmbarquePrevisto, linha.EmbarcadaEm, linha.ChegadaPrevista, linha.ChegadaEm, linha.DesembaracadaEm,
		linha.DocumentoEmbarque, linha.Declaracao, linha.Observacoes, linha.CanceladaMotivo, linha.AtualizadoEm,
		linha.ID, gravacao.VersaoLida, gravacao.StatusAnterior.String(),
	)
}

func gravarItensDaImportacao(ctx context.Context, tx *sqlx.Tx, importacao *domainimportacao.Importacao) error {
	ids := make([]uuid.UUID, 0, len(importacao.Itens))
	for _, item := range importacao.Itens {
		ids = append(ids, item.ID)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM importacao_item WHERE importacao_id = $1 AND NOT (id = ANY($2::uuid[]))`,
		importacao.ID, pq.Array(textosDeIDs(ids)),
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	for _, item := range importacao.Itens {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO importacao_item (`+colunasImportacaoItem+`)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE
			SET quantidade = EXCLUDED.quantidade,
			    valor_moeda_centesimos = EXCLUDED.valor_moeda_centesimos
			WHERE importacao_item.importacao_id = EXCLUDED.importacao_id
		`, item.ID, importacao.ID, item.ProdutoID, item.Quantidade, item.Valor.Centesimos(), item.CriadoEm); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	return nil
}

func inserirHistoricoDaImportacao(ctx context.Context, tx *sqlx.Tx, historico *domainimportacao.Historico) error {
	var anterior *string
	if historico.StatusAnterior != nil {
		valor := historico.StatusAnterior.String()
		anterior = &valor
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO importacao_historico (id, importacao_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		historico.ID, historico.ImportacaoID, historico.UsuarioID, historico.Acao.String(),
		anterior, historico.StatusNovo.String(), historico.Detalhe, historico.CriadoEm,
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func paraLinhaImportacao(importacao *domainimportacao.Importacao) importacaoLinha {
	return importacaoLinha{
		ID:                        importacao.ID,
		EmpresaID:                 importacao.EmpresaID,
		Numero:                    importacao.Numero,
		FornecedorID:              importacao.FornecedorID,
		Referencia:                importacao.Referencia,
		Moeda:                     importacao.Moeda.String(),
		Status:                    importacao.Status.String(),
		ValorTotalMoedaCentesimos: importacao.ValorTotal.Centesimos(),
		EmbarquePrevisto:          importacao.EmbarquePrevisto,
		EmbarcadaEm:               importacao.EmbarcadaEm,
		ChegadaPrevista:           importacao.ChegadaPrevista,
		ChegadaEm:                 importacao.ChegadaEm,
		DesembaracadaEm:           importacao.DesembaracadaEm,
		DocumentoEmbarque:         importacao.DocumentoEmbarque,
		Declaracao:                importacao.Declaracao,
		Observacoes:               importacao.Observacoes,
		CanceladaMotivo:           importacao.CanceladaMotivo,
		UsuarioID:                 importacao.UsuarioID,
		Versao:                    importacao.Versao,
		CriadoEm:                  importacao.CriadoEm,
		AtualizadoEm:              importacao.AtualizadoEm,
	}
}

func paraDominioImportacao(linha *importacaoLinha) *domainimportacao.Importacao {
	return &domainimportacao.Importacao{
		ID:                linha.ID,
		EmpresaID:         linha.EmpresaID,
		Numero:            linha.Numero,
		FornecedorID:      linha.FornecedorID,
		Referencia:        linha.Referencia,
		Moeda:             domainfornecedor.MoedaDe(linha.Moeda),
		Status:            domainimportacao.StatusDe(linha.Status),
		ValorTotal:        domainimportacao.ValorMoedaDe(linha.ValorTotalMoedaCentesimos),
		EmbarquePrevisto:  dataDaImportacao(linha.EmbarquePrevisto),
		EmbarcadaEm:       dataDaImportacao(linha.EmbarcadaEm),
		ChegadaPrevista:   dataDaImportacao(linha.ChegadaPrevista),
		ChegadaEm:         dataDaImportacao(linha.ChegadaEm),
		DesembaracadaEm:   dataDaImportacao(linha.DesembaracadaEm),
		DocumentoEmbarque: linha.DocumentoEmbarque,
		Declaracao:        linha.Declaracao,
		Observacoes:       linha.Observacoes,
		CanceladaMotivo:   linha.CanceladaMotivo,
		UsuarioID:         linha.UsuarioID,
		Versao:            linha.Versao,
		CriadoEm:          linha.CriadoEm,
		AtualizadoEm:      linha.AtualizadoEm,
	}
}

func dataDaImportacao(valor *time.Time) *time.Time {
	if valor == nil {
		return nil
	}

	data := domainimportacao.DataDe(*valor)

	return &data
}
