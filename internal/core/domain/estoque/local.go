package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoNomeLocal = 120

type Local struct {
	ID           uuid.UUID
	EmpresaID    uuid.UUID
	Nome         string
	Ativo        bool
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

func NovoLocal(empresaID uuid.UUID, nome string) (*Local, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa do local não informada")
	}

	nome, err := validarNomeLocal(nome)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &Local{
		ID:           uuid.New(),
		EmpresaID:    empresaID,
		Nome:         nome,
		Ativo:        true,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}, nil
}

func (l *Local) Renomear(nome string, agora time.Time) error {
	nome, err := validarNomeLocal(nome)
	if err != nil {
		return err
	}

	l.Nome = nome
	l.AtualizadoEm = agora

	return nil
}

func (l *Local) Ativar(agora time.Time) {
	l.Ativo = true
	l.AtualizadoEm = agora
}

func (l *Local) Desativar(possuiSaldo bool, agora time.Time) error {
	if possuiSaldo {
		return domain.ErroValidacao("local com saldo não pode ser desativado; transfira ou baixe o saldo antes")
	}

	l.Ativo = false
	l.AtualizadoEm = agora

	return nil
}

func validarNomeLocal(nome string) (string, error) {
	nome = strings.TrimSpace(nome)

	switch {
	case nome == "":
		return nome, domain.ErroValidacao("nome do local é obrigatório")
	case utf8.RuneCountInString(nome) > tamanhoMaximoNomeLocal:
		return nome, domain.ErroValidacao("nome do local deve ter no máximo 120 caracteres")
	}

	return nome, nil
}
