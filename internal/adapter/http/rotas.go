package http

import (
	"net/http"
	"time"

	clientehandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/cliente"
	empresahandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/empresa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/estoque"
	financeirohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/financeiro"
	fornecedorhandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/fornecedor"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	pedidohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/pedido"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	solicitacaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsinempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	portsinestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsinfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	portsinfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	portsinlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsinpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsinsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
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
	pedidoUseCase portsinpedido.PedidoUseCase,
	entregaUseCase portsinpedido.EntregaUseCase,
	consultaPedidosUseCase portsinpedido.ConsultaUseCase,
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
		pedidoUseCase, entregaUseCase, consultaPedidosUseCase,
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
	pedidoUseCase portsinpedido.PedidoUseCase,
	entregaUseCase portsinpedido.EntregaUseCase,
	consultaPedidosUseCase portsinpedido.ConsultaUseCase,
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

	arquivoHandler := solicitacaohandlers.NewArquivoHandler(arquivoUseCase)
	protegido("POST /api/v1/arquivos", middleware.EstenderPrazo(prazoDeArquivo, arquivoHandler.Enviar))
	protegido("GET /api/v1/arquivos", arquivoHandler.Listar)
	protegido("GET /api/v1/arquivos/{id}", middleware.EstenderPrazo(prazoDeArquivo, arquivoHandler.Baixar))
	protegido("DELETE /api/v1/arquivos/{id}", arquivoHandler.Remover)

	aprovadorHandler := solicitacaohandlers.NewAprovadorHandler(aprovadorUseCase)
	administrador("POST /api/v1/aprovadores", aprovadorHandler.Designar)
	administrador("GET /api/v1/aprovadores", aprovadorHandler.Listar)
	administrador("DELETE /api/v1/aprovadores/{id}", aprovadorHandler.Remover)

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

	return mux
}
