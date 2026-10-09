//go:build e2e

package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func criarCategoria(t *testing.T, servidor *httptest.Server, nome string, categoriaPaiID *uuid.UUID) uuid.UUID {
	t.Helper()

	corpo := map[string]any{"nome": nome}
	if categoriaPaiID != nil {
		corpo["categoria_pai_id"] = *categoriaPaiID
	}

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/categorias", corpo)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[estoquedto.CategoriaResponse](t, resposta).Dados.ID
}

func criarProduto(t *testing.T, servidor *httptest.Server, categoriaID uuid.UUID, nome string) uuid.UUID {
	t.Helper()

	return criarProdutoComCodigo(t, servidor, categoriaID, nome, "SKU-001")
}

func criarProdutoComCodigo(t *testing.T, servidor *httptest.Server, categoriaID uuid.UUID, nome, codigo string) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/produtos", map[string]any{
		"categoria_id":   categoriaID,
		"nome":           nome,
		"descricao":      "Produto criado no teste",
		"codigo":         codigo,
		"unidade_medida": "un",
		"preco_centavos": 2590,
		"estoque_minimo": 5,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[estoquedto.ProdutoResponse](t, resposta).Dados.ID
}

func contemProduto(pagina dto.Paginado[estoquedto.ProdutoResponse], id uuid.UUID) bool {
	for _, produto := range pagina.Dados {
		if produto.ID == id {
			return true
		}
	}

	return false
}

func contemProdutoPublico(pagina dto.Paginado[estoquedto.ProdutoPublicoResponse], id uuid.UUID) bool {
	return produtoPublico(pagina, id).ID != uuid.Nil
}

func produtoPublico(pagina dto.Paginado[estoquedto.ProdutoPublicoResponse], id uuid.UUID) estoquedto.ProdutoPublicoResponse {
	for _, produto := range pagina.Dados {
		if produto.ID == id {
			return produto
		}
	}

	return estoquedto.ProdutoPublicoResponse{}
}

func contemTipo(pagina dto.Paginado[estoquedto.MovimentoResponse], tipo string) bool {
	for _, movimento := range pagina.Dados {
		if movimento.Tipo == tipo {
			return true
		}
	}

	return false
}

