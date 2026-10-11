CREATE TABLE produto_reserva (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL,
    local_id UUID NOT NULL,
    quantidade INT NOT NULL,
    origem VARCHAR(20) NOT NULL,
    origem_id UUID,
    status VARCHAR(10) NOT NULL,
    usuario_id UUID NOT NULL,
    observacao VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    encerrada_em TIMESTAMPTZ,
    CONSTRAINT pk_produto_reserva PRIMARY KEY (id),
    CONSTRAINT chk_produto_reserva_quantidade CHECK (quantidade > 0),
    CONSTRAINT chk_produto_reserva_origem CHECK (origem IN ('manual')),
    CONSTRAINT chk_produto_reserva_status CHECK (status IN ('ativa', 'baixada', 'liberada')),
    CONSTRAINT chk_produto_reserva_encerrada CHECK ((status = 'ativa') = (encerrada_em IS NULL)),
    CONSTRAINT fk_produto_reserva_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_reserva_local FOREIGN KEY (local_id) REFERENCES estoque_local (id) ON DELETE RESTRICT,
    CONSTRAINT fk_produto_reserva_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_reserva_produto ON produto_reserva (produto_id, criado_em DESC, id DESC);
CREATE INDEX idx_produto_reserva_local ON produto_reserva (local_id, status);
CREATE INDEX idx_produto_reserva_usuario ON produto_reserva (usuario_id);

ALTER TABLE produto_movimento
    ADD COLUMN reserva_id UUID,
    ADD CONSTRAINT fk_produto_movimento_reserva FOREIGN KEY (reserva_id) REFERENCES produto_reserva (id) ON DELETE RESTRICT,
    ADD CONSTRAINT chk_produto_movimento_reserva CHECK (reserva_id IS NULL OR tipo = 'saida');

CREATE UNIQUE INDEX uq_produto_movimento_reserva ON produto_movimento (reserva_id) WHERE reserva_id IS NOT NULL;
