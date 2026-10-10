package http

import (
	"net/http"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/saude"
	solicitacaohandlers "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsinlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
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
	historicoUseCase portsinlead.LeadHistoryUseCase,
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
	tokens portsinautenticacao.TokenService,
) http.Handler {
	return middleware.CORS(origensCORS, novasRotas(
		banco,
		leadUseCase, funilUseCase, etapaUseCase, historicoUseCase,
		usuarioUseCase, cargoUseCase,
		solicitacaoUseCase, arquivoUseCase, aprovadorUseCase,
		categoriaUseCase, produtoUseCase, movimentoUseCase, resumoUseCase,
		imagemUseCase, publicoUseCase,
		tokens,
	))
}

func novasRotas(
	banco *sqlx.DB,
	leadUseCase portsinlead.LeadUseCase,
	funilUseCase portsinlead.FunilUseCase,
	etapaUseCase portsinlead.EtapaUseCase,
	historicoUseCase portsinlead.LeadHistoryUseCase,
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

	historicoHandler := lead.NewLeadHistoryHandler(historicoUseCase)
	comercial("GET /api/v1/leads/{lead_id}/historico", historicoHandler.Listar)

	cargoHandler := usuarios.NewCargoHandler(cargoUseCase)
	administrador("POST /api/v1/cargos", cargoHandler.Criar)
	protegido("GET /api/v1/cargos", cargoHandler.Listar)
	protegido("GET /api/v1/cargos/busca", cargoHandler.ObterPorNome)
	protegido("GET /api/v1/cargos/{id}", cargoHandler.Obter)
	administrador("PUT /api/v1/cargos/{id}", cargoHandler.Atualizar)
	administrador("DELETE /api/v1/cargos/{id}", cargoHandler.Remover)

	usuarioHandler := usuarios.NewUsuarioHandler(usuarioUseCase, tokens)
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
	protegido("GET /api/v1/produtos/{id}/saldo", produtoHandler.Saldo)
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

	mux.HandleFunc("GET /", saude.NaoEncontrado)

	return mux
}
