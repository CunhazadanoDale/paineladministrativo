CREATE TABLE entrega (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pedido_id UUID NOT NULL,
    status VARCHAR(10) NOT NULL,
    transportadora VARCHAR(160),
    frete_tipo VARCHAR(3) NOT NULL,
    frete_centavos BIGINT,
    previsao DATE,
    saiu_em DATE,
    entregue_em DATE,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_entrega PRIMARY KEY (id),
    CONSTRAINT chk_entrega_status CHECK (status IN ('pendente', 'saiu', 'entregue', 'cancelada')),
    CONSTRAINT chk_entrega_transportadora CHECK (transportadora IS NULL OR btrim(transportadora) <> ''),
    CONSTRAINT chk_entrega_frete_tipo CHECK (frete_tipo IN ('cif', 'fob')),
    CONSTRAINT chk_entrega_frete_centavos CHECK (frete_centavos IS NULL OR frete_centavos >= 0),
    CONSTRAINT chk_entrega_frete_cif CHECK (frete_tipo <> 'cif' OR frete_centavos IS NOT NULL),
    CONSTRAINT chk_entrega_saiu_em CHECK ((status IN ('saiu', 'entregue')) = (saiu_em IS NOT NULL)),
    CONSTRAINT chk_entrega_entregue_em CHECK ((status = 'entregue') = (entregue_em IS NOT NULL)),
    CONSTRAINT chk_entrega_datas CHECK (entregue_em IS NULL OR entregue_em >= saiu_em),
    CONSTRAINT chk_entrega_versao CHECK (versao >= 1),
    CONSTRAINT fk_entrega_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_entrega_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_entrega_pedido ON entrega (pedido_id, criado_em, id);
CREATE INDEX idx_entrega_status ON entrega (status, previsao, id);
CREATE INDEX idx_entrega_usuario ON entrega (usuario_id);

CREATE TABLE entrega_item (
    entrega_id UUID NOT NULL,
    pedido_item_id UUID NOT NULL,
    local_id UUID NOT NULL,
    quantidade INT NOT NULL,
    movimento_id UUID,
    CONSTRAINT pk_entrega_item PRIMARY KEY (entrega_id, pedido_item_id),
    CONSTRAINT chk_entrega_item_quantidade CHECK (quantidade > 0),
    CONSTRAINT fk_entrega_item_entrega FOREIGN KEY (entrega_id) REFERENCES entrega (id) ON DELETE RESTRICT,
    CONSTRAINT fk_entrega_item_pedido_item FOREIGN KEY (pedido_item_id) REFERENCES pedido_item (id) ON DELETE RESTRICT,
    CONSTRAINT fk_entrega_item_local FOREIGN KEY (local_id) REFERENCES estoque_local (id) ON DELETE RESTRICT,
    CONSTRAINT fk_entrega_item_movimento FOREIGN KEY (movimento_id) REFERENCES produto_movimento (id) ON DELETE RESTRICT
);

CREATE INDEX idx_entrega_item_pedido_item ON entrega_item (pedido_item_id);
CREATE INDEX idx_entrega_item_local ON entrega_item (local_id);
CREATE UNIQUE INDEX uq_entrega_item_movimento ON entrega_item (movimento_id) WHERE movimento_id IS NOT NULL;
