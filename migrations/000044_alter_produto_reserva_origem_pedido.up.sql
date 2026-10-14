ALTER TABLE produto_reserva
    DROP CONSTRAINT chk_produto_reserva_origem,
    ADD CONSTRAINT chk_produto_reserva_origem CHECK (origem IN ('manual', 'pedido_item')),
    ADD CONSTRAINT chk_produto_reserva_origem_pedido_item CHECK (origem <> 'pedido_item' OR origem_id IS NOT NULL);

CREATE INDEX idx_produto_reserva_origem ON produto_reserva (origem_id, status) WHERE origem_id IS NOT NULL;
