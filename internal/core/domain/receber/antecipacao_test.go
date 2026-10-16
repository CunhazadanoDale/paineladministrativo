package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

func dadosDoBordero(liquido int64) domainreceber.DadosAntecipacao {
	numero := " B-77 "

	return domainreceber.DadosAntecipacao{
		FornecedorID:         uuid.New(),
		Numero:               &numero,
		OperadoEm:            data(2026, 10, 14),
		TaxaBP:               250,
		ValorLiquidoCentavos: liquido,
		ContaBancariaID:      uuid.New(),
	}
}

func titulosDaEmpresa(t *testing.T, quantos int) []*domainreceber.Titulo {
	t.Helper()

	empresa := uuid.New()
	titulos := make([]*domainreceber.Titulo, 0, quantos)
	for range quantos {
		titulo := novoTituloDeTeste(t)
		titulo.EmpresaID = empresa
		titulos = append(titulos, titulo)
	}

	return titulos
}

func TestNovaAntecipacaoAntecipaOsTitulos(t *testing.T) {
	titulos := titulosDaEmpresa(t, 2)
	if err := titulos[1].EnviarAoBanco(uuid.New(), agora); err != nil {
		t.Fatal(err)
	}
	titulos[1].RecebidoCentavos = 50000

	antecipacao, historicos, err := domainreceber.NovaAntecipacao(titulos[0].EmpresaID, uuid.New(), dadosDoBordero(240000), titulos, agora)
	if err != nil {
		t.Fatalf("NovaAntecipacao falhou: %v", err)
	}
	if antecipacao.ValorBrutoCentavos != 250000 || antecipacao.EncargosCentavos() != 10000 || *antecipacao.Numero != "B-77" ||
		antecipacao.Status != domainreceber.StatusAntecipacaoAtiva || len(antecipacao.Titulos) != 2 {
		t.Fatalf("antecipação = %+v", antecipacao)
	}
	vinculo := antecipacao.Vinculo(titulos[1].ID)
	if vinculo == nil || vinculo.ValorCentavos != 100000 || vinculo.StatusAnterior != domainreceber.StatusEmBanco || !vinculo.Vigente {
		t.Fatalf("vínculo = %+v", vinculo)
	}
	for _, titulo := range titulos {
		if titulo.Status != domainreceber.StatusAntecipado {
			t.Fatalf("título = %+v", titulo)
		}
	}
	if len(historicos) != 2 || historicos[1].Evento != domainreceber.EventoAntecipado || *historicos[1].DeStatus != domainreceber.StatusEmBanco ||
		historicos[1].Descricao != "borderô B-77" {
		t.Fatalf("históricos = %+v", historicos)
	}
}

func TestNovaAntecipacaoValida(t *testing.T) {
	casos := map[string]struct {
		preparar func([]*domainreceber.Titulo, *domainreceber.DadosAntecipacao) []*domainreceber.Titulo
		erro     error
		trecho   string
	}{
		"sem títulos": {func(_ []*domainreceber.Titulo, _ *domainreceber.DadosAntecipacao) []*domainreceber.Titulo { return nil }, domain.ErrValidacao, "pelo menos um título"},
		"repetido": {func(ts []*domainreceber.Titulo, _ *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			return []*domainreceber.Titulo{ts[0], ts[0]}
		}, domain.ErrValidacao, "título repetido"},
		"outra empresa": {func(ts []*domainreceber.Titulo, _ *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			ts[1].EmpresaID = uuid.New()
			return ts
		}, domain.ErrValidacao, "de outra empresa"},
		"já antecipado": {func(ts []*domainreceber.Titulo, _ *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			ts[1].Status = domainreceber.StatusAntecipado
			return ts
		}, domain.ErrConflito, "carteira ou em banco"},
		"origem cancelada": {func(ts []*domainreceber.Titulo, _ *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			momento := agora
			ts[0].OrigemCanceladaEm = &momento
			return ts
		}, domain.ErrConflito, "origem cancelada"},
		"líquido maior que o bruto": {func(ts []*domainreceber.Titulo, d *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			d.ValorLiquidoCentavos = 300001
			return ts
		}, domain.ErrValidacao, "no máximo o bruto (300000 centavos)"},
		"taxa inválida": {func(ts []*domainreceber.Titulo, d *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			d.TaxaBP = 10001
			return ts
		}, domain.ErrValidacao, "taxa"},
		"data futura": {func(ts []*domainreceber.Titulo, d *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			d.OperadoEm = data(2026, 10, 16)
			return ts
		}, domain.ErrValidacao, "no futuro"},
		"sem fomento": {func(ts []*domainreceber.Titulo, d *domainreceber.DadosAntecipacao) []*domainreceber.Titulo {
			d.FornecedorID = uuid.Nil
			return ts
		}, domain.ErrValidacao, "fomento"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			titulos := titulosDaEmpresa(t, 2)
			dados := dadosDoBordero(290000)
			usados := caso.preparar(titulos, &dados)

			_, _, err := domainreceber.NovaAntecipacao(titulos[0].EmpresaID, uuid.New(), dados, usados, agora)
			esperarErro(t, err, caso.erro, caso.trecho)
		})
	}
}

