ALTER TABLE solicitacao
    ADD COLUMN fornecedor_id UUID,
    ADD COLUMN categoria_financeira_id UUID,
    ADD COLUMN competencia DATE,
    ADD COLUMN estimado BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN origem VARCHAR(20) NOT NULL DEFAULT 'manual',
    ADD COLUMN origem_id UUID,
    ADD COLUMN versao INT NOT NULL DEFAULT 1;

UPDATE solicitacao SET competencia = date_trunc('month', prazo_pagamento)::date;

ALTER TABLE solicitacao
    ALTER COLUMN competencia SET NOT NULL,
    ALTER COLUMN forma_pagamento TYPE VARCHAR(20),
    DROP CONSTRAINT chk_solicitacao_forma_pagamento,
    ADD CONSTRAINT chk_solicitacao_forma_pagamento CHECK (
        forma_pagamento IN ('pix', 'cartao', 'boleto', 'transferencia', 'debito_automatico')
    ),
    ADD CONSTRAINT chk_solicitacao_competencia CHECK (competencia = date_trunc('month', competencia)::date),
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia')),
    ADD CONSTRAINT chk_solicitacao_origem_id CHECK ((origem = 'manual') = (origem_id IS NULL)),
    ADD CONSTRAINT chk_solicitacao_versao CHECK (versao >= 1),
    ADD CONSTRAINT fk_solicitacao_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_solicitacao_categoria_financeira
        FOREIGN KEY (categoria_financeira_id) REFERENCES categoria_financeira (id) ON DELETE RESTRICT;

CREATE INDEX idx_solicitacao_vencimento ON solicitacao (empresa_id, status, prazo_pagamento, id);
CREATE INDEX idx_solicitacao_em_aberto
    ON solicitacao (prazo_pagamento) WHERE status IN ('pendente_aprovacao', 'aprovado');
CREATE INDEX idx_solicitacao_fornecedor ON solicitacao (fornecedor_id) WHERE fornecedor_id IS NOT NULL;
CREATE INDEX idx_solicitacao_categoria_financeira
    ON solicitacao (categoria_financeira_id) WHERE categoria_financeira_id IS NOT NULL;
