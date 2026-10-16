package postgres

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portspedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type recebimentoLinha struct {
	ID                     uuid.UUID  `db:"id"`
	TituloID               uuid.UUID  `db:"titulo_id"`
	Tipo                   string     `db:"tipo"`
	ContaBancariaID        *uuid.UUID `db:"conta_bancaria_id"`
	RecebidoEm             time.Time  `db:"recebido_em"`
	ValorCentavos          int64      `db:"valor_centavos"`
	JurosCentavos          int64      `db:"juros_centavos"`
	DescontoCentavos       int64      `db:"desconto_centavos"`
	StatusTituloAnterior   string     `db:"status_titulo_anterior"`
	RegistrouEntradaPedido bool       `db:"registrou_entrada_pedido"`
	Estornado              bool       `db:"estornado"`
	UsuarioID              uuid.UUID  `db:"usuario_id"`
	CriadoEm               time.Time  `db:"criado_em"`
	EstornoID              *uuid.UUID `db:"estorno_id"`
	EstornoMotivo          *string    `db:"estorno_motivo"`
	EstornadoEm            *time.Time `db:"estornado_em"`
	EstornoUsuarioID       *uuid.UUID `db:"estorno_usuario_id"`
	EstornoCriadoEm        *time.Time `db:"estorno_criado_em"`
}

func (r *TituloReceberRepository) Receber(ctx context.Context, baixa *portsout.Baixa) error {
	if baixa == nil || baixa.Titulo == nil || baixa.Recebimento == nil || baixa.Historico == nil {
		return domain.ErroValidacao("baixa de título incompleta")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	titulo := baixa.Titulo
	if err := exigirContaDeCobranca(ctx, tx, baixa.Recebimento.ContaBancariaID, titulo.EmpresaID); err != nil {
		return err
	}

	versao, err := atualizarTitulo(ctx, tx, titulo, baixa.VersaoLida, baixa.StatusAnterior)
	if err != nil {
		return err
	}
	if err := inserirRecebimento(ctx, tx, baixa.Recebimento); err != nil {
		return err
	}
	if err := inserirHistoricoDoTitulo(ctx, tx, baixa.Historico); err != nil {
		return err
	}

	versaoDoPedido := 0
	if entrada := baixa.Entrada; entrada != nil {
		if versaoDoPedido, err = atualizarPedido(ctx, tx, &portspedido.Gravacao{
			Pedido:         entrada.Pedido,
			VersaoLida:     entrada.VersaoLida,
			StatusAnterior: entrada.StatusAnterior,
		}); err != nil {
			return err
		}
		if err := inserirHistoricoDoPedido(ctx, tx, entrada.Historico); err != nil {
			return err
		}
	}

	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	titulo.Versao = versao
	if baixa.Entrada != nil {
		baixa.Entrada.Pedido.Versao = versaoDoPedido
	}

	return nil
}

func (r *TituloReceberRepository) EstornarRecebimento(ctx context.Context, estorno *portsout.EstornoDaBaixa) error {
	if estorno == nil || estorno.Titulo == nil || estorno.Recebimento == nil || estorno.Recebimento.Estorno == nil || estorno.Historico == nil {
		return domain.ErroValidacao("estorno de recebimento incompleto")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	resultado, err := tx.ExecContext(ctx, `
		UPDATE titulo_recebimento SET estornado = true
		WHERE id = $1 AND titulo_id = $2 AND NOT estornado AND NOT registrou_entrada_pedido
	`, estorno.Recebimento.ID, estorno.Titulo.ID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domain.ErroConflito("o recebimento já foi estornado")
	}

	dados := estorno.Recebimento.Estorno
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO titulo_recebimento_estorno (id, recebimento_id, motivo, estornado_em, usuario_id, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, dados.ID, dados.RecebimentoID, dados.Motivo, dados.EstornadoEm, dados.UsuarioID, dados.CriadoEm); err != nil {
		return tratarErroDeGravacao(err)
	}

	versao, err := atualizarTitulo(ctx, tx, estorno.Titulo, estorno.VersaoLida, estorno.StatusAnterior)
	if err != nil {
		return err
	}
	if err := inserirHistoricoDoTitulo(ctx, tx, estorno.Historico); err != nil {
		return err
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	estorno.Titulo.Versao = versao

	return nil
}

func (r *TituloReceberRepository) ListarRecebimentos(ctx context.Context, tituloID uuid.UUID) ([]*domainreceber.Recebimento, error) {
	var linhas []*recebimentoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT r.id, r.titulo_id, r.tipo, r.conta_bancaria_id, r.recebido_em, r.valor_centavos, r.juros_centavos,
		       r.desconto_centavos, r.status_titulo_anterior, r.registrou_entrada_pedido, r.estornado, r.usuario_id,
		       r.criado_em, e.id AS estorno_id, e.motivo AS estorno_motivo, e.estornado_em,
		       e.usuario_id AS estorno_usuario_id, e.criado_em AS estorno_criado_em
		FROM titulo_recebimento r
		LEFT JOIN titulo_recebimento_estorno e ON e.recebimento_id = r.id
		WHERE r.titulo_id = $1
		ORDER BY r.criado_em ASC, r.id ASC
	`, tituloID); err != nil {
		return nil, err
	}

	recebimentos := make([]*domainreceber.Recebimento, 0, len(linhas))
	for _, linha := range linhas {
		recebimento := &domainreceber.Recebimento{
			ID:                     linha.ID,
			TituloID:               linha.TituloID,
			Tipo:                   domainreceber.TipoRecebimentoDe(linha.Tipo),
			ContaBancariaID:        linha.ContaBancariaID,
			RecebidoEm:             domainreceber.DataDe(linha.RecebidoEm),
			ValorCentavos:          linha.ValorCentavos,
			JurosCentavos:          linha.JurosCentavos,
			DescontoCentavos:       linha.DescontoCentavos,
			StatusTituloAnterior:   domainreceber.StatusDe(linha.StatusTituloAnterior),
			RegistrouEntradaPedido: linha.RegistrouEntradaPedido,
			Estornado:              linha.Estornado,
			UsuarioID:              linha.UsuarioID,
			CriadoEm:               linha.CriadoEm,
		}
		if linha.EstornoID != nil {
			recebimento.Estorno = &domainreceber.EstornoRecebimento{
				ID:            *linha.EstornoID,
				RecebimentoID: linha.ID,
				Motivo:        *linha.EstornoMotivo,
				EstornadoEm:   domainreceber.DataDe(*linha.EstornadoEm),
				UsuarioID:     *linha.EstornoUsuarioID,
				CriadoEm:      *linha.EstornoCriadoEm,
			}
		}
		recebimentos = append(recebimentos, recebimento)
	}

	return recebimentos, nil
}

func inserirRecebimento(ctx context.Context, tx *sqlx.Tx, recebimento *domainreceber.Recebimento) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO titulo_recebimento (
			id, titulo_id, tipo, conta_bancaria_id, recebido_em, valor_centavos, juros_centavos, desconto_centavos,
			status_titulo_anterior, registrou_entrada_pedido, estornado, usuario_id, criado_em
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, recebimento.ID, recebimento.TituloID, recebimento.Tipo.String(), recebimento.ContaBancariaID, recebimento.RecebidoEm,
		recebimento.ValorCentavos, recebimento.JurosCentavos, recebimento.DescontoCentavos,
		recebimento.StatusTituloAnterior.String(), recebimento.RegistrouEntradaPedido, recebimento.Estornado,
		recebimento.UsuarioID, recebimento.CriadoEm); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}
