package resposta

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const tamanhoMaximoCSV = 5 << 20

func CorpoCSV(w http.ResponseWriter, r *http.Request, cabecalho []string) ([]dto.LinhaCSV, bool) {
	defer r.Body.Close()

	linhas, err := dto.LerCSV(http.MaxBytesReader(w, r.Body, tamanhoMaximoCSV), cabecalho, domain.LimiteDeLinhasDaImportacao)
	if err != nil {
		var erroTamanho *http.MaxBytesError
		if errors.As(err, &erroTamanho) {
			dto.EscreverErro(w, http.StatusRequestEntityTooLarge, "arquivo excede o limite de 5MB")
			return nil, false
		}
		ResponderErro(w, err)
		return nil, false
	}

	return linhas, true
}

func ModeloCSV(w http.ResponseWriter, nomeArquivo string, cabecalho, exemplo []string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(nomeArquivo)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(dto.ModeloCSV(cabecalho, exemplo))); err != nil {
		middleware.AnotarErro(w, fmt.Errorf("escrever modelo csv: %w", err))
	}
}
