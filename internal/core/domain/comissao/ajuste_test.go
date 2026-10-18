package comissao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

func dadosDeAjuste() domaincomissao.DadosAjuste {
	return domaincomissao.DadosAjuste{
		EmpresaID:       uuid.New(),
		RepresentanteID: uuid.New(),
		Competencia:     "2026-10",
		ValorCentavos:   -1500,
		Motivo:          "  devolução de 1 peça  ",
	}
}

func TestNovoAjusteAceitaValorNegativoEAparaOMotivo(t *testing.T) {
	pedidoID := uuid.New()
	dados := dadosDeAjuste()
	dados.PedidoID = &pedidoID

	ajuste, err := domaincomissao.NovoAjuste(dados, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovoAjuste: %v", err)
	}
	if ajuste.ValorCentavos != -1500 || ajuste.Motivo != "devolução de 1 peça" || ajuste.Competencia.String() != "2026-10" ||
		ajuste.PedidoID == nil || *ajuste.PedidoID != pedidoID || ajuste.ID == uuid.Nil {
		t.Fatalf("ajuste = %+v", ajuste)
	}

	dados.PedidoID = &uuid.Nil
	semPedido, err := domaincomissao.NovoAjuste(dados, uuid.New(), agora)
	if err != nil || semPedido.PedidoID != nil {
		t.Fatalf("pedido nulo = %+v, %v", semPedido, err)
	}
}

func TestNovoAjusteUsaOMesCorrenteEmBrasilia(t *testing.T) {
	dados := dadosDeAjuste()
	dados.Competencia = "2026-11"
	viradaEmUTC := time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)

	_, err := domaincomissao.NovoAjuste(dados, uuid.New(), viradaEmUTC)
	esperarErro(t, err, domain.ErrValidacao, "posterior ao mês corrente")

	if _, err := domaincomissao.NovoAjuste(dados, uuid.New(), viradaEmUTC.Add(3*time.Hour)); err != nil {
		t.Fatalf("novembro já começou em Brasília: %v", err)
	}
}

func TestNovoAjusteRecusaDadosInvalidos(t *testing.T) {
	casos := []struct {
		nome    string
		ajustar func(*domaincomissao.DadosAjuste)
		trecho  string
	}{
		{"sem empresa", func(d *domaincomissao.DadosAjuste) { d.EmpresaID = uuid.Nil }, "empresa do ajuste"},
		{"sem representante", func(d *domaincomissao.DadosAjuste) { d.RepresentanteID = uuid.Nil }, "representante do ajuste"},
		{"valor zero", func(d *domaincomissao.DadosAjuste) { d.ValorCentavos = 0 }, "não pode ser zero"},
		{"competência inválida", func(d *domaincomissao.DadosAjuste) { d.Competencia = "10/2026" }, "AAAA-MM"},
		{"competência futura", func(d *domaincomissao.DadosAjuste) { d.Competencia = "2026-11" }, "posterior ao mês corrente"},
		{"sem motivo", func(d *domaincomissao.DadosAjuste) { d.Motivo = "   " }, "motivo do ajuste é obrigatório"},
		{"motivo longo", func(d *domaincomissao.DadosAjuste) { d.Motivo = strings.Repeat("a", 1001) }, "no máximo 1000"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeAjuste()
			caso.ajustar(&dados)

			_, err := domaincomissao.NovoAjuste(dados, uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}
