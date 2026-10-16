package notafiscal_test

import (
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
	"github.com/google/uuid"
)

func TestImportarGravaANotaDoCliente(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(123)

	importacao := c.importar(t, c.financeiro, documento)

	nota := importacao.Nota
	if nota.EmpresaID != c.mia || nota.ClienteID == nil || *nota.ClienteID != c.cliente.ID || nota.Chave.Valor() != documento.Chave() {
		t.Errorf("nota = empresa %v cliente %v chave %q", nota.EmpresaID, nota.ClienteID, nota.Chave.Valor())
	}
	if nota.Natureza != domainnotafiscal.NaturezaVenda || nota.UsuarioID != c.financeiro || len(nota.Duplicatas) != 2 {
		t.Errorf("natureza %q usuário %v duplicatas %d", nota.Natureza, nota.UsuarioID, len(nota.Duplicatas))
	}
	if nota.Itens[0].ProdutoID == nil || *nota.Itens[0].ProdutoID != c.piso.ID || nota.Itens[1].ProdutoID != nil {
		t.Errorf("produtos dos itens = %v %v", nota.Itens[0].ProdutoID, nota.Itens[1].ProdutoID)
	}
	if !slices.Equal(importacao.Avisos, []string{portsin.AvisoItensSemProduto}) {
		t.Errorf("avisos = %v", importacao.Avisos)
	}
	if _, ok := c.m.notas[nota.ID]; !ok {
		t.Error("nota não foi gravada")
	}
}

func TestImportarCriaClienteQueNaoExiste(t *testing.T) {
	c := novoCenario(t)
	documento := nfe.Nova(cnpjMia, cpfClienteNovo, 200)
	documento.NomeDestinatario = "Maria da Silva"

	importacao := c.importar(t, c.administrador, documento)

	if !slices.Contains(importacao.Avisos, portsin.AvisoClienteCriado) {
		t.Errorf("avisos = %v, esperado cliente_criado", importacao.Avisos)
	}
	cliente := c.m.clientes[*importacao.Nota.ClienteID]
	if cliente == nil || cliente.Documento.Valor() != cpfClienteNovo || cliente.Nome != "Maria da Silva" ||
		cliente.Tipo != domaincliente.TipoClientePF || cliente.UF.String() != "PB" || cliente.Cidade != "Joao Pessoa" ||
		cliente.EmailNF == nil || *cliente.EmailNF != "nf@exemplo.com.br" || cliente.RepresentanteID != nil {
		t.Errorf("cliente criado = %+v", cliente)
	}
}

func TestImportarCriaClienteSemEmailInvalidoEReaproveitaConcorrente(t *testing.T) {
	c := novoCenario(t)
	documento := nfe.Nova(cnpjMia, cpfClienteNovo, 201)
	documento.Email = "sem-arroba"

	importacao := c.importar(t, c.financeiro, documento)
	if cliente := c.m.clientes[*importacao.Nota.ClienteID]; cliente == nil || cliente.EmailNF != nil {
		t.Errorf("cliente com e-mail inválido = %+v", cliente)
	}

	outra := novoCenario(t)
	outra.m.conflitoNoCliente = true
	concorrente := outra.importar(t, outra.financeiro, nfe.Nova(cnpjMia, cpfClienteNovo, 202))
	if slices.Contains(concorrente.Avisos, portsin.AvisoClienteCriado) {
		t.Errorf("avisos = %v, cliente veio de outra importação", concorrente.Avisos)
	}
	if outra.m.clientes[*concorrente.Nota.ClienteID] == nil {
		t.Error("a nota não aponta para o cliente que venceu a corrida")
	}
}

func TestImportarClienteNovoSemEnderecoFalha(t *testing.T) {
	c := novoCenario(t)
	documento := nfe.Nova(cnpjMia, cpfClienteNovo, 203)
	documento.UF = ""

	_, err := c.notas.Importar(c.ctx, portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, documento.XML())})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem UF = %v, esperado validação", err)
	}
	if len(c.m.clientes) != 1 || len(c.m.notas) != 0 {
		t.Errorf("gravou cliente ou nota: %d clientes, %d notas", len(c.m.clientes), len(c.m.notas))
	}
}

