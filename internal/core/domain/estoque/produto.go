package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoNomeProduto = 200

type Produto struct {
	ID               uuid.UUID
	CategoriaID      uuid.UUID
	Nome             string
	Slug             Slug
	Descricao        string
	Codigo           *string
	UnidadeMedida    UnidadeMedida
	Preco            *Preco
	PrecoPromocional *Preco
	Saldo            Saldo
	EstoqueMinimo    *EstoqueMinimo
	Peso             *Peso
	Destaque         bool
	Ativo            bool
	CriadoEm         time.Time
	AtualizadoEm     time.Time
	Imagens          []*Imagem
}

type ProdutoInput struct {
	Categoria        *Categoria
	Nome             string
	Slug             Slug
	Descricao        string
	Codigo           *string
	UnidadeMedida    UnidadeMedida
	Preco            *Preco
	PrecoPromocional *Preco
	EstoqueMinimo    *EstoqueMinimo
	Peso             *Peso
	Destaque         bool
}

func NovoProduto(input ProdutoInput) (*Produto, error) {
	if err := validarProdutoInput(input); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &Produto{
		ID:               uuid.New(),
		CategoriaID:      input.Categoria.ID,
		Nome:             strings.TrimSpace(input.Nome),
		Slug:             input.Slug,
		Descricao:        strings.TrimSpace(input.Descricao),
		Codigo:           normalizarCodigo(input.Codigo),
		UnidadeMedida:    input.UnidadeMedida,
		Preco:            input.Preco,
		PrecoPromocional: input.PrecoPromocional,
		Saldo:            SaldoDe(0),
		EstoqueMinimo:    input.EstoqueMinimo,
		Peso:             input.Peso,
		Destaque:         input.Destaque,
		Ativo:            true,
		CriadoEm:         agora,
		AtualizadoEm:     agora,
	}, nil
}

func validarProdutoInput(input ProdutoInput) error {
	if input.Categoria == nil {
		return domain.ErroValidacao("categoria do produto é obrigatória")
	}
	if !input.Categoria.EhSubcategoria() {
		return domain.ErroValidacao("produto deve estar vinculado a uma subcategoria")
	}
	if !input.Categoria.Ativo {
		return domain.ErroValidacao("categoria do produto deve estar ativa")
	}

	nome := strings.TrimSpace(input.Nome)
	if nome == "" {
		return domain.ErroValidacao("nome do produto é obrigatório")
	}
	if utf8.RuneCountInString(nome) > tamanhoMaximoNomeProduto {
		return domain.ErroValidacao("nome do produto deve ter no máximo 200 caracteres")
	}
	if input.Slug.Vazio() {
		return domain.ErroValidacao("slug do produto é obrigatório")
	}
	if err := input.UnidadeMedida.Validado(); err != nil {
		return err
	}
	if codigo := normalizarCodigo(input.Codigo); codigo != nil && utf8.RuneCountInString(*codigo) > tamanhoMaximoCodigoProduto {
		return domain.ErroValidacao("código do produto deve ter no máximo 60 caracteres")
	}
	if input.PrecoPromocional != nil && input.Preco == nil {
		return domain.ErroValidacao("preço promocional exige preço cheio")
	}
	if input.PrecoPromocional != nil && input.Preco != nil && !input.PrecoPromocional.MenorQue(*input.Preco) {
		return domain.ErroValidacao("preço promocional deve ser menor que o preço cheio")
	}

	return nil
}

const tamanhoMaximoCodigoProduto = 60

func normalizarCodigo(codigo *string) *string {
	if codigo == nil {
		return nil
	}

	normalizado := strings.TrimSpace(*codigo)
	if normalizado == "" {
		return nil
	}

	return &normalizado
}

func (p *Produto) AlterarDados(input ProdutoInput, agora time.Time) error {
	if err := validarProdutoInput(input); err != nil {
		return err
	}

	p.CategoriaID = input.Categoria.ID
	p.Nome = strings.TrimSpace(input.Nome)
	p.Descricao = strings.TrimSpace(input.Descricao)
	p.Codigo = normalizarCodigo(input.Codigo)
	p.UnidadeMedida = input.UnidadeMedida
	p.Preco = input.Preco
	p.PrecoPromocional = input.PrecoPromocional
	p.EstoqueMinimo = input.EstoqueMinimo
	p.Peso = input.Peso
	p.Destaque = input.Destaque
	p.AtualizadoEm = agora

	return nil
}

func (p *Produto) Movimentar(tipo TipoMovimento, quantidade Quantidade, usuarioID uuid.UUID, documentoRef *string, observacao string, agora time.Time) (*Movimento, error) {
	if err := tipo.Validado(); err != nil {
		return nil, err
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário da movimentação não informado")
	}

	novoSaldo, err := p.Saldo.Aplicar(tipo.Efeito(quantidade))
	if err != nil {
		return nil, err
	}

	movimento, err := NovoMovimento(p.ID, tipo, quantidade, novoSaldo, usuarioID, documentoRef, observacao, agora)
	if err != nil {
		return nil, err
	}

	p.Saldo = novoSaldo
	p.AtualizadoEm = agora

	return movimento, nil
}

func (p *Produto) AlterarPreco(preco *Preco, precoPromocional *Preco, agora time.Time) error {
	if precoPromocional != nil && preco == nil {
		return domain.ErroValidacao("preço promocional exige preço cheio")
	}
	if precoPromocional != nil && preco != nil && !precoPromocional.MenorQue(*preco) {
		return domain.ErroValidacao("preço promocional deve ser menor que o preço cheio")
	}

	p.Preco = preco
	p.PrecoPromocional = precoPromocional
	p.AtualizadoEm = agora

	return nil
}

func (p *Produto) AlternarAtivo(ativo bool, agora time.Time) {
	p.Ativo = ativo
	p.AtualizadoEm = agora
}

func (p *Produto) AlternarDestaque(destaque bool, agora time.Time) {
	p.Destaque = destaque
	p.AtualizadoEm = agora
}

func (p *Produto) EstoqueBaixo() bool {
	return p.EstoqueMinimo != nil && p.EstoqueMinimo.Atingido(p.Saldo)
}
