package http

import (
	"net/http"
	"time"

	caixahandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/caixa"
	clientehandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/cliente"
	comissaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/comissao"
	dividahandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/divida"
	drehandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/dre"
	empresahandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/empresa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/estoque"
	financeirohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/financeiro"
	fornecedorhandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/fornecedor"
	importacaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/importacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	notafiscalhandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/notafiscal"
	pedidohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/pedido"
	receberhandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	solicitacaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/solicitacao"
	tabelaprecohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/tabelapreco"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
	portsindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	portsinempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	portsinestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsinfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	portsinfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	portsinimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsinlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsinnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	portsinpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsinreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsinsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/jmoiron/sqlx"
)

const prazoDeArquivo = 5 * time.Minute

func NewRouter(
	banco *sqlx.DB,
	origensCORS []string,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoricoUseCase,
	usuarioUseCase portsinusuarios.UsuarioUseCase,
	cargoUseCase portsinusuarios.CargoUseCase,
	solicitacaoUseCase portsinsolicitacao.SolicitacaoUseCase,
	arquivoUseCase portsinsolicitacao.ArquivoUseCase,
	aprovadorUseCase portsinsolicitacao.AprovadorUseCase,
	categoriaUseCase portsinestoque.CategoriaUseCase,
	produtoUseCase portsinestoque.ProdutoUseCase,
	movimentoUseCase portsinestoque.MovimentoUseCase,
	resumoUseCase portsinestoque.ResumoUseCase,
	imagemUseCase portsinestoque.ImagemUseCase,
	publicoUseCase portsinestoque.PublicoUseCase,
	empresaUseCase portsinempresa.EmpresaUseCase,
	localUseCase portsinestoque.LocalUseCase,
	reservaUseCase portsinestoque.ReservaUseCase,
	clienteUseCase portsincliente.ClienteUseCase,
	obraUseCase portsincliente.ObraUseCase,
	grupoClienteUseCase portsincliente.GrupoUseCase,
	representanteUseCase portsincliente.RepresentanteUseCase,
	importacaoClientesUseCase portsincliente.ImportacaoUseCase,
	fornecedorUseCase portsinfornecedor.FornecedorUseCase,
	contaBancariaUseCase portsinfinanceiro.ContaBancariaUseCase,
	categoriaFinanceiraUseCase portsinfinanceiro.CategoriaFinanceiraUseCase,
	despesaRecorrenteUseCase portsinsolicitacao.DespesaRecorrenteUseCase,
	resumoContasAPagarUseCase portsinsolicitacao.ResumoContasAPagarUseCase,
	pedidoUseCase portsinpedido.PedidoUseCase,
	entregaUseCase portsinpedido.EntregaUseCase,
	consultaPedidosUseCase portsinpedido.ConsultaUseCase,
	tituloUseCase portsinreceber.TituloUseCase,
	entradaDoPedidoUseCase portsinreceber.EntradaDoPedidoUseCase,
	recebimentoUseCase portsinreceber.RecebimentoUseCase,
	antecipacaoUseCase portsinreceber.AntecipacaoUseCase,
	resumoContasAReceberUseCase portsinreceber.ResumoContasAReceberUseCase,
	notaFiscalUseCase portsinnotafiscal.NotaFiscalUseCase,
	eventoNotaFiscalUseCase portsinnotafiscal.EventoUseCase,
	percentualComissaoUseCase portsincomissao.PercentualUseCase,
	comissaoUseCase portsincomissao.ComissaoUseCase,
	ajusteComissaoUseCase portsincomissao.AjusteUseCase,
	fechamentoComissaoUseCase portsincomissao.FechamentoUseCase,
	tabelaPrecoUseCase portsintabelapreco.TabelaPrecoUseCase,
	aberturaCaixaUseCase portsincaixa.AberturaUseCase,
	extratoUseCase portsincaixa.ExtratoUseCase,
	saldoCaixaUseCase portsincaixa.SaldoUseCase,
	conciliacaoUseCase portsincaixa.ConciliacaoUseCase,
	lancamentoCaixaUseCase portsincaixa.LancamentoCaixaUseCase,
	fechamentoCaixaUseCase portsincaixa.FechamentoUseCase,
	projecaoCaixaUseCase portsincaixa.ProjecaoUseCase,
	importacaoUseCase portsinimportacao.ImportacaoUseCase,
	financeiroImportacaoUseCase portsinimportacao.FinanceiroUseCase,
	recebimentoImportacaoUseCase portsinimportacao.RecebimentoUseCase,
	consultaImportacaoUseCase portsinimportacao.ConsultaUseCase,
	dividaUseCase portsindivida.DividaUseCase,
	resumoDREUseCase portsindre.ResumoDREUseCase,
	faturamentoUseCase portsindre.FaturamentoUseCase,
	relatorioSemanalUseCase portsindre.RelatorioSemanalUseCase,
	importacaoLeadsUseCase portsinlead.ImportacaoUseCase,
	importacaoProdutosUseCase portsinestoque.ImportacaoProdutoUseCase,
	importacaoTabelaPrecoUseCase portsintabelapreco.ImportacaoUseCase,
	tokens portsinautenticacao.TokenService,
) http.Handler {
	return middleware.CORS(origensCORS, middleware.ErrosDeRoteamento(novasRotas(
		banco,
		leadUseCase, funilUseCase, etapaUseCase, historicoUseCase,
		usuarioUseCase, cargoUseCase,
		solicitacaoUseCase, arquivoUseCase, aprovadorUseCase,
		categoriaUseCase, produtoUseCase, movimentoUseCase, resumoUseCase,
		imagemUseCase, publicoUseCase,
		empresaUseCase,
		localUseCase,
		reservaUseCase,
		clienteUseCase, obraUseCase, grupoClienteUseCase, representanteUseCase,
		importacaoClientesUseCase,
		fornecedorUseCase,
		contaBancariaUseCase, categoriaFinanceiraUseCase,
		despesaRecorrenteUseCase,
		resumoContasAPagarUseCase,
		pedidoUseCase, entregaUseCase, consultaPedidosUseCase,
		tituloUseCase, entradaDoPedidoUseCase, recebimentoUseCase, antecipacaoUseCase,
		resumoContasAReceberUseCase,
		notaFiscalUseCase, eventoNotaFiscalUseCase,
		percentualComissaoUseCase, comissaoUseCase, ajusteComissaoUseCase, fechamentoComissaoUseCase,
		tabelaPrecoUseCase,
		aberturaCaixaUseCase, extratoUseCase, saldoCaixaUseCase,
		conciliacaoUseCase,
		lancamentoCaixaUseCase,
		fechamentoCaixaUseCase,
		projecaoCaixaUseCase,
		importacaoUseCase, financeiroImportacaoUseCase, recebimentoImportacaoUseCase, consultaImportacaoUseCase,
		dividaUseCase,
		resumoDREUseCase, faturamentoUseCase, relatorioSemanalUseCase,
		importacaoLeadsUseCase,
		importacaoProdutosUseCase,
		importacaoTabelaPrecoUseCase,
		tokens,
	)))
}

