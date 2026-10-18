package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.AntecipacaoRepository = (*AntecipacaoRepository)(nil)

type AntecipacaoRepository struct {
	db *sqlx.DB
}

func NewAntecipacaoRepository(db *sqlx.DB) *AntecipacaoRepository {
	return &AntecipacaoRepository{db: db}
}

var guardaDaAntecipacao = guardaDeVersao{
	tabela:        "antecipacao",
	naoEncontrado: domainreceber.ErroAntecipacaoNaoEncontrada,
	alterado:      domainreceber.ErroAntecipacaoAlterada,
}

type antecipacaoLinha struct {
	ID                   uuid.UUID  `db:"id"`
	EmpresaID            uuid.UUID  `db:"empresa_id"`
	FornecedorID         uuid.UUID  `db:"fornecedor_id"`
	Numero               *string    `db:"numero"`
	OperadoEm            time.Time  `db:"operado_em"`
	TaxaBP               int        `db:"taxa_bp"`
	ValorBrutoCentavos   int64      `db:"valor_bruto_centavos"`
	ValorLiquidoCentavos int64      `db:"valor_liquido_centavos"`
	ContaBancariaID      uuid.UUID  `db:"conta_bancaria_id"`
	Status               string     `db:"status"`
	EstornoMotivo        *string    `db:"estorno_motivo"`
	EstornadoEm          *time.Time `db:"estornado_em"`
	UsuarioID            uuid.UUID  `db:"usuario_id"`
	Versao               int        `db:"versao"`
	CriadoEm             time.Time  `db:"criado_em"`
	AtualizadoEm         time.Time  `db:"atualizado_em"`
}

type antecipacaoTituloLinha struct {
	AntecipacaoID  uuid.UUID `db:"antecipacao_id"`
	TituloID       uuid.UUID `db:"titulo_id"`
	ValorCentavos  int64     `db:"valor_centavos"`
	StatusAnterior string    `db:"status_anterior"`
	Vigente        bool      `db:"vigente"`
}

type clienteParaAntecipacao struct {
	Nome              string `db:"nome"`
	ProibeAntecipacao bool   `db:"proibe_antecipacao"`
}

const colunasAntecipacao = `
	id, empresa_id, fornecedor_id, numero, operado_em, taxa_bp, valor_bruto_centavos, valor_liquido_centavos,
	conta_bancaria_id, status, estorno_motivo, estornado_em, usuario_id, versao, criado_em, atualizado_em
`