func borderoDeTeste(t *testing.T) (*domainreceber.Antecipacao, map[uuid.UUID]*domainreceber.Titulo) {
	t.Helper()

	titulos := titulosDaEmpresa(t, 2)
	antecipacao, _, err := domainreceber.NovaAntecipacao(titulos[0].EmpresaID, uuid.New(), dadosDoBordero(290000), titulos, agora)
	if err != nil {
		t.Fatal(err)
	}

	return antecipacao, map[uuid.UUID]*domainreceber.Titulo{titulos[0].ID: titulos[0], titulos[1].ID: titulos[1]}
}

func TestEstornarAntecipacaoDevolveOsTitulos(t *testing.T) {
	antecipacao, titulos := borderoDeTeste(t)

	_, err := antecipacao.Estornar(" ", titulos, uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "motivo do estorno da antecipação")

	historicos, err := antecipacao.Estornar("lançado no fomento errado", titulos, uuid.New(), agora)
	if err != nil {
		t.Fatalf("Estornar falhou: %v", err)
	}
	if antecipacao.Status != domainreceber.StatusAntecipacaoEstornada || *antecipacao.EstornoMotivo != "lançado no fomento errado" || antecipacao.EstornadoEm == nil {
		t.Fatalf("antecipação = %+v", antecipacao)
	}
	for _, titulo := range titulos {
		if titulo.Status != domainreceber.StatusCarteira || antecipacao.Vinculo(titulo.ID).Vigente {
			t.Fatalf("título = %+v", titulo)
		}
	}
	if len(historicos) != 2 || historicos[0].Evento != domainreceber.EventoAntecipacaoEstornada {
		t.Fatalf("históricos = %+v", historicos)
	}

	_, err = antecipacao.Estornar("de novo", titulos, uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "já foi estornada")
}

func TestEstornarAntecipacaoComTituloMovimentado(t *testing.T) {
	antecipacao, titulos := borderoDeTeste(t)
	for _, titulo := range titulos {
		titulo.RecebidoCentavos = 10
		break
	}

	_, err := antecipacao.Estornar("x", titulos, uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "recebido ou recomprado")
	if antecipacao.Status != domainreceber.StatusAntecipacaoAtiva {
		t.Fatal("a antecipação não muda quando o estorno é recusado")
	}
}

func TestConferirVersaoDaAntecipacao(t *testing.T) {
	antecipacao, _ := borderoDeTeste(t)

	esperarErro(t, antecipacao.ConferirVersao(0), domain.ErrValidacao, "versão da antecipação")
	esperarErro(t, antecipacao.ConferirVersao(2), domain.ErrConflito, "alterada por outra pessoa")
}

func TestRecomprarTituloAntecipado(t *testing.T) {
	antecipacao, titulos := borderoDeTeste(t)
	var titulo *domainreceber.Titulo
	for _, item := range titulos {
		titulo = item
		break
	}
	vinculo := antecipacao.Vinculo(titulo.ID)
	dados := domainreceber.DadosRecompra{
		RecompradoEm: data(2026, 10, 15), EncargosCentavos: 4500, Vencimento: data(2026, 10, 20),
		CategoriaFinanceiraID: uuid.New(), Motivo: "cliente não pagou",
	}

	carteira := novoTituloDeTeste(t)
	_, _, err := carteira.Recomprar(vinculo, dados, uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "só o título antecipado")

	semMotivo := dados
	semMotivo.Motivo = ""
	_, _, err = titulo.Recomprar(vinculo, semMotivo, uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "motivo da recompra")
	semCategoria := dados
	semCategoria.CategoriaFinanceiraID = uuid.Nil
	_, _, err = titulo.Recomprar(vinculo, semCategoria, uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "categoria financeira da recompra")

	recompra, historico, err := titulo.Recomprar(vinculo, dados, uuid.New(), agora)
	if err != nil {
		t.Fatalf("Recomprar falhou: %v", err)
	}
	if recompra.ValorCentavos != 154500 || recompra.AntecipacaoID != antecipacao.ID || titulo.Status != domainreceber.StatusCarteira || vinculo.Vigente {
		t.Fatalf("recompra = %+v, título = %+v", recompra, titulo)
	}
	if historico.Evento != domainreceber.EventoRecomprado || *historico.DeStatus != domainreceber.StatusAntecipado {
		t.Fatalf("histórico = %+v", historico)
	}
	if recompra.Observacao(titulo) != "recompra do título ALG-10/1: cliente não pagou" {
		t.Fatalf("observação = %q", recompra.Observacao(titulo))
	}

	titulo.Status = domainreceber.StatusAntecipado
	_, _, err = titulo.Recomprar(vinculo, dados, uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "antecipação vigente")

	_, _, err = (&domainreceber.Titulo{Status: domainreceber.StatusAntecipado}).Recomprar(nil, dados, uuid.New(), time.Now())
	esperarErro(t, err, domain.ErrConflito, "antecipação vigente")
}
