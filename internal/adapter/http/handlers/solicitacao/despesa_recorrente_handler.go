package solicitacao

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

type DespesaRecorrenteHandler struct {
	usecase portsin.DespesaRecorrenteUseCase
}

func NewDespesaRecorrenteHandler(usecase portsin.DespesaRecorrenteUseCase) *DespesaRecorrenteHandler {
	return &DespesaRecorrenteHandler{usecase: usecase}
}

func (h *DespesaRecorrenteHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.CriarDespesaRecorrenteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	despesa, err := h.usecase.Criar(r.Context(), portsin.CriarDespesaRecorrenteInput{
		UsuarioID: usuario.ID,
		EmpresaID: requisicao.EmpresaID,
		Dados:     requisicao.ParaDados(),
	})
	escreverDespesaRecorrente(w, http.StatusCreated, despesa, err)
}

func (h *DespesaRecorrenteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	fornecedor, err := resposta.ConsultaUUID(r, "fornecedor")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	ativa, err := resposta.ConsultaBooleanaOpcional(r, "ativa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	despesas, err := h.usecase.Listar(r.Context(), portsin.ListarDespesasRecorrentesInput{
		UsuarioID:  usuario.ID,
		Empresa:    empresa,
		Ativa:      ativa,
		Fornecedor: fornecedor,
		Filtro:     paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[solicitacaodto.DespesaRecorrenteResponse]{
		Dados:   solicitacaodto.NovasDespesaRecorrenteResponses(despesas),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *DespesaRecorrenteHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	despesa, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverDespesaRecorrente(w, http.StatusOK, despesa, err)
}

func (h *DespesaRecorrenteHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.AlterarDespesaRecorrenteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	despesa, err := h.usecase.Alterar(r.Context(), portsin.AlterarDespesaRecorrenteInput{
		UsuarioID: usuario.ID,
		DespesaID: id,
		Dados:     requisicao.ParaDados(),
		Versao:    requisicao.Versao,
	})
	escreverDespesaRecorrente(w, http.StatusOK, despesa, err)
}

func (h *DespesaRecorrenteHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Ativar)
}

func (h *DespesaRecorrenteHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Desativar)
}

func (h *DespesaRecorrenteHandler) Remover(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id, usuario.ID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *DespesaRecorrenteHandler) Gerar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resultado, err := h.usecase.Gerar(r.Context(), portsin.GerarDespesasRecorrentesInput{
		UsuarioID:   usuario.ID,
		Competencia: resposta.ConsultaTexto(r, "competencia"),
		Empresa:     empresa,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[solicitacaodto.ResultadoGeracaoResponse]{
		Dados: solicitacaodto.NovoResultadoGeracaoResponse(resultado),
	})
}

func (h *DespesaRecorrenteHandler) mudarSituacao(
	w http.ResponseWriter,
	r *http.Request,
	mudar func(context.Context, portsin.MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.SituacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	despesa, err := mudar(r.Context(), portsin.MudarSituacaoDespesaRecorrenteInput{
		UsuarioID: usuario.ID,
		DespesaID: id,
		Versao:    requisicao.Versao,
	})
	escreverDespesaRecorrente(w, http.StatusOK, despesa, err)
}

func escreverDespesaRecorrente(w http.ResponseWriter, status int, despesa *domainsolicitacao.DespesaRecorrente, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[solicitacaodto.DespesaRecorrenteResponse]{
		Dados: solicitacaodto.NovaDespesaRecorrenteResponse(despesa),
	})
}
