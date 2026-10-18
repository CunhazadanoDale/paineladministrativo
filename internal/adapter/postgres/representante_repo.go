package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.RepresentanteRepository = (*RepresentanteRepository)(nil)

type RepresentanteRepository struct {
	db *sqlx.DB
}

func NewRepresentanteRepository(db *sqlx.DB) *RepresentanteRepository {
	return &RepresentanteRepository{db: db}
}

type representanteLinha struct {
	ID                    uuid.UUID  `db:"id"`
	Nome                  string     `db:"nome"`
	NomeBusca             string     `db:"nome_busca"`
	Documento             *string    `db:"documento"`
	UsuarioID             *uuid.UUID `db:"usuario_id"`
	Email                 *string    `db:"email"`
	Telefone              *string    `db:"telefone"`
	ComissaoProntaEntrega int        `db:"comissao_pronta_entrega_bp"`
	ComissaoImportacao    int        `db:"comissao_importacao_bp"`
	FornecedorID          *uuid.UUID `db:"fornecedor_id"`
	Ativo                 bool       `db:"ativo"`
	CriadoEm              time.Time  `db:"criado_em"`
	AtualizadoEm          time.Time  `db:"atualizado_em"`
}

const colunasRepresentante = `
	id, nome, nome_busca, documento, usuario_id, email, telefone,
	comissao_pronta_entrega_bp, comissao_importacao_bp, fornecedor_id, ativo, criado_em, atualizado_em
`

func (r *RepresentanteRepository) Criar(ctx context.Context, representante *domaincliente.Representante) (uuid.UUID, error) {
	if _, err := r.db.NamedExecContext(ctx, `
		INSERT INTO representante (
			id, nome, nome_busca, documento, usuario_id, email, telefone,
			comissao_pronta_entrega_bp, comissao_importacao_bp, fornecedor_id, ativo, criado_em, atualizado_em
		)
		VALUES (
			:id, :nome, :nome_busca, :documento, :usuario_id, :email, :telefone,
			:comissao_pronta_entrega_bp, :comissao_importacao_bp, :fornecedor_id, :ativo, :criado_em, :atualizado_em
		)
	`, paraLinhaRepresentante(representante)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return representante.ID, nil
}

func (r *RepresentanteRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Representante, error) {
	return r.obterPor(ctx, `id = $1`, id)
}

func (r *RepresentanteRepository) ObterPorDocumento(ctx context.Context, documento string) (*domaincliente.Representante, error) {
	return r.obterPor(ctx, `documento = $1`, documento)
}

func (r *RepresentanteRepository) ObterPorUsuario(ctx context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	return r.obterPor(ctx, `usuario_id = $1`, usuarioID)
}

func (r *RepresentanteRepository) obterPor(ctx context.Context, condicao string, valor any) (*domaincliente.Representante, error) {
	var linha representanteLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasRepresentante+` FROM representante WHERE `+condicao, valor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioRepresentante(&linha), nil
}

func (r *RepresentanteRepository) Listar(ctx context.Context, filtro portsout.RepresentanteFiltro) ([]*domaincliente.Representante, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasRepresentante + ` FROM representante`
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 5)

	if condicaoBusca, argumentosBusca := buscaPorNomeOuDocumento(filtro.Busca, len(argumentos)); condicaoBusca != "" {
		argumentos = append(argumentos, argumentosBusca...)
		condicoes = append(condicoes, condicaoBusca)
	}
	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY nome_busca ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*representanteLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domaincliente.Representante, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioRepresentante(linha))
	}

	return itens, nil
}

func (r *RepresentanteRepository) Atualizar(ctx context.Context, representante *domaincliente.Representante) (bool, error) {
	linha := paraLinhaRepresentante(representante)
	resultado, err := r.db.ExecContext(ctx, `
		UPDATE representante
		SET nome = $1,
		    nome_busca = $2,
		    documento = $3,
		    usuario_id = $4,
		    email = $5,
		    telefone = $6,
		    comissao_pronta_entrega_bp = $7,
		    comissao_importacao_bp = $8,
		    ativo = $9,
		    atualizado_em = $10,
		    fornecedor_id = $12
		WHERE id = $11
	`,
		linha.Nome, linha.NomeBusca, linha.Documento, linha.UsuarioID, linha.Email, linha.Telefone,
		linha.ComissaoProntaEntrega, linha.ComissaoImportacao, linha.Ativo, linha.AtualizadoEm, linha.ID,
		linha.FornecedorID,
	)

	return linhasAfetadas(resultado, err)
}

func buscaPorNomeOuDocumento(busca string, posicaoAtual int) (string, []any) {
	termo := domaincliente.NomeDeBusca(busca)
	if termo == "" {
		return "", nil
	}

	nome := "$" + strconv.Itoa(posicaoAtual+1)
	argumentos := []any{"%" + escaparBusca(termo) + "%"}

	termoDocumento := domaincliente.TermoDeDocumento(busca)
	if termoDocumento == "" {
		return "nome_busca LIKE " + nome, argumentos
	}

	documento := "$" + strconv.Itoa(posicaoAtual+2)
	argumentos = append(argumentos, escaparBusca(termoDocumento)+"%")

	return "(nome_busca LIKE " + nome + " OR documento LIKE " + documento + ")", argumentos
}

func paraLinhaRepresentante(representante *domaincliente.Representante) representanteLinha {
	linha := representanteLinha{
		ID:                    representante.ID,
		Nome:                  representante.Nome,
		NomeBusca:             representante.NomeBusca,
		UsuarioID:             representante.UsuarioID,
		Email:                 representante.Email,
		Telefone:              representante.Telefone,
		ComissaoProntaEntrega: representante.ComissaoProntaEntrega.PontosBase(),
		ComissaoImportacao:    representante.ComissaoImportacao.PontosBase(),
		FornecedorID:          representante.FornecedorID,
		Ativo:                 representante.Ativo,
		CriadoEm:              representante.CriadoEm,
		AtualizadoEm:          representante.AtualizadoEm,
	}
	if representante.Documento != nil {
		documento := representante.Documento.Valor()
		linha.Documento = &documento
	}

	return linha
}

func paraDominioRepresentante(linha *representanteLinha) *domaincliente.Representante {
	representante := &domaincliente.Representante{
		ID:                    linha.ID,
		Nome:                  linha.Nome,
		NomeBusca:             linha.NomeBusca,
		UsuarioID:             linha.UsuarioID,
		Email:                 linha.Email,
		Telefone:              linha.Telefone,
		ComissaoProntaEntrega: domaincliente.PercentualDe(linha.ComissaoProntaEntrega),
		ComissaoImportacao:    domaincliente.PercentualDe(linha.ComissaoImportacao),
		FornecedorID:          linha.FornecedorID,
		Ativo:                 linha.Ativo,
		CriadoEm:              linha.CriadoEm,
		AtualizadoEm:          linha.AtualizadoEm,
	}
	if linha.Documento != nil {
		documento := domaincliente.DocumentoDe(*linha.Documento)
		representante.Documento = &documento
	}

	return representante
}
