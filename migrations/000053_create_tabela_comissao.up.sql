CREATE TABLE pedido_comissao (
    pedido_id UUID NOT NULL,
    representante_id UUID NOT NULL,
    percentual_bp INT NOT NULL,
    origem VARCHAR(10) NOT NULL,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pedido_comissao PRIMARY KEY (pedido_id),
    CONSTRAINT chk_pedido_comissao_percentual CHECK (percentual_bp BETWEEN 0 AND 10000),
    CONSTRAINT chk_pedido_comissao_origem CHECK (origem IN ('padrao', 'manual')),
    CONSTRAINT chk_pedido_comissao_versao CHECK (versao >= 1),
    CONSTRAINT fk_pedido_comissao_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_comissao_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_comissao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_pedido_comissao_representante ON pedido_comissao (representante_id);
CREATE INDEX idx_pedido_comissao_usuario ON pedido_comissao (usuario_id);

CREATE TABLE comissao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    representante_id UUID NOT NULL,
    pedido_id UUID NOT NULL,
    entrega_id UUID NOT NULL,
    competencia DATE NOT NULL,
    base_centavos BIGINT NOT NULL,
    percentual_bp INT NOT NULL,
    valor_centavos BIGINT NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_comissao PRIMARY KEY (id),
    CONSTRAINT chk_comissao_competencia CHECK (competencia = date_trunc('month', competencia)::date),
    CONSTRAINT chk_comissao_base CHECK (base_centavos > 0),
    CONSTRAINT chk_comissao_percentual CHECK (percentual_bp BETWEEN 0 AND 10000),
    CONSTRAINT chk_comissao_valor CHECK (valor_centavos BETWEEN 0 AND base_centavos),
    CONSTRAINT fk_comissao_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_entrega FOREIGN KEY (entrega_id) REFERENCES entrega (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_comissao_entrega ON comissao (entrega_id);
CREATE INDEX idx_comissao_competencia ON comissao (empresa_id, competencia, representante_id, id);
CREATE INDEX idx_comissao_representante ON comissao (representante_id, competencia, id);
CREATE INDEX idx_comissao_pedido ON comissao (pedido_id);
CREATE INDEX idx_comissao_usuario ON comissao (usuario_id);
