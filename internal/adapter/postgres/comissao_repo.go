package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ComissaoRepository = (*ComissaoRepository)(nil)

type ComissaoRepository struct {
	db *sqlx.DB
}

func NewComissaoRepository(db *sqlx.DB) *ComissaoRepository {
	return &ComissaoRepository{db: db}
}

type percentualPedidoLinha struct {
	PedidoID        uuid.UUID `db:"pedido_id"`
	RepresentanteID uuid.UUID `db:"representante_id"`
	PercentualBP    int       `db:"percentual_bp"`
	Origem          string    `db:"origem"`
	UsuarioID       uuid.UUID `db:"usuario_id"`
	Versao          int       `db:"versao"`
	CriadoEm        time.Time `db:"criado_em"`
	AtualizadoEm    time.Time `db:"atualizado_em"`
}

type comissaoLinha struct {
	ID              uuid.UUID `db:"id"`
	EmpresaID       uuid.UUID `db:"empresa_id"`
	RepresentanteID uuid.UUID `db:"representante_id"`
	PedidoID        uuid.UUID `db:"pedido_id"`
	EntregaID       uuid.UUID `db:"entrega_id"`
	Competencia     time.Time `db:"competencia"`
	BaseCentavos    int64     `db:"base_centavos"`
	PercentualBP    int       `db:"percentual_bp"`
	ValorCentavos   int64     `db:"valor_centavos"`
	UsuarioID       uuid.UUID `db:"usuario_id"`
	CriadoEm        time.Time `db:"criado_em"`
}

type entregaParaApurarLinha struct {
	EntregaID                 uuid.UUID  `db:"entrega_id"`
	PedidoID                  uuid.UUID  `db:"pedido_id"`
	PedidoNumero              int        `db:"pedido_numero"`
	EmpresaID                 uuid.UUID  `db:"empresa_id"`
	RepresentanteID           uuid.UUID  `db:"representante_id"`
	Modelo                    string     `db:"modelo"`
	SaiuEm                    time.Time  `db:"saiu_em"`
	PercentualRepresentanteID *uuid.UUID `db:"percentual_representante_id"`
	PercentualBP              *int       `db:"percentual_bp"`
	PercentualOrigem          *string    `db:"percentual_origem"`
	PercentualUsuarioID       *uuid.UUID `db:"percentual_usuario_id"`
	PercentualVersao          *int       `db:"percentual_versao"`
	PercentualCriadoEm        *time.Time `db:"percentual_criado_em"`
	PercentualAtualizadoEm    *time.Time `db:"percentual_atualizado_em"`
}

type itemSaidoLinha struct {
	EntregaID             uuid.UUID `db:"entrega_id"`
	Quantidade            int       `db:"quantidade"`
	PrecoUnitarioCentavos int64     `db:"preco_unitario_centavos"`
}

const colunasPercentualPedido = `
	pedido_id, representante_id, percentual_bp, origem, usuario_id, versao, criado_em, atualizado_em
`

const colunasComissao = `
	id, empresa_id, representante_id, pedido_id, entrega_id, competencia, base_centavos, percentual_bp,
	valor_centavos, usuario_id, criado_em
`

func (r *ComissaoRepository) ObterPercentual(ctx context.Context, pedidoID uuid.UUID) (*domaincomissao.PercentualPedido, error) {
	var linha percentualPedidoLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasPercentualPedido+` FROM pedido_comissao WHERE pedido_id = $1`, pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioPercentualPedido(&linha), nil
}