func TestFluxoCompletoDeEstoque(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	raizID := criarCategoria(t, servidor, "Alvenaria", nil)
	subID := criarCategoria(t, servidor, "Cimento", &raizID)

	respostaRaiz := envia(t, servidor, http.MethodGet, "/api/v1/categorias/"+raizID.String(), nil)
	conferirStatus(t, respostaRaiz, http.StatusOK)
	raiz := decodificarEnvelope[estoquedto.CategoriaResponse](t, respostaRaiz).Dados
	if raiz.Slug != "alvenaria" || raiz.CategoriaPaiID != nil {
		t.Errorf("categoria raiz = %+v, esperado slug alvenaria sem pai", raiz)
	}

	respostaCategoriaRepetida := envia(t, servidor, http.MethodPost, "/api/v1/categorias", map[string]any{
		"nome": "Alvenaria",
	})
	conferirStatus(t, respostaCategoriaRepetida, http.StatusConflict)

	respostaDesativarRaiz := envia(t, servidor, http.MethodPatch,
		"/api/v1/categorias/"+raizID.String()+"/desativar", nil)
	conferirStatus(t, respostaDesativarRaiz, http.StatusConflict)

	respostaCategoriaNaRaiz := envia(t, servidor, http.MethodPost, "/api/v1/produtos", map[string]any{
		"categoria_id":   raizID,
		"nome":           "Areia grossa",
		"unidade_medida": "m3",
	})
	conferirStatus(t, respostaCategoriaNaRaiz, http.StatusBadRequest)

	produtoID := criarProduto(t, servidor, subID, "Cimento CP II 50")

	respostaProdutoRepetido := envia(t, servidor, http.MethodPost, "/api/v1/produtos", map[string]any{
		"categoria_id":   subID,
		"nome":           "Cimento CP II 50",
		"unidade_medida": "un",
	})
	conferirStatus(t, respostaProdutoRepetido, http.StatusConflict)

	caminho := "/api/v1/produtos/" + produtoID.String()

	respostaDetalhe := envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, respostaDetalhe, http.StatusOK)
	produto := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaDetalhe).Dados
	if produto.Slug != "cimento-cp-ii-50" || produto.Saldo != 0 || !produto.Ativo {
		t.Errorf("produto criado = %+v, esperado slug cimento-cp-ii-50 zerado e ativo", produto)
	}
	if !produto.EstoqueBaixo {
		t.Error("produto com saldo 0 e mínimo 5 deveria sair como estoque baixo")
	}

	respostaSemToken := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/produtos", nil)
	conferirStatus(t, respostaSemToken, http.StatusUnauthorized)

	respostaEntrada := envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"tipo":          "entrada",
		"quantidade":    10,
		"documento_ref": "NF-100",
		"observacao":    "reposicao do mes",
	})
	conferirStatus(t, respostaEntrada, http.StatusCreated)
	movimento := decodificarEnvelope[estoquedto.MovimentoResponse](t, respostaEntrada).Dados
	if movimento.SaldoApos != 10 || movimento.Quantidade != 10 || movimento.ProdutoID != produtoID {
		t.Errorf("movimento criado = %+v, esperado entrada de 10 com saldo 10", movimento)
	}

	respostaSaidaMaior := envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"tipo":       "saida",
		"quantidade": 100,
	})
	conferirStatus(t, respostaSaidaMaior, http.StatusBadRequest)

	respostaSaida := envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{
		"tipo":       "saida",
		"quantidade": 6,
	})
	conferirStatus(t, respostaSaida, http.StatusCreated)

	respostaSaldo := envia(t, servidor, http.MethodGet, caminho+"/saldo", nil)
	conferirStatus(t, respostaSaldo, http.StatusOK)
	saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, respostaSaldo).Dados
	if saldo.Saldo != 4 || saldo.ProdutoID != produtoID {
		t.Errorf("saldo = %+v, esperado 4 para o produto %s", saldo, produtoID)
	}

	respostaMovimentos := envia(t, servidor, http.MethodGet, caminho+"/movimentos", nil)
	conferirStatus(t, respostaMovimentos, http.StatusOK)
	historico := decodificarPagina[estoquedto.MovimentoResponse](t, respostaMovimentos)
	if len(historico.Dados) != 2 {
		t.Fatalf("histórico com %d movimentos, esperado 2", len(historico.Dados))
	}
	if !contemTipo(historico, "entrada") || !contemTipo(historico, "saida") {
		t.Errorf("histórico = %+v, esperado entrada e saida", historico.Dados)
	}

	respostaEstoqueBaixo := envia(t, servidor, http.MethodGet, "/api/v1/produtos?estoque_baixo=true", nil)
	conferirStatus(t, respostaEstoqueBaixo, http.StatusOK)
	if pagina := decodificarPagina[estoquedto.ProdutoResponse](t, respostaEstoqueBaixo); !contemProduto(pagina, produtoID) {
		t.Error("produto com saldo 4 e mínimo 5 não apareceu no filtro estoque_baixo")
	}

	respostaBusca := envia(t, servidor, http.MethodGet, "/api/v1/produtos?busca=SKU-001", nil)
	conferirStatus(t, respostaBusca, http.StatusOK)
	if pagina := decodificarPagina[estoquedto.ProdutoResponse](t, respostaBusca); !contemProduto(pagina, produtoID) {
		t.Error("produto não apareceu na busca por código")
	}

	respostaDestaque := envia(t, servidor, http.MethodPatch, caminho+"/destaque", map[string]any{
		"destaque": true,
	})
	conferirStatus(t, respostaDestaque, http.StatusOK)
	if destacado := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaDestaque).Dados; !destacado.Destaque {
		t.Error("produto não saiu destacado após o alternar")
	}

	respostaDestaques := envia(t, servidor, http.MethodGet, "/api/v1/produtos?destaque=true", nil)
	conferirStatus(t, respostaDestaques, http.StatusOK)
	if pagina := decodificarPagina[estoquedto.ProdutoResponse](t, respostaDestaques); !contemProduto(pagina, produtoID) {
		t.Error("produto não apareceu no filtro destaque")
	}

	respostaDesativar := envia(t, servidor, http.MethodPatch, caminho+"/desativar", nil)
	conferirStatus(t, respostaDesativar, http.StatusOK)
	if inativo := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaDesativar).Dados; inativo.Ativo {
		t.Error("produto seguiu ativo após o desativar")
	}

	respostaInativos := envia(t, servidor, http.MethodGet, "/api/v1/produtos?ativo=false", nil)
	conferirStatus(t, respostaInativos, http.StatusOK)
	if pagina := decodificarPagina[estoquedto.ProdutoResponse](t, respostaInativos); !contemProduto(pagina, produtoID) {
		t.Error("produto não apareceu no filtro de inativos")
	}

	respostaAtivar := envia(t, servidor, http.MethodPatch, caminho+"/ativar", nil)
	conferirStatus(t, respostaAtivar, http.StatusOK)
	if ativo := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaAtivar).Dados; !ativo.Ativo {
		t.Error("produto seguiu inativo após o ativar")
	}

	respostaAlterar := envia(t, servidor, http.MethodPut, caminho, map[string]any{
		"categoria_id":   subID,
		"nome":           "Cimento CP II 50",
		"descricao":      "Descrição atualizada",
		"unidade_medida": "sc",
		"preco_centavos": 2790,
		"estoque_minimo": 8,
	})
	conferirStatus(t, respostaAlterar, http.StatusOK)
	alterado := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaAlterar).Dados
	if alterado.Slug != "cimento-cp-ii-50" || alterado.Saldo != 4 {
		t.Errorf("produto alterado = %+v, esperado slug e saldo preservados", alterado)
	}
	if alterado.PrecoCentavos == nil || *alterado.PrecoCentavos != 2790 {
		t.Errorf("preço após edição = %v, esperado 2790", alterado.PrecoCentavos)
	}

	respostaCategorias := envia(t, servidor, http.MethodGet, "/api/v1/categorias?ativo=true", nil)
	conferirStatus(t, respostaCategorias, http.StatusOK)
	categorias := decodificarPagina[estoquedto.CategoriaResponse](t, respostaCategorias)
	if len(categorias.Dados) != 2 {
		t.Errorf("categorias ativas = %d, esperado 2", len(categorias.Dados))
	}
}

