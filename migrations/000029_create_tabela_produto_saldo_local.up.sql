ALTER TABLE produto
    ADD COLUMN quantidade_reservada INT NOT NULL DEFAULT 0,
    ADD COLUMN versao_estoque BIGINT NOT NULL DEFAULT 0;

ALTER TABLE produto
    ADD CONSTRAINT chk_produto_quantidade_reservada CHECK (
        quantidade_reservada >= 0 AND quantidade_reservada <= quantidade_atual
    );

CREATE TABLE produto_saldo_local (
    produto_id UUID NOT NULL,
    local_id UUID NOT NULL,
    quantidade INT NOT NULL DEFAULT 0,
    reservado INT NOT NULL DEFAULT 0,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_produto_saldo_local PRIMARY KEY (produto_id, local_id),
    CONSTRAINT chk_produto_saldo_local_quantidade CHECK (quantidade >= 0),
    CONSTRAINT chk_produto_saldo_local_reservado CHECK (reservado >= 0 AND reservado <= quantidade),
    CONSTRAINT fk_produto_saldo_local_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_saldo_local_local FOREIGN KEY (local_id) REFERENCES estoque_local (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_saldo_local_local ON produto_saldo_local (local_id, produto_id);
