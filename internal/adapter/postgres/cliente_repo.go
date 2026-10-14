package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.ClienteRepository = (*ClienteRepository)(nil)

type ClienteRepository struct {
	db *sqlx.DB
}

func NewClienteRepository(db *sqlx.DB) *ClienteRepository {
	return &ClienteRepository{db: db}
}

type clienteLinha struct {
	ID                      uuid.UUID  `db:"id"`
	Documento               string     `db:"documento"`
	TipoPessoa              string     `db:"tipo_pessoa"`
	Nome                    string     `db:"nome"`
	NomeFantasia            *string    `db:"nome_fantasia"`
	NomeBusca               string     `db:"nome_busca"`
	Tipo                    string     `db:"tipo"`
	GrupoID                 *uuid.UUID `db:"grupo_id"`
	UF                      string     `db:"uf"`
	Cidade                  string     `db:"cidade"`
	EmpresaRelacionamentoID *uuid.UUID `db:"empresa_relacionamento_id"`
	RepresentanteID         *uuid.UUID `db:"representante_id"`
	EmailNF                 *string    `db:"email_nf"`
	Telefone                *string    `db:"telefone"`
	Observacoes             string     `db:"observacoes"`
	ExigeNumeroPedidoNaNF   bool       `db:"exige_numero_pedido_na_nf"`
	ProibeAntecipacao       bool       `db:"proibe_antecipacao"`
	LeadID                  *uuid.UUID `db:"lead_id"`
	Ativo                   bool       `db:"ativo"`
	CriadoEm                time.Time  `db:"criado_em"`
	AtualizadoEm            time.Time  `db:"atualizado_em"`
}

const colunasCliente = `
	id, documento, tipo_pessoa, nome, nome_fantasia, nome_busca, tipo, grupo_id, uf, cidade,
	empresa_relacionamento_id, representante_id, email_nf, telefone, observacoes,
	exige_numero_pedido_na_nf, proibe_antecipacao, lead_id, ativo, criado_em, atualizado_em
`

func (r *ClienteRepository) Criar(ctx context.Context, cliente *domaincliente.Cliente) (uuid.UUID, error) {
	if _, err := r.db.NamedExecContext(ctx, `
		INSERT INTO cliente (
			id, documento, tipo_pessoa, nome, nome_fantasia, nome_busca, tipo, grupo_id, uf, cidade,
			empresa_relacionamento_id, representante_id, email_nf, telefone, observacoes,
			exige_numero_pedido_na_nf, proibe_antecipacao, lead_id, ativo, criado_em, atualizado_em
		)
		VALUES (
			:id, :documento, :tipo_pessoa, :nome, :nome_fantasia, :nome_busca, :tipo, :grupo_id, :uf, :cidade,
			:empresa_relacionamento_id, :representante_id, :email_nf, :telefone, :observacoes,
			:exige_numero_pedido_na_nf, :proibe_antecipacao, :lead_id, :ativo, :criado_em, :atualizado_em
		)
	`, paraLinhaCliente(cliente)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return cliente.ID, nil
}

func (r *ClienteRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	return r.obterPor(ctx, `id = $1`, id)
}

func (r *ClienteRepository) ObterPorDocumento(ctx context.Context, documento string) (*domaincliente.Cliente, error) {
	return r.obterPor(ctx, `documento = $1`, documento)
}

func (r *ClienteRepository) obterPor(ctx context.Context, condicao string, valor any) (*domaincliente.Cliente, error) {
	var linha clienteLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasCliente+` FROM cliente WHERE `+condicao, valor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioCliente(&linha), nil
}

func (r *ClienteRepository) Listar(ctx context.Context, filtro portsout.ClienteFiltro) ([]*domaincliente.Cliente, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasCliente + ` FROM cliente`
	condicoes := make([]string, 0, 7)
	argumentos := make([]any, 0, 10)

	if condicaoBusca, argumentosBusca := buscaPorNomeOuDocumento(filtro.Busca, len(argumentos)); condicaoBusca != "" {
		argumentos = append(argumentos, argumentosBusca...)
		condicoes = append(condicoes, condicaoBusca)
	}
	filtrosExatos := []struct {
		coluna string
		valor  any
		ativo  bool
	}{
		{"grupo_id", filtro.GrupoID, filtro.GrupoID != nil},
		{"representante_id", filtro.RepresentanteID, filtro.RepresentanteID != nil},
		{"tipo", string(filtro.Tipo), filtro.Tipo != ""},
		{"uf", filtro.UF, filtro.UF != ""},
		{"ativo", filtro.Ativo, filtro.Ativo != nil},
		{"lead_id", filtro.LeadID, filtro.LeadID != nil},
	}
	for _, exato := range filtrosExatos {
		if exato.ativo {
			argumentos = append(argumentos, exato.valor)
			condicoes = append(condicoes, condicao(exato.coluna, len(argumentos)))
		}
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome_busca ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*clienteLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domaincliente.Cliente, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioCliente(linha))
	}

	return itens, nil
}