func TestImportarEntreEmpresasEhTransferenciaSemCliente(t *testing.T) {
	c := novoCenario(t)

	importacao := c.importar(t, c.administrador, nfe.Nova(cnpjMia, cnpjRed, 300))

	nota := importacao.Nota
	if !nota.EntreEmpresas || nota.Natureza != domainnotafiscal.NaturezaTransferencia || nota.ClienteID != nil ||
		nota.EmpresaDestinatariaID == nil || *nota.EmpresaDestinatariaID != c.red {
		t.Errorf("transferência = %+v", nota)
	}
	if len(c.m.clientes) != 1 {
		t.Errorf("criou cliente para empresa do grupo: %d", len(c.m.clientes))
	}
	if slices.Contains(importacao.Avisos, portsin.AvisoNumeroPedidoAusente) {
		t.Errorf("avisos = %v", importacao.Avisos)
	}
}

func TestImportarVinculaPedidoEEntrega(t *testing.T) {
	c := novoCenario(t)

	importacao, err := c.notas.Importar(c.ctx, portsin.ImportarInput{
		UsuarioID: c.financeiro,
		ArquivoID: c.enviar(c.financeiro, c.notaDaMia(400).XML()),
		PedidoID:  &c.pedido.ID,
		EntregaID: &c.entrega.ID,
	})
	if err != nil {
		t.Fatalf("Importar = %v", err)
	}
	if importacao.Nota.PedidoID == nil || *importacao.Nota.PedidoID != c.pedido.ID ||
		importacao.Nota.EntregaID == nil || *importacao.Nota.EntregaID != c.entrega.ID {
		t.Errorf("vínculo = %v %v", importacao.Nota.PedidoID, importacao.Nota.EntregaID)
	}
}

func TestImportarAvisaNumeroDePedidoAusente(t *testing.T) {
	c := novoCenario(t)
	c.cliente.ExigeNumeroPedidoNaNF = true
	documento := c.notaDaMia(500)
	documento.Itens[1].Pedido = ""

	importacao := c.importar(t, c.financeiro, documento)
	if !slices.Contains(importacao.Avisos, portsin.AvisoNumeroPedidoAusente) {
		t.Errorf("avisos = %v, esperado numero_pedido_ausente", importacao.Avisos)
	}

	comPedido := c.importar(t, c.financeiro, c.notaDaMia(501))
	if slices.Contains(comPedido.Avisos, portsin.AvisoNumeroPedidoAusente) {
		t.Errorf("nota com xPed avisou falta de pedido: %v", comPedido.Avisos)
	}
}

func TestImportarRecusa(t *testing.T) {
	casos := []struct {
		nome     string
		montar   func(c *cenario) portsin.ImportarInput
		esperado error
	}{
		{"perfil sem financeiro", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.comercial, ArquivoID: c.enviar(c.comercial, c.notaDaMia(1).XML())}
		}, domain.ErrPermissao},
		{"arquivo de outro usuário", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.administrador, c.notaDaMia(1).XML())}
		}, domain.ErrPermissao},
		{"arquivo inexistente", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: uuid.New()}
		}, domain.ErrNotFound},
		{"arquivo que não é XML", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviarComTipo(c.financeiro, c.notaDaMia(1).XML(), "application/pdf")}
		}, domain.ErrValidacao},
		{"arquivo de solicitação", func(c *cenario) portsin.ImportarInput {
			id := c.enviar(c.financeiro, c.notaDaMia(1).XML())
			c.m.daSolicitacao[id] = true
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: id}
		}, domain.ErrValidacao},
		{"XML inválido", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, []byte("<pedido/>"))}
		}, domain.ErrValidacao},
		{"emitente fora do cadastro", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.administrador, ArquivoID: c.enviar(c.administrador, nfe.Nova(cnpjDesconhecido, cnpjCliente, 1).XML())}
		}, domain.ErrValidacao},
		{"emitente fora do acesso", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiroRed, ArquivoID: c.enviar(c.financeiroRed, c.notaDaMia(1).XML())}
		}, domain.ErrPermissao},
		{"emitente inativa", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, nfe.Nova(cnpjInativa, cnpjCliente, 1).XML())}
		}, domain.ErrValidacao},
		{"pedido inexistente", func(c *cenario) portsin.ImportarInput {
			pedido := uuid.New()
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, c.notaDaMia(1).XML()), PedidoID: &pedido}
		}, domain.ErrNotFound},
		{"entrega sem pedido", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, c.notaDaMia(1).XML()), EntregaID: &c.entrega.ID}
		}, domain.ErrValidacao},
		{"pedido de outro cliente", func(c *cenario) portsin.ImportarInput {
			return portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, nfe.Nova(cnpjMia, cpfClienteNovo, 1).XML()), PedidoID: &c.pedido.ID}
		}, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := novoCenario(t)
			_, err := c.notas.Importar(c.ctx, caso.montar(c))
			if !errors.Is(err, caso.esperado) {
				t.Errorf("erro = %v, esperado %v", err, caso.esperado)
			}
			if len(c.m.notas) != 0 || len(c.m.clientes) != 1 {
				t.Errorf("recusa gravou dados: %d notas, %d clientes", len(c.m.notas), len(c.m.clientes))
			}
		})
	}
}

