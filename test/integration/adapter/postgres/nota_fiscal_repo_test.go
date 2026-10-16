package postgres_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

var _ portsnotafiscal.ProdutoRepository = (*postgres.ProdutoRepository)(nil)

type cenarioBancoNotas struct {
	*cenarioBancoPedidos
	notas    *postgres.NotaFiscalRepository
	arquivos *postgres.ArquivoRepository
}

func novoCenarioNotas(t *testing.T) *cenarioBancoNotas {
	t.Helper()

	base := novoCenarioPedidos(t)

	return &cenarioBancoNotas{
		cenarioBancoPedidos: base,
		notas:               postgres.NewNotaFiscalRepository(base.banco),
		arquivos:            postgres.NewArquivoRepository(base.banco),
	}
}

func (c *cenarioBancoNotas) novoArquivo(t *testing.T) uuid.UUID {
	t.Helper()

	id := uuid.New()
	if _, err := c.arquivos.Criar(c.ctx, &domainsolicitacao.Arquivo{
		ID:             id,
		ProprietarioID: c.usuarioID,
		Nome:           "nfe.xml",
		Chave:          c.usuarioID.String() + "/" + id.String() + "/nfe.xml",
		ContentType:    "application/xml",
		Tamanho:        2048,
		CriadoEm:       time.Now().UTC(),
	}); err != nil {
		t.Fatalf("criação do arquivo falhou: %v", err)
	}

	return id
}

func (c *cenarioBancoNotas) novaNota(numero int, empresaID uuid.UUID, emitidaEm time.Time) *domainnotafiscal.NotaFiscal {
	clienteID := c.clienteID
	agora := time.Now().UTC()
	notaID := uuid.New()
	pedidoCliente := "PC-77"

	return &domainnotafiscal.NotaFiscal{
		ID:                    notaID,
		Chave:                 domainnotafiscal.ChaveDe(fmt.Sprintf("2526101122233300018155001%09d1123456780", numero)),
		EmpresaID:             empresaID,
		Numero:                numero,
		Serie:                 1,
		EmitidaEm:             emitidaEm,
		TipoOperacao:          domainnotafiscal.TipoOperacaoSaida,
		Finalidade:            domainnotafiscal.FinalidadeNormal,
		Natureza:              domainnotafiscal.NaturezaVenda,
		NaturezaOperacao:      "VENDA",
		ClienteID:             &clienteID,
		DestinatarioDocumento: "11444777000161",
		DestinatarioNome:      "Construtora Alfa",
		CFOPPredominante:      "5102",
		ValorTotalCentavos:    108850,
		ValorProdutosCentavos: 109000,
		ValorDescontoCentavos: 150,
		NumeroPedidoCliente:   &pedidoCliente,
		Protocolo:             "325260000012345",
		AutorizadaEm:          emitidaEm.Add(time.Minute),
		Status:                domainnotafiscal.StatusAutorizada,
		UsuarioID:             c.usuarioID,
		Versao:                1,
		CriadoEm:              agora,
		AtualizadoEm:          agora,
		Itens: []*domainnotafiscal.Item{
			{NotaFiscalID: notaID, NumeroItem: 1, Codigo: "CIM-01", Descricao: "Cimento", ProdutoID: &c.produto.ID, NCM: "25232910",
				CFOP: "5102", Unidade: "SC", QuantidadeDecimilesimos: 105000, ValorUnitario: "100.0000000000", ValorCentavos: 105000},
			{NotaFiscalID: notaID, NumeroItem: 2, Codigo: "REJ-02", Descricao: "Rejunte", NCM: "38245000", CFOP: "5102",
				Unidade: "KG", QuantidadeDecimilesimos: 20000, ValorUnitario: "20", ValorCentavos: 4000, DescontoCentavos: 150, PedidoCliente: &pedidoCliente},
		},
		Duplicatas: []*domainnotafiscal.Duplicata{
			{ID: uuid.New(), NotaFiscalID: notaID, Numero: "002", Vencimento: time.Date(2026, 12, 9, 0, 0, 0, 0, time.UTC), ValorCentavos: 54425},
			{ID: uuid.New(), NotaFiscalID: notaID, Numero: "001", Vencimento: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), ValorCentavos: 54425},
		},
	}
}

