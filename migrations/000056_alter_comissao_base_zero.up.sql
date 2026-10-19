ALTER TABLE comissao
    DROP CONSTRAINT chk_comissao_base,
    ADD CONSTRAINT chk_comissao_base CHECK (base_centavos >= 0);
