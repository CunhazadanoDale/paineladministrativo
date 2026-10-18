package notafiscal

import (
	"fmt"
	"sort"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

func (n *NotaFiscal) PodeTerTitulos() bool {
	return n.Natureza.GeraTitulo() && !n.EntreEmpresas && n.ClienteID != nil && len(n.Duplicatas) > 0
}

func (n *NotaFiscal) GeraTitulos() bool {
	return n.Status == StatusAutorizada && n.PodeTerTitulos()
}

func (n *NotaFiscal) NumeroDoTitulo() string {
	return fmt.Sprintf("NF %d-%d", n.Serie, n.Numero)
}

func (n *NotaFiscal) ParcelasDosTitulos() []domainreceber.ParcelaDeOrigem {
	duplicatas := make([]*Duplicata, len(n.Duplicatas))
	copy(duplicatas, n.Duplicatas)
	sort.SliceStable(duplicatas, func(i, j int) bool {
		if !duplicatas[i].Vencimento.Equal(duplicatas[j].Vencimento) {
			return duplicatas[i].Vencimento.Before(duplicatas[j].Vencimento)
		}
		return duplicatas[i].Numero < duplicatas[j].Numero
	})

	parcelas := make([]domainreceber.ParcelaDeOrigem, 0, len(duplicatas))
	for indice, duplicata := range duplicatas {
		parcelas = append(parcelas, domainreceber.ParcelaDeOrigem{
			OrigemID:      duplicata.ID,
			Numero:        n.NumeroDoTitulo(),
			Parcela:       indice + 1,
			Vencimento:    duplicata.Vencimento,
			ValorCentavos: duplicata.ValorCentavos,
		})
	}

	return parcelas
}

func (n *NotaFiscal) DuplicataIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(n.Duplicatas))
	for _, duplicata := range n.Duplicatas {
		ids = append(ids, duplicata.ID)
	}

	return ids
}

func (n *NotaFiscal) MotivoDoCancelamento() string {
	return "cancelamento da NF-e " + n.Chave.Valor()
}
