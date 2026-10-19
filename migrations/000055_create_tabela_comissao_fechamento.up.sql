ALTER TABLE representante
    ADD COLUMN fornecedor_id UUID,
    ADD CONSTRAINT fk_representante_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT;

CREATE INDEX idx_representante_fornecedor ON representante (fornecedor_id) WHERE fornecedor_id IS NOT NULL;

ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN ('manual', 'recorrencia', 'recompra', 'repasse_fomento', 'comissao'));

CREATE TABLE comissao_fechamento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    representante_id UUID NOT NULL,
    competencia DATE NOT NULL,
    comissoes_centavos BIGINT NOT NULL,
    ajustes_centavos BIGINT NOT NULL,
    total_centavos BIGINT NOT NULL,
    vencimento DATE NOT NULL,
    categoria_financeira_id UUID NOT NULL,
    solicitacao_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_comissao_fechamento PRIMARY KEY (id),
    CONSTRAINT chk_comissao_fechamento_competencia CHECK (competencia = date_trunc('month', competencia)::date),
    CONSTRAINT chk_comissao_fechamento_comissoes CHECK (comissoes_centavos >= 0),
    CONSTRAINT chk_comissao_fechamento_total CHECK (total_centavos > 0 AND total_centavos = comissoes_centavos + ajustes_centavos),
    CONSTRAINT chk_comissao_fechamento_vencimento CHECK (vencimento = (competencia + INTERVAL '1 month 14 days')::date),
    CONSTRAINT fk_comissao_fechamento_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_fechamento_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_fechamento_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_fechamento_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_fechamento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_comissao_fechamento ON comissao_fechamento (empresa_id, representante_id, competencia);
CREATE UNIQUE INDEX uq_comissao_fechamento_solicitacao ON comissao_fechamento (solicitacao_id);
CREATE INDEX idx_comissao_fechamento_representante ON comissao_fechamento (representante_id, competencia, id);
CREATE INDEX idx_comissao_fechamento_categoria_financeira ON comissao_fechamento (categoria_financeira_id);
CREATE INDEX idx_comissao_fechamento_usuario ON comissao_fechamento (usuario_id);

ALTER TABLE comissao
    ADD COLUMN fechamento_id UUID,
    ADD CONSTRAINT fk_comissao_fechamento FOREIGN KEY (fechamento_id) REFERENCES comissao_fechamento (id) ON DELETE RESTRICT;

CREATE INDEX idx_comissao_fechamento ON comissao (fechamento_id) WHERE fechamento_id IS NOT NULL;
CREATE INDEX idx_comissao_aberta ON comissao (empresa_id, representante_id, competencia, id) WHERE fechamento_id IS NULL;

ALTER TABLE comissao_ajuste
    ADD COLUMN fechamento_id UUID,
    ADD CONSTRAINT fk_comissao_ajuste_fechamento FOREIGN KEY (fechamento_id) REFERENCES comissao_fechamento (id) ON DELETE RESTRICT;

CREATE INDEX idx_comissao_ajuste_fechamento ON comissao_ajuste (fechamento_id) WHERE fechamento_id IS NOT NULL;
CREATE INDEX idx_comissao_ajuste_aberto ON comissao_ajuste (empresa_id, representante_id, competencia, id) WHERE fechamento_id IS NULL;
