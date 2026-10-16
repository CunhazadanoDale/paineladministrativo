package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.TituloReceberRepository = (*TituloReceberRepository)(nil)

type TituloReceberRepository struct {
	db *sqlx.DB
}

func NewTituloReceberRepository(db *sqlx.DB) *TituloReceberRepository {
	return &TituloReceberRepository{db: db}
}

var guardaDoTitulo = guardaDeVersao{
	tabela:        "titulo_receber",
	naoEncontrado: domainreceber.ErroTituloNaoEncontrado,
	alterado:      domainreceber.ErroVersaoDivergente,
	conflito: func(restricao string) error {
		if restricao == "uq_titulo_receber_numero" {
			return domainreceber.ErroNumeroRepetido()
		}
		return nil
	},
}

type tituloLinha struct {
	ID                    uuid.UUID  `db:"id"`
	EmpresaID             uuid.UUID  `db:"empresa_id"`
	ClienteID             uuid.UUID  `db:"cliente_id"`
	Origem                string     `db:"origem"`
	OrigemID              *uuid.UUID `db:"origem_id"`
	Numero                string     `db:"numero"`
	Parcela               int        `db:"parcela"`
	Vencimento            time.Time  `db:"vencimento"`
	ValorCentavos         int64      `db:"valor_centavos"`
	CategoriaFinanceiraID *uuid.UUID `db:"categoria_financeira_id"`
	Competencia           time.Time  `db:"competencia"`
	ContaBancariaID       *uuid.UUID `db:"conta_bancaria_id"`
	Status                string     `db:"status"`
	RecebidoCentavos      int64      `db:"recebido_centavos"`
	BoletoEnviadoEm       *time.Time `db:"boleto_enviado_em"`
	BoletoCanal           *string    `db:"boleto_canal"`
	OrigemCanceladaEm     *time.Time `db:"origem_cancelada_em"`
	CanceladoMotivo       *string    `db:"cancelado_motivo"`
	Observacao            string     `db:"observacao"`
	UsuarioID             uuid.UUID  `db:"usuario_id"`
	Versao                int        `db:"versao"`
	CriadoEm              time.Time  `db:"criado_em"`
	AtualizadoEm          time.Time  `db:"atualizado_em"`
}

type tituloHistoricoLinha struct {
	ID         uuid.UUID `db:"id"`
	TituloID   uuid.UUID `db:"titulo_id"`
	UsuarioID  uuid.UUID `db:"usuario_id"`
	Evento     string    `db:"evento"`
	DeStatus   *string   `db:"de_status"`
	ParaStatus string    `db:"para_status"`
	Descricao  string    `db:"descricao"`
	CriadoEm   time.Time `db:"criado_em"`
}

const colunasTitulo = `
	id, empresa_id, cliente_id, origem, origem_id, numero, parcela, vencimento, valor_centavos,
	categoria_financeira_id, competencia, conta_bancaria_id, status, recebido_centavos, boleto_enviado_em,
	boleto_canal, origem_cancelada_em, cancelado_motivo, observacao, usuario_id, versao, criado_em, atualizado_em
`

