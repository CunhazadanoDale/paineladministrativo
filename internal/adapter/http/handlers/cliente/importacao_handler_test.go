package cliente

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type importacaoUseCaseFalso struct {
	input portsin.ImportarClientesInput
}

func (f *importacaoUseCaseFalso) Importar(_ context.Context, input portsin.ImportarClientesInput) (*domaincliente.RelatorioImportacao, error) {
	f.input = input
	relatorio := &domaincliente.RelatorioImportacao{Simulacao: input.Simular, Total: len(input.Linhas), Importadas: len(input.Linhas) - 1}
	relatorio.Rejeitar(3, "123", "documento deve ser um CPF ou CNPJ válido")

	return relatorio, nil
}

const cabecalhoCSV = "documento;nome;nome_fantasia;tipo;grupo;uf;cidade;email_nf;telefone;representante_documento;exige_numero_pedido_na_nf;proibe_antecipacao;observacoes"

func TestImportarLeCSVComBOMAspasEQuebraWindows(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	corpo := "\xef\xbb\xbf" + strings.ToUpper(cabecalhoCSV) + "\r\n" +
		"11.222.333/0001-81;Construtora Águia;Águia;construtora;Grupo Águia;PB;João Pessoa;nf@aguia.com.br;;529.982.247-25;sim;nao;\"pagar; sempre dia 10\"\r\n" +
		"\r\n" +
		"123;Loja;;loja;;PB;JP;;;;;;\r\n"

	resposta := executaProtegido(t, NewImportacaoHandler(usecase).Importar, http.MethodPost, "/api/v1/clientes/importacao?simular=true", "", corpo)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if !usecase.input.Simular || len(usecase.input.Linhas) != 2 || usecase.input.UsuarioID == uuid.Nil {
		t.Fatalf("input repassado = %+v", usecase.input)
	}
	primeira := usecase.input.Linhas[0]
	if primeira.Numero != 2 || primeira.Documento != "11.222.333/0001-81" || primeira.Grupo != "Grupo Águia" ||
		primeira.RepresentanteDocumento != "529.982.247-25" || primeira.ExigeNumeroPedidoNaNF != "sim" || primeira.Observacoes != "pagar; sempre dia 10" {
		t.Errorf("primeira linha = %+v", primeira)
	}
	if usecase.input.Linhas[1].Numero != 4 {
		t.Errorf("número da segunda linha = %d, esperado 4 (a linha em branco conta)", usecase.input.Linhas[1].Numero)
	}

	var envelope dto.Resposta[clientedto.RelatorioImportacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if !envelope.Dados.Simulacao || envelope.Dados.Total != 2 || len(envelope.Dados.Rejeitadas) != 1 || envelope.Dados.Rejeitadas[0].Linha != 3 {
		t.Errorf("relatório devolvido = %+v", envelope.Dados)
	}
}

func TestImportarRecusaArquivoMalFormado(t *testing.T) {
	casos := []struct {
		nome     string
		corpo    string
		mensagem string
	}{
		{"vazio", "", "erro de validação: arquivo vazio"},
		{"cabeçalho errado", "documento;nome\n", "erro de validação: linha 1 deve ter 13 colunas separadas por ;"},
		{"coluna trocada", strings.Replace(cabecalhoCSV, "grupo", "grupo_cliente", 1) + "\n", "erro de validação: cabeçalho deve ser " + cabecalhoCSV},
		{"linha com colunas a menos", cabecalhoCSV + "\n11222333000181;Águia\n", "erro de validação: linha 2 deve ter 13 colunas separadas por ;"},
		{"aspas abertas", cabecalhoCSV + "\n\"11222333000181;Águia;;construtora;;PB;JP;;;;;;\n", "erro de validação: csv inválido na linha 2"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			usecase := &importacaoUseCaseFalso{}
			resposta := executaProtegido(t, NewImportacaoHandler(usecase).Importar, http.MethodPost, "/api/v1/clientes/importacao", "", caso.corpo)

			apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
			if mensagem := apoioteste.MensagemDoEnvelope(t, resposta); mensagem != caso.mensagem {
				t.Errorf("mensagem %q, esperado %q", mensagem, caso.mensagem)
			}
			if usecase.input.Linhas != nil {
				t.Error("não deveria chegar ao usecase com arquivo inválido")
			}
		})
	}
}