func TestResumoDeEstoqueConsolidaDados(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	raizID := criarCategoria(t, servidor, "Alvenaria", nil)
	subID := criarCategoria(t, servidor, "Cimento", &raizID)
	produtoID := criarProduto(t, servidor, subID, "Cimento CP II 50")

	respostaSemToken := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/estoque/resumo", nil)
	conferirStatus(t, respostaSemToken, http.StatusUnauthorized)

	respostaEntrada := envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
		"tipo":       "entrada",
		"quantidade": 10,
	})
	conferirStatus(t, respostaEntrada, http.StatusCreated)

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/estoque/resumo", nil)
	conferirStatus(t, resposta, http.StatusOK)

	resumo := decodificarEnvelope[estoquedto.ResumoResponse](t, resposta).Dados
	if resumo.TotalProdutos != 1 || resumo.ProdutosAtivos != 1 {
		t.Errorf("totais = %d/%d, esperado 1 produto ativo", resumo.TotalProdutos, resumo.ProdutosAtivos)
	}
	if resumo.ValorEstoqueCentavos != 25900 {
		t.Errorf("valor em estoque = %d, esperado 25900 centavos", resumo.ValorEstoqueCentavos)
	}
	if resumo.ProdutosEstoqueBaixo != 0 {
		t.Errorf("produtos com estoque baixo = %d, esperado 0", resumo.ProdutosEstoqueBaixo)
	}
	if len(resumo.UltimosMovimentos) != 1 {
		t.Fatalf("últimos movimentos = %d, esperado 1", len(resumo.UltimosMovimentos))
	}

	movimento := resumo.UltimosMovimentos[0]
	if movimento.ProdutoID != produtoID || movimento.ProdutoNome != "Cimento CP II 50" {
		t.Errorf("movimento = %+v, esperado o produto criado com o nome", movimento)
	}
	if movimento.Tipo != "entrada" || movimento.Quantidade != 10 || movimento.SaldoApos != 10 {
		t.Errorf("movimento = %+v, esperado entrada de 10 com saldo 10", movimento)
	}
}