func (c *cenarioBancoNotas) criarNota(t *testing.T, numero int, empresaID uuid.UUID, emitidaEm time.Time) *domainnotafiscal.NotaFiscal {
	t.Helper()

	nota := c.novaNota(numero, empresaID, emitidaEm)
	nota.ArquivoID = c.novoArquivo(t)
	if err := c.notas.Criar(c.ctx, nota); err != nil {
		t.Fatalf("criar a nota falhou: %v", err)
	}

	return nota
}

func (c *cenarioBancoNotas) obterNota(t *testing.T, id uuid.UUID) *domainnotafiscal.NotaFiscal {
	t.Helper()

	nota, err := c.notas.Obter(c.ctx, id)
	if err != nil || nota == nil {
		t.Fatalf("obter a nota = %+v, %v", nota, err)
	}

	return nota
}

func (c *cenarioBancoNotas) pedidoComEntrega(t *testing.T) (*domainpedido.Pedido, *domainpedido.Entrega) {
	t.Helper()

	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, pedido)
	entrega := c.novaEntrega(t, pedido, c.galpao, 2)

	return c.obter(t, pedido.ID), entrega
}

func vincular(t *testing.T, nota *domainnotafiscal.NotaFiscal, pedido *domainpedido.Pedido, entrega *domainpedido.Entrega) *portsnotafiscal.Alteracao {
	t.Helper()

	versao, status := nota.Versao, nota.Status
	if err := nota.Vincular(pedido, entrega, time.Now().UTC()); err != nil {
		t.Fatalf("Vincular falhou: %v", err)
	}

	return &portsnotafiscal.Alteracao{Nota: nota, VersaoLida: versao, StatusAnterior: status, ConferirVinculo: true}
}

func TestNotaFiscalRepositoryCriaEObtemComItensEDuplicatas(t *testing.T) {
	c := novoCenarioNotas(t)
	emitida := time.Date(2026, 10, 10, 12, 30, 0, 0, time.UTC)
	nota := c.criarNota(t, 123, c.mia, emitida)

	lida := c.obterNota(t, nota.ID)
	if lida.Chave.Valor() != nota.Chave.Valor() || lida.EmpresaID != c.mia || lida.Numero != 123 || !lida.EmitidaEm.Equal(emitida) {
		t.Errorf("cabeçalho = %+v", lida)
	}
	if lida.Natureza != domainnotafiscal.NaturezaVenda || lida.Status != domainnotafiscal.StatusAutorizada || lida.Versao != 1 ||
		lida.ClienteID == nil || *lida.ClienteID != c.clienteID || lida.NumeroPedidoCliente == nil || *lida.NumeroPedidoCliente != "PC-77" {
		t.Errorf("situação = %+v", lida)
	}
	if len(lida.Itens) != 2 || lida.Itens[0].ProdutoID == nil || *lida.Itens[0].ProdutoID != c.produto.ID ||
		lida.Itens[1].ProdutoID != nil || lida.Itens[1].DescontoCentavos != 150 || lida.Itens[0].QuantidadeDecimilesimos != 105000 {
		t.Errorf("itens = %+v %+v", lida.Itens[0], lida.Itens[1])
	}
	if len(lida.Duplicatas) != 2 || lida.Duplicatas[0].Numero != "001" ||
		!lida.Duplicatas[0].Vencimento.Equal(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("duplicatas em ordem de vencimento = %+v", lida.Duplicatas)
	}

	porChave, err := c.notas.ObterPorChave(c.ctx, nota.Chave.Valor())
	if err != nil || porChave == nil || porChave.ID != nota.ID {
		t.Errorf("ObterPorChave = %+v, %v", porChave, err)
	}

	ausente, err := c.notas.Obter(c.ctx, uuid.New())
	if err != nil || ausente != nil {
		t.Errorf("nota inexistente = %+v, %v", ausente, err)
	}
}

