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

func (s *Solicitacao) Editar(dados DadosEdicao, agora time.Time) ([]string, error) {
	if s.Status != StatusPendenteAprovacao {
		return nil, domain.ErroConflito("só é possível editar solicitação pendente de aprovação")
	}
	if err := dados.FormaPagamento.Validado(); err != nil {
		return nil, err
	}

	competencia := CompetenciaDe(dados.Prazo.Data())
	if dados.Classificacao.Competencia != nil {
		competencia = *dados.Classificacao.Competencia
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
	if s.Status != StatusAprovado {
		return PrazoPagamento{}, domain.ErroConflito("só é possível adiar solicitação aprovada")
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

func mesmoDia(a, b time.Time) bool {
	return inicioDoDia(a).Equal(inicioDoDia(b))
}

func mesmoID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
