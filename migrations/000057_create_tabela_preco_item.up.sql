CREATE TABLE tabela_preco_item (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL,
    modelo VARCHAR(16) NOT NULL,
    preco_centavos BIGINT NOT NULL,
    vigente_de DATE NOT NULL,
    observacao VARCHAR(500) NOT NULL DEFAULT '',
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_tabela_preco_item PRIMARY KEY (id),
    CONSTRAINT chk_tabela_preco_item_modelo CHECK (modelo IN ('pronta_entrega', 'importacao')),
    CONSTRAINT chk_tabela_preco_item_preco CHECK (preco_centavos > 0),
    CONSTRAINT fk_tabela_preco_item_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT,
    CONSTRAINT fk_tabela_preco_item_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_tabela_preco_item ON tabela_preco_item (produto_id, modelo, vigente_de);
CREATE INDEX idx_tabela_preco_item_vigencia ON tabela_preco_item (modelo, vigente_de, id);
CREATE INDEX idx_tabela_preco_item_usuario ON tabela_preco_item (usuario_id);
