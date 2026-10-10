package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/config"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/autenticacao"
	httpapi "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
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
	registrador := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := executar(registrador); err != nil {
		registrador.Error("a api parou", slog.String("erro", err.Error()))
		os.Exit(1)
	}
}

func executar(registrador *slog.Logger) error {
	cfg := config.LoadConfig()

	if err := cfg.Validar(); err != nil {
		return err
	}

	banco, err := postgres.ConnectionDB(cfg.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("não conectei no banco de dados: %w", err)
	}
	defer banco.Close()

	storageArquivos, err := montarStorage(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("não montei o storage de arquivos: %w", err)
	}

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
		cfg.CORSOrigins,
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
			storageArquivos,
			usuarioRepository,
			cargoRepository,
			aprovadorRepository,
		),
		solicitacaousecases.NewAprovadorUsecase(aprovadorRepository, usuarioRepository),
		estoqueusecases.NewCategoriaUsecase(categoriaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewProdutoUsecase(produtoRepository, categoriaRepository, imagemRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewMovimentoUsecase(produtoRepository, movimentoRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository),
		estoqueusecases.NewImagemUsecase(imagemRepository, produtoRepository, categoriaRepository, arquivoRepository, storageArquivos, usuarioRepository, cargoRepository),
		estoqueusecases.NewPublicoUsecase(categoriaRepository, produtoRepository, imagemRepository),
		autenticacao.NovoTokenService(cfg.JWTSecret, cfg.JWTExpiracao),
	)

	servidor := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           middleware.Registrar(registrador, rotas),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	falhas := make(chan error, 1)
	go ouvir(registrador, servidor, cfg.AppPort, falhas)

	select {
	case err := <-falhas:
		return err
	case <-aguardarSinal():
	}

	registrador.Info("encerrando a api")

	ctx, cancel := context.WithTimeout(context.Background(), tempoDeEncerramento)
	defer cancel()

	if err := servidor.Shutdown(ctx); err != nil {
		return fmt.Errorf("não encerrei a api a tempo: %w", err)
	}

	registrador.Info("api encerrada")

	return nil
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

func ouvir(registrador *slog.Logger, servidor *http.Server, porta string, falhas chan<- error) {
	registrador.Info("api ouvindo", slog.String("endereco", "http://localhost:"+porta))

	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		falhas <- fmt.Errorf("o servidor parou: %w", err)
	}
}

func aguardarSinal() <-chan os.Signal {
	parar := make(chan os.Signal, 1)
	signal.Notify(parar, os.Interrupt, syscall.SIGTERM)

	return parar
}
