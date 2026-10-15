package pedido_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
)

func TestNovoModeloNormalizaEValida(t *testing.T) {
	modelo, err := domainpedido.NovoModelo("  Importacao ")
	if err != nil || modelo != domainpedido.ModeloImportacao {
		t.Errorf("NovoModelo = %q, %v; esperado importacao", modelo, err)
	}
	if _, err := domainpedido.NovoModelo("consignado"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("modelo desconhecido = %v, esperado erro de validação", err)
	}
}

func TestCondicaoDoModelo(t *testing.T) {
	pronta := domainpedido.CondicaoDoModelo(domainpedido.ModeloProntaEntrega)
	if pronta.Entrada().PontosBase() != 0 || pronta.PrazoBoletoDias() == nil || *pronta.PrazoBoletoDias() != 30 || pronta.PrazoEntregaDias() != nil {
		t.Errorf("condição da pronta entrega = entrada %d, boleto %v, entrega %v",
			pronta.Entrada().PontosBase(), pronta.PrazoBoletoDias(), pronta.PrazoEntregaDias())
	}
	if pronta.PrazoLimiteDaEntrada(time.Now()) != nil {
		t.Error("pronta entrega não calcula prazo pela entrada")
	}

	importacao := domainpedido.CondicaoDoModelo(domainpedido.ModeloImportacao)
	if importacao.Entrada().PontosBase() != 3000 || importacao.PrazoBoletoDias() != nil || *importacao.PrazoEntregaDias() != 120 {
		t.Errorf("condição da importação = entrada %d, boleto %v, entrega %v",
			importacao.Entrada().PontosBase(), importacao.PrazoBoletoDias(), importacao.PrazoEntregaDias())
	}

	limite := importacao.PrazoLimiteDaEntrada(time.Date(2026, 10, 14, 22, 30, 0, 0, time.UTC))
	if limite == nil || !limite.Equal(time.Date(2027, 2, 11, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("prazo-limite = %v, esperado 2027-02-11", limite)
	}
}

func TestCondicaoDeNaoCompartilhaPonteiros(t *testing.T) {
	prazo := 45
	condicao := domainpedido.CondicaoDe(0, &prazo, nil)
	prazo = 10
	*condicao.PrazoBoletoDias() = 99

	if *condicao.PrazoBoletoDias() != 45 {
		t.Errorf("prazo do boleto = %d, esperado 45", *condicao.PrazoBoletoDias())
	}
}

func TestStatusDoPedido(t *testing.T) {
	casos := []struct {
		de, para domainpedido.Status
		pode     bool
	}{
		{domainpedido.StatusProposta, domainpedido.StatusConfirmado, true},
		{domainpedido.StatusProposta, domainpedido.StatusEntregue, false},
		{domainpedido.StatusConfirmado, domainpedido.StatusParcialmenteEntregue, true},
		{domainpedido.StatusConfirmado, domainpedido.StatusEntregue, true},
		{domainpedido.StatusParcialmenteEntregue, domainpedido.StatusConfirmado, false},
		{domainpedido.StatusParcialmenteEntregue, domainpedido.StatusCancelado, true},
		{domainpedido.StatusEntregue, domainpedido.StatusCancelado, false},
		{domainpedido.StatusCancelado, domainpedido.StatusProposta, false},
	}

	for _, caso := range casos {
		if got := caso.de.PodeTransicionarPara(caso.para); got != caso.pode {
			t.Errorf("%s → %s = %v, esperado %v", caso.de, caso.para, got, caso.pode)
		}
	}
	if !domainpedido.StatusEntregue.Final() || !domainpedido.StatusCancelado.Final() || domainpedido.StatusConfirmado.Final() {
		t.Error("só entregue e cancelado são finais")
	}
	if _, err := domainpedido.NovoStatus("aberto"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status desconhecido = %v, esperado erro de validação", err)
	}
}

func TestNovoFrete(t *testing.T) {
	valor := int64(15000)
	negativo := int64(-1)

	casos := []struct {
		nome      string
		tipo      domainpedido.TipoFrete
		centavos  *int64
		sentinela error
	}{
		{"cif com valor", domainpedido.TipoFreteCIF, &valor, nil},
		{"fob sem valor", domainpedido.TipoFreteFOB, nil, nil},
		{"fob com valor", domainpedido.TipoFreteFOB, &valor, nil},
		{"cif sem valor", domainpedido.TipoFreteCIF, nil, domain.ErrValidacao},
		{"valor negativo", domainpedido.TipoFreteFOB, &negativo, domain.ErrValidacao},
		{"tipo desconhecido", domainpedido.TipoFreteDe("cpt"), nil, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			frete, err := domainpedido.NovoFrete(caso.tipo, caso.centavos)
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.sentinela == nil && frete.Tipo() != caso.tipo {
				t.Errorf("tipo = %q, esperado %q", frete.Tipo(), caso.tipo)
			}
		})
	}

	tipo, err := domainpedido.NovoTipoFrete(" CIF ")
	if err != nil || tipo != domainpedido.TipoFreteCIF {
		t.Errorf("NovoTipoFrete = %q, %v; esperado cif", tipo, err)
	}
}

func TestPrecoVezesRecusaEstouro(t *testing.T) {
	if _, err := domainpedido.NovoPreco(-1); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("preço negativo = %v, esperado erro de validação", err)
	}

	total, err := domainpedido.PrecoDe(1250).Vezes(4)
	if err != nil || total != 5000 {
		t.Errorf("1250 × 4 = %d, %v; esperado 5000", total, err)
	}
	if _, err := domainpedido.PrecoDe(math.MaxInt64 / 2).Vezes(3); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("estouro = %v, esperado erro de validação", err)
	}
}

func TestDataDeIgnoraHoraEFuso(t *testing.T) {
	momento := time.Date(2026, 10, 14, 23, 59, 0, 0, time.FixedZone("BRT", -3*60*60))

	if data := domainpedido.DataDe(momento); !data.Equal(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("DataDe = %v, esperado 2026-10-14 UTC", data)
	}
}

func TestHojeUsaOFusoDoGrupo(t *testing.T) {
	noiteEmBrasilia := time.Date(2026, 10, 15, 2, 30, 0, 0, time.UTC)

	if hoje := domainpedido.Hoje(noiteEmBrasilia); !hoje.Equal(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Hoje às 23h30 de Brasília = %v, esperado 2026-10-14", hoje)
	}
	if inicio := domainpedido.InicioDoDia(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)); !inicio.Equal(time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("InicioDoDia = %v, esperado 03:00 UTC", inicio)
	}

	pedido := confirmadoDeTeste(t, domainpedido.ModeloImportacao)
	if err := pedido.RegistrarEntrada(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), 100, noiteEmBrasilia); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("entrada com a data de amanhã em Brasília = %v, esperado erro de validação", err)
	}
}
