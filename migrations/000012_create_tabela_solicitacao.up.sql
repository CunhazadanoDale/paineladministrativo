CREATE TABLE solicitacao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    solicitante_id UUID NOT NULL,
    aprovador_id UUID,
    valor_estimado BIGINT NOT NULL,
    prazo_pagamento DATE NOT NULL,
    observacao VARCHAR(1000) NOT NULL,
    forma_pagamento VARCHAR(10) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pendente_aprovacao',
    rejeitacao_motivo VARCHAR(500),
    aprovado_em TIMESTAMPTZ,
    rejeitado_em TIMESTAMPTZ,
    cancelado_em TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_solicitacao PRIMARY KEY (id),
    CONSTRAINT chk_solicitacao_valor CHECK (valor_estimado > 0),
    CONSTRAINT chk_solicitacao_observacao CHECK (btrim(observacao) <> ''),
    CONSTRAINT chk_solicitacao_forma_pagamento CHECK (forma_pagamento IN ('pix', 'cartao', 'boleto')),
    CONSTRAINT chk_solicitacao_status CHECK (
        status IN ('pendente_aprovacao', 'aprovado', 'pago', 'rejeitado', 'cancelado')
    ),
    CONSTRAINT chk_solicitacao_rejeitado CHECK (status <> 'rejeitado' OR btrim(rejeitacao_motivo) <> ''),
    CONSTRAINT fk_solicitacao_solicitante FOREIGN KEY (solicitante_id) REFERENCES usuario (id) ON DELETE RESTRICT,
    CONSTRAINT fk_solicitacao_aprovador FOREIGN KEY (aprovador_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_solicitacao_status_criado ON solicitacao (status, criado_em DESC);
CREATE INDEX idx_solicitacao_solicitante ON solicitacao (solicitante_id, criado_em DESC);
CREATE INDEX idx_solicitacao_aprovador ON solicitacao (aprovador_id, status) WHERE aprovador_id IS NOT NULL;
