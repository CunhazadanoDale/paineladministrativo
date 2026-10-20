ALTER TABLE importacao
    ADD COLUMN cambio_micro BIGINT,
    ADD COLUMN recebida_em DATE,
    ADD COLUMN local_recebimento_id UUID,
    ADD CONSTRAINT chk_importacao_cambio CHECK (cambio_micro IS NULL OR cambio_micro > 0),
    ADD CONSTRAINT chk_importacao_recebimento CHECK (
        (status = 'recebida') = (recebida_em IS NOT NULL)
        AND (recebida_em IS NULL) = (local_recebimento_id IS NULL)
        AND (recebida_em IS NULL OR recebida_em >= desembaracada_em)
        AND (cambio_micro IS NULL OR recebida_em IS NOT NULL)
    ),
    ADD CONSTRAINT fk_importacao_local_recebimento FOREIGN KEY (local_recebimento_id) REFERENCES estoque_local (id) ON DELETE RESTRICT;

CREATE INDEX idx_importacao_local_recebimento ON importacao (local_recebimento_id) WHERE local_recebimento_id IS NOT NULL;

ALTER TABLE importacao_item
    ADD COLUMN quantidade_recebida INT,
    ADD COLUMN custo_unitario_centavos BIGINT,
    ADD COLUMN movimento_id UUID,
    ADD CONSTRAINT chk_importacao_item_recebida CHECK (
        (quantidade_recebida IS NULL OR quantidade_recebida BETWEEN 0 AND quantidade)
        AND (custo_unitario_centavos IS NULL OR custo_unitario_centavos >= 0)
        AND ((movimento_id IS NOT NULL) = (quantidade_recebida IS NOT NULL AND quantidade_recebida > 0))
        AND ((custo_unitario_centavos IS NOT NULL) = (movimento_id IS NOT NULL))
    ),
    ADD CONSTRAINT uq_importacao_item_movimento UNIQUE (movimento_id),
    ADD CONSTRAINT fk_importacao_item_movimento FOREIGN KEY (movimento_id) REFERENCES produto_movimento (id) ON DELETE RESTRICT;

ALTER TABLE importacao_historico
    DROP CONSTRAINT chk_importacao_historico_acao,
    ADD CONSTRAINT chk_importacao_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'embarcada', 'desembaracada', 'cancelada',
        'parcela_incluida', 'parcela_alterada', 'parcela_removida', 'parcela_cambio_fechado', 'parcela_reaberta',
        'custo_lancado', 'custo_cancelado', 'alocacao_alterada', 'recebida'
    ));

CREATE TABLE importacao_alocacao (
    importacao_item_id UUID NOT NULL,
    pedido_item_id UUID NOT NULL,
    quantidade INT NOT NULL,
    reservada INT,
    reserva_id UUID,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao_alocacao PRIMARY KEY (importacao_item_id, pedido_item_id),
    CONSTRAINT uq_importacao_alocacao_reserva UNIQUE (reserva_id),
    CONSTRAINT chk_importacao_alocacao_quantidade CHECK (quantidade > 0),
    CONSTRAINT chk_importacao_alocacao_reservada CHECK (
        (reservada IS NULL OR reservada BETWEEN 0 AND quantidade)
        AND ((reserva_id IS NOT NULL) = (reservada IS NOT NULL AND reservada > 0))
    ),
    CONSTRAINT fk_importacao_alocacao_item FOREIGN KEY (importacao_item_id) REFERENCES importacao_item (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_alocacao_pedido_item FOREIGN KEY (pedido_item_id) REFERENCES pedido_item (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_alocacao_reserva FOREIGN KEY (reserva_id) REFERENCES produto_reserva (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_alocacao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_alocacao_pedido_item ON importacao_alocacao (pedido_item_id);
CREATE INDEX idx_importacao_alocacao_usuario ON importacao_alocacao (usuario_id);
