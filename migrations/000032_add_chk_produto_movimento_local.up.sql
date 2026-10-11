ALTER TABLE produto_movimento
    ADD CONSTRAINT chk_produto_movimento_local CHECK (local_id IS NOT NULL AND saldo_local_apos IS NOT NULL) NOT VALID;
