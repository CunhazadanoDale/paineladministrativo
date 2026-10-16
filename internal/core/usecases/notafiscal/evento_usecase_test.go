package notafiscal_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
	"github.com/google/uuid"
)

func (c *cenario) importarEvento(usuarioID uuid.UUID, evento nfe.Evento) (*portsin.ImportacaoEvento, error) {
	return c.eventos.Importar(c.ctx, portsin.ImportarEventoInput{UsuarioID: usuarioID, ArquivoID: c.enviar(usuarioID, evento.XML())})
}

func TestImportarCancelamentoCancelaANota(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(10)
	nota := c.importar(t, c.financeiro, documento).Nota

	importacao, err := c.importarEvento(c.financeiro, nfe.Cancelamento(documento.Chave(), cnpjMia))
	if err != nil {
		t.Fatalf("Importar evento = %v", err)
	}
	if importacao.Nota.Status != domainnotafiscal.StatusCancelada || importacao.Nota.Versao != 2 {
		t.Errorf("nota = %q versão %d", importacao.Nota.Status, importacao.Nota.Versao)
	}
	if importacao.Evento.Tipo != domainnotafiscal.TipoEventoCancelamento || importacao.Evento.UsuarioID != c.financeiro {
		t.Errorf("evento = %+v", importacao.Evento)
	}
	if c.m.notas[nota.ID].Status != domainnotafiscal.StatusCancelada {
		t.Error("cancelamento não foi gravado")
	}

	eventos, err := c.eventos.Listar(c.ctx, nota.ID, c.comercial)
	if err != nil || len(eventos) != 1 {
		t.Errorf("eventos = %d, %v", len(eventos), err)
	}

	if _, err := c.importarEvento(c.financeiro, nfe.Cancelamento(documento.Chave(), cnpjMia)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar de novo = %v, esperado conflito", err)
	}
}

func TestImportarCartaDeCorrecao(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(11)
	nota := c.importar(t, c.financeiro, documento).Nota

	importacao, err := c.importarEvento(c.financeiro, nfe.CartaCorrecao(documento.Chave(), cnpjMia, 1))
	if err != nil {
		t.Fatalf("Importar carta = %v", err)
	}
	if importacao.Nota.Status != domainnotafiscal.StatusAutorizada || importacao.Evento.Tipo != domainnotafiscal.TipoEventoCartaCorrecao {
		t.Errorf("carta = nota %q evento %q", importacao.Nota.Status, importacao.Evento.Tipo)
	}

	if _, err := c.importarEvento(c.financeiro, nfe.CartaCorrecao(documento.Chave(), cnpjMia, 1)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("mesma sequência = %v, esperado conflito", err)
	}
	if _, err := c.importarEvento(c.financeiro, nfe.CartaCorrecao(documento.Chave(), cnpjMia, 2)); err != nil {
		t.Errorf("segunda carta = %v", err)
	}

	eventos, _ := c.eventos.Listar(c.ctx, nota.ID, c.financeiro)
	if len(eventos) != 2 {
		t.Errorf("eventos = %d, esperado 2", len(eventos))
	}
}

func TestImportarEventoRecusa(t *testing.T) {
	c := novoCenario(t)
	documento := c.notaDaMia(12)
	nota := c.importar(t, c.financeiro, documento).Nota

	if _, err := c.importarEvento(c.financeiro, nfe.Cancelamento(c.notaDaMia(99).Chave(), cnpjMia)); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("nota não importada = %v, esperado não encontrado", err)
	}
	if _, err := c.importarEvento(c.financeiroRed, nfe.Cancelamento(documento.Chave(), cnpjMia)); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("outra empresa = %v, esperado permissão", err)
	}
	if _, err := c.importarEvento(c.comercial, nfe.Cancelamento(documento.Chave(), cnpjMia)); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial = %v, esperado permissão", err)
	}
	if _, err := c.importarEvento(c.financeiro, nfe.Cancelamento(documento.Chave(), cnpjCliente)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("autor diferente do emitente = %v, esperado validação", err)
	}
	if _, err := c.eventos.Importar(c.ctx, portsin.ImportarEventoInput{UsuarioID: c.financeiro, ArquivoID: nota.ArquivoID}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("arquivo da nota como evento = %v, esperado conflito", err)
	}
	if _, err := c.eventos.Listar(c.ctx, nota.ID, c.financeiroRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("listar de outra empresa = %v, esperado permissão", err)
	}

	if c.m.notas[nota.ID].Status != domainnotafiscal.StatusAutorizada || len(c.m.eventos) != 0 {
		t.Error("evento recusado alterou a nota")
	}
}
