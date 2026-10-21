package divida

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const taxaMaximaBP = 100000

type PeriodoTaxa string

const (
	PeriodoMensal PeriodoTaxa = "mensal"
	PeriodoAnual  PeriodoTaxa = "anual"
)

type Indexador string

const (
	IndexadorPrefixado Indexador = "prefixado"
	IndexadorCDI       Indexador = "cdi"
	IndexadorSelic     Indexador = "selic"
	IndexadorIPCA      Indexador = "ipca"
	IndexadorTR        Indexador = "tr"
	IndexadorOutro     Indexador = "outro"
)

type Taxa struct {
	pontosBase int
	periodo    PeriodoTaxa
}

func NovaTaxa(pontosBase *int, periodo string) (*Taxa, error) {
	periodo = strings.ToLower(strings.TrimSpace(periodo))
	if pontosBase == nil && periodo == "" {
		return nil, nil
	}
	if pontosBase == nil || periodo == "" {
		return nil, domain.ErroValidacao("taxa e período da taxa devem ser informados juntos")
	}
	if *pontosBase < 0 || *pontosBase > taxaMaximaBP {
		return nil, domain.ErroValidacao("taxa deve estar entre 0 e 100000 pontos-base")
	}

	switch PeriodoTaxa(periodo) {
	case PeriodoMensal, PeriodoAnual:
	default:
		return nil, domain.ErroValidacao("período da taxa deve ser mensal ou anual")
	}

	return &Taxa{pontosBase: *pontosBase, periodo: PeriodoTaxa(periodo)}, nil
}

func TaxaDe(pontosBase int, periodo string) Taxa {
	return Taxa{pontosBase: pontosBase, periodo: PeriodoTaxa(periodo)}
}

func (t Taxa) PontosBase() int {
	return t.pontosBase
}

func (t Taxa) Periodo() PeriodoTaxa {
	return t.periodo
}

func (p PeriodoTaxa) String() string {
	return string(p)
}

func NovoIndexador(valor string) (Indexador, error) {
	indexador := Indexador(strings.ToLower(strings.TrimSpace(valor)))
	if indexador == "" {
		return IndexadorPrefixado, nil
	}

	switch indexador {
	case IndexadorPrefixado, IndexadorCDI, IndexadorSelic, IndexadorIPCA, IndexadorTR, IndexadorOutro:
		return indexador, nil
	default:
		return "", domain.ErroValidacao("indexador deve ser prefixado, cdi, selic, ipca, tr ou outro")
	}
}

func IndexadorDe(valor string) Indexador {
	return Indexador(valor)
}

func (i Indexador) String() string {
	return string(i)
}
