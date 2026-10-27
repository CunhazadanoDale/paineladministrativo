package tabelapreco_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	tabelaprecousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/tabelapreco"
	"github.com/google/uuid"
)

var (
	_ portsout.TabelaPrecoImportacaoRepository = (*repoItens)(nil)
	_ portsout.ProdutoImportacaoRepository     = repoProdutos{}
)

func (r *repoItens) ExisteVigencia(_ context.Context, produtoID uuid.UUID, modelo domainpedido.Modelo, vigenteDe time.Time) (bool, error) {
	for _, item := range r.criados {
		if item.ProdutoID == produtoID && item.Modelo == modelo && item.VigenteDe.Equal(vigenteDe) {
			return true, nil
		}
	}

	return false, nil
}

func (r repoProdutos) IDsPorCodigos(_ context.Context, codigos []string) (map[string]uuid.UUID, error) {
	encontrados := map[string]uuid.UUID{}
	for _, codigo := range codigos {
		for _, produto := range r.m.produtos {
			if produto.Codigo != nil && *produto.Codigo == codigo {
				encontrados[codigo] = produto.ID
			}
		}
	}

	return encontrados, nil
}

func (c *cenario) comCodigo(produtoID uuid.UUID, codigo string) {
	c.m.produtos[produtoID].Codigo = &codigo
}

func (c *cenario) importacao() *tabelaprecousecases.ImportacaoUsecaseImpl {
	return tabelaprecousecases.NewImportacaoUsecase(c.itens, repoProdutos{c.m}, repoUsuarios{c.m}, repoCargos{c.m})
}

func linhaDePreco(numero int, codigo, modelo, preco, vigenteDe string) portsin.LinhaImportacao {
	return portsin.LinhaImportacao{Numero: numero, Codigo: codigo, Modelo: modelo, Preco: preco, VigenteDe: vigenteDe}
}

func TestImportarTabelaDePrecosGravaValidosERejeitaInvalidos(t *testing.T) {
	c := novoCenario()
	c.comCodigo(c.ativo, "POR-60")
	c.comCodigo(c.inativo, "VELHO")
	if _, err := c.usecase.Criar(c.ctx, portsin.CriarItemInput{UsuarioID: c.administrador, Dados: c.dados(c.ativo)}); err != nil {
		t.Fatalf("lançamento existente falhou: %v", err)
	}

	relatorio, err := c.importacao().Importar(c.ctx, portsin.ImportarItensInput{
		UsuarioID: c.administrador,
		Linhas: []portsin.LinhaImportacao{
			linhaDePreco(2, "POR-60", "pronta_entrega", "R$ 139,90", "01/12/2026"),
			linhaDePreco(3, "POR-60", "Importacao", "99,90", "2026-12-01"),
			linhaDePreco(4, "POR-60", "pronta_entrega", "149,90", "1/12/2026"),
			linhaDePreco(5, "POR-60", "pronta_entrega", "129,90", "01/11/2026"),
			linhaDePreco(6, "NAO-EXISTE", "pronta_entrega", "10", "01/12/2026"),
			linhaDePreco(7, "VELHO", "pronta_entrega", "10", "01/12/2026"),
			linhaDePreco(8, "POR-60", "atacado", "10", "01/12/2026"),
			linhaDePreco(9, "POR-60", "pronta_entrega", "", "02/12/2026"),
			linhaDePreco(10, "POR-60", "pronta_entrega", "10", ""),
			linhaDePreco(11, "", "pronta_entrega", "10", "02/12/2026"),
		},
	})

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if relatorio.Total != 10 || relatorio.Importadas != 2 {
		t.Errorf("relatório = %+v", relatorio)
	}
	esperados := map[int]string{
		4:  "produto, modelo e início repetidos no arquivo (linha 2)",
		5:  "já existe preço deste produto e modelo com início nesta data",
		6:  "produto com código NAO-EXISTE não encontrado",
		7:  "produto inativo não recebe preço novo",
		9:  "preco é obrigatório",
		10: "vigente_de é obrigatório",
		11: "código do produto é obrigatório",
	}
	motivos := map[int]string{}
	for _, rejeitada := range relatorio.Rejeitadas {
		motivos[rejeitada.Linha] = rejeitada.Motivo
	}
	for linha, motivo := range esperados {
		if motivos[linha] != motivo {
			t.Errorf("linha %d: motivo %q, esperado %q", linha, motivos[linha], motivo)
		}
	}
	if motivos[8] == "" || len(motivos) != 8 {
		t.Errorf("motivos = %v", motivos)
	}

	if len(c.itens.criados) != 3 {
		t.Fatalf("lançamentos = %d, esperado 3", len(c.itens.criados))
	}
	novo := c.itens.criados[1]
	if novo.PrecoCentavos != 13990 || novo.Modelo != domainpedido.ModeloDe("pronta_entrega") || novo.UsuarioID != c.administrador ||
		!novo.VigenteDe.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lançamento importado = %+v", novo)
	}
}

func TestImportarTabelaDePrecosSimulandoNaoGrava(t *testing.T) {
	c := novoCenario()
	c.comCodigo(c.ativo, "POR-60")

	relatorio, err := c.importacao().Importar(c.ctx, portsin.ImportarItensInput{
		UsuarioID: c.administrador,
		Simular:   true,
		Linhas:    []portsin.LinhaImportacao{linhaDePreco(2, "POR-60", "pronta_entrega", "10", "01/12/2026")},
	})

	if err != nil || !relatorio.Simulacao || relatorio.Importadas != 1 {
		t.Fatalf("relatório = %+v, erro = %v", relatorio, err)
	}
	if len(c.itens.criados) != 0 {
		t.Errorf("simulação gravou %d lançamentos", len(c.itens.criados))
	}
}

func TestImportarTabelaDePrecosExigeAdministrador(t *testing.T) {
	c := novoCenario()

	for _, usuario := range []uuid.UUID{c.comercial, c.financeiro, c.semPerfil} {
		if _, err := c.importacao().Importar(c.ctx, portsin.ImportarItensInput{UsuarioID: usuario}); !errors.Is(err, domain.ErrPermissao) {
			t.Errorf("erro = %v, esperado permissão", err)
		}
	}
}
