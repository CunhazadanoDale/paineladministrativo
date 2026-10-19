package caixa

import (
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type SaldoContaResponse struct {
	ContaBancariaID        uuid.UUID `json:"conta_bancaria_id"`
	EmpresaID              uuid.UUID `json:"empresa_id"`
	Apelido                string    `json:"apelido"`
	Banco                  string    `json:"banco"`
	BancoNome              string    `json:"banco_nome"`
	Ativa                  bool      `json:"ativa"`
	DataCorte              *string   `json:"data_corte"`
	SaldoInicialCentavos   *int64    `json:"saldo_inicial_centavos"`
	SaldoAtualCentavos     *int64    `json:"saldo_atual_centavos"`
	UltimoLancamentoEm     *string   `json:"ultimo_lancamento_em"`
	SaldoBancoCentavos     *int64    `json:"saldo_banco_centavos"`
	SaldoBancoEm           *string   `json:"saldo_banco_em"`
	DiferencaBancoCentavos *int64    `json:"diferenca_banco_centavos"`
}

func NovasSaldoContaResponses(saldos []*domaincaixa.SaldoConta) []SaldoContaResponse {
	respostas := make([]SaldoContaResponse, 0, len(saldos))
	for _, saldo := range saldos {
		respostas = append(respostas, SaldoContaResponse{
			ContaBancariaID:        saldo.ContaBancariaID,
			EmpresaID:              saldo.EmpresaID,
			Apelido:                saldo.Apelido,
			Banco:                  saldo.Banco.String(),
			BancoNome:              saldo.Banco.Nome(),
			Ativa:                  saldo.Ativa,
			DataCorte:              formatarDataOpcional(saldo.DataCorte),
			SaldoInicialCentavos:   saldo.SaldoInicialCentavos,
			SaldoAtualCentavos:     saldo.SaldoAtualCentavos(),
			UltimoLancamentoEm:     formatarDataOpcional(saldo.UltimoLancamentoEm),
			SaldoBancoCentavos:     saldo.SaldoBancoCentavos,
			SaldoBancoEm:           formatarDataOpcional(saldo.SaldoBancoEm),
			DiferencaBancoCentavos: saldo.DiferencaDoBancoCentavos(),
		})
	}

	return respostas
}
