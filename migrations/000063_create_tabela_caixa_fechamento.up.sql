CREATE TABLE caixa_fechamento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    conta_bancaria_id UUID NOT NULL,
    competencia DATE NOT NULL,
    saldo_inicial_centavos BIGINT NOT NULL,
    entradas_centavos BIGINT NOT NULL,
    saidas_centavos BIGINT NOT NULL,
    saldo_final_centavos BIGINT NOT NULL,
    saldo_banco_centavos BIGINT NOT NULL,
    vigente BOOLEAN NOT NULL DEFAULT true,
    reaberto_em TIMESTAMPTZ,
    reaberto_por UUID,
    motivo_reabertura VARCHAR(500),
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_caixa_fechamento PRIMARY KEY (id),
    CONSTRAINT chk_caixa_fechamento_competencia CHECK (EXTRACT(DAY FROM competencia) = 1),
    CONSTRAINT chk_caixa_fechamento_movimento CHECK (entradas_centavos >= 0 AND saidas_centavos >= 0),
    CONSTRAINT chk_caixa_fechamento_saldo_final CHECK (
        saldo_final_centavos = saldo_inicial_centavos + entradas_centavos - saidas_centavos
    ),
    CONSTRAINT chk_caixa_fechamento_saldo_banco CHECK (saldo_banco_centavos = saldo_final_centavos),
    CONSTRAINT chk_caixa_fechamento_reabertura CHECK (
        vigente = (reaberto_em IS NULL)
        AND (reaberto_em IS NULL) = (reaberto_por IS NULL)
        AND (reaberto_por IS NULL) = (motivo_reabertura IS NULL)
        AND (motivo_reabertura IS NULL OR btrim(motivo_reabertura) <> '')
    ),
    CONSTRAINT fk_caixa_fechamento_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_caixa_fechamento_reaberto_por FOREIGN KEY (reaberto_por) REFERENCES usuario (id) ON DELETE RESTRICT,
    CONSTRAINT fk_caixa_fechamento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_caixa_fechamento_vigente ON caixa_fechamento (conta_bancaria_id, competencia) WHERE vigente;
CREATE INDEX idx_caixa_fechamento_conta ON caixa_fechamento (conta_bancaria_id, competencia DESC, id);
CREATE INDEX idx_caixa_fechamento_reaberto_por ON caixa_fechamento (reaberto_por) WHERE reaberto_por IS NOT NULL;
CREATE INDEX idx_caixa_fechamento_usuario ON caixa_fechamento (usuario_id);
