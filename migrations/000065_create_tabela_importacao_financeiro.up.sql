ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN (
        'manual', 'recorrencia', 'recompra', 'repasse_fomento', 'comissao', 'importacao_parcela', 'importacao_custo'
    ));

ALTER TABLE importacao_historico
    DROP CONSTRAINT chk_importacao_historico_acao,
    ADD CONSTRAINT chk_importacao_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'embarcada', 'desembaracada', 'cancelada',
        'parcela_incluida', 'parcela_alterada', 'parcela_removida', 'parcela_cambio_fechado', 'parcela_reaberta',
        'custo_lancado', 'custo_cancelado'
    ));

CREATE TABLE importacao_parcela (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    importacao_id UUID NOT NULL,
    descricao VARCHAR(120) NOT NULL,
    valor_moeda_centesimos BIGINT NOT NULL,
    vencimento_previsto DATE NOT NULL,
    status VARCHAR(14) NOT NULL,
    cambio_micro BIGINT,
    valor_centavos BIGINT,
    vencimento DATE,
    solicitacao_id UUID,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao_parcela PRIMARY KEY (id),
    CONSTRAINT uq_importacao_parcela_solicitacao UNIQUE (solicitacao_id),
    CONSTRAINT chk_importacao_parcela_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_importacao_parcela_valor CHECK (valor_moeda_centesimos > 0),
    CONSTRAINT chk_importacao_parcela_status CHECK (status IN ('aberta', 'cambio_fechado', 'cancelada')),
    CONSTRAINT chk_importacao_parcela_cambio CHECK (
        (status = 'aberta') = (cambio_micro IS NULL)
        AND (cambio_micro IS NULL) = (valor_centavos IS NULL)
        AND (cambio_micro IS NULL) = (vencimento IS NULL)
        AND (cambio_micro IS NULL) = (solicitacao_id IS NULL)
        AND (cambio_micro IS NULL OR cambio_micro > 0)
        AND (valor_centavos IS NULL OR valor_centavos > 0)
    ),
    CONSTRAINT fk_importacao_parcela_importacao FOREIGN KEY (importacao_id) REFERENCES importacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_parcela_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_parcela_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_parcela_importacao ON importacao_parcela (importacao_id, criado_em, id);
CREATE INDEX idx_importacao_parcela_usuario ON importacao_parcela (usuario_id);

CREATE TABLE importacao_custo (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    importacao_id UUID NOT NULL,
    tipo VARCHAR(30) NOT NULL,
    descricao VARCHAR(255) NOT NULL,
    fornecedor_id UUID NOT NULL,
    valor_centavos BIGINT NOT NULL,
    vencimento DATE NOT NULL,
    compoe_custo BOOLEAN NOT NULL,
    posterior BOOLEAN NOT NULL DEFAULT false,
    categoria_financeira_id UUID NOT NULL,
    solicitacao_id UUID NOT NULL,
    cancelado_em TIMESTAMPTZ,
    cancelado_por UUID,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao_custo PRIMARY KEY (id),
    CONSTRAINT uq_importacao_custo_solicitacao UNIQUE (solicitacao_id),
    CONSTRAINT chk_importacao_custo_tipo CHECK (tipo IN (
        'frete_internacional', 'seguro', 'imposto_importacao', 'ipi', 'pis', 'cofins', 'icms', 'taxa_siscomex',
        'despachante', 'armazenagem', 'frete_nacional', 'outros'
    )),
    CONSTRAINT chk_importacao_custo_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_importacao_custo_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_importacao_custo_posterior CHECK (NOT (posterior AND compoe_custo)),
    CONSTRAINT chk_importacao_custo_cancelado CHECK ((cancelado_em IS NULL) = (cancelado_por IS NULL)),
    CONSTRAINT fk_importacao_custo_importacao FOREIGN KEY (importacao_id) REFERENCES importacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_custo_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_custo_categoria FOREIGN KEY (categoria_financeira_id) REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_custo_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_custo_cancelado_por FOREIGN KEY (cancelado_por) REFERENCES usuario (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_custo_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_custo_importacao ON importacao_custo (importacao_id, criado_em, id);
CREATE INDEX idx_importacao_custo_fornecedor ON importacao_custo (fornecedor_id);
CREATE INDEX idx_importacao_custo_categoria ON importacao_custo (categoria_financeira_id);
CREATE INDEX idx_importacao_custo_usuario ON importacao_custo (usuario_id);
CREATE INDEX idx_importacao_custo_cancelado_por ON importacao_custo (cancelado_por) WHERE cancelado_por IS NOT NULL;
