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
	caixausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/caixa"
	clienteusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/cliente"
	comissaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/comissao"
	dividausecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/divida"
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
			postgres.NewFornecedorRepository(banco),
			postgres.NewCategoriaFinanceiraRepository(banco),
			postgres.NewContaBancariaRepository(banco),
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
			reservaRepository, usuarioRepository, cargoRepository, empresaRepository,
		),
		estoqueusecases.NewResumoUsecase(produtoRepository, movimentoRepository, empresaRepository),
		estoqueusecases.NewImagemUsecase(imagemRepository, produtoRepository, categoriaRepository, arquivoRepository, armazenamento, usuarioRepository, cargoRepository),
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
			notaFiscalRepository, arquivoRepository, armazenamento, empresaRepository, clienteRepository, produtoRepository,
			pedidoRepository, entregaRepository, postgres.NewTituloReceberRepository(banco), representanteRepository,
			usuarioRepository, cargoRepository,
		),
		notafiscalusecases.NewEventoUsecase(
			notaFiscalRepository, arquivoRepository, armazenamento, empresaRepository, clienteRepository,
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
			postgres.NewExtratoRepository(banco), postgres.NewCaixaAberturaRepository(banco), arquivoRepository, armazenamento,
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
		dividausecases.NewDividaUsecase(
			postgres.NewDividaRepository(banco), postgres.NewDividaConsultaRepository(banco), postgres.NewFornecedorRepository(banco), postgres.NewCategoriaFinanceiraRepository(banco),
			postgres.NewContaBancariaRepository(banco), empresaRepository, usuarioRepository, cargoRepository,
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
		autenticacao.NovoTokenService(segredoDoTeste, time.Hour),
	)

	servidor := httptest.NewServer(rotas)
	t.Cleanup(servidor.Close)

	return servidor
}
