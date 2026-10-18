package comissao_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

var agora = time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC)

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func representanteDeTeste() *domaincliente.Representante {
	return &domaincliente.Representante{
		ID:                    uuid.New(),
		Nome:                  "Carlos",
		ComissaoProntaEntrega: domaincliente.PercentualDe(250),
		ComissaoImportacao:    domaincliente.PercentualDe(400),
		Ativo:                 true,
	}
}

func entregaDeTeste(representanteID uuid.UUID) domaincomissao.EntregaParaApurar {
	return domaincomissao.EntregaParaApurar{
		EntregaID:       uuid.New(),
		PedidoID:        uuid.New(),
		EmpresaID:       uuid.New(),
		RepresentanteID: representanteID,
		Modelo:          domainpedido.ModeloProntaEntrega,
		SaiuEm:          time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Itens: []domaincomissao.ItemSaido{
			{Quantidade: 10, PrecoUnitarioCentavos: 1500},
			{Quantidade: 3, PrecoUnitarioCentavos: 2333},
		},
	}
}

func TestApuracaoSemPercentualCopiaOPadraoDoRepresentantePeloModelo(t *testing.T) {
	representante := representanteDeTeste()
	entrega := entregaDeTeste(representante.ID)

	apuracao, err := domaincomissao.NovaApuracao(entrega, representante, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}

	if !apuracao.PercentualNovo || apuracao.Percentual.Origem != domaincomissao.OrigemPercentualPadrao ||
		apuracao.Percentual.Percentual.PontosBase() != 250 || apuracao.Percentual.RepresentanteID != representante.ID {
		t.Fatalf("percentual = %+v", apuracao.Percentual)
	}

	comissao := apuracao.Comissao
	if comissao.BaseCentavos != 21999 || comissao.ValorCentavos != 550 || comissao.Percentual.PontosBase() != 250 {
		t.Fatalf("base %d, valor %d, percentual %d", comissao.BaseCentavos, comissao.ValorCentavos, comissao.Percentual.PontosBase())
	}
	if comissao.Competencia.String() != "2026-09" || comissao.EntregaID != entrega.EntregaID || comissao.EmpresaID != entrega.EmpresaID {
		t.Fatalf("comissão = %+v", comissao)
	}
}

