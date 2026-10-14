package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.EntregaRepository = (*EntregaRepository)(nil)

type EntregaRepository struct {
	db *sqlx.DB
}

func NewEntregaRepository(db *sqlx.DB) *EntregaRepository {
	return &EntregaRepository{db: db}
}

type entregaLinha struct {
	ID             uuid.UUID  `db:"id"`
	PedidoID       uuid.UUID  `db:"pedido_id"`
	Status         string     `db:"status"`
	Transportadora *string    `db:"transportadora"`
	FreteTipo      string     `db:"frete_tipo"`
	FreteCentavos  *int64     `db:"frete_centavos"`
	Previsao       *time.Time `db:"previsao"`
	SaiuEm         *time.Time `db:"saiu_em"`
	EntregueEm     *time.Time `db:"entregue_em"`
	UsuarioID      uuid.UUID  `db:"usuario_id"`
	Versao         int        `db:"versao"`
	CriadoEm       time.Time  `db:"criado_em"`
	AtualizadoEm   time.Time  `db:"atualizado_em"`
}

type entregaItemLinha struct {
	EntregaID    uuid.UUID  `db:"entrega_id"`
	PedidoItemID uuid.UUID  `db:"pedido_item_id"`
	LocalID      uuid.UUID  `db:"local_id"`
	Quantidade   int        `db:"quantidade"`
	MovimentoID  *uuid.UUID `db:"movimento_id"`
}

const colunasEntrega = `
	id, pedido_id, status, transportadora, frete_tipo, frete_centavos, previsao, saiu_em,
	entregue_em, usuario_id, versao, criado_em, atualizado_em
`

const colunasEntregaItem = `entrega_id, pedido_item_id, local_id, quantidade, movimento_id`

func (r *EntregaRepository) Obter(ctx context.Context, id uuid.UUID) (*domainpedido.Entrega, error) {
	var linha entregaLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasEntrega+` FROM entrega WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	entregas, err := r.comItens(ctx, []*entregaLinha{&linha})
	if err != nil {
		return nil, err
	}

	return entregas[0], nil
}

func (r *EntregaRepository) ListarDoPedido(ctx context.Context, pedidoID uuid.UUID) ([]*domainpedido.Entrega, error) {
	var linhas []*entregaLinha
	if err := r.db.SelectContext(ctx, &linhas,
		`SELECT `+colunasEntrega+` FROM entrega WHERE pedido_id = $1 ORDER BY criado_em ASC, id ASC`, pedidoID,
	); err != nil {
		return nil, err
	}

	return r.comItens(ctx, linhas)
}

func (r *EntregaRepository) Listar(ctx context.Context, filtro portsout.EntregaFiltro) ([]*domainpedido.Entrega, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 4)
	argumentos := make([]any, 0, 6)

	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes = append(condicoes, "pedido_id IN (SELECT id FROM pedido WHERE empresa_id = $"+strconv.Itoa(len(argumentos))+")")
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes,
			"pedido_id IN (SELECT id FROM pedido WHERE empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[]))")
	}
	if filtro.Status != "" {
		argumentos = append(argumentos, filtro.Status.String())
		condicoes = append(condicoes, condicao("status", len(argumentos)))
	}
	if filtro.PrevisaoAte != nil {
		argumentos = append(argumentos, domainpedido.DataDe(*filtro.PrevisaoAte))
		condicoes = append(condicoes, "previsao <= $"+strconv.Itoa(len(argumentos)))
	}

	query := `SELECT ` + colunasEntrega + ` FROM entrega`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY previsao ASC NULLS LAST, criado_em ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*entregaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	return r.comItens(ctx, linhas)
}

func (r *EntregaRepository) comItens(ctx context.Context, linhas []*entregaLinha) ([]*domainpedido.Entrega, error) {
	entregas := make([]*domainpedido.Entrega, 0, len(linhas))
	if len(linhas) == 0 {
		return entregas, nil
	}

	ids := make([]uuid.UUID, 0, len(linhas))
	porID := make(map[uuid.UUID]*domainpedido.Entrega, len(linhas))
	for _, linha := range linhas {
		entrega := paraDominioEntrega(linha)
		entregas = append(entregas, entrega)
		porID[entrega.ID] = entrega
		ids = append(ids, entrega.ID)
	}

	var itens []*entregaItemLinha
	if err := r.db.SelectContext(ctx, &itens, `
		SELECT ei.entrega_id, ei.pedido_item_id, ei.local_id, ei.quantidade, ei.movimento_id
		FROM entrega_item ei
		JOIN pedido_item pi ON pi.id = ei.pedido_item_id
		WHERE ei.entrega_id = ANY($1::uuid[])
		ORDER BY ei.entrega_id, pi.id ASC
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return nil, err
	}

	for _, item := range itens {
		entrega := porID[item.EntregaID]
		entrega.Itens = append(entrega.Itens, &domainpedido.EntregaItem{
			EntregaID:    item.EntregaID,
			PedidoItemID: item.PedidoItemID,
			LocalID:      item.LocalID,
			Quantidade:   item.Quantidade,
			MovimentoID:  item.MovimentoID,
		})
	}

	return entregas, nil
}

