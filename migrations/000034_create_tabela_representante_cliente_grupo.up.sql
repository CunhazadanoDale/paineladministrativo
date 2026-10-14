CREATE TABLE cliente_grupo (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(120) NOT NULL,
    nome_busca VARCHAR(120) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_cliente_grupo PRIMARY KEY (id),
    CONSTRAINT chk_cliente_grupo_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_cliente_grupo_nome_busca CHECK (btrim(nome_busca) <> ''),
    CONSTRAINT uq_cliente_grupo_nome_busca UNIQUE (nome_busca)
);

CREATE INDEX idx_cliente_grupo_ativo ON cliente_grupo (ativo, nome_busca, id);

CREATE TABLE representante (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(160) NOT NULL,
    nome_busca VARCHAR(160) NOT NULL,
    documento VARCHAR(14),
    usuario_id UUID,
    email VARCHAR(255),
    telefone VARCHAR(40),
    comissao_pronta_entrega_bp INT NOT NULL DEFAULT 0,
    comissao_importacao_bp INT NOT NULL DEFAULT 0,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_representante PRIMARY KEY (id),
    CONSTRAINT chk_representante_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_representante_documento CHECK (
        documento IS NULL OR documento ~ '^[0-9]{11}$' OR documento ~ '^[0-9A-Z]{12}[0-9]{2}$'
    ),
    CONSTRAINT chk_representante_email CHECK (email IS NULL OR email LIKE '%_@_%'),
    CONSTRAINT chk_representante_comissao_pronta_entrega CHECK (comissao_pronta_entrega_bp BETWEEN 0 AND 10000),
    CONSTRAINT chk_representante_comissao_importacao CHECK (comissao_importacao_bp BETWEEN 0 AND 10000),
    CONSTRAINT fk_representante_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_representante_documento ON representante (documento) WHERE documento IS NOT NULL;
CREATE UNIQUE INDEX uq_representante_usuario ON representante (usuario_id) WHERE usuario_id IS NOT NULL;
CREATE INDEX idx_representante_ativo ON representante (ativo, nome_busca, id);
