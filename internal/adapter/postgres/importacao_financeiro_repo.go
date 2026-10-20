package postgres

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type importacaoParcelaLinha struct {
	ID                   uuid.UUID  `db:"id"`
	ImportacaoID         uuid.UUID  `db:"importacao_id"`
	Descricao            string     `db:"descricao"`
	ValorMoedaCentesimos int64      `db:"valor_moeda_centesimos"`
	VencimentoPrevisto   time.Time  `db:"vencimento_previsto"`
	Status               string     `db:"status"`
	CambioMicro          *int64     `db:"cambio_micro"`
	ValorCentavos        *int64     `db:"valor_centavos"`
	Vencimento           *time.Time `db:"vencimento"`
	SolicitacaoID        *uuid.UUID `db:"solicitacao_id"`
	StatusContaAPagar    *string    `db:"status_conta_a_pagar"`
	UsuarioID            uuid.UUID  `db:"usuario_id"`
	CriadoEm             time.Time  `db:"criado_em"`
	AtualizadoEm         time.Time  `db:"atualizado_em"`
}

type importacaoCustoLinha struct {
	ID                    uuid.UUID  `db:"id"`
	ImportacaoID          uuid.UUID  `db:"importacao_id"`
	Tipo                  string     `db:"tipo"`
	Descricao             string     `db:"descricao"`
	FornecedorID          uuid.UUID  `db:"fornecedor_id"`
	ValorCentavos         int64      `db:"valor_centavos"`
	Vencimento            time.Time  `db:"vencimento"`
	CompoeCusto           bool       `db:"compoe_custo"`
	Posterior             bool       `db:"posterior"`
	CategoriaFinanceiraID uuid.UUID  `db:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID  `db:"solicitacao_id"`
	StatusContaAPagar     string     `db:"status_conta_a_pagar"`
	CanceladoEm           *time.Time `db:"cancelado_em"`
	CanceladoPor          *uuid.UUID `db:"cancelado_por"`
	UsuarioID             uuid.UUID  `db:"usuario_id"`
	CriadoEm              time.Time  `db:"criado_em"`
}

func carregarFinanceiro(ctx context.Context, tx *sqlx.Tx, importacao *domainimportacao.Importacao) error {
	var parcelas []*importacaoParcelaLinha
	if err := tx.SelectContext(ctx, &parcelas, `
		SELECT p.id, p.importacao_id, p.descricao, p.valor_moeda_centesimos, p.vencimento_previsto, p.status,
		       p.cambio_micro, p.valor_centavos, p.vencimento, p.solicitacao_id, s.status AS status_conta_a_pagar,
		       p.usuario_id, p.criado_em, p.atualizado_em
		FROM importacao_parcela p
		LEFT JOIN solicitacao s ON s.id = p.solicitacao_id
		WHERE p.importacao_id = $1
		ORDER BY p.criado_em ASC, p.id ASC
	`, importacao.ID); err != nil {
		return err
	}

	var custos []*importacaoCustoLinha
	if err := tx.SelectContext(ctx, &custos, `
		SELECT c.id, c.importacao_id, c.tipo, c.descricao, c.fornecedor_id, c.valor_centavos, c.vencimento,
		       c.compoe_custo, c.posterior, c.categoria_financeira_id, c.solicitacao_id, s.status AS status_conta_a_pagar,
		       c.cancelado_em, c.cancelado_por, c.usuario_id, c.criado_em
		FROM importacao_custo c
		JOIN solicitacao s ON s.id = c.solicitacao_id
		WHERE c.importacao_id = $1
		ORDER BY c.criado_em ASC, c.id ASC
	`, importacao.ID); err != nil {
		return err
	}

	importacao.Parcelas = make([]*domainimportacao.Parcela, 0, len(parcelas))
	for _, linha := range parcelas {
		importacao.Parcelas = append(importacao.Parcelas, paraDominioParcela(linha))
	}
	importacao.Custos = make([]*domainimportacao.Custo, 0, len(custos))
	for _, linha := range custos {
		status := domainsolicitacao.StatusDe(linha.StatusContaAPagar)
		importacao.Custos = append(importacao.Custos, &domainimportacao.Custo{
			ID:                    linha.ID,
			ImportacaoID:          linha.ImportacaoID,
			Tipo:                  domainimportacao.TipoCustoDe(linha.Tipo),
			Descricao:             linha.Descricao,
			FornecedorID:          linha.FornecedorID,
			ValorCentavos:         linha.ValorCentavos,
			Vencimento:            domainimportacao.DataDe(linha.Vencimento),
			CompoeCusto:           linha.CompoeCusto,
			Posterior:             linha.Posterior,
			CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
			SolicitacaoID:         linha.SolicitacaoID,
			StatusContaAPagar:     &status,
			CanceladoEm:           linha.CanceladoEm,
			CanceladoPor:          linha.CanceladoPor,
			UsuarioID:             linha.UsuarioID,
			CriadoEm:              linha.CriadoEm,
		})
	}

	return nil
}

func gravarFinanceiroDaImportacao(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) error {
	if err := exigirContasEncerradas(ctx, tx, gravacao.ContasExigidasEncerradas); err != nil {
		return err
	}
	for _, conta := range gravacao.ContasAPagar {
		criada, err := gerarSolicitacaoDeOrigem(ctx, tx, conta.Solicitacao, conta.Historico)
		if err != nil {
			return err
		}
		if !criada {
			return domain.ErroConflito("a conta a pagar deste lançamento já foi gerada; recarregue a importação")
		}
	}

	importacaoID := gravacao.Importacao.ID
	for _, id := range gravacao.ParcelasRemovidas {
		resultado, err := tx.ExecContext(ctx,
			`DELETE FROM importacao_parcela WHERE id = $1 AND importacao_id = $2 AND status = 'aberta'`, id, importacaoID)
		if err := exigirUmaLinha(resultado, err, "parcela não encontrada ou com câmbio fechado"); err != nil {
			return err
		}
	}
	for _, parcela := range gravacao.ParcelasNovas {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO importacao_parcela (
				id, importacao_id, descricao, valor_moeda_centesimos, vencimento_previsto, status, cambio_micro,
				valor_centavos, vencimento, solicitacao_id, usuario_id, criado_em, atualizado_em
			) VALUES (
				:id, :importacao_id, :descricao, :valor_moeda_centesimos, :vencimento_previsto, :status, :cambio_micro,
				:valor_centavos, :vencimento, :solicitacao_id, :usuario_id, :criado_em, :atualizado_em
			)
		`, paraLinhaParcela(parcela)); err != nil {
			return tratarErroDeGravacao(err)
		}
	}
	for _, parcela := range gravacao.ParcelasAlteradas {
		linha := paraLinhaParcela(parcela)
		resultado, err := tx.ExecContext(ctx, `
			UPDATE importacao_parcela
			SET descricao = $1,
			    valor_moeda_centesimos = $2,
			    vencimento_previsto = $3,
			    status = $4,
			    cambio_micro = $5,
			    valor_centavos = $6,
			    vencimento = $7,
			    solicitacao_id = $8,
			    atualizado_em = $9
			WHERE id = $10
			  AND importacao_id = $11
			  AND status <> 'cancelada'
		`, linha.Descricao, linha.ValorMoedaCentesimos, linha.VencimentoPrevisto, linha.Status, linha.CambioMicro,
			linha.ValorCentavos, linha.Vencimento, linha.SolicitacaoID, linha.AtualizadoEm, linha.ID, importacaoID)
		if err := exigirUmaLinha(resultado, err, "parcela não encontrada ou cancelada"); err != nil {
			return err
		}
	}
	for _, custo := range gravacao.CustosNovos {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO importacao_custo (
				id, importacao_id, tipo, descricao, fornecedor_id, valor_centavos, vencimento, compoe_custo, posterior,
				categoria_financeira_id, solicitacao_id, cancelado_em, cancelado_por, usuario_id, criado_em
			) VALUES (
				:id, :importacao_id, :tipo, :descricao, :fornecedor_id, :valor_centavos, :vencimento, :compoe_custo, :posterior,
				:categoria_financeira_id, :solicitacao_id, :cancelado_em, :cancelado_por, :usuario_id, :criado_em
			)
		`, paraLinhaCusto(custo)); err != nil {
			return tratarErroDeGravacao(err)
		}
	}
	for _, custo := range gravacao.CustosCancelados {
		resultado, err := tx.ExecContext(ctx, `
			UPDATE importacao_custo
			SET cancelado_em = $1, cancelado_por = $2
			WHERE id = $3 AND importacao_id = $4 AND cancelado_em IS NULL
		`, custo.CanceladoEm, custo.CanceladoPor, custo.ID, importacaoID)
		if err := exigirUmaLinha(resultado, err, "custo não encontrado ou já cancelado"); err != nil {
			return err
		}
	}

	return nil
}

func exigirContasEncerradas(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	var abertas int
	if err := tx.GetContext(ctx, &abertas, `
		WITH travadas AS (
			SELECT status
			FROM solicitacao
			WHERE id = ANY($1::uuid[])
			ORDER BY id
			FOR SHARE
		)
		SELECT $2 - count(*) FILTER (WHERE status IN ('cancelado', 'rejeitado'))
		FROM travadas
	`, pq.Array(textosDeIDs(ids)), len(ids)); err != nil {
		return tratarErroDeGravacao(err)
	}
	if abertas > 0 {
		return domainimportacao.ErroContasAPagarAbertas()
	}

	return nil
}

func exigirUmaLinha(resultado interface{ RowsAffected() (int64, error) }, err error, mensagem string) error {
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas != 1 {
		return domain.ErroConflito(mensagem + "; recarregue a importação")
	}

	return nil
}

func paraLinhaParcela(parcela *domainimportacao.Parcela) importacaoParcelaLinha {
	linha := importacaoParcelaLinha{
		ID:                   parcela.ID,
		ImportacaoID:         parcela.ImportacaoID,
		Descricao:            parcela.Descricao,
		ValorMoedaCentesimos: parcela.Valor.Centesimos(),
		VencimentoPrevisto:   parcela.VencimentoPrevisto,
		Status:               parcela.Status.String(),
		ValorCentavos:        parcela.ValorCentavos,
		Vencimento:           parcela.Vencimento,
		SolicitacaoID:        parcela.SolicitacaoID,
		UsuarioID:            parcela.UsuarioID,
		CriadoEm:             parcela.CriadoEm,
		AtualizadoEm:         parcela.AtualizadoEm,
	}
	if parcela.Cambio != nil {
		micro := parcela.Cambio.Micro()
		linha.CambioMicro = &micro
	}

	return linha
}

func paraDominioParcela(linha *importacaoParcelaLinha) *domainimportacao.Parcela {
	parcela := &domainimportacao.Parcela{
		ID:                 linha.ID,
		ImportacaoID:       linha.ImportacaoID,
		Descricao:          linha.Descricao,
		Valor:              domainimportacao.ValorMoedaDe(linha.ValorMoedaCentesimos),
		VencimentoPrevisto: domainimportacao.DataDe(linha.VencimentoPrevisto),
		Status:             domainimportacao.StatusParcelaDe(linha.Status),
		ValorCentavos:      linha.ValorCentavos,
		Vencimento:         dataDaImportacao(linha.Vencimento),
		SolicitacaoID:      linha.SolicitacaoID,
		UsuarioID:          linha.UsuarioID,
		CriadoEm:           linha.CriadoEm,
		AtualizadoEm:       linha.AtualizadoEm,
	}
	if linha.CambioMicro != nil {
		cambio := domainimportacao.CambioDe(*linha.CambioMicro)
		parcela.Cambio = &cambio
	}
	if linha.StatusContaAPagar != nil {
		status := domainsolicitacao.StatusDe(*linha.StatusContaAPagar)
		parcela.StatusContaAPagar = &status
	}

	return parcela
}

func paraLinhaCusto(custo *domainimportacao.Custo) importacaoCustoLinha {
	return importacaoCustoLinha{
		ID:                    custo.ID,
		ImportacaoID:          custo.ImportacaoID,
		Tipo:                  custo.Tipo.String(),
		Descricao:             custo.Descricao,
		FornecedorID:          custo.FornecedorID,
		ValorCentavos:         custo.ValorCentavos,
		Vencimento:            custo.Vencimento,
		CompoeCusto:           custo.CompoeCusto,
		Posterior:             custo.Posterior,
		CategoriaFinanceiraID: custo.CategoriaFinanceiraID,
		SolicitacaoID:         custo.SolicitacaoID,
		CanceladoEm:           custo.CanceladoEm,
		CanceladoPor:          custo.CanceladoPor,
		UsuarioID:             custo.UsuarioID,
		CriadoEm:              custo.CriadoEm,
	}
}