func TestApuracaoDeImportacaoUsaOPercentualDeImportacao(t *testing.T) {
	representante := representanteDeTeste()
	entrega := entregaDeTeste(representante.ID)
	entrega.Modelo = domainpedido.ModeloImportacao

	apuracao, err := domaincomissao.NovaApuracao(entrega, representante, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	if apuracao.Comissao.ValorCentavos != 880 {
		t.Fatalf("valor = %d", apuracao.Comissao.ValorCentavos)
	}
}

func TestApuracaoUsaOPercentualVigenteDoPedido(t *testing.T) {
	representante := representanteDeTeste()
	entrega := entregaDeTeste(representante.ID)
	manual, err := domaincomissao.NovoPercentualManual(entrega.PedidoID, representante.ID, uuid.New(), 500, agora)
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	entrega.Percentual = manual

	apuracao, err := domaincomissao.NovaApuracao(entrega, nil, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	if apuracao.PercentualNovo || apuracao.Comissao.ValorCentavos != 1100 {
		t.Fatalf("apuração = %+v, valor %d", apuracao, apuracao.Comissao.ValorCentavos)
	}
}

func TestApuracaoIgnoraPercentualDeOutroRepresentante(t *testing.T) {
	representante := representanteDeTeste()
	entrega := entregaDeTeste(representante.ID)
	antigo, err := domaincomissao.NovoPercentualManual(entrega.PedidoID, uuid.New(), uuid.New(), 900, agora)
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	entrega.Percentual = antigo

	apuracao, err := domaincomissao.NovaApuracao(entrega, representante, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	if !apuracao.PercentualNovo || apuracao.Comissao.Percentual.PontosBase() != 250 {
		t.Fatalf("apuração = %+v", apuracao.Percentual)
	}

	_, err = domaincomissao.NovaApuracao(entrega, nil, uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "representante do pedido não encontrado")
}

func TestApuracaoComPercentualZeroGeraComissaoZero(t *testing.T) {
	representante := representanteDeTeste()
	representante.ComissaoProntaEntrega = domaincliente.PercentualDe(0)

	apuracao, err := domaincomissao.NovaApuracao(entregaDeTeste(representante.ID), representante, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	if apuracao.Comissao.ValorCentavos != 0 || apuracao.Comissao.BaseCentavos != 21999 {
		t.Fatalf("comissão = %+v", apuracao.Comissao)
	}
}

func TestApuracaoArredondaMeioCentavoParaCima(t *testing.T) {
	representante := representanteDeTeste()
	representante.ComissaoProntaEntrega = domaincliente.PercentualDe(5000)
	entrega := entregaDeTeste(representante.ID)
	entrega.Itens = []domaincomissao.ItemSaido{{Quantidade: 1, PrecoUnitarioCentavos: 3}}

	apuracao, err := domaincomissao.NovaApuracao(entrega, representante, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	if apuracao.Comissao.ValorCentavos != 2 {
		t.Fatalf("valor = %d", apuracao.Comissao.ValorCentavos)
	}
}

func TestApuracaoRecusaEntregaInvalida(t *testing.T) {
	representante := representanteDeTeste()
	casos := []struct {
		nome    string
		ajustar func(*domaincomissao.EntregaParaApurar)
		trecho  string
	}{
		{"sem representante", func(e *domaincomissao.EntregaParaApurar) { e.RepresentanteID = uuid.Nil }, "sem representante"},
		{"sem saída", func(e *domaincomissao.EntregaParaApurar) { e.SaiuEm = time.Time{} }, "sem data de saída"},
		{"sem itens", func(e *domaincomissao.EntregaParaApurar) { e.Itens = nil }, "sem itens"},
		{"preço zero", func(e *domaincomissao.EntregaParaApurar) { e.Itens[0].PrecoUnitarioCentavos = 0 }, "quantidade ou preço"},
		{"estouro", func(e *domaincomissao.EntregaParaApurar) {
			e.Itens = []domaincomissao.ItemSaido{{Quantidade: 2, PrecoUnitarioCentavos: math.MaxInt64 / 2}, {Quantidade: 1, PrecoUnitarioCentavos: 10}}
		}, "excede o limite"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			entrega := entregaDeTeste(representante.ID)
			caso.ajustar(&entrega)

			_, err := domaincomissao.NovaApuracao(entrega, representante, uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}

func TestPercentualManualRedefineEConfereVersao(t *testing.T) {
	percentual, err := domaincomissao.NovoPercentualManual(uuid.New(), uuid.New(), uuid.New(), 300, agora)
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	if percentual.Versao != 1 || percentual.Origem != domaincomissao.OrigemPercentualManual {
		t.Fatalf("percentual = %+v", percentual)
	}

	esperarErro(t, percentual.ConferirVersao(0), domain.ErrValidacao, "versão")
	esperarErro(t, percentual.ConferirVersao(2), domain.ErrConflito, "recarregue")

	usuarioID := uuid.New()
	if err := percentual.Redefinir(450, usuarioID, agora.Add(time.Hour)); err != nil {
		t.Fatalf("Redefinir: %v", err)
	}
	if percentual.Percentual.PontosBase() != 450 || percentual.UsuarioID != usuarioID {
		t.Fatalf("redefinido = %+v", percentual)
	}

	esperarErro(t, percentual.Redefinir(10001, usuarioID, agora), domain.ErrValidacao, "10000 pontos-base")
	_, err = domaincomissao.NovoPercentualManual(uuid.New(), uuid.Nil, uuid.New(), 300, agora)
	esperarErro(t, err, domain.ErrValidacao, "sem representante")
}

func TestCompetenciaSeguinteViraOAno(t *testing.T) {
	competencia, err := domaincomissao.NovaCompetencia("2026-12")
	if err != nil {
		t.Fatalf("NovaCompetencia: %v", err)
	}
	if competencia.Seguinte().String() != "2027-01" {
		t.Fatalf("seguinte = %s", competencia.Seguinte())
	}
}