func (r *ComissaoRepository) DefinirPercentual(ctx context.Context, percentual *domaincomissao.PercentualPedido, versaoLida *int) error {
	if percentual == nil {
		return domain.ErroValidacao("comissão do pedido não informada")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirPedidoComissionavel(ctx, tx, percentual.PedidoID, percentual.RepresentanteID); err != nil {
		return err
	}

	var versao int
	if versaoLida == nil {
		err = tx.GetContext(ctx, &versao, inserirOuSubstituirPercentual+` RETURNING versao`, argumentosDoPercentual(percentual)...)
	} else {
		err = tx.GetContext(ctx, &versao, `
			UPDATE pedido_comissao
			SET percentual_bp = $1, origem = $2, usuario_id = $3, versao = versao + 1, atualizado_em = $4
			WHERE pedido_id = $5 AND representante_id = $6 AND versao = $7
			RETURNING versao
		`, percentual.Percentual.PontosBase(), percentual.Origem.String(), percentual.UsuarioID, percentual.AtualizadoEm,
			percentual.PedidoID, percentual.RepresentanteID, *versaoLida)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domaincomissao.ErroVersaoDivergente()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	percentual.Versao = versao

	return nil
}

func (r *ComissaoRepository) ListarEntregasParaApurar(ctx context.Context, filtro portsout.EntregaFiltro) ([]*domaincomissao.EntregaParaApurar, error) {
	condicoes := []string{
		"e.status IN ('saiu', 'entregue')",
		"p.representante_id IS NOT NULL",
		"NOT EXISTS (SELECT 1 FROM comissao c WHERE c.entrega_id = e.id)",
	}
	argumentos := make([]any, 0, 5)
	adicionar := func(expressao string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, expressao+" $"+strconv.Itoa(len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("p.empresa_id =", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "p.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.SaidaAntesDe != nil {
		adicionar("e.saiu_em <", *filtro.SaidaAntesDe)
	}

	pagina := filtro.PaginacaoFiltro
	if pagina.Page < 1 {
		pagina.Page = 1
	}
	argumentos = append(argumentos, pagina.Size, (pagina.Page-1)*pagina.Size)

	var linhas []*entregaParaApurarLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT e.id AS entrega_id, p.id AS pedido_id, p.numero AS pedido_numero, p.empresa_id, p.representante_id,
			p.modelo, e.saiu_em, pc.representante_id AS percentual_representante_id, pc.percentual_bp,
			pc.origem AS percentual_origem, pc.usuario_id AS percentual_usuario_id, pc.versao AS percentual_versao,
			pc.criado_em AS percentual_criado_em, pc.atualizado_em AS percentual_atualizado_em
		FROM entrega e
		JOIN pedido p ON p.id = e.pedido_id
		LEFT JOIN pedido_comissao pc ON pc.pedido_id = p.id
		WHERE `+unirCondicoes(condicoes)+`
		ORDER BY e.saiu_em ASC, e.id ASC`+limiteOffset(argumentos), argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	entregas := make([]*domaincomissao.EntregaParaApurar, 0, len(linhas))
	porID := make(map[uuid.UUID]*domaincomissao.EntregaParaApurar, len(linhas))
	ids := make([]uuid.UUID, 0, len(linhas))
	for _, linha := range linhas {
		entrega := paraDominioEntregaParaApurar(linha)
		entregas = append(entregas, entrega)
		porID[entrega.EntregaID] = entrega
		ids = append(ids, entrega.EntregaID)
	}
	if len(ids) == 0 {
		return entregas, nil
	}

	var itens []*itemSaidoLinha
	if err := r.db.SelectContext(ctx, &itens, `
		SELECT ei.entrega_id, ei.quantidade, pi.preco_unitario_centavos
		FROM entrega_item ei
		JOIN pedido_item pi ON pi.id = ei.pedido_item_id
		WHERE ei.entrega_id = ANY($1::uuid[])
		ORDER BY ei.entrega_id, ei.pedido_item_id
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return nil, tratarErroDeConsulta(err)
	}
	for _, item := range itens {
		entrega := porID[item.EntregaID]
		entrega.Itens = append(entrega.Itens, domaincomissao.ItemSaido{
			Quantidade:            item.Quantidade,
			PrecoUnitarioCentavos: item.PrecoUnitarioCentavos,
		})
	}

	return entregas, nil
}

func (r *ComissaoRepository) GravarApuracao(ctx context.Context, apuracao *domaincomissao.Apuracao) (bool, error) {
	if apuracao == nil || apuracao.Comissao == nil || apuracao.Percentual == nil {
		return false, domain.ErroValidacao("apuração de comissão incompleta")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	comissao := apuracao.Comissao
	if err := exigirEntregaSaidaDoRepresentante(ctx, tx, comissao.EntregaID, comissao.RepresentanteID); err != nil {
		return false, err
	}
	if apuracao.PercentualNovo {
		if _, err := tx.ExecContext(ctx, inserirOuSubstituirPercentual, argumentosDoPercentual(apuracao.Percentual)...); err != nil {
			return false, tratarErroDeGravacao(err)
		}
	}
	if err := conferirPercentualTravado(ctx, tx, comissao); err != nil {
		return false, err
	}

	consulta, argumentos, err := tx.BindNamed(`
		INSERT INTO comissao (`+colunasComissao+`) VALUES (
			:id, :empresa_id, :representante_id, :pedido_id, :entrega_id, :competencia, :base_centavos, :percentual_bp,
			:valor_centavos, :usuario_id, :criado_em
		)
		ON CONFLICT (entrega_id) DO NOTHING
		RETURNING id
	`, paraLinhaComissao(comissao))
	if err != nil {
		return false, err
	}

	var id uuid.UUID
	err = tx.GetContext(ctx, &id, consulta, argumentos...)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return false, err
	}

	return true, nil
}

func (r *ComissaoRepository) Listar(ctx context.Context, filtro portsout.ComissaoFiltro) ([]*domaincomissao.Comissao, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 5)
	argumentos := make([]any, 0, 7)
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
	if filtro.RepresentanteID != nil {
		adicionar("representante_id =", *filtro.RepresentanteID)
	}
	if filtro.PedidoID != nil {
		adicionar("pedido_id =", *filtro.PedidoID)
	}
	if filtro.Competencia != nil {
		adicionar("competencia =", filtro.Competencia.Inicio())
	}

	query := `SELECT ` + colunasComissao + ` FROM comissao`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY competencia DESC, criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*comissaoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	comissoes := make([]*domaincomissao.Comissao, 0, len(linhas))
	for _, linha := range linhas {
		comissoes = append(comissoes, paraDominioComissao(linha))
	}

	return comissoes, nil
}

const inserirOuSubstituirPercentual = `
	INSERT INTO pedido_comissao (` + colunasPercentualPedido + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (pedido_id) DO UPDATE SET
		representante_id = EXCLUDED.representante_id,
		percentual_bp = EXCLUDED.percentual_bp,
		origem = EXCLUDED.origem,
		usuario_id = EXCLUDED.usuario_id,
		versao = pedido_comissao.versao + 1,
		atualizado_em = EXCLUDED.atualizado_em
	WHERE pedido_comissao.representante_id <> EXCLUDED.representante_id
`

func argumentosDoPercentual(percentual *domaincomissao.PercentualPedido) []any {
	return []any{
		percentual.PedidoID, percentual.RepresentanteID, percentual.Percentual.PontosBase(), percentual.Origem.String(),
		percentual.UsuarioID, percentual.Versao, percentual.CriadoEm, percentual.AtualizadoEm,
	}
}

func exigirPedidoComissionavel(ctx context.Context, tx *sqlx.Tx, pedidoID, representanteID uuid.UUID) error {
	var pedido struct {
		Status          string     `db:"status"`
		RepresentanteID *uuid.UUID `db:"representante_id"`
	}
	err := tx.GetContext(ctx, &pedido, `SELECT status, representante_id FROM pedido WHERE id = $1 FOR SHARE`, pedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincomissao.ErroPedidoNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if domainpedido.StatusDe(pedido.Status) == domainpedido.StatusCancelado {
		return domaincomissao.ErroPedidoCancelado()
	}
	if pedido.RepresentanteID == nil || *pedido.RepresentanteID != representanteID {
		return domaincomissao.ErroVersaoDivergente()
	}

	var comSaida bool
	if err := tx.GetContext(ctx, &comSaida, `
		SELECT EXISTS (SELECT 1 FROM entrega WHERE pedido_id = $1 AND status IN ('saiu', 'entregue'))
	`, pedidoID); err != nil {
		return tratarErroDeGravacao(err)
	}
	if comSaida {
		return domaincomissao.ErroPercentualTravado()
	}

	return nil
}

func exigirEntregaSaidaDoRepresentante(ctx context.Context, tx *sqlx.Tx, entregaID, representanteID uuid.UUID) error {
	var atual *uuid.UUID
	err := tx.GetContext(ctx, &atual, `
		SELECT p.representante_id
		FROM entrega e
		JOIN pedido p ON p.id = e.pedido_id
		WHERE e.id = $1 AND e.status IN ('saiu', 'entregue')
		FOR SHARE OF e, p
	`, entregaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErroValidacao("entrega sem saída não gera comissão")
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if atual == nil || *atual != representanteID {
		return domaincomissao.ErroRepresentanteMudouNaApuracao()
	}

	return nil
}

func conferirPercentualTravado(ctx context.Context, tx *sqlx.Tx, comissao *domaincomissao.Comissao) error {
	var vigente struct {
		RepresentanteID uuid.UUID `db:"representante_id"`
		PercentualBP    int       `db:"percentual_bp"`
	}
	err := tx.GetContext(ctx, &vigente, `
		SELECT representante_id, percentual_bp FROM pedido_comissao WHERE pedido_id = $1 FOR SHARE
	`, comissao.PedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincomissao.ErroPercentualMudouNaApuracao()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if vigente.RepresentanteID != comissao.RepresentanteID || vigente.PercentualBP != comissao.Percentual.PontosBase() {
		return domaincomissao.ErroPercentualMudouNaApuracao()
	}

	return nil
}

func paraDominioPercentualPedido(linha *percentualPedidoLinha) *domaincomissao.PercentualPedido {
	return &domaincomissao.PercentualPedido{
		PedidoID:        linha.PedidoID,
		RepresentanteID: linha.RepresentanteID,
		Percentual:      domaincliente.PercentualDe(linha.PercentualBP),
		Origem:          domaincomissao.OrigemPercentualDe(linha.Origem),
		UsuarioID:       linha.UsuarioID,
		Versao:          linha.Versao,
		CriadoEm:        linha.CriadoEm,
		AtualizadoEm:    linha.AtualizadoEm,
	}
}

func paraDominioEntregaParaApurar(linha *entregaParaApurarLinha) *domaincomissao.EntregaParaApurar {
	entrega := &domaincomissao.EntregaParaApurar{
		EntregaID:       linha.EntregaID,
		PedidoID:        linha.PedidoID,
		PedidoNumero:    linha.PedidoNumero,
		EmpresaID:       linha.EmpresaID,
		RepresentanteID: linha.RepresentanteID,
		Modelo:          domainpedido.ModeloDe(linha.Modelo),
		SaiuEm:          linha.SaiuEm,
		Itens:           []domaincomissao.ItemSaido{},
	}
	if linha.PercentualRepresentanteID != nil && linha.PercentualBP != nil && linha.PercentualOrigem != nil &&
		linha.PercentualUsuarioID != nil && linha.PercentualVersao != nil && linha.PercentualCriadoEm != nil &&
		linha.PercentualAtualizadoEm != nil {
		entrega.Percentual = paraDominioPercentualPedido(&percentualPedidoLinha{
			PedidoID:        linha.PedidoID,
			RepresentanteID: *linha.PercentualRepresentanteID,
			PercentualBP:    *linha.PercentualBP,
			Origem:          *linha.PercentualOrigem,
			UsuarioID:       *linha.PercentualUsuarioID,
			Versao:          *linha.PercentualVersao,
			CriadoEm:        *linha.PercentualCriadoEm,
			AtualizadoEm:    *linha.PercentualAtualizadoEm,
		})
	}

	return entrega
}

func paraDominioComissao(linha *comissaoLinha) *domaincomissao.Comissao {
	return &domaincomissao.Comissao{
		ID:              linha.ID,
		EmpresaID:       linha.EmpresaID,
		RepresentanteID: linha.RepresentanteID,
		PedidoID:        linha.PedidoID,
		EntregaID:       linha.EntregaID,
		Competencia:     domaincomissao.CompetenciaDe(linha.Competencia),
		BaseCentavos:    linha.BaseCentavos,
		Percentual:      domaincliente.PercentualDe(linha.PercentualBP),
		ValorCentavos:   linha.ValorCentavos,
		UsuarioID:       linha.UsuarioID,
		CriadoEm:        linha.CriadoEm,
	}
}

func paraLinhaComissao(comissao *domaincomissao.Comissao) *comissaoLinha {
	return &comissaoLinha{
		ID:              comissao.ID,
		EmpresaID:       comissao.EmpresaID,
		RepresentanteID: comissao.RepresentanteID,
		PedidoID:        comissao.PedidoID,
		EntregaID:       comissao.EntregaID,
		Competencia:     comissao.Competencia.Inicio(),
		BaseCentavos:    comissao.BaseCentavos,
		PercentualBP:    comissao.Percentual.PontosBase(),
		ValorCentavos:   comissao.ValorCentavos,
		UsuarioID:       comissao.UsuarioID,
		CriadoEm:        comissao.CriadoEm,
	}
}