func TestEscritaDeEstoqueExigeAdministrador(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	raizID := criarCategoria(t, servidor, "Ferragens", nil)
	subID := criarCategoria(t, servidor, "Parafusos", &raizID)
	produtoID := criarProduto(t, servidor, subID, "Parafuso sextavado")

	cargoID := criarCargo(t, servidor, "Operador de estoque", false)
	criarUsuario(t, servidor, "Dina Lopes", "dina.estoque@exemplo.com", "senhaForte123", cargoID)
	tokenOperador := autenticar(t, servidor, "dina.estoque@exemplo.com", "senhaForte123").Token

	respostaLeitura := enviaComToken(t, servidor, http.MethodGet, "/api/v1/produtos", nil, tokenOperador)
	conferirStatus(t, respostaLeitura, http.StatusOK)

	respostaCriarProduto := enviaComToken(t, servidor, http.MethodPost, "/api/v1/produtos", map[string]any{
		"categoria_id":   subID,
		"nome":           "Produto sem permissão",
		"unidade_medida": "un",
	}, tokenOperador)
	conferirStatus(t, respostaCriarProduto, http.StatusForbidden)

	respostaCriarCategoria := enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias", map[string]any{
		"nome": "Categoria sem permissão",
	}, tokenOperador)
	conferirStatus(t, respostaCriarCategoria, http.StatusForbidden)

	respostaMovimentar := enviaComToken(t, servidor, http.MethodPost,
		"/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
			"tipo":       "entrada",
			"quantidade": 5,
		}, tokenOperador)
	conferirStatus(t, respostaMovimentar, http.StatusForbidden)

	respostaDesativar := enviaComToken(t, servidor, http.MethodPatch,
		"/api/v1/categorias/"+raizID.String()+"/desativar", nil, tokenOperador)
	conferirStatus(t, respostaDesativar, http.StatusForbidden)
}

