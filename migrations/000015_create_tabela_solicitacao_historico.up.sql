CREATE TABLE solicitacao_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    solicitacao_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    de_status VARCHAR(30),
    para_status VARCHAR(30) NOT NULL,
    descricao VARCHAR(500) NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_solicitacao_historico PRIMARY KEY (id),
    CONSTRAINT chk_sh_para_status CHECK (
        para_status IN ('pendente_aprovacao', 'aprovado', 'pago', 'rejeitado', 'cancelado')
    ),
    CONSTRAINT fk_sh_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE CASCADE,
    CONSTRAINT fk_sh_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_sh_solicitacao ON solicitacao_historico (solicitacao_id, criado_em DESC);
