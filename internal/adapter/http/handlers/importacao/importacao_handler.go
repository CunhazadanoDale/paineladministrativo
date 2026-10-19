package importacao

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
)

type ImportacaoHandler struct {
	usecase portsin.ImportacaoUseCase
}

func NewImportacaoHandler(usecase portsin.ImportacaoUseCase) *ImportacaoHandler {
	return &ImportacaoHandler{usecase: usecase}
}

func (h *ImportacaoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.CriarImportacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Criar(r.Context(), portsin.CriarImportacaoInput{UsuarioID: usuario.ID, EmpresaID: requisicao.EmpresaID, Dados: dados})
	escreverDetalhe(w, http.StatusCreated, visao, err)
}

func (h *ImportacaoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarImportacoesInput{
		UsuarioID: usuario.ID,
		Status:    resposta.ConsultaTexto(r, "status"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.FornecedorID, err = resposta.ConsultaUUID(r, "fornecedor"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Numero, err = consultaInteiro(r, "numero"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ChegadaAte, err = consultaData(r, "chegada_ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	lista, err := h.usecase.Listar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[importacaodto.ImportacaoResponse]{
		Dados:   importacaodto.NovaImportacaoResponses(lista),
		Pagina:  input.Filtro.Page,
		Tamanho: input.Filtro.Size,
	})
}

func (h *ImportacaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	visao, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *ImportacaoHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.AlterarImportacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Alterar(r.Context(), portsin.AlterarImportacaoInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, Dados: dados,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *ImportacaoHandler) Embarcar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.EmbarcarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	embarcadaEm, err := importacaodto.ParaData(requisicao.EmbarcadaEm, "data do embarque")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	chegadaPrevista, err := importacaodto.ParaData(requisicao.ChegadaPrevista, "chegada prevista")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Embarcar(r.Context(), portsin.EmbarcarInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao,
		EmbarcadaEm: embarcadaEm, ChegadaPrevista: chegadaPrevista, DocumentoEmbarque: requisicao.DocumentoEmbarque,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *ImportacaoHandler) Desembaracar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.DesembaracarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	chegadaEm, err := importacaodto.ParaData(requisicao.ChegadaEm, "data da chegada")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	desembaracadaEm, err := importacaodto.ParaData(requisicao.DesembaracadaEm, "data do desembaraço")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Desembaracar(r.Context(), portsin.DesembaracarInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao,
		ChegadaEm: chegadaEm, DesembaracadaEm: desembaracadaEm, Declaracao: requisicao.Declaracao,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *ImportacaoHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.CancelarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.Cancelar(r.Context(), portsin.CancelarImportacaoInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, Motivo: requisicao.Motivo,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *ImportacaoHandler) ListarHistorico(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	historico, err := h.usecase.ListarHistorico(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]importacaodto.HistoricoResponse]{Dados: importacaodto.NovoHistoricoResponses(historico)})
}

func escreverDetalhe(w http.ResponseWriter, status int, visao *portsin.Visao, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[importacaodto.ImportacaoDetalheResponse]{Dados: importacaodto.NovaImportacaoDetalheResponse(visao)})
}

func consultaInteiro(r *http.Request, nome string) (*int, error) {
	valor := resposta.ConsultaTexto(r, nome)
	if valor == "" {
		return nil, nil
	}

	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 1 {
		return nil, domain.ErroValidacao("parâmetro " + nome + " deve ser um número inteiro positivo")
	}

	return &numero, nil
}

func consultaData(r *http.Request, nome string) (*time.Time, error) {
	data, err := importacaodto.ParaData(resposta.ConsultaTexto(r, nome), "parâmetro "+nome)
	if err != nil || data.IsZero() {
		return nil, err
	}

	return &data, nil
}
