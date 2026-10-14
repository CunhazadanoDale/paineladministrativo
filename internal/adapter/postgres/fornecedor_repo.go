package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/fornecedor"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.FornecedorRepository = (*FornecedorRepository)(nil)

type FornecedorRepository struct {
	db *sqlx.DB
}

func NewFornecedorRepository(db *sqlx.DB) *FornecedorRepository {
	return &FornecedorRepository{db: db}
}

type fornecedorLinha struct {
	ID                       uuid.UUID `db:"id"`
	Pais                     string    `db:"pais"`
	Documento                *string   `db:"documento"`
	TipoPessoa               *string   `db:"tipo_pessoa"`
	IdentificacaoEstrangeira *string   `db:"identificacao_estrangeira"`
	Nome                     string    `db:"nome"`
	NomeFantasia             *string   `db:"nome_fantasia"`
	NomeBusca                string    `db:"nome_busca"`
	MoedaPadrao              string    `db:"moeda_padrao"`
	UF                       *string   `db:"uf"`
	Cidade                   *string   `db:"cidade"`
	ContatoNome              *string   `db:"contato_nome"`
	Email                    *string   `db:"email"`
	Telefone                 *string   `db:"telefone"`
	Observacoes              string    `db:"observacoes"`
	Ativo                    bool      `db:"ativo"`
	Versao                   int       `db:"versao"`
	CriadoEm                 time.Time `db:"criado_em"`
	AtualizadoEm             time.Time `db:"atualizado_em"`
}

const colunasFornecedor = `
	id, pais, documento, tipo_pessoa, identificacao_estrangeira, nome, nome_fantasia, nome_busca,
	moeda_padrao, uf, cidade, contato_nome, email, telefone, observacoes, ativo, versao,
	criado_em, atualizado_em
`

func (r *FornecedorRepository) Criar(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) (uuid.UUID, error) {
	if _, err := r.db.NamedExecContext(ctx, `
		INSERT INTO fornecedor (
			id, pais, documento, tipo_pessoa, identificacao_estrangeira, nome, nome_fantasia, nome_busca,
			moeda_padrao, uf, cidade, contato_nome, email, telefone, observacoes, ativo, versao,
			criado_em, atualizado_em
		)
		VALUES (
			:id, :pais, :documento, :tipo_pessoa, :identificacao_estrangeira, :nome, :nome_fantasia, :nome_busca,
			:moeda_padrao, :uf, :cidade, :contato_nome, :email, :telefone, :observacoes, :ativo, :versao,
			:criado_em, :atualizado_em
		)
	`, paraLinhaFornecedor(fornecedor)); err != nil {
		return uuid.Nil, erroDeGravacaoDoFornecedor(err)
	}

	return fornecedor.ID, nil
}

func (r *FornecedorRepository) Obter(ctx context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return r.obterPor(ctx, `id = $1`, id)
}

func (r *FornecedorRepository) ObterPorDocumento(ctx context.Context, documento string) (*domainfornecedor.Fornecedor, error) {
	return r.obterPor(ctx, `documento = $1`, documento)
}

func (r *FornecedorRepository) ObterPorIdentificacaoEstrangeira(
	ctx context.Context,
	pais domainfornecedor.Pais,
	identificacao string,
) (*domainfornecedor.Fornecedor, error) {
	return r.obterPor(ctx, `pais = $1 AND identificacao_estrangeira = $2`, pais.String(), identificacao)
}

func (r *FornecedorRepository) obterPor(ctx context.Context, condicao string, valores ...any) (*domainfornecedor.Fornecedor, error) {
	var linha fornecedorLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasFornecedor+` FROM fornecedor WHERE `+condicao, valores...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioFornecedor(&linha), nil
}

func (r *FornecedorRepository) Listar(ctx context.Context, filtro portsout.FornecedorFiltro) ([]*domainfornecedor.Fornecedor, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasFornecedor + ` FROM fornecedor`
	condicoes := make([]string, 0, 4)
	argumentos := make([]any, 0, 8)

	if condicaoBusca, argumentosBusca := buscaDeFornecedor(filtro.Busca, len(argumentos)); condicaoBusca != "" {
		argumentos = append(argumentos, argumentosBusca...)
		condicoes = append(condicoes, condicaoBusca)
	}
	filtrosExatos := []struct {
		coluna string
		valor  any
		ativo  bool
	}{
		{"pais", filtro.Pais.String(), filtro.Pais != ""},
		{"moeda_padrao", filtro.Moeda.String(), filtro.Moeda != ""},
		{"ativo", filtro.Ativo, filtro.Ativo != nil},
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

	var linhas []*fornecedorLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainfornecedor.Fornecedor, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioFornecedor(linha))
	}

	return itens, nil
}

func (r *FornecedorRepository) Atualizar(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) (bool, error) {
	linha := paraLinhaFornecedor(fornecedor)

	var novaVersao int
	err := r.db.GetContext(ctx, &novaVersao, `
		UPDATE fornecedor
		SET identificacao_estrangeira = $1,
		    nome = $2,
		    nome_fantasia = $3,
		    nome_busca = $4,
		    moeda_padrao = $5,
		    uf = $6,
		    cidade = $7,
		    contato_nome = $8,
		    email = $9,
		    telefone = $10,
		    observacoes = $11,
		    ativo = $12,
		    atualizado_em = $13,
		    versao = versao + 1
		WHERE id = $14
		  AND versao = $15
		RETURNING versao
	`,
		linha.IdentificacaoEstrangeira, linha.Nome, linha.NomeFantasia, linha.NomeBusca, linha.MoedaPadrao,
		linha.UF, linha.Cidade, linha.ContatoNome, linha.Email, linha.Telefone, linha.Observacoes,
		linha.Ativo, linha.AtualizadoEm, linha.ID, linha.Versao,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return r.versaoDivergente(ctx, fornecedor.ID)
	}
	if err != nil {
		return false, erroDeGravacaoDoFornecedor(err)
	}

	fornecedor.Versao = novaVersao

	return true, nil
}

