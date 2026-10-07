package http

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	solicitacaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsinsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/jmoiron/sqlx"
)

func NewRouter(
	banco *sqlx.DB,
	origensCORS []string,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoryUseCase,
	usuarioUseCase portsinusuarios.UsuarioUseCase,
	cargoUseCase portsinusuarios.CargoUseCase,
	solicitacaoUseCase portsinsolicitacao.SolicitacaoUseCase,
	arquivoUseCase portsinsolicitacao.ArquivoUseCase,
	aprovadorUseCase portsinsolicitacao.AprovadorUseCase,
	tokens portsinautenticacao.TokenService,
) http.Handler {
	return middleware.CORS(origensCORS, novasRotas(
		banco,
		leadUseCase, funilUseCase, etapaUseCase, historicoUseCase,
		usuarioUseCase, cargoUseCase,
		solicitacaoUseCase, arquivoUseCase, aprovadorUseCase,
		tokens,
	))
}

func novasRotas(
	banco *sqlx.DB,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoryUseCase,
	usuarioUseCase portsinusuarios.UsuarioUseCase,
	cargoUseCase portsinusuarios.CargoUseCase,
	solicitacaoUseCase portsinsolicitacao.SolicitacaoUseCase,
	arquivoUseCase portsinsolicitacao.ArquivoUseCase,
	aprovadorUseCase portsinsolicitacao.AprovadorUseCase,
	tokens portsinautenticacao.TokenService,
) *http.ServeMux {
	mux := http.NewServeMux()

	protegido := func(padrao string, proximo http.HandlerFunc) {
		mux.Handle(padrao, middleware.Autenticar(tokens, usuarioUseCase, proximo))
	}

	administrador := func(padrao string, proximo http.HandlerFunc) {
		protegido(padrao, middleware.ExigirAdministrador(usuarioUseCase, proximo))
	}

	mux.HandleFunc("GET /health", saude.Responder)
	mux.HandleFunc("GET /health/db", saude.ResponderComBanco(banco))

	leadHandler := lead.NewLeadHandler(leadUseCase)
	protegido("POST /api/v1/leads", leadHandler.Criar)
	protegido("GET /api/v1/leads", leadHandler.Listar)
	protegido("GET /api/v1/leads/{id}", leadHandler.Obter)
	protegido("PUT /api/v1/leads/{id}", leadHandler.Atualizar)
	protegido("DELETE /api/v1/leads/{id}", leadHandler.Remover)
	protegido("PATCH /api/v1/leads/{id}/etapa", leadHandler.MoverEtapa)
	protegido("GET /api/v1/funils/{funil_id}/leads", leadHandler.ListarPorFunil)
	protegido("GET /api/v1/etapas/{etapa_id}/leads", leadHandler.ListarPorEtapa)
	protegido("GET /api/v1/funils/{funil_id}/leads/contagem", leadHandler.ContarPorFunil)
	protegido("GET /api/v1/etapas/{etapa_id}/leads/contagem", leadHandler.ContarPorEtapa)

	funilHandler := lead.NewFunilHandler(funilUseCase)
	protegido("POST /api/v1/funils", funilHandler.Criar)
	protegido("GET /api/v1/funils", funilHandler.Listar)
	protegido("GET /api/v1/funils/{funil_id}", funilHandler.Obter)
	protegido("PUT /api/v1/funils/{funil_id}", funilHandler.Atualizar)
	protegido("DELETE /api/v1/funils/{funil_id}", funilHandler.Remover)

	etapaHandler := lead.NewEtapaHandler(etapaUseCase)
	protegido("POST /api/v1/etapas", etapaHandler.Criar)
	protegido("GET /api/v1/funils/{funil_id}/etapas", etapaHandler.ListarPorFunil)
	protegido("PUT /api/v1/funils/{funil_id}/etapas/ordem", etapaHandler.Reordenar)
	protegido("GET /api/v1/etapas/{etapa_id}", etapaHandler.Obter)
	protegido("PUT /api/v1/etapas/{etapa_id}", etapaHandler.Atualizar)
	protegido("DELETE /api/v1/etapas/{etapa_id}", etapaHandler.Remover)
	protegido("GET /api/v1/etapas/{etapa_id}/proxima", etapaHandler.Proxima)
	protegido("GET /api/v1/etapas/{etapa_id}/anterior", etapaHandler.Anterior)

	historicoHandler := lead.NewLeadHistoryHandler(historicoUseCase)
	protegido("GET /api/v1/leads/{lead_id}/historico", historicoHandler.Listar)
	protegido("POST /api/v1/leads/{lead_id}/historico", historicoHandler.RegistrarMovimentacao)

	cargoHandler := usuarios.NewCargoHandler(cargoUseCase)
	administrador("POST /api/v1/cargos", cargoHandler.Criar)
	protegido("GET /api/v1/cargos", cargoHandler.Listar)
	protegido("GET /api/v1/cargos/busca", cargoHandler.ObterPorNome)
	protegido("GET /api/v1/cargos/{id}", cargoHandler.Obter)
	administrador("PUT /api/v1/cargos/{id}", cargoHandler.Atualizar)
	administrador("DELETE /api/v1/cargos/{id}", cargoHandler.Remover)

	usuarioHandler := usuarios.NewUsuarioHandler(usuarioUseCase, tokens)
	administrador("POST /api/v1/usuarios", usuarioHandler.Criar)
	administrador("GET /api/v1/usuarios", usuarioHandler.Listar)
	mux.HandleFunc("POST /api/v1/usuarios/autenticar", usuarioHandler.Autenticar)
	administrador("GET /api/v1/usuarios/{id}", usuarioHandler.Obter)
	administrador("PUT /api/v1/usuarios/{id}", usuarioHandler.Atualizar)
	administrador("DELETE /api/v1/usuarios/{id}", usuarioHandler.Remover)
	administrador("POST /api/v1/usuarios/{id}/senha", usuarioHandler.TrocarSenha)
	administrador("PATCH /api/v1/usuarios/{id}/ativar", usuarioHandler.Ativar)
	administrador("PATCH /api/v1/usuarios/{id}/desativar", usuarioHandler.Desativar)

	solicitacaoHandler := solicitacaohandlers.NewSolicitacaoHandler(solicitacaoUseCase)
	protegido("POST /api/v1/solicitacoes", solicitacaoHandler.Criar)
	protegido("GET /api/v1/solicitacoes", solicitacaoHandler.Listar)
	protegido("GET /api/v1/solicitacoes/{id}", solicitacaoHandler.Obter)
	protegido("POST /api/v1/solicitacoes/{id}/aprovar", solicitacaoHandler.Aprovar)
	protegido("POST /api/v1/solicitacoes/{id}/rejeitar", solicitacaoHandler.Rejeitar)
	protegido("POST /api/v1/solicitacoes/{id}/cancelar", solicitacaoHandler.Cancelar)
	protegido("POST /api/v1/solicitacoes/{id}/pagamentos", solicitacaoHandler.RegistrarPagamento)
	protegido("GET /api/v1/solicitacoes/{id}/pagamento", solicitacaoHandler.ObterPagamento)
	protegido("GET /api/v1/solicitacoes/{id}/historico", solicitacaoHandler.ListarHistorico)
	protegido("GET /api/v1/solicitacoes/{id}/arquivos", solicitacaoHandler.ListarArquivos)

	arquivoHandler := solicitacaohandlers.NewArquivoHandler(arquivoUseCase)
	protegido("POST /api/v1/arquivos", arquivoHandler.Enviar)
	protegido("GET /api/v1/arquivos", arquivoHandler.Listar)
	protegido("GET /api/v1/arquivos/{id}", arquivoHandler.Baixar)
	protegido("DELETE /api/v1/arquivos/{id}", arquivoHandler.Remover)

	aprovadorHandler := solicitacaohandlers.NewAprovadorHandler(aprovadorUseCase)
	administrador("POST /api/v1/aprovadores", aprovadorHandler.Designar)
	administrador("GET /api/v1/aprovadores", aprovadorHandler.Listar)
	administrador("DELETE /api/v1/aprovadores/{id}", aprovadorHandler.Remover)

	mux.HandleFunc("GET /", saude.NaoEncontrado)

	return mux
}
