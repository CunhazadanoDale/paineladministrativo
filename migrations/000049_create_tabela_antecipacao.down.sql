DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM antecipacao) THEN
        RAISE EXCEPTION 'rollback da 000049 bloqueado: há borderôs gravados e os títulos antecipados ficariam sem o vínculo';
    END IF;
END $$;

DROP TABLE repasse_fomento;
DROP TABLE recompra;
DROP TABLE antecipacao_titulo;
DROP TABLE antecipacao;
