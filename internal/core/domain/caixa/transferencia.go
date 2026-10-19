package caixa

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type Transferencia struct {
	ID                    uuid.UUID
	ContaOrigemID         uuid.UUID
	ContaDestinoID        uuid.UUID
	Data                  time.Time
	ValorCentavos         int64
	EntreEmpresas         bool
	CategoriaFinanceiraID uuid.UUID
	Descricao             string
	Estornado             bool
	UsuarioID             uuid.UUID
	CriadoEm              time.Time
}

type DadosTransferencia struct {
	Lancamento  *LancamentoExtrato
	Conta       *domainfinanceiro.ContaBancaria
	Contraparte *domainfinanceiro.ContaBancaria
	Categoria   *domainfinanceiro.CategoriaFinanceira
	Descricao   string
	UsuarioID   uuid.UUID
	Agora       time.Time
}

type TransferenciaRegistrada struct {
	Transferencia *Transferencia
	Conciliacao   *Conciliacao
}

func NovaTransferencia(dados DadosTransferencia) (*TransferenciaRegistrada, error) {
	linha := dados.Lancamento
	if linha == nil {
		return nil, domain.ErroValidacao("lançamento do extrato não informado")
	}
	if dados.Conta == nil || dados.Conta.ID != linha.ContaBancariaID {
		return nil, domain.ErroValidacao("conta do lançamento não informada")
	}
	if dados.Contraparte == nil {
		return nil, domain.ErroValidacao("conta da outra ponta não encontrada")
	}
	if dados.Contraparte.ID == dados.Conta.ID {
		return nil, domain.ErroValidacao("a outra ponta da transferência precisa ser outra conta")
	}
	if !dados.Contraparte.Ativa {
		return nil, domain.ErroValidacao("conta da outra ponta está inativa")
	}
	if err := exigirCategoriaLancavel(dados.Categoria); err != nil {
		return nil, err
	}
	if dados.Categoria.Natureza != domainfinanceiro.NaturezaTransferencia {
		return nil, domain.ErroValidacao("transferência entre contas só recebe categoria de transferencia")
	}

	transferencia := &Transferencia{
		ID:                    uuid.New(),
		Data:                  linha.Data,
		EntreEmpresas:         dados.Conta.EmpresaID != dados.Contraparte.EmpresaID,
		CategoriaFinanceiraID: dados.Categoria.ID,
		UsuarioID:             dados.UsuarioID,
		CriadoEm:              dados.Agora,
	}

	movimento := &MovimentoSistema{
		ID:              transferencia.ID,
		ContaBancariaID: dados.Conta.ID,
		EmpresaID:       dados.Conta.EmpresaID,
		Data:            linha.Data,
		ValorCentavos:   linha.ValorCentavos,
		OrigemID:        transferencia.ID,
	}
	if linha.Credito() {
		transferencia.ContaOrigemID, transferencia.ContaDestinoID = dados.Contraparte.ID, dados.Conta.ID
		transferencia.ValorCentavos = linha.ValorCentavos
		movimento.Tipo = MovimentoTransferenciaEntrada
	} else {
		transferencia.ContaOrigemID, transferencia.ContaDestinoID = dados.Conta.ID, dados.Contraparte.ID
		transferencia.ValorCentavos = -linha.ValorCentavos
		movimento.Tipo = MovimentoTransferenciaSaida
	}

	transferencia.Descricao = descricaoOuDaLinha(dados.Descricao, linha)
	movimento.Descricao = transferencia.Descricao

	conciliacao, err := NovaConciliacao(DadosConciliacao{
		Lancamento: linha,
		Movimentos: []*MovimentoSistema{movimento},
		Modo:       ConciliacaoTransferencia,
		UsuarioID:  dados.UsuarioID,
		Agora:      dados.Agora,
	})
	if err != nil {
		return nil, err
	}

	return &TransferenciaRegistrada{Transferencia: transferencia, Conciliacao: conciliacao}, nil
}
