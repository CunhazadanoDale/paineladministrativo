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
	EmpresaID             uuid.UUID   `json:"empresa_id"`
	ValorCentavos         int64       `json:"valor_centavos"`
	PrazoPagamento        string      `json:"prazo_pagamento"`
	Observacao            string      `json:"observacao"`
	FormaPagamento        string      `json:"forma_pagamento"`
	FornecedorID          *uuid.UUID  `json:"fornecedor_id"`
	CategoriaFinanceiraID *uuid.UUID  `json:"categoria_financeira_id"`
	Competencia           string      `json:"competencia"`
	Estimado              bool        `json:"estimado"`
	ArquivoIDs            []uuid.UUID `json:"arquivo_ids"`
}

func (r CriarSolicitacaoRequest) ParaInput(solicitanteID uuid.UUID) (portsin.CriarSolicitacaoInput, error) {
	prazo, err := paraPrazoPagamento(r.PrazoPagamento)
	if err != nil {
		return portsin.CriarSolicitacaoInput{}, err
	}

	return portsin.CriarSolicitacaoInput{
		SolicitanteID:         solicitanteID,
		EmpresaID:             r.EmpresaID,
		ValorCentavos:         r.ValorCentavos,
		PrazoPagamento:        prazo,
		Observacao:            r.Observacao,
		FormaPagamento:        r.FormaPagamento,
		FornecedorID:          r.FornecedorID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Competencia:           r.Competencia,
		Estimado:              r.Estimado,
		ArquivoIDs:            r.ArquivoIDs,
	}, nil
}

type EditarSolicitacaoRequest struct {
	ValorCentavos         int64      `json:"valor_centavos"`
	PrazoPagamento        string     `json:"prazo_pagamento"`
	Observacao            string     `json:"observacao"`
	FormaPagamento        string     `json:"forma_pagamento"`
	FornecedorID          *uuid.UUID `json:"fornecedor_id"`
	CategoriaFinanceiraID *uuid.UUID `json:"categoria_financeira_id"`
	Competencia           string     `json:"competencia"`
	Estimado              bool       `json:"estimado"`
	Versao                int        `json:"versao"`
}

func (r EditarSolicitacaoRequest) ParaInput(solicitacaoID, usuarioID uuid.UUID) (portsin.EditarSolicitacaoInput, error) {
	prazo, err := paraPrazoPagamento(r.PrazoPagamento)
	if err != nil {
		return portsin.EditarSolicitacaoInput{}, err
	}

	return portsin.EditarSolicitacaoInput{
		SolicitacaoID:         solicitacaoID,
		UsuarioID:             usuarioID,
		ValorCentavos:         r.ValorCentavos,
		PrazoPagamento:        prazo,
		Observacao:            r.Observacao,
		FormaPagamento:        r.FormaPagamento,
		FornecedorID:          r.FornecedorID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Competencia:           r.Competencia,
		Estimado:              r.Estimado,
		Versao:                r.Versao,
	}, nil
}

type AdiarSolicitacaoRequest struct {
	Vencimento string `json:"vencimento"`
	Motivo     string `json:"motivo"`
	Versao     int    `json:"versao"`
}

func (r AdiarSolicitacaoRequest) ParaInput(solicitacaoID, usuarioID uuid.UUID) (portsin.AdiarSolicitacaoInput, error) {
	valor := strings.TrimSpace(r.Vencimento)
	if valor == "" {
		return portsin.AdiarSolicitacaoInput{}, domain.ErroValidacao("vencimento é obrigatório")
	}

	vencimento, err := time.Parse(formatoDataPagamento, valor)
	if err != nil {
		return portsin.AdiarSolicitacaoInput{}, domain.ErroValidacao("vencimento deve estar no formato AAAA-MM-DD")
	}

	return portsin.AdiarSolicitacaoInput{
		SolicitacaoID: solicitacaoID,
		UsuarioID:     usuarioID,
		Vencimento:    vencimento,
		Motivo:        r.Motivo,
		Versao:        r.Versao,
	}, nil
}

type ClassificarSolicitacaoRequest struct {
	FornecedorID          *uuid.UUID `json:"fornecedor_id"`
	CategoriaFinanceiraID *uuid.UUID `json:"categoria_financeira_id"`
	Competencia           string     `json:"competencia"`
	Versao                int        `json:"versao"`
}

