package http

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsinlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
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
) http.Handler {
	return middleware.CORS(origensCORS, novasRotas(banco, leadUseCase, funilUseCase, etapaUseCase, historicoUseCase, usuarioUseCase, cargoUseCase))
}

func novasRotas(
	banco *sqlx.DB,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoryUseCase,
	usuarioUseCase portsinusuarios.UsuarioUseCase,
	cargoUseCase portsinusuarios.CargoUseCase,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", saude.Responder)
	mux.HandleFunc("GET /health/db", saude.ResponderComBanco(banco))

	leadHandler := lead.NewLeadHandler(leadUseCase)
	mux.HandleFunc("POST /api/v1/leads", leadHandler.Criar)
	mux.HandleFunc("GET /api/v1/leads", leadHandler.Listar)
	mux.HandleFunc("GET /api/v1/leads/{id}", leadHandler.Obter)
	mux.HandleFunc("PUT /api/v1/leads/{id}", leadHandler.Atualizar)
	mux.HandleFunc("DELETE /api/v1/leads/{id}", leadHandler.Remover)
	mux.HandleFunc("PATCH /api/v1/leads/{id}/etapa", leadHandler.MoverEtapa)
	mux.HandleFunc("GET /api/v1/funils/{funil_id}/leads", leadHandler.ListarPorFunil)
	mux.HandleFunc("GET /api/v1/etapas/{etapa_id}/leads", leadHandler.ListarPorEtapa)
	mux.HandleFunc("GET /api/v1/funils/{funil_id}/leads/contagem", leadHandler.ContarPorFunil)
	mux.HandleFunc("GET /api/v1/etapas/{etapa_id}/leads/contagem", leadHandler.ContarPorEtapa)

	funilHandler := lead.NewFunilHandler(funilUseCase)
	mux.HandleFunc("POST /api/v1/funils", funilHandler.Criar)
	mux.HandleFunc("GET /api/v1/funils", funilHandler.Listar)
	mux.HandleFunc("GET /api/v1/funils/{funil_id}", funilHandler.Obter)
	mux.HandleFunc("PUT /api/v1/funils/{funil_id}", funilHandler.Atualizar)
	mux.HandleFunc("DELETE /api/v1/funils/{funil_id}", funilHandler.Remover)

	etapaHandler := lead.NewEtapaHandler(etapaUseCase)
	mux.HandleFunc("POST /api/v1/etapas", etapaHandler.Criar)
	mux.HandleFunc("GET /api/v1/funils/{funil_id}/etapas", etapaHandler.ListarPorFunil)
	mux.HandleFunc("PUT /api/v1/funils/{funil_id}/etapas/ordem", etapaHandler.Reordenar)
	mux.HandleFunc("GET /api/v1/etapas/{etapa_id}", etapaHandler.Obter)
	mux.HandleFunc("PUT /api/v1/etapas/{etapa_id}", etapaHandler.Atualizar)
	mux.HandleFunc("DELETE /api/v1/etapas/{etapa_id}", etapaHandler.Remover)
	mux.HandleFunc("GET /api/v1/etapas/{etapa_id}/proxima", etapaHandler.Proxima)
	mux.HandleFunc("GET /api/v1/etapas/{etapa_id}/anterior", etapaHandler.Anterior)

	historicoHandler := lead.NewLeadHistoryHandler(historicoUseCase)
	mux.HandleFunc("GET /api/v1/leads/{lead_id}/historico", historicoHandler.Listar)
	mux.HandleFunc("POST /api/v1/leads/{lead_id}/historico", historicoHandler.RegistrarMovimentacao)

	cargoHandler := usuarios.NewCargoHandler(cargoUseCase)
	mux.HandleFunc("POST /api/v1/cargos", cargoHandler.Criar)
	mux.HandleFunc("GET /api/v1/cargos", cargoHandler.Listar)
	mux.HandleFunc("GET /api/v1/cargos/busca", cargoHandler.ObterPorNome)
	mux.HandleFunc("GET /api/v1/cargos/{id}", cargoHandler.Obter)
	mux.HandleFunc("PUT /api/v1/cargos/{id}", cargoHandler.Atualizar)
	mux.HandleFunc("DELETE /api/v1/cargos/{id}", cargoHandler.Remover)

	usuarioHandler := usuarios.NewUsuarioHandler(usuarioUseCase)
	mux.HandleFunc("POST /api/v1/usuarios", usuarioHandler.Criar)
	mux.HandleFunc("GET /api/v1/usuarios", usuarioHandler.Listar)
	mux.HandleFunc("POST /api/v1/usuarios/autenticar", usuarioHandler.Autenticar)
	mux.HandleFunc("GET /api/v1/usuarios/{id}", usuarioHandler.Obter)
	mux.HandleFunc("PUT /api/v1/usuarios/{id}", usuarioHandler.Atualizar)
	mux.HandleFunc("DELETE /api/v1/usuarios/{id}", usuarioHandler.Remover)
	mux.HandleFunc("POST /api/v1/usuarios/{id}/senha", usuarioHandler.TrocarSenha)
	mux.HandleFunc("PATCH /api/v1/usuarios/{id}/ativar", usuarioHandler.Ativar)
	mux.HandleFunc("PATCH /api/v1/usuarios/{id}/desativar", usuarioHandler.Desativar)

	mux.HandleFunc("GET /", saude.NaoEncontrado)

	return mux
}
