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
	empresausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/empresa"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
	leadusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/lead"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/usuarios"
	"github.com/jmoiron/sqlx"
)

const segredoDoTeste = "segredo-de-teste-do-e2e"

func NovoServidor(t *testing.T, banco *sqlx.DB) *httptest.Server {
	t.Helper()

	cargoRepository := postgres.NewCargoRepository(banco)
	usuarioRepository := postgres.NewUsuarioRepository(banco)
	armazenamento := disco.Novo(t.TempDir())
	solicitacaoRepository := postgres.NewSolicitacaoRepository(banco)
	arquivoRepository := postgres.NewArquivoRepository(banco)
	aprovadorRepository := postgres.NewAprovadorRepository(banco)
	categoriaRepository := postgres.NewCategoriaRepository(banco)
	produtoRepository := postgres.NewProdutoRepository(banco)
	movimentoRepository := postgres.NewMovimentoRepository(banco)
	imagemRepository := postgres.NewImagemRepository(banco)
	empresaRepository := postgres.NewEmpresaRepository(banco)
	estoqueRepository := postgres.NewEstoqueRepository(banco)
	localRepository := postgres.NewLocalRepository(banco)
	loteRepository := postgres.NewLoteRepository(banco)

	rotas := httpapi.NewRouter(
		banco,
		[]string{"*"},
		leadusecases.NewLeadUsecase(postgres.NewLeadRepository(banco), postgres.NewEtapaRepository(banco)),
		leadusecases.NewFunilUsecase(postgres.NewFunilRepository(banco)),
		leadusecases.NewEtapaUsecase(postgres.NewEtapaRepository(banco)),
		leadusecases.NewLeadHistoricoUsecase(postgres.NewLeadHistoricoRepository(banco)),
		usuarios.NewUsuarioUsecase(usuarioRepository, cargoRepository),
		usuarios.NewCargoUsecase(cargoRepository, usuarioRepository),
		solicitacaousecases.NewSolicitacaoUsecase(
			solicitacaoRepository,
			arquivoRepository,
			aprovadorRepository,
			usuarioRepository,
			cargoRepository,
			empresaRepository,
		),
		solicitacaousecases.NewArquivoUsecase(
			arquivoRepository,
			solicitacaoRepository,
			armazenamento,
			usuarioRepository,
			cargoRepository,
			aprovadorRepository,
			empresaRepository,
		),
		solicitacaousecases.NewAprovadorUsecase(aprovadorRepository, usuarioRepository),
		estoqueusecases.NewCategoriaUsecase(categoriaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewProdutoUsecase(produtoRepository, categoriaRepository, imagemRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewMovimentoUsecase(
			produtoRepository, movimentoRepository, estoqueRepository, localRepository, loteRepository,
			usuarioRepository, cargoRepository, empresaRepository,
		),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository, empresaRepository),
		estoqueusecases.NewImagemUsecase(imagemRepository, produtoRepository, categoriaRepository, arquivoRepository, armazenamento, usuarioRepository, cargoRepository),
		estoqueusecases.NewPublicoUsecase(categoriaRepository, produtoRepository, imagemRepository),
		empresausecases.NewEmpresaUsecase(empresaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewLocalUsecase(localRepository, estoqueRepository, usuarioRepository, cargoRepository, empresaRepository),
		autenticacao.NovoTokenService(segredoDoTeste, time.Hour),
	)

	servidor := httptest.NewServer(rotas)
	t.Cleanup(servidor.Close)

	return servidor
}
