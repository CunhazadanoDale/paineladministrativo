package receber

import (
	"net/http"

	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
)

type TituloHandler struct {
	usecase portsin.TituloUseCase
}

func NewTituloHandler(usecase portsin.TituloUseCase) *TituloHandler {
	return &TituloHandler{usecase: usecase}
}

func (h *TituloHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.CriarTituloRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, err := h.usecase.Criar(r.Context(), portsin.CriarTituloInput{
		UsuarioID:       usuario.ID,
		EmpresaID:       requisicao.EmpresaID,
		ClienteID:       requisicao.ClienteID,
		ContaBancariaID: requisicao.ContaBancariaID,
		Dados:           dados,
	})
	escreverItem(w, http.StatusCreated, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input, err := listarInput(r)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	input.UsuarioID = usuario.ID

	titulos, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, titulos, err, receberdto.NovoTituloResponses)
}

func listarInput(r *http.Request) (portsin.ListarTitulosInput, error) {
	input := portsin.ListarTitulosInput{
		Status: resposta.ConsultaTexto(r, "status"),
		Origem: resposta.ConsultaTexto(r, "origem"),
		Filtro: resposta.ConsultaPaginacao(r),
	}

	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		return input, err
	}
	if input.ClienteID, err = resposta.ConsultaUUID(r, "cliente"); err != nil {
		return input, err
	}
	if input.OrigemID, err = resposta.ConsultaUUID(r, "origem_id"); err != nil {
		return input, err
	}
	if input.ContaBancariaID, err = resposta.ConsultaUUID(r, "conta"); err != nil {
		return input, err
	}
	if input.VencimentoDe, err = consultaData(r, "vencimento_de"); err != nil {
		return input, err
	}
	if input.VencimentoAte, err = consultaData(r, "vencimento_ate"); err != nil {
		return input, err
	}
	if input.Vencidos, err = consultaVerdadeira(r, "vencidos"); err != nil {
		return input, err
	}
	if input.OrigemCancelada, err = consultaVerdadeira(r, "origem_cancelada"); err != nil {
		return input, err
	}

	return input, nil
}

func (h *TituloHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	titulo, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) ListarHistorico(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	historico, err := h.usecase.ListarHistorico(r.Context(), id, usuario.ID)
	escreverLista(w, historico, err, receberdto.NovoHistoricoResponses)
}

func (h *TituloHandler) Editar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.EditarTituloRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, err := h.usecase.Editar(r.Context(), portsin.EditarTituloInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao, Dados: dados,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) Prorrogar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.ProrrogarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	vencimento, err := receberdto.ParaData(requisicao.Vencimento, "vencimento")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, err := h.usecase.Prorrogar(r.Context(), portsin.ProrrogarTituloInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao, Vencimento: vencimento, Motivo: requisicao.Motivo,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) EnviarAoBanco(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.EnviarAoBancoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	titulo, err := h.usecase.EnviarAoBanco(r.Context(), portsin.EnviarAoBancoInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao, ContaBancariaID: requisicao.ContaBancariaID,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) RetirarDoBanco(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	titulo, err := h.usecase.RetirarDoBanco(r.Context(), portsin.TransicaoTituloInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) RegistrarEnvioBoleto(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.BoletoEnviadoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	enviadoEm, err := receberdto.ParaMomento(requisicao.EnviadoEm, "envio do boleto")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, err := h.usecase.RegistrarEnvioBoleto(r.Context(), portsin.RegistrarEnvioBoletoInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao, Canal: requisicao.Canal, EnviadoEm: enviadoEm,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.CancelarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	titulo, err := h.usecase.Cancelar(r.Context(), portsin.CancelarTituloInput{
		UsuarioID: usuario.ID, TituloID: id, Versao: requisicao.Versao, Motivo: requisicao.Motivo,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}

func (h *TituloHandler) BoletosPendentes(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.BoletosPendentesInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Dias, err = consultaDias(r); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulos, err := h.usecase.BoletosPendentes(r.Context(), input)
	escreverPagina(w, input.Filtro, titulos, err, receberdto.NovoTituloResponses)
}
