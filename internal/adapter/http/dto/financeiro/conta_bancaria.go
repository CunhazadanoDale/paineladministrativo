package financeiro

import (
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

type CriarContaBancariaRequest struct {
	EmpresaID     uuid.UUID `json:"empresa_id"`
	Banco         string    `json:"banco"`
	Tipo          string    `json:"tipo"`
	Agencia       string    `json:"agencia"`
	AgenciaDigito *string   `json:"agencia_digito"`
	Conta         string    `json:"conta"`
	ContaDigito   *string   `json:"conta_digito"`
	Apelido       string    `json:"apelido"`
}

func (r CriarContaBancariaRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarContaBancariaInput {
	return portsin.CriarContaBancariaInput{
		UsuarioID:     usuarioID,
		EmpresaID:     r.EmpresaID,
		Banco:         r.Banco,
		Tipo:          r.Tipo,
		Agencia:       r.Agencia,
		AgenciaDigito: r.AgenciaDigito,
		Conta:         r.Conta,
		ContaDigito:   r.ContaDigito,
		Apelido:       r.Apelido,
	}
}

type AlterarContaBancariaRequest struct {
	Apelido string `json:"apelido"`
	Versao  int    `json:"versao"`
}

type SituacaoRequest struct {
	Versao int `json:"versao"`
}

type ContaBancariaResponse struct {
	ID               uuid.UUID `json:"id"`
	EmpresaID        uuid.UUID `json:"empresa_id"`
	Banco            string    `json:"banco"`
	BancoNome        string    `json:"banco_nome"`
	Tipo             string    `json:"tipo"`
	Agencia          string    `json:"agencia"`
	AgenciaDigito    *string   `json:"agencia_digito"`
	AgenciaFormatada string    `json:"agencia_formatada"`
	Conta            string    `json:"conta"`
	ContaDigito      *string   `json:"conta_digito"`
	ContaFormatada   string    `json:"conta_formatada"`
	Apelido          string    `json:"apelido"`
	Ativa            bool      `json:"ativa"`
	Versao           int       `json:"versao"`
	CriadoEm         time.Time `json:"criado_em"`
	AtualizadoEm     time.Time `json:"atualizado_em"`
}

func NovaContaBancariaResponse(conta *domainfinanceiro.ContaBancaria) ContaBancariaResponse {
	return ContaBancariaResponse{
		ID:               conta.ID,
		EmpresaID:        conta.EmpresaID,
		Banco:            conta.Banco.String(),
		BancoNome:        conta.Banco.Nome(),
		Tipo:             conta.Tipo.String(),
		Agencia:          conta.Agencia.Numero(),
		AgenciaDigito:    conta.Agencia.Digito(),
		AgenciaFormatada: conta.Agencia.Formatado(),
		Conta:            conta.Conta.Numero(),
		ContaDigito:      conta.Conta.Digito(),
		ContaFormatada:   conta.Conta.Formatado(),
		Apelido:          conta.Apelido,
		Ativa:            conta.Ativa,
		Versao:           conta.Versao,
		CriadoEm:         conta.CriadoEm,
		AtualizadoEm:     conta.AtualizadoEm,
	}
}

func NovasContaBancariaResponses(contas []*domainfinanceiro.ContaBancaria) []ContaBancariaResponse {
	respostas := make([]ContaBancariaResponse, 0, len(contas))
	for _, conta := range contas {
		respostas = append(respostas, NovaContaBancariaResponse(conta))
	}

	return respostas
}

type BancoResponse struct {
	Codigo string `json:"codigo"`
	Nome   string `json:"nome"`
}

func NovosBancoResponses(bancos []domainfinanceiro.Banco) []BancoResponse {
	respostas := make([]BancoResponse, 0, len(bancos))
	for _, banco := range bancos {
		respostas = append(respostas, BancoResponse{Codigo: banco.String(), Nome: banco.Nome()})
	}

	return respostas
}