func TestImportarRecusaArquivoAcimaDe5MB(t *testing.T) {
	linha := "\n11222333000181;" + strings.Repeat("a", 2000) + ";;construtora;;PB;JP;;;;;;"
	corpo := cabecalhoCSV + strings.Repeat(linha, 3000)

	resposta := executaProtegido(t, NewImportacaoHandler(&importacaoUseCaseFalso{}).Importar, http.MethodPost, "/api/v1/clientes/importacao", "", corpo)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusRequestEntityTooLarge)
}

func TestImportarRecusaMaisDe5000Linhas(t *testing.T) {
	corpo := cabecalhoCSV + strings.Repeat("\n11222333000181;A;;construtora;;PB;JP;;;;;;", 5001)

	resposta := executaProtegido(t, NewImportacaoHandler(&importacaoUseCaseFalso{}).Importar, http.MethodPost, "/api/v1/clientes/importacao", "", corpo)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestConverterLeadRepassaLeadEDados(t *testing.T) {
	cliente := clienteDeTeste(t)
	usecase := &clienteUseCaseFalso{cliente: cliente}
	leadID := uuid.New()

	resposta := executaProtegido(t, NewClienteHandler(usecase).ConverterLead, http.MethodPost, "/api/v1/leads/"+leadID.String()+"/converter", leadID.String(),
		`{"documento":"11.222.333/0001-81","tipo":"construtora","uf":"PB","cidade":"João Pessoa"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	input := usecase.inputConverter
	if input.LeadID != leadID || input.Documento != "11.222.333/0001-81" || input.Dados.Tipo != "construtora" || input.Dados.Nome != "" {
		t.Errorf("input repassado = %+v", input)
	}
}

func TestImportarConverteCSVDoExcelEmWindows1252(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	corpo := cabecalhoCSV + "\r\n11222333000181;Constru\xe7\xe3o \x93\xc1guia\x94;;construtora;;SP;S\xe3o Paulo;;;;;;\r\n"

	resposta := executaProtegido(t, NewImportacaoHandler(usecase).Importar, http.MethodPost, "/api/v1/clientes/importacao", "", corpo)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if len(usecase.input.Linhas) != 1 {
		t.Fatalf("linhas repassadas = %+v", usecase.input.Linhas)
	}
	if linha := usecase.input.Linhas[0]; linha.Nome != "Construção “Águia”" || linha.Cidade != "São Paulo" {
		t.Errorf("linha convertida = %+v", linha)
	}
}

func TestImportarRecusaSimularComValorInvalido(t *testing.T) {
	for _, valor := range []string{"sim", "talvez"} {
		usecase := &importacaoUseCaseFalso{}
		resposta := executaProtegido(t, NewImportacaoHandler(usecase).Importar, http.MethodPost,
			"/api/v1/clientes/importacao?simular="+valor, "", cabecalhoCSV+"\n")

		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
		if usecase.input.Linhas != nil {
			t.Errorf("simular=%s chegou ao usecase", valor)
		}
	}

	usecase := &importacaoUseCaseFalso{}
	resposta := executaProtegido(t, NewImportacaoHandler(usecase).Importar, http.MethodPost, "/api/v1/clientes/importacao?simular=false", "", cabecalhoCSV+"\n")
	if resposta.Code != http.StatusOK || usecase.input.Simular {
		t.Errorf("simular=false: status %d, simular %v", resposta.Code, usecase.input.Simular)
	}
}

func TestModeloDeClientesDevolveCabecalhoEExemplo(t *testing.T) {
	resposta := executaProtegido(t, NewImportacaoHandler(&importacaoUseCaseFalso{}).Modelo, http.MethodGet, "/api/v1/clientes/importacao/modelo", "", "")

	if resposta.Code != http.StatusOK || !strings.HasPrefix(resposta.Body.String(), "\xef\xbb\xbf"+cabecalhoCSV+"\r\n11.222.333/0001-81;") {
		t.Errorf("status %d, corpo %q", resposta.Code, resposta.Body.String())
	}
	if disposicao := resposta.Header().Get("Content-Disposition"); !strings.Contains(disposicao, "modelo-clientes.csv") {
		t.Errorf("Content-Disposition = %q", disposicao)
	}
}
