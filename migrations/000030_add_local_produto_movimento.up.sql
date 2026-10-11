ALTER TABLE produto_movimento ALTER COLUMN tipo TYPE VARCHAR(15);

ALTER TABLE produto_movimento DROP CONSTRAINT chk_produto_movimento_tipo;

ALTER TABLE produto_movimento
    ADD CONSTRAINT chk_produto_movimento_tipo CHECK (
        tipo IN ('entrada', 'saida', 'ajuste', 'perda', 'devolucao', 'implantacao', 'transferencia')
    );

ALTER TABLE produto_movimento
    ADD COLUMN local_id UUID,
    ADD COLUMN local_destino_id UUID,
    ADD COLUMN saldo_local_apos INT,
    ADD COLUMN custo_unitario_centavos BIGINT;

ALTER TABLE produto_movimento
    ADD CONSTRAINT fk_produto_movimento_local FOREIGN KEY (local_id) REFERENCES estoque_local (id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_produto_movimento_local_destino FOREIGN KEY (local_destino_id) REFERENCES estoque_local (id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_produto_movimento_destino CHECK (
        (tipo = 'transferencia') = (local_destino_id IS NOT NULL)
        AND (local_destino_id IS NULL OR local_destino_id <> local_id)
    ),
    ADD CONSTRAINT chk_produto_movimento_saldo_local_apos CHECK (saldo_local_apos IS NULL OR saldo_local_apos >= 0),
    ADD CONSTRAINT chk_produto_movimento_custo CHECK (custo_unitario_centavos IS NULL OR custo_unitario_centavos >= 0);

CREATE INDEX idx_produto_movimento_local ON produto_movimento (local_id, criado_em DESC) WHERE local_id IS NOT NULL;
CREATE INDEX idx_produto_movimento_local_destino ON produto_movimento (local_destino_id) WHERE local_destino_id IS NOT NULL;
