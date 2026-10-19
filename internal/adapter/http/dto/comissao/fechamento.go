package comissao

import (
	"time"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type FecharRequest struct {
	EmpresaID             uuid.UUID  `json:"empresa_id"`
	Competencia           string     `json:"competencia"`
	CategoriaFinanceiraID uuid.UUID  `json:"categoria_financeira_id"`
	RepresentanteID       *uuid.UUID `json:"representante_id"`
}

type FechamentoResponse struct {
	ID                    uuid.UUID `json:"id"`
	EmpresaID             uuid.UUID `json:"empresa_id"`
	RepresentanteID       uuid.UUID `json:"representante_id"`
	Competencia           string    `json:"competencia"`
	ComissoesCentavos     int64     `json:"comissoes_centavos"`
	AjustesCentavos       int64     `json:"ajustes_centavos"`
	TotalCentavos         int64     `json:"total_centavos"`
	Vencimento            string    `json:"vencimento"`
	CategoriaFinanceiraID uuid.UUID `json:"categoria_financeira_id"`
	SolicitacaoID         uuid.UUID `json:"solicitacao_id"`
	StatusContaAPagar     string    `json:"status_conta_a_pagar"`
	UsuarioID             uuid.UUID `json:"usuario_id"`
	CriadoEm              time.Time `json:"criado_em"`
}

type FalhaFechamentoResponse struct {
	RepresentanteID uuid.UUID `json:"representante_id"`
	Motivo          string    `json:"motivo"`
}

type ResultadoFechamentoResponse struct {
	Competencia string                    `json:"competencia"`
	Apuracao    ResultadoApuracaoResponse `json:"apuracao"`
	Fechados    []FechamentoResponse      `json:"fechados"`
	Falhas      []FalhaFechamentoResponse `json:"falhas"`
}

type FechamentoDetalheResponse struct {
	FechamentoResponse
	Comissoes []ComissaoResponse `json:"comissoes"`
	Ajustes   []AjusteResponse   `json:"ajustes"`
}

func NovoFechamentoResponse(item *domaincomissao.Fechamento) FechamentoResponse {
	return FechamentoResponse{
		ID:                    item.ID,
		EmpresaID:             item.EmpresaID,
		RepresentanteID:       item.RepresentanteID,
		Competencia:           item.Competencia.String(),
		ComissoesCentavos:     item.ComissoesCentavos,
		AjustesCentavos:       item.AjustesCentavos,
		TotalCentavos:         item.TotalCentavos,
		Vencimento:            item.Vencimento.Format(formatoData),
		CategoriaFinanceiraID: item.CategoriaFinanceiraID,
		SolicitacaoID:         item.SolicitacaoID,
		StatusContaAPagar:     item.StatusContaAPagar,
		UsuarioID:             item.UsuarioID,
		CriadoEm:              item.CriadoEm,
	}
}

func NovoFechamentoResponses(itens []*domaincomissao.Fechamento) []FechamentoResponse {
	respostas := make([]FechamentoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoFechamentoResponse(item))
	}

	return respostas
}

func NovoResultadoFechamentoResponse(resultado domaincomissao.ResultadoFechamento) ResultadoFechamentoResponse {
	falhas := make([]FalhaFechamentoResponse, 0, len(resultado.Falhas))
	for _, falha := range resultado.Falhas {
		falhas = append(falhas, FalhaFechamentoResponse{RepresentanteID: falha.RepresentanteID, Motivo: falha.Motivo})
	}

	return ResultadoFechamentoResponse{
		Competencia: resultado.Competencia.String(),
		Apuracao:    NovoResultadoApuracaoResponse(resultado.Apuracao),
		Fechados:    NovoFechamentoResponses(resultado.Fechados),
		Falhas:      falhas,
	}
}

func NovoFechamentoDetalheResponse(item *domaincomissao.FechamentoComItens) FechamentoDetalheResponse {
	return FechamentoDetalheResponse{
		FechamentoResponse: NovoFechamentoResponse(item.Fechamento),
		Comissoes:          NovoComissaoResponses(item.Comissoes),
		Ajustes:            NovoAjusteResponses(item.Ajustes),
	}
}
