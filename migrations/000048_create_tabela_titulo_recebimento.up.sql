CREATE TABLE titulo_recebimento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    titulo_id UUID NOT NULL,
    tipo VARCHAR(10) NOT NULL,
    conta_bancaria_id UUID,
    recebido_em DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    juros_centavos BIGINT NOT NULL DEFAULT 0,
    desconto_centavos BIGINT NOT NULL DEFAULT 0,
    status_titulo_anterior VARCHAR(12) NOT NULL,
    registrou_entrada_pedido BOOLEAN NOT NULL DEFAULT false,
    estornado BOOLEAN NOT NULL DEFAULT false,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_titulo_recebimento PRIMARY KEY (id),
    CONSTRAINT chk_titulo_recebimento_tipo CHECK (tipo IN ('conta', 'fomento')),
    CONSTRAINT chk_titulo_recebimento_conta CHECK ((tipo = 'conta') = (conta_bancaria_id IS NOT NULL)),
    CONSTRAINT chk_titulo_recebimento_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_titulo_recebimento_juros CHECK (juros_centavos >= 0),
    CONSTRAINT chk_titulo_recebimento_desconto CHECK (desconto_centavos >= 0),
    CONSTRAINT chk_titulo_recebimento_status_anterior CHECK (
        status_titulo_anterior IN ('carteira', 'em_banco', 'antecipado')
    ),
    CONSTRAINT fk_titulo_recebimento_titulo FOREIGN KEY (titulo_id) REFERENCES titulo_receber (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_recebimento_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_recebimento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_titulo_recebimento_titulo ON titulo_recebimento (titulo_id, criado_em, id);
CREATE INDEX idx_titulo_recebimento_conta ON titulo_recebimento (conta_bancaria_id, recebido_em)
    WHERE conta_bancaria_id IS NOT NULL;
CREATE INDEX idx_titulo_recebimento_usuario ON titulo_recebimento (usuario_id);

CREATE TABLE titulo_recebimento_estorno (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    recebimento_id UUID NOT NULL,
    motivo VARCHAR(500) NOT NULL,
    estornado_em DATE NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_titulo_recebimento_estorno PRIMARY KEY (id),
    CONSTRAINT uq_titulo_recebimento_estorno_recebimento UNIQUE (recebimento_id),
    CONSTRAINT chk_titulo_recebimento_estorno_motivo CHECK (btrim(motivo) <> ''),
    CONSTRAINT fk_titulo_recebimento_estorno_recebimento FOREIGN KEY (recebimento_id)
        REFERENCES titulo_recebimento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_recebimento_estorno_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_titulo_recebimento_estorno_usuario ON titulo_recebimento_estorno (usuario_id);
