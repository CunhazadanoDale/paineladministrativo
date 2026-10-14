CREATE TABLE fornecedor (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pais CHAR(2) NOT NULL DEFAULT 'BR',
    documento VARCHAR(14),
    tipo_pessoa VARCHAR(2),
    identificacao_estrangeira VARCHAR(40),
    nome VARCHAR(200) NOT NULL,
    nome_fantasia VARCHAR(120),
    nome_busca VARCHAR(200) NOT NULL,
    moeda_padrao CHAR(3) NOT NULL DEFAULT 'BRL',
    uf CHAR(2),
    cidade VARCHAR(120),
    contato_nome VARCHAR(160),
    email VARCHAR(255),
    telefone VARCHAR(40),
    observacoes VARCHAR(2000) NOT NULL DEFAULT '',
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_fornecedor PRIMARY KEY (id),
    CONSTRAINT chk_fornecedor_pais CHECK (pais ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_fornecedor_tipo_pessoa CHECK (tipo_pessoa IS NULL OR tipo_pessoa IN ('pf', 'pj')),
    CONSTRAINT chk_fornecedor_tipo_pessoa_documento CHECK ((documento IS NULL) = (tipo_pessoa IS NULL)),
    CONSTRAINT chk_fornecedor_documento CHECK (
        documento IS NULL
        OR (tipo_pessoa = 'pf' AND documento ~ '^[0-9]{11}$')
        OR (tipo_pessoa = 'pj' AND documento ~ '^[0-9A-Z]{12}[0-9]{2}$')
    ),
    CONSTRAINT chk_fornecedor_documento_pais CHECK ((pais = 'BR') = (documento IS NOT NULL)),
    CONSTRAINT chk_fornecedor_identificacao_estrangeira CHECK (
        identificacao_estrangeira IS NULL
        OR (pais <> 'BR' AND btrim(identificacao_estrangeira) <> '')
    ),
    CONSTRAINT chk_fornecedor_endereco_pais CHECK (pais = 'BR' OR (uf IS NULL AND cidade IS NULL)),
    CONSTRAINT chk_fornecedor_moeda_padrao CHECK (moeda_padrao IN ('BRL', 'USD', 'EUR', 'CNY')),
    CONSTRAINT chk_fornecedor_uf CHECK (uf IS NULL OR uf IN (
        'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA',
        'PB', 'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
    )),
    CONSTRAINT chk_fornecedor_cidade CHECK (cidade IS NULL OR btrim(cidade) <> ''),
    CONSTRAINT chk_fornecedor_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_fornecedor_nome_busca CHECK (btrim(nome_busca) <> ''),
    CONSTRAINT chk_fornecedor_email CHECK (email IS NULL OR email LIKE '%_@_%'),
    CONSTRAINT chk_fornecedor_versao CHECK (versao >= 1)
);

CREATE UNIQUE INDEX uq_fornecedor_documento ON fornecedor (documento) WHERE documento IS NOT NULL;
CREATE UNIQUE INDEX uq_fornecedor_identificacao_estrangeira ON fornecedor (pais, identificacao_estrangeira)
    WHERE identificacao_estrangeira IS NOT NULL;
CREATE INDEX idx_fornecedor_nome_busca ON fornecedor (nome_busca, id);
CREATE INDEX idx_fornecedor_ativo ON fornecedor (ativo, nome_busca, id);