func (r ClassificarSolicitacaoRequest) ParaInput(solicitacaoID, usuarioID uuid.UUID) portsin.ClassificarSolicitacaoInput {
	return portsin.ClassificarSolicitacaoInput{
		SolicitacaoID:         solicitacaoID,
		UsuarioID:             usuarioID,
		FornecedorID:          r.FornecedorID,
		CategoriaFinanceiraID: r.CategoriaFinanceiraID,
		Competencia:           r.Competencia,
		Versao:                r.Versao,
	}
}

func DataDeConsulta(valor, nome string) (*time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return nil, nil
	}

	data, err := time.Parse(formatoDataPagamento, valor)
	if err != nil {
		return nil, domain.ErroValidacao("consulta " + nome + " deve estar no formato AAAA-MM-DD")
	}

	return &data, nil
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
	ID                    uuid.UUID   `json:"id"`
	SolicitanteID         uuid.UUID   `json:"solicitante_id"`
	EmpresaID             uuid.UUID   `json:"empresa_id"`
	AprovadorID           *uuid.UUID  `json:"aprovador_id"`
	ValorCentavos         int64       `json:"valor_centavos"`
	PrazoPagamento        time.Time   `json:"prazo_pagamento"`
	Observacao            string      `json:"observacao"`
	FormaPagamento        string      `json:"forma_pagamento"`
	FornecedorID          *uuid.UUID  `json:"fornecedor_id"`
	CategoriaFinanceiraID *uuid.UUID  `json:"categoria_financeira_id"`
	Competencia           string      `json:"competencia"`
	Estimado              bool        `json:"estimado"`
	Origem                string      `json:"origem"`
	OrigemID              *uuid.UUID  `json:"origem_id"`
	Vencida               bool        `json:"vencida"`
	Versao                int         `json:"versao"`
	Status                string      `json:"status"`
	MotivoRejeicao        *string     `json:"motivo_rejeicao"`
	AprovadoEm            *time.Time  `json:"aprovado_em"`
	RejeitadoEm           *time.Time  `json:"rejeitado_em"`
	CanceladoEm           *time.Time  `json:"cancelado_em"`
	CriadoEm              time.Time   `json:"criado_em"`
	AtualizadoEm          time.Time   `json:"atualizado_em"`
	ArquivoIDs            []uuid.UUID `json:"arquivo_ids"`
}

func NovaSolicitacaoResponse(item *domainsolicitacao.Solicitacao) SolicitacaoResponse {
	resposta := SolicitacaoResponse{
		ID:                    item.ID,
		SolicitanteID:         item.SolicitanteID,
		EmpresaID:             item.EmpresaID,
		AprovadorID:           item.AprovadorID,
		ValorCentavos:         item.Valor.Centavos(),
		PrazoPagamento:        item.Prazo.Data(),
		Observacao:            item.Observacao.Texto(),
		FormaPagamento:        item.FormaPagamento.String(),
		FornecedorID:          item.FornecedorID,
		CategoriaFinanceiraID: item.CategoriaFinanceiraID,
		Competencia:           item.Competencia.String(),
		Estimado:              item.Estimado,
		Origem:                item.Origem.String(),
		OrigemID:              item.OrigemID,
		Vencida:               item.Vencida(time.Now()),
		Versao:                item.Versao,
		Status:                item.Status.String(),
		AprovadoEm:            item.AprovadoEm,
		RejeitadoEm:           item.RejeitadoEm,
		CanceladoEm:           item.CanceladoEm,
		CriadoEm:              item.CriadoEm,
		AtualizadoEm:          item.AtualizadoEm,
		ArquivoIDs:            item.ArquivoIDs,
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
	Evento        string    `json:"evento"`
	Descricao     string    `json:"descricao"`
	CriadoEm      time.Time `json:"criado_em"`
}

func NovoHistoricoResponse(item *domainsolicitacao.Historico) HistoricoResponse {
	resposta := HistoricoResponse{
		ID:            item.ID,
		SolicitacaoID: item.SolicitacaoID,
		UsuarioID:     item.UsuarioID,
		ParaStatus:    item.ParaStatus.String(),
		Evento:        item.Evento.String(),
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
