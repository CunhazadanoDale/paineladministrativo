package caixa

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const versaoInicial = 1

type Abertura struct {
	ContaBancariaID uuid.UUID
	DataCorte       time.Time
	SaldoCentavos   int64
	UsuarioID       uuid.UUID
	Versao          int
	CriadoEm        time.Time
	AtualizadoEm    time.Time
}

type DadosAbertura struct {
	ContaBancariaID uuid.UUID
	DataCorte       time.Time
	SaldoCentavos   int64
	UsuarioID       uuid.UUID
}

func NovaAbertura(dados DadosAbertura, agora time.Time) (*Abertura, error) {
	if dados.ContaBancariaID == uuid.Nil {
		return nil, domain.ErroValidacao("conta bancária do saldo inicial não informada")
	}
	if dados.UsuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário não informado")
	}

	dataCorte, err := validarDataCorte(dados.DataCorte, agora)
	if err != nil {
		return nil, err
	}

	return &Abertura{
		ContaBancariaID: dados.ContaBancariaID,
		DataCorte:       dataCorte,
		SaldoCentavos:   dados.SaldoCentavos,
		UsuarioID:       dados.UsuarioID,
		Versao:          versaoInicial,
		CriadoEm:        agora,
		AtualizadoEm:    agora,
	}, nil
}

func (a *Abertura) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão do saldo inicial é obrigatória")
	}
	if informada != a.Versao {
		return ErroAberturaAlterada()
	}

	return nil
}

func (a *Abertura) Redefinir(dataCorte time.Time, saldoCentavos int64, usuarioID uuid.UUID, agora time.Time) error {
	if usuarioID == uuid.Nil {
		return domain.ErroValidacao("usuário não informado")
	}

	data, err := validarDataCorte(dataCorte, agora)
	if err != nil {
		return err
	}

	a.DataCorte = data
	a.SaldoCentavos = saldoCentavos
	a.UsuarioID = usuarioID
	a.AtualizadoEm = agora

	return nil
}

func (a *Abertura) Abrange(data time.Time) bool {
	return DataDe(data).After(a.DataCorte)
}

func validarDataCorte(valor, agora time.Time) (time.Time, error) {
	if valor.IsZero() {
		return time.Time{}, domain.ErroValidacao("data de corte do saldo inicial é obrigatória")
	}

	data := DataDe(valor)
	if !data.Before(Hoje(agora)) {
		return time.Time{}, domain.ErroValidacao("data de corte do saldo inicial deve ser anterior a hoje")
	}

	return data, nil
}
