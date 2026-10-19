package importacao

import (
	"math/big"
	"strconv"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

type CustoDoItem struct {
	ItemID                  uuid.UUID
	ProdutoID               uuid.UUID
	Quantidade              int
	Valor                   ValorMoeda
	ValorMercadoriaCentavos int64
	CustosCentavos          int64
	CustoTotalCentavos      int64
	CustoUnitarioCentavos   *int64
}

type CalculoCusto struct {
	Valor                      ValorMoeda
	ParcelasFechadas           ValorMoeda
	ParcelasFechadasCentavos   int64
	SemCambioFechado           ValorMoeda
	CambioInformado            *Cambio
	CambioMedio                Cambio
	ValorMercadoriaCentavos    int64
	CustosCentavos             int64
	CustosForaDoRateioCentavos int64
	Itens                      []CustoDoItem
}

func (i *Importacao) CalcularCusto(cambioInformado *int64, recebidas map[uuid.UUID]int) (*CalculoCusto, error) {
	calculo := &CalculoCusto{Valor: i.ValorTotal}
	for _, parcela := range i.Parcelas {
		if parcela.Status == StatusParcelaCambioFechado {
			calculo.ParcelasFechadas = ValorMoedaDe(calculo.ParcelasFechadas.Centesimos() + parcela.Valor.Centesimos())
			calculo.ParcelasFechadasCentavos += *parcela.ValorCentavos
		}
	}
	calculo.SemCambioFechado = ValorMoedaDe(max(calculo.Valor.Centesimos()-calculo.ParcelasFechadas.Centesimos(), 0))

	calculo.ValorMercadoriaCentavos = calculo.ParcelasFechadasCentavos
	if calculo.SemCambioFechado.Centesimos() > 0 {
		cambio, err := i.cambioDoRestante(cambioInformado, calculo.SemCambioFechado)
		if err != nil {
			return nil, err
		}
		restante, err := cambio.EmReais(calculo.SemCambioFechado)
		if err != nil {
			return nil, err
		}
		calculo.CambioInformado = &cambio
		calculo.ValorMercadoriaCentavos += restante
	}
	calculo.CambioMedio = cambioMedio(calculo.ValorMercadoriaCentavos, calculo.Valor.Centesimos())

	for _, custo := range i.Custos {
		switch {
		case custo.EntraNoRateio():
			calculo.CustosCentavos += custo.ValorCentavos
		case custo.Ativo():
			calculo.CustosForaDoRateioCentavos += custo.ValorCentavos
		}
	}

	if err := calculo.distribuir(i.Itens, recebidas); err != nil {
		return nil, err
	}

	return calculo, nil
}

func (i *Importacao) cambioDoRestante(informado *int64, restante ValorMoeda) (Cambio, error) {
	if i.Moeda == domainfornecedor.MoedaReal {
		return CambioDoReal(), nil
	}
	if informado == nil {
		return Cambio{}, domain.ErroValidacao("informe o câmbio para os " + strconv.FormatInt(restante.Centesimos(), 10) +
			" centésimos de " + i.Moeda.String() + " ainda sem câmbio fechado")
	}

	return NovoCambio(*informado)
}

func (c *CalculoCusto) distribuir(itens []*Item, recebidas map[uuid.UUID]int) error {
	mercadoria, err := partilhar(c.ValorMercadoriaCentavos, pesosPeloValor(itens))
	if err != nil {
		return err
	}

	quantidades := make([]int, len(itens))
	pesosDosCustos := make([]int64, len(itens))
	for indice, item := range itens {
		quantidades[indice] = item.Quantidade
		if recebidas != nil {
			quantidades[indice] = recebidas[item.ID]
		}
		if quantidades[indice] > 0 {
			pesosDosCustos[indice] = mercadoria[indice]
		}
	}
	if soma(pesosDosCustos) == 0 {
		for indice := range itens {
			pesosDosCustos[indice] = int64(quantidades[indice])
		}
	}
	custos, err := partilhar(c.CustosCentavos, pesosDosCustos)
	if err != nil {
		return err
	}

	c.Itens = make([]CustoDoItem, 0, len(itens))
	for indice, item := range itens {
		linha := CustoDoItem{
			ItemID:                  item.ID,
			ProdutoID:               item.ProdutoID,
			Quantidade:              quantidades[indice],
			Valor:                   item.Valor,
			ValorMercadoriaCentavos: mercadoria[indice],
			CustosCentavos:          custos[indice],
			CustoTotalCentavos:      mercadoria[indice] + custos[indice],
		}
		if linha.Quantidade > 0 {
			unitario, err := dividirArredondando(big.NewInt(linha.CustoTotalCentavos), big.NewInt(int64(linha.Quantidade)))
			if err != nil {
				return err
			}
			linha.CustoUnitarioCentavos = &unitario
		}
		c.Itens = append(c.Itens, linha)
	}

	return nil
}

func pesosPeloValor(itens []*Item) []int64 {
	pesos := make([]int64, len(itens))
	for indice, item := range itens {
		pesos[indice] = item.Valor.Centesimos()
	}
	if soma(pesos) == 0 {
		for indice, item := range itens {
			pesos[indice] = int64(item.Quantidade)
		}
	}

	return pesos
}

func partilhar(total int64, pesos []int64) ([]int64, error) {
	partes := make([]int64, len(pesos))
	base := soma(pesos)
	if total == 0 || base == 0 {
		return partes, nil
	}

	ultimo := -1
	for indice, peso := range pesos {
		if peso > 0 {
			ultimo = indice
		}
	}

	var distribuido int64
	for indice, peso := range pesos {
		if indice == ultimo {
			partes[indice] = total - distribuido
			break
		}
		parte := new(big.Int).Quo(new(big.Int).Mul(big.NewInt(total), big.NewInt(peso)), big.NewInt(base))
		if !parte.IsInt64() {
			return nil, domain.ErroValidacao("valor em reais acima do limite permitido")
		}
		partes[indice] = parte.Int64()
		distribuido += partes[indice]
	}

	return partes, nil
}

func soma(valores []int64) int64 {
	var total int64
	for _, valor := range valores {
		total += valor
	}

	return total
}
