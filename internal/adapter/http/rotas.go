package http

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	"github.com/jmoiron/sqlx"
)

func NewRouter(
	banco *sqlx.DB,
	leadUseCase portsin.LeadUseCase,
	funilUseCase portsin.FunilUseCase,
	etapaUseCase portsin.EtapaUseCase,
	historicoUseCase portsin.LeadHistoryUseCase,
) http.Handler {
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

	mux.HandleFunc("GET /", saude.NaoEncontrado)

	return mux
}
