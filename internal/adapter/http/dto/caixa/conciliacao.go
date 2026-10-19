package caixa

import (
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

type ReferenciaRequest struct {
	Tipo string    `json:"tipo"`
	ID   uuid.UUID `json:"id"`
}

type ConciliarRequest struct {
	Movimentos []ReferenciaRequest `json:"movimentos"`
}

func (r ConciliarRequest) ParaInput(usuarioID, lancamentoID uuid.UUID) portsin.ConciliarInput {
	movimentos := make([]portsin.ReferenciaInput, 0, len(r.Movimentos))
	for _, movimento := range r.Movimentos {
		movimentos = append(movimentos, portsin.ReferenciaInput{Tipo: movimento.Tipo, ID: movimento.ID})
	}

	return portsin.ConciliarInput{UsuarioID: usuarioID, LancamentoID: lancamentoID, Movimentos: movimentos}
}

type ConciliarAutomaticoRequest struct {
	ContaBancariaID uuid.UUID `json:"conta_bancaria_id"`
	De              string    `json:"de"`
	Ate             string    `json:"ate"`
}

func (r ConciliarAutomaticoRequest) ParaInput(usuarioID uuid.UUID) (portsin.ConciliarAutomaticoInput, error) {
	de, err := ParaData(r.De, "de")
	if err != nil {
		return portsin.ConciliarAutomaticoInput{}, err
	}
	ate, err := ParaData(r.Ate, "ate")
	if err != nil {
		return portsin.ConciliarAutomaticoInput{}, err
	}

	return portsin.ConciliarAutomaticoInput{UsuarioID: usuarioID, ContaBancariaID: r.ContaBancariaID, De: de, Ate: ate}, nil
}

type DesfazerConciliacaoRequest struct {
	Motivo string `json:"motivo"`
}

type ResultadoAutomaticoResponse struct {
	Conciliadas  int  `json:"conciliadas"`
	Ambiguos     int  `json:"ambiguos"`
	SemPar       int  `json:"sem_par"`
	Falhas       int  `json:"falhas"`
	RestamLinhas bool `json:"restam_linhas"`
}

func NovoResultadoAutomaticoResponse(resultado *portsin.ResultadoAutomatico) ResultadoAutomaticoResponse {
	return ResultadoAutomaticoResponse{
		Conciliadas:  resultado.Conciliadas,
		Ambiguos:     resultado.Ambiguos,
		SemPar:       resultado.SemPar,
		Falhas:       resultado.Falhas,
		RestamLinhas: resultado.RestamLinhas,
	}
}

type MovimentoResponse struct {
	Tipo            string     `json:"tipo"`
	ID              uuid.UUID  `json:"id"`
	ContaBancariaID *uuid.UUID `json:"conta_bancaria_id"`
	EmpresaID       uuid.UUID  `json:"empresa_id"`
	Data            string     `json:"data"`
	ValorCentavos   int64      `json:"valor_centavos"`
	Descricao       string     `json:"descricao"`
	OrigemID        uuid.UUID  `json:"origem_id"`
	Estornado       bool       `json:"estornado"`
	ConciliacaoID   *uuid.UUID `json:"conciliacao_id"`
}

func NovosMovimentoResponses(movimentos []*domaincaixa.MovimentoSistema) []MovimentoResponse {
	respostas := make([]MovimentoResponse, 0, len(movimentos))
	for _, movimento := range movimentos {
		resposta := MovimentoResponse{
			Tipo:          movimento.Tipo.String(),
			ID:            movimento.ID,
			EmpresaID:     movimento.EmpresaID,
			Data:          formatarData(movimento.Data),
			ValorCentavos: movimento.ValorCentavos,
			Descricao:     movimento.Descricao,
			OrigemID:      movimento.OrigemID,
			Estornado:     movimento.Estornado,
			ConciliacaoID: movimento.ConciliacaoID,
		}
		if movimento.ContaBancariaID != uuid.Nil {
			conta := movimento.ContaBancariaID
			resposta.ContaBancariaID = &conta
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}

type ItemConciliacaoResponse struct {
	Tipo          string    `json:"tipo"`
	ID            uuid.UUID `json:"id"`
	ValorCentavos int64     `json:"valor_centavos"`
	Estornado     bool      `json:"estornado"`
}

type ConciliacaoResponse struct {
	ID                  uuid.UUID                 `json:"id"`
	ExtratoLancamentoID uuid.UUID                 `json:"extrato_lancamento_id"`
	ContaBancariaID     uuid.UUID                 `json:"conta_bancaria_id"`
	Modo                string                    `json:"modo"`
	Vigente             bool                      `json:"vigente"`
	Invalidada          bool                      `json:"invalidada"`
	Movimentos          []ItemConciliacaoResponse `json:"movimentos"`
	UsuarioID           uuid.UUID                 `json:"usuario_id"`
	CriadoEm            time.Time                 `json:"criado_em"`
	DesfeitaEm          *time.Time                `json:"desfeita_em"`
	DesfeitaPor         *uuid.UUID                `json:"desfeita_por"`
	MotivoDesfazer      *string                   `json:"motivo_desfazer"`
}

func NovaConciliacaoResponse(conciliacao *domaincaixa.Conciliacao) ConciliacaoResponse {
	resposta := ConciliacaoResponse{
		ID:                  conciliacao.ID,
		ExtratoLancamentoID: conciliacao.ExtratoLancamentoID,
		ContaBancariaID:     conciliacao.ContaBancariaID,
		Modo:                conciliacao.Modo.String(),
		Vigente:             conciliacao.Vigente,
		Invalidada:          conciliacao.Invalidada(),
		Movimentos:          make([]ItemConciliacaoResponse, 0, len(conciliacao.Itens)),
		UsuarioID:           conciliacao.UsuarioID,
		CriadoEm:            conciliacao.CriadoEm,
		DesfeitaEm:          conciliacao.DesfeitaEm,
		DesfeitaPor:         conciliacao.DesfeitaPor,
		MotivoDesfazer:      conciliacao.MotivoDesfazer,
	}
	for _, item := range conciliacao.Itens {
		resposta.Movimentos = append(resposta.Movimentos, ItemConciliacaoResponse{
			Tipo: item.Tipo.String(), ID: item.AlvoID, ValorCentavos: item.ValorCentavos, Estornado: item.Estornado,
		})
	}

	return resposta
}
