package lead

import (
	"time"

	"github.com/google/uuid"
)

type Lead struct {
	ID uuid.UUID `db:"id"`
	Nome  string `db:"nome"`
	Email string `db:"email"`
	Telefone string `db:"telefone"`
	Ativo bool `db:"ativo"`

	Origem string `db:"origem"`
	CriadoEm time.Time `db:"criado_em"`
	AtualizadoEm time.Time `db:"atualizado_em"`

	EtapaID uuid.UUID `db:"etapa_id"`
}

type LeadHistorico struct {
	ID uuid.UUID `db:"id"`
	LeadID uuid.UUID `db:"lead_id"`
	EtapAnteriorID uuid.UUID `db:"etapa_anterior_id"`
	EtapaAtualID uuid.UUID `db:"etapa_atual_id"`
	MovidoEm time.Time `db:"movido_em"`
}