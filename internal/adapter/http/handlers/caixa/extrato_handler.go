package caixa

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type ExtratoHandler struct {
	usecase portsin.ExtratoUseCase
}

func NewExtratoHandler(usecase portsin.ExtratoUseCase) *ExtratoHandler {
	return &ExtratoHandler{usecase: usecase}
}

func (h *ExtratoHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.ImportarExtratoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	relatorio, err := h.usecase.Importar(r.Context(), requisicao.ParaInput(usuario.ID))
	status := http.StatusCreated
	if requisicao.Simular {
		status = http.StatusOK
	}
	escreverItem(w, status, relatorio, err, caixadto.NovoRelatorioImportacaoResponse)
}

func (h *ExtratoHandler) ListarImportacoes(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	conta, err := resposta.ConsultaUUID(r, "conta")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	importacoes, err := h.usecase.ListarImportacoes(r.Context(), portsin.ListarImportacoesInput{
		UsuarioID:       usuario.ID,
		Empresa:         empresa,
		ContaBancariaID: conta,
		Filtro:          paginacao,
	})
	escreverPagina(w, paginacao, importacoes, err, caixadto.NovasImportacaoResponses)
}

func (h *ExtratoHandler) DesfazerImportacao(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.DesfazerImportacao(r.Context(), id, usuario.ID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *ExtratoHandler) ListarLancamentos(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarLancamentosInput{
		UsuarioID: usuario.ID,
		Situacao:  resposta.ConsultaTexto(r, "situacao"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}

	conta, err := resposta.ConsultaUUID(r, "conta")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if conta == nil {
		resposta.ResponderErro(w, domain.ErroValidacao("parâmetro conta é obrigatório"))
		return
	}
	input.ContaBancariaID = *conta

	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	lancamentos, err := h.usecase.ListarLancamentos(r.Context(), input)
	escreverPagina(w, input.Filtro, lancamentos, err, caixadto.NovasLancamentoResponses)
}
