package financeiro

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	financeirodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
)

type ContaBancariaHandler struct {
	usecase portsin.ContaBancariaUseCase
}

func NewContaBancariaHandler(usecase portsin.ContaBancariaUseCase) *ContaBancariaHandler {
	return &ContaBancariaHandler{usecase: usecase}
}

func (h *ContaBancariaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.CriarContaBancariaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	conta, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	escreverConta(w, http.StatusCreated, conta, err)
}

func (h *ContaBancariaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
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
	contas, err := h.usecase.Listar(r.Context(), portsin.ListarContasBancariasInput{
		UsuarioID: usuario.ID,
		Empresa:   empresa,
		Banco:     resposta.ConsultaTexto(r, "banco"),
		Tipo:      resposta.ConsultaTexto(r, "tipo"),
		Ativa:     ativa,
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[financeirodto.ContaBancariaResponse]{
		Dados:   financeirodto.NovasContaBancariaResponses(contas),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *ContaBancariaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	conta, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverConta(w, http.StatusOK, conta, err)
}

func (h *ContaBancariaHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.AlterarContaBancariaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	conta, err := h.usecase.Alterar(r.Context(), portsin.AlterarContaBancariaInput{
		UsuarioID: usuario.ID,
		ContaID:   id,
		Apelido:   requisicao.Apelido,
		Versao:    requisicao.Versao,
	})
	escreverConta(w, http.StatusOK, conta, err)
}

func (h *ContaBancariaHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Ativar)
}

func (h *ContaBancariaHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Desativar)
}

func (h *ContaBancariaHandler) Remover(w http.ResponseWriter, r *http.Request) {
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

func (h *ContaBancariaHandler) Bancos(w http.ResponseWriter, _ *http.Request) {
	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]financeirodto.BancoResponse]{
		Dados: financeirodto.NovosBancoResponses(h.usecase.Bancos()),
	})
}

func (h *ContaBancariaHandler) mudarSituacao(
	w http.ResponseWriter,
	r *http.Request,
	mudar func(context.Context, portsin.MudarSituacaoContaBancariaInput) (*domainfinanceiro.ContaBancaria, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.SituacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	conta, err := mudar(r.Context(), portsin.MudarSituacaoContaBancariaInput{
		UsuarioID: usuario.ID,
		ContaID:   id,
		Versao:    requisicao.Versao,
	})
	escreverConta(w, http.StatusOK, conta, err)
}

func escreverConta(w http.ResponseWriter, status int, conta *domainfinanceiro.ContaBancaria, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[financeirodto.ContaBancariaResponse]{Dados: financeirodto.NovaContaBancariaResponse(conta)})
}
