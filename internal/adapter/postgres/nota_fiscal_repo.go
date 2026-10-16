package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.NotaFiscalRepository = (*NotaFiscalRepository)(nil)

type NotaFiscalRepository struct {
	db *sqlx.DB
}

func NewNotaFiscalRepository(db *sqlx.DB) *NotaFiscalRepository {
	return &NotaFiscalRepository{db: db}
}

var guardaDaNotaFiscal = guardaDeVersao{
	tabela:        "nota_fiscal",
	naoEncontrado: func() error { return domain.ErroNaoEncontrado("nota fiscal não encontrada") },
	alterado:      domainnotafiscal.ErroVersaoDivergente,
	conflito:      conflitoDaNotaFiscal,
}

func conflitoDaNotaFiscal(restricao string) error {
	switch restricao {
	case "uq_nota_fiscal_chave":
		return domain.ErroConflito("já existe uma nota fiscal com esta chave de acesso")
	case "uq_nota_fiscal_arquivo", "uq_nota_fiscal_evento_arquivo":
		return domain.ErroConflito("arquivo já importado em outra nota fiscal ou evento")
	case "uq_nota_fiscal_entrega_vigente":
		return domain.ErroConflito("a entrega já tem uma nota fiscal autorizada")
	case "uq_nota_fiscal_evento":
		return domain.ErroConflito("evento já registrado nesta nota fiscal")
	case "uq_nota_fiscal_duplicata_numero":
		return domain.ErroConflito("número de duplicata repetido na nota")
	default:
		return nil
	}
}

type notaFiscalLinha struct {
	ID                    uuid.UUID  `db:"id"`
	Chave                 string     `db:"chave"`
	EmpresaID             uuid.UUID  `db:"empresa_id"`
	Numero                int        `db:"numero"`
	Serie                 int        `db:"serie"`
	EmitidaEm             time.Time  `db:"emitida_em"`
	TipoOperacao          string     `db:"tipo_operacao"`
	Finalidade            string     `db:"finalidade"`
	Natureza              string     `db:"natureza"`
	NaturezaOperacao      string     `db:"natureza_operacao"`
	EntreEmpresas         bool       `db:"entre_empresas"`
	ClienteID             *uuid.UUID `db:"cliente_id"`
	EmpresaDestinatariaID *uuid.UUID `db:"empresa_destinataria_id"`
	DestinatarioDocumento string     `db:"destinatario_documento"`
	DestinatarioNome      string     `db:"destinatario_nome"`
	CFOPPredominante      string     `db:"cfop_predominante"`
	ValorTotalCentavos    int64      `db:"valor_total_centavos"`
	ValorProdutosCentavos int64      `db:"valor_produtos_centavos"`
	ValorFreteCentavos    int64      `db:"valor_frete_centavos"`
	ValorDescontoCentavos int64      `db:"valor_desconto_centavos"`
	NumeroPedidoCliente   *string    `db:"numero_pedido_cliente"`
	Protocolo             string     `db:"protocolo"`
	AutorizadaEm          time.Time  `db:"autorizada_em"`
	Status                string     `db:"status"`
	PedidoID              *uuid.UUID `db:"pedido_id"`
	EntregaID             *uuid.UUID `db:"entrega_id"`
	ArquivoID             uuid.UUID  `db:"arquivo_id"`
	UsuarioID             uuid.UUID  `db:"usuario_id"`
	Versao                int        `db:"versao"`
	CriadoEm              time.Time  `db:"criado_em"`
	AtualizadoEm          time.Time  `db:"atualizado_em"`
}

type notaFiscalItemLinha struct {
	NotaFiscalID            uuid.UUID  `db:"nota_fiscal_id"`
	NumeroItem              int        `db:"numero_item"`
	Codigo                  string     `db:"codigo"`
	Descricao               string     `db:"descricao"`
	ProdutoID               *uuid.UUID `db:"produto_id"`
	NCM                     string     `db:"ncm"`
	CFOP                    string     `db:"cfop"`
	Unidade                 string     `db:"unidade"`
	QuantidadeDecimilesimos int64      `db:"quantidade_decimilesimos"`
	ValorUnitario           string     `db:"valor_unitario"`
	ValorCentavos           int64      `db:"valor_centavos"`
	DescontoCentavos        int64      `db:"desconto_centavos"`
	PedidoCliente           *string    `db:"pedido_cliente"`
}

