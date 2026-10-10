package helpers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func CriarFunil(t *testing.T, banco *sqlx.DB, nome string) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	if err := banco.Get(
		&id,
		`INSERT INTO funil (nome) VALUES ($1) RETURNING funil_id`,
		nome,
	); err != nil {
		t.Fatalf("não consegui criar o funil %q: %v", nome, err)
	}

	return id
}

func CriarEtapas(t *testing.T, banco *sqlx.DB, funilID uuid.UUID, nomes ...string) []uuid.UUID {
	t.Helper()

	ids := make([]uuid.UUID, 0, len(nomes))
	for posicao, nome := range nomes {
		var id uuid.UUID
		if err := banco.Get(
			&id,
			`INSERT INTO etapa (nome, ordem, funil_id) VALUES ($1, $2, $3) RETURNING etapa_id`,
			nome,
			posicao+1,
			funilID,
		); err != nil {
			t.Fatalf("não consegui criar a etapa %q: %v", nome, err)
		}

		ids = append(ids, id)
	}

	return ids
}

func CriarLead(t *testing.T, banco *sqlx.DB, etapaID uuid.UUID, nome string) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	if err := banco.Get(
		&id,
		`INSERT INTO lead (nome, etapa_id) VALUES ($1, $2) RETURNING id`,
		nome,
		etapaID,
	); err != nil {
		t.Fatalf("não consegui criar o lead %q: %v", nome, err)
	}

	return id
}

func DesativarLead(t *testing.T, banco *sqlx.DB, leadID uuid.UUID) {
	t.Helper()

	if _, err := banco.Exec(
		`UPDATE lead SET ativo = FALSE WHERE id = $1`,
		leadID,
	); err != nil {
		t.Fatalf("não consegui desativar o lead %s: %v", leadID, err)
	}
}
