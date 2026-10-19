CREATE TABLE conciliacao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    extrato_lancamento_id UUID NOT NULL,
    conta_bancaria_id UUID NOT NULL,
    modo VARCHAR(14) NOT NULL,
    vigente BOOLEAN NOT NULL DEFAULT true,
    desfeita_em TIMESTAMPTZ,
    desfeita_por UUID,
    motivo_desfazer VARCHAR(500),
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_conciliacao PRIMARY KEY (id),
    CONSTRAINT chk_conciliacao_modo CHECK (modo IN ('manual', 'automatica')),
    CONSTRAINT chk_conciliacao_desfeita CHECK (
        vigente = (desfeita_em IS NULL)
        AND (desfeita_em IS NULL) = (desfeita_por IS NULL)
        AND (desfeita_por IS NULL) = (motivo_desfazer IS NULL)
        AND (motivo_desfazer IS NULL OR btrim(motivo_desfazer) <> '')
    ),
    CONSTRAINT fk_conciliacao_extrato_lancamento FOREIGN KEY (extrato_lancamento_id)
        REFERENCES extrato_lancamento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_conciliacao_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_conciliacao_desfeita_por FOREIGN KEY (desfeita_por) REFERENCES usuario (id) ON DELETE RESTRICT,
    CONSTRAINT fk_conciliacao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_conciliacao_lancamento_vigente ON conciliacao (extrato_lancamento_id) WHERE vigente;
CREATE INDEX idx_conciliacao_lancamento ON conciliacao (extrato_lancamento_id);
CREATE INDEX idx_conciliacao_conta ON conciliacao (conta_bancaria_id, criado_em DESC, id);
CREATE INDEX idx_conciliacao_desfeita_por ON conciliacao (desfeita_por) WHERE desfeita_por IS NOT NULL;
CREATE INDEX idx_conciliacao_usuario ON conciliacao (usuario_id);

CREATE TABLE conciliacao_item (
    conciliacao_id UUID NOT NULL,
    alvo_tipo VARCHAR(20) NOT NULL,
    alvo_id UUID NOT NULL,
    valor_centavos BIGINT NOT NULL,
    vigente BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT pk_conciliacao_item PRIMARY KEY (conciliacao_id, alvo_tipo, alvo_id),
    CONSTRAINT chk_conciliacao_item_alvo_tipo CHECK (alvo_tipo IN ('pagamento', 'recebimento', 'antecipacao')),
    CONSTRAINT chk_conciliacao_item_valor CHECK (valor_centavos <> 0),
    CONSTRAINT fk_conciliacao_item_conciliacao FOREIGN KEY (conciliacao_id)
        REFERENCES conciliacao (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_conciliacao_item_alvo_vigente ON conciliacao_item (alvo_tipo, alvo_id) WHERE vigente;
CREATE INDEX idx_conciliacao_item_alvo ON conciliacao_item (alvo_tipo, alvo_id);
