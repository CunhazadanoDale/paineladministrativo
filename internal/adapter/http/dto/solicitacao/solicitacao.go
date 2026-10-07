package solicitacao

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

const formatoDataPagamento = "2006-01-02"

type CriarSolicitacaoRequest struct {
	ValorCentavos  int64       `json:"valor_centavos"`
	PrazoPagamento string      `json:"prazo_pagamento"`
	Observacao     string      `json:"observacao"`
	FormaPagamento string      `json:"forma_pagamento"`
	ArquivoIDs     []uuid.UUID `json:"arquivo_ids"`
}

func (r CriarSolicitacaoRequest) ParaInput(solicitanteID uuid.UUID) (portsin.CriarSolicitacaoInput, error) {
	prazo, err := paraPrazoPagamento(r.PrazoPagamento)
	if err != nil {
		return portsin.CriarSolicitacaoInput{}, err
	}

	return portsin.CriarSolicitacaoInput{
		SolicitanteID:  solicitanteID,
		ValorCentavos:  r.ValorCentavos,
		PrazoPagamento: prazo,
		Observacao:     r.Observacao,
		FormaPagamento: r.FormaPagamento,
		ArquivoIDs:     r.ArquivoIDs,
	}, nil
}

type RejeitarRequest struct {
	Motivo string `json:"motivo"`
}

type RegistrarPagamentoRequest struct {
	ValorCentavos        int64      `json:"valor_centavos"`
	ComprovanteArquivoID *uuid.UUID `json:"comprovante_arquivo_id"`
	PagoEm               string     `json:"pago_em"`
}

func (r RegistrarPagamentoRequest) ParaInput(solicitacaoID, usuarioID uuid.UUID) (portsin.RegistrarPagamentoInput, error) {
	input := portsin.RegistrarPagamentoInput{
		SolicitacaoID:        solicitacaoID,
		UsuarioID:            usuarioID,
		ValorCentavos:        r.ValorCentavos,
		ComprovanteArquivoID: r.ComprovanteArquivoID,
	}

	pagoEm := strings.TrimSpace(r.PagoEm)
	if pagoEm == "" {
		return input, nil
	}

	data, err := time.Parse(time.RFC3339, pagoEm)
	if err != nil {
		return portsin.RegistrarPagamentoInput{}, domain.ErroValidacao("data do pagamento deve estar no formato AAAA-MM-DDTHH:MM:SSZ")
	}
	input.PagoEm = data

	return input, nil
}

type SolicitacaoResponse struct {
	ID             uuid.UUID   `json:"id"`
	SolicitanteID  uuid.UUID   `json:"solicitante_id"`
	AprovadorID    *uuid.UUID  `json:"aprovador_id"`
	ValorCentavos  int64       `json:"valor_centavos"`
	PrazoPagamento time.Time   `json:"prazo_pagamento"`
	Observacao     string      `json:"observacao"`
	FormaPagamento string      `json:"forma_pagamento"`
	Status         string      `json:"status"`
	MotivoRejeicao *string     `json:"motivo_rejeicao"`
	AprovadoEm     *time.Time  `json:"aprovado_em"`
	RejeitadoEm    *time.Time  `json:"rejeitado_em"`
	CanceladoEm    *time.Time  `json:"cancelado_em"`
	CriadoEm       time.Time   `json:"criado_em"`
	AtualizadoEm   time.Time   `json:"atualizado_em"`
	ArquivoIDs     []uuid.UUID `json:"arquivo_ids"`
}

func NovaSolicitacaoResponse(item *domainsolicitacao.Solicitacao) SolicitacaoResponse {
	resposta := SolicitacaoResponse{
		ID:             item.ID,
		SolicitanteID:  item.SolicitanteID,
		AprovadorID:    item.AprovadorID,
		ValorCentavos:  item.Valor.Centavos(),
		PrazoPagamento: item.Prazo.Data(),
		Observacao:     item.Observacao.Texto(),
		FormaPagamento: item.FormaPagamento.String(),
		Status:         item.Status.String(),
		AprovadoEm:     item.AprovadoEm,
		RejeitadoEm:    item.RejeitadoEm,
		CanceladoEm:    item.CanceladoEm,
		CriadoEm:       item.CriadoEm,
		AtualizadoEm:   item.AtualizadoEm,
		ArquivoIDs:     item.ArquivoIDs,
	}

	if item.MotivoRejeicao != nil {
		motivo := item.MotivoRejeicao.Texto()
		resposta.MotivoRejeicao = &motivo
	}
	if resposta.ArquivoIDs == nil {
		resposta.ArquivoIDs = []uuid.UUID{}
	}

	return resposta
}

func NovaSolicitacaoResponses(itens []*domainsolicitacao.Solicitacao) []SolicitacaoResponse {
	respostas := make([]SolicitacaoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaSolicitacaoResponse(item))
	}

	return respostas
}

type PagamentoResponse struct {
	ID                   uuid.UUID  `json:"id"`
	SolicitacaoID        uuid.UUID  `json:"solicitacao_id"`
	ComprovanteArquivoID *uuid.UUID `json:"comprovante_arquivo_id"`
	ValorCentavos        int64      `json:"valor_centavos"`
	PagoEm               time.Time  `json:"pago_em"`
	CriadoEm             time.Time  `json:"criado_em"`
}

func NovoPagamentoResponse(item *domainsolicitacao.Pagamento) PagamentoResponse {
	return PagamentoResponse{
		ID:                   item.ID,
		SolicitacaoID:        item.SolicitacaoID,
		ComprovanteArquivoID: item.ComprovanteArquivoID,
		ValorCentavos:        item.Valor.Centavos(),
		PagoEm:               item.PagoEm,
		CriadoEm:             item.CriadoEm,
	}
}

type HistoricoResponse struct {
	ID            uuid.UUID `json:"id"`
	SolicitacaoID uuid.UUID `json:"solicitacao_id"`
	UsuarioID     uuid.UUID `json:"usuario_id"`
	DeStatus      *string   `json:"de_status"`
	ParaStatus    string    `json:"para_status"`
	Descricao     string    `json:"descricao"`
	CriadoEm      time.Time `json:"criado_em"`
}

func NovoHistoricoResponse(item *domainsolicitacao.Historico) HistoricoResponse {
	resposta := HistoricoResponse{
		ID:            item.ID,
		SolicitacaoID: item.SolicitacaoID,
		UsuarioID:     item.UsuarioID,
		ParaStatus:    item.ParaStatus.String(),
		Descricao:     item.Descricao,
		CriadoEm:      item.CriadoEm,
	}

	if item.DeStatus != nil {
		deStatus := item.DeStatus.String()
		resposta.DeStatus = &deStatus
	}

	return resposta
}

func NovoHistoricoResponses(itens []*domainsolicitacao.Historico) []HistoricoResponse {
	respostas := make([]HistoricoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoHistoricoResponse(item))
	}

	return respostas
}

func paraPrazoPagamento(valor string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, domain.ErroValidacao("prazo de pagamento é obrigatório")
	}

	if data, err := time.Parse(time.RFC3339, valor); err == nil {
		return data, nil
	}

	if data, err := time.Parse(formatoDataPagamento, valor); err == nil {
		return data, nil
	}

	return time.Time{}, domain.ErroValidacao("prazo de pagamento deve estar no formato AAAA-MM-DD")
}
