package solicitacao

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const versaoInicial = 1

type Solicitacao struct {
	ID                    uuid.UUID
	SolicitanteID         uuid.UUID
	EmpresaID             uuid.UUID
	AprovadorID           *uuid.UUID
	Valor                 Valor
	Prazo                 PrazoPagamento
	Observacao            Observacao
	FormaPagamento        FormaPagamento
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID *uuid.UUID
	Competencia           Competencia
	Estimado              bool
	Origem                Origem
	OrigemID              *uuid.UUID
	Status                Status
	MotivoRejeicao        *MotivoRejeicao
	AprovadoEm            *time.Time
	RejeitadoEm           *time.Time
	CanceladoEm           *time.Time
	Versao                int
	CriadoEm              time.Time
	AtualizadoEm          time.Time
	ArquivoIDs            []uuid.UUID
}

func ErroSolicitacaoAlterada() error {
	return domain.ErroConflito("solicitação alterada por outra operação, recarregue e tente novamente")
}

func ErroVersaoObrigatoria() error {
	return domain.ErroValidacao("versão da solicitação é obrigatória")
}

func ErroEmpresaNaoEncontrada() error {
	return domain.ErroValidacao("empresa da solicitação não encontrada")
}

func ErroEmpresaInativa() error {
	return domain.ErroValidacao("empresa inativa não recebe solicitação nova")
}

func NovaSolicitacao(
	solicitanteID uuid.UUID,
	empresaID uuid.UUID,
	valor Valor,
	prazo PrazoPagamento,
	observacao Observacao,
	formaPagamento FormaPagamento,
	classificacao Classificacao,
	estimado bool,
	arquivoIDs []uuid.UUID,
) (*Solicitacao, error) {
	if solicitanteID == uuid.Nil {
		return nil, domain.ErroValidacao("solicitante da solicitação não informado")
	}
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa da solicitação não informada")
	}
	if err := formaPagamento.Validado(); err != nil {
		return nil, err
	}

	competencia := CompetenciaDe(prazo.Data())
	if classificacao.Competencia != nil {
		competencia = *classificacao.Competencia
	}

	agora := time.Now().UTC()

	return &Solicitacao{
		ID:                    uuid.New(),
		SolicitanteID:         solicitanteID,
		EmpresaID:             empresaID,
		Valor:                 valor,
		Prazo:                 prazo,
		Observacao:            observacao,
		FormaPagamento:        formaPagamento,
		FornecedorID:          classificacao.FornecedorID,
		CategoriaFinanceiraID: classificacao.CategoriaFinanceiraID,
		Competencia:           competencia,
		Estimado:              estimado,
		Origem:                OrigemManual,
		Status:                StatusPendenteAprovacao,
		Versao:                versaoInicial,
		CriadoEm:              agora,
		AtualizadoEm:          agora,
		ArquivoIDs:            arquivoIDs,
	}, nil
}

func (s *Solicitacao) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return ErroVersaoObrigatoria()
	}
	if informada != s.Versao {
		return ErroSolicitacaoAlterada()
	}

	return nil
}

func (s *Solicitacao) Classificar(classificacao Classificacao, agora time.Time) error {
	if s.Status != StatusAprovado {
		return domain.ErroConflito("só é possível classificar solicitação aprovada")
	}
	if classificacao.CategoriaFinanceiraID == nil {
		return domain.ErroValidacao("categoria financeira é obrigatória na classificação")
	}
	if classificacao.Competencia == nil {
		return domain.ErroValidacao("competência é obrigatória na classificação")
	}

	s.FornecedorID = classificacao.FornecedorID
	s.CategoriaFinanceiraID = classificacao.CategoriaFinanceiraID
	s.Competencia = *classificacao.Competencia
	s.Versao++
	s.AtualizadoEm = agora

	return nil
}

func (s *Solicitacao) Vencida(agora time.Time) bool {
	if s.Status != StatusPendenteAprovacao && s.Status != StatusAprovado {
		return false
	}

	return inicioDoDia(s.Prazo.Data()).Before(inicioDoDia(agora))
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
	s.Versao++

	return nil
}
