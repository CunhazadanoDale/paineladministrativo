package importacao

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

const tamanhoMaximoDescricaoCusto = 255

type TipoCusto string

const (
	TipoCustoFreteInternacional TipoCusto = "frete_internacional"
	TipoCustoSeguro             TipoCusto = "seguro"
	TipoCustoImpostoImportacao  TipoCusto = "imposto_importacao"
	TipoCustoIPI                TipoCusto = "ipi"
	TipoCustoPIS                TipoCusto = "pis"
	TipoCustoCOFINS             TipoCusto = "cofins"
	TipoCustoICMS               TipoCusto = "icms"
	TipoCustoTaxaSiscomex       TipoCusto = "taxa_siscomex"
	TipoCustoDespachante        TipoCusto = "despachante"
	TipoCustoArmazenagem        TipoCusto = "armazenagem"
	TipoCustoFreteNacional      TipoCusto = "frete_nacional"
	TipoCustoOutros             TipoCusto = "outros"
)

var descricoesDosTipos = map[TipoCusto]string{
	TipoCustoFreteInternacional: "Frete internacional",
	TipoCustoSeguro:             "Seguro internacional",
	TipoCustoImpostoImportacao:  "Imposto de importação",
	TipoCustoIPI:                "IPI na importação",
	TipoCustoPIS:                "PIS-importação",
	TipoCustoCOFINS:             "COFINS-importação",
	TipoCustoICMS:               "ICMS na importação",
	TipoCustoTaxaSiscomex:       "Taxa Siscomex",
	TipoCustoDespachante:        "Despachante aduaneiro",
	TipoCustoArmazenagem:        "Armazenagem",
	TipoCustoFreteNacional:      "Frete nacional",
	TipoCustoOutros:             "Outros custos",
}

func NovoTipoCusto(valor string) (TipoCusto, error) {
	tipo := TipoCusto(strings.ToLower(strings.TrimSpace(valor)))
	if _, ok := descricoesDosTipos[tipo]; !ok {
		return "", domain.ErroValidacao("tipo do custo deve ser frete_internacional, seguro, imposto_importacao, ipi, pis, cofins, " +
			"icms, taxa_siscomex, despachante, armazenagem, frete_nacional ou outros")
	}

	return tipo, nil
}

func TipoCustoDe(valor string) TipoCusto {
	return TipoCusto(valor)
}

func (t TipoCusto) String() string {
	return string(t)
}

type Custo struct {
	ID                    uuid.UUID
	ImportacaoID          uuid.UUID
	Tipo                  TipoCusto
	Descricao             string
	FornecedorID          uuid.UUID
	ValorCentavos         int64
	Vencimento            time.Time
	CompoeCusto           bool
	Posterior             bool
	CategoriaFinanceiraID uuid.UUID
	SolicitacaoID         uuid.UUID
	StatusContaAPagar     *domainsolicitacao.Status
	CanceladoEm           *time.Time
	CanceladoPor          *uuid.UUID
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type DadosCusto struct {
	Tipo                  TipoCusto
	Descricao             string
	FornecedorID          uuid.UUID
	ValorCentavos         int64
	Vencimento            time.Time
	CompoeCusto           bool
	CategoriaFinanceiraID uuid.UUID
}

func (c *Custo) Ativo() bool {
	return c.CanceladoEm == nil
}

func (c *Custo) EntraNoRateio() bool {
	return c.Ativo() && c.CompoeCusto
}

func (c *Custo) contaEncerrada() bool {
	return c.StatusContaAPagar != nil && contaAPagarEncerrada(*c.StatusContaAPagar)
}

func novoCusto(importacao *Importacao, usuarioID uuid.UUID, dados DadosCusto, agora time.Time) (*Custo, error) {
	if _, ok := descricoesDosTipos[dados.Tipo]; !ok {
		return nil, domain.ErroValidacao("tipo do custo inválido")
	}
	if dados.FornecedorID == uuid.Nil {
		return nil, domain.ErroValidacao("fornecedor do custo é obrigatório")
	}
	if dados.CategoriaFinanceiraID == uuid.Nil {
		return nil, domain.ErroValidacao("categoria financeira do custo é obrigatória")
	}
	if dados.ValorCentavos <= 0 {
		return nil, domain.ErroValidacao("valor do custo deve ser maior que zero")
	}
	if dados.Vencimento.IsZero() {
		return nil, domain.ErroValidacao(masculino("vencimento do custo").obrigatorio())
	}

	descricao := strings.TrimSpace(dados.Descricao)
	if descricao == "" {
		descricao = descricoesDosTipos[dados.Tipo]
	}
	descricao, err := textoLimitado(descricao, "descrição do custo", tamanhoMaximoDescricaoCusto)
	if err != nil {
		return nil, err
	}

	posterior := importacao.Status == StatusRecebida

	return &Custo{
		ID:                    uuid.Must(uuid.NewV7()),
		ImportacaoID:          importacao.ID,
		Tipo:                  dados.Tipo,
		Descricao:             descricao,
		FornecedorID:          dados.FornecedorID,
		ValorCentavos:         dados.ValorCentavos,
		Vencimento:            DataDe(dados.Vencimento),
		CompoeCusto:           dados.CompoeCusto && !posterior,
		Posterior:             posterior,
		CategoriaFinanceiraID: dados.CategoriaFinanceiraID,
		UsuarioID:             usuarioID,
		CriadoEm:              agora,
	}, nil
}
