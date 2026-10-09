CREATE TABLE produto_movimento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL,
    tipo VARCHAR(12) NOT NULL,
    quantidade INT NOT NULL,
    saldo_apos INT NOT NULL,
    usuario_id UUID NOT NULL,
    documento_ref VARCHAR(80),
    observacao VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_produto_movimento PRIMARY KEY (id),
    CONSTRAINT chk_produto_movimento_tipo CHECK (
        tipo IN ('entrada', 'saida', 'ajuste', 'perda', 'devolucao')
    ),
    CONSTRAINT chk_produto_movimento_quantidade CHECK (quantidade <> 0),
    CONSTRAINT chk_produto_movimento_saldo_apos CHECK (saldo_apos >= 0),
    CONSTRAINT fk_produto_movimento_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_movimento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_movimento_produto ON produto_movimento (produto_id, criado_em DESC);
CREATE INDEX idx_produto_movimento_usuario ON produto_movimento (usuario_id, criado_em DESC);