func TestNotaFiscalRepositoryRecusaChaveEArquivoRepetidos(t *testing.T) {
	c := novoCenarioNotas(t)
	original := c.criarNota(t, 10, c.mia, time.Now().UTC())

	repetida := c.novaNota(10, c.mia, time.Now().UTC())
	repetida.ArquivoID = c.novoArquivo(t)
	err := c.notas.Criar(c.ctx, repetida)
	if !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "chave de acesso") {
		t.Errorf("chave repetida = %v, esperado conflito de chave", err)
	}

	mesmoArquivo := c.novaNota(11, c.mia, time.Now().UTC())
	mesmoArquivo.ArquivoID = original.ArquivoID
	err = c.notas.Criar(c.ctx, mesmoArquivo)
	if !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "arquivo já importado") {
		t.Errorf("arquivo repetido = %v, esperado conflito de arquivo", err)
	}

	if nota, _ := c.notas.Obter(c.ctx, mesmoArquivo.ID); nota != nil {
		t.Error("a nota recusada deixou registro")
	}

	vinculado, err := c.notas.ArquivoVinculado(c.ctx, original.ArquivoID)
	if err != nil || !vinculado {
		t.Errorf("ArquivoVinculado = %v, %v", vinculado, err)
	}
	livre, err := c.notas.ArquivoVinculado(c.ctx, c.novoArquivo(t))
	if err != nil || livre {
		t.Errorf("arquivo livre = %v, %v", livre, err)
	}
}

func TestNotaFiscalRepositoryVinculaPedidoEEntregaComGuarda(t *testing.T) {
	c := novoCenarioNotas(t)
	pedido, entrega := c.pedidoComEntrega(t)
	nota := c.criarNota(t, 20, c.mia, time.Now().UTC())

	alteracao := vincular(t, nota, pedido, entrega)
	if err := c.notas.Atualizar(c.ctx, alteracao); err != nil {
		t.Fatalf("Atualizar = %v", err)
	}
	if nota.Versao != 2 {
		t.Errorf("versão depois do vínculo = %d, esperado 2", nota.Versao)
	}
	lida := c.obterNota(t, nota.ID)
	if lida.PedidoID == nil || *lida.PedidoID != pedido.ID || lida.EntregaID == nil || *lida.EntregaID != entrega.ID || lida.Versao != 2 {
		t.Errorf("vínculo gravado = pedido %v entrega %v versão %d", lida.PedidoID, lida.EntregaID, lida.Versao)
	}

	velha := c.obterNota(t, nota.ID)
	alteracaoVelha := vincular(t, velha, pedido, nil)
	alteracaoVelha.VersaoLida = 1
	if err := c.notas.Atualizar(c.ctx, alteracaoVelha); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	outra := c.criarNota(t, 21, c.mia, time.Now().UTC())
	err := c.notas.Atualizar(c.ctx, vincular(t, outra, pedido, entrega))
	if !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "já tem uma nota fiscal autorizada") {
		t.Errorf("segunda nota na entrega = %v, esperado conflito", err)
	}

	inexistente := c.novaNota(22, c.mia, time.Now().UTC())
	err = c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{Nota: inexistente, VersaoLida: 1, StatusAnterior: domainnotafiscal.StatusAutorizada})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("nota inexistente = %v, esperado não encontrado", err)
	}
}

func TestNotaFiscalRepositoryConfereOPedidoNaTransacao(t *testing.T) {
	c := novoCenarioNotas(t)
	proposta := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	nota := c.criarNota(t, 30, c.mia, time.Now().UTC())

	pedidoID := proposta.ID
	nota.PedidoID = &pedidoID
	err := c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{
		Nota: nota, VersaoLida: 1, StatusAnterior: domainnotafiscal.StatusAutorizada, ConferirVinculo: true,
	})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pedido em proposta = %v, esperado conflito", err)
	}

	nova := c.novaNota(31, c.mia, time.Now().UTC())
	nova.ArquivoID = c.novoArquivo(t)
	nova.PedidoID = &pedidoID
	if err := c.notas.Criar(c.ctx, nova); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("criar com pedido em proposta = %v, esperado conflito", err)
	}
}

