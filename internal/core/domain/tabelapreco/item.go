package tabelapreco

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

const tamanhoMaximoObservacao = 500

type Item struct {
	ID            uuid.UUID
	ProdutoID     uuid.UUID
	Modelo        domainpedido.Modelo
	PrecoCentavos int64
	VigenteDe     time.Time
	Observacao    string
	UsuarioID     uuid.UUID
	CriadoEm      time.Time
}

type DadosItem struct {
	ProdutoID     uuid.UUID
	Modelo        string
	PrecoCentavos int64
	VigenteDe     time.Time
	Observacao    string
}

func NovoItem(dados DadosItem, usuarioID uuid.UUID, agora time.Time) (*Item, error) {
	switch {
	case dados.ProdutoID == uuid.Nil:
		return nil, domain.ErroValidacao("produto do preço é obrigatório")
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário não informado")
	case dados.PrecoCentavos <= 0:
		return nil, domain.ErroValidacao("preço da tabela deve ser maior que zero")
	case dados.VigenteDe.IsZero():
		return nil, domain.ErroValidacao("início da vigência é obrigatório")
	}

	modelo, err := domainpedido.NovoModelo(dados.Modelo)
	if err != nil {
		return nil, err
	}

	observacao := strings.TrimSpace(dados.Observacao)
	if utf8.RuneCountInString(observacao) > tamanhoMaximoObservacao {
		return nil, domain.ErroValidacao("observação do preço deve ter no máximo 500 caracteres")
	}

	return &Item{
		ID:            uuid.New(),
		ProdutoID:     dados.ProdutoID,
		Modelo:        modelo,
		PrecoCentavos: dados.PrecoCentavos,
		VigenteDe:     domainpedido.DataDe(dados.VigenteDe),
		Observacao:    observacao,
		UsuarioID:     usuarioID,
		CriadoEm:      agora,
	}, nil
}

func (i *Item) Futuro(agora time.Time) bool {
	return i.VigenteDe.After(domainpedido.Hoje(agora))
}
