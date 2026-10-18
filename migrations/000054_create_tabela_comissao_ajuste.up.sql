CREATE TABLE comissao_ajuste (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    representante_id UUID NOT NULL,
    pedido_id UUID,
    competencia DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    motivo VARCHAR(1000) NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_comissao_ajuste PRIMARY KEY (id),
    CONSTRAINT chk_comissao_ajuste_competencia CHECK (competencia = date_trunc('month', competencia)::date),
    CONSTRAINT chk_comissao_ajuste_valor CHECK (valor_centavos <> 0),
    CONSTRAINT chk_comissao_ajuste_motivo CHECK (btrim(motivo) <> ''),
    CONSTRAINT fk_comissao_ajuste_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_ajuste_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_ajuste_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_comissao_ajuste_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_comissao_ajuste_competencia ON comissao_ajuste (empresa_id, competencia, representante_id, id);
CREATE INDEX idx_comissao_ajuste_representante ON comissao_ajuste (representante_id, competencia, id);
CREATE INDEX idx_comissao_ajuste_pedido ON comissao_ajuste (pedido_id) WHERE pedido_id IS NOT NULL;
CREATE INDEX idx_comissao_ajuste_usuario ON comissao_ajuste (usuario_id);
