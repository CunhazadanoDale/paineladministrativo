package caixa

import (
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

const (
	tipoCredito = "credito"
	tipoDebito  = "debito"
)

type ImportarExtratoRequest struct {
	ArquivoID       uuid.UUID  `json:"arquivo_id"`
	ContaBancariaID *uuid.UUID `json:"conta_bancaria_id"`
	Simular         bool       `json:"simular"`
}

func (r ImportarExtratoRequest) ParaInput(usuarioID uuid.UUID) portsin.ImportarExtratoInput {
	return portsin.ImportarExtratoInput{
		UsuarioID:       usuarioID,
		ArquivoID:       r.ArquivoID,
		ContaBancariaID: r.ContaBancariaID,
		Simular:         r.Simular,
	}
}

type LinhaIgnoradaResponse struct {
	Sequencia     int    `json:"sequencia"`
	Data          string `json:"data"`
	ValorCentavos int64  `json:"valor_centavos"`
	Descricao     string `json:"descricao"`
	Motivo        string `json:"motivo"`
}

type RelatorioImportacaoResponse struct {
	Simulacao          bool                    `json:"simulacao"`
	ImportacaoID       *uuid.UUID              `json:"importacao_id"`
	ContaBancariaID    uuid.UUID               `json:"conta_bancaria_id"`
	ArquivoID          uuid.UUID               `json:"arquivo_id"`
	PeriodoInicio      string                  `json:"periodo_inicio"`
	PeriodoFim         string                  `json:"periodo_fim"`
	SaldoBancoCentavos *int64                  `json:"saldo_banco_centavos"`
	SaldoBancoEm       *string                 `json:"saldo_banco_em"`
	Total              int                     `json:"total"`
	Novas              int                     `json:"novas"`
	Repetidas          int                     `json:"repetidas"`
	Ignoradas          []LinhaIgnoradaResponse `json:"ignoradas"`
}

func NovoRelatorioImportacaoResponse(relatorio *portsin.RelatorioImportacao) RelatorioImportacaoResponse {
	importacao := relatorio.Importacao
	resposta := RelatorioImportacaoResponse{
		Simulacao:          relatorio.Simulacao,
		ContaBancariaID:    importacao.ContaBancariaID,
		ArquivoID:          importacao.ArquivoID,
		PeriodoInicio:      formatarData(importacao.PeriodoInicio),
		PeriodoFim:         formatarData(importacao.PeriodoFim),
		SaldoBancoCentavos: importacao.SaldoBancoCentavos,
		SaldoBancoEm:       formatarDataOpcional(importacao.SaldoBancoEm),
		Total:              relatorio.Total,
		Novas:              importacao.LinhasNovas,
		Repetidas:          importacao.LinhasRepetidas,
		Ignoradas:          make([]LinhaIgnoradaResponse, 0, len(relatorio.Ignoradas)),
	}
	if !relatorio.Simulacao {
		id := importacao.ID
		resposta.ImportacaoID = &id
	}
	for _, linha := range relatorio.Ignoradas {
		resposta.Ignoradas = append(resposta.Ignoradas, LinhaIgnoradaResponse{
			Sequencia:     linha.Sequencia,
			Data:          formatarData(linha.Data),
			ValorCentavos: linha.ValorCentavos,
			Descricao:     linha.Descricao,
			Motivo:        linha.Motivo.String(),
		})
	}

	return resposta
}

type ImportacaoResponse struct {
	ID                 uuid.UUID `json:"id"`
	ContaBancariaID    uuid.UUID `json:"conta_bancaria_id"`
	ArquivoID          uuid.UUID `json:"arquivo_id"`
	PeriodoInicio      string    `json:"periodo_inicio"`
	PeriodoFim         string    `json:"periodo_fim"`
	SaldoBancoCentavos *int64    `json:"saldo_banco_centavos"`
	SaldoBancoEm       *string   `json:"saldo_banco_em"`
	LinhasNovas        int       `json:"linhas_novas"`
	LinhasRepetidas    int       `json:"linhas_repetidas"`
	LinhasIgnoradas    int       `json:"linhas_ignoradas"`
	UsuarioID          uuid.UUID `json:"usuario_id"`
	CriadoEm           time.Time `json:"criado_em"`
}

func NovasImportacaoResponses(importacoes []*domaincaixa.ImportacaoExtrato) []ImportacaoResponse {
	respostas := make([]ImportacaoResponse, 0, len(importacoes))
	for _, importacao := range importacoes {
		respostas = append(respostas, ImportacaoResponse{
			ID:                 importacao.ID,
			ContaBancariaID:    importacao.ContaBancariaID,
			ArquivoID:          importacao.ArquivoID,
			PeriodoInicio:      formatarData(importacao.PeriodoInicio),
			PeriodoFim:         formatarData(importacao.PeriodoFim),
			SaldoBancoCentavos: importacao.SaldoBancoCentavos,
			SaldoBancoEm:       formatarDataOpcional(importacao.SaldoBancoEm),
			LinhasNovas:        importacao.LinhasNovas,
			LinhasRepetidas:    importacao.LinhasRepetidas,
			LinhasIgnoradas:    importacao.LinhasIgnoradas,
			UsuarioID:          importacao.UsuarioID,
			CriadoEm:           importacao.CriadoEm,
		})
	}

	return respostas
}

type LancamentoResponse struct {
	ID              uuid.UUID `json:"id"`
	ContaBancariaID uuid.UUID `json:"conta_bancaria_id"`
	ImportacaoID    uuid.UUID `json:"importacao_id"`
	Data            string    `json:"data"`
	ValorCentavos   int64     `json:"valor_centavos"`
	Tipo            string    `json:"tipo"`
	Descricao       string    `json:"descricao"`
	Documento       *string   `json:"documento"`
}

func NovasLancamentoResponses(lancamentos []*domaincaixa.LancamentoExtrato) []LancamentoResponse {
	respostas := make([]LancamentoResponse, 0, len(lancamentos))
	for _, lancamento := range lancamentos {
		tipo := tipoDebito
		if lancamento.Credito() {
			tipo = tipoCredito
		}
		respostas = append(respostas, LancamentoResponse{
			ID:              lancamento.ID,
			ContaBancariaID: lancamento.ContaBancariaID,
			ImportacaoID:    lancamento.ImportacaoID,
			Data:            formatarData(lancamento.Data),
			ValorCentavos:   lancamento.ValorCentavos,
			Tipo:            tipo,
			Descricao:       lancamento.Descricao,
			Documento:       lancamento.Documento,
		})
	}

	return respostas
}
