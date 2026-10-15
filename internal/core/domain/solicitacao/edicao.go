package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type DadosEdicao struct {
	Valor          Valor
	Prazo          PrazoPagamento
	Observacao     Observacao
	FormaPagamento FormaPagamento
	Classificacao  Classificacao
	Estimado       bool
}

func (s *Solicitacao) PodeSerEditadaPor(usuarioID uuid.UUID, administrador bool) bool {
	return administrador || s.SolicitanteID == usuarioID
}

func ErroCompetenciaDaOrigemFixa() error {
	return domain.ErroValidacao("competência de solicitação gerada por outro módulo não pode mudar")
}

func (s *Solicitacao) ExigirEditavel() error {
	if s.Status != StatusPendenteAprovacao {
		return domain.ErroConflito("só é possível editar solicitação pendente de aprovação")
	}

	return nil
}

func (s *Solicitacao) ExigirAdiavel() error {
	if s.Status != StatusAprovado {
		return domain.ErroConflito("só é possível adiar solicitação aprovada")
	}

	return nil
}

func (s *Solicitacao) Editar(dados DadosEdicao, agora time.Time) ([]string, error) {
	if err := s.ExigirEditavel(); err != nil {
		return nil, err
	}
	if err := dados.FormaPagamento.Validado(); err != nil {
		return nil, err
	}

	competencia, err := s.competenciaAposAlteracao(dados.Classificacao.Competencia, dados.Prazo)
	if err != nil {
		return nil, err
	}

	alterados := make([]string, 0, 8)
	marcar := func(alterado bool, campo string) {
		if alterado {
			alterados = append(alterados, campo)
		}
	}
	marcar(s.Valor != dados.Valor, "valor")
	marcar(!mesmoDia(s.Prazo.Data(), dados.Prazo.Data()), "vencimento")
	marcar(s.Observacao != dados.Observacao, "observação")
	marcar(s.FormaPagamento != dados.FormaPagamento, "forma de pagamento")
	marcar(!mesmoID(s.FornecedorID, dados.Classificacao.FornecedorID), "fornecedor")
	marcar(!mesmoID(s.CategoriaFinanceiraID, dados.Classificacao.CategoriaFinanceiraID), "categoria financeira")
	marcar(s.Competencia != competencia, "competência")
	marcar(s.Estimado != dados.Estimado, "estimado")

	if len(alterados) == 0 {
		return alterados, nil
	}

	s.Valor = dados.Valor
	s.Prazo = dados.Prazo
	s.Observacao = dados.Observacao
	s.FormaPagamento = dados.FormaPagamento
	s.FornecedorID = dados.Classificacao.FornecedorID
	s.CategoriaFinanceiraID = dados.Classificacao.CategoriaFinanceiraID
	s.Competencia = competencia
	s.Estimado = dados.Estimado
	s.Versao++
	s.AtualizadoEm = agora

	return alterados, nil
}

func (s *Solicitacao) Adiar(novoVencimento PrazoPagamento, agora time.Time) (PrazoPagamento, error) {
	if err := s.ExigirAdiavel(); err != nil {
		return PrazoPagamento{}, err
	}
	if !inicioDoDia(novoVencimento.Data()).After(inicioDoDia(s.Prazo.Data())) {
		return PrazoPagamento{}, domain.ErroValidacao("novo vencimento deve ser posterior ao atual")
	}

	anterior := s.Prazo
	s.Prazo = novoVencimento
	s.Versao++
	s.AtualizadoEm = agora

	return anterior, nil
}

func (s *Solicitacao) competenciaAposAlteracao(informada *Competencia, prazo PrazoPagamento) (Competencia, error) {
	if s.Origem != OrigemManual {
		if informada != nil && *informada != s.Competencia {
			return Competencia{}, ErroCompetenciaDaOrigemFixa()
		}

		return s.Competencia, nil
	}
	if informada != nil {
		return *informada, nil
	}

	return CompetenciaDe(prazo.Data()), nil
}

func (s *Solicitacao) MesmoVencimento(prazo PrazoPagamento) bool {
	return mesmoDia(s.Prazo.Data(), prazo.Data())
}

func mesmoDia(a, b time.Time) bool {
	return inicioDoDia(a).Equal(inicioDoDia(b))
}

func mesmoID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