func (r *ClienteRepository) Atualizar(ctx context.Context, cliente *domaincliente.Cliente) (bool, error) {
	linha := paraLinhaCliente(cliente)
	resultado, err := r.db.ExecContext(ctx, `
		UPDATE cliente
		SET nome = $1,
		    nome_fantasia = $2,
		    nome_busca = $3,
		    tipo = $4,
		    grupo_id = $5,
		    uf = $6,
		    cidade = $7,
		    empresa_relacionamento_id = $8,
		    representante_id = $9,
		    email_nf = $10,
		    telefone = $11,
		    observacoes = $12,
		    exige_numero_pedido_na_nf = $13,
		    proibe_antecipacao = $14,
		    ativo = $15,
		    atualizado_em = $16
		WHERE id = $17
	`,
		linha.Nome, linha.NomeFantasia, linha.NomeBusca, linha.Tipo, linha.GrupoID, linha.UF, linha.Cidade,
		linha.EmpresaRelacionamentoID, linha.RepresentanteID, linha.EmailNF, linha.Telefone, linha.Observacoes,
		linha.ExigeNumeroPedidoNaNF, linha.ProibeAntecipacao, linha.Ativo, linha.AtualizadoEm, linha.ID,
	)

	return linhasAfetadas(resultado, err)
}

func (r *ClienteRepository) Remover(ctx context.Context, id uuid.UUID) (bool, error) {
	resultado, err := r.db.ExecContext(ctx, `DELETE FROM cliente WHERE id = $1`, id)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErro(err)
	}

	return linhas > 0, nil
}

func paraLinhaCliente(cliente *domaincliente.Cliente) clienteLinha {
	return clienteLinha{
		ID:                      cliente.ID,
		Documento:               cliente.Documento.Valor(),
		TipoPessoa:              cliente.Documento.Pessoa().String(),
		Nome:                    cliente.Nome,
		NomeFantasia:            cliente.NomeFantasia,
		NomeBusca:               cliente.NomeBusca,
		Tipo:                    cliente.Tipo.String(),
		GrupoID:                 cliente.GrupoID,
		UF:                      cliente.UF.String(),
		Cidade:                  cliente.Cidade,
		EmpresaRelacionamentoID: cliente.EmpresaRelacionamentoID,
		RepresentanteID:         cliente.RepresentanteID,
		EmailNF:                 cliente.EmailNF,
		Telefone:                cliente.Telefone,
		Observacoes:             cliente.Observacoes,
		ExigeNumeroPedidoNaNF:   cliente.ExigeNumeroPedidoNaNF,
		ProibeAntecipacao:       cliente.ProibeAntecipacao,
		LeadID:                  cliente.LeadID,
		Ativo:                   cliente.Ativo,
		CriadoEm:                cliente.CriadoEm,
		AtualizadoEm:            cliente.AtualizadoEm,
	}
}

func paraDominioCliente(linha *clienteLinha) *domaincliente.Cliente {
	return &domaincliente.Cliente{
		ID:                      linha.ID,
		Documento:               domaincliente.DocumentoDe(linha.Documento),
		Nome:                    linha.Nome,
		NomeFantasia:            linha.NomeFantasia,
		NomeBusca:               linha.NomeBusca,
		Tipo:                    domaincliente.TipoClienteDe(linha.Tipo),
		GrupoID:                 linha.GrupoID,
		UF:                      domainempresa.UFDe(linha.UF),
		Cidade:                  linha.Cidade,
		EmpresaRelacionamentoID: linha.EmpresaRelacionamentoID,
		RepresentanteID:         linha.RepresentanteID,
		EmailNF:                 linha.EmailNF,
		Telefone:                linha.Telefone,
		Observacoes:             linha.Observacoes,
		ExigeNumeroPedidoNaNF:   linha.ExigeNumeroPedidoNaNF,
		ProibeAntecipacao:       linha.ProibeAntecipacao,
		LeadID:                  linha.LeadID,
		Ativo:                   linha.Ativo,
		CriadoEm:                linha.CriadoEm,
		AtualizadoEm:            linha.AtualizadoEm,
	}
}
