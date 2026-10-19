DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM comissao WHERE base_centavos = 0) THEN
        RAISE EXCEPTION 'rollback da 000056 bloqueado: há comissões com base zero de entregas bonificadas';
    END IF;
END $$;

ALTER TABLE comissao
    DROP CONSTRAINT chk_comissao_base,
    ADD CONSTRAINT chk_comissao_base CHECK (base_centavos > 0);
