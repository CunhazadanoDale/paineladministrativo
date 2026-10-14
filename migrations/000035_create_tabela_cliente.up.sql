CREATE TABLE cliente (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    documento VARCHAR(14) NOT NULL,
    tipo_pessoa VARCHAR(2) NOT NULL,
    nome VARCHAR(200) NOT NULL,
    nome_fantasia VARCHAR(120),
    nome_busca VARCHAR(200) NOT NULL,
    tipo VARCHAR(12) NOT NULL,
    grupo_id UUID,
    uf CHAR(2) NOT NULL,
    cidade VARCHAR(120) NOT NULL,
    empresa_relacionamento_id UUID,
    representante_id UUID,
    email_nf VARCHAR(255),
    telefone VARCHAR(40),
    observacoes VARCHAR(2000) NOT NULL DEFAULT '',
    exige_numero_pedido_na_nf BOOLEAN NOT NULL DEFAULT FALSE,
    proibe_antecipacao BOOLEAN NOT NULL DEFAULT FALSE,
    lead_id UUID,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_cliente PRIMARY KEY (id),
    CONSTRAINT chk_cliente_tipo_pessoa CHECK (tipo_pessoa IN ('pf', 'pj')),
    CONSTRAINT chk_cliente_documento CHECK (
        (tipo_pessoa = 'pf' AND documento ~ '^[0-9]{11}$')
        OR (tipo_pessoa = 'pj' AND documento ~ '^[0-9A-Z]{12}[0-9]{2}$')
    ),
    CONSTRAINT chk_cliente_tipo CHECK (tipo IN ('construtora', 'loja', 'pf', 'outro')),
    CONSTRAINT chk_cliente_tipo_pf CHECK ((tipo = 'pf') = (tipo_pessoa = 'pf')),
    CONSTRAINT chk_cliente_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_cliente_nome_busca CHECK (btrim(nome_busca) <> ''),
    CONSTRAINT chk_cliente_cidade CHECK (btrim(cidade) <> ''),
    CONSTRAINT chk_cliente_uf CHECK (uf IN (
        'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA',
        'PB', 'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
    )),
    CONSTRAINT chk_cliente_email_nf CHECK (email_nf IS NULL OR email_nf LIKE '%_@_%'),
    CONSTRAINT uq_cliente_documento UNIQUE (documento),
    CONSTRAINT fk_cliente_grupo FOREIGN KEY (grupo_id) REFERENCES cliente_grupo (id) ON DELETE RESTRICT,
    CONSTRAINT fk_cliente_empresa_relacionamento FOREIGN KEY (empresa_relacionamento_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_cliente_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_cliente_lead FOREIGN KEY (lead_id) REFERENCES lead (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_cliente_lead ON cliente (lead_id) WHERE lead_id IS NOT NULL;
CREATE INDEX idx_cliente_nome_busca ON cliente (nome_busca, id);
CREATE INDEX idx_cliente_grupo ON cliente (grupo_id) WHERE grupo_id IS NOT NULL;
CREATE INDEX idx_cliente_representante ON cliente (representante_id, nome_busca, id) WHERE representante_id IS NOT NULL;
CREATE INDEX idx_cliente_empresa_relacionamento ON cliente (empresa_relacionamento_id) WHERE empresa_relacionamento_id IS NOT NULL;
