CREATE TABLE cargo (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(80) NOT NULL,
    descricao VARCHAR(255) NOT NULL DEFAULT '',
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT pk_cargo PRIMARY KEY (id),
    CONSTRAINT chk_cargo_nome CHECK (btrim(nome) <> '')
);

CREATE INDEX idx_cargo_ativos_nome ON cargo (nome) WHERE ativo = TRUE;
