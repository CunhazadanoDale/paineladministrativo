package notafiscal_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
)

func TestImportarVendaGeraTitulosDasDuplicatas(t *testing.T) {
	c := novoCenario(t)

	importacao := c.importar(t, c.financeiro, c.notaDaMia(123))

	if len(c.m.geracoes) != 1 || len(importacao.TitulosGerados) != 2 || importacao.ErroDosTitulos != nil {
		t.Fatalf("gerações %d, títulos %v, erro %v", len(c.m.geracoes), importacao.TitulosGerados, importacao.ErroDosTitulos)
	}
	geracao := c.m.geracoes[0]
	nota := importacao.Nota
	if geracao.Origem != domainreceber.OrigemNotaFiscalDuplicata || geracao.EmpresaID != c.mia || geracao.ClienteID != c.cliente.ID ||
		geracao.UsuarioID != c.financeiro {
		t.Errorf("geração = %+v", geracao)
	}
	primeiro := geracao.Titulos[0]
	if *primeiro.OrigemID != nota.Duplicatas[0].ID || primeiro.Numero != "NF 1-123" || primeiro.Parcela != 1 ||
		primeiro.ValorCentavos != 54500 || primeiro.ContaBancariaID != nil || primeiro.Status != domainreceber.StatusCarteira {
		t.Errorf("primeiro título = %+v", primeiro)
	}
	if slices.Contains(importacao.Avisos, portsin.AvisoTitulosPendentes) {
		t.Errorf("avisos = %v", importacao.Avisos)
	}
}

func TestImportarSemTitulosForaDaVenda(t *testing.T) {
	casos := map[string]nfe.NFe{
		"entre empresas": nfe.Nova(cnpjMia, cnpjRed, 10),
		"sem duplicatas": func() nfe.NFe { n := nfe.Nova(cnpjMia, cnpjCliente, 11); n.Duplicatas = nil; return n }(),
		"remessa": func() nfe.NFe {
			n := nfe.Nova(cnpjMia, cnpjCliente, 12)
			n.Itens[0].CFOP, n.Itens[1].CFOP = "5116", "5116"
			return n
		}(),
		"devolução": func() nfe.NFe { n := nfe.Nova(cnpjMia, cnpjCliente, 13); n.Finalidade = "4"; return n }(),
	}

	for nome, documento := range casos {
		c := novoCenario(t)
		importacao := c.importar(t, c.financeiro, documento)
		if len(c.m.geracoes) != 0 || len(importacao.TitulosGerados) != 0 {
			t.Errorf("%s gerou títulos: %d", nome, len(c.m.geracoes))
		}
	}
}

func TestImportarComFalhaNosTitulosGuardaANotaEAvisa(t *testing.T) {
	c := novoCenario(t)
	falha := errors.New("conexão perdida")
	c.m.erroDosTitulos = falha

	importacao := c.importar(t, c.financeiro, c.notaDaMia(124))

	if !errors.Is(importacao.ErroDosTitulos, falha) || !slices.Contains(importacao.Avisos, portsin.AvisoTitulosPendentes) {
		t.Errorf("erro %v avisos %v", importacao.ErroDosTitulos, importacao.Avisos)
	}
	if _, ok := c.m.notas[importacao.Nota.ID]; !ok {
		t.Fatal("a nota deveria continuar gravada")
	}

	c.m.erroDosTitulos = nil
	sincronizacao, err := c.notas.SincronizarTitulos(c.ctx, importacao.Nota.ID, c.financeiro)
	if err != nil || len(sincronizacao.Gerados) != 2 || len(sincronizacao.JaExistentes) != 0 {
		t.Fatalf("sincronizar = %+v, %v", sincronizacao, err)
	}

	repetida, err := c.notas.SincronizarTitulos(c.ctx, importacao.Nota.ID, c.financeiro)
	if err != nil || len(repetida.Gerados) != 0 || len(repetida.JaExistentes) != 2 {
		t.Errorf("sincronizar de novo = %+v, %v", repetida, err)
	}
}

func TestSincronizarTitulosRecusa(t *testing.T) {
	c := novoCenario(t)
	venda := c.importar(t, c.financeiro, c.notaDaMia(125)).Nota
	transferencia := c.importar(t, c.financeiro, nfe.Nova(cnpjMia, cnpjRed, 126)).Nota

	if _, err := c.notas.SincronizarTitulos(c.ctx, transferencia.ID, c.financeiro); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("transferência = %v, esperado validação", err)
	}
	if _, err := c.notas.SincronizarTitulos(c.ctx, venda.ID, c.comercial); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial = %v, esperado permissão", err)
	}
	if _, err := c.notas.SincronizarTitulos(c.ctx, venda.ID, c.financeiroRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("outra empresa = %v, esperado permissão", err)
	}
}

func TestCancelamentoDaNotaCancelaOsTitulos(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(127)
	nota := c.importar(t, c.financeiro, documento).Nota

	importacao, err := c.importarEvento(c.financeiro, nfe.Cancelamento(documento.Chave(), cnpjMia))
	if err != nil {
		t.Fatalf("cancelamento = %v", err)
	}
	if len(c.m.cancelamentos) != 1 || len(importacao.TitulosCancelados) != 2 || importacao.ErroDosTitulos != nil {
		t.Fatalf("cancelamentos %d, títulos %v, erro %v", len(c.m.cancelamentos), importacao.TitulosCancelados, importacao.ErroDosTitulos)
	}
	cancelamento := c.m.cancelamentos[0]
	if cancelamento.Origem != domainreceber.OrigemNotaFiscalDuplicata || len(cancelamento.OrigemIDs) != 2 ||
		cancelamento.Motivo != "cancelamento da NF-e "+documento.Chave() || cancelamento.UsuarioID != c.financeiro {
		t.Errorf("cancelamento = %+v", cancelamento)
	}

	sincronizacao, err := c.notas.SincronizarTitulos(c.ctx, nota.ID, c.financeiro)
	if err != nil || len(sincronizacao.Cancelados) != 0 || sincronizacao.Ignorados != 2 || len(c.m.cancelamentos) != 2 {
		t.Errorf("sincronizar nota cancelada = %+v, %v", sincronizacao, err)
	}
}

func TestCancelamentoComFalhaNosTitulosAvisa(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(128)
	c.importar(t, c.financeiro, documento)
	c.m.erroDosTitulos = errors.New("conexão perdida")

	importacao, err := c.importarEvento(c.financeiro, nfe.Cancelamento(documento.Chave(), cnpjMia))
	if err != nil {
		t.Fatalf("cancelamento = %v", err)
	}
	if importacao.ErroDosTitulos == nil || !slices.Contains(importacao.Avisos, portsin.AvisoTitulosPendentes) ||
		importacao.Nota.Status != "cancelada" {
		t.Errorf("importação = erro %v avisos %v status %q", importacao.ErroDosTitulos, importacao.Avisos, importacao.Nota.Status)
	}
}

func TestCartaDeCorrecaoNaoMexeNosTitulos(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(129)
	c.importar(t, c.financeiro, documento)

	if _, err := c.importarEvento(c.financeiro, nfe.CartaCorrecao(documento.Chave(), cnpjMia, 1)); err != nil {
		t.Fatalf("carta = %v", err)
	}
	if len(c.m.cancelamentos) != 0 {
		t.Errorf("carta de correção cancelou títulos: %d", len(c.m.cancelamentos))
	}
}
