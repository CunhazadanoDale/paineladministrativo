package empresa

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoRazaoSocial  = 200
	tamanhoMaximoNomeFantasia = 120
)

type Empresa struct {
	ID               uuid.UUID
	RazaoSocial      string
	NomeFantasia     string
	CNPJ             CNPJ
	UF               UF
	RegimeTributario RegimeTributario
	Ativa            bool
	CriadoEm         time.Time
	AtualizadoEm     time.Time
}

type DadosEmpresa struct {
	RazaoSocial      string
	NomeFantasia     string
	UF               UF
	RegimeTributario RegimeTributario
}

func NovaEmpresa(cnpj CNPJ, dados DadosEmpresa) (*Empresa, error) {
	if cnpj.Vazio() {
		return nil, domain.ErroValidacao("cnpj da empresa é obrigatório")
	}

	dados, err := validarDados(dados)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()

	return &Empresa{
		ID:               uuid.New(),
		RazaoSocial:      dados.RazaoSocial,
		NomeFantasia:     dados.NomeFantasia,
		CNPJ:             cnpj,
		UF:               dados.UF,
		RegimeTributario: dados.RegimeTributario,
		Ativa:            true,
		CriadoEm:         agora,
		AtualizadoEm:     agora,
	}, nil
}

func (e *Empresa) Alterar(dados DadosEmpresa, agora time.Time) error {
	dados, err := validarDados(dados)
	if err != nil {
		return err
	}

	e.RazaoSocial = dados.RazaoSocial
	e.NomeFantasia = dados.NomeFantasia
	e.UF = dados.UF
	e.RegimeTributario = dados.RegimeTributario
	e.AtualizadoEm = agora

	return nil
}

func (e *Empresa) Ativar(agora time.Time) {
	e.Ativa = true
	e.AtualizadoEm = agora
}

func (e *Empresa) Desativar(agora time.Time) {
	e.Ativa = false
	e.AtualizadoEm = agora
}

func validarDados(dados DadosEmpresa) (DadosEmpresa, error) {
	dados.RazaoSocial = strings.TrimSpace(dados.RazaoSocial)
	dados.NomeFantasia = strings.TrimSpace(dados.NomeFantasia)

	switch {
	case dados.RazaoSocial == "":
		return dados, domain.ErroValidacao("razão social da empresa é obrigatória")
	case utf8.RuneCountInString(dados.RazaoSocial) > tamanhoMaximoRazaoSocial:
		return dados, domain.ErroValidacao("razão social da empresa deve ter no máximo 200 caracteres")
	case dados.NomeFantasia == "":
		return dados, domain.ErroValidacao("nome fantasia da empresa é obrigatório")
	case utf8.RuneCountInString(dados.NomeFantasia) > tamanhoMaximoNomeFantasia:
		return dados, domain.ErroValidacao("nome fantasia da empresa deve ter no máximo 120 caracteres")
	}

	if err := dados.UF.Validado(); err != nil {
		return dados, err
	}
	if err := dados.RegimeTributario.Validado(); err != nil {
		return dados, err
	}

	return dados, nil
}