func (r *FornecedorRepository) versaoDivergente(ctx context.Context, id uuid.UUID) (bool, error) {
	var existe bool
	if err := r.db.GetContext(ctx, &existe, `SELECT EXISTS(SELECT 1 FROM fornecedor WHERE id = $1)`, id); err != nil {
		return false, err
	}
	if !existe {
		return false, nil
	}

	return false, domainfornecedor.ErroVersaoDivergente()
}

func (r *FornecedorRepository) Remover(ctx context.Context, id uuid.UUID) (bool, error) {
	resultado, err := r.db.ExecContext(ctx, `DELETE FROM fornecedor WHERE id = $1`, id)
	if err != nil {
		return false, tratarErro(err)
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErro(err)
	}

	return linhas > 0, nil
}

func buscaDeFornecedor(busca string, posicaoAtual int) (string, []any) {
	termo := domaincliente.NomeDeBusca(busca)
	if termo == "" {
		return "", nil
	}

	colunas := []string{"nome_busca"}
	argumentos := []any{"%" + escaparBusca(termo) + "%"}

	if termoDocumento := domaincliente.TermoDeDocumento(busca); termoDocumento != "" {
		colunas = append(colunas, "documento")
		argumentos = append(argumentos, escaparBusca(termoDocumento)+"%")
	}

	colunas = append(colunas, "identificacao_estrangeira")
	argumentos = append(argumentos, escaparBusca(strings.ToUpper(strings.TrimSpace(busca)))+"%")

	alternativas := make([]string, 0, len(colunas))
	for indice, coluna := range colunas {
		alternativas = append(alternativas, coluna+" LIKE $"+strconv.Itoa(posicaoAtual+indice+1))
	}

	return "(" + strings.Join(alternativas, " OR ") + ")", argumentos
}

func erroDeGravacaoDoFornecedor(err error) error {
	var erroPostgres *pq.Error
	if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
		switch erroPostgres.Constraint {
		case "uq_fornecedor_documento":
			return domain.ErroConflito("já existe um fornecedor com este documento")
		case "uq_fornecedor_identificacao_estrangeira":
			return domain.ErroConflito("já existe um fornecedor com esta identificação neste país")
		}
	}

	return tratarErroDeGravacao(err)
}

func paraLinhaFornecedor(fornecedor *domainfornecedor.Fornecedor) fornecedorLinha {
	linha := fornecedorLinha{
		ID:                       fornecedor.ID,
		Pais:                     fornecedor.Pais.String(),
		IdentificacaoEstrangeira: fornecedor.IdentificacaoEstrangeira,
		Nome:                     fornecedor.Nome,
		NomeFantasia:             fornecedor.NomeFantasia,
		NomeBusca:                fornecedor.NomeBusca,
		MoedaPadrao:              fornecedor.MoedaPadrao.String(),
		Cidade:                   fornecedor.Cidade,
		ContatoNome:              fornecedor.ContatoNome,
		Email:                    fornecedor.Email,
		Telefone:                 fornecedor.Telefone,
		Observacoes:              fornecedor.Observacoes,
		Ativo:                    fornecedor.Ativo,
		Versao:                   fornecedor.Versao,
		CriadoEm:                 fornecedor.CriadoEm,
		AtualizadoEm:             fornecedor.AtualizadoEm,
	}
	if !fornecedor.Documento.Vazio() {
		documento := fornecedor.Documento.Valor()
		pessoa := fornecedor.Documento.Pessoa().String()
		linha.Documento = &documento
		linha.TipoPessoa = &pessoa
	}
	if fornecedor.UF != nil {
		uf := fornecedor.UF.String()
		linha.UF = &uf
	}

	return linha
}

func paraDominioFornecedor(linha *fornecedorLinha) *domainfornecedor.Fornecedor {
	fornecedor := &domainfornecedor.Fornecedor{
		ID:                       linha.ID,
		Pais:                     domainfornecedor.PaisDe(linha.Pais),
		IdentificacaoEstrangeira: linha.IdentificacaoEstrangeira,
		Nome:                     linha.Nome,
		NomeFantasia:             linha.NomeFantasia,
		NomeBusca:                linha.NomeBusca,
		MoedaPadrao:              domainfornecedor.MoedaDe(linha.MoedaPadrao),
		Cidade:                   linha.Cidade,
		ContatoNome:              linha.ContatoNome,
		Email:                    linha.Email,
		Telefone:                 linha.Telefone,
		Observacoes:              linha.Observacoes,
		Ativo:                    linha.Ativo,
		Versao:                   linha.Versao,
		CriadoEm:                 linha.CriadoEm,
		AtualizadoEm:             linha.AtualizadoEm,
	}
	if linha.Documento != nil {
		fornecedor.Documento = domaincliente.DocumentoDe(*linha.Documento)
	}
	if linha.UF != nil {
		uf := domainempresa.UFDe(*linha.UF)
		fornecedor.UF = &uf
	}

	return fornecedor
}