func (r *AntecipacaoRepository) Criar(ctx context.Context, gravacao *portsout.GravacaoAntecipacao) error {
	if gravacao == nil || gravacao.Antecipacao == nil || len(gravacao.Titulos) == 0 {
		return domain.ErroValidacao("antecipação sem títulos")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	antecipacao := gravacao.Antecipacao
	if err := exigirFomentoAtivo(ctx, tx, antecipacao.FornecedorID); err != nil {
		return err
	}
	conta := antecipacao.ContaBancariaID
	if err := exigirContaDeCobranca(ctx, tx, &conta, antecipacao.EmpresaID); err != nil {
		return err
	}
	for _, alterado := range gravacao.Titulos {
		if err := exigirClienteQuePermiteAntecipacao(ctx, tx, alterado.Titulo.ClienteID); err != nil {
			return err
		}
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO antecipacao (`+colunasAntecipacao+`) VALUES (
			:id, :empresa_id, :fornecedor_id, :numero, :operado_em, :taxa_bp, :valor_bruto_centavos, :valor_liquido_centavos,
			:conta_bancaria_id, :status, :estorno_motivo, :estornado_em, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaAntecipacao(antecipacao)); err != nil {
		return tratarErroDeGravacao(err)
	}
	for _, vinculo := range antecipacao.Titulos {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO antecipacao_titulo (antecipacao_id, titulo_id, valor_centavos, status_anterior, vigente)
			VALUES ($1, $2, $3, $4, $5)
		`, vinculo.AntecipacaoID, vinculo.TituloID, vinculo.ValorCentavos, vinculo.StatusAnterior.String(), vinculo.Vigente); err != nil {
			return traduzirVinculoDaAntecipacao(err)
		}
	}

	versoes, err := atualizarTitulosAlterados(ctx, tx, gravacao.Titulos)
	if err != nil {
		return err
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}
	aplicarVersoes(gravacao.Titulos, versoes)

	return nil
}

func (r *AntecipacaoRepository) Obter(ctx context.Context, id uuid.UUID) (*domainreceber.Antecipacao, error) {
	var linha antecipacaoLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasAntecipacao+` FROM antecipacao WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	antecipacoes := []*domainreceber.Antecipacao{paraDominioAntecipacao(&linha)}
	if err := r.carregarTitulos(ctx, antecipacoes); err != nil {
		return nil, err
	}

	return antecipacoes[0], nil
}

func (r *AntecipacaoRepository) ObterVigenteDoTitulo(ctx context.Context, tituloID uuid.UUID) (*domainreceber.Antecipacao, error) {
	var antecipacaoID uuid.UUID
	err := r.db.GetContext(ctx, &antecipacaoID, `SELECT antecipacao_id FROM antecipacao_titulo WHERE titulo_id = $1 AND vigente`, tituloID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return r.Obter(ctx, antecipacaoID)
}

func (r *AntecipacaoRepository) Listar(ctx context.Context, filtro portsout.AntecipacaoFiltro) ([]*domainreceber.Antecipacao, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 6)
	argumentos := make([]any, 0, 8)
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
	if filtro.FornecedorID != nil {
		adicionar("fornecedor_id =", *filtro.FornecedorID)
	}
	if filtro.Status != "" {
		adicionar("status =", filtro.Status.String())
	}
	if filtro.De != nil {
		adicionar("operado_em >=", *filtro.De)
	}
	if filtro.Ate != nil {
		adicionar("operado_em <=", *filtro.Ate)
	}

	query := `SELECT ` + colunasAntecipacao + ` FROM antecipacao`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY operado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*antecipacaoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	antecipacoes := make([]*domainreceber.Antecipacao, 0, len(linhas))
	for _, linha := range linhas {
		antecipacoes = append(antecipacoes, paraDominioAntecipacao(linha))
	}
	if err := r.carregarTitulos(ctx, antecipacoes); err != nil {
		return nil, err
	}

	return antecipacoes, nil
}

func (r *AntecipacaoRepository) Estornar(ctx context.Context, estorno *portsout.EstornoDaAntecipacao) error {
	if estorno == nil || estorno.Antecipacao == nil {
		return domain.ErroValidacao("estorno de antecipação sem antecipação")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	antecipacao := estorno.Antecipacao
	versao, err := guardaDaAntecipacao.atualizar(ctx, tx, antecipacao.ID, `
		UPDATE antecipacao
		SET status = $1, estorno_motivo = $2, estornado_em = $3, atualizado_em = $4, versao = versao + 1
		WHERE id = $5 AND versao = $6 AND status = 'ativa'
		RETURNING versao
	`, antecipacao.Status.String(), antecipacao.EstornoMotivo, antecipacao.EstornadoEm, antecipacao.AtualizadoEm,
		antecipacao.ID, estorno.VersaoLida)
	if err != nil {
		return err
	}

	resultado, err := tx.ExecContext(ctx, `UPDATE antecipacao_titulo SET vigente = false WHERE antecipacao_id = $1 AND vigente`, antecipacao.ID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas, err := resultado.RowsAffected(); err != nil || int(linhas) != len(antecipacao.Titulos) {
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		return domain.ErroConflito("a antecipação tem título recebido ou recomprado e não pode ser estornada")
	}

	versoes, err := atualizarTitulosAlterados(ctx, tx, estorno.Titulos)
	if err != nil {
		return err
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	antecipacao.Versao = versao
	aplicarVersoes(estorno.Titulos, versoes)

	return nil
}

func (r *AntecipacaoRepository) Recomprar(ctx context.Context, gravacao *portsout.GravacaoRecompra) error {
	if gravacao == nil || gravacao.Recompra == nil || gravacao.Titulo.Titulo == nil || gravacao.ContaAPagar.Solicitacao == nil {
		return domain.ErroValidacao("recompra incompleta")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	recompra := gravacao.Recompra
	resultado, err := tx.ExecContext(ctx, `
		UPDATE antecipacao_titulo SET vigente = false
		WHERE antecipacao_id = $1 AND titulo_id = $2 AND vigente
	`, recompra.AntecipacaoID, recompra.TituloID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas, err := resultado.RowsAffected(); err != nil || linhas == 0 {
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		return domain.ErroConflito("o título não está numa antecipação vigente")
	}

	versoes, err := atualizarTitulosAlterados(ctx, tx, []portsout.TituloAlterado{gravacao.Titulo})
	if err != nil {
		return err
	}
	if err := gerarContaAPagar(ctx, tx, gravacao.ContaAPagar); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO recompra (
			id, titulo_id, antecipacao_id, recomprado_em, encargos_centavos, valor_centavos, vencimento,
			categoria_financeira_id, solicitacao_id, motivo, usuario_id, criado_em
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, recompra.ID, recompra.TituloID, recompra.AntecipacaoID, recompra.RecompradoEm, recompra.EncargosCentavos,
		recompra.ValorCentavos, recompra.Vencimento, recompra.CategoriaFinanceiraID, gravacao.ContaAPagar.Solicitacao.ID,
		recompra.Motivo, recompra.UsuarioID, recompra.CriadoEm); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	recompra.SolicitacaoID = gravacao.ContaAPagar.Solicitacao.ID
	aplicarVersoes([]portsout.TituloAlterado{gravacao.Titulo}, versoes)

	return nil
}

func (r *AntecipacaoRepository) carregarTitulos(ctx context.Context, antecipacoes []*domainreceber.Antecipacao) error {
	if len(antecipacoes) == 0 {
		return nil
	}

	porID := make(map[uuid.UUID]*domainreceber.Antecipacao, len(antecipacoes))
	ids := make([]uuid.UUID, 0, len(antecipacoes))
	for _, antecipacao := range antecipacoes {
		antecipacao.Titulos = []*domainreceber.AntecipacaoTitulo{}
		porID[antecipacao.ID] = antecipacao
		ids = append(ids, antecipacao.ID)
	}

	var linhas []*antecipacaoTituloLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT antecipacao_id, titulo_id, valor_centavos, status_anterior, vigente
		FROM antecipacao_titulo
		WHERE antecipacao_id = ANY($1::uuid[])
		ORDER BY antecipacao_id, titulo_id
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return err
	}

	for _, linha := range linhas {
		antecipacao := porID[linha.AntecipacaoID]
		antecipacao.Titulos = append(antecipacao.Titulos, &domainreceber.AntecipacaoTitulo{
			AntecipacaoID:  linha.AntecipacaoID,
			TituloID:       linha.TituloID,
			ValorCentavos:  linha.ValorCentavos,
			StatusAnterior: domainreceber.StatusDe(linha.StatusAnterior),
			Vigente:        linha.Vigente,
		})
	}

	return nil
}

func atualizarTitulosAlterados(ctx context.Context, tx *sqlx.Tx, titulos []portsout.TituloAlterado) ([]int, error) {
	versoes := make([]int, len(titulos))
	for indice, alterado := range titulos {
		versao, err := atualizarTitulo(ctx, tx, alterado.Titulo, alterado.VersaoLida, alterado.StatusAnterior)
		if err != nil {
			return nil, err
		}
		versoes[indice] = versao
		if alterado.Historico != nil {
			if err := inserirHistoricoDoTitulo(ctx, tx, alterado.Historico); err != nil {
				return nil, err
			}
		}
	}

	return versoes, nil
}

func aplicarVersoes(titulos []portsout.TituloAlterado, versoes []int) {
	for indice, alterado := range titulos {
		alterado.Titulo.Versao = versoes[indice]
	}
}

func gerarContaAPagar(ctx context.Context, tx *sqlx.Tx, conta portsout.ContaAPagarGerada) error {
	criada, err := gerarSolicitacaoDeOrigem(ctx, tx, conta.Solicitacao, conta.Historico)
	if err != nil {
		return err
	}
	if !criada {
		return domainreceber.ErroContaAPagarJaGerada()
	}

	return nil
}

func exigirFomentoAtivo(ctx context.Context, tx *sqlx.Tx, fornecedorID uuid.UUID) error {
	var ativo bool
	err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM fornecedor WHERE id = $1 FOR SHARE`, fornecedorID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroFomentoNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativo {
		return domainreceber.ErroFomentoInativo()
	}

	return nil
}

func exigirClienteQuePermiteAntecipacao(ctx context.Context, tx *sqlx.Tx, clienteID uuid.UUID) error {
	var cliente clienteParaAntecipacao
	err := tx.GetContext(ctx, &cliente, `SELECT nome, proibe_antecipacao FROM cliente WHERE id = $1 FOR SHARE`, clienteID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainreceber.ErroClienteNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if cliente.ProibeAntecipacao {
		return domainreceber.ErroClienteProibeAntecipacao(cliente.Nome)
	}

	return nil
}

func traduzirVinculoDaAntecipacao(err error) error {
	var erroPostgres *pq.Error
	if errors.As(err, &erroPostgres) && erroPostgres.Constraint == "uq_antecipacao_titulo_vigente" {
		return domainreceber.ErroTituloJaAntecipado()
	}

	return tratarErroDeGravacao(err)
}

func paraLinhaAntecipacao(antecipacao *domainreceber.Antecipacao) antecipacaoLinha {
	return antecipacaoLinha{
		ID:                   antecipacao.ID,
		EmpresaID:            antecipacao.EmpresaID,
		FornecedorID:         antecipacao.FornecedorID,
		Numero:               antecipacao.Numero,
		OperadoEm:            antecipacao.OperadoEm,
		TaxaBP:               antecipacao.TaxaBP,
		ValorBrutoCentavos:   antecipacao.ValorBrutoCentavos,
		ValorLiquidoCentavos: antecipacao.ValorLiquidoCentavos,
		ContaBancariaID:      antecipacao.ContaBancariaID,
		Status:               antecipacao.Status.String(),
		EstornoMotivo:        antecipacao.EstornoMotivo,
		EstornadoEm:          antecipacao.EstornadoEm,
		UsuarioID:            antecipacao.UsuarioID,
		Versao:               antecipacao.Versao,
		CriadoEm:             antecipacao.CriadoEm,
		AtualizadoEm:         antecipacao.AtualizadoEm,
	}
}

func paraDominioAntecipacao(linha *antecipacaoLinha) *domainreceber.Antecipacao {
	return &domainreceber.Antecipacao{
		ID:                   linha.ID,
		EmpresaID:            linha.EmpresaID,
		FornecedorID:         linha.FornecedorID,
		Numero:               linha.Numero,
		OperadoEm:            domainreceber.DataDe(linha.OperadoEm),
		TaxaBP:               linha.TaxaBP,
		ValorBrutoCentavos:   linha.ValorBrutoCentavos,
		ValorLiquidoCentavos: linha.ValorLiquidoCentavos,
		ContaBancariaID:      linha.ContaBancariaID,
		Status:               domainreceber.StatusAntecipacaoDe(linha.Status),
		EstornoMotivo:        linha.EstornoMotivo,
		EstornadoEm:          linha.EstornadoEm,
		UsuarioID:            linha.UsuarioID,
		Versao:               linha.Versao,
		CriadoEm:             linha.CriadoEm,
		AtualizadoEm:         linha.AtualizadoEm,
	}
}
