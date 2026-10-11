CREATE TABLE produto_lote (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL,
    local_id UUID NOT NULL,
    movimento_id UUID NOT NULL,
    lote_origem_id UUID,
    custo_unitario_centavos BIGINT,
    quantidade_inicial INT NOT NULL,
    quantidade_atual INT NOT NULL,
    entrada_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_produto_lote PRIMARY KEY (id),
    CONSTRAINT chk_produto_lote_custo CHECK (custo_unitario_centavos IS NULL OR custo_unitario_centavos >= 0),
    CONSTRAINT chk_produto_lote_quantidade_inicial CHECK (quantidade_inicial > 0),
    CONSTRAINT chk_produto_lote_quantidade_atual CHECK (quantidade_atual >= 0 AND quantidade_atual <= quantidade_inicial),
    CONSTRAINT chk_produto_lote_origem CHECK (lote_origem_id IS NULL OR lote_origem_id <> id),
    CONSTRAINT fk_produto_lote_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_lote_local FOREIGN KEY (local_id) REFERENCES estoque_local (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_lote_movimento FOREIGN KEY (movimento_id) REFERENCES produto_movimento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_lote_origem FOREIGN KEY (lote_origem_id) REFERENCES produto_lote (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_lote_fifo ON produto_lote (produto_id, local_id, entrada_em, id) WHERE quantidade_atual > 0;
CREATE INDEX idx_produto_lote_local ON produto_lote (local_id);
CREATE INDEX idx_produto_lote_movimento ON produto_lote (movimento_id);
CREATE INDEX idx_produto_lote_origem ON produto_lote (lote_origem_id) WHERE lote_origem_id IS NOT NULL;

CREATE TABLE produto_movimento_lote (
    movimento_id UUID NOT NULL,
    lote_id UUID NOT NULL,
    quantidade INT NOT NULL,
    custo_unitario_centavos BIGINT,
    CONSTRAINT pk_produto_movimento_lote PRIMARY KEY (movimento_id, lote_id),
    CONSTRAINT chk_produto_movimento_lote_quantidade CHECK (quantidade > 0),
    CONSTRAINT chk_produto_movimento_lote_custo CHECK (custo_unitario_centavos IS NULL OR custo_unitario_centavos >= 0),
    CONSTRAINT fk_produto_movimento_lote_movimento FOREIGN KEY (movimento_id) REFERENCES produto_movimento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_movimento_lote_lote FOREIGN KEY (lote_id) REFERENCES produto_lote (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_movimento_lote_lote ON produto_movimento_lote (lote_id);
