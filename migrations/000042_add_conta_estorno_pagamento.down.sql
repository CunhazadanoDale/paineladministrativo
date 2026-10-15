DROP TABLE pagamento_estorno;

DROP INDEX idx_pagamento_conta_bancaria;
DROP INDEX idx_pagamento_solicitacao;
DROP INDEX uq_pagamento_solicitacao_vigente;

ALTER TABLE pagamento
    ADD CONSTRAINT uq_pagamento_solicitacao UNIQUE (solicitacao_id),
    DROP CONSTRAINT fk_pagamento_usuario,
    DROP CONSTRAINT fk_pagamento_conta_bancaria,
    DROP COLUMN estornado,
    DROP COLUMN usuario_id,
    DROP COLUMN conta_bancaria_id;
