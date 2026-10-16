DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM solicitacao WHERE origem IN ('recompra', 'repasse_fomento')) THEN
        RAISE EXCEPTION 'rollback da 000050 bloqueado: há contas a pagar geradas por recompra ou repasse ao fomento';
    END IF;
END $$;

DROP INDEX uq_solicitacao_origem;

ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia'));
