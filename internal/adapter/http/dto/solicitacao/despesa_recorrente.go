package solicitacao

import (
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type DespesaRecorrenteRequest struct {
	FornecedorID          *uuid.UUID `json:"fornecedor_id"`
	CategoriaFinanceiraID uuid.UUID  `json:"categoria_financeira_id"`
	Descricao             string     `json:"descricao"`
	ValorCentavos         int64      `json:"valor_centavos"`
	Estimado              bool       `json:"estimado"`
	FormaPagamento        string     `json:"forma_pagamento"`
	DiaVencimento         int        `json:"dia_vencimento"`
	VencimentoMesSeguinte bool       `json:"vencimento_mes_seguinte"`
	Inicio                string     `json:"inicio"`
	Fim                   string     `json:"fim"`
}

func (r DespesaRecorrenteRequest) ParaDados() portsin.DadosDespesaRecorrenteInput {
	return portsin.DadosDespesaRecorrenteInput{
		FornecedorID:          r.FornecedorID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Descricao:             r.Descricao,
		ValorCentavos:         r.ValorCentavos,
		Estimado:              r.Estimado,
		FormaPagamento:        r.FormaPagamento,
		DiaVencimento:         r.DiaVencimento,
		VencimentoMesSeguinte: r.VencimentoMesSeguinte,
		Inicio:                r.Inicio,
		Fim:                   r.Fim,
	}
}

type CriarDespesaRecorrenteRequest struct {
	EmpresaID uuid.UUID `json:"empresa_id"`
	DespesaRecorrenteRequest
}

type AlterarDespesaRecorrenteRequest struct {
	DespesaRecorrenteRequest
	Versao int `json:"versao"`
}

type SituacaoRequest struct {
	Versao int `json:"versao"`
}

type DespesaRecorrenteResponse struct {
	ID                    uuid.UUID  `json:"id"`
	EmpresaID             uuid.UUID  `json:"empresa_id"`
	FornecedorID          *uuid.UUID `json:"fornecedor_id"`
	CategoriaFinanceiraID uuid.UUID  `json:"categoria_financeira_id"`
	Descricao             string     `json:"descricao"`
	ValorCentavos         int64      `json:"valor_centavos"`
	Estimado              bool       `json:"estimado"`
	FormaPagamento        string     `json:"forma_pagamento"`
	DiaVencimento         int        `json:"dia_vencimento"`
	VencimentoMesSeguinte bool       `json:"vencimento_mes_seguinte"`
	Inicio                string     `json:"inicio"`
	Fim                   *string    `json:"fim"`
	Ativa                 bool       `json:"ativa"`
	Versao                int        `json:"versao"`
	CriadoEm              time.Time  `json:"criado_em"`
	AtualizadoEm          time.Time  `json:"atualizado_em"`
}

func NovaDespesaRecorrenteResponse(item *domainsolicitacao.DespesaRecorrente) DespesaRecorrenteResponse {
	resposta := DespesaRecorrenteResponse{
		ID:                    item.ID,
		EmpresaID:             item.EmpresaID,
		FornecedorID:          item.FornecedorID,
		CategoriaFinanceiraID: item.CategoriaFinanceiraID,
		Descricao:             item.Descricao.Texto(),
		ValorCentavos:         item.Valor.Centavos(),
		Estimado:              item.Estimado,
		FormaPagamento:        item.FormaPagamento.String(),
		DiaVencimento:         item.DiaVencimento.Dia(),
		VencimentoMesSeguinte: item.VencimentoMesSeguinte,
		Inicio:                item.Inicio.String(),
		Ativa:                 item.Ativa,
		Versao:                item.Versao,
		CriadoEm:              item.CriadoEm,
		AtualizadoEm:          item.AtualizadoEm,
	}
	if item.Fim != nil {
		fim := item.Fim.String()
		resposta.Fim = &fim
	}

	return resposta
}

func NovasDespesaRecorrenteResponses(itens []*domainsolicitacao.DespesaRecorrente) []DespesaRecorrenteResponse {
	respostas := make([]DespesaRecorrenteResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaDespesaRecorrenteResponse(item))
	}

	return respostas
}

type RecorrenciaRecusadaResponse struct {
	DespesaRecorrenteID uuid.UUID `json:"despesa_recorrente_id"`
	Motivo              string    `json:"motivo"`
}

type ResultadoGeracaoResponse struct {
	Competencia  string                        `json:"competencia"`
	Geradas      []uuid.UUID                   `json:"geradas"`
	JaExistentes int                           `json:"ja_existentes"`
	Recusadas    []RecorrenciaRecusadaResponse `json:"recusadas"`
}

func NovoResultadoGeracaoResponse(resultado portsin.ResultadoGeracao) ResultadoGeracaoResponse {
	resposta := ResultadoGeracaoResponse{
		Competencia:  resultado.Competencia,
		Geradas:      resultado.Geradas,
		JaExistentes: resultado.JaExistentes,
		Recusadas:    make([]RecorrenciaRecusadaResponse, 0, len(resultado.Recusadas)),
	}
	if resposta.Geradas == nil {
		resposta.Geradas = []uuid.UUID{}
	}
	for _, recusada := range resultado.Recusadas {
		resposta.Recusadas = append(resposta.Recusadas, RecorrenciaRecusadaResponse{
			DespesaRecorrenteID: recusada.DespesaRecorrenteID,
			Motivo:              recusada.Motivo,
		})
	}

	return resposta
}
