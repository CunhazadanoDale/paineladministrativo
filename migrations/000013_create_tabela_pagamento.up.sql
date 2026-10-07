CREATE TABLE pagamento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    solicitacao_id UUID NOT NULL,
    comprovante_arquivo_id UUID,
    valor BIGINT NOT NULL,
    pago_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pagamento PRIMARY KEY (id),
    CONSTRAINT chk_pagamento_valor CHECK (valor > 0),
    CONSTRAINT uq_pagamento_solicitacao UNIQUE (solicitacao_id),
    CONSTRAINT fk_pagamento_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pagamento_comprovante FOREIGN KEY (comprovante_arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT
);
