DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM importacao WHERE status = 'recebida') THEN
        RAISE EXCEPTION 'rollback da 000066 bloqueado: há importações recebidas';
    END IF;
    IF EXISTS (SELECT 1 FROM importacao_historico WHERE acao IN ('alocacao_alterada', 'recebida')) THEN
        RAISE EXCEPTION 'rollback da 000066 bloqueado: há histórico de alocação ou recebimento';
    END IF;
END $$;

DROP TABLE importacao_alocacao;

ALTER TABLE importacao_historico
    DROP CONSTRAINT chk_importacao_historico_acao,
    ADD CONSTRAINT chk_importacao_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'embarcada', 'desembaracada', 'cancelada',
        'parcela_incluida', 'parcela_alterada', 'parcela_removida', 'parcela_cambio_fechado', 'parcela_reaberta',
        'custo_lancado', 'custo_cancelado'
    ));

ALTER TABLE importacao_item
    DROP CONSTRAINT fk_importacao_item_movimento,
    DROP CONSTRAINT uq_importacao_item_movimento,
    DROP CONSTRAINT chk_importacao_item_recebida,
    DROP COLUMN movimento_id,
    DROP COLUMN custo_unitario_centavos,
    DROP COLUMN quantidade_recebida;

DROP INDEX idx_importacao_local_recebimento;

ALTER TABLE importacao
    DROP CONSTRAINT fk_importacao_local_recebimento,
    DROP CONSTRAINT chk_importacao_recebimento,
    DROP CONSTRAINT chk_importacao_cambio,
    DROP COLUMN local_recebimento_id,
    DROP COLUMN recebida_em,
    DROP COLUMN cambio_micro;
