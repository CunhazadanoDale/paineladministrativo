CREATE TABLE produto_imagem (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL,
    arquivo_id UUID NOT NULL,
    ordem INT NOT NULL DEFAULT 0,
    alt VARCHAR(160) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_produto_imagem PRIMARY KEY (id),
    CONSTRAINT chk_produto_imagem_ordem CHECK (ordem >= 0),
    CONSTRAINT uq_produto_imagem UNIQUE (produto_id, arquivo_id),
    CONSTRAINT fk_produto_imagem_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE CASCADE,
    CONSTRAINT fk_produto_imagem_arquivo FOREIGN KEY (arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT
);

CREATE INDEX idx_produto_imagem_produto ON produto_imagem (produto_id, ordem);
