CREATE TABLE antecipacao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    fornecedor_id UUID NOT NULL,
    numero VARCHAR(30),
    operado_em DATE NOT NULL,
    taxa_bp INT NOT NULL DEFAULT 0,
    valor_bruto_centavos BIGINT NOT NULL,
    valor_liquido_centavos BIGINT NOT NULL,
    conta_bancaria_id UUID NOT NULL,
    status VARCHAR(10) NOT NULL,
    estorno_motivo VARCHAR(500),
    estornado_em TIMESTAMPTZ,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_antecipacao PRIMARY KEY (id),
    CONSTRAINT chk_antecipacao_numero CHECK (numero IS NULL OR btrim(numero) <> ''),
    CONSTRAINT chk_antecipacao_taxa CHECK (taxa_bp BETWEEN 0 AND 10000),
    CONSTRAINT chk_antecipacao_valor_bruto CHECK (valor_bruto_centavos > 0),
    CONSTRAINT chk_antecipacao_valor_liquido CHECK (valor_liquido_centavos > 0 AND valor_liquido_centavos <= valor_bruto_centavos),
    CONSTRAINT chk_antecipacao_status CHECK (status IN ('ativa', 'estornada')),
    CONSTRAINT chk_antecipacao_estorno CHECK (
        (status = 'estornada') = (estorno_motivo IS NOT NULL)
        AND (estorno_motivo IS NULL) = (estornado_em IS NULL)
        AND (estorno_motivo IS NULL OR btrim(estorno_motivo) <> '')
    ),
    CONSTRAINT chk_antecipacao_versao CHECK (versao >= 1),
    CONSTRAINT fk_antecipacao_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_antecipacao_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    CONSTRAINT fk_antecipacao_conta_bancaria FOREIGN KEY (conta_bancaria_id) REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_antecipacao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_antecipacao_empresa ON antecipacao (empresa_id, operado_em DESC, id);
CREATE INDEX idx_antecipacao_fornecedor ON antecipacao (fornecedor_id);
CREATE INDEX idx_antecipacao_conta_bancaria ON antecipacao (conta_bancaria_id, operado_em);
CREATE INDEX idx_antecipacao_usuario ON antecipacao (usuario_id);

CREATE TABLE antecipacao_titulo (
    antecipacao_id UUID NOT NULL,
    titulo_id UUID NOT NULL,
    valor_centavos BIGINT NOT NULL,
    status_anterior VARCHAR(12) NOT NULL,
    vigente BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT pk_antecipacao_titulo PRIMARY KEY (antecipacao_id, titulo_id),
    CONSTRAINT chk_antecipacao_titulo_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_antecipacao_titulo_status_anterior CHECK (status_anterior IN ('carteira', 'em_banco')),
    CONSTRAINT fk_antecipacao_titulo_antecipacao FOREIGN KEY (antecipacao_id) REFERENCES antecipacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_antecipacao_titulo_titulo FOREIGN KEY (titulo_id) REFERENCES titulo_receber (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_antecipacao_titulo_vigente ON antecipacao_titulo (titulo_id) WHERE vigente;
CREATE INDEX idx_antecipacao_titulo_titulo ON antecipacao_titulo (titulo_id);

CREATE TABLE recompra (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    titulo_id UUID NOT NULL,
    antecipacao_id UUID NOT NULL,
    recomprado_em DATE NOT NULL,
    encargos_centavos BIGINT NOT NULL DEFAULT 0,
    valor_centavos BIGINT NOT NULL,
    vencimento DATE NOT NULL,
    categoria_financeira_id UUID NOT NULL,
    solicitacao_id UUID NOT NULL,
    motivo VARCHAR(500) NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_recompra PRIMARY KEY (id),
    CONSTRAINT uq_recompra_solicitacao UNIQUE (solicitacao_id),
    CONSTRAINT chk_recompra_encargos CHECK (encargos_centavos >= 0),
    CONSTRAINT chk_recompra_valor CHECK (valor_centavos > encargos_centavos),
    CONSTRAINT chk_recompra_motivo CHECK (btrim(motivo) <> ''),
    CONSTRAINT fk_recompra_antecipacao_titulo FOREIGN KEY (antecipacao_id, titulo_id)
        REFERENCES antecipacao_titulo (antecipacao_id, titulo_id) ON DELETE RESTRICT,
    CONSTRAINT fk_recompra_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_recompra_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_recompra_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_recompra_titulo ON recompra (titulo_id, criado_em);
CREATE INDEX idx_recompra_categoria_financeira ON recompra (categoria_financeira_id);
CREATE INDEX idx_recompra_usuario ON recompra (usuario_id);

CREATE TABLE repasse_fomento (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    recebimento_id UUID NOT NULL,
    antecipacao_id UUID NOT NULL,
    valor_centavos BIGINT NOT NULL,
    vencimento DATE NOT NULL,
    categoria_financeira_id UUID NOT NULL,
    solicitacao_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_repasse_fomento PRIMARY KEY (id),
    CONSTRAINT uq_repasse_fomento_recebimento UNIQUE (recebimento_id),
    CONSTRAINT uq_repasse_fomento_solicitacao UNIQUE (solicitacao_id),
    CONSTRAINT chk_repasse_fomento_valor CHECK (valor_centavos > 0),
    CONSTRAINT fk_repasse_fomento_recebimento FOREIGN KEY (recebimento_id) REFERENCES titulo_recebimento (id) ON DELETE RESTRICT,
    CONSTRAINT fk_repasse_fomento_antecipacao FOREIGN KEY (antecipacao_id) REFERENCES antecipacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_repasse_fomento_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_repasse_fomento_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_repasse_fomento_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_repasse_fomento_antecipacao ON repasse_fomento (antecipacao_id);
CREATE INDEX idx_repasse_fomento_categoria_financeira ON repasse_fomento (categoria_financeira_id);
CREATE INDEX idx_repasse_fomento_usuario ON repasse_fomento (usuario_id);