type notaFiscalDuplicataLinha struct {
	ID            uuid.UUID `db:"id"`
	NotaFiscalID  uuid.UUID `db:"nota_fiscal_id"`
	Numero        string    `db:"numero"`
	Vencimento    time.Time `db:"vencimento"`
	ValorCentavos int64     `db:"valor_centavos"`
}

type notaFiscalEventoLinha struct {
	ID            uuid.UUID `db:"id"`
	NotaFiscalID  uuid.UUID `db:"nota_fiscal_id"`
	Tipo          string    `db:"tipo"`
	Sequencia     int       `db:"sequencia"`
	OcorridoEm    time.Time `db:"ocorrido_em"`
	Protocolo     string    `db:"protocolo"`
	Justificativa string    `db:"justificativa"`
	ArquivoID     uuid.UUID `db:"arquivo_id"`
	UsuarioID     uuid.UUID `db:"usuario_id"`
	CriadoEm      time.Time `db:"criado_em"`
}

type itemPendenteLinha struct {
	notaFiscalItemLinha
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Chave      string    `db:"chave"`
	NumeroNota int       `db:"numero_nota"`
	Serie      int       `db:"serie"`
	EmitidaEm  time.Time `db:"emitida_em"`
}

type entregaSemNotaLinha struct {
	EntregaID      uuid.UUID  `db:"entrega_id"`
	PedidoID       uuid.UUID  `db:"pedido_id"`
	EmpresaID      uuid.UUID  `db:"empresa_id"`
	PedidoNumero   int        `db:"pedido_numero"`
	ClienteID      uuid.UUID  `db:"cliente_id"`
	Status         string     `db:"status"`
	SaiuEm         *time.Time `db:"saiu_em"`
	EntregueEm     *time.Time `db:"entregue_em"`
	Transportadora *string    `db:"transportadora"`
}

const colunasNotaFiscal = `
	id, chave, empresa_id, numero, serie, emitida_em, tipo_operacao, finalidade, natureza,
	natureza_operacao, entre_empresas, cliente_id, empresa_destinataria_id, destinatario_documento,
	destinatario_nome, cfop_predominante, valor_total_centavos, valor_produtos_centavos,
	valor_frete_centavos, valor_desconto_centavos, numero_pedido_cliente, protocolo, autorizada_em,
	status, pedido_id, entrega_id, arquivo_id, usuario_id, versao, criado_em, atualizado_em
`

const colunasNotaFiscalItem = `
	nota_fiscal_id, numero_item, codigo, descricao, produto_id, ncm, cfop, unidade,
	quantidade_decimilesimos, valor_unitario, valor_centavos, desconto_centavos, pedido_cliente
`

const colunasNotaFiscalEvento = `
	id, nota_fiscal_id, tipo, sequencia, ocorrido_em, protocolo, justificativa, arquivo_id, usuario_id, criado_em
`

func (r *NotaFiscalRepository) Criar(ctx context.Context, nota *domainnotafiscal.NotaFiscal) error {
	if nota == nil {
		return domain.ErroValidacao("nota fiscal não informada")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := conferirVinculoDaNota(ctx, tx, nota); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO nota_fiscal (`+colunasNotaFiscal+`)
		VALUES (
			:id, :chave, :empresa_id, :numero, :serie, :emitida_em, :tipo_operacao, :finalidade, :natureza,
			:natureza_operacao, :entre_empresas, :cliente_id, :empresa_destinataria_id, :destinatario_documento,
			:destinatario_nome, :cfop_predominante, :valor_total_centavos, :valor_produtos_centavos,
			:valor_frete_centavos, :valor_desconto_centavos, :numero_pedido_cliente, :protocolo, :autorizada_em,
			:status, :pedido_id, :entrega_id, :arquivo_id, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaNotaFiscal(nota)); err != nil {
		return guardaDaNotaFiscal.traduzir(err)
	}

	for _, item := range nota.Itens {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO nota_fiscal_item (`+colunasNotaFiscalItem+`)
			VALUES (
				:nota_fiscal_id, :numero_item, :codigo, :descricao, :produto_id, :ncm, :cfop, :unidade,
				:quantidade_decimilesimos, :valor_unitario, :valor_centavos, :desconto_centavos, :pedido_cliente
			)
		`, paraLinhaNotaFiscalItem(nota.ID, item)); err != nil {
			return guardaDaNotaFiscal.traduzir(err)
		}
	}

	for _, duplicata := range nota.Duplicatas {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO nota_fiscal_duplicata (id, nota_fiscal_id, numero, vencimento, valor_centavos)
			VALUES ($1, $2, $3, $4, $5)
		`, duplicata.ID, nota.ID, duplicata.Numero, duplicata.Vencimento, duplicata.ValorCentavos); err != nil {
			return guardaDaNotaFiscal.traduzir(err)
		}
	}

	return guardaDaNotaFiscal.traduzir(tx.Commit())
}

