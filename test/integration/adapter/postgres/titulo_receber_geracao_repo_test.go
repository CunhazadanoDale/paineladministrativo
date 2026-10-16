package postgres_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

func (c *cenarioBancoTitulos) inserirPedidoDeImportacao(t *testing.T, status string) uuid.UUID {
	t.Helper()

	var numero int
	if err := c.banco.Get(&numero, `SELECT count(*) + 1 FROM pedido WHERE empresa_id = $1`, c.empresaID); err != nil {
		t.Fatal(err)
	}
	motivo := any(nil)
	if status == "cancelado" {
		motivo = "desistência"
	}

	id := uuid.New()
	if _, err := c.banco.Exec(`
		INSERT INTO pedido (id, empresa_id, numero, cliente_id, modelo, entrada_bp, prazo_entrega_dias, status,
		                    valor_total_centavos, cancelado_motivo, usuario_id)
		VALUES ($1, $2, $3, $4, 'importacao', 3000, 120, $5, 100000, $6, $7)
	`, id, c.empresaID, numero, c.clienteID, status, motivo, c.solicitanteID); err != nil {
		t.Fatalf("inserir pedido falhou: %v", err)
	}

	return id
}

func (c *cenarioBancoTitulos) geracao(t *testing.T, origem domainreceber.Origem, conta *uuid.UUID, parcelas ...domainreceber.ParcelaDeOrigem) *domainreceber.Geracao {
	t.Helper()

	geracao, err := domainreceber.NovaGeracao(origem, c.empresaID, c.clienteID, conta, parcelas, c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("geração de teste inválida: %v", err)
	}

	return geracao
}

func parcelaDoPedido(pedidoID uuid.UUID, numero string) domainreceber.ParcelaDeOrigem {
	return domainreceber.ParcelaDeOrigem{
		OrigemID: pedidoID, Numero: numero, Parcela: 1, Vencimento: diaDoTitulo(9, 30), ValorCentavos: 30000,
	}
}

func (c *cenarioBancoTitulos) cancelamento(t *testing.T, origem domainreceber.Origem, ids ...uuid.UUID) domainreceber.CancelamentoDeOrigem {
	t.Helper()

	cancelamento, err := domainreceber.NovoCancelamentoDeOrigem(origem, ids, "origem cancelada", c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("cancelamento de teste inválido: %v", err)
	}

	return cancelamento
}

func TestGerarDeOrigemEIdempotente(t *testing.T) {
	c := novoCenarioTitulos(t)
	pedidoID := c.inserirPedidoDeImportacao(t, "confirmado")

	primeira := c.geracao(t, domainreceber.OrigemPedido, &c.contaID, parcelaDoPedido(pedidoID, "PED-1"))
	resultado, err := c.titulos.GerarDeOrigem(c.ctx, primeira)
	if err != nil {
		t.Fatalf("GerarDeOrigem falhou: %v", err)
	}
	if len(resultado.Gerados) != 1 || len(resultado.JaExistentes) != 0 || resultado.Gerados[0] != primeira.Titulos[0].ID {
		t.Fatalf("resultado = %+v", resultado)
	}

	gerado := c.obterTitulo(t, resultado.Gerados[0])
	if gerado.Origem != domainreceber.OrigemPedido || *gerado.OrigemID != pedidoID || gerado.Status != domainreceber.StatusEmBanco ||
		gerado.CategoriaFinanceiraID != nil || !gerado.Vencimento.Equal(diaDoTitulo(9, 30)) {
		t.Fatalf("título gerado = %+v", gerado)
	}
	historico, err := c.titulos.ListarHistorico(c.ctx, gerado.ID)
	if err != nil || len(historico) != 1 || historico[0].Evento != domainreceber.EventoGerado {
		t.Fatalf("histórico = %+v, %v", historico, err)
	}

	segunda := c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(pedidoID, "PED-1"))
	repetido, err := c.titulos.GerarDeOrigem(c.ctx, segunda)
	if err != nil {
		t.Fatalf("gerar de novo falhou: %v", err)
	}
	if len(repetido.Gerados) != 0 || len(repetido.JaExistentes) != 1 || repetido.JaExistentes[0] != gerado.ID {
		t.Fatalf("resultado repetido = %+v", repetido)
	}
	historico, _ = c.titulos.ListarHistorico(c.ctx, gerado.ID)
	if len(historico) != 1 {
		t.Fatalf("regerar não grava histórico: %+v", historico)
	}
}

