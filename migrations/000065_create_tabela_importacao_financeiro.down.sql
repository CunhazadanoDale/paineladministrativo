DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM solicitacao WHERE origem IN ('importacao_parcela', 'importacao_custo')) THEN
        RAISE EXCEPTION 'rollback da 000065 bloqueado: há contas a pagar geradas por importação';
    END IF;
    IF EXISTS (SELECT 1 FROM importacao_historico WHERE acao LIKE 'parcela_%' OR acao LIKE 'custo_%') THEN
        RAISE EXCEPTION 'rollback da 000065 bloqueado: há histórico de parcelas ou custos de importação';
    END IF;
END $$;

DROP TABLE importacao_custo;
DROP TABLE importacao_parcela;

ALTER TABLE importacao_historico
    DROP CONSTRAINT chk_importacao_historico_acao,
    ADD CONSTRAINT chk_importacao_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'embarcada', 'desembaracada', 'cancelada'
    ));

ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia', 'recompra', 'repasse_fomento', 'comissao'));
