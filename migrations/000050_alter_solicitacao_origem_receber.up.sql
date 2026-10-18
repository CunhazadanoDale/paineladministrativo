ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia', 'recompra', 'repasse_fomento'));

CREATE UNIQUE INDEX uq_solicitacao_origem ON solicitacao (origem, origem_id)
    WHERE origem NOT IN ('manual', 'recorrencia');