func TestNotaFiscalRepositoryCancelaPorEventoELiberaAEntrega(t *testing.T) {
	c := novoCenarioNotas(t)
	pedido, entrega := c.pedidoComEntrega(t)
	nota := c.criarNota(t, 40, c.mia, time.Now().UTC())
	if err := c.notas.Atualizar(c.ctx, vincular(t, nota, pedido, entrega)); err != nil {
		t.Fatalf("vincular = %v", err)
	}

	evento := &domainnotafiscal.Evento{
		ID: uuid.New(), NotaFiscalID: nota.ID, Tipo: domainnotafiscal.TipoEventoCancelamento, Sequencia: 1,
		OcorridoEm: time.Now().UTC(), Protocolo: "325260000099999", Justificativa: "Erro de valor",
		ArquivoID: c.novoArquivo(t), UsuarioID: c.usuarioID, CriadoEm: time.Now().UTC(),
	}
	versao := nota.Versao
	nota.Status = domainnotafiscal.StatusCancelada
	if err := c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{
		Nota: nota, VersaoLida: versao, StatusAnterior: domainnotafiscal.StatusAutorizada, Evento: evento,
	}); err != nil {
		t.Fatalf("cancelar = %v", err)
	}

	if lida := c.obterNota(t, nota.ID); lida.Status != domainnotafiscal.StatusCancelada || lida.Versao != versao+1 {
		t.Errorf("nota cancelada = %q versão %d", lida.Status, lida.Versao)
	}
	eventos, err := c.notas.ListarEventos(c.ctx, nota.ID)
	if err != nil || len(eventos) != 1 || eventos[0].Tipo != domainnotafiscal.TipoEventoCancelamento || eventos[0].Justificativa != "Erro de valor" {
		t.Errorf("eventos = %+v, %v", eventos, err)
	}

	repetido := *evento
	repetido.ID = uuid.New()
	repetido.ArquivoID = c.novoArquivo(t)
	err = c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{
		Nota: nota, VersaoLida: nota.Versao, StatusAnterior: domainnotafiscal.StatusCancelada, Evento: &repetido,
	})
	if !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "evento já registrado") {
		t.Errorf("evento repetido = %v, esperado conflito", err)
	}

	substituta := c.criarNota(t, 41, c.mia, time.Now().UTC())
	if err := c.notas.Atualizar(c.ctx, vincular(t, substituta, pedido, entrega)); err != nil {
		t.Errorf("nota substituta na entrega liberada = %v", err)
	}
}

func TestNotaFiscalRepositoryLigaProdutoAoItem(t *testing.T) {
	c := novoCenarioNotas(t)
	nota := c.criarNota(t, 50, c.mia, time.Now().UTC())
	c.criarNota(t, 51, c.red, time.Now().UTC())

	pendentes, err := c.notas.ListarItensSemProduto(c.ctx, portsnotafiscal.ItemPendenteFiltro{Empresa: &c.mia})
	if err != nil || len(pendentes) != 1 || pendentes[0].NotaFiscalID != nota.ID || pendentes[0].Item.Codigo != "REJ-02" || pendentes[0].NumeroNota != 50 {
		t.Fatalf("pendentes da MIA = %+v, %v", pendentes, err)
	}
	todos, err := c.notas.ListarItensSemProduto(c.ctx, portsnotafiscal.ItemPendenteFiltro{})
	if err != nil || len(todos) != 2 {
		t.Errorf("pendentes de todas = %d, %v", len(todos), err)
	}
	nenhum, err := c.notas.ListarItensSemProduto(c.ctx, portsnotafiscal.ItemPendenteFiltro{EmpresaIDs: []uuid.UUID{}})
	if err != nil || len(nenhum) != 0 {
		t.Errorf("acesso vazio = %d, %v", len(nenhum), err)
	}

	versao := nota.Versao
	item, err := nota.VincularProduto(2, c.segundo.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("VincularProduto = %v", err)
	}
	if err := c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{
		Nota: nota, VersaoLida: versao, StatusAnterior: nota.Status, ItensAlterados: []*domainnotafiscal.Item{item},
	}); err != nil {
		t.Fatalf("Atualizar item = %v", err)
	}

	lida := c.obterNota(t, nota.ID)
	if lida.Itens[1].ProdutoID == nil || *lida.Itens[1].ProdutoID != c.segundo.ID {
		t.Errorf("produto do item = %v", lida.Itens[1].ProdutoID)
	}
	if pendentes, _ := c.notas.ListarItensSemProduto(c.ctx, portsnotafiscal.ItemPendenteFiltro{Empresa: &c.mia}); len(pendentes) != 0 {
		t.Errorf("pendentes depois do vínculo = %d", len(pendentes))
	}
}