func novasRotas(
	banco *sqlx.DB,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoricoUseCase,
	usuarioUseCase portsinusuarios.UsuarioUseCase,
	cargoUseCase portsinusuarios.CargoUseCase,
	solicitacaoUseCase portsinsolicitacao.SolicitacaoUseCase,
	arquivoUseCase portsinsolicitacao.ArquivoUseCase,
	aprovadorUseCase portsinsolicitacao.AprovadorUseCase,
	categoriaUseCase portsinestoque.CategoriaUseCase,
	produtoUseCase portsinestoque.ProdutoUseCase,
	movimentoUseCase portsinestoque.MovimentoUseCase,
	resumoUseCase portsinestoque.ResumoUseCase,
	imagemUseCase portsinestoque.ImagemUseCase,
	publicoUseCase portsinestoque.PublicoUseCase,
	empresaUseCase portsinempresa.EmpresaUseCase,
	localUseCase portsinestoque.LocalUseCase,
	reservaUseCase portsinestoque.ReservaUseCase,
	clienteUseCase portsincliente.ClienteUseCase,
	obraUseCase portsincliente.ObraUseCase,
	grupoClienteUseCase portsincliente.GrupoUseCase,
	representanteUseCase portsincliente.RepresentanteUseCase,
	importacaoClientesUseCase portsincliente.ImportacaoUseCase,
	fornecedorUseCase portsinfornecedor.FornecedorUseCase,
	contaBancariaUseCase portsinfinanceiro.ContaBancariaUseCase,
	categoriaFinanceiraUseCase portsinfinanceiro.CategoriaFinanceiraUseCase,
	despesaRecorrenteUseCase portsinsolicitacao.DespesaRecorrenteUseCase,
	resumoContasAPagarUseCase portsinsolicitacao.ResumoContasAPagarUseCase,
	pedidoUseCase portsinpedido.PedidoUseCase,
	entregaUseCase portsinpedido.EntregaUseCase,
	consultaPedidosUseCase portsinpedido.ConsultaUseCase,
	tituloUseCase portsinreceber.TituloUseCase,
	entradaDoPedidoUseCase portsinreceber.EntradaDoPedidoUseCase,
	recebimentoUseCase portsinreceber.RecebimentoUseCase,
	antecipacaoUseCase portsinreceber.AntecipacaoUseCase,
	resumoContasAReceberUseCase portsinreceber.ResumoContasAReceberUseCase,
	notaFiscalUseCase portsinnotafiscal.NotaFiscalUseCase,
	eventoNotaFiscalUseCase portsinnotafiscal.EventoUseCase,
	percentualComissaoUseCase portsincomissao.PercentualUseCase,
	comissaoUseCase portsincomissao.ComissaoUseCase,
	ajusteComissaoUseCase portsincomissao.AjusteUseCase,
	fechamentoComissaoUseCase portsincomissao.FechamentoUseCase,
	tabelaPrecoUseCase portsintabelapreco.TabelaPrecoUseCase,
	aberturaCaixaUseCase portsincaixa.AberturaUseCase,
	extratoUseCase portsincaixa.ExtratoUseCase,
	saldoCaixaUseCase portsincaixa.SaldoUseCase,
	conciliacaoUseCase portsincaixa.ConciliacaoUseCase,
	lancamentoCaixaUseCase portsincaixa.LancamentoCaixaUseCase,
	fechamentoCaixaUseCase portsincaixa.FechamentoUseCase,
	projecaoCaixaUseCase portsincaixa.ProjecaoUseCase,
	importacaoUseCase portsinimportacao.ImportacaoUseCase,
	financeiroImportacaoUseCase portsinimportacao.FinanceiroUseCase,
	recebimentoImportacaoUseCase portsinimportacao.RecebimentoUseCase,
	consultaImportacaoUseCase portsinimportacao.ConsultaUseCase,
	dividaUseCase portsindivida.DividaUseCase,
	resumoDREUseCase portsindre.ResumoDREUseCase,
	faturamentoUseCase portsindre.FaturamentoUseCase,
	relatorioSemanalUseCase portsindre.RelatorioSemanalUseCase,
	importacaoLeadsUseCase portsinlead.ImportacaoUseCase,
	importacaoProdutosUseCase portsinestoque.ImportacaoProdutoUseCase,
	importacaoTabelaPrecoUseCase portsintabelapreco.ImportacaoUseCase,
	tokens portsinautenticacao.TokenService,
) *http.ServeMux {
	mux := http.NewServeMux()

	protegido := func(padrao string, proximo http.HandlerFunc) {
		mux.Handle(padrao, middleware.Autenticar(tokens, usuarioUseCase, proximo))
	}

	administrador := func(padrao string, proximo http.HandlerFunc) {
		protegido(padrao, middleware.ExigirAdministrador(usuarioUseCase, proximo))
	}

	comercial := func(padrao string, proximo http.HandlerFunc) {
		protegido(padrao, middleware.ExigirComercial(usuarioUseCase, proximo))
	}

	financeiro := func(padrao string, proximo http.HandlerFunc) {
		protegido(padrao, middleware.ExigirFinanceiro(usuarioUseCase, proximo))
	}

	mux.HandleFunc("GET /health", saude.Responder)
	mux.HandleFunc("GET /health/db", saude.ResponderComBanco(banco))

	leadHandler := lead.NewLeadHandler(leadUseCase)
	comercial("POST /api/v1/leads", leadHandler.Criar)
	comercial("GET /api/v1/leads", leadHandler.Listar)
	comercial("GET /api/v1/leads/{id}", leadHandler.Obter)
	comercial("PUT /api/v1/leads/{id}", leadHandler.Atualizar)
	comercial("DELETE /api/v1/leads/{id}", leadHandler.Remover)
	comercial("PATCH /api/v1/leads/{id}/etapa", leadHandler.MoverEtapa)
	comercial("GET /api/v1/funils/{funil_id}/leads", leadHandler.ListarPorFunil)
	comercial("GET /api/v1/etapas/{etapa_id}/leads", leadHandler.ListarPorEtapa)
	comercial("GET /api/v1/funils/{funil_id}/leads/contagem", leadHandler.ContarPorFunil)
	comercial("GET /api/v1/etapas/{etapa_id}/leads/contagem", leadHandler.ContarPorEtapa)

	funilHandler := lead.NewFunilHandler(funilUseCase)
	administrador("POST /api/v1/funils", funilHandler.Criar)
	comercial("GET /api/v1/funils", funilHandler.Listar)
	comercial("GET /api/v1/funils/{funil_id}", funilHandler.Obter)
	administrador("PUT /api/v1/funils/{funil_id}", funilHandler.Atualizar)
	administrador("DELETE /api/v1/funils/{funil_id}", funilHandler.Remover)

	etapaHandler := lead.NewEtapaHandler(etapaUseCase)
	administrador("POST /api/v1/etapas", etapaHandler.Criar)
	comercial("GET /api/v1/funils/{funil_id}/etapas", etapaHandler.ListarPorFunil)
	administrador("PUT /api/v1/funils/{funil_id}/etapas/ordem", etapaHandler.Reordenar)
	comercial("GET /api/v1/etapas/{etapa_id}", etapaHandler.Obter)
	administrador("PUT /api/v1/etapas/{etapa_id}", etapaHandler.Atualizar)
	administrador("DELETE /api/v1/etapas/{etapa_id}", etapaHandler.Remover)
	comercial("GET /api/v1/etapas/{etapa_id}/proxima", etapaHandler.Proxima)
	comercial("GET /api/v1/etapas/{etapa_id}/anterior", etapaHandler.Anterior)

	historicoHandler := lead.NewLeadHistoricoHandler(historicoUseCase)
	comercial("GET /api/v1/leads/{lead_id}/historico", historicoHandler.Listar)

	cargoHandler := usuarios.NewCargoHandler(cargoUseCase)
	administrador("POST /api/v1/cargos", cargoHandler.Criar)
	protegido("GET /api/v1/cargos", cargoHandler.Listar)
	protegido("GET /api/v1/cargos/busca", cargoHandler.ObterPorNome)
	protegido("GET /api/v1/cargos/{id}", cargoHandler.Obter)
	administrador("PUT /api/v1/cargos/{id}", cargoHandler.Atualizar)
	administrador("DELETE /api/v1/cargos/{id}", cargoHandler.Remover)

	usuarioHandler := usuarios.NewUsuarioHandler(usuarioUseCase, tokens, middleware.NovoLimitadorDeTentativas())
	administrador("POST /api/v1/usuarios", usuarioHandler.Criar)
	administrador("GET /api/v1/usuarios", usuarioHandler.Listar)
	mux.HandleFunc("POST /api/v1/usuarios/autenticar", usuarioHandler.Autenticar)
	administrador("GET /api/v1/usuarios/{id}", usuarioHandler.Obter)
	administrador("PUT /api/v1/usuarios/{id}", usuarioHandler.Atualizar)
	administrador("DELETE /api/v1/usuarios/{id}", usuarioHandler.Remover)
	administrador("POST /api/v1/usuarios/{id}/senha", usuarioHandler.TrocarSenha)
	administrador("PATCH /api/v1/usuarios/{id}/ativar", usuarioHandler.Ativar)
	administrador("PATCH /api/v1/usuarios/{id}/desativar", usuarioHandler.Desativar)

	sessaoHandler := usuarios.NewSessaoHandler(usuarioUseCase)
	protegido("GET /api/v1/sessao", sessaoHandler.Obter)
	protegido("POST /api/v1/sessao/senha", sessaoHandler.TrocarSenha)
	protegido("POST /api/v1/sessao/encerrar", sessaoHandler.Encerrar)

	solicitacaoHandler := solicitacaohandlers.NewSolicitacaoHandler(solicitacaoUseCase)
	protegido("POST /api/v1/solicitacoes", solicitacaoHandler.Criar)
	protegido("GET /api/v1/solicitacoes", solicitacaoHandler.Listar)
	protegido("GET /api/v1/solicitacoes/{id}", solicitacaoHandler.Obter)
	protegido("POST /api/v1/solicitacoes/{id}/aprovar", solicitacaoHandler.Aprovar)
	protegido("POST /api/v1/solicitacoes/{id}/rejeitar", solicitacaoHandler.Rejeitar)
	protegido("POST /api/v1/solicitacoes/{id}/cancelar", solicitacaoHandler.Cancelar)
	protegido("POST /api/v1/solicitacoes/{id}/pagamento", solicitacaoHandler.RegistrarPagamento)
	protegido("GET /api/v1/solicitacoes/{id}/pagamento", solicitacaoHandler.ObterPagamento)
	protegido("GET /api/v1/solicitacoes/{id}/historico", solicitacaoHandler.ListarHistorico)
	protegido("GET /api/v1/solicitacoes/{id}/arquivos", solicitacaoHandler.ListarArquivos)
	protegido("PATCH /api/v1/solicitacoes/{id}/classificacao", solicitacaoHandler.Classificar)
	protegido("PUT /api/v1/solicitacoes/{id}", solicitacaoHandler.Editar)
	protegido("POST /api/v1/solicitacoes/{id}/adiar", solicitacaoHandler.Adiar)
	protegido("POST /api/v1/solicitacoes/{id}/pagamento/estorno", solicitacaoHandler.EstornarPagamento)
	protegido("GET /api/v1/solicitacoes/{id}/pagamentos", solicitacaoHandler.ListarPagamentos)
	protegido("GET /api/v1/solicitacoes/pendencias", solicitacaoHandler.Pendencias)

	arquivoHandler := solicitacaohandlers.NewArquivoHandler(arquivoUseCase)
	protegido("POST /api/v1/arquivos", middleware.EstenderPrazo(prazoDeArquivo, arquivoHandler.Enviar))
	protegido("GET /api/v1/arquivos", arquivoHandler.Listar)
	protegido("GET /api/v1/arquivos/{id}", middleware.EstenderPrazo(prazoDeArquivo, arquivoHandler.Baixar))
	protegido("DELETE /api/v1/arquivos/{id}", arquivoHandler.Remover)

	aprovadorHandler := solicitacaohandlers.NewAprovadorHandler(aprovadorUseCase)
	administrador("POST /api/v1/aprovadores", aprovadorHandler.Designar)
	administrador("GET /api/v1/aprovadores", aprovadorHandler.Listar)
	administrador("DELETE /api/v1/aprovadores/{id}", aprovadorHandler.Remover)

	despesaRecorrenteHandler := solicitacaohandlers.NewDespesaRecorrenteHandler(despesaRecorrenteUseCase)
	financeiro("POST /api/v1/despesas-recorrentes", despesaRecorrenteHandler.Criar)
	financeiro("GET /api/v1/despesas-recorrentes", despesaRecorrenteHandler.Listar)
	financeiro("POST /api/v1/despesas-recorrentes/gerar", despesaRecorrenteHandler.Gerar)
	financeiro("GET /api/v1/despesas-recorrentes/{id}", despesaRecorrenteHandler.Obter)
	financeiro("PUT /api/v1/despesas-recorrentes/{id}", despesaRecorrenteHandler.Alterar)
	financeiro("PATCH /api/v1/despesas-recorrentes/{id}/ativar", despesaRecorrenteHandler.Ativar)
	financeiro("PATCH /api/v1/despesas-recorrentes/{id}/desativar", despesaRecorrenteHandler.Desativar)
	administrador("DELETE /api/v1/despesas-recorrentes/{id}", despesaRecorrenteHandler.Remover)

	resumoContasAPagarHandler := solicitacaohandlers.NewResumoContasAPagarHandler(resumoContasAPagarUseCase)
	financeiro("GET /api/v1/contas-a-pagar/resumo", resumoContasAPagarHandler.Consultar)

	categoriaHandler := estoque.NewCategoriaHandler(categoriaUseCase)
	administrador("POST /api/v1/categorias", categoriaHandler.Criar)
	protegido("GET /api/v1/categorias", categoriaHandler.Listar)
	protegido("GET /api/v1/categorias/{id}", categoriaHandler.Obter)
	administrador("PUT /api/v1/categorias/{id}", categoriaHandler.Alterar)
	administrador("PATCH /api/v1/categorias/{id}/ativar", categoriaHandler.Ativar)
	administrador("PATCH /api/v1/categorias/{id}/desativar", categoriaHandler.Desativar)

	produtoHandler := estoque.NewProdutoHandler(produtoUseCase)
	administrador("POST /api/v1/produtos", produtoHandler.Criar)
	protegido("GET /api/v1/produtos", produtoHandler.Listar)
	protegido("GET /api/v1/produtos/{id}", produtoHandler.Obter)
	administrador("PUT /api/v1/produtos/{id}", produtoHandler.Alterar)
	administrador("PATCH /api/v1/produtos/{id}/ativar", produtoHandler.Ativar)
	administrador("PATCH /api/v1/produtos/{id}/desativar", produtoHandler.Desativar)
	administrador("PATCH /api/v1/produtos/{id}/destaque", produtoHandler.AlternarDestaque)

	movimentoHandler := estoque.NewMovimentoHandler(movimentoUseCase)
	administrador("POST /api/v1/produtos/{id}/movimentos", movimentoHandler.Movimentar)
	protegido("GET /api/v1/produtos/{id}/movimentos", movimentoHandler.Listar)

	resumoHandler := estoque.NewResumoHandler(resumoUseCase)
	protegido("GET /api/v1/estoque/resumo", resumoHandler.Consultar)

	imagemHandler := estoque.NewImagemHandler(imagemUseCase)
	administrador("POST /api/v1/produtos/{id}/imagens", imagemHandler.Anexar)
	administrador("DELETE /api/v1/produtos/{id}/imagens/{imagemId}", imagemHandler.Remover)

	publicoHandler := estoque.NewPublicoHandler(publicoUseCase)
	mux.HandleFunc("GET /api/v1/publico/categorias", publicoHandler.ListarCategorias)
	mux.HandleFunc("GET /api/v1/publico/produtos", publicoHandler.ListarProdutos)
	mux.HandleFunc("GET /api/v1/publico/produtos/{slug}", publicoHandler.ObterProduto)
	mux.HandleFunc("GET /api/v1/publico/destaques", publicoHandler.ListarDestaques)

	imagemPublicaHandler := estoque.NewImagemPublicaHandler(imagemUseCase, arquivoUseCase)
	mux.HandleFunc("GET /api/v1/publico/imagens/{id}", middleware.EstenderPrazo(prazoDeArquivo, imagemPublicaHandler.Servir))

	empresaHandler := empresahandlers.NewEmpresaHandler(empresaUseCase)
	administrador("POST /api/v1/empresas", empresaHandler.Criar)
	protegido("GET /api/v1/empresas", empresaHandler.Listar)
	protegido("GET /api/v1/empresas/{id}", empresaHandler.Obter)
	administrador("PUT /api/v1/empresas/{id}", empresaHandler.Alterar)
	administrador("PATCH /api/v1/empresas/{id}/ativar", empresaHandler.Ativar)
	administrador("PATCH /api/v1/empresas/{id}/desativar", empresaHandler.Desativar)
	administrador("DELETE /api/v1/empresas/{id}", empresaHandler.Remover)
	administrador("GET /api/v1/usuarios/{id}/empresas", empresaHandler.ObterVinculos)
	administrador("PUT /api/v1/usuarios/{id}/empresas", empresaHandler.DefinirVinculos)

	localHandler := estoque.NewLocalHandler(localUseCase)
	administrador("POST /api/v1/estoque/locais", localHandler.Criar)
	protegido("GET /api/v1/estoque/locais", localHandler.Listar)
	protegido("GET /api/v1/estoque/locais/{id}", localHandler.Obter)
	administrador("PUT /api/v1/estoque/locais/{id}", localHandler.Renomear)
	administrador("PATCH /api/v1/estoque/locais/{id}/ativar", localHandler.Ativar)
	administrador("PATCH /api/v1/estoque/locais/{id}/desativar", localHandler.Desativar)
	administrador("DELETE /api/v1/estoque/locais/{id}", localHandler.Remover)
	protegido("GET /api/v1/estoque/locais/{id}/saldos", localHandler.ListarSaldos)
	protegido("GET /api/v1/produtos/{id}/saldo", movimentoHandler.Saldo)
	protegido("GET /api/v1/produtos/{id}/lotes", movimentoHandler.ListarLotes)
	administrador("POST /api/v1/produtos/{id}/transferencias", movimentoHandler.Transferir)

	reservaHandler := estoque.NewReservaHandler(reservaUseCase)
	administrador("POST /api/v1/produtos/{id}/reservas", reservaHandler.Reservar)
	protegido("GET /api/v1/produtos/{id}/reservas", reservaHandler.Listar)
	administrador("POST /api/v1/reservas/{id}/liberar", reservaHandler.Liberar)

	clienteHandler := clientehandlers.NewClienteHandler(clienteUseCase)
	comercial("POST /api/v1/clientes", clienteHandler.Criar)
	protegido("GET /api/v1/clientes", clienteHandler.Listar)
	protegido("GET /api/v1/clientes/{id}", clienteHandler.Obter)
	comercial("PUT /api/v1/clientes/{id}", clienteHandler.Alterar)
	comercial("PATCH /api/v1/clientes/{id}/ativar", clienteHandler.Ativar)
	comercial("PATCH /api/v1/clientes/{id}/desativar", clienteHandler.Desativar)
	administrador("DELETE /api/v1/clientes/{id}", clienteHandler.Remover)

	obraHandler := clientehandlers.NewObraHandler(obraUseCase)
	comercial("POST /api/v1/clientes/{id}/obras", obraHandler.Criar)
	protegido("GET /api/v1/clientes/{id}/obras", obraHandler.Listar)
	protegido("GET /api/v1/obras/{id}", obraHandler.Obter)
	comercial("PUT /api/v1/obras/{id}", obraHandler.Alterar)
	comercial("PATCH /api/v1/obras/{id}/ativar", obraHandler.Ativar)
	comercial("PATCH /api/v1/obras/{id}/desativar", obraHandler.Desativar)

	grupoClienteHandler := clientehandlers.NewGrupoHandler(grupoClienteUseCase)
	comercial("POST /api/v1/cliente-grupos", grupoClienteHandler.Criar)
	protegido("GET /api/v1/cliente-grupos", grupoClienteHandler.Listar)
	protegido("GET /api/v1/cliente-grupos/{id}", grupoClienteHandler.Obter)
	comercial("PUT /api/v1/cliente-grupos/{id}", grupoClienteHandler.Renomear)
	comercial("PATCH /api/v1/cliente-grupos/{id}/ativar", grupoClienteHandler.Ativar)
	comercial("PATCH /api/v1/cliente-grupos/{id}/desativar", grupoClienteHandler.Desativar)

	representanteHandler := clientehandlers.NewRepresentanteHandler(representanteUseCase)
	administrador("POST /api/v1/representantes", representanteHandler.Criar)
	comercial("GET /api/v1/representantes", representanteHandler.Listar)
	comercial("GET /api/v1/representantes/{id}", representanteHandler.Obter)
	administrador("PUT /api/v1/representantes/{id}", representanteHandler.Alterar)
	administrador("PATCH /api/v1/representantes/{id}/ativar", representanteHandler.Ativar)
	administrador("PATCH /api/v1/representantes/{id}/desativar", representanteHandler.Desativar)

	comercial("POST /api/v1/leads/{id}/converter", clienteHandler.ConverterLead)
	importacaoClientesHandler := clientehandlers.NewImportacaoHandler(importacaoClientesUseCase)
	administrador("POST /api/v1/clientes/importacao", middleware.EstenderPrazo(prazoDeArquivo, importacaoClientesHandler.Importar))

	fornecedorHandler := fornecedorhandlers.NewFornecedorHandler(fornecedorUseCase)
	financeiro("POST /api/v1/fornecedores", fornecedorHandler.Criar)
	protegido("GET /api/v1/fornecedores", fornecedorHandler.Listar)
	protegido("GET /api/v1/fornecedores/{id}", fornecedorHandler.Obter)
	financeiro("PUT /api/v1/fornecedores/{id}", fornecedorHandler.Alterar)
	financeiro("PATCH /api/v1/fornecedores/{id}/ativar", fornecedorHandler.Ativar)
	financeiro("PATCH /api/v1/fornecedores/{id}/desativar", fornecedorHandler.Desativar)
	administrador("DELETE /api/v1/fornecedores/{id}", fornecedorHandler.Remover)

	contaBancariaHandler := financeirohandlers.NewContaBancariaHandler(contaBancariaUseCase)
	financeiro("POST /api/v1/contas-bancarias", contaBancariaHandler.Criar)
	financeiro("GET /api/v1/contas-bancarias", contaBancariaHandler.Listar)
	financeiro("GET /api/v1/contas-bancarias/{id}", contaBancariaHandler.Obter)
	financeiro("PUT /api/v1/contas-bancarias/{id}", contaBancariaHandler.Alterar)
	financeiro("PATCH /api/v1/contas-bancarias/{id}/ativar", contaBancariaHandler.Ativar)
	financeiro("PATCH /api/v1/contas-bancarias/{id}/desativar", contaBancariaHandler.Desativar)
	administrador("DELETE /api/v1/contas-bancarias/{id}", contaBancariaHandler.Remover)
	protegido("GET /api/v1/bancos", contaBancariaHandler.Bancos)

	categoriaFinanceiraHandler := financeirohandlers.NewCategoriaFinanceiraHandler(categoriaFinanceiraUseCase)
	financeiro("POST /api/v1/categorias-financeiras", categoriaFinanceiraHandler.Criar)
	protegido("GET /api/v1/categorias-financeiras", categoriaFinanceiraHandler.Listar)
	protegido("GET /api/v1/categorias-financeiras/arvore", categoriaFinanceiraHandler.Arvore)
	protegido("GET /api/v1/categorias-financeiras/{id}", categoriaFinanceiraHandler.Obter)
	financeiro("PUT /api/v1/categorias-financeiras/{id}", categoriaFinanceiraHandler.Alterar)
	financeiro("PATCH /api/v1/categorias-financeiras/{id}/ativar", categoriaFinanceiraHandler.Ativar)
	financeiro("PATCH /api/v1/categorias-financeiras/{id}/desativar", categoriaFinanceiraHandler.Desativar)
	administrador("DELETE /api/v1/categorias-financeiras/{id}", categoriaFinanceiraHandler.Remover)

	pedidoHandler := pedidohandlers.NewPedidoHandler(pedidoUseCase)
	comercial("POST /api/v1/pedidos", pedidoHandler.Criar)
	protegido("GET /api/v1/pedidos", pedidoHandler.Listar)
	protegido("GET /api/v1/pedidos/{id}", pedidoHandler.Obter)
	comercial("PUT /api/v1/pedidos/{id}", pedidoHandler.Alterar)
	comercial("POST /api/v1/pedidos/{id}/confirmar", pedidoHandler.Confirmar)
	financeiro("POST /api/v1/pedidos/{id}/entrada", pedidoHandler.RegistrarEntrada)
	comercial("POST /api/v1/pedidos/{id}/reservar-pendencias", pedidoHandler.ReservarPendencias)
	comercial("POST /api/v1/pedidos/{id}/cancelar", pedidoHandler.Cancelar)
	protegido("GET /api/v1/pedidos/{id}/historico", pedidoHandler.ListarHistorico)

	entregaHandler := pedidohandlers.NewEntregaHandler(entregaUseCase)
	comercial("POST /api/v1/pedidos/{id}/entregas", entregaHandler.Criar)
	protegido("GET /api/v1/pedidos/{id}/entregas", entregaHandler.ListarDoPedido)
	protegido("GET /api/v1/entregas", entregaHandler.Listar)
	protegido("GET /api/v1/entregas/{id}", entregaHandler.Obter)
	comercial("POST /api/v1/entregas/{id}/saiu", entregaHandler.RegistrarSaida)
	comercial("POST /api/v1/entregas/{id}/entregue", entregaHandler.Concluir)
	comercial("POST /api/v1/entregas/{id}/cancelar", entregaHandler.Cancelar)

	consultaPedidosHandler := pedidohandlers.NewConsultaHandler(consultaPedidosUseCase)
	protegido("GET /api/v1/pedidos/pendencias", consultaPedidosHandler.Pendencias)
	protegido("GET /api/v1/pedidos/prazos", consultaPedidosHandler.Prazos)
	protegido("GET /api/v1/pedidos/cancelamentos", consultaPedidosHandler.Cancelamentos)

	tituloHandler := receberhandlers.NewTituloHandler(tituloUseCase)
	financeiro("POST /api/v1/titulos-receber", tituloHandler.Criar)
	financeiro("GET /api/v1/titulos-receber", tituloHandler.Listar)
	financeiro("GET /api/v1/titulos-receber/{id}", tituloHandler.Obter)
	financeiro("GET /api/v1/titulos-receber/{id}/historico", tituloHandler.ListarHistorico)
	financeiro("PUT /api/v1/titulos-receber/{id}", tituloHandler.Editar)
	financeiro("POST /api/v1/titulos-receber/{id}/prorrogar", tituloHandler.Prorrogar)
	financeiro("POST /api/v1/titulos-receber/{id}/enviar-banco", tituloHandler.EnviarAoBanco)
	financeiro("POST /api/v1/titulos-receber/{id}/retirar-banco", tituloHandler.RetirarDoBanco)
	financeiro("POST /api/v1/titulos-receber/{id}/boleto-enviado", tituloHandler.RegistrarEnvioBoleto)
	financeiro("POST /api/v1/titulos-receber/{id}/cancelar", tituloHandler.Cancelar)

	entradaDoPedidoHandler := receberhandlers.NewEntradaDoPedidoHandler(entradaDoPedidoUseCase)
	financeiro("POST /api/v1/pedidos/{id}/titulo-entrada", entradaDoPedidoHandler.GerarTitulo)

	recebimentoHandler := receberhandlers.NewRecebimentoHandler(recebimentoUseCase)
	financeiro("POST /api/v1/titulos-receber/{id}/recebimentos", recebimentoHandler.Receber)
	financeiro("GET /api/v1/titulos-receber/{id}/recebimentos", recebimentoHandler.Listar)
	financeiro("POST /api/v1/titulos-receber/{id}/recebimentos/{recebimento_id}/estorno", recebimentoHandler.Estornar)

	antecipacaoHandler := receberhandlers.NewAntecipacaoHandler(antecipacaoUseCase)
	financeiro("POST /api/v1/antecipacoes", antecipacaoHandler.Criar)
	financeiro("GET /api/v1/antecipacoes", antecipacaoHandler.Listar)
	financeiro("GET /api/v1/antecipacoes/{id}", antecipacaoHandler.Obter)
	financeiro("POST /api/v1/antecipacoes/{id}/estorno", antecipacaoHandler.Estornar)
	financeiro("POST /api/v1/titulos-receber/{id}/recompra", antecipacaoHandler.Recomprar)

	financeiro("GET /api/v1/titulos-receber/boletos-pendentes", tituloHandler.BoletosPendentes)
	resumoContasAReceberHandler := receberhandlers.NewResumoContasAReceberHandler(resumoContasAReceberUseCase)
	financeiro("GET /api/v1/contas-a-receber/resumo", resumoContasAReceberHandler.Consultar)

	notaFiscalHandler := notafiscalhandlers.NewNotaFiscalHandler(notaFiscalUseCase)
	financeiro("POST /api/v1/notas-fiscais/importacao", notaFiscalHandler.Importar)
	protegido("GET /api/v1/notas-fiscais", notaFiscalHandler.Listar)
	protegido("GET /api/v1/notas-fiscais/itens-sem-produto", notaFiscalHandler.ListarItensSemProduto)
	protegido("GET /api/v1/notas-fiscais/{id}", notaFiscalHandler.Obter)
	protegido("GET /api/v1/notas-fiscais/{id}/xml", middleware.EstenderPrazo(prazoDeArquivo, notaFiscalHandler.BaixarXML))
	financeiro("PUT /api/v1/notas-fiscais/{id}/vinculo", notaFiscalHandler.Vincular)
	financeiro("PUT /api/v1/notas-fiscais/{id}/itens/{numero}/produto", notaFiscalHandler.VincularProduto)
	protegido("GET /api/v1/entregas/sem-nota-fiscal", notaFiscalHandler.ListarEntregasSemNota)
	financeiro("POST /api/v1/notas-fiscais/{id}/titulos", notaFiscalHandler.SincronizarTitulos)

	eventoNotaFiscalHandler := notafiscalhandlers.NewEventoHandler(eventoNotaFiscalUseCase)
	financeiro("POST /api/v1/notas-fiscais/eventos", eventoNotaFiscalHandler.Importar)
	protegido("GET /api/v1/notas-fiscais/{id}/eventos", eventoNotaFiscalHandler.Listar)

	percentualComissaoHandler := comissaohandlers.NewPercentualHandler(percentualComissaoUseCase)
	protegido("GET /api/v1/pedidos/{id}/comissao", percentualComissaoHandler.Obter)
	comercial("PUT /api/v1/pedidos/{id}/comissao", percentualComissaoHandler.Definir)

	comissaoHandler := comissaohandlers.NewComissaoHandler(comissaoUseCase)
	financeiro("POST /api/v1/comissoes/apurar", comissaoHandler.Apurar)
	financeiro("GET /api/v1/comissoes/pendencias", comissaoHandler.Pendencias)
	protegido("GET /api/v1/comissoes", comissaoHandler.Listar)

	administrador("PUT /api/v1/representantes/{id}/fornecedor", representanteHandler.DefinirFornecedor)

	ajusteComissaoHandler := comissaohandlers.NewAjusteHandler(ajusteComissaoUseCase)
	financeiro("POST /api/v1/comissoes/ajustes", ajusteComissaoHandler.Criar)
	protegido("GET /api/v1/comissoes/ajustes", ajusteComissaoHandler.Listar)

	fechamentoComissaoHandler := comissaohandlers.NewFechamentoHandler(fechamentoComissaoUseCase)
	financeiro("POST /api/v1/comissoes/fechamentos", fechamentoComissaoHandler.Fechar)
	protegido("GET /api/v1/comissoes/fechamentos", fechamentoComissaoHandler.Listar)
	protegido("GET /api/v1/comissoes/fechamentos/{id}", fechamentoComissaoHandler.Obter)
	protegido("GET /api/v1/comissoes/resumo", comissaoHandler.Resumo)

	tabelaPrecoHandler := tabelaprecohandlers.NewTabelaPrecoHandler(tabelaPrecoUseCase)
	administrador("POST /api/v1/tabela-precos", tabelaPrecoHandler.Criar)
	protegido("GET /api/v1/tabela-precos", tabelaPrecoHandler.Listar)
	protegido("GET /api/v1/tabela-precos/vigentes", tabelaPrecoHandler.Vigentes)
	administrador("DELETE /api/v1/tabela-precos/{id}", tabelaPrecoHandler.Remover)
	protegido("GET /api/v1/pedidos/{id}/proposta", pedidoHandler.Documento)
	protegido("GET /api/v1/modelos-venda", pedidoHandler.ModelosDeVenda)

	aberturaCaixaHandler := caixahandlers.NewAberturaHandler(aberturaCaixaUseCase)
	financeiro("PUT /api/v1/caixa/contas/{id}/abertura", aberturaCaixaHandler.Definir)
	financeiro("GET /api/v1/caixa/contas/{id}/abertura", aberturaCaixaHandler.Obter)

	saldoCaixaHandler := caixahandlers.NewSaldoHandler(saldoCaixaUseCase)
	financeiro("GET /api/v1/caixa/saldos", saldoCaixaHandler.Listar)

	extratoHandler := caixahandlers.NewExtratoHandler(extratoUseCase)
	financeiro("POST /api/v1/extratos/importacao", middleware.EstenderPrazo(prazoDeArquivo, extratoHandler.Importar))
	financeiro("GET /api/v1/extratos/importacoes", extratoHandler.ListarImportacoes)
	administrador("DELETE /api/v1/extratos/importacoes/{id}", extratoHandler.DesfazerImportacao)
	financeiro("GET /api/v1/extratos/lancamentos", extratoHandler.ListarLancamentos)

	conciliacaoHandler := caixahandlers.NewConciliacaoHandler(conciliacaoUseCase)
	financeiro("GET /api/v1/extratos/lancamentos/{id}/sugestoes", conciliacaoHandler.Sugestoes)
	financeiro("POST /api/v1/extratos/lancamentos/{id}/conciliacao", conciliacaoHandler.Conciliar)
	financeiro("POST /api/v1/extratos/conciliar-automatico", conciliacaoHandler.ConciliarAutomatico)
	financeiro("GET /api/v1/conciliacoes/{id}", conciliacaoHandler.Obter)
	financeiro("POST /api/v1/conciliacoes/{id}/desfazer", conciliacaoHandler.Desfazer)
	financeiro("GET /api/v1/caixa/pendencias-sistema", conciliacaoHandler.Pendencias)
	financeiro("GET /api/v1/caixa/pagamentos-sem-conta", conciliacaoHandler.PagamentosSemConta)

	lancamentoCaixaHandler := caixahandlers.NewLancamentoCaixaHandler(lancamentoCaixaUseCase)
	financeiro("POST /api/v1/extratos/lancamentos/{id}/classificacao", lancamentoCaixaHandler.Classificar)
	financeiro("POST /api/v1/extratos/lancamentos/{id}/transferencia", lancamentoCaixaHandler.Transferir)
	financeiro("GET /api/v1/caixa/lancamentos", lancamentoCaixaHandler.Listar)
	financeiro("GET /api/v1/caixa/transferencias", lancamentoCaixaHandler.ListarTransferencias)

	fechamentoCaixaHandler := caixahandlers.NewFechamentoHandler(fechamentoCaixaUseCase)
	financeiro("GET /api/v1/caixa/fechamentos/previa", fechamentoCaixaHandler.Previa)
	financeiro("POST /api/v1/caixa/fechamentos", fechamentoCaixaHandler.Fechar)
	financeiro("GET /api/v1/caixa/fechamentos", fechamentoCaixaHandler.Listar)
	financeiro("GET /api/v1/caixa/fechamentos/{id}", fechamentoCaixaHandler.Obter)
	administrador("POST /api/v1/caixa/fechamentos/{id}/reabrir", fechamentoCaixaHandler.Reabrir)

	projecaoCaixaHandler := caixahandlers.NewProjecaoHandler(projecaoCaixaUseCase)
	financeiro("GET /api/v1/caixa/projecao", projecaoCaixaHandler.Projetar)
	financeiro("GET /api/v1/caixa/movimento-diario", projecaoCaixaHandler.MovimentoDiario)

	importacaoHandler := importacaohandlers.NewImportacaoHandler(importacaoUseCase)
	financeiro("POST /api/v1/importacoes", importacaoHandler.Criar)
	protegido("GET /api/v1/importacoes", importacaoHandler.Listar)
	protegido("GET /api/v1/importacoes/{id}", importacaoHandler.Obter)
	financeiro("PUT /api/v1/importacoes/{id}", importacaoHandler.Alterar)
	financeiro("POST /api/v1/importacoes/{id}/embarcar", importacaoHandler.Embarcar)
	financeiro("POST /api/v1/importacoes/{id}/desembaracar", importacaoHandler.Desembaracar)
	financeiro("POST /api/v1/importacoes/{id}/cancelar", importacaoHandler.Cancelar)
	protegido("GET /api/v1/importacoes/{id}/historico", importacaoHandler.ListarHistorico)

	financeiroImportacaoHandler := importacaohandlers.NewFinanceiroHandler(financeiroImportacaoUseCase)
	financeiro("POST /api/v1/importacoes/{id}/parcelas", financeiroImportacaoHandler.IncluirParcela)
	financeiro("PUT /api/v1/importacoes/{id}/parcelas/{parcela_id}", financeiroImportacaoHandler.AlterarParcela)
	financeiro("DELETE /api/v1/importacoes/{id}/parcelas/{parcela_id}", financeiroImportacaoHandler.RemoverParcela)
	financeiro("POST /api/v1/importacoes/{id}/parcelas/{parcela_id}/cambio", financeiroImportacaoHandler.FecharCambio)
	financeiro("POST /api/v1/importacoes/{id}/parcelas/{parcela_id}/reabrir", financeiroImportacaoHandler.ReabrirParcela)
	financeiro("POST /api/v1/importacoes/{id}/custos", financeiroImportacaoHandler.LancarCusto)
	financeiro("POST /api/v1/importacoes/{id}/custos/{custo_id}/cancelar", financeiroImportacaoHandler.CancelarCusto)
	financeiro("GET /api/v1/importacoes/{id}/custo-previsto", financeiroImportacaoHandler.CustoPrevisto)

	recebimentoImportacaoHandler := importacaohandlers.NewRecebimentoHandler(recebimentoImportacaoUseCase)
	protegido("PUT /api/v1/importacoes/{id}/alocacoes", recebimentoImportacaoHandler.DefinirAlocacoes)
	financeiro("POST /api/v1/importacoes/{id}/receber", recebimentoImportacaoHandler.Receber)

	consultaImportacaoHandler := importacaohandlers.NewConsultaHandler(consultaImportacaoUseCase)
	protegido("GET /api/v1/importacoes/alertas", consultaImportacaoHandler.Alertas)
	protegido("GET /api/v1/pedidos/{id}/importacoes", consultaImportacaoHandler.ImportacoesDoPedido)

	dividaHandler := dividahandlers.NewDividaHandler(dividaUseCase)
	financeiro("POST /api/v1/dividas", dividaHandler.Criar)
	financeiro("GET /api/v1/dividas", dividaHandler.Listar)
	financeiro("POST /api/v1/dividas/gerar", dividaHandler.GerarContas)
	financeiro("GET /api/v1/dividas/resumo", dividaHandler.Resumo)
	financeiro("GET /api/v1/dividas/{id}", dividaHandler.Obter)
	financeiro("PUT /api/v1/dividas/{id}", dividaHandler.Alterar)
	financeiro("PUT /api/v1/dividas/{id}/parcelas/{parcela_id}", dividaHandler.AlterarParcela)
	financeiro("POST /api/v1/dividas/{id}/cancelar", dividaHandler.Cancelar)
	financeiro("GET /api/v1/dividas/{id}/historico", dividaHandler.ListarHistorico)
	financeiro("POST /api/v1/dividas/{id}/renegociar", dividaHandler.Renegociar)
	financeiro("POST /api/v1/dividas/{id}/parcelas/{parcela_id}/reabrir", dividaHandler.ReabrirParcela)

	resumoDREHandler := drehandlers.NewResumoHandler(resumoDREUseCase)
	financeiro("GET /api/v1/dre/resumo", resumoDREHandler.Consultar)

	faturamentoHandler := drehandlers.NewFaturamentoHandler(faturamentoUseCase)
	financeiro("GET /api/v1/dre/faturamento", faturamentoHandler.Consultar)

	relatorioSemanalHandler := drehandlers.NewRelatorioSemanalHandler(relatorioSemanalUseCase)
	financeiro("GET /api/v1/dre/relatorio-semanal", relatorioSemanalHandler.Consultar)

	importacaoLeadsHandler := lead.NewImportacaoHandler(importacaoLeadsUseCase)
	comercial("POST /api/v1/leads/importacao", middleware.EstenderPrazo(prazoDeArquivo, importacaoLeadsHandler.Importar))
	comercial("GET /api/v1/leads/importacao/modelo", importacaoLeadsHandler.Modelo)

	importacaoProdutosHandler := estoque.NewImportacaoProdutoHandler(importacaoProdutosUseCase)
	administrador("POST /api/v1/produtos/importacao", middleware.EstenderPrazo(prazoDeArquivo, importacaoProdutosHandler.Importar))
	administrador("GET /api/v1/produtos/importacao/modelo", importacaoProdutosHandler.Modelo)

	importacaoTabelaPrecoHandler := tabelaprecohandlers.NewImportacaoHandler(importacaoTabelaPrecoUseCase)
	administrador("POST /api/v1/tabela-precos/importacao", middleware.EstenderPrazo(prazoDeArquivo, importacaoTabelaPrecoHandler.Importar))
	administrador("GET /api/v1/tabela-precos/importacao/modelo", importacaoTabelaPrecoHandler.Modelo)

	return mux
}