func TestGerarDeOrigemConcorrenteGeraUmaVez(t *testing.T) {
	c := novoCenarioTitulos(t)
	pedidoID := c.inserirPedidoDeImportacao(t, "confirmado")

	const tentativas = 6
	var espera sync.WaitGroup
	resultados := make([]domainreceber.ResultadoGeracao, tentativas)
	erros := make([]error, tentativas)
	for i := range tentativas {
		geracao := c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(pedidoID, "PED-1"))
		espera.Add(1)
		go func() {
			defer espera.Done()
			resultados[i], erros[i] = c.titulos.GerarDeOrigem(c.ctx, geracao)
		}()
	}
	espera.Wait()

	gerados, existentes := 0, 0
	for i := range tentativas {
		if erros[i] != nil {
			t.Fatalf("tentativa %d falhou: %v", i, erros[i])
		}
		gerados += len(resultados[i].Gerados)
		existentes += len(resultados[i].JaExistentes)
	}
	if gerados != 1 || existentes != tentativas-1 {
		t.Fatalf("gerados = %d, já existentes = %d", gerados, existentes)
	}

	origem := domainreceber.OrigemPedido
	if ids := c.listar(t, portsreceber.TituloFiltro{Origem: origem, OrigemID: &pedidoID}); len(ids) != 1 {
		t.Fatalf("títulos do pedido = %v", ids)
	}
}

func TestGerarDeOrigemRecusaOrigemInvalida(t *testing.T) {
	c := novoCenarioTitulos(t)

	cancelado := c.inserirPedidoDeImportacao(t, "cancelado")
	_, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(cancelado, "PED-1")))
	esperarErro(t, err, domain.ErrConflito, "origem cancelada não gera título")

	_, err = c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(uuid.New(), "PED-2")))
	esperarErro(t, err, domain.ErrValidacao, "origem do título não encontrada")

	_, err = c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemNotaFiscalDuplicata, nil, parcelaDoPedido(uuid.New(), "1234")))
	esperarErro(t, err, domain.ErrValidacao, "origem ainda não suportada")

	if ids := c.listar(t, portsreceber.TituloFiltro{}); len(ids) != 0 {
		t.Fatalf("nenhum título deveria existir: %v", ids)
	}
}

func TestGerarDeOrigemTudoOuNada(t *testing.T) {
	c := novoCenarioTitulos(t)
	primeiro := c.inserirPedidoDeImportacao(t, "confirmado")
	segundo := c.inserirPedidoDeImportacao(t, "confirmado")
	if _, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(primeiro, "PED-1"))); err != nil {
		t.Fatal(err)
	}

	terceiro := c.inserirPedidoDeImportacao(t, "confirmado")
	_, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil,
		parcelaDoPedido(terceiro, "PED-3"), parcelaDoPedido(segundo, "ped-1")))
	esperarErro(t, err, domain.ErrConflito, "número e parcela")

	if ids := c.listar(t, portsreceber.TituloFiltro{}); len(ids) != 1 {
		t.Fatalf("a geração que falhou não grava nenhuma parcela: %v", ids)
	}

	if _, err := c.banco.Exec(`UPDATE empresa SET ativa = false WHERE id = $1`, c.empresaID); err != nil {
		t.Fatal(err)
	}
	_, err = c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(terceiro, "PED-3")))
	esperarErro(t, err, domain.ErrValidacao, "empresa inativa")
}

