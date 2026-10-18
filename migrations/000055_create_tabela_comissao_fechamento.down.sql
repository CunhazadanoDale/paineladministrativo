DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM solicitacao WHERE origem = 'comissao') THEN
        RAISE EXCEPTION 'rollback da 000055 bloqueado: há contas a pagar geradas por fechamento de comissão';
    END IF;
END $$;

ALTER TABLE comissao_ajuste DROP COLUMN fechamento_id;
ALTER TABLE comissao DROP COLUMN fechamento_id;

DROP TABLE comissao_fechamento;

ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia', 'recompra', 'repasse_fomento'));

ALTER TABLE representante DROP COLUMN fornecedor_id;
