CREATE TABLE usuario (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(200) NOT NULL,
    email VARCHAR(255) NOT NULL,
    senha VARCHAR(255) NOT NULL,
    cargo_id UUID NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    ultimo_login TIMESTAMPTZ NOT NULL DEFAULT now(),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_usuario PRIMARY KEY (id),
    CONSTRAINT chk_usuario_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_usuario_email CHECK (email LIKE '%_@_%'),
    CONSTRAINT uq_usuario_email UNIQUE (email),
    CONSTRAINT fk_usuario_cargo FOREIGN KEY (cargo_id) REFERENCES cargo (id) ON DELETE RESTRICT
);

CREATE INDEX idx_usuario_cargo ON usuario (cargo_id);

CREATE INDEX idx_usuario_ativos_nome ON usuario (nome) WHERE ativo = TRUE;
