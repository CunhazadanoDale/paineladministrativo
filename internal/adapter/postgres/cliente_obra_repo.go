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

var _ portsout.ObraRepository = (*ClienteObraRepository)(nil)

type ClienteObraRepository struct {
	db *sqlx.DB
}

func NewClienteObraRepository(db *sqlx.DB) *ClienteObraRepository {
	return &ClienteObraRepository{db: db}
}

type clienteObraLinha struct {
	ID           uuid.UUID `db:"id"`
	ClienteID    uuid.UUID `db:"cliente_id"`
	Nome         string    `db:"nome"`
	CNPJ         *string   `db:"cnpj"`
	Logradouro   string    `db:"logradouro"`
	Numero       *string   `db:"numero"`
	Complemento  *string   `db:"complemento"`
	Bairro       *string   `db:"bairro"`
	Cidade       string    `db:"cidade"`
	UF           string    `db:"uf"`
	CEP          *string   `db:"cep"`
	Ativo        bool      `db:"ativo"`
	CriadoEm     time.Time `db:"criado_em"`
	AtualizadoEm time.Time `db:"atualizado_em"`
}

const colunasClienteObra = `
	id, cliente_id, nome, cnpj, logradouro, numero, complemento, bairro, cidade, uf, cep,
	ativo, criado_em, atualizado_em
`

func (r *ClienteObraRepository) Criar(ctx context.Context, obra *domaincliente.Obra) (uuid.UUID, error) {
	if _, err := r.db.NamedExecContext(ctx, `
		INSERT INTO cliente_obra (
			id, cliente_id, nome, cnpj, logradouro, numero, complemento, bairro, cidade, uf, cep,
			ativo, criado_em, atualizado_em
		)
		VALUES (
			:id, :cliente_id, :nome, :cnpj, :logradouro, :numero, :complemento, :bairro, :cidade, :uf, :cep,
			:ativo, :criado_em, :atualizado_em
		)
	`, paraLinhaObra(obra)); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return obra.ID, nil
}

func (r *ClienteObraRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Obra, error) {
	var linha clienteObraLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasClienteObra+` FROM cliente_obra WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominioObra(&linha), nil
}

func (r *ClienteObraRepository) Listar(ctx context.Context, filtro portsout.ObraFiltro) ([]*domaincliente.Obra, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	argumentos := []any{filtro.ClienteID}
	condicoes := []string{condicao("cliente_id", 1)}
	if filtro.Ativo != nil {
		argumentos = append(argumentos, *filtro.Ativo)
		condicoes = append(condicoes, condicao("ativo", len(argumentos)))
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query := `SELECT ` + colunasClienteObra + ` FROM cliente_obra WHERE ` + unirCondicoes(condicoes) +
		` ORDER BY nome ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*clienteObraLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domaincliente.Obra, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominioObra(linha))
	}

	return itens, nil
}

func (r *ClienteObraRepository) Atualizar(ctx context.Context, obra *domaincliente.Obra) (bool, error) {
	linha := paraLinhaObra(obra)
	resultado, err := r.db.ExecContext(ctx, `
		UPDATE cliente_obra
		SET nome = $1,
		    cnpj = $2,
		    logradouro = $3,
		    numero = $4,
		    complemento = $5,
		    bairro = $6,
		    cidade = $7,
		    uf = $8,
		    cep = $9,
		    ativo = $10,
		    atualizado_em = $11
		WHERE id = $12
	`,
		linha.Nome, linha.CNPJ, linha.Logradouro, linha.Numero, linha.Complemento, linha.Bairro,
		linha.Cidade, linha.UF, linha.CEP, linha.Ativo, linha.AtualizadoEm, linha.ID,
	)

	return linhasAfetadas(resultado, err)
}

func paraLinhaObra(obra *domaincliente.Obra) clienteObraLinha {
	linha := clienteObraLinha{
		ID:           obra.ID,
		ClienteID:    obra.ClienteID,
		Nome:         obra.Nome,
		Logradouro:   obra.Endereco.Logradouro,
		Numero:       obra.Endereco.Numero,
		Complemento:  obra.Endereco.Complemento,
		Bairro:       obra.Endereco.Bairro,
		Cidade:       obra.Endereco.Cidade,
		UF:           obra.Endereco.UF.String(),
		CEP:          obra.Endereco.CEP,
		Ativo:        obra.Ativo,
		CriadoEm:     obra.CriadoEm,
		AtualizadoEm: obra.AtualizadoEm,
	}
	if obra.CNPJ != nil {
		cnpj := obra.CNPJ.Digitos()
		linha.CNPJ = &cnpj
	}

	return linha
}

func paraDominioObra(linha *clienteObraLinha) *domaincliente.Obra {
	obra := &domaincliente.Obra{
		ID:        linha.ID,
		ClienteID: linha.ClienteID,
		Nome:      linha.Nome,
		Endereco: domaincliente.Endereco{
			Logradouro:  linha.Logradouro,
			Numero:      linha.Numero,
			Complemento: linha.Complemento,
			Bairro:      linha.Bairro,
			Cidade:      linha.Cidade,
			UF:          domainempresa.UFDe(linha.UF),
			CEP:         linha.CEP,
		},
		Ativo:        linha.Ativo,
		CriadoEm:     linha.CriadoEm,
		AtualizadoEm: linha.AtualizadoEm,
	}
	if linha.CNPJ != nil {
		cnpj := domainempresa.CNPJDe(*linha.CNPJ)
		obra.CNPJ = &cnpj
	}

	return obra
}