func TestImagensDeProdutoNoPainel(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	raizID := criarCategoria(t, servidor, "Alvenaria", nil)
	subID := criarCategoria(t, servidor, "Cimento", &raizID)
	produtoID := criarProduto(t, servidor, subID, "Cimento CP II 50")
	produtoSecundario := criarProdutoComCodigo(t, servidor, subID, "Areia grossa", "SKU-002")
	caminho := "/api/v1/produtos/" + produtoID.String()

	cargoID := criarCargo(t, servidor, "Operador de estoque", false)
	criarUsuario(t, servidor, "Dina Lopes", "dina.estoque@exemplo.com", "senhaForte123", cargoID)
	tokenOperador := autenticar(t, servidor, "dina.estoque@exemplo.com", "senhaForte123").Token

	conteudoPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	respostaUpload := enviaMultipart(t, servidor, "/api/v1/arquivos", "capa.png", conteudoPNG, tokenDoTeste)
	conferirStatus(t, respostaUpload, http.StatusCreated)
	arquivoCapa := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaUpload).Dados

	respostaUploadVerso := enviaMultipart(t, servidor, "/api/v1/arquivos", "verso.png", conteudoPNG, tokenDoTeste)
	conferirStatus(t, respostaUploadVerso, http.StatusCreated)
	arquivoVerso := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaUploadVerso).Dados

	respostaUploadPdf := enviaMultipart(t, servidor, "/api/v1/arquivos", "ficha.pdf",
		[]byte("%PDF-1.4 conteudo"), tokenDoTeste)
	conferirStatus(t, respostaUploadPdf, http.StatusCreated)
	arquivoPdf := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaUploadPdf).Dados

	respostaAnexar := envia(t, servidor, http.MethodPost, caminho+"/imagens", map[string]any{
		"arquivo_id": arquivoCapa.ID,
		"ordem":      1,
		"alt":        "Saco de cimento",
	})
	conferirStatus(t, respostaAnexar, http.StatusCreated)
	imagem := decodificarEnvelope[estoquedto.ImagemResponse](t, respostaAnexar).Dados
	if imagem.ProdutoID != produtoID || imagem.ArquivoID != arquivoCapa.ID || imagem.Ordem != 1 {
		t.Errorf("imagem = %+v, esperado produto, arquivo e ordem da requisição", imagem)
	}
	if imagem.ID == uuid.Nil || imagem.Alt != "Saco de cimento" {
		t.Errorf("imagem = %+v, esperado identificador gerado e texto alternativo", imagem)
	}

	respostaRepetida := envia(t, servidor, http.MethodPost, caminho+"/imagens", map[string]any{
		"arquivo_id": arquivoCapa.ID,
	})
	conferirStatus(t, respostaRepetida, http.StatusConflict)

	respostaProdutoInexistente := envia(t, servidor, http.MethodPost,
		"/api/v1/produtos/"+uuid.NewString()+"/imagens", map[string]any{
			"arquivo_id": arquivoCapa.ID,
		})
	conferirStatus(t, respostaProdutoInexistente, http.StatusNotFound)

	respostaPdf := envia(t, servidor, http.MethodPost, caminho+"/imagens", map[string]any{
		"arquivo_id": arquivoPdf.ID,
	})
	conferirStatus(t, respostaPdf, http.StatusBadRequest)

	respostaOperador := enviaComToken(t, servidor, http.MethodPost, caminho+"/imagens", map[string]any{
		"arquivo_id": arquivoVerso.ID,
	}, tokenOperador)
	conferirStatus(t, respostaOperador, http.StatusForbidden)

	respostaVerso := envia(t, servidor, http.MethodPost, caminho+"/imagens", map[string]any{
		"arquivo_id": arquivoVerso.ID,
		"ordem":      0,
		"alt":        "Verso do saco",
	})
	conferirStatus(t, respostaVerso, http.StatusCreated)
	segundaImagem := decodificarEnvelope[estoquedto.ImagemResponse](t, respostaVerso).Dados

	respostaDetalhe := envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, respostaDetalhe, http.StatusOK)
	detalhe := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaDetalhe).Dados
	if len(detalhe.Imagens) != 2 {
		t.Fatalf("imagens do produto = %d, esperado 2", len(detalhe.Imagens))
	}
	if detalhe.Imagens[0].ID != segundaImagem.ID || detalhe.Imagens[1].ID != imagem.ID {
		t.Errorf("ordem das imagens = [%s %s], esperado pela ordem declarada",
			detalhe.Imagens[0].ID, detalhe.Imagens[1].ID)
	}

	respostaOutroProduto := envia(t, servidor, http.MethodDelete,
		"/api/v1/produtos/"+produtoSecundario.String()+"/imagens/"+imagem.ID.String(), nil)
	conferirStatus(t, respostaOutroProduto, http.StatusNotFound)

	respostaRemover := envia(t, servidor, http.MethodDelete,
		caminho+"/imagens/"+imagem.ID.String(), nil)
	conferirStatus(t, respostaRemover, http.StatusNoContent)
	if corpo := lerCorpo(t, respostaRemover); len(corpo) != 0 {
		t.Errorf("corpo da remoção = %q, esperado vazio", corpo)
	}

	respostaRemoverDeNovo := envia(t, servidor, http.MethodDelete,
		caminho+"/imagens/"+imagem.ID.String(), nil)
	conferirStatus(t, respostaRemoverDeNovo, http.StatusNotFound)

	respostaAposRemover := envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, respostaAposRemover, http.StatusOK)
	depois := decodificarEnvelope[estoquedto.ProdutoResponse](t, respostaAposRemover).Dados
	if len(depois.Imagens) != 1 || depois.Imagens[0].ID != segundaImagem.ID {
		t.Errorf("imagens restantes = %+v, esperado só a segunda", depois.Imagens)
	}
}

