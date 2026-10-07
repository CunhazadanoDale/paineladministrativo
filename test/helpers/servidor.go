//go:build e2e

package helpers

import (
	"net/http/httptest"
	"testing"

	httpapi "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/usuarios"
	"github.com/jmoiron/sqlx"
)

// NovoServidor monta a aplicação inteira — repositórios, usecases e rotas —
// e devolve um servidor HTTP de teste.
//
// É o mesmo caminho que o main sobe, só apontado para o banco de teste. Por
// depender da camada HTTP o arquivo fica atrás da tag `e2e`:
//
//	go test -tags=e2e ./test/...
func NovoServidor(t *testing.T, banco *sqlx.DB) *httptest.Server {
	t.Helper()

	cargoRepository := postgres.NewCargoRepository(banco)

	rotas := httpapi.NewRouter(
		banco,
		[]string{"*"},
		leadpoint.NewLeadUsecase(postgres.NewLeadRepository(banco)),
		leadpoint.NewFunilUsecase(postgres.NewFunilRepo(banco)),
		leadpoint.NewEtapaUsecase(postgres.NewEtapaRepository(banco)),
		leadpoint.NewLeadHistoryUsecase(postgres.NewLeadHistoryRepository(banco)),
		usuarios.NewUsuarioUsecase(postgres.NewUsuarioRepository(banco), cargoRepository),
		usuarios.NewCargoUsecase(cargoRepository),
	)

	servidor := httptest.NewServer(rotas)
	t.Cleanup(servidor.Close)

	return servidor
}