func TestNotaFiscalRepositoryListaComFiltros(t *testing.T) {
	c := novoCenarioNotas(t)
	pedido, _ := c.pedidoComEntrega(t)
	outubro := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	setembro := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	vinculada := c.criarNota(t, 60, c.mia, outubro)
	if err := c.notas.Atualizar(c.ctx, vincular(t, vinculada, pedido, nil)); err != nil {
		t.Fatalf("vincular = %v", err)
	}
	semPedido := c.criarNota(t, 61, c.mia, setembro)

	transferencia := c.novaNota(62, c.mia, outubro.Add(2*time.Hour))
	transferencia.ArquivoID = c.novoArquivo(t)
	transferencia.ClienteID = nil
	transferencia.EmpresaDestinatariaID = &c.red
	transferencia.EntreEmpresas = true
	transferencia.Natureza = domainnotafiscal.NaturezaTransferencia
	if err := c.notas.Criar(c.ctx, transferencia); err != nil {
		t.Fatalf("criar transferência = %v", err)
	}
	daRed := c.criarNota(t, 63, c.red, outubro.Add(3*time.Hour))

	ids := func(notas []*domainnotafiscal.NotaFiscal) []uuid.UUID {
		resultado := make([]uuid.UUID, 0, len(notas))
		for _, nota := range notas {
			resultado = append(resultado, nota.ID)
		}
		return resultado
	}
	verdadeiro := true
	diaDez := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	numero := 61
	representante := uuid.New()

	casos := []struct {
		nome     string
		filtro   portsnotafiscal.NotaFiscalFiltro
		esperado []uuid.UUID
	}{
		{"todas", portsnotafiscal.NotaFiscalFiltro{}, []uuid.UUID{daRed.ID, transferencia.ID, vinculada.ID, semPedido.ID}},
		{"empresa", portsnotafiscal.NotaFiscalFiltro{Empresa: &c.red}, []uuid.UUID{daRed.ID}},
		{"acesso vazio", portsnotafiscal.NotaFiscalFiltro{EmpresaIDs: []uuid.UUID{}}, []uuid.UUID{}},
		{"acesso da MIA", portsnotafiscal.NotaFiscalFiltro{EmpresaIDs: []uuid.UUID{c.mia}}, []uuid.UUID{transferencia.ID, vinculada.ID, semPedido.ID}},
		{"pedido", portsnotafiscal.NotaFiscalFiltro{PedidoID: &pedido.ID}, []uuid.UUID{vinculada.ID}},
		{"sem pedido", portsnotafiscal.NotaFiscalFiltro{SemPedido: true, Empresa: &c.mia}, []uuid.UUID{semPedido.ID}},
		{"entre empresas", portsnotafiscal.NotaFiscalFiltro{EntreEmpresas: &verdadeiro}, []uuid.UUID{transferencia.ID}},
		{"natureza", portsnotafiscal.NotaFiscalFiltro{Natureza: domainnotafiscal.NaturezaTransferencia}, []uuid.UUID{transferencia.ID}},
		{"cliente", portsnotafiscal.NotaFiscalFiltro{ClienteID: &c.clienteID, Empresa: &c.mia}, []uuid.UUID{vinculada.ID, semPedido.ID}},
		{"dia dez", portsnotafiscal.NotaFiscalFiltro{EmitidaDe: &diaDez, EmitidaAte: &diaDez, Empresa: &c.mia}, []uuid.UUID{transferencia.ID, vinculada.ID}},
		{"até setembro", portsnotafiscal.NotaFiscalFiltro{EmitidaAte: &setembro}, []uuid.UUID{semPedido.ID}},
		{"número", portsnotafiscal.NotaFiscalFiltro{Numero: &numero}, []uuid.UUID{semPedido.ID}},
		{"chave", portsnotafiscal.NotaFiscalFiltro{Chave: daRed.Chave.Valor()}, []uuid.UUID{daRed.ID}},
		{"status", portsnotafiscal.NotaFiscalFiltro{Status: domainnotafiscal.StatusCancelada}, []uuid.UUID{}},
		{"representante", portsnotafiscal.NotaFiscalFiltro{RepresentanteID: &representante}, []uuid.UUID{}},
	}

	for _, caso := range casos {
		notas, err := c.notas.Listar(c.ctx, caso.filtro)
		if err != nil {
			t.Errorf("%s: Listar = %v", caso.nome, err)
			continue
		}
		obtidos := ids(notas)
		if fmt.Sprint(obtidos) != fmt.Sprint(caso.esperado) {
			t.Errorf("%s: notas = %v, esperado %v", caso.nome, obtidos, caso.esperado)
		}
	}

	pagina, err := c.notas.Listar(c.ctx, portsnotafiscal.NotaFiscalFiltro{PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 3}})
	if err != nil || len(pagina) != 1 || pagina[0].ID != semPedido.ID {
		t.Errorf("segunda página = %v, %v", ids(pagina), err)
	}
}

