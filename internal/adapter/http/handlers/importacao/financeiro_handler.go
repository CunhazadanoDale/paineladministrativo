package importacao

import (
	"net/http"
	"strconv"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type FinanceiroHandler struct {
	usecase portsin.FinanceiroUseCase
}

func NewFinanceiroHandler(usecase portsin.FinanceiroUseCase) *FinanceiroHandler {
	return &FinanceiroHandler{usecase: usecase}
}

func (h *FinanceiroHandler) IncluirParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.ParcelaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.IncluirParcela(r.Context(), portsin.IncluirParcelaInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, Dados: dados,
	})
	escreverDetalhe(w, http.StatusCreated, visao, err)
}

func (h *FinanceiroHandler) AlterarParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, parcelaID, ok := usuarioImportacaoEFilho(w, r, "parcela_id")
	if !ok {
		return
	}

	var requisicao importacaodto.ParcelaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.AlterarParcela(r.Context(), portsin.AlterarParcelaInput{
		UsuarioID: usuario.ID, ImportacaoID: id, ParcelaID: parcelaID, Versao: requisicao.Versao, Dados: dados,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *FinanceiroHandler) RemoverParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, parcelaID, ok := usuarioImportacaoEFilho(w, r, "parcela_id")
	if !ok {
		return
	}

	versao, err := consultaVersao(r)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.RemoverParcela(r.Context(), portsin.ParcelaInput{
		UsuarioID: usuario.ID, ImportacaoID: id, ParcelaID: parcelaID, Versao: versao,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *FinanceiroHandler) FecharCambio(w http.ResponseWriter, r *http.Request) {
	usuario, id, parcelaID, ok := usuarioImportacaoEFilho(w, r, "parcela_id")
	if !ok {
		return
	}

	var requisicao importacaodto.CambioRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	vencimento, err := importacaodto.ParaData(requisicao.Vencimento, "vencimento da conta a pagar")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.FecharCambio(r.Context(), portsin.FecharCambioInput{
		UsuarioID: usuario.ID, ImportacaoID: id, ParcelaID: parcelaID, Versao: requisicao.Versao,
		CambioMicro: requisicao.CambioMicro, Vencimento: vencimento, CategoriaFinanceiraID: requisicao.CategoriaFinanceiraID,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *FinanceiroHandler) ReabrirParcela(w http.ResponseWriter, r *http.Request) {
	usuario, id, parcelaID, ok := usuarioImportacaoEFilho(w, r, "parcela_id")
	if !ok {
		return
	}

	var requisicao importacaodto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.ReabrirParcela(r.Context(), portsin.ParcelaInput{
		UsuarioID: usuario.ID, ImportacaoID: id, ParcelaID: parcelaID, Versao: requisicao.Versao,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *FinanceiroHandler) LancarCusto(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.CustoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	vencimento, err := importacaodto.ParaData(requisicao.Vencimento, "vencimento do custo")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	compoeCusto := true
	if requisicao.CompoeCusto != nil {
		compoeCusto = *requisicao.CompoeCusto
	}

	visao, err := h.usecase.LancarCusto(r.Context(), portsin.LancarCustoInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, Tipo: requisicao.Tipo,
		Descricao: requisicao.Descricao, FornecedorID: requisicao.FornecedorID, ValorCentavos: requisicao.ValorCentavos,
		Vencimento: vencimento, CompoeCusto: compoeCusto, CategoriaFinanceiraID: requisicao.CategoriaFinanceiraID,
	})
	escreverDetalhe(w, http.StatusCreated, visao, err)
}

func (h *FinanceiroHandler) CancelarCusto(w http.ResponseWriter, r *http.Request) {
	usuario, id, custoID, ok := usuarioImportacaoEFilho(w, r, "custo_id")
	if !ok {
		return
	}

	var requisicao importacaodto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.CancelarCusto(r.Context(), portsin.CustoInput{
		UsuarioID: usuario.ID, ImportacaoID: id, CustoID: custoID, Versao: requisicao.Versao,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *FinanceiroHandler) CustoPrevisto(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	cambio, err := consultaInteiroLongo(r, "cambio_micro")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	calculo, err := h.usecase.CustoPrevisto(r.Context(), id, usuario.ID, cambio)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[importacaodto.CustoPrevistoResponse]{Dados: importacaodto.NovoCustoPrevistoResponse(calculo)})
}

func usuarioImportacaoEFilho(w http.ResponseWriter, r *http.Request, nome string) (*domainusuarios.Usuario, uuid.UUID, uuid.UUID, bool) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return nil, uuid.Nil, uuid.Nil, false
	}

	filho, ok := resposta.ParametroUUID(w, r, nome)
	if !ok {
		return nil, uuid.Nil, uuid.Nil, false
	}

	return usuario, id, filho, true
}

func consultaVersao(r *http.Request) (int, error) {
	versao, err := consultaInteiro(r, "versao")
	if err != nil {
		return 0, err
	}
	if versao == nil {
		return 0, nil
	}

	return *versao, nil
}

func consultaInteiroLongo(r *http.Request, nome string) (*int64, error) {
	valor := resposta.ConsultaTexto(r, nome)
	if valor == "" {
		return nil, nil
	}

	numero, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || numero < 1 {
		return nil, domain.ErroValidacao("parâmetro " + nome + " deve ser um número inteiro positivo")
	}

	return &numero, nil
}
