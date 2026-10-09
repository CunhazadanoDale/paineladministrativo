package estoque

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type imagemUseCaseFalso struct {
	erro        error
	imagem      *domainestoque.Imagem
	inputAnexar portsin.AnexarImagemInput
	removido    struct {
		usuarioID uuid.UUID
		produtoID uuid.UUID
		imagemID  uuid.UUID
	}
	publica *domainestoque.Imagem
}

func (f *imagemUseCaseFalso) Anexar(_ context.Context, input portsin.AnexarImagemInput) (*domainestoque.Imagem, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	f.inputAnexar = input

	return f.imagem, nil
}

func (f *imagemUseCaseFalso) Remover(_ context.Context, usuarioID, produtoID, imagemID uuid.UUID) error {
	if f.erro != nil {
		return f.erro
	}

	f.removido.usuarioID = usuarioID
	f.removido.produtoID = produtoID
	f.removido.imagemID = imagemID

	return nil
}

func (f *imagemUseCaseFalso) ObterPublica(_ context.Context, imagemID uuid.UUID) (*domainestoque.Imagem, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	if f.publica == nil {
		return nil, domain.ErroNaoEncontrado("imagem não encontrada")
	}

	return f.publica, nil
}

var _ portsin.ImagemUseCase = (*imagemUseCaseFalso)(nil)

func TestAnexarImagemRespondeCreatedERepassaInput(t *testing.T) {
	produtoID := uuid.New()
	arquivoID := uuid.New()
	usecase := &imagemUseCaseFalso{imagem: &domainestoque.Imagem{
		ID:        uuid.New(),
		ProdutoID: produtoID,
		ArquivoID: arquivoID,
		Ordem:     2,
		Alt:       "Saco de cimento",
	}}

	resposta := executaProtegido(t, NewImagemHandler(usecase).Anexar, http.MethodPost,
		"/api/v1/produtos/"+produtoID.String()+"/imagens", produtoID.String(),
		`{"arquivo_id":"`+arquivoID.String()+`","ordem":2,"alt":"Saco de cimento"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}

	if usecase.inputAnexar.ProdutoID != produtoID || usecase.inputAnexar.ArquivoID != arquivoID {
		t.Errorf("input = %+v, esperado produto e arquivo da requisição", usecase.inputAnexar)
	}
	if usecase.inputAnexar.Ordem != 2 || usecase.inputAnexar.Alt != "Saco de cimento" {
		t.Errorf("input = %+v, esperado ordem e texto alternativo da requisição", usecase.inputAnexar)
	}
	if usecase.inputAnexar.UsuarioID == uuid.Nil {
		t.Error("input sem usuário autenticado")
	}

	var corpo struct {
		Dados struct {
			ID        uuid.UUID `json:"id"`
			ProdutoID uuid.UUID `json:"produto_id"`
			ArquivoID uuid.UUID `json:"arquivo_id"`
			Ordem     int       `json:"ordem"`
			Alt       string    `json:"alt"`
		} `json:"dados"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("não decodifiquei a resposta: %v", err)
	}
	if corpo.Dados.ID != usecase.imagem.ID || corpo.Dados.Ordem != 2 {
		t.Errorf("corpo = %+v, esperado a imagem criada", corpo.Dados)
	}
	if corpo.Dados.ProdutoID != produtoID || corpo.Dados.ArquivoID != arquivoID {
		t.Errorf("corpo = %+v, esperado produto e arquivo", corpo.Dados)
	}
}

func TestAnexarImagemComErroPropagaStatus(t *testing.T) {
	produtoID := uuid.New()
	usecase := &imagemUseCaseFalso{erro: domain.ErroConflito("arquivo já anexado a este produto")}

	resposta := executaProtegido(t, NewImagemHandler(usecase).Anexar, http.MethodPost,
		"/api/v1/produtos/"+produtoID.String()+"/imagens", produtoID.String(),
		`{"arquivo_id":"`+uuid.NewString()+`"}`)

	if resposta.Code != http.StatusConflict {
		t.Errorf("status %d, esperado %d", resposta.Code, http.StatusConflict)
	}
}

func TestRemoverImagemResponde204ERepassaParametros(t *testing.T) {
	produtoID := uuid.New()
	imagemID := uuid.New()
	usecase := &imagemUseCaseFalso{}

	resposta := executaComParametros(t, NewImagemHandler(usecase).Remover, http.MethodDelete,
		"/api/v1/produtos/"+produtoID.String()+"/imagens/"+imagemID.String(),
		map[string]string{"id": produtoID.String(), "imagemId": imagemID.String()},
		"", true)

	if resposta.Code != http.StatusNoContent {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusNoContent, resposta.Body.String())
	}
	if usecase.removido.produtoID != produtoID || usecase.removido.imagemID != imagemID {
		t.Errorf("remoção = %+v, esperado produto e imagem da rota", usecase.removido)
	}
	if usecase.removido.usuarioID == uuid.Nil {
		t.Error("remoção sem usuário autenticado")
	}
}

func TestRemoverImagemComIdentificadorInvalidoResponde400(t *testing.T) {
	usecase := &imagemUseCaseFalso{}

	resposta := executaComParametros(t, NewImagemHandler(usecase).Remover, http.MethodDelete,
		"/api/v1/produtos/nao-eh-uuid/imagens/tambem-nao",
		map[string]string{"id": "nao-eh-uuid", "imagemId": "tambem-nao"},
		"", true)

	if resposta.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", resposta.Code, http.StatusBadRequest)
	}
}
