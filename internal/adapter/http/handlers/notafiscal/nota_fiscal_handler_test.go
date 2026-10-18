package notafiscal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	notafiscaldto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	"github.com/google/uuid"
)

type notaUseCaseFalso struct {
	erro           error
	nota           *domainnotafiscal.NotaFiscal
	avisos         []string
	inputImportar  portsin.ImportarInput
	inputListar    portsin.ListarNotasInput
	inputVincular  portsin.VincularInput
	inputProduto   portsin.VincularProdutoInput
	inputPendentes portsin.ListarItensSemProdutoInput
	inputEntregas  portsin.ListarEntregasSemNotaInput
	conteudoDoXML  []byte
	arquivoDoXML   *domainsolicitacao.Arquivo
	notaBaixada    uuid.UUID
	sincronizacao  *portsin.SincronizacaoTitulos
	sincronizada   uuid.UUID
	erroDosTitulos error
}

func (f *notaUseCaseFalso) Importar(_ context.Context, input portsin.ImportarInput) (*portsin.Importacao, error) {
	f.inputImportar = input
	return &portsin.Importacao{Nota: f.nota, Avisos: f.avisos, ErroDosTitulos: f.erroDosTitulos}, f.erro
}

func (f *notaUseCaseFalso) Obter(context.Context, uuid.UUID, uuid.UUID) (*domainnotafiscal.NotaFiscal, error) {
	return f.nota, f.erro
}

func (f *notaUseCaseFalso) Listar(_ context.Context, input portsin.ListarNotasInput) ([]*domainnotafiscal.NotaFiscal, error) {
	f.inputListar = input
	return []*domainnotafiscal.NotaFiscal{f.nota}, f.erro
}

func (f *notaUseCaseFalso) Vincular(_ context.Context, input portsin.VincularInput) (*domainnotafiscal.NotaFiscal, error) {
	f.inputVincular = input
	return f.nota, f.erro
}

func (f *notaUseCaseFalso) VincularProduto(_ context.Context, input portsin.VincularProdutoInput) (*domainnotafiscal.NotaFiscal, error) {
	f.inputProduto = input
	return f.nota, f.erro
}

func (f *notaUseCaseFalso) BaixarXML(_ context.Context, notaFiscalID, _ uuid.UUID) (*domainsolicitacao.Arquivo, io.ReadCloser, error) {
	f.notaBaixada = notaFiscalID
	if f.erro != nil {
		return nil, nil, f.erro
	}
	return f.arquivoDoXML, io.NopCloser(bytes.NewReader(f.conteudoDoXML)), nil
}

func (f *notaUseCaseFalso) ListarItensSemProduto(_ context.Context, input portsin.ListarItensSemProdutoInput) ([]*domainnotafiscal.ItemPendente, error) {
	f.inputPendentes = input
	return []*domainnotafiscal.ItemPendente{}, f.erro
}

func (f *notaUseCaseFalso) ListarEntregasSemNota(_ context.Context, input portsin.ListarEntregasSemNotaInput) ([]*domainnotafiscal.EntregaSemNota, error) {
	f.inputEntregas = input
	return []*domainnotafiscal.EntregaSemNota{}, f.erro
}

func (f *notaUseCaseFalso) SincronizarTitulos(_ context.Context, notaFiscalID, _ uuid.UUID) (*portsin.SincronizacaoTitulos, error) {
	f.sincronizada = notaFiscalID
	return f.sincronizacao, f.erro
}

var _ portsin.NotaFiscalUseCase = (*notaUseCaseFalso)(nil)

func notaDeTeste() *domainnotafiscal.NotaFiscal {
	clienteID := uuid.New()
	produtoID := uuid.New()
	id := uuid.New()

	return &domainnotafiscal.NotaFiscal{
		ID: id, Chave: domainnotafiscal.ChaveDe("25261011222333000181550010000001231123456789"), EmpresaID: uuid.New(),
		Numero: 123, Serie: 1, Natureza: domainnotafiscal.NaturezaVenda, Status: domainnotafiscal.StatusAutorizada,
		ClienteID: &clienteID, ValorTotalCentavos: 109000, Versao: 1,
		Itens: []*domainnotafiscal.Item{
			{NotaFiscalID: id, NumeroItem: 1, Codigo: "PISO-01", ProdutoID: &produtoID, QuantidadeDecimilesimos: 105000, ValorUnitario: "100", ValorCentavos: 105000},
		},
		Duplicatas: []*domainnotafiscal.Duplicata{
			{ID: uuid.New(), NotaFiscalID: id, Numero: "001", Vencimento: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), ValorCentavos: 109000},
		},
	}
}

