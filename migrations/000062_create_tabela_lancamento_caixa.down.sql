DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM conciliacao WHERE modo IN ('classificacao', 'transferencia'))
        OR EXISTS (
            SELECT 1 FROM conciliacao_item
            WHERE alvo_tipo IN ('lancamento_caixa', 'transferencia_saida', 'transferencia_entrada')
        ) THEN
        RAISE EXCEPTION 'rollback da 000062 bloqueado: há conciliações de lançamento de caixa ou de transferência gravadas';
    END IF;
END $$;

ALTER TABLE conciliacao_item DROP CONSTRAINT chk_conciliacao_item_alvo_tipo;
ALTER TABLE conciliacao_item ADD CONSTRAINT chk_conciliacao_item_alvo_tipo
    CHECK (alvo_tipo IN ('pagamento', 'recebimento', 'antecipacao'));
ALTER TABLE conciliacao_item ALTER COLUMN alvo_tipo TYPE VARCHAR(20);

ALTER TABLE conciliacao DROP CONSTRAINT chk_conciliacao_modo;
ALTER TABLE conciliacao ADD CONSTRAINT chk_conciliacao_modo CHECK (modo IN ('manual', 'automatica'));

DROP TABLE transferencia_conta;
DROP TABLE lancamento_caixa;
