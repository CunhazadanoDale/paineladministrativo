package solicitacao

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type SolicitacaoHandler struct {
	usecase portsin.SolicitacaoUseCase
}

func NewSolicitacaoHandler(usecase portsin.SolicitacaoUseCase) *SolicitacaoHandler {
	return &SolicitacaoHandler{usecase: usecase}
}

func (h *SolicitacaoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := usuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.CriarSolicitacaoRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	input, err := requisicao.ParaInput(usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	id, err := h.usecase.Criar(r.Context(), input)
	if err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	escreverSolicitacao(w, http.StatusCreated, item)
}

func (h *SolicitacaoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := usuarioDoContexto(w, r)
	if !ok {
		return
	}

	paginacao := consultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarSolicitacoesInput{
		UsuarioID: usuario.ID,
		Escopo:    consultaTexto(r, "escopo"),
		Status:    consultaTexto(r, "status"),
		Filtro:    paginacao,
	})
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[solicitacaodto.SolicitacaoResponse]{
		Dados:   solicitacaodto.NovaSolicitacaoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *SolicitacaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	escreverSolicitacao(w, http.StatusOK, item)
}

func (h *SolicitacaoHandler) Aprovar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.Aprovar(r.Context(), id, usuario.ID); err != nil {
		responderErro(w, err)
		return
	}

	h.responderAtualizada(w, r, id, usuario.ID)
}

func (h *SolicitacaoHandler) Rejeitar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.RejeitarRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Rejeitar(r.Context(), id, usuario.ID, requisicao.Motivo); err != nil {
		responderErro(w, err)
		return
	}

	h.responderAtualizada(w, r, id, usuario.ID)
}

func (h *SolicitacaoHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.Cancelar(r.Context(), id, usuario.ID); err != nil {
		responderErro(w, err)
		return
	}

	h.responderAtualizada(w, r, id, usuario.ID)
}

func (h *SolicitacaoHandler) RegistrarPagamento(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao solicitacaodto.RegistrarPagamentoRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	input, err := requisicao.ParaInput(id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	if err := h.usecase.RegistrarPagamento(r.Context(), input); err != nil {
		responderErro(w, err)
		return
	}

	pagamento, err := h.usecase.ObterPagamento(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[solicitacaodto.PagamentoResponse]{
		Dados: solicitacaodto.NovoPagamentoResponse(pagamento),
	})
}

func (h *SolicitacaoHandler) ObterPagamento(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	pagamento, err := h.usecase.ObterPagamento(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[solicitacaodto.PagamentoResponse]{
		Dados: solicitacaodto.NovoPagamentoResponse(pagamento),
	})
}

func (h *SolicitacaoHandler) ListarHistorico(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	itens, err := h.usecase.ListarHistorico(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]solicitacaodto.HistoricoResponse]{
		Dados: solicitacaodto.NovoHistoricoResponses(itens),
	})
}

func (h *SolicitacaoHandler) ListarArquivos(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := usuarioEId(w, r)
	if !ok {
		return
	}

	itens, err := h.usecase.ListarArquivos(r.Context(), id, usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]solicitacaodto.ArquivoResponse]{
		Dados: solicitacaodto.NovoArquivoResponses(itens),
	})
}

func (h *SolicitacaoHandler) responderAtualizada(w http.ResponseWriter, r *http.Request, solicitacaoID, usuarioID uuid.UUID) {
	item, err := h.usecase.Obter(r.Context(), solicitacaoID, usuarioID)
	if err != nil {
		responderErro(w, err)
		return
	}

	escreverSolicitacao(w, http.StatusOK, item)
}

func escreverSolicitacao(w http.ResponseWriter, status int, item *domainsolicitacao.Solicitacao) {
	dto.EscreverJSON(w, status, dto.Resposta[solicitacaodto.SolicitacaoResponse]{
		Dados: solicitacaodto.NovaSolicitacaoResponse(item),
	})
}
