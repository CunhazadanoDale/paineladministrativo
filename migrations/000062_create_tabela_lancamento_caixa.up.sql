CREATE TABLE lancamento_caixa (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    conta_bancaria_id UUID NOT NULL,
    data DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    categoria_financeira_id UUID NOT NULL,
    competencia DATE NOT NULL,
    descricao VARCHAR(255) NOT NULL,
    estornado BOOLEAN NOT NULL DEFAULT false,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_lancamento_caixa PRIMARY KEY (id),
    CONSTRAINT chk_lancamento_caixa_valor CHECK (valor_centavos <> 0),
    CONSTRAINT chk_lancamento_caixa_competencia CHECK (EXTRACT(DAY FROM competencia) = 1),
    CONSTRAINT chk_lancamento_caixa_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT fk_lancamento_caixa_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_lancamento_caixa_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_lancamento_caixa_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_lancamento_caixa_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_lancamento_caixa_conta ON lancamento_caixa (conta_bancaria_id, data, id);
CREATE INDEX idx_lancamento_caixa_competencia ON lancamento_caixa (empresa_id, competencia) WHERE NOT estornado;
CREATE INDEX idx_lancamento_caixa_categoria ON lancamento_caixa (categoria_financeira_id);
CREATE INDEX idx_lancamento_caixa_usuario ON lancamento_caixa (usuario_id);

CREATE TABLE transferencia_conta (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    conta_origem_id UUID NOT NULL,
    conta_destino_id UUID NOT NULL,
    data DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    entre_empresas BOOLEAN NOT NULL,
    categoria_financeira_id UUID NOT NULL,
    descricao VARCHAR(255) NOT NULL,
    estornado BOOLEAN NOT NULL DEFAULT false,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_transferencia_conta PRIMARY KEY (id),
    CONSTRAINT chk_transferencia_conta_contas CHECK (conta_origem_id <> conta_destino_id),
    CONSTRAINT chk_transferencia_conta_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_transferencia_conta_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT fk_transferencia_conta_origem FOREIGN KEY (conta_origem_id) REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_transferencia_conta_destino FOREIGN KEY (conta_destino_id) REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_transferencia_conta_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_transferencia_conta_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_transferencia_conta_origem ON transferencia_conta (conta_origem_id, data, id);
CREATE INDEX idx_transferencia_conta_destino ON transferencia_conta (conta_destino_id, data, id);
CREATE INDEX idx_transferencia_conta_categoria ON transferencia_conta (categoria_financeira_id);
CREATE INDEX idx_transferencia_conta_usuario ON transferencia_conta (usuario_id);

ALTER TABLE conciliacao DROP CONSTRAINT chk_conciliacao_modo;
ALTER TABLE conciliacao ADD CONSTRAINT chk_conciliacao_modo
    CHECK (modo IN ('manual', 'automatica', 'classificacao', 'transferencia'));

ALTER TABLE conciliacao_item ALTER COLUMN alvo_tipo TYPE VARCHAR(30);
ALTER TABLE conciliacao_item DROP CONSTRAINT chk_conciliacao_item_alvo_tipo;
ALTER TABLE conciliacao_item ADD CONSTRAINT chk_conciliacao_item_alvo_tipo
    CHECK (alvo_tipo IN (
        'pagamento', 'recebimento', 'antecipacao', 'lancamento_caixa', 'transferencia_saida', 'transferencia_entrada'
    ));
