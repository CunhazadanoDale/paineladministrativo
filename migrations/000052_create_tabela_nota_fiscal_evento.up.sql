CREATE TABLE nota_fiscal_evento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nota_fiscal_id UUID NOT NULL,
    tipo VARCHAR(16) NOT NULL,
    sequencia INT NOT NULL,
    ocorrido_em TIMESTAMPTZ NOT NULL,
    protocolo VARCHAR(15) NOT NULL,
    justificativa VARCHAR(1000) NOT NULL,
    arquivo_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_nota_fiscal_evento PRIMARY KEY (id),
    CONSTRAINT uq_nota_fiscal_evento UNIQUE (nota_fiscal_id, tipo, sequencia),
    CONSTRAINT uq_nota_fiscal_evento_arquivo UNIQUE (arquivo_id),
    CONSTRAINT chk_nota_fiscal_evento_tipo CHECK (tipo IN ('cancelamento', 'carta_correcao')),
    CONSTRAINT chk_nota_fiscal_evento_sequencia CHECK (sequencia BETWEEN 1 AND 20),
    CONSTRAINT chk_nota_fiscal_evento_protocolo CHECK (protocolo ~ '^[0-9]{15}$'),
    CONSTRAINT chk_nota_fiscal_evento_justificativa CHECK (btrim(justificativa) <> ''),
    CONSTRAINT fk_nota_fiscal_evento_nota_fiscal FOREIGN KEY (nota_fiscal_id) REFERENCES nota_fiscal (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_evento_arquivo FOREIGN KEY (arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_evento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_nota_fiscal_evento_nota_fiscal ON nota_fiscal_evento (nota_fiscal_id, ocorrido_em, id);
CREATE INDEX idx_nota_fiscal_evento_usuario ON nota_fiscal_evento (usuario_id);