func TestVitrinePublicaDeEstoque(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	raizID := criarCategoria(t, servidor, "Materiais", nil)
	subID := criarCategoria(t, servidor, "Alvenaria", &raizID)
	comEstoqueID := criarProduto(t, servidor, subID, "Cimento CP II 50")
	semEstoqueID := criarProdutoComCodigo(t, servidor, subID, "Areia grossa", "SKU-002")
	inativoID := criarProdutoComCodigo(t, servidor, subID, "Tijolo 8 furos", "SKU-003")
	outroRaizID := criarCategoria(t, servidor, "Hidraulica", nil)
	outroSubID := criarCategoria(t, servidor, "Tubos", &outroRaizID)
	foraDaArvoreID := criarProdutoComCodigo(t, servidor, outroSubID, "Tubo PVC 100mm", "SKU-004")

	for _, produtoID := range []uuid.UUID{comEstoqueID, inativoID, foraDaArvoreID} {
		respostaEntrada := envia(t, servidor, http.MethodPost,
			"/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
				"tipo":       "entrada",
				"quantidade": 6,
			})
		conferirStatus(t, respostaEntrada, http.StatusCreated)
	}

	respostaDesativar := envia(t, servidor, http.MethodPatch,
		"/api/v1/produtos/"+inativoID.String()+"/desativar", nil)
	conferirStatus(t, respostaDesativar, http.StatusOK)

	for _, produtoID := range []uuid.UUID{comEstoqueID, inativoID} {
		respostaDestaque := envia(t, servidor, http.MethodPatch,
			"/api/v1/produtos/"+produtoID.String()+"/destaque", map[string]any{
				"destaque": true,
			})
		conferirStatus(t, respostaDestaque, http.StatusOK)
	}

	conteudoPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	respostaUpload := enviaMultipart(t, servidor, "/api/v1/arquivos", "capa.png", conteudoPNG, tokenDoTeste)
	conferirStatus(t, respostaUpload, http.StatusCreated)
	arquivoCapa := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaUpload).Dados

	respostaAnexar := envia(t, servidor, http.MethodPost,
		"/api/v1/produtos/"+comEstoqueID.String()+"/imagens", map[string]any{
			"arquivo_id": arquivoCapa.ID,
			"ordem":      0,
			"alt":        "Saco de cimento",
		})
	conferirStatus(t, respostaAnexar, http.StatusCreated)
	imagemID := decodificarEnvelope[estoquedto.ImagemResponse](t, respostaAnexar).Dados.ID

	respostaSemToken := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/publico/categorias", nil)
	conferirStatus(t, respostaSemToken, http.StatusOK)
	categorias := decodificarEnvelope[[]estoquedto.CategoriaPublicaResponse](t, respostaSemToken).Dados
	if len(categorias) != 2 {
		t.Fatalf("categorias raiz públicas = %d, esperado 2", len(categorias))
	}
	for _, categoria := range categorias {
		if categoria.Slug == "materiais" {
			if len(categoria.Filhas) != 1 || categoria.Filhas[0].Slug != "alvenaria" {
				t.Errorf("filhas de materiais = %+v, esperado alvenaria", categoria.Filhas)
			}
		}
	}

	respostaProdutos := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/publico/produtos", nil)
	conferirStatus(t, respostaProdutos, http.StatusOK)
	pagina := decodificarPagina[estoquedto.ProdutoPublicoResponse](t, respostaProdutos)
	if len(pagina.Dados) != 2 {
		t.Fatalf("produtos públicos = %d, esperado só os ativos com estoque", len(pagina.Dados))
	}
	for _, oculto := range []uuid.UUID{semEstoqueID, inativoID} {
		if contemProdutoPublico(pagina, oculto) {
			t.Errorf("produto %s não deveria aparecer público", oculto)
		}
	}
	if !contemProdutoPublico(pagina, comEstoqueID) {
		t.Fatal("produto com estoque não apareceu na vitrine")
	}
	publicado := produtoPublico(pagina, comEstoqueID)
	if publicado.Slug != "cimento-cp-ii-50" {
		t.Errorf("produto = %+v, esperado o cimento com estoque", publicado)
	}
	if publicado.PrecoCentavos == nil || *publicado.PrecoCentavos != 2590 {
		t.Errorf("preço público = %v, esperado 2590 centavos", publicado.PrecoCentavos)
	}
	if len(publicado.Imagens) != 1 || publicado.Imagens[0].Alt != "Saco de cimento" {
		t.Errorf("imagens públicas = %+v, esperado a imagem anexada", publicado.Imagens)
	}

	respostaRaiz := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos?categoria=materiais", nil)
	conferirStatus(t, respostaRaiz, http.StatusOK)
	if paginaRaiz := decodificarPagina[estoquedto.ProdutoPublicoResponse](t, respostaRaiz); len(paginaRaiz.Dados) != 1 {
		t.Errorf("produtos da raiz = %d, esperado 1 (expansão das subcategorias)", len(paginaRaiz.Dados))
	}

	respostaCategoriaDesconhecida := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos?categoria=que-nao-existe", nil)
	conferirStatus(t, respostaCategoriaDesconhecida, http.StatusNotFound)

	respostaDetalhe := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos/cimento-cp-ii-50", nil)
	conferirStatus(t, respostaDetalhe, http.StatusOK)

	var bruto map[string]any
	decodificar(t, respostaDetalhe, &bruto)
	dados, ok := bruto["dados"].(map[string]any)
	if !ok {
		t.Fatal("detalhe público fora do envelope esperado")
	}
	for _, campo := range []string{"saldo", "codigo", "ativo", "estoque_minimo", "criado_em"} {
		if _, exposto := dados[campo]; exposto {
			t.Errorf("campo interno %q exposto na vitrine pública", campo)
		}
	}
	if _, existe := dados["imagens"]; !existe {
		t.Error("detalhe público sem a lista de imagens")
	}

	respostaSlugDesconhecido := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos/slug-que-nao-existe", nil)
	conferirStatus(t, respostaSlugDesconhecido, http.StatusNotFound)

	respostaSemEstoque := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos/areia-grossa", nil)
	conferirStatus(t, respostaSemEstoque, http.StatusNotFound)

	respostaInativo := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/produtos/tijolo-8-furos", nil)
	conferirStatus(t, respostaInativo, http.StatusNotFound)

	respostaDestaques := enviaSemToken(t, servidor, http.MethodGet, "/api/v1/publico/destaques", nil)
	conferirStatus(t, respostaDestaques, http.StatusOK)
	destaques := decodificarPagina[estoquedto.ProdutoPublicoResponse](t, respostaDestaques)
	if len(destaques.Dados) != 1 || destaques.Dados[0].ID != comEstoqueID {
		t.Errorf("destaques públicos = %+v, esperado só o cimento com estoque", destaques.Dados)
	}

	respostaImagem := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/imagens/"+imagemID.String(), nil)
	conferirStatus(t, respostaImagem, http.StatusOK)
	if contentType := respostaImagem.Header.Get("Content-Type"); contentType != "image/png" {
		t.Errorf("content-type = %q, esperado image/png", contentType)
	}
	if corpo := lerCorpo(t, respostaImagem); !bytes.Equal(corpo, conteudoPNG) {
		t.Errorf("corpo da imagem = %v, esperado os bytes enviados no upload", corpo)
	}

	respostaImagemInexistente := enviaSemToken(t, servidor, http.MethodGet,
		"/api/v1/publico/imagens/"+uuid.NewString(), nil)
	conferirStatus(t, respostaImagemInexistente, http.StatusNotFound)
}

func lerCorpo(t *testing.T, resposta *http.Response) []byte {
	t.Helper()

	conteudo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatalf("não li o corpo da resposta: %v", err)
	}

	return conteudo
}
