package fornecedor

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	fornecedordto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/fornecedor"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
)

type FornecedorHandler struct {
	usecase portsin.FornecedorUseCase
}

func NewFornecedorHandler(usecase portsin.FornecedorUseCase) *FornecedorHandler {
	return &FornecedorHandler{usecase: usecase}
}

func (h *FornecedorHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao fornecedordto.CriarFornecedorRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	fornecedor, err := h.usecase.Criar(r.Context(), portsin.CriarFornecedorInput{
		UsuarioID: usuario.ID,
		Pais:      requisicao.Pais,
		Documento: requisicao.Documento,
		Dados:     requisicao.ParaDados(),
	})
	escreverItem(w, http.StatusCreated, fornecedor, err)
}

func (h *FornecedorHandler) Listar(w http.ResponseWriter, r *http.Request) {
	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativo")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	itens, err := h.usecase.Listar(r.Context(), portsin.ListarFornecedoresInput{
		Busca:  resposta.ConsultaTexto(r, "busca"),
		Pais:   resposta.ConsultaTexto(r, "pais"),
		Moeda:  resposta.ConsultaTexto(r, "moeda"),
		Ativo:  ativo,
		Filtro: paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[fornecedordto.FornecedorResponse]{
		Dados:   fornecedordto.NovoFornecedorResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *FornecedorHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	fornecedor, err := h.usecase.Obter(r.Context(), id)
	escreverItem(w, http.StatusOK, fornecedor, err)
}

func (h *FornecedorHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao fornecedordto.AlterarFornecedorRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	fornecedor, err := h.usecase.Alterar(r.Context(), portsin.AlterarFornecedorInput{
		UsuarioID:    usuario.ID,
		FornecedorID: id,
		Versao:       requisicao.Versao,
		Dados:        requisicao.ParaDados(),
	})
	escreverItem(w, http.StatusOK, fornecedor, err)
}

func (h *FornecedorHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Ativar)
}

func (h *FornecedorHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Desativar)
}

func (h *FornecedorHandler) Remover(w http.ResponseWriter, r *http.Request) {
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

func (h *FornecedorHandler) mudarSituacao(
	w http.ResponseWriter,
	r *http.Request,
	mudar func(context.Context, portsin.SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao fornecedordto.SituacaoFornecedorRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	fornecedor, err := mudar(r.Context(), portsin.SituacaoFornecedorInput{
		UsuarioID:    usuario.ID,
		FornecedorID: id,
		Versao:       requisicao.Versao,
	})
	escreverItem(w, http.StatusOK, fornecedor, err)
}

func escreverItem(w http.ResponseWriter, status int, fornecedor *domainfornecedor.Fornecedor, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[fornecedordto.FornecedorResponse]{Dados: fornecedordto.NovoFornecedorResponse(fornecedor)})
}
