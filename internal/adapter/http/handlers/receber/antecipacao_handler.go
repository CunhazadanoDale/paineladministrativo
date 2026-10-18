package receber

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
)

type AntecipacaoHandler struct {
	usecase portsin.AntecipacaoUseCase
}

func NewAntecipacaoHandler(usecase portsin.AntecipacaoUseCase) *AntecipacaoHandler {
	return &AntecipacaoHandler{usecase: usecase}
}

func (h *AntecipacaoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.CriarAntecipacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	operadoEm, err := receberdto.ParaData(requisicao.OperadoEm, "data da operação")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	antecipacao, err := h.usecase.Criar(r.Context(), portsin.CriarAntecipacaoInput{
		UsuarioID:            usuario.ID,
		EmpresaID:            requisicao.EmpresaID,
		FornecedorID:         requisicao.FornecedorID,
		Numero:               requisicao.Numero,
		OperadoEm:            operadoEm,
		TaxaBP:               requisicao.TaxaBP,
		ValorLiquidoCentavos: requisicao.ValorLiquidoCentavos,
		ContaBancariaID:      requisicao.ContaBancariaID,
		TituloIDs:            requisicao.TituloIDs,
	})
	escreverItem(w, http.StatusCreated, antecipacao, err, receberdto.NovaAntecipacaoResponse)
}

func (h *AntecipacaoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarAntecipacoesInput{
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
	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	antecipacoes, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, antecipacoes, err, receberdto.NovaAntecipacaoResponses)
}

func (h *AntecipacaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	antecipacao, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, antecipacao, err, receberdto.NovaAntecipacaoResponse)
}

func (h *AntecipacaoHandler) Estornar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.EstornarAntecipacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	antecipacao, err := h.usecase.Estornar(r.Context(), portsin.EstornarAntecipacaoInput{
		UsuarioID: usuario.ID, AntecipacaoID: id, Versao: requisicao.Versao, Motivo: requisicao.Motivo,
	})
	escreverItem(w, http.StatusOK, antecipacao, err, receberdto.NovaAntecipacaoResponse)
}

func (h *AntecipacaoHandler) Recomprar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.RecompraRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	recompradoEm, err := receberdto.ParaData(requisicao.RecompradoEm, "data da recompra")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	vencimento, err := receberdto.ParaData(requisicao.Vencimento, "vencimento da recompra")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, recompra, err := h.usecase.Recomprar(r.Context(), portsin.RecomprarInput{
		UsuarioID:             usuario.ID,
		TituloID:              id,
		Versao:                requisicao.Versao,
		RecompradoEm:          recompradoEm,
		EncargosCentavos:      requisicao.EncargosCentavos,
		Vencimento:            vencimento,
		CategoriaFinanceiraID: requisicao.CategoriaFinanceiraID,
		Motivo:                requisicao.Motivo,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[receberdto.RecompraRegistradaResponse]{
		Dados: receberdto.NovaRecompraRegistradaResponse(titulo, recompra),
	})
}
