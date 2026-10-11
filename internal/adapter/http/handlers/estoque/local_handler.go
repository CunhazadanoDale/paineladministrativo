package estoque

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type LocalHandler struct {
	usecase portsin.LocalUseCase
}

func NewLocalHandler(usecase portsin.LocalUseCase) *LocalHandler {
	return &LocalHandler{usecase: usecase}
}

func (h *LocalHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.CriarLocalRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	local, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	h.escreverLocal(w, http.StatusCreated, local, err)
}

func (h *LocalHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativos")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarLocaisInput{
		UsuarioID: usuario.ID,
		Empresa:   empresa,
		Ativo:     ativo,
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.LocalResponse]{
		Dados:   estoquedto.NovoLocalResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *LocalHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	local, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	h.escreverLocal(w, http.StatusOK, local, err)
}

func (h *LocalHandler) Renomear(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.RenomearLocalRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	local, err := h.usecase.Renomear(r.Context(), requisicao.ParaInput(usuario.ID, id))
	h.escreverLocal(w, http.StatusOK, local, err)
}

func (h *LocalHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.alterarSituacao(w, r, h.usecase.Ativar)
}

func (h *LocalHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.alterarSituacao(w, r, h.usecase.Desativar)
}

func (h *LocalHandler) Remover(w http.ResponseWriter, r *http.Request) {
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

func (h *LocalHandler) ListarSaldos(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.ListarSaldos(r.Context(), portsin.ListarSaldosDoLocalInput{
		UsuarioID: usuario.ID,
		LocalID:   id,
		ComSaldo:  resposta.ConsultaBooleana(r, "com_saldo"),
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.SaldoLocalResponse]{
		Dados:   estoquedto.NovoSaldoLocalResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *LocalHandler) alterarSituacao(
	w http.ResponseWriter,
	r *http.Request,
	alterar func(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	local, err := alterar(r.Context(), id, usuario.ID)
	h.escreverLocal(w, http.StatusOK, local, err)
}

func (h *LocalHandler) escreverLocal(w http.ResponseWriter, status int, local *domainestoque.Local, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[estoquedto.LocalResponse]{
		Dados: estoquedto.NovoLocalResponse(local),
	})
}
