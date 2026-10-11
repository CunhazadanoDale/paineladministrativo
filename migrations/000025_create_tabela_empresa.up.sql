CREATE TABLE empresa (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    razao_social VARCHAR(200) NOT NULL,
    nome_fantasia VARCHAR(120) NOT NULL,
    cnpj CHAR(14) NOT NULL,
    uf CHAR(2) NOT NULL,
    regime_tributario VARCHAR(10) NOT NULL,
    ativa BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_empresa PRIMARY KEY (id),
    CONSTRAINT chk_empresa_razao_social CHECK (btrim(razao_social) <> ''),
    CONSTRAINT chk_empresa_nome_fantasia CHECK (btrim(nome_fantasia) <> ''),
    CONSTRAINT chk_empresa_cnpj CHECK (cnpj ~ '^[0-9A-Z]{12}[0-9]{2}$'),
    CONSTRAINT chk_empresa_uf CHECK (uf IN (
        'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA',
        'PB', 'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
    )),
    CONSTRAINT chk_empresa_regime_tributario CHECK (regime_tributario IN ('simples', 'presumido', 'real')),
    CONSTRAINT uq_empresa_cnpj UNIQUE (cnpj)
);

CREATE INDEX idx_empresa_ativa_nome ON empresa (nome_fantasia) WHERE ativa = TRUE;