func (r *NotaFiscalRepository) Obter(ctx context.Context, id uuid.UUID) (*domainnotafiscal.NotaFiscal, error) {
	return r.obterPor(ctx, "id = $1", id)
}

func (r *NotaFiscalRepository) ObterPorChave(ctx context.Context, chave string) (*domainnotafiscal.NotaFiscal, error) {
	return r.obterPor(ctx, "chave = $1", chave)
}

func (r *NotaFiscalRepository) Listar(ctx context.Context, filtro portsout.NotaFiscalFiltro) ([]*domainnotafiscal.NotaFiscal, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 12)
	argumentos := make([]any, 0, 14)
	adicionar := func(expressao string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, expressao+strconv.Itoa(len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("empresa_id = $", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		adicionar("empresa_id = ANY($", pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes[len(condicoes)-1] += "::uuid[])"
	}
	if filtro.ClienteID != nil {
		adicionar("cliente_id = $", *filtro.ClienteID)
	}
	if filtro.RepresentanteID != nil {
		adicionar("cliente_id IN (SELECT id FROM cliente WHERE representante_id = $", *filtro.RepresentanteID)
		condicoes[len(condicoes)-1] += ")"
	}
	if filtro.PedidoID != nil {
		adicionar("pedido_id = $", *filtro.PedidoID)
	}
	if filtro.Natureza != "" {
		adicionar("natureza = $", filtro.Natureza.String())
	}
	if filtro.Status != "" {
		adicionar("status = $", filtro.Status.String())
	}
	if filtro.EntreEmpresas != nil {
		adicionar("entre_empresas = $", *filtro.EntreEmpresas)
	}
	if filtro.SemPedido {
		condicoes = append(condicoes, "pedido_id IS NULL AND NOT entre_empresas")
	}
	if filtro.EmitidaDe != nil {
		adicionar("emitida_em >= $", domainpedido.InicioDoDia(*filtro.EmitidaDe))
	}
	if filtro.EmitidaAte != nil {
		adicionar("emitida_em < $", domainpedido.InicioDoDia(filtro.EmitidaAte.AddDate(0, 0, 1)))
	}
	if filtro.Numero != nil {
		adicionar("numero = $", *filtro.Numero)
	}
	if filtro.Chave != "" {
		adicionar("chave = $", filtro.Chave)
	}

	query := `SELECT ` + colunasNotaFiscal + ` FROM nota_fiscal`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY emitida_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*notaFiscalLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	notas := make([]*domainnotafiscal.NotaFiscal, 0, len(linhas))
	for _, linha := range linhas {
		notas = append(notas, paraDominioNotaFiscal(linha))
	}

	return notas, nil
}

func (r *NotaFiscalRepository) Atualizar(ctx context.Context, alteracao *portsout.Alteracao) error {
	if alteracao == nil || alteracao.Nota == nil {
		return domain.ErroValidacao("alteração de nota fiscal sem nota")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	nota := alteracao.Nota
	if alteracao.ConferirVinculo {
		if err := conferirVinculoDaNota(ctx, tx, nota); err != nil {
			return err
		}
	}

	versao, err := guardaDaNotaFiscal.atualizar(ctx, tx, nota.ID, `
		UPDATE nota_fiscal
		SET status = $1,
		    pedido_id = $2,
		    entrega_id = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE id = $5
		  AND versao = $6
		  AND status = $7
		RETURNING versao
	`,
		nota.Status.String(), nota.PedidoID, nota.EntregaID, nota.AtualizadoEm,
		nota.ID, alteracao.VersaoLida, alteracao.StatusAnterior.String(),
	)
	if err != nil {
		return err
	}

	for _, item := range alteracao.ItensAlterados {
		resultado, err := tx.ExecContext(ctx, `
			UPDATE nota_fiscal_item
			SET produto_id = $1
			WHERE nota_fiscal_id = $2
			  AND numero_item = $3
		`, item.ProdutoID, nota.ID, item.NumeroItem)
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		linhas, err := resultado.RowsAffected()
		if err != nil {
			return tratarErroDeGravacao(err)
		}
		if linhas == 0 {
			return domain.ErroNaoEncontrado("item da nota fiscal não encontrado")
		}
	}

	if evento := alteracao.Evento; evento != nil {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO nota_fiscal_evento (`+colunasNotaFiscalEvento+`)
			VALUES (:id, :nota_fiscal_id, :tipo, :sequencia, :ocorrido_em, :protocolo, :justificativa, :arquivo_id, :usuario_id, :criado_em)
		`, paraLinhaNotaFiscalEvento(evento)); err != nil {
			return guardaDaNotaFiscal.traduzir(err)
		}
	}

	if err := guardaDaNotaFiscal.traduzir(tx.Commit()); err != nil {
		return err
	}

	nota.Versao = versao

	return nil
}

func (r *NotaFiscalRepository) ListarEventos(ctx context.Context, notaFiscalID uuid.UUID) ([]*domainnotafiscal.Evento, error) {
	var linhas []*notaFiscalEventoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT `+colunasNotaFiscalEvento+`
		FROM nota_fiscal_evento
		WHERE nota_fiscal_id = $1
		ORDER BY ocorrido_em ASC, id ASC
	`, notaFiscalID); err != nil {
		return nil, err
	}

	eventos := make([]*domainnotafiscal.Evento, 0, len(linhas))
	for _, linha := range linhas {
		eventos = append(eventos, &domainnotafiscal.Evento{
			ID:            linha.ID,
			NotaFiscalID:  linha.NotaFiscalID,
			Tipo:          domainnotafiscal.TipoEventoDe(linha.Tipo),
			Sequencia:     linha.Sequencia,
			OcorridoEm:    linha.OcorridoEm,
			Protocolo:     linha.Protocolo,
			Justificativa: linha.Justificativa,
			ArquivoID:     linha.ArquivoID,
			UsuarioID:     linha.UsuarioID,
			CriadoEm:      linha.CriadoEm,
		})
	}

	return eventos, nil
}

func (r *NotaFiscalRepository) ListarItensSemProduto(ctx context.Context, filtro portsout.ItemPendenteFiltro) ([]*domainnotafiscal.ItemPendente, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := []string{"i.produto_id IS NULL"}
	argumentos := make([]any, 0, 4)
	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes = append(condicoes, condicao("n.empresa_id", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "n.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	argumentos = append(argumentos, filtro.Size, offset)

	var linhas []*itemPendenteLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT i.nota_fiscal_id, i.numero_item, i.codigo, i.descricao, i.produto_id, i.ncm, i.cfop, i.unidade,
		       i.quantidade_decimilesimos, i.valor_unitario, i.valor_centavos, i.desconto_centavos, i.pedido_cliente,
		       n.empresa_id, n.chave, n.numero AS numero_nota, n.serie, n.emitida_em
		FROM nota_fiscal_item i
		JOIN nota_fiscal n ON n.id = i.nota_fiscal_id
		WHERE `+unirCondicoes(condicoes)+`
		ORDER BY n.emitida_em DESC, i.nota_fiscal_id DESC, i.numero_item ASC`+limiteOffset(argumentos),
		argumentos...,
	); err != nil {
		return nil, err
	}

	pendentes := make([]*domainnotafiscal.ItemPendente, 0, len(linhas))
	for _, linha := range linhas {
		pendentes = append(pendentes, &domainnotafiscal.ItemPendente{
			NotaFiscalID: linha.NotaFiscalID,
			EmpresaID:    linha.EmpresaID,
			Chave:        domainnotafiscal.ChaveDe(linha.Chave),
			NumeroNota:   linha.NumeroNota,
			Serie:        linha.Serie,
			EmitidaEm:    linha.EmitidaEm,
			Item:         paraDominioNotaFiscalItem(&linha.notaFiscalItemLinha),
		})
	}

	return pendentes, nil
}

func (r *NotaFiscalRepository) ListarEntregasSemNota(ctx context.Context, filtro portsout.EntregaSemNotaFiltro) ([]*domainnotafiscal.EntregaSemNota, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := []string{
		"e.status IN ('saiu', 'entregue')",
		"NOT EXISTS (SELECT 1 FROM nota_fiscal n WHERE n.entrega_id = e.id AND n.status = 'autorizada')",
	}
	argumentos := make([]any, 0, 6)
	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes = append(condicoes, condicao("p.empresa_id", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "p.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ClienteID != nil {
		argumentos = append(argumentos, *filtro.ClienteID)
		condicoes = append(condicoes, condicao("p.cliente_id", len(argumentos)))
	}
	if filtro.RepresentanteID != nil {
		argumentos = append(argumentos, *filtro.RepresentanteID)
		condicoes = append(condicoes, condicao("p.representante_id", len(argumentos)))
	}
	argumentos = append(argumentos, filtro.Size, offset)

	var linhas []*entregaSemNotaLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT e.id AS entrega_id, e.pedido_id, p.empresa_id, p.numero AS pedido_numero, p.cliente_id,
		       e.status, e.saiu_em, e.entregue_em, e.transportadora
		FROM entrega e
		JOIN pedido p ON p.id = e.pedido_id
		WHERE `+unirCondicoes(condicoes)+`
		ORDER BY e.saiu_em ASC, e.id ASC`+limiteOffset(argumentos),
		argumentos...,
	); err != nil {
		return nil, err
	}

	entregas := make([]*domainnotafiscal.EntregaSemNota, 0, len(linhas))
	for _, linha := range linhas {
		entregas = append(entregas, &domainnotafiscal.EntregaSemNota{
			EntregaID:      linha.EntregaID,
			PedidoID:       linha.PedidoID,
			EmpresaID:      linha.EmpresaID,
			PedidoNumero:   linha.PedidoNumero,
			ClienteID:      linha.ClienteID,
			Status:         linha.Status,
			SaiuEm:         dataDoBanco(linha.SaiuEm),
			EntregueEm:     dataDoBanco(linha.EntregueEm),
			Transportadora: linha.Transportadora,
		})
	}

	return entregas, nil
}

func (r *NotaFiscalRepository) ArquivoVinculado(ctx context.Context, arquivoID uuid.UUID) (bool, error) {
	var vinculado bool
	err := r.db.GetContext(ctx, &vinculado, `
		SELECT EXISTS (SELECT 1 FROM nota_fiscal WHERE arquivo_id = $1)
		    OR EXISTS (SELECT 1 FROM nota_fiscal_evento WHERE arquivo_id = $1)
	`, arquivoID)

	return vinculado, err
}

func (r *NotaFiscalRepository) obterPor(ctx context.Context, criterio string, valor any) (*domainnotafiscal.NotaFiscal, error) {
	var linha notaFiscalLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasNotaFiscal+` FROM nota_fiscal WHERE `+criterio, valor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var itens []*notaFiscalItemLinha
	if err := r.db.SelectContext(ctx, &itens, `
		SELECT `+colunasNotaFiscalItem+` FROM nota_fiscal_item WHERE nota_fiscal_id = $1 ORDER BY numero_item ASC
	`, linha.ID); err != nil {
		return nil, err
	}

	var duplicatas []*notaFiscalDuplicataLinha
	if err := r.db.SelectContext(ctx, &duplicatas, `
		SELECT id, nota_fiscal_id, numero, vencimento, valor_centavos
		FROM nota_fiscal_duplicata
		WHERE nota_fiscal_id = $1
		ORDER BY vencimento ASC, numero ASC, id ASC
	`, linha.ID); err != nil {
		return nil, err
	}

	nota := paraDominioNotaFiscal(&linha)
	nota.Itens = make([]*domainnotafiscal.Item, 0, len(itens))
	for _, item := range itens {
		nota.Itens = append(nota.Itens, paraDominioNotaFiscalItem(item))
	}
	nota.Duplicatas = make([]*domainnotafiscal.Duplicata, 0, len(duplicatas))
	for _, duplicata := range duplicatas {
		nota.Duplicatas = append(nota.Duplicatas, &domainnotafiscal.Duplicata{
			ID:            duplicata.ID,
			NotaFiscalID:  duplicata.NotaFiscalID,
			Numero:        duplicata.Numero,
			Vencimento:    domainpedido.DataDe(duplicata.Vencimento),
			ValorCentavos: duplicata.ValorCentavos,
		})
	}

	return nota, nil
}

func conferirVinculoDaNota(ctx context.Context, tx *sqlx.Tx, nota *domainnotafiscal.NotaFiscal) error {
	if nota.PedidoID == nil {
		return nil
	}

	var pedido struct {
		Status string `db:"status"`
	}
	err := tx.GetContext(ctx, &pedido, `SELECT status FROM pedido WHERE id = $1 FOR SHARE`, *nota.PedidoID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErroValidacao("pedido não encontrado")
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	status := domainpedido.StatusDe(pedido.Status)
	if status == domainpedido.StatusProposta || status == domainpedido.StatusCancelado {
		return domain.ErroConflito("o pedido mudou de situação e não recebe mais nota fiscal; recarregue e tente de novo")
	}

	if nota.EntregaID == nil {
		return nil
	}

	var entrega struct {
		PedidoID uuid.UUID `db:"pedido_id"`
		Status   string    `db:"status"`
	}
	err = tx.GetContext(ctx, &entrega, `SELECT pedido_id, status FROM entrega WHERE id = $1 FOR SHARE`, *nota.EntregaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErroValidacao("entrega não encontrada")
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if entrega.PedidoID != *nota.PedidoID {
		return domain.ErroValidacao("a entrega não pertence ao pedido")
	}
	if domainpedido.StatusEntregaDe(entrega.Status) == domainpedido.StatusEntregaCancelada {
		return domain.ErroConflito("a entrega foi cancelada e não recebe mais nota fiscal; recarregue e tente de novo")
	}

	return nil
}

func paraLinhaNotaFiscal(nota *domainnotafiscal.NotaFiscal) notaFiscalLinha {
	return notaFiscalLinha{
		ID:                    nota.ID,
		Chave:                 nota.Chave.Valor(),
		EmpresaID:             nota.EmpresaID,
		Numero:                nota.Numero,
		Serie:                 nota.Serie,
		EmitidaEm:             nota.EmitidaEm,
		TipoOperacao:          nota.TipoOperacao.String(),
		Finalidade:            nota.Finalidade.String(),
		Natureza:              nota.Natureza.String(),
		NaturezaOperacao:      nota.NaturezaOperacao,
		EntreEmpresas:         nota.EntreEmpresas,
		ClienteID:             nota.ClienteID,
		EmpresaDestinatariaID: nota.EmpresaDestinatariaID,
		DestinatarioDocumento: nota.DestinatarioDocumento,
		DestinatarioNome:      nota.DestinatarioNome,
		CFOPPredominante:      nota.CFOPPredominante,
		ValorTotalCentavos:    nota.ValorTotalCentavos,
		ValorProdutosCentavos: nota.ValorProdutosCentavos,
		ValorFreteCentavos:    nota.ValorFreteCentavos,
		ValorDescontoCentavos: nota.ValorDescontoCentavos,
		NumeroPedidoCliente:   nota.NumeroPedidoCliente,
		Protocolo:             nota.Protocolo,
		AutorizadaEm:          nota.AutorizadaEm,
		Status:                nota.Status.String(),
		PedidoID:              nota.PedidoID,
		EntregaID:             nota.EntregaID,
		ArquivoID:             nota.ArquivoID,
		UsuarioID:             nota.UsuarioID,
		Versao:                nota.Versao,
		CriadoEm:              nota.CriadoEm,
		AtualizadoEm:          nota.AtualizadoEm,
	}
}

func paraLinhaNotaFiscalItem(notaFiscalID uuid.UUID, item *domainnotafiscal.Item) notaFiscalItemLinha {
	return notaFiscalItemLinha{
		NotaFiscalID:            notaFiscalID,
		NumeroItem:              item.NumeroItem,
		Codigo:                  item.Codigo,
		Descricao:               item.Descricao,
		ProdutoID:               item.ProdutoID,
		NCM:                     item.NCM,
		CFOP:                    item.CFOP,
		Unidade:                 item.Unidade,
		QuantidadeDecimilesimos: item.QuantidadeDecimilesimos,
		ValorUnitario:           item.ValorUnitario,
		ValorCentavos:           item.ValorCentavos,
		DescontoCentavos:        item.DescontoCentavos,
		PedidoCliente:           item.PedidoCliente,
	}
}

func paraLinhaNotaFiscalEvento(evento *domainnotafiscal.Evento) notaFiscalEventoLinha {
	return notaFiscalEventoLinha{
		ID:            evento.ID,
		NotaFiscalID:  evento.NotaFiscalID,
		Tipo:          evento.Tipo.String(),
		Sequencia:     evento.Sequencia,
		OcorridoEm:    evento.OcorridoEm,
		Protocolo:     evento.Protocolo,
		Justificativa: evento.Justificativa,
		ArquivoID:     evento.ArquivoID,
		UsuarioID:     evento.UsuarioID,
		CriadoEm:      evento.CriadoEm,
	}
}

func paraDominioNotaFiscal(linha *notaFiscalLinha) *domainnotafiscal.NotaFiscal {
	return &domainnotafiscal.NotaFiscal{
		ID:                    linha.ID,
		Chave:                 domainnotafiscal.ChaveDe(linha.Chave),
		EmpresaID:             linha.EmpresaID,
		Numero:                linha.Numero,
		Serie:                 linha.Serie,
		EmitidaEm:             linha.EmitidaEm,
		TipoOperacao:          domainnotafiscal.TipoOperacaoDe(linha.TipoOperacao),
		Finalidade:            domainnotafiscal.FinalidadeDe(linha.Finalidade),
		Natureza:              domainnotafiscal.NaturezaDe(linha.Natureza),
		NaturezaOperacao:      linha.NaturezaOperacao,
		EntreEmpresas:         linha.EntreEmpresas,
		ClienteID:             linha.ClienteID,
		EmpresaDestinatariaID: linha.EmpresaDestinatariaID,
		DestinatarioDocumento: linha.DestinatarioDocumento,
		DestinatarioNome:      linha.DestinatarioNome,
		CFOPPredominante:      linha.CFOPPredominante,
		ValorTotalCentavos:    linha.ValorTotalCentavos,
		ValorProdutosCentavos: linha.ValorProdutosCentavos,
		ValorFreteCentavos:    linha.ValorFreteCentavos,
		ValorDescontoCentavos: linha.ValorDescontoCentavos,
		NumeroPedidoCliente:   linha.NumeroPedidoCliente,
		Protocolo:             linha.Protocolo,
		AutorizadaEm:          linha.AutorizadaEm,
		Status:                domainnotafiscal.StatusDe(linha.Status),
		PedidoID:              linha.PedidoID,
		EntregaID:             linha.EntregaID,
		ArquivoID:             linha.ArquivoID,
		UsuarioID:             linha.UsuarioID,
		Versao:                linha.Versao,
		CriadoEm:              linha.CriadoEm,
		AtualizadoEm:          linha.AtualizadoEm,
	}
}

func paraDominioNotaFiscalItem(linha *notaFiscalItemLinha) *domainnotafiscal.Item {
	return &domainnotafiscal.Item{
		NotaFiscalID:            linha.NotaFiscalID,
		NumeroItem:              linha.NumeroItem,
		Codigo:                  linha.Codigo,
		Descricao:               linha.Descricao,
		ProdutoID:               linha.ProdutoID,
		NCM:                     linha.NCM,
		CFOP:                    linha.CFOP,
		Unidade:                 linha.Unidade,
		QuantidadeDecimilesimos: linha.QuantidadeDecimilesimos,
		ValorUnitario:           linha.ValorUnitario,
		ValorCentavos:           linha.ValorCentavos,
		DescontoCentavos:        linha.DescontoCentavos,
		PedidoCliente:           linha.PedidoCliente,
	}
}
