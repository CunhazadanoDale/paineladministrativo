ALTER TABLE pagamento
    ADD COLUMN conta_bancaria_id UUID,
    ADD COLUMN usuario_id UUID,
    ADD COLUMN estornado BOOLEAN NOT NULL DEFAULT false,
    ADD CONSTRAINT fk_pagamento_conta_bancaria FOREIGN KEY (conta_bancaria_id) REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_pagamento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT,
    DROP CONSTRAINT uq_pagamento_solicitacao;

CREATE UNIQUE INDEX uq_pagamento_solicitacao_vigente ON pagamento (solicitacao_id) WHERE NOT estornado;
CREATE INDEX idx_pagamento_solicitacao ON pagamento (solicitacao_id, criado_em, id);
CREATE INDEX idx_pagamento_conta_bancaria ON pagamento (conta_bancaria_id, pago_em) WHERE conta_bancaria_id IS NOT NULL;

CREATE TABLE pagamento_estorno (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pagamento_id UUID NOT NULL,
    motivo VARCHAR(500) NOT NULL,
    estornado_em TIMESTAMPTZ NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pagamento_estorno PRIMARY KEY (id),
    CONSTRAINT uq_pagamento_estorno_pagamento UNIQUE (pagamento_id),
    CONSTRAINT chk_pagamento_estorno_motivo CHECK (btrim(motivo) <> ''),
    CONSTRAINT fk_pagamento_estorno_pagamento FOREIGN KEY (pagamento_id) REFERENCES pagamento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pagamento_estorno_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_pagamento_estorno_usuario ON pagamento_estorno (usuario_id);
