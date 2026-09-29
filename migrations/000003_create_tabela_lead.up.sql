CREATE TABLE lead (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(200) NOT NULL,
    email VARCHAR(255) NOT NULL DEFAULT '',
    telefone VARCHAR(40) NOT NULL DEFAULT '',
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    origem VARCHAR(80) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    etapa_id UUID NOT NULL,
    CONSTRAINT pk_lead PRIMARY KEY (id),
    CONSTRAINT chk_lead_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_lead_email CHECK (email = '' OR email LIKE '%_@_%'),
    CONSTRAINT fk_lead_etapa FOREIGN KEY (etapa_id) REFERENCES etapa (etapa_id) ON DELETE RESTRICT
);

CREATE INDEX idx_lead_etapa ON lead (etapa_id);

CREATE INDEX idx_lead_ativos_criado_em ON lead (criado_em DESC) WHERE ativo = TRUE;
