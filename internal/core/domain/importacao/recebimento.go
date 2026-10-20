package importacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type RecebimentoInput struct {
	RecebidaEm  time.Time
	LocalID     uuid.UUID
	CambioMicro *int64
	Recebidas   map[uuid.UUID]int
}

func (i *Importacao) Receber(input RecebimentoInput, agora time.Time) (*CalculoCusto, error) {
	if !i.Status.PodeTransicionarPara(StatusRecebida) {
		return nil, erroTransicao(i.Status, StatusRecebida)
	}
	if input.LocalID == uuid.Nil {
		return nil, domain.ErroValidacao(masculino("local do recebimento").obrigatorio())
	}

	recebidaEm, err := dataDoFato(input.RecebidaEm, agora, feminino("data do recebimento"))
	if err != nil {
		return nil, err
	}
	if recebidaEm.Before(*i.DesembaracadaEm) {
		return nil, domain.ErroValidacao("data do recebimento não pode ser anterior ao desembaraço em " + formatarData(*i.DesembaracadaEm))
	}

	if err := i.exigirContasVigentesNoCusto(); err != nil {
		return nil, err
	}

	recebidas, err := i.quantidadesRecebidas(input.Recebidas)
	if err != nil {
		return nil, err
	}

	calculo, err := i.CalcularCusto(input.CambioMicro, recebidas)
	if err != nil {
		return nil, err
	}

	for indice, item := range i.Itens {
		quantidade := recebidas[item.ID]
		item.QuantidadeRecebida = &quantidade
		item.CustoUnitarioCentavos = calculo.Itens[indice].CustoUnitarioCentavos
	}
	localID := input.LocalID
	i.Status = StatusRecebida
	i.RecebidaEm = &recebidaEm
	i.LocalRecebimentoID = &localID
	i.Cambio = calculo.CambioInformado
	i.AtualizadoEm = agora

	return calculo, nil
}

func (i *Importacao) quantidadesRecebidas(informadas map[uuid.UUID]int) (map[uuid.UUID]int, error) {
	for itemID := range informadas {
		if i.Item(itemID) == nil {
			return nil, domain.ErroValidacao("item recebido não pertence à importação")
		}
	}

	recebidas := make(map[uuid.UUID]int, len(i.Itens))
	total := 0
	for _, item := range i.Itens {
		quantidade, informada := informadas[item.ID]
		if !informada {
			quantidade = item.Quantidade
		}
		if quantidade < 0 || quantidade > item.Quantidade {
			return nil, domain.ErroValidacao("quantidade recebida deve estar entre zero e a quantidade do item")
		}
		recebidas[item.ID] = quantidade
		total += quantidade
	}
	if total == 0 {
		return nil, domain.ErroValidacao("nada foi recebido; cancele a importação em vez de receber")
	}

	return recebidas, nil
}

func (i *Importacao) exigirContasVigentesNoCusto() error {
	for _, parcela := range i.Parcelas {
		if parcela.Status == StatusParcelaCambioFechado && parcela.contaEncerrada() {
			return domain.ErroConflito("a conta a pagar da parcela \"" + parcela.Descricao +
				"\" foi cancelada ou rejeitada; reabra a parcela antes de receber")
		}
	}
	for _, custo := range i.Custos {
		if custo.EntraNoRateio() && custo.contaEncerrada() {
			return domain.ErroConflito("a conta a pagar do custo \"" + custo.Descricao +
				"\" foi cancelada ou rejeitada; cancele o custo antes de receber")
		}
	}

	return nil
}
