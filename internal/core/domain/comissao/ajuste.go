package comissao

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoMotivo = 1000

type Ajuste struct {
	ID              uuid.UUID
	EmpresaID       uuid.UUID
	RepresentanteID uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     Competencia
	ValorCentavos   int64
	Motivo          string
	UsuarioID       uuid.UUID
	CriadoEm        time.Time
}

type DadosAjuste struct {
	EmpresaID       uuid.UUID
	RepresentanteID uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     string
	ValorCentavos   int64
	Motivo          string
}

func NovoAjuste(dados DadosAjuste, usuarioID uuid.UUID, agora time.Time) (*Ajuste, error) {
	switch {
	case dados.EmpresaID == uuid.Nil:
		return nil, domain.ErroValidacao("empresa do ajuste é obrigatória")
	case dados.RepresentanteID == uuid.Nil:
		return nil, domain.ErroValidacao("representante do ajuste é obrigatório")
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário não informado")
	case dados.ValorCentavos == 0:
		return nil, domain.ErroValidacao("valor do ajuste não pode ser zero")
	}

	competencia, err := NovaCompetencia(dados.Competencia)
	if err != nil {
		return nil, err
	}
	if competencia.Inicio().After(CompetenciaDe(Hoje(agora)).Inicio()) {
		return nil, domain.ErroValidacao("competência do ajuste não pode ser posterior ao mês corrente")
	}

	motivo := strings.TrimSpace(dados.Motivo)
	if motivo == "" {
		return nil, domain.ErroValidacao("motivo do ajuste é obrigatório")
	}
	if utf8.RuneCountInString(motivo) > tamanhoMaximoMotivo {
		return nil, domain.ErroValidacao("motivo do ajuste deve ter no máximo 1000 caracteres")
	}

	var pedidoID *uuid.UUID
	if dados.PedidoID != nil && *dados.PedidoID != uuid.Nil {
		copia := *dados.PedidoID
		pedidoID = &copia
	}

	return &Ajuste{
		ID:              uuid.New(),
		EmpresaID:       dados.EmpresaID,
		RepresentanteID: dados.RepresentanteID,
		PedidoID:        pedidoID,
		Competencia:     competencia,
		ValorCentavos:   dados.ValorCentavos,
		Motivo:          motivo,
		UsuarioID:       usuarioID,
		CriadoEm:        agora,
	}, nil
}
