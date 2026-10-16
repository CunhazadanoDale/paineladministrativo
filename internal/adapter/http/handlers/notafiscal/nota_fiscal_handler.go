package notafiscal

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	notafiscaldto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
)

type NotaFiscalHandler struct {
	usecase portsin.NotaFiscalUseCase
}

func NewNotaFiscalHandler(usecase portsin.NotaFiscalUseCase) *NotaFiscalHandler {
	return &NotaFiscalHandler{usecase: usecase}
}

func (h *NotaFiscalHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao notafiscaldto.ImportarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	importacao, err := h.usecase.Importar(r.Context(), requisicao.ParaInput(usuario.ID))
	escreverItem(w, http.StatusCreated, importacao, err, notafiscaldto.NovaImportacaoResponse)
}

func (h *NotaFiscalHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarNotasInput{
		UsuarioID: usuario.ID,
		Natureza:  resposta.ConsultaTexto(r, "natureza"),
		Status:    resposta.ConsultaTexto(r, "status"),
		Chave:     resposta.ConsultaTexto(r, "chave"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ClienteID, err = resposta.ConsultaUUID(r, "cliente"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.PedidoID, err = resposta.ConsultaUUID(r, "pedido"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.EntreEmpresas, err = resposta.ConsultaBooleanaOpcional(r, "entre_empresas"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	semPedido, err := resposta.ConsultaBooleanaOpcional(r, "sem_pedido")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	input.SemPedido = semPedido != nil && *semPedido
	if input.EmitidaDe, err = consultaData(r, "emitida_de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.EmitidaAte, err = consultaData(r, "emitida_ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Numero, err = consultaInteiro(r, "numero"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	notas, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, notas, err, notafiscaldto.NovaNotaFiscalResponses)
}

func (h *NotaFiscalHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	nota, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, nota, err, notafiscaldto.NovaNotaFiscalDetalheResponse)
}

func (h *NotaFiscalHandler) Vincular(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao notafiscaldto.VinculoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	nota, err := h.usecase.Vincular(r.Context(), requisicao.ParaInput(usuario.ID, id))
	escreverItem(w, http.StatusOK, nota, err, notafiscaldto.NovaNotaFiscalDetalheResponse)
}

func (h *NotaFiscalHandler) VincularProduto(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	numeroItem, err := parametroInteiro(r, "numero")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	var requisicao notafiscaldto.ProdutoDoItemRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	nota, err := h.usecase.VincularProduto(r.Context(), requisicao.ParaInput(usuario.ID, id, numeroItem))
	escreverItem(w, http.StatusOK, nota, err, notafiscaldto.NovaNotaFiscalDetalheResponse)
}

func (h *NotaFiscalHandler) BaixarXML(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	arquivo, conteudo, err := h.usecase.BaixarXML(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	defer func() { _ = conteudo.Close() }()

	w.Header().Set("Content-Type", arquivo.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(arquivo.Nome)))
	w.Header().Set("Content-Length", strconv.FormatInt(arquivo.Tamanho, 10))

	if _, err := io.Copy(w, conteudo); err != nil {
		middleware.AnotarErro(w, fmt.Errorf("envio do XML da nota fiscal interrompido: %w", err))
	}
}

func (h *NotaFiscalHandler) ListarItensSemProduto(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarItensSemProdutoInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pendentes, err := h.usecase.ListarItensSemProduto(r.Context(), input)
	escreverPagina(w, input.Filtro, pendentes, err, notafiscaldto.NovosItensPendentesResponse)
}

func (h *NotaFiscalHandler) ListarEntregasSemNota(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarEntregasSemNotaInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ClienteID, err = resposta.ConsultaUUID(r, "cliente"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	entregas, err := h.usecase.ListarEntregasSemNota(r.Context(), input)
	escreverPagina(w, input.Filtro, entregas, err, notafiscaldto.NovasEntregasSemNotaResponse)
}
