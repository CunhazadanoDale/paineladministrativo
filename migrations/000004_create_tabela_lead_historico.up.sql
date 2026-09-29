CREATE TABLE lead_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL,
    etapa_anterior_id UUID NOT NULL,
    etapa_atual_id UUID NOT NULL,
    movido_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_lead_historico PRIMARY KEY (id),
    CONSTRAINT chk_lead_historico_etapas CHECK (etapa_anterior_id <> etapa_atual_id),
    CONSTRAINT fk_lead_historico_lead FOREIGN KEY (lead_id) REFERENCES lead (id) ON DELETE CASCADE
);

CREATE INDEX idx_lead_historico_lead ON lead_historico (lead_id, movido_em DESC);
