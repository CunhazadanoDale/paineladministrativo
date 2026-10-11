DELETE FROM produto_movimento WHERE tipo IN ('implantacao', 'transferencia');

DROP INDEX idx_produto_movimento_local_destino;

DROP INDEX idx_produto_movimento_local;

ALTER TABLE produto_movimento
    DROP CONSTRAINT chk_produto_movimento_custo,
    DROP CONSTRAINT chk_produto_movimento_saldo_local_apos,
    DROP CONSTRAINT chk_produto_movimento_destino,
    DROP CONSTRAINT fk_produto_movimento_local_destino,
    DROP CONSTRAINT fk_produto_movimento_local;

ALTER TABLE produto_movimento
    DROP COLUMN custo_unitario_centavos,
    DROP COLUMN saldo_local_apos,
    DROP COLUMN local_destino_id,
    DROP COLUMN local_id;

ALTER TABLE produto_movimento DROP CONSTRAINT chk_produto_movimento_tipo;

ALTER TABLE produto_movimento
    ADD CONSTRAINT chk_produto_movimento_tipo CHECK (
        tipo IN ('entrada', 'saida', 'ajuste', 'perda', 'devolucao')
    );

ALTER TABLE produto_movimento ALTER COLUMN tipo TYPE VARCHAR(12);