func TestImportarRecusaChaveEArquivoRepetidos(t *testing.T) {
	c := novoCenario(t)
	primeira := c.importar(t, c.financeiro, c.notaDaMia(600))

	_, err := c.notas.Importar(c.ctx, portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: c.enviar(c.financeiro, c.notaDaMia(600).XML())})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("chave repetida = %v, esperado conflito", err)
	}

	_, err = c.notas.Importar(c.ctx, portsin.ImportarInput{UsuarioID: c.financeiro, ArquivoID: primeira.Nota.ArquivoID})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("arquivo repetido = %v, esperado conflito", err)
	}
}

func TestObterRespeitaEmpresaERepresentante(t *testing.T) {
	c := novoCenario(t)
	venda := c.importar(t, c.financeiro, c.notaDaMia(700)).Nota
	transferencia := c.importar(t, c.financeiro, nfe.Nova(cnpjMia, cnpjRed, 701)).Nota
	semCarteira := c.importar(t, c.financeiro, nfe.Nova(cnpjMia, cpfClienteNovo, 702)).Nota

	for nome, usuario := range map[string]uuid.UUID{"comercial": c.comercial, "financeiro": c.financeiro, "administrador": c.administrador} {
		if _, err := c.notas.Obter(c.ctx, venda.ID, usuario); err != nil {
			t.Errorf("%s = %v", nome, err)
		}
	}
	if _, err := c.notas.Obter(c.ctx, venda.ID, c.financeiroRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("outra empresa = %v, esperado permissão", err)
	}

	if _, err := c.notas.Obter(c.ctx, venda.ID, c.vendedor); err != nil {
		t.Errorf("representante do cliente = %v", err)
	}
	if _, err := c.notas.Obter(c.ctx, transferencia.ID, c.vendedor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("representante em transferência = %v, esperado permissão", err)
	}
	if _, err := c.notas.Obter(c.ctx, semCarteira.ID, c.vendedor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("representante em cliente de fora da carteira = %v, esperado permissão", err)
	}
	if _, err := c.notas.Obter(c.ctx, uuid.New(), c.financeiro); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("nota inexistente = %v, esperado não encontrado", err)
	}
}

func TestListarAplicaOEscopo(t *testing.T) {
	c := novoCenario(t)
	c.importar(t, c.financeiro, c.notaDaMia(800))
	c.importar(t, c.administrador, nfe.Nova(cnpjRed, cnpjCliente, 801))

	daMia, err := c.notas.Listar(c.ctx, portsin.ListarNotasInput{UsuarioID: c.financeiro})
	if err != nil || len(daMia) != 1 || daMia[0].EmpresaID != c.mia {
		t.Errorf("notas do financeiro da MIA = %d, %v", len(daMia), err)
	}
	if !mesmasEmpresas(c.m.filtroNotas.EmpresaIDs, c.mia, c.inativa) || c.m.filtroNotas.RepresentanteID != nil {
		t.Errorf("filtro = %+v", c.m.filtroNotas)
	}

	todas, err := c.notas.Listar(c.ctx, portsin.ListarNotasInput{UsuarioID: c.administrador})
	if err != nil || len(todas) != 2 || c.m.filtroNotas.EmpresaIDs != nil {
		t.Errorf("notas do administrador = %d, %v, filtro %v", len(todas), err, c.m.filtroNotas.EmpresaIDs)
	}

	if _, err := c.notas.Listar(c.ctx, portsin.ListarNotasInput{UsuarioID: c.vendedor}); err != nil ||
		c.m.filtroNotas.RepresentanteID == nil || *c.m.filtroNotas.RepresentanteID != c.representante.ID {
		t.Errorf("representante = %v, filtro %v", err, c.m.filtroNotas.RepresentanteID)
	}

	if _, err := c.notas.Listar(c.ctx, portsin.ListarNotasInput{UsuarioID: c.financeiro, Empresa: &c.red}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v, esperado permissão", err)
	}
}

