CREATE TABLE etapa (
    etapa_id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(150) NOT NULL,
    ordem INTEGER NOT NULL,
    funil_id UUID NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT pk_etapa PRIMARY KEY (etapa_id),
    CONSTRAINT chk_etapa_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_etapa_ordem CHECK (ordem > 0),
    CONSTRAINT fk_etapa_funil FOREIGN KEY (funil_id) REFERENCES funil (funil_id) ON DELETE CASCADE,
    CONSTRAINT uq_etapa_funil_ordem UNIQUE (funil_id, ordem) DEFERRABLE INITIALLY DEFERRED
);
