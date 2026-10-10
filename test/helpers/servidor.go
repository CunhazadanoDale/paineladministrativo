//go:build e2e

package helpers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/autenticacao"
	httpapi "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/storage/disco"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/usuarios"
	"github.com/jmoiron/sqlx"
)

const segredoDoTeste = "segredo-de-teste-do-e2e"

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
	usuarioRepository := postgres.NewUsuarioRepository(banco)
	solicitacaoRepository := postgres.NewSolicitacaoRepository(banco)
	arquivoRepository := postgres.NewArquivoRepository(banco)
	aprovadorRepository := postgres.NewAprovadorRepository(banco)
	categoriaRepository := postgres.NewCategoriaRepository(banco)
	produtoRepository := postgres.NewProdutoRepository(banco)
	movimentoRepository := postgres.NewMovimentoRepository(banco)
	imagemRepository := postgres.NewImagemRepository(banco)

	rotas := httpapi.NewRouter(
		banco,
		[]string{"*"},
		leadpoint.NewLeadUsecase(postgres.NewLeadRepository(banco), postgres.NewEtapaRepository(banco)),
		leadpoint.NewFunilUsecase(postgres.NewFunilRepo(banco)),
		leadpoint.NewEtapaUsecase(postgres.NewEtapaRepository(banco)),
		leadpoint.NewLeadHistoryUsecase(postgres.NewLeadHistoryRepository(banco)),
		usuarios.NewUsuarioUsecase(usuarioRepository, cargoRepository),
		usuarios.NewCargoUsecase(cargoRepository, usuarioRepository),
		solicitacaousecases.NewSolicitacaoUsecase(
			solicitacaoRepository,
			arquivoRepository,
			aprovadorRepository,
			usuarioRepository,
			cargoRepository,
		),
		solicitacaousecases.NewArquivoUsecase(
			arquivoRepository,
			solicitacaoRepository,
			disco.Novo(t.TempDir()),
			usuarioRepository,
			cargoRepository,
			aprovadorRepository,
		),
		solicitacaousecases.NewAprovadorUsecase(aprovadorRepository, usuarioRepository),
		estoqueusecases.NewCategoriaUsecase(categoriaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewProdutoUsecase(produtoRepository, categoriaRepository, imagemRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewMovimentoUsecase(produtoRepository, movimentoRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository),
		estoqueusecases.NewImagemUsecase(imagemRepository, produtoRepository, arquivoRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewPublicoUsecase(categoriaRepository, produtoRepository, imagemRepository),
		autenticacao.NovoTokenService(segredoDoTeste, time.Hour),
	)

	servidor := httptest.NewServer(rotas)
	t.Cleanup(servidor.Close)

	return servidor
}