func TestListarConverteOsFiltros(t *testing.T) {
	c := novoCenario(t)
	de := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)
	ate := time.Date(2026, 10, 31, 2, 0, 0, 0, time.UTC)
	verdadeiro := true

	if _, err := c.notas.Listar(c.ctx, portsin.ListarNotasInput{
		UsuarioID: c.administrador, Natureza: "Remessa", Status: "autorizada", EmitidaDe: &de, EmitidaAte: &ate,
		EntreEmpresas: &verdadeiro, SemPedido: true, Chave: "123",
	}); err != nil {
		t.Fatalf("Listar = %v", err)
	}
	filtro := c.m.filtroNotas
	if filtro.Natureza != domainnotafiscal.NaturezaRemessa || filtro.Status != domainnotafiscal.StatusAutorizada ||
		!filtro.EmitidaDe.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) || !filtro.EmitidaAte.Equal(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)) ||
		filtro.EntreEmpresas == nil || !filtro.SemPedido || filtro.Chave != "123" || filtro.Size != domain.TamanhoPaginaPadrao {
		t.Errorf("filtro = %+v", filtro)
	}

	invalidos := []portsin.ListarNotasInput{
		{UsuarioID: c.administrador, Natureza: "bonificacao"},
		{UsuarioID: c.administrador, Status: "denegada"},
		{UsuarioID: c.administrador, EmitidaDe: &ate, EmitidaAte: &de},
	}
	for _, input := range invalidos {
		if _, err := c.notas.Listar(c.ctx, input); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("filtro %+v = %v, esperado validação", input, err)
		}
	}
}

func TestVincularPedidoDepoisDaImportacao(t *testing.T) {
	c := novoCenario(t)
	nota := c.importar(t, c.financeiro, c.notaDaMia(900)).Nota

	vinculada, err := c.notas.Vincular(c.ctx, portsin.VincularInput{
		UsuarioID: c.financeiro, NotaFiscalID: nota.ID, Versao: 1, PedidoID: &c.pedido.ID, EntregaID: &c.entrega.ID,
	})
	if err != nil {
		t.Fatalf("Vincular = %v", err)
	}
	if vinculada.Versao != 2 || vinculada.PedidoID == nil || *vinculada.PedidoID != c.pedido.ID {
		t.Errorf("nota vinculada = versão %d pedido %v", vinculada.Versao, vinculada.PedidoID)
	}

	if _, err := c.notas.Vincular(c.ctx, portsin.VincularInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, Versao: 1}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	desfeita, err := c.notas.Vincular(c.ctx, portsin.VincularInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, Versao: 2})
	if err != nil || desfeita.PedidoID != nil || desfeita.EntregaID != nil {
		t.Errorf("desfazer = %v, %v", desfeita, err)
	}

	if _, err := c.notas.Vincular(c.ctx, portsin.VincularInput{UsuarioID: c.comercial, NotaFiscalID: nota.ID, Versao: 3}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial = %v, esperado permissão", err)
	}
	if _, err := c.notas.Vincular(c.ctx, portsin.VincularInput{UsuarioID: c.financeiroRed, NotaFiscalID: nota.ID, Versao: 3}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("financeiro de outra empresa = %v, esperado permissão", err)
	}

	proposta := &domainpedido.Pedido{ID: uuid.New(), EmpresaID: c.mia, ClienteID: c.cliente.ID, Status: domainpedido.StatusProposta}
	c.m.pedidos[proposta.ID] = proposta
	if _, err := c.notas.Vincular(c.ctx, portsin.VincularInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, Versao: 3, PedidoID: &proposta.ID}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pedido em proposta = %v, esperado validação", err)
	}
}

