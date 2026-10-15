package pedido_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func localDeTeste(t *testing.T, empresaID uuid.UUID) *domainestoque.Local {
	t.Helper()

	local, err := domainestoque.NovoLocal(empresaID, "Galpão "+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}

	return local
}

func freteFOB() domainpedido.Frete {
	return domainpedido.FreteDe(domainpedido.TipoFreteFOB, nil)
}

func entregaDeTeste(t *testing.T, pedido *domainpedido.Pedido, local *domainestoque.Local, quantidades ...int) *domainpedido.Entrega {
	t.Helper()

	itens := make([]domainpedido.EntregaItemInput, 0, len(quantidades))
	for indice, quantidade := range quantidades {
		itens = append(itens, domainpedido.EntregaItemInput{
			PedidoItemID: pedido.Itens[indice].ID,
			Local:        local,
			Quantidade:   quantidade,
		})
	}

	entrega, err := pedido.NovaEntrega(uuid.New(), domainpedido.DadosEntrega{Frete: freteFOB(), Itens: itens}, nil, agora)
	if err != nil {
		t.Fatalf("NovaEntrega falhou: %v", err)
	}

	return entrega
}

func TestNovaEntregaNasceDoPedido(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)
	transportadora := "  Rodonaves  "
	frete := int64(18000)

	entrega, err := pedido.NovaEntrega(uuid.New(), domainpedido.DadosEntrega{
		Transportadora: &transportadora,
		Frete:          domainpedido.FreteDe(domainpedido.TipoFreteCIF, &frete),
		Previsao:       &agora,
		Itens: []domainpedido.EntregaItemInput{
			{PedidoItemID: pedido.Itens[0].ID, Local: local, Quantidade: 6},
		},
	}, nil, agora)
	if err != nil {
		t.Fatalf("NovaEntrega falhou: %v", err)
	}

	if entrega.Status != domainpedido.StatusEntregaPendente || entrega.PedidoID != pedido.ID || entrega.Versao != 1 {
		t.Errorf("entrega nasceu com status %q, pedido %v, versão %d", entrega.Status, entrega.PedidoID, entrega.Versao)
	}
	if entrega.Transportadora == nil || *entrega.Transportadora != "Rodonaves" {
		t.Errorf("transportadora = %v", entrega.Transportadora)
	}
	if entrega.Previsao == nil || !entrega.Previsao.Equal(domainpedido.DataDe(agora)) {
		t.Errorf("previsão = %v, esperado só a data", entrega.Previsao)
	}
	if len(entrega.Itens) != 1 || entrega.Itens[0].EntregaID != entrega.ID || entrega.Itens[0].LocalID != local.ID {
		t.Errorf("itens da entrega = %+v", entrega.Itens)
	}
	if pedido.Itens[0].QuantidadeEntregue != 0 {
		t.Error("criar entrega não pode mexer no entregue")
	}
}

