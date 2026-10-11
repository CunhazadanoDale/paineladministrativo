package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type MovimentoHandler struct {
	usecase portsin.MovimentoUseCase
}

func NewMovimentoHandler(usecase portsin.MovimentoUseCase) *MovimentoHandler {
	return &MovimentoHandler{usecase: usecase}
}

func (h *MovimentoHandler) Movimentar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.MovimentarEstoqueRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	movimento, err := h.usecase.Movimentar(r.Context(), requisicao.ParaInput(usuario.ID, id))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.MovimentoResponse]{
		Dados: estoquedto.NovoMovimentoResponse(movimento),
	})
}

func (h *MovimentoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	local, err := resposta.ConsultaUUID(r, "local")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarMovimentosInput{
		UsuarioID: usuario.ID,
		ProdutoID: &id,
		LocalID:   local,
		Tipo:      resposta.ConsultaTexto(r, "tipo"),
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.MovimentoResponse]{
		Dados:   estoquedto.NovoMovimentoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *MovimentoHandler) Transferir(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.TransferirEstoqueRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	movimento, err := h.usecase.Transferir(r.Context(), requisicao.ParaInput(usuario.ID, id))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.MovimentoResponse]{
		Dados: estoquedto.NovoMovimentoResponse(movimento),
	})
}

func (h *MovimentoHandler) Saldo(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	saldo, err := h.usecase.Saldo(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.SaldoResponse]{
		Dados: estoquedto.NovoSaldoResponse(saldo),
	})
}

func (h *MovimentoHandler) ListarLotes(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	local, err := resposta.ConsultaUUID(r, "local")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.ListarLotes(r.Context(), portsin.ListarLotesInput{
		UsuarioID: usuario.ID,
		ProdutoID: id,
		LocalID:   local,
		ComSaldo:  resposta.ConsultaBooleana(r, "com_saldo"),
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.LoteResponse]{
		Dados:   estoquedto.NovoLoteResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}
