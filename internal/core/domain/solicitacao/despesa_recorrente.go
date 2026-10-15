package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type DespesaRecorrente struct {
	ID                    uuid.UUID
	EmpresaID             uuid.UUID
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Descricao             Observacao
	Valor                 Valor
	Estimado              bool
	FormaPagamento        FormaPagamento
	DiaVencimento         DiaVencimento
	VencimentoMesSeguinte bool
	Inicio                Competencia
	Fim                   *Competencia
	Ativa                 bool
	Versao                int
	CriadoEm              time.Time
	AtualizadoEm          time.Time
}

type DadosDespesaRecorrente struct {
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Descricao             Observacao
	Valor                 Valor
	Estimado              bool
	FormaPagamento        FormaPagamento
	DiaVencimento         DiaVencimento
	VencimentoMesSeguinte bool
	Inicio                Competencia
	Fim                   *Competencia
}

func ErroDespesaRecorrenteNaoEncontrada() error {
	return domain.ErroNaoEncontrado("despesa recorrente não encontrada")
}

func ErroDespesaRecorrenteAlterada() error {
	return domain.ErroConflito("a despesa recorrente foi alterada por outra pessoa; recarregue e tente de novo")
}

func ErroEmpresaDaRecorrenciaNaoEncontrada() error {
	return domain.ErroValidacao("empresa da despesa recorrente não encontrada")
}

func ErroEmpresaInativaNaRecorrencia() error {
	return domain.ErroValidacao("empresa inativa não recebe despesa recorrente")
}

func ErroDespesaRecorrenteComSolicitacoes() error {
	return domain.ErroValidacao("despesa recorrente já gerou solicitações e não pode ser excluída; desative-a")
}

func ErroDespesaRecorrenteIndisponivel() error {
	return domain.ErroValidacao("despesa recorrente inativa ou excluída")
}

func NovaDespesaRecorrente(empresaID uuid.UUID, dados DadosDespesaRecorrente) (*DespesaRecorrente, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa da despesa recorrente não informada")
	}

	agora := time.Now().UTC()
	despesa := &DespesaRecorrente{
		ID:        uuid.New(),
		EmpresaID: empresaID,
		Ativa:     true,
		Versao:    versaoInicial,
		CriadoEm:  agora,
	}
	if err := despesa.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return despesa, nil
}

func (d *DespesaRecorrente) Alterar(dados DadosDespesaRecorrente, agora time.Time) error {
	if dados.CategoriaFinanceiraID == uuid.Nil {
		return domain.ErroValidacao("categoria financeira é obrigatória na despesa recorrente")
	}
	if err := dados.FormaPagamento.Validado(); err != nil {
		return err
	}
	if dados.Fim != nil && dados.Fim.Inicio().Before(dados.Inicio.Inicio()) {
		return domain.ErroValidacao("fim da despesa recorrente não pode ser anterior ao início")
	}

	d.FornecedorID = dados.FornecedorID
	d.CategoriaFinanceiraID = dados.CategoriaFinanceiraID
	d.Descricao = dados.Descricao
	d.Valor = dados.Valor
	d.Estimado = dados.Estimado
	d.FormaPagamento = dados.FormaPagamento
	d.DiaVencimento = dados.DiaVencimento
	d.VencimentoMesSeguinte = dados.VencimentoMesSeguinte
	d.Inicio = dados.Inicio
	d.Fim = dados.Fim
	d.AtualizadoEm = agora

	return nil
}

func (d *DespesaRecorrente) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da despesa recorrente é obrigatória")
	}
	if informada != d.Versao {
		return ErroDespesaRecorrenteAlterada()
	}

	return nil
}

func (d *DespesaRecorrente) Ativar(agora time.Time) bool {
	if d.Ativa {
		return false
	}

	d.Ativa = true
	d.AtualizadoEm = agora

	return true
}

func (d *DespesaRecorrente) Desativar(agora time.Time) bool {
	if !d.Ativa {
		return false
	}

	d.Ativa = false
	d.AtualizadoEm = agora

	return true
}

func (d *DespesaRecorrente) VigenteEm(competencia Competencia) bool {
	if !d.Ativa || competencia.Inicio().Before(d.Inicio.Inicio()) {
		return false
	}

	return d.Fim == nil || !competencia.Inicio().After(d.Fim.Inicio())
}

func (d *DespesaRecorrente) VencimentoNa(competencia Competencia) PrazoPagamento {
	mes := competencia.Inicio()
	if d.VencimentoMesSeguinte {
		mes = mes.AddDate(0, 1, 0)
	}

	ultimoDia := mes.AddDate(0, 1, -1).Day()
	dia := min(d.DiaVencimento.Dia(), ultimoDia)

	return PrazoPagamentoDe(time.Date(mes.Year(), mes.Month(), dia, 0, 0, 0, 0, time.UTC))
}

func (d *DespesaRecorrente) GerarSolicitacao(solicitanteID uuid.UUID, competencia Competencia, agora time.Time) (*Solicitacao, error) {
	if !d.VigenteEm(competencia) {
		return nil, domain.ErroValidacao("despesa recorrente fora da vigência na competência " + competencia.String())
	}

	origemID := d.ID
	categoriaID := d.CategoriaFinanceiraID

	return &Solicitacao{
		ID:                    uuid.New(),
		SolicitanteID:         solicitanteID,
		EmpresaID:             d.EmpresaID,
		Valor:                 d.Valor,
		Prazo:                 d.VencimentoNa(competencia),
		Observacao:            d.Descricao,
		FormaPagamento:        d.FormaPagamento,
		FornecedorID:          d.FornecedorID,
		CategoriaFinanceiraID: &categoriaID,
		Competencia:           competencia,
		Estimado:              d.Estimado,
		Origem:                OrigemRecorrencia,
		OrigemID:              &origemID,
		Status:                StatusPendenteAprovacao,
		Versao:                versaoInicial,
		CriadoEm:              agora,
		AtualizadoEm:          agora,
	}, nil
}