func TestNovaEntregaValida(t *testing.T) {
	longa := strings.Repeat("t", 161)

	casos := []struct {
		nome      string
		montar    func(*domainpedido.Pedido, *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int)
		sentinela error
	}{
		{"sem itens", func(_ *domainpedido.Pedido, _ *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB()}, nil
		}, domain.ErrValidacao},
		{"sem frete", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 1}}}, nil
		}, domain.ErrValidacao},
		{"transportadora longa", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Transportadora: &longa, Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 1}}}, nil
		}, domain.ErrValidacao},
		{"item de outro pedido", func(_ *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: uuid.New(), Local: l, Quantidade: 1}}}, nil
		}, domain.ErrValidacao},
		{"sem local", func(p *domainpedido.Pedido, _ *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Quantidade: 1}}}, nil
		}, domain.ErrValidacao},
		{"local de outra empresa", func(p *domainpedido.Pedido, _ *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			outro, _ := domainestoque.NovoLocal(uuid.New(), "Galpão da CE")
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: outro, Quantidade: 1}}}, nil
		}, domain.ErrValidacao},
		{"quantidade zero", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 0}}}, nil
		}, domain.ErrValidacao},
		{"acima do restante", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 11}}}, nil
		}, domain.ErrValidacao},
		{"acima do ainda não programado", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 5}}},
				map[uuid.UUID]int{p.Itens[0].ID: 6}
		}, domain.ErrValidacao},
		{"item repetido", func(p *domainpedido.Pedido, l *domainestoque.Local) (domainpedido.DadosEntrega, map[uuid.UUID]int) {
			return domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{
				{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 1},
				{PedidoItemID: p.Itens[0].ID, Local: l, Quantidade: 1},
			}}, nil
		}, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
			dados, programado := caso.montar(pedido, localDeTeste(t, pedido.EmpresaID))

			if _, err := pedido.NovaEntrega(uuid.New(), dados, programado, agora); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	proposta := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	dados := domainpedido.DadosEntrega{Frete: freteFOB(), Itens: []domainpedido.EntregaItemInput{
		{PedidoItemID: proposta.Itens[0].ID, Local: localDeTeste(t, proposta.EmpresaID), Quantidade: 1},
	}}
	if _, err := proposta.NovaEntrega(uuid.New(), dados, nil, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("entrega de proposta = %v, esperado conflito", err)
	}
}

func TestRegistrarSaidaAvancaOPedido(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)

	primeira := entregaDeTeste(t, pedido, local, 6, 4)
	if err := pedido.RegistrarSaida(primeira, agora, agora); err != nil {
		t.Fatalf("primeira saída falhou: %v", err)
	}
	if pedido.Status != domainpedido.StatusParcialmenteEntregue || pedido.Itens[0].QuantidadeEntregue != 6 || pedido.Itens[1].QuantidadeEntregue != 4 {
		t.Errorf("depois da primeira saída: %q, entregues %d e %d", pedido.Status, pedido.Itens[0].QuantidadeEntregue, pedido.Itens[1].QuantidadeEntregue)
	}
	if primeira.Status != domainpedido.StatusEntregaSaiu || primeira.SaiuEm == nil {
		t.Errorf("entrega = %q, saiu em %v", primeira.Status, primeira.SaiuEm)
	}

	segunda := entregaDeTeste(t, pedido, local, 4)
	if err := pedido.RegistrarSaida(segunda, agora, agora); err != nil {
		t.Fatalf("segunda saída falhou: %v", err)
	}
	if pedido.Status != domainpedido.StatusEntregue {
		t.Errorf("status = %q, esperado entregue", pedido.Status)
	}
	if err := pedido.RegistrarSaida(segunda, agora, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("saída de pedido entregue = %v, esperado conflito", err)
	}
}

func TestRegistrarSaidaValidaSemMexerNoPedido(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)

	futura := entregaDeTeste(t, pedido, local, 2)
	if err := pedido.RegistrarSaida(futura, agora.AddDate(0, 0, 1), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída no futuro = %v, esperado erro de validação", err)
	}
	if pedido.Itens[0].QuantidadeEntregue != 0 || pedido.Status != domainpedido.StatusConfirmado || futura.Status != domainpedido.StatusEntregaPendente {
		t.Error("saída inválida mexeu no pedido ou na entrega")
	}

	excedente := entregaDeTeste(t, pedido, local, 8)
	paralela := entregaDeTeste(t, pedido, local, 8)
	if err := pedido.RegistrarSaida(excedente, agora, agora); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	if err := pedido.RegistrarSaida(paralela, agora, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída acima do restante = %v, esperado erro de validação", err)
	}

	outro := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	if err := outro.RegistrarSaida(futura, agora, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída de entrega de outro pedido = %v, esperado erro de validação", err)
	}
}

