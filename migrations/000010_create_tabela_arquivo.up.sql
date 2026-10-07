CREATE TABLE arquivo (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    proprietario_id UUID NOT NULL,
    nome VARCHAR(255) NOT NULL,
    chave VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    tamanho BIGINT NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_arquivo PRIMARY KEY (id),
    CONSTRAINT chk_arquivo_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_arquivo_tamanho CHECK (tamanho > 0),
    CONSTRAINT uq_arquivo_chave UNIQUE (chave),
    CONSTRAINT fk_arquivo_proprietario FOREIGN KEY (proprietario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_arquivo_proprietario ON arquivo (proprietario_id);
