CREATE TABLE aprovador (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_aprovador PRIMARY KEY (id),
    CONSTRAINT uq_aprovador_usuario UNIQUE (usuario_id),
    CONSTRAINT fk_aprovador_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE CASCADE
);
