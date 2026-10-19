CREATE TABLE caixa_abertura (
    conta_bancaria_id UUID NOT NULL,
    data_corte DATE NOT NULL,
    saldo_centavos BIGINT NOT NULL,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_caixa_abertura PRIMARY KEY (conta_bancaria_id),
    CONSTRAINT chk_caixa_abertura_versao CHECK (versao >= 1),
    CONSTRAINT fk_caixa_abertura_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_caixa_abertura_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_caixa_abertura_usuario ON caixa_abertura (usuario_id);
