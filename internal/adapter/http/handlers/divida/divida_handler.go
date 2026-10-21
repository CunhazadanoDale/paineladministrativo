package divida

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	dividadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/divida"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
)

type DividaHandler struct {
	usecase portsin.DividaUseCase
}

func NewDividaHandler(usecase portsin.DividaUseCase) *DividaHandler {
	return &DividaHandler{usecase: usecase}
}

func (h *DividaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao dividadto.CriarDividaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	input, err := requisicao.ParaInput(usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Criar(r.Context(), input)
	escreverDetalhe(w, http.StatusCreated, visao, err)
}

func (h *DividaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarDividasInput{
		UsuarioID:  usuario.ID,
		CredorTipo: resposta.ConsultaTexto(r, "credor_tipo"),
		Status:     resposta.ConsultaTexto(r, "status"),
		Filtro:     resposta.ConsultaPaginacao(r),
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

	dividas, err := h.usecase.Listar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[dividadto.DividaResponse]{
		Dados:   dividadto.NovasDividaResponses(dividas),
		Pagina:  input.Filtro.Page,
		Tamanho: input.Filtro.Size,
	})
}

func (h *DividaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	visao, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *DividaHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao dividadto.AlterarDividaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Alterar(r.Context(), portsin.AlterarDividaInput{
		UsuarioID: usuario.ID, DividaID: id, Versao: requisicao.Versao, Dados: dados,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *DividaHandler) AlterarParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}
	parcelaID, ok := resposta.ParametroUUID(w, r, "parcela_id")
	if !ok {
		return
	}

	var requisicao dividadto.AlterarParcelaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	vencimento, err := dividadto.ParaData(requisicao.Vencimento, "vencimento da parcela")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.AlterarParcela(r.Context(), portsin.AlterarParcelaInput{
		UsuarioID: usuario.ID, DividaID: id, ParcelaID: parcelaID, Versao: requisicao.Versao,
		Vencimento: vencimento, JurosCentavos: requisicao.JurosCentavos,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *DividaHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao dividadto.CancelarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.Cancelar(r.Context(), portsin.CancelarDividaInput{
		UsuarioID: usuario.ID, DividaID: id, Versao: requisicao.Versao, Motivo: requisicao.Motivo,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *DividaHandler) Renegociar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao dividadto.RenegociarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	nova, err := requisicao.Nova.ParaNova()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Renegociar(r.Context(), portsin.RenegociarDividaInput{
		UsuarioID: usuario.ID, DividaID: id, Versao: requisicao.Versao, Motivo: requisicao.Motivo, Nova: nova,
	})
	escreverDetalhe(w, http.StatusCreated, visao, err)
}

func (h *DividaHandler) ReabrirParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}
	parcelaID, ok := resposta.ParametroUUID(w, r, "parcela_id")
	if !ok {
		return
	}

	var requisicao dividadto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.ReabrirParcela(r.Context(), portsin.ReabrirParcelaInput{
		UsuarioID: usuario.ID, DividaID: id, ParcelaID: parcelaID, Versao: requisicao.Versao,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *DividaHandler) ListarHistorico(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	historico, err := h.usecase.ListarHistorico(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]dividadto.HistoricoResponse]{Dados: dividadto.NovoHistoricoResponses(historico)})
}

func (h *DividaHandler) GerarContas(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resultado, err := h.usecase.GerarContas(r.Context(), portsin.GerarContasInput{
		UsuarioID:   usuario.ID,
		Competencia: resposta.ConsultaTexto(r, "competencia"),
		Empresa:     empresa,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[dividadto.ResultadoGeracaoResponse]{
		Dados: dividadto.NovoResultadoGeracaoResponse(resultado),
	})
}

func escreverDetalhe(w http.ResponseWriter, status int, visao *portsin.Visao, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[dividadto.DividaDetalheResponse]{Dados: dividadto.NovaDividaDetalheResponse(visao)})
}