func TestGerarDeOrigemAceitaClienteInativo(t *testing.T) {
	c := novoCenarioTitulos(t)
	pedidoID := c.inserirPedidoDeImportacao(t, "confirmado")
	if _, err := c.banco.Exec(`UPDATE cliente SET ativo = false WHERE id = $1`, c.clienteID); err != nil {
		t.Fatal(err)
	}

	if _, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(pedidoID, "PED-1"))); err != nil {
		t.Fatalf("o documento de origem já existe; cliente inativo não impede a geração: %v", err)
	}
}

func TestCancelarPorOrigem(t *testing.T) {
	c := novoCenarioTitulos(t)
	semMovimento := c.inserirPedidoDeImportacao(t, "confirmado")
	antecipado := c.inserirPedidoDeImportacao(t, "confirmado")
	parcial := c.inserirPedidoDeImportacao(t, "confirmado")
	resultado, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil,
		parcelaDoPedido(semMovimento, "PED-1"), parcelaDoPedido(antecipado, "PED-2"), parcelaDoPedido(parcial, "PED-3")))
	if err != nil {
		t.Fatal(err)
	}
	idSemMovimento, idAntecipado, idParcial := resultado.Gerados[0], resultado.Gerados[1], resultado.Gerados[2]
	if _, err := c.banco.Exec(`UPDATE titulo_receber SET status = 'antecipado' WHERE id = $1`, idAntecipado); err != nil {
		t.Fatal(err)
	}
	if _, err := c.banco.Exec(`UPDATE titulo_receber SET recebido_centavos = 1000 WHERE id = $1`, idParcial); err != nil {
		t.Fatal(err)
	}

	cancelamento := c.cancelamento(t, domainreceber.OrigemPedido, semMovimento, antecipado, parcial, uuid.New())
	efeito, err := c.titulos.CancelarPorOrigem(c.ctx, cancelamento)
	if err != nil {
		t.Fatalf("CancelarPorOrigem falhou: %v", err)
	}
	if len(efeito.Cancelados) != 1 || efeito.Cancelados[0] != idSemMovimento || len(efeito.Marcados) != 2 || efeito.Ignorados != 1 {
		t.Fatalf("efeito = %+v", efeito)
	}

	cancelado := c.obterTitulo(t, idSemMovimento)
	if cancelado.Status != domainreceber.StatusCancelado || *cancelado.CanceladoMotivo != "origem cancelada" || cancelado.Versao != 2 {
		t.Fatalf("título sem movimento = %+v", cancelado)
	}
	marcado := c.obterTitulo(t, idAntecipado)
	if marcado.Status != domainreceber.StatusAntecipado || marcado.OrigemCanceladaEm == nil {
		t.Fatalf("título antecipado = %+v", marcado)
	}
	esperarMesmosIDs(t, c.listar(t, portsreceber.TituloFiltro{OrigemCancelada: true}), idAntecipado, idParcial)

	historico, _ := c.titulos.ListarHistorico(c.ctx, idParcial)
	if len(historico) != 2 || historico[1].Evento != domainreceber.EventoOrigemCancelada {
		t.Fatalf("histórico do parcial = %+v", historico)
	}

	repetido, err := c.titulos.CancelarPorOrigem(c.ctx, cancelamento)
	if err != nil || len(repetido.Cancelados) != 0 || len(repetido.Marcados) != 0 || repetido.Ignorados != 4 {
		t.Fatalf("cancelar de novo = %+v, %v", repetido, err)
	}

	if _, err := c.titulos.CancelarPorOrigem(c.ctx, domainreceber.CancelamentoDeOrigem{}); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("cancelamento vazio = %v", err)
	}
}

func esperarMesmosIDs(t *testing.T, obtidos []uuid.UUID, esperados ...uuid.UUID) {
	t.Helper()

	comparar := func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) }
	obtidos, esperados = slices.Clone(obtidos), slices.Clone(esperados)
	slices.SortFunc(obtidos, comparar)
	slices.SortFunc(esperados, comparar)
	esperarIDs(t, obtidos, esperados...)
}
