ALTER TABLE solicitacao
    DROP CONSTRAINT chk_solicitacao_origem,
    ADD CONSTRAINT chk_solicitacao_origem CHECK (origem IN (
        'manual', 'recorrencia', 'recompra', 'repasse_fomento', 'comissao', 'importacao_parcela', 'importacao_custo',
        'divida_amortizacao', 'divida_juros'
    ));

CREATE TABLE divida (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    credor_tipo VARCHAR(12) NOT NULL,
    fornecedor_id UUID NOT NULL,
    descricao VARCHAR(120) NOT NULL,
    contrato VARCHAR(60),
    valor_original_centavos BIGINT,
    contratada_em DATE,
    taxa_bp INT,
    taxa_periodo VARCHAR(6),
    indexador VARCHAR(10) NOT NULL DEFAULT 'prefixado',
    saldo_inicial_centavos BIGINT NOT NULL,
    data_saldo DATE NOT NULL,
    conta_liberacao_id UUID,
    liberada_em DATE,
    categoria_amortizacao_id UUID NOT NULL,
    categoria_juros_id UUID NOT NULL,
    divida_origem_id UUID,
    status VARCHAR(12) NOT NULL,
    motivo VARCHAR(1000),
    observacoes VARCHAR(2000) NOT NULL DEFAULT '',
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_divida PRIMARY KEY (id),
    CONSTRAINT uq_divida_origem UNIQUE (divida_origem_id),
    CONSTRAINT chk_divida_credor_tipo CHECK (credor_tipo IN ('banco', 'investidor', 'fisco', 'fornecedor')),
    CONSTRAINT chk_divida_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_divida_valor_original CHECK (valor_original_centavos IS NULL OR valor_original_centavos > 0),
    CONSTRAINT chk_divida_taxa CHECK (
        (taxa_bp IS NULL) = (taxa_periodo IS NULL)
        AND (taxa_bp IS NULL OR taxa_bp BETWEEN 0 AND 100000)
        AND (taxa_periodo IS NULL OR taxa_periodo IN ('mensal', 'anual'))
    ),
    CONSTRAINT chk_divida_indexador CHECK (indexador IN ('prefixado', 'cdi', 'selic', 'ipca', 'tr', 'outro')),
    CONSTRAINT chk_divida_saldo CHECK (saldo_inicial_centavos > 0),
    CONSTRAINT chk_divida_liberacao CHECK ((conta_liberacao_id IS NULL) = (liberada_em IS NULL)),
    CONSTRAINT chk_divida_status CHECK (status IN ('ativa', 'renegociada', 'cancelada')),
    CONSTRAINT chk_divida_motivo CHECK ((status = 'ativa') = (motivo IS NULL)),
    CONSTRAINT chk_divida_origem_propria CHECK (divida_origem_id IS NULL OR divida_origem_id <> id),
    CONSTRAINT fk_divida_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_conta_liberacao FOREIGN KEY (conta_liberacao_id) REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_categoria_amortizacao FOREIGN KEY (categoria_amortizacao_id) REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_categoria_juros FOREIGN KEY (categoria_juros_id) REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_origem FOREIGN KEY (divida_origem_id) REFERENCES divida (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_divida_empresa ON divida (empresa_id, status);
CREATE INDEX idx_divida_fornecedor ON divida (fornecedor_id);
CREATE INDEX idx_divida_conta_liberacao ON divida (conta_liberacao_id) WHERE conta_liberacao_id IS NOT NULL;
CREATE INDEX idx_divida_categoria_amortizacao ON divida (categoria_amortizacao_id);
CREATE INDEX idx_divida_categoria_juros ON divida (categoria_juros_id);
CREATE INDEX idx_divida_usuario ON divida (usuario_id);
CREATE INDEX idx_divida_criado_em ON divida (criado_em DESC, id DESC);

CREATE TABLE divida_parcela (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    divida_id UUID NOT NULL,
    numero INT NOT NULL,
    vencimento DATE NOT NULL,
    amortizacao_centavos BIGINT NOT NULL,
    juros_centavos BIGINT NOT NULL,
    status VARCHAR(10) NOT NULL,
    gerada_em TIMESTAMPTZ,
    substituta_id UUID,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_divida_parcela PRIMARY KEY (id),
    CONSTRAINT uq_divida_parcela_substituta UNIQUE (substituta_id),
    CONSTRAINT chk_divida_parcela_numero CHECK (numero > 0),
    CONSTRAINT chk_divida_parcela_valores CHECK (
        amortizacao_centavos >= 0 AND juros_centavos >= 0 AND amortizacao_centavos + juros_centavos > 0
    ),
    CONSTRAINT chk_divida_parcela_status CHECK (status IN ('prevista', 'gerada', 'cancelada')),
    CONSTRAINT chk_divida_parcela_gerada CHECK (
        (status <> 'prevista' OR gerada_em IS NULL) AND (status <> 'gerada' OR gerada_em IS NOT NULL)
    ),
    CONSTRAINT chk_divida_parcela_substituta CHECK (substituta_id IS NULL OR status = 'cancelada'),
    CONSTRAINT fk_divida_parcela_divida FOREIGN KEY (divida_id) REFERENCES divida (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_parcela_substituta FOREIGN KEY (substituta_id) REFERENCES divida_parcela (id) ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED
);

CREATE UNIQUE INDEX uq_divida_parcela_numero ON divida_parcela (divida_id, numero) WHERE status <> 'cancelada';
CREATE UNIQUE INDEX uq_divida_parcela_vencimento ON divida_parcela (divida_id, vencimento) WHERE status <> 'cancelada';
CREATE INDEX idx_divida_parcela_divida ON divida_parcela (divida_id, numero, criado_em, id);
CREATE INDEX idx_divida_parcela_prevista ON divida_parcela (vencimento) WHERE status = 'prevista';

CREATE TABLE divida_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    divida_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    acao VARCHAR(20) NOT NULL,
    status_anterior VARCHAR(12),
    status_novo VARCHAR(12) NOT NULL,
    detalhe VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_divida_historico PRIMARY KEY (id),
    CONSTRAINT chk_divida_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'parcela_alterada', 'contas_geradas', 'parcela_reaberta', 'renegociada', 'cancelada'
    )),
    CONSTRAINT fk_divida_historico_divida FOREIGN KEY (divida_id) REFERENCES divida (id) ON DELETE RESTRICT,
    CONSTRAINT fk_divida_historico_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_divida_historico_divida ON divida_historico (divida_id, criado_em, id);
CREATE INDEX idx_divida_historico_usuario ON divida_historico (usuario_id);
