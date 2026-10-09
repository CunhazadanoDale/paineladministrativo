CREATE TABLE produto (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    categoria_id UUID NOT NULL,
    nome VARCHAR(200) NOT NULL,
    slug VARCHAR(220) NOT NULL,
    descricao TEXT NOT NULL DEFAULT '',
    codigo VARCHAR(60),
    unidade_medida VARCHAR(10) NOT NULL,
    preco BIGINT,
    preco_promocional BIGINT,
    quantidade_atual INT NOT NULL DEFAULT 0,
    estoque_minimo INT,
    peso_kg NUMERIC(10,3),
    destaque BOOLEAN NOT NULL DEFAULT FALSE,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_produto PRIMARY KEY (id),
    CONSTRAINT chk_produto_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_produto_slug CHECK (btrim(slug) <> ''),
    CONSTRAINT chk_produto_unidade_medida CHECK (
        unidade_medida IN ('un', 'm', 'm2', 'm3', 'kg', 't', 'cx', 'sc', 'pct', 'lt')
    ),
    CONSTRAINT chk_produto_preco CHECK (preco IS NULL OR preco > 0),
    CONSTRAINT chk_produto_preco_promocional CHECK (preco_promocional IS NULL OR preco_promocional > 0),
    CONSTRAINT chk_produto_promocional_exige_preco CHECK (preco_promocional IS NULL OR preco IS NOT NULL),
    CONSTRAINT chk_produto_promocional_menor CHECK (preco_promocional IS NULL OR preco_promocional < preco),
    CONSTRAINT chk_produto_quantidade_atual CHECK (quantidade_atual >= 0),
    CONSTRAINT chk_produto_estoque_minimo CHECK (estoque_minimo IS NULL OR estoque_minimo >= 0),
    CONSTRAINT chk_produto_peso_kg CHECK (peso_kg IS NULL OR peso_kg >= 0),
    CONSTRAINT fk_produto_categoria FOREIGN KEY (categoria_id) REFERENCES categoria (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_produto_slug ON produto (slug);
CREATE UNIQUE INDEX uq_produto_codigo ON produto (codigo) WHERE codigo IS NOT NULL;
CREATE INDEX idx_produto_categoria ON produto (categoria_id, nome);
CREATE INDEX idx_produto_ativo ON produto (ativo, nome);
CREATE INDEX idx_produto_nome ON produto (nome);
CREATE INDEX idx_produto_destaque ON produto (destaque, nome) WHERE destaque AND ativo;
CREATE INDEX idx_produto_estoque_baixo ON produto (estoque_minimo) WHERE ativo AND estoque_minimo IS NOT NULL;
