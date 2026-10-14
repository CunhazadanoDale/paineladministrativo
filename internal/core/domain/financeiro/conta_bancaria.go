package financeiro

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoApelido = 60

type ContaBancaria struct {
	ID           uuid.UUID
	EmpresaID    uuid.UUID
	Banco        Banco
	Tipo         TipoConta
	Agencia      NumeroBancario
	Conta        NumeroBancario
	Apelido      string
	Ativa        bool
	Versao       int
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

type DadosContaBancaria struct {
	EmpresaID uuid.UUID
	Banco     Banco
	Tipo      TipoConta
	Agencia   NumeroBancario
	Conta     NumeroBancario
	Apelido   string
}

func NovaContaBancaria(dados DadosContaBancaria) (*ContaBancaria, error) {
	if dados.EmpresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa da conta bancária não informada")
	}
	if err := dados.Banco.Validado(); err != nil {
		return nil, err
	}
	if err := dados.Tipo.Validado(); err != nil {
		return nil, err
	}
	if dados.Agencia.Vazio() {
		return nil, domain.ErroValidacao("agência é obrigatória")
	}
	if dados.Conta.Vazio() {
		return nil, domain.ErroValidacao("conta é obrigatória")
	}

	apelido, err := textoObrigatorio(dados.Apelido, "apelido da conta bancária", tamanhoMaximoApelido)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &ContaBancaria{
		ID:           uuid.New(),
		EmpresaID:    dados.EmpresaID,
		Banco:        dados.Banco,
		Tipo:         dados.Tipo,
		Agencia:      dados.Agencia,
		Conta:        dados.Conta,
		Apelido:      apelido,
		Ativa:        true,
		Versao:       versaoInicial,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

func ErroContaBancariaAlterada() error {
	return domain.ErroConflito("a conta bancária foi alterada por outra pessoa; recarregue e tente de novo")
}

func (c *ContaBancaria) ConferirVersao(informada int) error {
	return conferirVersao(informada, c.Versao, "da conta bancária", ErroContaBancariaAlterada())
}

func (c *ContaBancaria) AlterarApelido(apelido string, agora time.Time) error {
	apelido, err := textoObrigatorio(apelido, "apelido da conta bancária", tamanhoMaximoApelido)
	if err != nil {
		return err
	}

	c.Apelido = apelido
	c.AtualizadoEm = agora

	return nil
}

func (c *ContaBancaria) Ativar(agora time.Time) bool {
	if c.Ativa {
		return false
	}

	c.Ativa = true
	c.AtualizadoEm = agora

	return true
}

func (c *ContaBancaria) Desativar(agora time.Time) bool {
	if !c.Ativa {
		return false
	}

	c.Ativa = false
	c.AtualizadoEm = agora

	return true
}
