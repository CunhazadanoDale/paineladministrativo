package caixa

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

const formatoCompetencia = "2006-01"

func ParaCompetencia(valor, campo string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, nil
	}
	if competencia, err := time.Parse(formatoCompetencia, valor); err == nil {
		return competencia, nil
	}

	return time.Time{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM")
}

type FecharRequest struct {
	ContaBancariaID    uuid.UUID `json:"conta_bancaria_id"`
	Competencia        string    `json:"competencia"`
	SaldoBancoCentavos *int64    `json:"saldo_banco_centavos"`
}

func (r FecharRequest) ParaInput(usuarioID uuid.UUID) (portsin.FecharInput, error) {
	competencia, err := ParaCompetencia(r.Competencia, "competencia")
	if err != nil {
		return portsin.FecharInput{}, err
	}

	return portsin.FecharInput{
		UsuarioID:          usuarioID,
		ContaBancariaID:    r.ContaBancariaID,
		Competencia:        competencia,
		SaldoBancoCentavos: r.SaldoBancoCentavos,
	}, nil
}

type ReabrirRequest struct {
	Motivo string `json:"motivo"`
}

type PreviaResponse struct {
	ContaBancariaID         uuid.UUID           `json:"conta_bancaria_id"`
	Competencia             string              `json:"competencia"`
	SaldoInicialCentavos    int64               `json:"saldo_inicial_centavos"`
	EntradasCentavos        int64               `json:"entradas_centavos"`
	SaidasCentavos          int64               `json:"saidas_centavos"`
	SaldoFinalCentavos      int64               `json:"saldo_final_centavos"`
	SaldoBancoCentavos      *int64              `json:"saldo_banco_centavos"`
	LancamentosPendentes    int                 `json:"lancamentos_pendentes"`
	ConciliacoesInvalidadas int                 `json:"conciliacoes_invalidadas"`
	PodeFechar              bool                `json:"pode_fechar"`
	MovimentosSemLinha      []MovimentoResponse `json:"movimentos_sem_linha"`
}

func NovaPreviaResponse(previa *domaincaixa.PreviaFechamento) PreviaResponse {
	saldoConfere := previa.SaldoBancoCentavos == nil || *previa.SaldoBancoCentavos == previa.SaldoFinalCentavos()

	return PreviaResponse{
		ContaBancariaID:         previa.ContaBancariaID,
		Competencia:             previa.Competencia.Format(formatoCompetencia),
		SaldoInicialCentavos:    previa.SaldoInicialCentavos,
		EntradasCentavos:        previa.EntradasCentavos,
		SaidasCentavos:          previa.SaidasCentavos,
		SaldoFinalCentavos:      previa.SaldoFinalCentavos(),
		SaldoBancoCentavos:      previa.SaldoBancoCentavos,
		LancamentosPendentes:    previa.LancamentosPendentes,
		ConciliacoesInvalidadas: previa.ConciliacoesInvalidadas,
		PodeFechar:              previa.Bloqueios() == nil && saldoConfere,
		MovimentosSemLinha:      NovosMovimentoResponses(previa.MovimentosSemLinha),
	}
}

type FechamentoResponse struct {
	ID                   uuid.UUID  `json:"id"`
	ContaBancariaID      uuid.UUID  `json:"conta_bancaria_id"`
	Competencia          string     `json:"competencia"`
	SaldoInicialCentavos int64      `json:"saldo_inicial_centavos"`
	EntradasCentavos     int64      `json:"entradas_centavos"`
	SaidasCentavos       int64      `json:"saidas_centavos"`
	SaldoFinalCentavos   int64      `json:"saldo_final_centavos"`
	SaldoBancoCentavos   int64      `json:"saldo_banco_centavos"`
	Vigente              bool       `json:"vigente"`
	ReabertoEm           *time.Time `json:"reaberto_em"`
	ReabertoPor          *uuid.UUID `json:"reaberto_por"`
	MotivoReabertura     *string    `json:"motivo_reabertura"`
	UsuarioID            uuid.UUID  `json:"usuario_id"`
	CriadoEm             time.Time  `json:"criado_em"`
}

func NovoFechamentoResponse(fechamento *domaincaixa.Fechamento) FechamentoResponse {
	return FechamentoResponse{
		ID:                   fechamento.ID,
		ContaBancariaID:      fechamento.ContaBancariaID,
		Competencia:          fechamento.Competencia.Format(formatoCompetencia),
		SaldoInicialCentavos: fechamento.SaldoInicialCentavos,
		EntradasCentavos:     fechamento.EntradasCentavos,
		SaidasCentavos:       fechamento.SaidasCentavos,
		SaldoFinalCentavos:   fechamento.SaldoFinalCentavos,
		SaldoBancoCentavos:   fechamento.SaldoBancoCentavos,
		Vigente:              fechamento.Vigente,
		ReabertoEm:           fechamento.ReabertoEm,
		ReabertoPor:          fechamento.ReabertoPor,
		MotivoReabertura:     fechamento.MotivoReabertura,
		UsuarioID:            fechamento.UsuarioID,
		CriadoEm:             fechamento.CriadoEm,
	}
}

func NovosFechamentoResponses(fechamentos []*domaincaixa.Fechamento) []FechamentoResponse {
	respostas := make([]FechamentoResponse, 0, len(fechamentos))
	for _, fechamento := range fechamentos {
		respostas = append(respostas, NovoFechamentoResponse(fechamento))
	}

	return respostas
}

type DetalheFechamentoResponse struct {
	FechamentoResponse
	ConciliacoesInvalidadas int                 `json:"conciliacoes_invalidadas"`
	MovimentosSemLinha      []MovimentoResponse `json:"movimentos_sem_linha"`
}

func NovoDetalheFechamentoResponse(detalhe *portsin.DetalheFechamento) DetalheFechamentoResponse {
	return DetalheFechamentoResponse{
		FechamentoResponse:      NovoFechamentoResponse(detalhe.Fechamento),
		ConciliacoesInvalidadas: detalhe.ConciliacoesInvalidadas,
		MovimentosSemLinha:      NovosMovimentoResponses(detalhe.MovimentosSemLinha),
	}
}
