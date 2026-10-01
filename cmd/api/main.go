package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/config"
	httpapi "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
)

const tempoDeEncerramento = 10 * time.Second

func main() {
	cfg := config.LoadConfig()

	banco, err := postgres.ConnectionDB(cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("não conectei no banco de dados: %v", err)
	}
	defer banco.Close()

	rotas := httpapi.NewRouter(
		banco,
		cfg.CORSOrigins,
		leadpoint.NewLeadUsecase(postgres.NewLeadRepository(banco)),
		leadpoint.NewFunilUsecase(postgres.NewFunilRepo(banco)),
		leadpoint.NewEtapaUsecase(postgres.NewEtapaRepository(banco)),
		leadpoint.NewLeadHistoryUsecase(postgres.NewLeadHistoryRepository(banco)),
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
