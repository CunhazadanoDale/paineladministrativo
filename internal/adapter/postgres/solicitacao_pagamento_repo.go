package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type pagamentoLinha struct {
	ID                   uuid.UUID  `db:"id"`
	SolicitacaoID        uuid.UUID  `db:"solicitacao_id"`
	ContaBancariaID      *uuid.UUID `db:"conta_bancaria_id"`
	UsuarioID            *uuid.UUID `db:"usuario_id"`
	ComprovanteArquivoID *uuid.UUID `db:"comprovante_arquivo_id"`
	ValorCentavos        int64      `db:"valor"`
	PagoEm               time.Time  `db:"pago_em"`
	Estornado            bool       `db:"estornado"`
	CriadoEm             time.Time  `db:"criado_em"`
	EstornoID            *uuid.UUID `db:"estorno_id"`
	EstornoMotivo        *string    `db:"estorno_motivo"`
	EstornadoEm          *time.Time `db:"estornado_em"`
	EstornoUsuarioID     *uuid.UUID `db:"estorno_usuario_id"`
	EstornoCriadoEm      *time.Time `db:"estorno_criado_em"`
}

const consultaPagamentos = `
	SELECT p.id, p.solicitacao_id, p.conta_bancaria_id, p.usuario_id, p.comprovante_arquivo_id, p.valor,
	       p.pago_em, p.estornado, p.criado_em,
	       e.id AS estorno_id, e.motivo AS estorno_motivo, e.estornado_em, e.usuario_id AS estorno_usuario_id,
	       e.criado_em AS estorno_criado_em
	FROM pagamento p
	LEFT JOIN pagamento_estorno e ON e.pagamento_id = p.id
`

type contaParaPagamento struct {
	EmpresaID uuid.UUID `db:"empresa_id"`
	Ativa     bool      `db:"ativa"`
}

