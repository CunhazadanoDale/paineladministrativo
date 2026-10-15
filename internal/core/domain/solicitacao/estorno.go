package solicitacao

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type MotivoEstorno struct {
	texto string
}

func NovoMotivoEstorno(texto string) (MotivoEstorno, error) {
	texto = strings.TrimSpace(texto)

	if texto == "" {
		return MotivoEstorno{}, domain.ErroValidacao("motivo do estorno é obrigatório")
	}
	if utf8.RuneCountInString(texto) > tamanhoMaximoMotivo {
		return MotivoEstorno{}, domain.ErroValidacao("motivo do estorno deve ter no máximo 500 caracteres")
	}

	return MotivoEstorno{texto: texto}, nil
}

func MotivoEstornoDe(texto string) MotivoEstorno {
	return MotivoEstorno{texto: texto}
}

func (m MotivoEstorno) Texto() string {
	return m.texto
}

type Estorno struct {
	ID          uuid.UUID
	PagamentoID uuid.UUID
	Motivo      MotivoEstorno
	EstornadoEm time.Time
	UsuarioID   uuid.UUID
	CriadoEm    time.Time
}

func (p *Pagamento) Estornar(motivo MotivoEstorno, usuarioID uuid.UUID, estornadoEm, agora time.Time) (*Estorno, error) {
	if p.Estornado {
		return nil, domain.ErroConflito("pagamento já estornado")
	}
	if estornadoEm.IsZero() {
		estornadoEm = agora
	}
	if estornadoEm.After(agora) {
		return nil, domain.ErroValidacao("data do estorno não pode estar no futuro")
	}
	if estornadoEm.Before(p.PagoEm) {
		return nil, domain.ErroValidacao("data do estorno não pode ser anterior ao pagamento")
	}

	estorno := &Estorno{
		ID:          uuid.New(),
		PagamentoID: p.ID,
		Motivo:      motivo,
		EstornadoEm: estornadoEm,
		UsuarioID:   usuarioID,
		CriadoEm:    agora,
	}
	p.Estornado = true
	p.Estorno = estorno

	return estorno, nil
}

func (s *Solicitacao) Estornar(agora time.Time) error {
	if s.Status != StatusPago {
		return domain.ErroConflito("só é possível estornar solicitação paga")
	}

	s.Status = StatusAprovado
	s.Versao++
	s.AtualizadoEm = agora

	return nil
}