func TestVincularProdutoAoItem(t *testing.T) {
	c := novoCenario(t)
	nota := c.importar(t, c.financeiro, c.notaDaMia(1000)).Nota

	alterada, err := c.notas.VincularProduto(c.ctx, portsin.VincularProdutoInput{
		UsuarioID: c.financeiro, NotaFiscalID: nota.ID, NumeroItem: 2, Versao: 1, ProdutoID: c.rejunte.ID,
	})
	if err != nil {
		t.Fatalf("VincularProduto = %v", err)
	}
	if alterada.Itens[1].ProdutoID == nil || *alterada.Itens[1].ProdutoID != c.rejunte.ID || alterada.Versao != 2 {
		t.Errorf("item = %v versão %d", alterada.Itens[1].ProdutoID, alterada.Versao)
	}
	if gravada := c.m.notas[nota.ID]; gravada.Itens[1].ProdutoID == nil {
		t.Error("produto do item não foi gravado")
	}

	casos := map[string]struct {
		input    portsin.VincularProdutoInput
		esperado error
	}{
		"produto inexistente": {portsin.VincularProdutoInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, NumeroItem: 1, Versao: 2, ProdutoID: uuid.New()}, domain.ErrNotFound},
		"item inexistente":    {portsin.VincularProdutoInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, NumeroItem: 9, Versao: 2, ProdutoID: c.piso.ID}, domain.ErrNotFound},
		"produto vazio":       {portsin.VincularProdutoInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, NumeroItem: 1, Versao: 2}, domain.ErrValidacao},
		"sem versão":          {portsin.VincularProdutoInput{UsuarioID: c.financeiro, NotaFiscalID: nota.ID, NumeroItem: 1, ProdutoID: c.piso.ID}, domain.ErrValidacao},
		"comercial":           {portsin.VincularProdutoInput{UsuarioID: c.comercial, NotaFiscalID: nota.ID, NumeroItem: 1, Versao: 2, ProdutoID: c.piso.ID}, domain.ErrPermissao},
	}
	for nome, caso := range casos {
		if _, err := c.notas.VincularProduto(c.ctx, caso.input); !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}
}

func TestBaixarXML(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(1100)
	nota := c.importar(t, c.financeiro, documento).Nota

	arquivo, conteudo, err := c.notas.BaixarXML(c.ctx, nota.ID, c.comercial)
	if err != nil {
		t.Fatalf("BaixarXML = %v", err)
	}
	defer func() { _ = conteudo.Close() }()
	lido, _ := io.ReadAll(conteudo)
	if arquivo.ID != nota.ArquivoID || string(lido) != string(documento.XML()) {
		t.Errorf("XML baixado = %q", arquivo.Nome)
	}

	if _, _, err := c.notas.BaixarXML(c.ctx, nota.ID, c.financeiroRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("outra empresa = %v, esperado permissão", err)
	}
}

func TestConsultasDePendencia(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.notas.ListarItensSemProduto(c.ctx, portsin.ListarItensSemProdutoInput{UsuarioID: c.financeiro}); err != nil ||
		!mesmasEmpresas(c.m.filtroPendentes.EmpresaIDs, c.mia, c.inativa) {
		t.Errorf("itens sem produto = %v, filtro %+v", err, c.m.filtroPendentes)
	}
	if _, err := c.notas.ListarItensSemProduto(c.ctx, portsin.ListarItensSemProdutoInput{UsuarioID: c.vendedor}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("representante em itens sem produto = %v, esperado permissão", err)
	}
	if _, err := c.notas.ListarItensSemProduto(c.ctx, portsin.ListarItensSemProdutoInput{UsuarioID: c.financeiro, Empresa: &c.red}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v, esperado permissão", err)
	}

	if _, err := c.notas.ListarEntregasSemNota(c.ctx, portsin.ListarEntregasSemNotaInput{UsuarioID: c.vendedor, ClienteID: &c.cliente.ID}); err != nil ||
		c.m.filtroEntregas.RepresentanteID == nil || *c.m.filtroEntregas.RepresentanteID != c.representante.ID ||
		c.m.filtroEntregas.ClienteID == nil {
		t.Errorf("entregas sem nota do representante = %v, filtro %+v", err, c.m.filtroEntregas)
	}
	if _, err := c.notas.ListarEntregasSemNota(c.ctx, portsin.ListarEntregasSemNotaInput{UsuarioID: c.administrador}); err != nil ||
		c.m.filtroEntregas.EmpresaIDs != nil || c.m.filtroEntregas.RepresentanteID != nil {
		t.Errorf("entregas sem nota do administrador = %v, filtro %+v", err, c.m.filtroEntregas)
	}
}

func mesmasEmpresas(obtidas []uuid.UUID, esperadas ...uuid.UUID) bool {
	if len(obtidas) != len(esperadas) {
		return false
	}
	for _, esperada := range esperadas {
		if !slices.Contains(obtidas, esperada) {
			return false
		}
	}

	return true
}
