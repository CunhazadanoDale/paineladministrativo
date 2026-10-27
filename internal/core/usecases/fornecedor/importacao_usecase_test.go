package fornecedor_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
)

func linhaNacional(numero int, documento, nome string) portsin.LinhaImportacao {
	return portsin.LinhaImportacao{Numero: numero, Documento: documento, Nome: nome, UF: "PB", Cidade: "João Pessoa"}
}

func linhaEstrangeira(numero int, pais, identificacao, nome string) portsin.LinhaImportacao {
	return portsin.LinhaImportacao{Numero: numero, Pais: pais, IdentificacaoEstrangeira: identificacao, Nome: nome, MoedaPadrao: "usd"}
}

func TestImportarFornecedoresGravaValidosERejeitaInvalidos(t *testing.T) {
	c := novoCenario()
	if _, err := c.usecase.Criar(c.ctx, portsin.CriarFornecedorInput{UsuarioID: c.financeiro, Pais: "BR", Documento: texto("33.400.004/0001-16"), Dados: portsin.DadosFornecedorInput{Nome: "Já cadastrado"}}); err != nil {
		t.Fatalf("criação do fornecedor existente falhou: %v", err)
	}

	relatorio, err := c.usecase.Importar(c.ctx, portsin.ImportarFornecedoresInput{
		UsuarioID: c.financeiro,
		Linhas: []portsin.LinhaImportacao{
			linhaNacional(2, "11.222.333/0001-81", "Cerâmica Paraíba"),
			linhaEstrangeira(3, "cn", "91330200ma2x", "Foshan Ceramics"),
			linhaNacional(4, "11222333000181", "Repetida"),
			linhaEstrangeira(5, "CN", "91330200MA2X", "Foshan de novo"),
			linhaEstrangeira(6, "US", "91330200MA2X", "Mesmo id em outro país"),
			linhaNacional(7, "33400004000116", "Já existe"),
			linhaNacional(8, "", "Sem documento"),
			linhaNacional(9, "123", "Documento ruim"),
			linhaEstrangeira(10, "BRA", "X1", "País ruim"),
			linhaNacional(11, "52998224725", ""),
		},
	})

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if relatorio.Total != 10 || relatorio.Importadas != 3 {
		t.Errorf("relatório = %+v", relatorio)
	}
	esperados := map[int]string{
		4: "documento repetido no arquivo (linha 2)",
		5: "identificação estrangeira repetida no arquivo (linha 3)",
		7: "já existe um fornecedor com este documento",
	}
	motivos := map[int]string{}
	for _, rejeitada := range relatorio.Rejeitadas {
		motivos[rejeitada.Linha] = rejeitada.Motivo
		if rejeitada.Linha == 4 && rejeitada.Chave != "11222333000181" {
			t.Errorf("chave da linha 4 = %q", rejeitada.Chave)
		}
	}
	for linha, motivo := range esperados {
		if motivos[linha] != motivo {
			t.Errorf("linha %d: motivo %q, esperado %q", linha, motivos[linha], motivo)
		}
	}
	for _, linha := range []int{8, 9, 10, 11} {
		if motivos[linha] == "" {
			t.Errorf("linha %d deveria ser rejeitada", linha)
		}
	}
	if len(c.fornecedores.itens) != 4 {
		t.Errorf("fornecedores gravados = %d, esperado 4", len(c.fornecedores.itens))
	}
}

func TestImportarFornecedoresSimulandoNaoGrava(t *testing.T) {
	c := novoCenario()

	relatorio, err := c.usecase.Importar(c.ctx, portsin.ImportarFornecedoresInput{
		UsuarioID: c.administrador,
		Simular:   true,
		Linhas:    []portsin.LinhaImportacao{linhaNacional(2, "11.222.333/0001-81", "Cerâmica")},
	})

	if err != nil || !relatorio.Simulacao || relatorio.Importadas != 1 {
		t.Fatalf("relatório = %+v, erro = %v", relatorio, err)
	}
	if len(c.fornecedores.itens) != 0 {
		t.Errorf("simulação gravou %d fornecedores", len(c.fornecedores.itens))
	}
}

func TestImportarFornecedoresExigeFinanceiro(t *testing.T) {
	c := novoCenario()

	if _, err := c.usecase.Importar(c.ctx, portsin.ImportarFornecedoresInput{UsuarioID: c.comercial}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("erro = %v, esperado permissão", err)
	}
}
