DROP INDEX idx_produto_reserva_origem;

ALTER TABLE produto_reserva
    DROP CONSTRAINT chk_produto_reserva_origem_pedido_item,
    DROP CONSTRAINT chk_produto_reserva_origem,
    ADD CONSTRAINT chk_produto_reserva_origem CHECK (origem IN ('manual'));
