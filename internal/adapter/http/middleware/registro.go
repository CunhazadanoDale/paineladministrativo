package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/google/uuid"
)

const CabecalhoRequisicao = "X-Request-Id"

type anotadorDeErro interface {
	AnotarErro(err error)
}

type respostaRegistrada struct {
	http.ResponseWriter
	status int
	erro   error
}

func (r *respostaRegistrada) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}

	r.ResponseWriter.WriteHeader(status)
}

func (r *respostaRegistrada) Write(conteudo []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	return r.ResponseWriter.Write(conteudo)
}

func (r *respostaRegistrada) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *respostaRegistrada) AnotarErro(err error) {
	r.erro = err
}

func AnotarErro(w http.ResponseWriter, err error) {
	for {
		if anotador, ok := w.(anotadorDeErro); ok {
			anotador.AnotarErro(err)
			return
		}

		embrulhado, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}

		w = embrulhado.Unwrap()
	}
}

func Registrar(registrador *slog.Logger, proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		requisicaoID := uuid.NewString()
		w.Header().Set(CabecalhoRequisicao, requisicaoID)

		resposta := &respostaRegistrada{ResponseWriter: w}

		defer func() {
			if recuperado := recover(); recuperado != nil {
				if recuperado == http.ErrAbortHandler {
					panic(recuperado)
				}

				resposta.AnotarErro(fmt.Errorf("panic: %v", recuperado))
				if resposta.status == 0 {
					dto.EscreverErro(resposta, http.StatusInternalServerError, "erro interno do servidor")
				}
			}

			registrarRequisicao(registrador, r, resposta, requisicaoID, time.Since(inicio))
		}()

		proximo.ServeHTTP(resposta, r)
	})
}

func registrarRequisicao(registrador *slog.Logger, r *http.Request, resposta *respostaRegistrada, requisicaoID string, duracao time.Duration) {
	status := resposta.status
	if status == 0 {
		status = http.StatusOK
	}

	atributos := []slog.Attr{
		slog.String("requisicao_id", requisicaoID),
		slog.String("metodo", r.Method),
		slog.String("caminho", r.URL.Path),
		slog.Int("status", status),
		slog.Int64("duracao_ms", duracao.Milliseconds()),
	}

	causa := resposta.erro
	nivel := slog.LevelInfo
	switch {
	case status >= http.StatusInternalServerError:
		nivel = slog.LevelError
		if causa == nil {
			causa = errors.New("causa não informada")
		}
	case causa != nil:
		nivel = slog.LevelWarn
	}
	if causa != nil {
		atributos = append(atributos, slog.String("erro", causa.Error()))
	}

	registrador.LogAttrs(r.Context(), nivel, "requisição atendida", atributos...)
}