func TestImportarRepassaArquivoEDevolveAvisos(t *testing.T) {
	usecase := &notaUseCaseFalso{nota: notaDeTeste(), avisos: []string{portsin.AvisoItensSemProduto}}
	arquivo, pedido := uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Importar, http.MethodPost, "/api/v1/notas-fiscais/importacao", nil,
		`{"arquivo_id":"`+arquivo.String()+`","pedido_id":"`+pedido.String()+`"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado 201: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.inputImportar.ArquivoID != arquivo || usecase.inputImportar.PedidoID == nil || *usecase.inputImportar.PedidoID != pedido ||
		usecase.inputImportar.EntregaID != nil {
		t.Errorf("input = %+v", usecase.inputImportar)
	}

	var envelope dto.Resposta[notafiscaldto.ImportacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	dados := envelope.Dados
	if dados.Numero != 123 || len(dados.Avisos) != 1 || len(dados.Itens) != 1 || dados.Itens[0].Quantidade != "10.5000" ||
		len(dados.Duplicatas) != 1 || dados.Natureza != "venda" {
		t.Errorf("importação devolvida = %+v", dados)
	}
}

func TestImportarRecusaCorpoInvalido(t *testing.T) {
	usecase := &notaUseCaseFalso{nota: notaDeTeste()}

	for _, corpo := range []string{`{"arquivo_id":"x"}`, `{"desconhecido":1}`, `{`} {
		resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Importar, http.MethodPost, "/api/v1/notas-fiscais/importacao", nil, corpo)
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
	if usecase.inputImportar.UsuarioID != uuid.Nil {
		t.Error("corpo inválido chegou ao usecase")
	}
}

func TestListarNotasConverteConsultas(t *testing.T) {
	usecase := &notaUseCaseFalso{nota: notaDeTeste()}
	empresa := uuid.New()

	resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Listar, http.MethodGet,
		"/api/v1/notas-fiscais?empresa="+empresa.String()+"&natureza=venda&status=autorizada&entre_empresas=false&sem_pedido=true"+
			"&emitida_de=2026-10-01&emitida_ate=2026-10-31&numero=123&chave=abc&pagina=2&tamanho=5", nil, "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	input := usecase.inputListar
	if input.Empresa == nil || *input.Empresa != empresa || input.Natureza != "venda" || input.Status != "autorizada" ||
		input.EntreEmpresas == nil || *input.EntreEmpresas || !input.SemPedido || input.Numero == nil || *input.Numero != 123 ||
		input.Chave != "abc" || input.EmitidaDe == nil || !input.EmitidaDe.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) ||
		input.EmitidaAte == nil || input.Filtro.Page != 2 || input.Filtro.Size != 5 {
		t.Errorf("input = %+v", input)
	}

	for _, consulta := range []string{"empresa=x", "entre_empresas=talvez", "sem_pedido=x", "emitida_de=01/10/2026", "numero=0", "cliente=x", "pedido=x"} {
		resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Listar, http.MethodGet, "/api/v1/notas-fiscais?"+consulta, nil, "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}

func TestVincularEProdutoDoItemRepassamVersao(t *testing.T) {
	usecase := &notaUseCaseFalso{nota: notaDeTeste()}
	handler := NewNotaFiscalHandler(usecase)
	id, pedido, produto := uuid.New(), uuid.New(), uuid.New()

	resposta := executaProtegido(t, handler.Vincular, http.MethodPut, "/api/v1/notas-fiscais/"+id.String()+"/vinculo",
		map[string]string{"id": id.String()}, `{"versao":3,"pedido_id":"`+pedido.String()+`","entrega_id":null}`)
	if resposta.Code != http.StatusOK || usecase.inputVincular.NotaFiscalID != id || usecase.inputVincular.Versao != 3 ||
		usecase.inputVincular.PedidoID == nil || usecase.inputVincular.EntregaID != nil {
		t.Errorf("vínculo = %d %+v", resposta.Code, usecase.inputVincular)
	}

	resposta = executaProtegido(t, handler.VincularProduto, http.MethodPut, "/api/v1/notas-fiscais/"+id.String()+"/itens/2/produto",
		map[string]string{"id": id.String(), "numero": "2"}, `{"versao":4,"produto_id":"`+produto.String()+`"}`)
	if resposta.Code != http.StatusOK || usecase.inputProduto.NumeroItem != 2 || usecase.inputProduto.Versao != 4 || usecase.inputProduto.ProdutoID != produto {
		t.Errorf("produto do item = %d %+v", resposta.Code, usecase.inputProduto)
	}

	resposta = executaProtegido(t, handler.VincularProduto, http.MethodPut, "/api/v1/notas-fiscais/"+id.String()+"/itens/x/produto",
		map[string]string{"id": id.String(), "numero": "x"}, `{"versao":4,"produto_id":"`+produto.String()+`"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestBaixarXMLDevolveAnexoSemInterpretacao(t *testing.T) {
	conteudo := []byte(`<nfeProc/>`)
	usecase := &notaUseCaseFalso{
		conteudoDoXML: conteudo,
		arquivoDoXML:  &domainsolicitacao.Arquivo{Nome: "nota 123.xml", ContentType: "application/xml", Tamanho: int64(len(conteudo))},
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).BaixarXML, http.MethodGet, "/api/v1/notas-fiscais/"+id.String()+"/xml",
		map[string]string{"id": id.String()}, "")

	if resposta.Code != http.StatusOK || resposta.Body.String() != string(conteudo) || usecase.notaBaixada != id {
		t.Fatalf("download = %d %q", resposta.Code, resposta.Body.String())
	}
	cabecalhos := map[string]string{
		"Content-Type":           "application/xml",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, no-store",
		"Content-Disposition":    "attachment; filename*=UTF-8''nota%20123.xml",
	}
	for nome, esperado := range cabecalhos {
		if obtido := resposta.Header().Get(nome); obtido != esperado {
			t.Errorf("%s = %q, esperado %q", nome, obtido, esperado)
		}
	}
}

func TestConsultasDePendenciaConvertemFiltros(t *testing.T) {
	usecase := &notaUseCaseFalso{}
	handler := NewNotaFiscalHandler(usecase)
	empresa, cliente := uuid.New(), uuid.New()

	resposta := executaProtegido(t, handler.ListarItensSemProduto, http.MethodGet, "/api/v1/notas-fiscais/itens-sem-produto?empresa="+empresa.String(), nil, "")
	if resposta.Code != http.StatusOK || usecase.inputPendentes.Empresa == nil || *usecase.inputPendentes.Empresa != empresa {
		t.Errorf("itens sem produto = %d %+v", resposta.Code, usecase.inputPendentes)
	}

	resposta = executaProtegido(t, handler.ListarEntregasSemNota, http.MethodGet, "/api/v1/entregas/sem-nota-fiscal?cliente="+cliente.String(), nil, "")
	if resposta.Code != http.StatusOK || usecase.inputEntregas.ClienteID == nil || *usecase.inputEntregas.ClienteID != cliente {
		t.Errorf("entregas sem nota = %d %+v", resposta.Code, usecase.inputEntregas)
	}

	resposta = executaProtegido(t, handler.ListarEntregasSemNota, http.MethodGet, "/api/v1/entregas/sem-nota-fiscal?empresa=x", nil, "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestErrosDaNotaViramStatus(t *testing.T) {
	casos := map[int]error{
		http.StatusBadRequest: domain.ErroValidacao("XML da NF-e malformado"),
		http.StatusForbidden:  domain.ErroPermissao("perfil sem acesso a esta empresa"),
		http.StatusNotFound:   domain.ErroNaoEncontrado("nota fiscal não encontrada"),
		http.StatusConflict:   domain.ErroConflito("já existe uma nota fiscal com esta chave de acesso"),
	}

	for status, erro := range casos {
		usecase := &notaUseCaseFalso{erro: erro}
		id := uuid.New()
		resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Obter, http.MethodGet, "/api/v1/notas-fiscais/"+id.String(),
			map[string]string{"id": id.String()}, "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, status)

		resposta = executaProtegido(t, NewNotaFiscalHandler(usecase).BaixarXML, http.MethodGet, "/api/v1/notas-fiscais/"+id.String()+"/xml",
			map[string]string{"id": id.String()}, "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, status)
	}
}

func TestSincronizarTitulosDevolveContagens(t *testing.T) {
	gerado := uuid.New()
	usecase := &notaUseCaseFalso{sincronizacao: &portsin.SincronizacaoTitulos{Gerados: []uuid.UUID{gerado}, Ignorados: 1}}
	id := uuid.New()

	resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).SincronizarTitulos, http.MethodPost, "/api/v1/notas-fiscais/"+id.String()+"/titulos",
		map[string]string{"id": id.String()}, "")

	if resposta.Code != http.StatusOK || usecase.sincronizada != id {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	var envelope dto.Resposta[notafiscaldto.SincronizacaoTitulosResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	dados := envelope.Dados
	if len(dados.Gerados) != 1 || dados.Gerados[0] != gerado || dados.JaExistentes == nil || dados.Cancelados == nil || dados.Ignorados != 1 {
		t.Errorf("sincronização = %+v", dados)
	}
}

func TestImportarComTitulosPendentesResponde201(t *testing.T) {
	usecase := &notaUseCaseFalso{
		nota: notaDeTeste(), avisos: []string{portsin.AvisoTitulosPendentes}, erroDosTitulos: errors.New("conexão perdida"),
	}

	resposta := executaProtegido(t, NewNotaFiscalHandler(usecase).Importar, http.MethodPost, "/api/v1/notas-fiscais/importacao", nil,
		`{"arquivo_id":"`+uuid.NewString()+`"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	var envelope dto.Resposta[notafiscaldto.ImportacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if len(envelope.Dados.Avisos) != 1 || envelope.Dados.Avisos[0] != portsin.AvisoTitulosPendentes || envelope.Dados.TitulosGerados == nil {
		t.Errorf("importação = %+v", envelope.Dados)
	}
}
