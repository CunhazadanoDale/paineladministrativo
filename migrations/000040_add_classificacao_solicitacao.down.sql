DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM solicitacao WHERE forma_pagamento IN ('transferencia', 'debito_automatico')) THEN
        RAISE EXCEPTION 'rollback da 000040 bloqueado: há solicitações com forma de pagamento transferencia ou debito_automatico';
    END IF;
END $$;

DROP INDEX idx_solicitacao_em_aberto;
DROP INDEX idx_solicitacao_categoria_financeira;
DROP INDEX idx_solicitacao_fornecedor;
DROP INDEX idx_solicitacao_vencimento;

ALTER TABLE solicitacao
    DROP CONSTRAINT fk_solicitacao_categoria_financeira,
    DROP CONSTRAINT fk_solicitacao_fornecedor,
    DROP CONSTRAINT chk_solicitacao_versao,
    DROP CONSTRAINT chk_solicitacao_origem_id,
    DROP CONSTRAINT chk_solicitacao_origem,
    DROP CONSTRAINT chk_solicitacao_competencia,
    DROP CONSTRAINT chk_solicitacao_forma_pagamento,
    ADD CONSTRAINT chk_solicitacao_forma_pagamento CHECK (forma_pagamento IN ('pix', 'cartao', 'boleto')),
    ALTER COLUMN forma_pagamento TYPE VARCHAR(10),
    DROP COLUMN versao,
    DROP COLUMN origem_id,
    DROP COLUMN origem,
    DROP COLUMN estimado,
    DROP COLUMN competencia,
    DROP COLUMN categoria_financeira_id,
    DROP COLUMN fornecedor_id;