func (r *TituloReceberRepository) Criar(ctx context.Context, titulo *domainreceber.Titulo, historico *domainreceber.Historico) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaTitulo(ctx, tx, titulo.EmpresaID); err != nil {
		return err
	}
	if err := exigirClienteParaTitulo(ctx, tx, titulo.ClienteID, true); err != nil {
		return err
	}
	if err := exigirCategoriaDeReceitaLancavel(ctx, tx, titulo.CategoriaFinanceiraID); err != nil {
		return err
	}
	if err := exigirContaDeCobranca(ctx, tx, titulo.ContaBancariaID, titulo.EmpresaID); err != nil {
		return err
	}
	if err := inserirTitulo(ctx, tx, titulo); err != nil {
		return err
	}
	if err := inserirHistoricoDoTitulo(ctx, tx, historico); err != nil {
		return err
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (r *TituloReceberRepository) Atualizar(ctx context.Context, alteracao *portsout.Alteracao) error {
	if alteracao == nil || alteracao.Titulo == nil {
		return domain.ErroValidacao("alteração de título sem título")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	titulo := alteracao.Titulo
	if alteracao.ConferirCategoria {
		if err := exigirCategoriaDeReceitaLancavel(ctx, tx, titulo.CategoriaFinanceiraID); err != nil {
			return err
		}
	}
	if alteracao.ConferirConta {
		if err := exigirContaDeCobranca(ctx, tx, titulo.ContaBancariaID, titulo.EmpresaID); err != nil {
			return err
		}
	}

	versao, err := atualizarTitulo(ctx, tx, titulo, alteracao.VersaoLida, alteracao.StatusAnterior)
	if err != nil {
		return err
	}
	if alteracao.Historico != nil {
		if err := inserirHistoricoDoTitulo(ctx, tx, alteracao.Historico); err != nil {
			return err
		}
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	titulo.Versao = versao

	return nil
}

func (r *TituloReceberRepository) Obter(ctx context.Context, id uuid.UUID) (*domainreceber.Titulo, error) {
	var linha tituloLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasTitulo+` FROM titulo_receber WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioTitulo(&linha), nil
}

func (r *TituloReceberRepository) Listar(ctx context.Context, filtro portsout.TituloFiltro) ([]*domainreceber.Titulo, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 10)
	argumentos := make([]any, 0, 12)
	adicionar := func(expressao string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, expressao+" $"+strconv.Itoa(len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("empresa_id =", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ClienteID != nil {
		adicionar("cliente_id =", *filtro.ClienteID)
	}
	if filtro.Status != "" {
		adicionar("status =", filtro.Status.String())
	}
	if filtro.Origem != "" {
		adicionar("origem =", filtro.Origem.String())
	}
	if filtro.OrigemID != nil {
		adicionar("origem_id =", *filtro.OrigemID)
	}
	if filtro.ContaBancariaID != nil {
		adicionar("conta_bancaria_id =", *filtro.ContaBancariaID)
	}
	if filtro.VencimentoDe != nil {
		adicionar("vencimento >=", *filtro.VencimentoDe)
	}
	if filtro.VencimentoAte != nil {
		adicionar("vencimento <=", *filtro.VencimentoAte)
	}
	if filtro.VencidosEm != nil {
		adicionar("status IN ('carteira', 'em_banco', 'antecipado') AND recebido_centavos < valor_centavos AND vencimento <", *filtro.VencidosEm)
	}
	if filtro.OrigemCancelada {
		condicoes = append(condicoes, "origem_cancelada_em IS NOT NULL")
	}

	query := `SELECT ` + colunasTitulo + ` FROM titulo_receber`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY vencimento ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*tituloLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	titulos := make([]*domainreceber.Titulo, 0, len(linhas))
	for _, linha := range linhas {
		titulos = append(titulos, paraDominioTitulo(linha))
	}

	return titulos, nil
}

func (r *TituloReceberRepository) ListarHistorico(ctx context.Context, tituloID uuid.UUID) ([]*domainreceber.Historico, error) {
	var linhas []*tituloHistoricoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT id, titulo_id, usuario_id, evento, de_status, para_status, descricao, criado_em
		FROM titulo_receber_historico
		WHERE titulo_id = $1
		ORDER BY criado_em ASC, id ASC
	`, tituloID); err != nil {
		return nil, err
	}

	historicos := make([]*domainreceber.Historico, 0, len(linhas))
	for _, linha := range linhas {
		historico := &domainreceber.Historico{
			ID:         linha.ID,
			TituloID:   linha.TituloID,
			UsuarioID:  linha.UsuarioID,
			Evento:     domainreceber.EventoDe(linha.Evento),
			ParaStatus: domainreceber.StatusDe(linha.ParaStatus),
			Descricao:  linha.Descricao,
			CriadoEm:   linha.CriadoEm,
		}
		if linha.DeStatus != nil {
			de := domainreceber.StatusDe(*linha.DeStatus)
			historico.DeStatus = &de
		}
		historicos = append(historicos, historico)
	}

	return historicos, nil
}

func inserirTitulo(ctx context.Context, tx *sqlx.Tx, titulo *domainreceber.Titulo) error {
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO titulo_receber (`+colunasTitulo+`) VALUES (
			:id, :empresa_id, :cliente_id, :origem, :origem_id, :numero, :parcela, :vencimento, :valor_centavos,
			:categoria_financeira_id, :competencia, :conta_bancaria_id, :status, :recebido_centavos, :boleto_enviado_em,
			:boleto_canal, :origem_cancelada_em, :cancelado_motivo, :observacao, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaTitulo(titulo)); err != nil {
		return guardaDoTitulo.traduzir(err)
	}

	return nil
}

func atualizarTitulo(ctx context.Context, tx *sqlx.Tx, titulo *domainreceber.Titulo, versaoLida int, anterior domainreceber.Status) (int, error) {
	linha := paraLinhaTitulo(titulo)

	return guardaDoTitulo.atualizar(ctx, tx, linha.ID, `
		UPDATE titulo_receber
		SET numero = $1,
		    parcela = $2,
		    vencimento = $3,
		    valor_centavos = $4,
		    categoria_financeira_id = $5,
		    competencia = $6,
		    conta_bancaria_id = $7,
		    status = $8,
		    recebido_centavos = $9,
		    boleto_enviado_em = $10,
		    boleto_canal = $11,
		    origem_cancelada_em = $12,
		    cancelado_motivo = $13,
		    observacao = $14,
		    atualizado_em = $15,
		    versao = versao + 1
		WHERE id = $16
		  AND versao = $17
		  AND status = $18
		RETURNING versao
	`,
		linha.Numero, linha.Parcela, linha.Vencimento, linha.ValorCentavos, linha.CategoriaFinanceiraID,
		linha.Competencia, linha.ContaBancariaID, linha.Status, linha.RecebidoCentavos, linha.BoletoEnviadoEm,
		linha.BoletoCanal, linha.OrigemCanceladaEm, linha.CanceladoMotivo, linha.Observacao, linha.AtualizadoEm,
		linha.ID, versaoLida, anterior.String(),
	)
}

func inserirHistoricoDoTitulo(ctx context.Context, tx *sqlx.Tx, historico *domainreceber.Historico) error {
	var de *string
	if historico.DeStatus != nil {
		valor := historico.DeStatus.String()
		de = &valor
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO titulo_receber_historico (id, titulo_id, usuario_id, evento, de_status, para_status, descricao, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, historico.ID, historico.TituloID, historico.UsuarioID, historico.Evento.String(), de,
		historico.ParaStatus.String(), historico.Descricao, historico.CriadoEm); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func exigirEmpresaAtivaParaTitulo(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroEmpresaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domainreceber.ErroEmpresaInativa()
	}

	return nil
}

func exigirClienteParaTitulo(ctx context.Context, tx *sqlx.Tx, clienteID uuid.UUID, exigirAtivo bool) error {
	var ativo bool
	err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM cliente WHERE id = $1 FOR SHARE`, clienteID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroClienteNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if exigirAtivo && !ativo {
		return domainreceber.ErroClienteInativo()
	}

	return nil
}

func exigirCategoriaDeReceitaLancavel(ctx context.Context, tx *sqlx.Tx, categoriaID *uuid.UUID) error {
	if categoriaID == nil {
		return nil
	}

	var categoria categoriaParaLancamento
	err := tx.GetContext(ctx, &categoria, `
		SELECT ativa, categoria_pai_id IS NOT NULL AS subcategoria, natureza
		FROM categoria_financeira
		WHERE id = $1
		FOR SHARE
	`, *categoriaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !categoria.Ativa || !categoria.Subcategoria {
		return domainreceber.ErroCategoriaFinanceiraNaoLancavel()
	}
	if categoria.Natureza != domainfinanceiro.NaturezaReceita.String() {
		return domainreceber.ErroCategoriaFinanceiraNaoEhReceita()
	}

	return nil
}

func exigirContaDeCobranca(ctx context.Context, tx *sqlx.Tx, contaBancariaID *uuid.UUID, empresaID uuid.UUID) error {
	if contaBancariaID == nil {
		return nil
	}

	var conta contaParaPagamento
	err := tx.GetContext(ctx, &conta, `SELECT empresa_id, ativa FROM conta_bancaria WHERE id = $1 FOR SHARE`, *contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroContaBancariaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if conta.EmpresaID != empresaID {
		return domainreceber.ErroContaBancariaDeOutraEmpresa()
	}
	if !conta.Ativa {
		return domainreceber.ErroContaBancariaInativa()
	}

	return exigirEmpresaAtivaParaTitulo(ctx, tx, empresaID)
}

func paraLinhaTitulo(titulo *domainreceber.Titulo) tituloLinha {
	linha := tituloLinha{
		ID:                    titulo.ID,
		EmpresaID:             titulo.EmpresaID,
		ClienteID:             titulo.ClienteID,
		Origem:                titulo.Origem.String(),
		OrigemID:              titulo.OrigemID,
		Numero:                titulo.Numero,
		Parcela:               titulo.Parcela,
		Vencimento:            titulo.Vencimento,
		ValorCentavos:         titulo.ValorCentavos,
		CategoriaFinanceiraID: titulo.CategoriaFinanceiraID,
		Competencia:           titulo.Competencia.Inicio(),
		ContaBancariaID:       titulo.ContaBancariaID,
		Status:                titulo.Status.String(),
		RecebidoCentavos:      titulo.RecebidoCentavos,
		BoletoEnviadoEm:       titulo.BoletoEnviadoEm,
		OrigemCanceladaEm:     titulo.OrigemCanceladaEm,
		CanceladoMotivo:       titulo.CanceladoMotivo,
		Observacao:            titulo.Observacao,
		UsuarioID:             titulo.UsuarioID,
		Versao:                titulo.Versao,
		CriadoEm:              titulo.CriadoEm,
		AtualizadoEm:          titulo.AtualizadoEm,
	}
	if titulo.BoletoCanal != nil {
		canal := titulo.BoletoCanal.String()
		linha.BoletoCanal = &canal
	}

	return linha
}

func paraDominioTitulo(linha *tituloLinha) *domainreceber.Titulo {
	titulo := &domainreceber.Titulo{
		ID:                    linha.ID,
		EmpresaID:             linha.EmpresaID,
		ClienteID:             linha.ClienteID,
		Origem:                domainreceber.OrigemDe(linha.Origem),
		OrigemID:              linha.OrigemID,
		Numero:                linha.Numero,
		Parcela:               linha.Parcela,
		Vencimento:            domainreceber.DataDe(linha.Vencimento),
		ValorCentavos:         linha.ValorCentavos,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		Competencia:           domainreceber.CompetenciaDe(linha.Competencia),
		ContaBancariaID:       linha.ContaBancariaID,
		Status:                domainreceber.StatusDe(linha.Status),
		RecebidoCentavos:      linha.RecebidoCentavos,
		BoletoEnviadoEm:       linha.BoletoEnviadoEm,
		OrigemCanceladaEm:     linha.OrigemCanceladaEm,
		CanceladoMotivo:       linha.CanceladoMotivo,
		Observacao:            linha.Observacao,
		UsuarioID:             linha.UsuarioID,
		Versao:                linha.Versao,
		CriadoEm:              linha.CriadoEm,
		AtualizadoEm:          linha.AtualizadoEm,
	}
	if linha.BoletoCanal != nil {
		canal := domainreceber.CanalBoletoDe(*linha.BoletoCanal)
		titulo.BoletoCanal = &canal
	}

	return titulo
}
