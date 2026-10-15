CREATE TABLE despesa_recorrente (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    fornecedor_id UUID,
    categoria_financeira_id UUID NOT NULL,
    descricao VARCHAR(1000) NOT NULL,
    valor_centavos BIGINT NOT NULL,
    estimado BOOLEAN NOT NULL DEFAULT false,
    forma_pagamento VARCHAR(20) NOT NULL,
    dia_vencimento SMALLINT NOT NULL,
    vencimento_mes_seguinte BOOLEAN NOT NULL DEFAULT false,
    inicio DATE NOT NULL,
    fim DATE,
    ativa BOOLEAN NOT NULL DEFAULT true,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_despesa_recorrente PRIMARY KEY (id),
    CONSTRAINT chk_despesa_recorrente_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_despesa_recorrente_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_despesa_recorrente_forma_pagamento CHECK (
        forma_pagamento IN ('pix', 'cartao', 'boleto', 'transferencia', 'debito_automatico')
    ),
    CONSTRAINT chk_despesa_recorrente_dia_vencimento CHECK (dia_vencimento BETWEEN 1 AND 31),
    CONSTRAINT chk_despesa_recorrente_inicio CHECK (inicio = date_trunc('month', inicio)::date),
    CONSTRAINT chk_despesa_recorrente_fim CHECK (fim IS NULL OR (fim = date_trunc('month', fim)::date AND fim >= inicio)),
    CONSTRAINT chk_despesa_recorrente_versao CHECK (versao >= 1),
    CONSTRAINT fk_despesa_recorrente_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_despesa_recorrente_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    CONSTRAINT fk_despesa_recorrente_categoria_financeira
        FOREIGN KEY (categoria_financeira_id) REFERENCES categoria_financeira (id) ON DELETE RESTRICT
);

CREATE INDEX idx_despesa_recorrente_empresa ON despesa_recorrente (empresa_id, ativa, id);
CREATE INDEX idx_despesa_recorrente_fornecedor ON despesa_recorrente (fornecedor_id) WHERE fornecedor_id IS NOT NULL;
CREATE INDEX idx_despesa_recorrente_categoria_financeira ON despesa_recorrente (categoria_financeira_id);

CREATE UNIQUE INDEX uq_solicitacao_recorrencia ON solicitacao (origem_id, competencia) WHERE origem = 'recorrencia';
