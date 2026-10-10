package solicitacao

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Solicitacao struct {
	ID             uuid.UUID
	SolicitanteID  uuid.UUID
	AprovadorID    *uuid.UUID
	Valor          Valor
	Prazo          PrazoPagamento
	Observacao     Observacao
	FormaPagamento FormaPagamento
	Status         Status
	MotivoRejeicao *MotivoRejeicao
	AprovadoEm     *time.Time
	RejeitadoEm    *time.Time
	CanceladoEm    *time.Time
	CriadoEm       time.Time
	AtualizadoEm   time.Time
	ArquivoIDs     []uuid.UUID
}

func NovaSolicitacao(
	solicitanteID uuid.UUID,
	valor Valor,
	prazo PrazoPagamento,
	observacao Observacao,
	formaPagamento FormaPagamento,
	arquivoIDs []uuid.UUID,
) (*Solicitacao, error) {
	if solicitanteID == uuid.Nil {
		return nil, domain.ErroValidacao("solicitante da solicitação não informado")
	}
	if err := formaPagamento.Validado(); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &Solicitacao{
		ID:             uuid.New(),
		SolicitanteID:  solicitanteID,
		Valor:          valor,
		Prazo:          prazo,
		Observacao:     observacao,
		FormaPagamento: formaPagamento,
		Status:         StatusPendenteAprovacao,
		CriadoEm:       agora,
		AtualizadoEm:   agora,
		ArquivoIDs:     arquivoIDs,
	}, nil
}

func (s *Solicitacao) Aprovar(aprovadorID uuid.UUID, agora time.Time) error {
	if aprovadorID == uuid.Nil {
		return domain.ErroValidacao("aprovador da solicitação não informado")
	}
	if aprovadorID == s.SolicitanteID {
		return domain.ErroPermissao("o solicitante não pode aprovar a própria solicitação")
	}
	if err := s.transicionarPara(StatusAprovado); err != nil {
		return err
	}

	s.AprovadorID = &aprovadorID
	s.AprovadoEm = &agora
	s.AtualizadoEm = agora

	return nil
}

func (s *Solicitacao) Rejeitar(aprovadorID uuid.UUID, motivo MotivoRejeicao, agora time.Time) error {
	if aprovadorID == uuid.Nil {
		return domain.ErroValidacao("aprovador da solicitação não informado")
	}
	if err := s.transicionarPara(StatusRejeitado); err != nil {
		return err
	}

	s.AprovadorID = &aprovadorID
	s.MotivoRejeicao = &motivo
	s.RejeitadoEm = &agora
	s.AtualizadoEm = agora

	return nil
}

func (s *Solicitacao) Cancelar(agora time.Time) error {
	if err := s.transicionarPara(StatusCancelado); err != nil {
		return err
	}

	s.CanceladoEm = &agora
	s.AtualizadoEm = agora

	return nil
}

func (s *Solicitacao) MarcarComoPago(agora time.Time) error {
	if err := s.transicionarPara(StatusPago); err != nil {
		return err
	}

	s.AtualizadoEm = agora

	return nil
}

func (s *Solicitacao) PodeSerCanceladoPor(usuarioID uuid.UUID, administrador bool) bool {
	return administrador || s.SolicitanteID == usuarioID
}

func (s *Solicitacao) transicionarPara(proximo Status) error {
	if !s.Status.PodeTransicionarPara(proximo) {
		return domain.ErroConflito(fmt.Sprintf(
			"não é possível mudar a solicitação de %q para %q", s.Status, proximo,
		))
	}

	s.Status = proximo

	return nil
}
