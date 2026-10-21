DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM solicitacao WHERE origem IN ('divida_amortizacao', 'divida_juros')) THEN
        RAISE EXCEPTION 'rollback da 000067 bloqueado: há contas a pagar geradas por dívida';
    END IF;
END $$;

DROP TABLE divida_historico;
DROP TABLE divida_parcela;
DROP TABLE divida;

ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN (
        'manual', 'recorrencia', 'recompra', 'repasse_fomento', 'comissao', 'importacao_parcela', 'importacao_custo'
    ));
