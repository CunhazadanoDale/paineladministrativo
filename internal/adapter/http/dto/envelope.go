package dto

import (
	"encoding/json"
	"net/http"
)

type Resposta[T any] struct {
	Dados T `json:"dados"`
}

type Paginado[T any] struct {
	Dados   []T `json:"dados"`
	Pagina  int `json:"pagina"`
	Tamanho int `json:"tamanho"`
}

type ErroInterno struct {
	Codigo   int    `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

type RespostaErro struct {
	Erro ErroInterno `json:"erro"`
}

func EscreverJSON(w http.ResponseWriter, status int, conteudo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if conteudo == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(conteudo)
}

func EscreverVazio(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

func EscreverErro(w http.ResponseWriter, status int, mensagem string) {
	EscreverJSON(w, status, RespostaErro{
		Erro: ErroInterno{Codigo: status, Mensagem: mensagem},
	})
}