func inserirEntrega(ctx context.Context, tx *sqlx.Tx, entrega *domainpedido.Entrega) error {
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO entrega (`+colunasEntrega+`)
		VALUES (
			:id, :pedido_id, :status, :transportadora, :frete_tipo, :frete_centavos, :previsao, :saiu_em,
			:entregue_em, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaEntrega(entrega)); err != nil {
		return tratarErroDeGravacao(err)
	}

	for _, item := range entrega.Itens {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO entrega_item (`+colunasEntregaItem+`)
			VALUES ($1, $2, $3, $4, $5)
		`, entrega.ID, item.PedidoItemID, item.LocalID, item.Quantidade, item.MovimentoID); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	return nil
}

func atualizarEntrega(ctx context.Context, tx *sqlx.Tx, alterada portsout.EntregaAlterada) (int, error) {
	entrega := alterada.Entrega

	return guardaDaEntrega.atualizar(ctx, tx, entrega.ID, `
		UPDATE entrega
		SET status = $1,
		    saiu_em = $2,
		    entregue_em = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE id = $5
		  AND pedido_id = $6
		  AND versao = $7
		  AND status = $8
		RETURNING versao
	`,
		entrega.Status.String(), entrega.SaiuEm, entrega.EntregueEm, entrega.AtualizadoEm,
		entrega.ID, entrega.PedidoID, alterada.VersaoLida, alterada.StatusAnterior.String(),
	)
}

func vincularMovimentos(ctx context.Context, tx *sqlx.Tx, entrega *domainpedido.Entrega) error {
	for _, item := range entrega.Itens {
		if item.MovimentoID == nil {
			continue
		}

		resultado, err := tx.ExecContext(ctx, `
			UPDATE entrega_item
			SET movimento_id = $1
			WHERE entrega_id = $2
			  AND pedido_item_id = $3
			  AND (movimento_id IS NULL OR movimento_id = $1)
		`, *item.MovimentoID, entrega.ID, item.PedidoItemID)
		if err != nil {
			return tratarErroDeGravacao(err)
		}

		linhas, err := resultado.RowsAffected()
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		if linhas == 0 {
			return domain.ErroConflito("item da entrega já baixado por outra operação, recarregue e tente novamente")
		}
	}

	return nil
}

func paraLinhaEntrega(entrega *domainpedido.Entrega) entregaLinha {
	return entregaLinha{
		ID:             entrega.ID,
		PedidoID:       entrega.PedidoID,
		Status:         entrega.Status.String(),
		Transportadora: entrega.Transportadora,
		FreteTipo:      entrega.Frete.Tipo().String(),
		FreteCentavos:  entrega.Frete.Centavos(),
		Previsao:       entrega.Previsao,
		SaiuEm:         entrega.SaiuEm,
		EntregueEm:     entrega.EntregueEm,
		UsuarioID:      entrega.UsuarioID,
		Versao:         entrega.Versao,
		CriadoEm:       entrega.CriadoEm,
		AtualizadoEm:   entrega.AtualizadoEm,
	}
}

func paraDominioEntrega(linha *entregaLinha) *domainpedido.Entrega {
	return &domainpedido.Entrega{
		ID:             linha.ID,
		PedidoID:       linha.PedidoID,
		Status:         domainpedido.StatusEntregaDe(linha.Status),
		Transportadora: linha.Transportadora,
		Frete:          domainpedido.FreteDe(domainpedido.TipoFreteDe(linha.FreteTipo), linha.FreteCentavos),
		Previsao:       dataDoBanco(linha.Previsao),
		SaiuEm:         dataDoBanco(linha.SaiuEm),
		EntregueEm:     dataDoBanco(linha.EntregueEm),
		UsuarioID:      linha.UsuarioID,
		Versao:         linha.Versao,
		CriadoEm:       linha.CriadoEm,
		AtualizadoEm:   linha.AtualizadoEm,
		Itens:          make([]*domainpedido.EntregaItem, 0),
	}
}
