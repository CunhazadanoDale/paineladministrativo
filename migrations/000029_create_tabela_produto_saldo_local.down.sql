DROP TABLE produto_saldo_local;

ALTER TABLE produto DROP CONSTRAINT chk_produto_quantidade_reservada;

ALTER TABLE produto
    DROP COLUMN versao_estoque,
    DROP COLUMN quantidade_reservada;
