package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type DadosSolicitacaoDeOrigem struct {
	EmpresaID             uuid.UUID
	SolicitanteID         uuid.UUID
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID uuid.UUID
	Origem                Origem
	OrigemID              uuid.UUID
	ValorCentavos         int64
	Vencimento            time.Time
	Competencia           *Competencia
	Observacao            string
	FormaPagamento        FormaPagamento
}

func NovaSolicitacaoDeOrigem(dados DadosSolicitacaoDeOrigem, agora time.Time) (*Solicitacao, *Historico, error) {
	switch {
	case !dados.Origem.Sistemica():
		return nil, nil, domain.ErroValidacao("origem da conta a pagar gerada deve ser recompra, repasse_fomento, comissao, importacao_parcela, importacao_custo, divida_amortizacao ou divida_juros")
	case dados.OrigemID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("conta a pagar gerada precisa do registro de origem")
	case dados.EmpresaID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("empresa da solicitação não informada")
	case dados.SolicitanteID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("solicitante da solicitação não informado")
	case dados.CategoriaFinanceiraID == uuid.Nil:
		return nil, nil, domain.ErroValidacao("conta a pagar gerada precisa de categoria financeira")
	case dados.Vencimento.IsZero():
		return nil, nil, domain.ErroValidacao("prazo de pagamento é obrigatório")
	}

	valor, err := NovoValor(dados.ValorCentavos)
	if err != nil {
		return nil, nil, err
	}
	observacao, err := NovaObservacao(dados.Observacao)
	if err != nil {
		return nil, nil, err
	}
	forma := dados.FormaPagamento
	if forma == "" {
		forma = FormaPagamentoTransferencia
	}
	if err := forma.Validado(); err != nil {
		return nil, nil, err
	}

	prazo := PrazoPagamentoDe(inicioDoDia(dados.Vencimento))
	competencia := CompetenciaDe(prazo.Data())
	if dados.Competencia != nil {
		competencia = *dados.Competencia
	}

	origemID := dados.OrigemID
	categoriaID := dados.CategoriaFinanceiraID
	solicitacao := &Solicitacao{
		ID:                    uuid.New(),
		SolicitanteID:         dados.SolicitanteID,
		EmpresaID:             dados.EmpresaID,
		Valor:                 valor,
		Prazo:                 prazo,
		Observacao:            observacao,
		FormaPagamento:        forma,
		FornecedorID:          dados.FornecedorID,
		CategoriaFinanceiraID: &categoriaID,
		Competencia:           competencia,
		Origem:                dados.Origem,
		OrigemID:              &origemID,
		Status:                StatusPendenteAprovacao,
		Versao:                versaoInicial,
		CriadoEm:              agora,
		AtualizadoEm:          agora,
	}
	historico := NovoHistorico(solicitacao.ID, dados.SolicitanteID, nil, solicitacao.Status, EventoCriada,
		"solicitação gerada por "+dados.Origem.String())

	return solicitacao, historico, nil
}
