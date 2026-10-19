ALTER TABLE pedido
    ADD COLUMN validade DATE,
    ADD COLUMN revisao INT NOT NULL DEFAULT 1,
    ADD COLUMN frete_tipo VARCHAR(3),
    ADD COLUMN frete_estimado_centavos BIGINT,
    ADD CONSTRAINT chk_pedido_revisao CHECK (revisao >= 1),
    ADD CONSTRAINT chk_pedido_frete_tipo CHECK (frete_tipo IS NULL OR frete_tipo IN ('cif', 'fob')),
    ADD CONSTRAINT chk_pedido_frete_estimado CHECK (
        frete_estimado_centavos IS NULL OR (frete_estimado_centavos >= 0 AND frete_tipo IS NOT NULL)
    ),
    ADD CONSTRAINT chk_pedido_frete_cif CHECK (frete_tipo IS DISTINCT FROM 'cif' OR frete_estimado_centavos IS NOT NULL);

UPDATE pedido
SET validade = (criado_em AT TIME ZONE 'America/Sao_Paulo')::date + 15
WHERE status = 'proposta';

ALTER TABLE pedido
    ADD CONSTRAINT chk_pedido_validade CHECK (status <> 'proposta' OR validade IS NOT NULL);

ALTER TABLE pedido_item
    ADD COLUMN preco_tabela_centavos BIGINT,
    ADD CONSTRAINT chk_pedido_item_preco_tabela CHECK (preco_tabela_centavos IS NULL OR preco_tabela_centavos > 0);
