package leadpoint

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"

type LeadUsecase struct {
	repo postgres.LeadRepository
	historico postgres.LeadHistoryRepository
}