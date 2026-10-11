DROP INDEX uq_produto_movimento_reserva;

ALTER TABLE produto_movimento
    DROP CONSTRAINT chk_produto_movimento_reserva,
    DROP CONSTRAINT fk_produto_movimento_reserva,
    DROP COLUMN reserva_id;

UPDATE produto SET quantidade_reservada = 0;

UPDATE produto_saldo_local SET reservado = 0;

DROP TABLE produto_reserva;
