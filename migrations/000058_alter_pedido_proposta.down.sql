ALTER TABLE pedido_item DROP COLUMN preco_tabela_centavos;

ALTER TABLE pedido
    DROP COLUMN frete_estimado_centavos,
    DROP COLUMN frete_tipo,
    DROP COLUMN revisao,
    DROP COLUMN validade;
