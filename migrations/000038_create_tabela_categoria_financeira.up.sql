CREATE TABLE categoria_financeira (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(120) NOT NULL,
    categoria_pai_id UUID,
    natureza VARCHAR(13) NOT NULL,
    linha_dre VARCHAR(20) NOT NULL,
    codigo VARCHAR(20),
    ordem INT NOT NULL DEFAULT 0,
    ativa BOOLEAN NOT NULL DEFAULT TRUE,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_categoria_financeira PRIMARY KEY (id),
    CONSTRAINT chk_categoria_financeira_nome CHECK (btrim(nome) <> ''),
    CONSTRAINT chk_categoria_financeira_natureza CHECK (natureza IN ('receita', 'despesa', 'transferencia')),
    CONSTRAINT chk_categoria_financeira_linha_dre CHECK (linha_dre IN (
        'receita_operacional', 'deducao_venda', 'compra_mercadoria', 'despesa_variavel',
        'despesa_fixa', 'resultado_financeiro', 'fora_dre', 'entre_contas'
    )),
    CONSTRAINT chk_categoria_financeira_natureza_linha CHECK (
        (linha_dre = 'receita_operacional' AND natureza = 'receita')
        OR (linha_dre IN ('deducao_venda', 'compra_mercadoria', 'despesa_variavel', 'despesa_fixa') AND natureza = 'despesa')
        OR (linha_dre IN ('resultado_financeiro', 'fora_dre') AND natureza IN ('receita', 'despesa'))
        OR (linha_dre = 'entre_contas' AND natureza = 'transferencia')
    ),
    CONSTRAINT chk_categoria_financeira_codigo CHECK (codigo IS NULL OR codigo ~ '^[0-9]+(\.[0-9]+)*$'),
    CONSTRAINT chk_categoria_financeira_ordem CHECK (ordem >= 0),
    CONSTRAINT chk_categoria_financeira_versao CHECK (versao >= 1),
    CONSTRAINT chk_categoria_financeira_pai CHECK (categoria_pai_id IS NULL OR categoria_pai_id <> id),
    CONSTRAINT uq_categoria_financeira_classificacao UNIQUE (id, natureza, linha_dre),
    CONSTRAINT fk_categoria_financeira_pai FOREIGN KEY (categoria_pai_id, natureza, linha_dre)
        REFERENCES categoria_financeira (id, natureza, linha_dre) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_categoria_financeira_raiz_nome ON categoria_financeira (lower(nome)) WHERE categoria_pai_id IS NULL;
CREATE UNIQUE INDEX uq_categoria_financeira_sub_nome ON categoria_financeira (categoria_pai_id, lower(nome)) WHERE categoria_pai_id IS NOT NULL;
CREATE UNIQUE INDEX uq_categoria_financeira_codigo ON categoria_financeira (codigo) WHERE codigo IS NOT NULL;
CREATE INDEX idx_categoria_financeira_pai ON categoria_financeira (categoria_pai_id, ordem, nome, id);
