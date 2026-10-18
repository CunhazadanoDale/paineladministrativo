package notafiscal_test

import (
	"testing"
	"time"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	"github.com/google/uuid"
)

func TestParcelasDosTitulosEmOrdemDeVencimento(t *testing.T) {
	nota := notaDoTeste(t)
	nota.Duplicatas[0], nota.Duplicatas[1] = nota.Duplicatas[1], nota.Duplicatas[0]

	parcelas := nota.ParcelasDosTitulos()

	if len(parcelas) != 2 {
		t.Fatalf("parcelas = %d, esperado 2", len(parcelas))
	}
	primeira, segunda := parcelas[0], parcelas[1]
	if primeira.Parcela != 1 || !primeira.Vencimento.Equal(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)) ||
		primeira.Numero != "NF 1-123" || primeira.ValorCentavos != 54425 || primeira.OrigemID == uuid.Nil {
		t.Errorf("primeira parcela = %+v", primeira)
	}
	if segunda.Parcela != 2 || segunda.OrigemID != nota.Duplicatas[0].ID {
		t.Errorf("segunda parcela = %+v", segunda)
	}
	if ids := nota.DuplicataIDs(); len(ids) != 2 {
		t.Errorf("ids das duplicatas = %v", ids)
	}
	if nota.MotivoDoCancelamento() != "cancelamento da NF-e "+nota.Chave.Valor() {
		t.Errorf("motivo = %q", nota.MotivoDoCancelamento())
	}
}

func TestGeraTitulosSoNaVendaAutorizadaComDuplicatas(t *testing.T) {
	if !notaDoTeste(t).GeraTitulos() {
		t.Error("venda autorizada com duplicatas deveria gerar títulos")
	}

	casos := map[string]func(*domainnotafiscal.NotaFiscal){
		"remessa":        func(n *domainnotafiscal.NotaFiscal) { n.Natureza = domainnotafiscal.NaturezaRemessa },
		"devolução":      func(n *domainnotafiscal.NotaFiscal) { n.Natureza = domainnotafiscal.NaturezaDevolucao },
		"cancelada":      func(n *domainnotafiscal.NotaFiscal) { n.Status = domainnotafiscal.StatusCancelada },
		"sem duplicatas": func(n *domainnotafiscal.NotaFiscal) { n.Duplicatas = nil },
		"entre empresas": func(n *domainnotafiscal.NotaFiscal) {
			n.EntreEmpresas = true
			n.Natureza = domainnotafiscal.NaturezaTransferencia
			n.ClienteID = nil
		},
	}
	for nome, ajuste := range casos {
		nota := notaDoTeste(t)
		ajuste(nota)
		if nota.GeraTitulos() {
			t.Errorf("%s não deveria gerar títulos", nome)
		}
	}
}