func (s *SolicitacaoRepository) CriarPagamento(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirContaParaPagamento(ctx, tx, pagamento.ContaBancariaID, solicitacao.EmpresaID); err != nil {
		return err
	}

	atualizado, err := atualizarStatusComGuarda(ctx, tx, solicitacao, historico.DeStatus)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !atualizado {
		return domainsolicitacao.ErroSolicitacaoAlterada()
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO pagamento (
			id, solicitacao_id, conta_bancaria_id, usuario_id, comprovante_arquivo_id, valor, pago_em, estornado, criado_em
		) VALUES (
			:id, :solicitacao_id, :conta_bancaria_id, :usuario_id, :comprovante_arquivo_id, :valor, :pago_em, :estornado, :criado_em
		)
	`, paraLinhaPagamento(pagamento)); err != nil {
		return tratarErroDeGravacao(err)
	}

	if pagamento.ComprovanteArquivoID != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO solicitacao_arquivo (solicitacao_id, arquivo_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, pagamento.SolicitacaoID, *pagamento.ComprovanteArquivoID); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return tratarErroDeGravacao(err)
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (s *SolicitacaoRepository) Estornar(
	ctx context.Context,
	solicitacao *domainsolicitacao.Solicitacao,
	versaoLida int,
	pagamento *domainsolicitacao.Pagamento,
	historico *domainsolicitacao.Historico,
) error {
	if pagamento.Estorno == nil {
		return domain.ErroValidacao("estorno do pagamento não informado")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := s.guarda.atualizar(ctx, tx, solicitacao.ID, `
		UPDATE solicitacao
		SET status = 'aprovado',
		    atualizado_em = $3,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2 AND status = 'pago'
		RETURNING versao
	`, solicitacao.ID, versaoLida, solicitacao.AtualizadoEm); err != nil {
		return err
	}

	resultado, err := tx.ExecContext(ctx, `
		UPDATE pagamento SET estornado = true WHERE id = $1 AND solicitacao_id = $2 AND NOT estornado
	`, pagamento.ID, solicitacao.ID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domain.ErroConflito("pagamento já estornado")
	}

	estorno := pagamento.Estorno
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pagamento_estorno (id, pagamento_id, motivo, estornado_em, usuario_id, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, estorno.ID, pagamento.ID, estorno.Motivo.Texto(), estorno.EstornadoEm, estorno.UsuarioID, estorno.CriadoEm); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return tratarErroDeGravacao(err)
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (s *SolicitacaoRepository) ObterPagamento(ctx context.Context, solicitacaoID uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	var linha pagamentoLinha
	err := s.db.GetContext(ctx, &linha, consultaPagamentos+` WHERE p.solicitacao_id = $1 AND NOT p.estornado`, solicitacaoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioPagamento(&linha), nil
}

func (s *SolicitacaoRepository) ListarPagamentos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Pagamento, error) {
	var linhas []*pagamentoLinha
	if err := s.db.SelectContext(ctx, &linhas, consultaPagamentos+` WHERE p.solicitacao_id = $1 ORDER BY p.criado_em, p.id`, solicitacaoID); err != nil {
		return nil, err
	}

	pagamentos := make([]*domainsolicitacao.Pagamento, 0, len(linhas))
	for _, linha := range linhas {
		pagamentos = append(pagamentos, paraDominioPagamento(linha))
	}

	return pagamentos, nil
}

func exigirContaParaPagamento(ctx context.Context, tx *sqlx.Tx, contaBancariaID *uuid.UUID, empresaID uuid.UUID) error {
	if contaBancariaID == nil {
		return domainsolicitacao.ErroContaBancariaObrigatoria()
	}

	var conta contaParaPagamento
	err := tx.GetContext(ctx, &conta, `SELECT empresa_id, ativa FROM conta_bancaria WHERE id = $1 FOR SHARE`, *contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroContaBancariaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if conta.EmpresaID != empresaID {
		return domainsolicitacao.ErroContaBancariaDeOutraEmpresa()
	}
	if !conta.Ativa {
		return domainsolicitacao.ErroContaBancariaInativa()
	}

	var empresaAtiva bool
	if err := tx.GetContext(ctx, &empresaAtiva, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID); err != nil {
		return tratarErroDeGravacao(err)
	}
	if !empresaAtiva {
		return domainsolicitacao.ErroEmpresaInativaNoPagamento()
	}

	return nil
}

func paraLinhaPagamento(pagamento *domainsolicitacao.Pagamento) pagamentoLinha {
	return pagamentoLinha{
		ID:                   pagamento.ID,
		SolicitacaoID:        pagamento.SolicitacaoID,
		ContaBancariaID:      pagamento.ContaBancariaID,
		UsuarioID:            pagamento.UsuarioID,
		ComprovanteArquivoID: pagamento.ComprovanteArquivoID,
		ValorCentavos:        pagamento.Valor.Centavos(),
		PagoEm:               pagamento.PagoEm,
		Estornado:            pagamento.Estornado,
		CriadoEm:             pagamento.CriadoEm,
	}
}

func paraDominioPagamento(linha *pagamentoLinha) *domainsolicitacao.Pagamento {
	pagamento := &domainsolicitacao.Pagamento{
		ID:                   linha.ID,
		SolicitacaoID:        linha.SolicitacaoID,
		ContaBancariaID:      linha.ContaBancariaID,
		UsuarioID:            linha.UsuarioID,
		ComprovanteArquivoID: linha.ComprovanteArquivoID,
		Valor:                domainsolicitacao.ValorDe(linha.ValorCentavos),
		PagoEm:               linha.PagoEm,
		Estornado:            linha.Estornado,
		CriadoEm:             linha.CriadoEm,
	}

	if linha.EstornoID != nil {
		pagamento.Estorno = &domainsolicitacao.Estorno{
			ID:          *linha.EstornoID,
			PagamentoID: linha.ID,
			Motivo:      domainsolicitacao.MotivoEstornoDe(*linha.EstornoMotivo),
			EstornadoEm: *linha.EstornadoEm,
			UsuarioID:   *linha.EstornoUsuarioID,
			CriadoEm:    *linha.EstornoCriadoEm,
		}
	}

	return pagamento
}
