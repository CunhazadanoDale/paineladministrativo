CREATE TABLE extrato_importacao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    conta_bancaria_id UUID NOT NULL,
    arquivo_id UUID NOT NULL,
    periodo_inicio DATE NOT NULL,
    periodo_fim DATE NOT NULL,
    saldo_banco_centavos BIGINT,
    saldo_banco_em DATE,
    linhas_novas INT NOT NULL DEFAULT 0,
    linhas_repetidas INT NOT NULL DEFAULT 0,
    linhas_ignoradas INT NOT NULL DEFAULT 0,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_extrato_importacao PRIMARY KEY (id),
    CONSTRAINT uq_extrato_importacao_arquivo UNIQUE (arquivo_id),
    CONSTRAINT chk_extrato_importacao_periodo CHECK (periodo_inicio <= periodo_fim),
    CONSTRAINT chk_extrato_importacao_saldo_banco CHECK ((saldo_banco_centavos IS NULL) = (saldo_banco_em IS NULL)),
    CONSTRAINT chk_extrato_importacao_linhas CHECK (
        linhas_novas >= 0 AND linhas_repetidas >= 0 AND linhas_ignoradas >= 0
    ),
    CONSTRAINT fk_extrato_importacao_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_extrato_importacao_arquivo FOREIGN KEY (arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT,
    CONSTRAINT fk_extrato_importacao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_extrato_importacao_conta ON extrato_importacao (conta_bancaria_id, criado_em DESC, id);
CREATE INDEX idx_extrato_importacao_usuario ON extrato_importacao (usuario_id);

CREATE TABLE extrato_lancamento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    conta_bancaria_id UUID NOT NULL,
    importacao_id UUID NOT NULL,
    sequencia INT NOT NULL,
    data DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    descricao VARCHAR(255) NOT NULL,
    documento VARCHAR(60),
    identificador VARCHAR(128) NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_extrato_lancamento PRIMARY KEY (id),
    CONSTRAINT uq_extrato_lancamento_identificador UNIQUE (conta_bancaria_id, identificador),
    CONSTRAINT chk_extrato_lancamento_sequencia CHECK (sequencia >= 1),
    CONSTRAINT chk_extrato_lancamento_valor CHECK (valor_centavos <> 0),
    CONSTRAINT chk_extrato_lancamento_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_extrato_lancamento_documento CHECK (documento IS NULL OR btrim(documento) <> ''),
    CONSTRAINT chk_extrato_lancamento_identificador CHECK (btrim(identificador) <> ''),
    CONSTRAINT fk_extrato_lancamento_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_extrato_lancamento_importacao FOREIGN KEY (importacao_id)
        REFERENCES extrato_importacao (id) ON DELETE RESTRICT
);

CREATE INDEX idx_extrato_lancamento_conta ON extrato_lancamento (conta_bancaria_id, data, id);
CREATE INDEX idx_extrato_lancamento_importacao ON extrato_lancamento (importacao_id);