func TestConcluirECancelarEntrega(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)
	entrega := entregaDeTeste(t, pedido, local, 2)

	if err := entrega.Concluir(agora, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("concluir pendente = %v, esperado conflito", err)
	}

	saiuEm := agora.AddDate(0, 0, -2)
	if err := pedido.RegistrarSaida(entrega, saiuEm, agora); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	if err := entrega.Cancelar(agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar entrega que saiu = %v, esperado conflito", err)
	}
	if err := entrega.Concluir(agora.AddDate(0, 0, -3), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("entregue antes da saída = %v, esperado erro de validação", err)
	}
	if err := entrega.Concluir(agora, agora); err != nil {
		t.Fatalf("Concluir falhou: %v", err)
	}
	if entrega.Status != domainpedido.StatusEntregaEntregue || entrega.EntregueEm == nil {
		t.Errorf("entrega = %q, entregue em %v", entrega.Status, entrega.EntregueEm)
	}

	pendente := entregaDeTeste(t, pedido, local, 1)
	if err := pendente.Cancelar(agora); err != nil || pendente.Status != domainpedido.StatusEntregaCancelada {
		t.Errorf("cancelar pendente = %v, status %q", err, pendente.Status)
	}
}

func TestCancelarPedidoComEntregas(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)

	concluida := entregaDeTeste(t, pedido, local, 2)
	if err := pedido.RegistrarSaida(concluida, agora, agora); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	emTransito := entregaDeTeste(t, pedido, local, 1)
	if err := pedido.RegistrarSaida(emTransito, agora, agora); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	if err := concluida.Concluir(agora, agora); err != nil {
		t.Fatalf("Concluir falhou: %v", err)
	}
	pendente := entregaDeTeste(t, pedido, local, 3)
	entregas := []*domainpedido.Entrega{concluida, emTransito, pendente}

	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "obra parada", Entregas: entregas}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("cancelar com entrega em trânsito = %v, esperado conflito", err)
	}
	if pendente.Status != domainpedido.StatusEntregaPendente || pedido.Status != domainpedido.StatusParcialmenteEntregue {
		t.Fatal("cancelamento recusado mexeu nas entregas ou no pedido")
	}

	if err := emTransito.Concluir(agora, agora); err != nil {
		t.Fatalf("Concluir falhou: %v", err)
	}
	canceladas, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "obra parada", Entregas: entregas}, agora)
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if len(canceladas) != 1 || canceladas[0] != pendente || pendente.Status != domainpedido.StatusEntregaCancelada {
		t.Errorf("entregas canceladas = %v, esperado só a pendente", canceladas)
	}
	if concluida.Status != domainpedido.StatusEntregaEntregue || pedido.Status != domainpedido.StatusCancelado {
		t.Errorf("entrega concluída %q e pedido %q", concluida.Status, pedido.Status)
	}
}

func TestProgramadoPorItemSomaSoAsPendentes(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	local := localDeTeste(t, pedido.EmpresaID)

	saiu := entregaDeTeste(t, pedido, local, 2)
	if err := pedido.RegistrarSaida(saiu, agora, agora); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	primeira := entregaDeTeste(t, pedido, local, 3, 1)
	segunda := entregaDeTeste(t, pedido, local, 1)
	cancelada := entregaDeTeste(t, pedido, local, 4)
	if err := cancelada.Cancelar(agora); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}

	programado := domainpedido.ProgramadoPorItem([]*domainpedido.Entrega{saiu, primeira, segunda, cancelada})

	if programado[pedido.Itens[0].ID] != 4 || programado[pedido.Itens[1].ID] != 1 {
		t.Errorf("programado = %v, esperado 4 e 1", programado)
	}
}

func TestEntregaConferirVersao(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
	entrega := entregaDeTeste(t, pedido, localDeTeste(t, pedido.EmpresaID), 1)

	if err := entrega.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão ausente = %v, esperado erro de validação", err)
	}
	if err := entrega.ConferirVersao(3); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v, esperado conflito", err)
	}
}
