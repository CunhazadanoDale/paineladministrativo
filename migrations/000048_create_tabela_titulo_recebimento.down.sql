DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM titulo_recebimento) THEN
        RAISE EXCEPTION 'rollback da 000048 bloqueado: há recebimentos de títulos e os saldos ficariam sem as baixas';
    END IF;
END $$;

DROP TABLE titulo_recebimento_estorno;
DROP TABLE titulo_recebimento;