func TestNotaFiscalRepositoryListaEntregasSemNota(t *testing.T) {
	c := novoCenarioNotas(t)
	c.entrada(t, c.produto, c.galpao, 10)
	pedido, entrega := c.pedidoComEntrega(t)

	pendentes, err := c.notas.ListarEntregasSemNota(c.ctx, portsnotafiscal.EntregaSemNotaFiltro{})
	if err != nil || len(pendentes) != 0 {
		t.Fatalf("entrega pendente não entra = %+v, %v", pendentes, err)
	}

	c.gravar(t, c.gravacaoDaSaida(t, pedido, entrega))

	saidas, err := c.notas.ListarEntregasSemNota(c.ctx, portsnotafiscal.EntregaSemNotaFiltro{Empresa: &c.mia})
	if err != nil || len(saidas) != 1 || saidas[0].EntregaID != entrega.ID || saidas[0].PedidoID != pedido.ID ||
		saidas[0].PedidoNumero != pedido.Numero || saidas[0].Status != "saiu" || saidas[0].SaiuEm == nil {
		t.Fatalf("entregas sem nota = %+v, %v", saidas, err)
	}
	if outras, _ := c.notas.ListarEntregasSemNota(c.ctx, portsnotafiscal.EntregaSemNotaFiltro{EmpresaIDs: []uuid.UUID{c.red}}); len(outras) != 0 {
		t.Errorf("acesso de outra empresa = %d", len(outras))
	}

	nota := c.criarNota(t, 70, c.mia, time.Now().UTC())
	if err := c.notas.Atualizar(c.ctx, vincular(t, nota, c.obter(t, pedido.ID), entrega)); err != nil {
		t.Fatalf("vincular = %v", err)
	}
	if restantes, _ := c.notas.ListarEntregasSemNota(c.ctx, portsnotafiscal.EntregaSemNotaFiltro{}); len(restantes) != 0 {
		t.Errorf("entrega com nota continua na lista: %d", len(restantes))
	}
}

func TestProdutoRepositoryIDsPorCodigos(t *testing.T) {
	c := novoCenarioNotas(t)
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE produto SET codigo = 'CIM-01' WHERE id = $1`, c.produto.ID); err != nil {
		t.Fatalf("definir código falhou: %v", err)
	}

	encontrados, err := c.produtos.IDsPorCodigos(c.ctx, []string{"CIM-01", "NAO-EXISTE"})
	if err != nil || len(encontrados) != 1 || encontrados["CIM-01"] != c.produto.ID {
		t.Errorf("IDsPorCodigos = %v, %v", encontrados, err)
	}

	vazio, err := c.produtos.IDsPorCodigos(c.ctx, nil)
	if err != nil || len(vazio) != 0 {
		t.Errorf("sem códigos = %v, %v", vazio, err)
	}
}
