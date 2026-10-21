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
	caixausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/caixa"
	clienteusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/cliente"
	comissaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/comissao"
	dreusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/dre"
	empresausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/empresa"
	estoqueusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/estoque"
	financeirousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/financeiro"
	fornecedorusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/fornecedor"
	importacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/importacao"
	leadusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/lead"
	notafiscalusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/notafiscal"
	pedidousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/pedido"
	receberusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/receber"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	tabelaprecousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/tabelapreco"
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
	empresaRepository := postgres.NewEmpresaRepository(banco)
	estoqueRepository := postgres.NewEstoqueRepository(banco)
	localRepository := postgres.NewLocalRepository(banco)
	loteRepository := postgres.NewLoteRepository(banco)
	reservaRepository := postgres.NewReservaRepository(banco)
	clienteRepository := postgres.NewClienteRepository(banco)
	clienteGrupoRepository := postgres.NewClienteGrupoRepository(banco)
	clienteObraRepository := postgres.NewClienteObraRepository(banco)
	representanteRepository := postgres.NewRepresentanteRepository(banco)
	pedidoRepository := postgres.NewPedidoRepository(banco)
	entregaRepository := postgres.NewEntregaRepository(banco)
	notaFiscalRepository := postgres.NewNotaFiscalRepository(banco)
	clienteUsecase := clienteusecases.NewClienteUsecase(
		clienteRepository, clienteGrupoRepository, representanteRepository, empresaRepository,
		postgres.NewLeadRepository(banco), usuarioRepository, cargoRepository,
	)

	rotas := httpapi.NewRouter(
		banco,
		cfg.CORSOrigins,
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
			postgres.NewFornecedorRepository(banco),
			postgres.NewCategoriaFinanceiraRepository(banco),
			postgres.NewContaBancariaRepository(banco),
		),
		solicitacaousecases.NewArquivoUsecase(
			arquivoRepository,
			solicitacaoRepository,
			storageArquivos,
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
			reservaRepository, usuarioRepository, cargoRepository, empresaRepository,
		),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository, empresaRepository),
		estoqueusecases.NewImagemUsecase(imagemRepository, produtoRepository, categoriaRepository, arquivoRepository, storageArquivos, usuarioRepository, cargoRepository),
		estoqueusecases.NewPublicoUsecase(categoriaRepository, produtoRepository, imagemRepository),
		empresausecases.NewEmpresaUsecase(empresaRepository, usuarioRepository, cargoRepository),
		estoqueusecases.NewLocalUsecase(localRepository, estoqueRepository, usuarioRepository, cargoRepository, empresaRepository),
		estoqueusecases.NewReservaUsecase(
			produtoRepository, estoqueRepository, localRepository, reservaRepository,
			usuarioRepository, cargoRepository, empresaRepository,
		),
		clienteUsecase,
		clienteusecases.NewObraUsecase(clienteObraRepository, clienteRepository, representanteRepository, usuarioRepository, cargoRepository),
		clienteusecases.NewGrupoUsecase(clienteGrupoRepository, representanteRepository, usuarioRepository, cargoRepository),
		clienteusecases.NewRepresentanteUsecase(representanteRepository, usuarioRepository, cargoRepository),
		clienteUsecase,
		fornecedorusecases.NewFornecedorUsecase(postgres.NewFornecedorRepository(banco), usuarioRepository, cargoRepository),
		financeirousecases.NewContaBancariaUsecase(postgres.NewContaBancariaRepository(banco), empresaRepository, usuarioRepository, cargoRepository),
		financeirousecases.NewCategoriaFinanceiraUsecase(postgres.NewCategoriaFinanceiraRepository(banco), usuarioRepository, cargoRepository),
		solicitacaousecases.NewDespesaRecorrenteUsecase(
			postgres.NewDespesaRecorrenteRepository(banco), solicitacaoRepository,
			postgres.NewFornecedorRepository(banco), postgres.NewCategoriaFinanceiraRepository(banco),
			aprovadorRepository, usuarioRepository, cargoRepository, empresaRepository,
		),
		solicitacaousecases.NewResumoContasAPagarUsecase(
			postgres.NewContasAPagarResumoRepository(banco), aprovadorRepository, usuarioRepository, cargoRepository, empresaRepository,
		),
		pedidousecases.NewPedidoUsecase(
			pedidoRepository, entregaRepository, clienteRepository, clienteObraRepository, representanteRepository,
			produtoRepository, postgres.NewTabelaPrecoRepository(banco), empresaRepository, estoqueRepository,
			reservaRepository, localRepository, usuarioRepository, cargoRepository,
		),
		pedidousecases.NewEntregaUsecase(
			pedidoRepository, entregaRepository, representanteRepository, empresaRepository,
			estoqueRepository, reservaRepository, localRepository, usuarioRepository, cargoRepository,
		),
		pedidousecases.NewConsultaUsecase(
			postgres.NewPedidoConsultaRepository(banco), representanteRepository, empresaRepository, usuarioRepository, cargoRepository,
		),
		receberusecases.NewTituloUsecase(
			postgres.NewTituloReceberRepository(banco), clienteRepository, postgres.NewContaBancariaRepository(banco),
			postgres.NewCategoriaFinanceiraRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
		),
		receberusecases.NewEntradaDoPedidoUsecase(
			postgres.NewTituloReceberRepository(banco), pedidoRepository, postgres.NewContaBancariaRepository(banco),
			empresaRepository, usuarioRepository, cargoRepository,
		),
		receberusecases.NewRecebimentoUsecase(
			postgres.NewTituloReceberRepository(banco), postgres.NewAntecipacaoRepository(banco), pedidoRepository,
			postgres.NewContaBancariaRepository(banco), postgres.NewCategoriaFinanceiraRepository(banco),
			empresaRepository, usuarioRepository, cargoRepository,
		),
		receberusecases.NewAntecipacaoUsecase(
			postgres.NewAntecipacaoRepository(banco), postgres.NewTituloReceberRepository(banco), clienteRepository,
			postgres.NewFornecedorRepository(banco), postgres.NewContaBancariaRepository(banco),
			postgres.NewCategoriaFinanceiraRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
		),
		receberusecases.NewResumoContasAReceberUsecase(
			postgres.NewContasAReceberResumoRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
		),
		notafiscalusecases.NewNotaFiscalUsecase(
			notaFiscalRepository, arquivoRepository, storageArquivos, empresaRepository, clienteRepository, produtoRepository,
			pedidoRepository, entregaRepository, postgres.NewTituloReceberRepository(banco), representanteRepository,
			usuarioRepository, cargoRepository,
		),
		notafiscalusecases.NewEventoUsecase(
			notaFiscalRepository, arquivoRepository, storageArquivos, empresaRepository, clienteRepository,
			postgres.NewTituloReceberRepository(banco), representanteRepository, usuarioRepository, cargoRepository,
		),
		comissaousecases.NewPercentualUsecase(
			postgres.NewComissaoRepository(banco), pedidoRepository, representanteRepository, empresaRepository,
			usuarioRepository, cargoRepository,
		),
		comissaousecases.NewComissaoUsecase(
			postgres.NewComissaoRepository(banco), representanteRepository, empresaRepository, usuarioRepository, cargoRepository,
		),
		comissaousecases.NewAjusteUsecase(
			postgres.NewComissaoRepository(banco), pedidoRepository, representanteRepository, empresaRepository,
			usuarioRepository, cargoRepository,
		),
		comissaousecases.NewFechamentoUsecase(
			postgres.NewComissaoRepository(banco), postgres.NewCategoriaFinanceiraRepository(banco), representanteRepository,
			empresaRepository, usuarioRepository, cargoRepository,
		),
		tabelaprecousecases.NewTabelaPrecoUsecase(
			postgres.NewTabelaPrecoRepository(banco), produtoRepository, usuarioRepository, cargoRepository,
		),
		caixausecases.NewAberturaUsecase(
			postgres.NewCaixaAberturaRepository(banco), postgres.NewContaBancariaRepository(banco), empresaRepository,
			usuarioRepository, cargoRepository,
		),
		caixausecases.NewExtratoUsecase(
			postgres.NewExtratoRepository(banco), postgres.NewCaixaAberturaRepository(banco), arquivoRepository, storageArquivos,
			postgres.NewContaBancariaRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
		),
		caixausecases.NewSaldoUsecase(postgres.NewCaixaSaldoRepository(banco), empresaRepository, usuarioRepository, cargoRepository),
		caixausecases.NewConciliacaoUsecase(
			postgres.NewConciliacaoRepository(banco), postgres.NewContaBancariaRepository(banco), empresaRepository,
			usuarioRepository, cargoRepository,
		),
		caixausecases.NewLancamentoCaixaUsecase(
			postgres.NewLancamentoCaixaRepository(banco), postgres.NewConciliacaoRepository(banco),
			postgres.NewCategoriaFinanceiraRepository(banco), postgres.NewContaBancariaRepository(banco), empresaRepository,
			usuarioRepository, cargoRepository,
		),
		caixausecases.NewFechamentoUsecase(
			postgres.NewCaixaFechamentoRepository(banco), postgres.NewCaixaAberturaRepository(banco),
			postgres.NewConciliacaoRepository(banco), postgres.NewContaBancariaRepository(banco), empresaRepository,
			usuarioRepository, cargoRepository,
		),
		caixausecases.NewProjecaoUsecase(
			postgres.NewCaixaProjecaoRepository(banco), postgres.NewDespesaRecorrenteRepository(banco),
			postgres.NewContaBancariaRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
		),
		importacaousecases.NewImportacaoUsecase(
			postgres.NewImportacaoRepository(banco), postgres.NewFornecedorRepository(banco), produtoRepository, empresaRepository,
			usuarioRepository, cargoRepository,
		),
		importacaousecases.NewFinanceiroUsecase(
			postgres.NewImportacaoRepository(banco), postgres.NewFornecedorRepository(banco), postgres.NewCategoriaFinanceiraRepository(banco),
			empresaRepository, usuarioRepository, cargoRepository,
		),
		importacaousecases.NewRecebimentoUsecase(
			postgres.NewImportacaoRepository(banco), estoqueRepository, localRepository, empresaRepository,
			usuarioRepository, cargoRepository,
		),
		importacaousecases.NewConsultaUsecase(
			postgres.NewImportacaoConsultaRepository(banco), postgres.NewImportacaoRepository(banco), empresaRepository,
			usuarioRepository, cargoRepository,
		),
		dreusecases.NewResumoDREUsecase(
			postgres.NewDREResumoRepository(banco), usuarioRepository, cargoRepository, empresaRepository,
		),
		dreusecases.NewFaturamentoUsecase(
			postgres.NewDREResumoRepository(banco), usuarioRepository, cargoRepository, empresaRepository,
		),
		dreusecases.NewRelatorioSemanalUsecase(
			postgres.NewDRERelatorioSemanalRepository(banco), postgres.NewCaixaSaldoRepository(banco),
			usuarioRepository, cargoRepository, empresaRepository,
		),
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
