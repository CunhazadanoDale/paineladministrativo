CREATE TABLE estoque_local (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    nome VARCHAR(120) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_estoque_local PRIMARY KEY (id),
    CONSTRAINT chk_estoque_local_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT fk_estoque_local_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_estoque_local_empresa_nome ON estoque_local (empresa_id, lower(nome));
CREATE INDEX idx_estoque_local_empresa ON estoque_local (empresa_id, nome, id);
