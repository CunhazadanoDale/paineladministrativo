CREATE TABLE categoria (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(120) NOT NULL,
    categoria_pai_id UUID,
    slug VARCHAR(220) NOT NULL,
    ordem INT NOT NULL DEFAULT 0,
    icone VARCHAR(60) NOT NULL DEFAULT '',
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_categoria PRIMARY KEY (id),
    CONSTRAINT chk_categoria_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_categoria_slug CHECK (btrim(slug) <> ''),
    CONSTRAINT chk_categoria_ordem CHECK (ordem >= 0),
    CONSTRAINT uq_categoria_slug UNIQUE (slug),
    CONSTRAINT fk_categoria_pai FOREIGN KEY (categoria_pai_id) REFERENCES categoria (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_categoria_raiz_nome ON categoria (nome) WHERE categoria_pai_id IS NULL;
CREATE UNIQUE INDEX uq_categoria_sub_nome ON categoria (categoria_pai_id, nome) WHERE categoria_pai_id IS NOT NULL;
CREATE INDEX idx_categoria_ativo_ordem ON categoria (ativo, ordem);
