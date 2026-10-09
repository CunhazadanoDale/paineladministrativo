package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/config"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/autenticacao"
	httpapi "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/storage/disco"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/storage/r2"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/usuarios"
)

const (
	storageDisco = "disco"
	storageR2    = "r2"
)

const tempoDeEncerramento = 10 * time.Second

func main() {
	cfg := config.LoadConfig()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET não configurada: defina o segredo usado nos tokens de acesso")
	}

	banco, err := postgres.ConnectionDB(cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("não conectei no banco de dados: %v", err)
	}
	defer banco.Close()

	storageArquivos, err := montarStorage(context.Background(), cfg)
	if err != nil {
		log.Fatalf("não montei o storage de arquivos: %v", err)
	}

	cargoRepository := postgres.NewCargoRepository(banco)
	usuarioRepository := postgres.NewUsuarioRepository(banco)
	solicitacaoRepository := postgres.NewSolicitacaoRepository(banco)
	arquivoRepository := postgres.NewArquivoRepository(banco)
	aprovadorRepository := postgres.NewAprovadorRepository(banco)
	categoriaRepository := postgres.NewCategoriaRepository(banco)
	produtoRepository := postgres.NewProdutoRepository(banco)
	movimentoRepository := postgres.NewMovimentoRepository(banco)

	rotas := httpapi.NewRouter(
		banco,
		cfg.CORSOrigins,
		leadpoint.NewLeadUsecase(postgres.NewLeadRepository(banco)),
		leadpoint.NewFunilUsecase(postgres.NewFunilRepo(banco)),
		leadpoint.NewEtapaUsecase(postgres.NewEtapaRepository(banco)),
		leadpoint.NewLeadHistoryUsecase(postgres.NewLeadHistoryRepository(banco)),
		usuarios.NewUsuarioUsecase(usuarioRepository, cargoRepository),
		usuarios.NewCargoUsecase(cargoRepository),
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
			storageArquivos,
			usuarioRepository,
			cargoRepository,
			aprovadorRepository,
		),
		solicitacaousecases.NewAprovadorUsecase(aprovadorRepository, usuarioRepository),
		estoqueusecases.NewCategoriaUsecase(categoriaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewProdutoUsecase(produtoRepository, categoriaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewMovimentoUsecase(produtoRepository, movimentoRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository),
		autenticacao.NovoTokenService(cfg.JWTSecret, cfg.JWTExpiracao),
	)

	servidor := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           rotas,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go ouvir(servidor, cfg.AppPort)

	aguardarSinal()

	log.Print("encerrando a api")

	ctx, cancel := context.WithTimeout(context.Background(), tempoDeEncerramento)
	defer cancel()

	if err := servidor.Shutdown(ctx); err != nil {
		log.Fatalf("não encerrei a api a tempo: %v", err)
	}

	log.Print("api encerrada")
}

func montarStorage(ctx context.Context, cfg *config.Config) (solicitacao.Storage, error) {
	switch cfg.StorageDriver {
	case storageDisco:
		return disco.Novo(cfg.StorageDir), nil
	case storageR2:
		if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" || cfg.R2Bucket == "" {
			return nil, errors.New("STORAGE_DRIVER=r2 exige R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY e R2_BUCKET")
		}

		return r2.Novo(ctx, r2.Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			Bucket:          cfg.R2Bucket,
		})
	default:
		return nil, fmt.Errorf("STORAGE_DRIVER %q desconhecido: use %q ou %q", cfg.StorageDriver, storageDisco, storageR2)
	}
}

func ouvir(servidor *http.Server, porta string) {
	log.Printf("api ouvindo em http://localhost:%s", porta)

	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("o servidor parou: %v", err)
	}
}

func aguardarSinal() {
	parar := make(chan os.Signal, 1)
	signal.Notify(parar, os.Interrupt, syscall.SIGTERM)

	<-parar
}
