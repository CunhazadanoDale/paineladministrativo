package caixa

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

type DefinirAberturaRequest struct {
	DataCorte     string `json:"data_corte"`
	SaldoCentavos *int64 `json:"saldo_centavos"`
	Versao        *int   `json:"versao"`
}

func (r DefinirAberturaRequest) ParaInput(usuarioID, contaBancariaID uuid.UUID) (portsin.DefinirAberturaInput, error) {
	dataCorte, err := ParaData(r.DataCorte, "data_corte")
	if err != nil {
		return portsin.DefinirAberturaInput{}, err
	}
	if r.SaldoCentavos == nil {
		return portsin.DefinirAberturaInput{}, domain.ErroValidacao("saldo_centavos é obrigatório")
	}

	return portsin.DefinirAberturaInput{
		UsuarioID:       usuarioID,
		ContaBancariaID: contaBancariaID,
		DataCorte:       dataCorte,
		SaldoCentavos:   *r.SaldoCentavos,
		Versao:          r.Versao,
	}, nil
}

type AberturaResponse struct {
	ContaBancariaID uuid.UUID `json:"conta_bancaria_id"`
	DataCorte       string    `json:"data_corte"`
	SaldoCentavos   int64     `json:"saldo_centavos"`
	UsuarioID       uuid.UUID `json:"usuario_id"`
	Versao          int       `json:"versao"`
	CriadoEm        time.Time `json:"criado_em"`
	AtualizadoEm    time.Time `json:"atualizado_em"`
}

func NovaAberturaResponse(abertura *domaincaixa.Abertura) AberturaResponse {
	return AberturaResponse{
		ContaBancariaID: abertura.ContaBancariaID,
		DataCorte:       formatarData(abertura.DataCorte),
		SaldoCentavos:   abertura.SaldoCentavos,
		UsuarioID:       abertura.UsuarioID,
		Versao:          abertura.Versao,
		CriadoEm:        abertura.CriadoEm,
		AtualizadoEm:    abertura.AtualizadoEm,
	}
}
