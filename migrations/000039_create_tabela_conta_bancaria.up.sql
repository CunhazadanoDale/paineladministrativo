CREATE TABLE conta_bancaria (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    banco CHAR(3) NOT NULL,
    tipo VARCHAR(10) NOT NULL,
    agencia VARCHAR(5) NOT NULL,
    agencia_digito VARCHAR(1),
    conta VARCHAR(20) NOT NULL,
    conta_digito VARCHAR(2),
    apelido VARCHAR(60) NOT NULL,
    ativa BOOLEAN NOT NULL DEFAULT TRUE,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_conta_bancaria PRIMARY KEY (id),
    CONSTRAINT chk_conta_bancaria_banco CHECK (banco IN ('001', '033', '077', '237', '341', '403', '748')),
    CONSTRAINT chk_conta_bancaria_tipo CHECK (tipo IN ('corrente', 'poupanca', 'pagamento', 'aplicacao')),
    CONSTRAINT chk_conta_bancaria_agencia CHECK (agencia ~ '^[0-9]{1,5}$' AND agencia ~ '[1-9]'),
    CONSTRAINT chk_conta_bancaria_agencia_digito CHECK (agencia_digito IS NULL OR agencia_digito ~ '^[0-9X]$'),
    CONSTRAINT chk_conta_bancaria_conta CHECK (conta ~ '^[0-9]{1,20}$' AND conta ~ '[1-9]'),
    CONSTRAINT chk_conta_bancaria_conta_digito CHECK (conta_digito IS NULL OR conta_digito ~ '^[0-9X]{1,2}$'),
    CONSTRAINT chk_conta_bancaria_apelido CHECK (btrim(apelido) <> ''),
    CONSTRAINT chk_conta_bancaria_versao CHECK (versao >= 1),
    CONSTRAINT fk_conta_bancaria_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_conta_bancaria_numero ON conta_bancaria (banco, ltrim(agencia, '0'), ltrim(conta, '0'));
CREATE UNIQUE INDEX uq_conta_bancaria_empresa_apelido ON conta_bancaria (empresa_id, lower(apelido));
CREATE INDEX idx_conta_bancaria_empresa ON conta_bancaria (empresa_id, apelido, id);
