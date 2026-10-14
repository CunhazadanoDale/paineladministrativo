CREATE TABLE cliente_obra (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    cliente_id UUID NOT NULL,
    nome VARCHAR(160) NOT NULL,
    cnpj VARCHAR(14),
    logradouro VARCHAR(200) NOT NULL,
    numero VARCHAR(20),
    complemento VARCHAR(120),
    bairro VARCHAR(120),
    cidade VARCHAR(120) NOT NULL,
    uf CHAR(2) NOT NULL,
    cep CHAR(8),
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_cliente_obra PRIMARY KEY (id),
    CONSTRAINT chk_cliente_obra_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_cliente_obra_cnpj CHECK (cnpj IS NULL OR cnpj ~ '^[0-9A-Z]{12}[0-9]{2}$'),
    CONSTRAINT chk_cliente_obra_logradouro CHECK (btrim(logradouro) <> ''),
    CONSTRAINT chk_cliente_obra_cidade CHECK (btrim(cidade) <> ''),
    CONSTRAINT chk_cliente_obra_uf CHECK (uf IN (
        'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA',
        'PB', 'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
    )),
    CONSTRAINT chk_cliente_obra_cep CHECK (cep IS NULL OR cep ~ '^[0-9]{8}$'),
    CONSTRAINT fk_cliente_obra_cliente FOREIGN KEY (cliente_id) REFERENCES cliente (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_cliente_obra_nome ON cliente_obra (cliente_id, lower(nome));
CREATE INDEX idx_cliente_obra_cliente ON cliente_obra (cliente_id, ativo, nome, id);
