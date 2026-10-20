CREATE TABLE importacao_sequencia (
    empresa_id UUID NOT NULL,
    ultimo_numero INT NOT NULL,
    CONSTRAINT pk_importacao_sequencia PRIMARY KEY (empresa_id),
    CONSTRAINT chk_importacao_sequencia_ultimo_numero CHECK (ultimo_numero >= 1),
    CONSTRAINT fk_importacao_sequencia_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT
);

CREATE TABLE importacao (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    numero INT NOT NULL,
    fornecedor_id UUID NOT NULL,
    referencia VARCHAR(60),
    moeda CHAR(3) NOT NULL,
    status VARCHAR(14) NOT NULL,
    valor_total_moeda_centesimos BIGINT NOT NULL,
    embarque_previsto DATE,
    embarcada_em DATE,
    chegada_prevista DATE,
    chegada_em DATE,
    desembaracada_em DATE,
    documento_embarque VARCHAR(60),
    declaracao VARCHAR(30),
    observacoes VARCHAR(2000) NOT NULL DEFAULT '',
    cancelada_motivo VARCHAR(1000),
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao PRIMARY KEY (id),
    CONSTRAINT uq_importacao_empresa_numero UNIQUE (empresa_id, numero),
    CONSTRAINT chk_importacao_numero CHECK (numero >= 1),
    CONSTRAINT chk_importacao_moeda CHECK (moeda IN ('BRL', 'USD', 'EUR', 'CNY')),
    CONSTRAINT chk_importacao_status CHECK (
        status IN ('aberta', 'embarcada', 'desembaracada', 'recebida', 'cancelada')
    ),
    CONSTRAINT chk_importacao_valor_total CHECK (valor_total_moeda_centesimos >= 0),
    CONSTRAINT chk_importacao_referencia CHECK (referencia IS NULL OR btrim(referencia) <> ''),
    CONSTRAINT chk_importacao_documento_embarque CHECK (documento_embarque IS NULL OR btrim(documento_embarque) <> ''),
    CONSTRAINT chk_importacao_declaracao CHECK (declaracao IS NULL OR btrim(declaracao) <> ''),
    CONSTRAINT chk_importacao_previsao CHECK (
        embarque_previsto IS NULL OR chegada_prevista IS NULL OR chegada_prevista >= embarque_previsto
    ),
    CONSTRAINT chk_importacao_embarque CHECK (
        (status <> 'aberta' OR embarcada_em IS NULL)
        AND (status NOT IN ('embarcada', 'desembaracada', 'recebida') OR embarcada_em IS NOT NULL)
        AND (embarcada_em IS NULL OR chegada_prevista IS NOT NULL)
    ),
    CONSTRAINT chk_importacao_desembaraco CHECK (
        (chegada_em IS NULL) = (desembaracada_em IS NULL)
        AND (desembaracada_em IS NULL) = (declaracao IS NULL)
        AND (desembaracada_em IS NULL OR embarcada_em IS NOT NULL)
        AND (chegada_em IS NULL OR chegada_em >= embarcada_em)
        AND (desembaracada_em IS NULL OR desembaracada_em >= chegada_em)
        AND (status NOT IN ('aberta', 'embarcada') OR desembaracada_em IS NULL)
        AND (status NOT IN ('desembaracada', 'recebida') OR desembaracada_em IS NOT NULL)
    ),
    CONSTRAINT chk_importacao_cancelada_motivo CHECK (
        (status = 'cancelada') = (cancelada_motivo IS NOT NULL)
        AND (cancelada_motivo IS NULL OR btrim(cancelada_motivo) <> '')
    ),
    CONSTRAINT chk_importacao_versao CHECK (versao >= 1),
    CONSTRAINT fk_importacao_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_fornecedor FOREIGN KEY (fornecedor_id) REFERENCES fornecedor (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_empresa ON importacao (empresa_id, criado_em DESC, id);
CREATE INDEX idx_importacao_empresa_status ON importacao (empresa_id, status);
CREATE INDEX idx_importacao_fornecedor ON importacao (fornecedor_id);
CREATE INDEX idx_importacao_usuario ON importacao (usuario_id);
CREATE INDEX idx_importacao_chegada_prevista ON importacao (chegada_prevista, id)
    WHERE chegada_prevista IS NOT NULL AND status IN ('aberta', 'embarcada', 'desembaracada');

CREATE TABLE importacao_item (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    importacao_id UUID NOT NULL,
    produto_id UUID NOT NULL,
    quantidade INT NOT NULL,
    valor_moeda_centesimos BIGINT NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao_item PRIMARY KEY (id),
    CONSTRAINT uq_importacao_item_produto UNIQUE (importacao_id, produto_id),
    CONSTRAINT chk_importacao_item_quantidade CHECK (quantidade > 0),
    CONSTRAINT chk_importacao_item_valor CHECK (valor_moeda_centesimos >= 0),
    CONSTRAINT fk_importacao_item_importacao FOREIGN KEY (importacao_id) REFERENCES importacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_item_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_item_produto ON importacao_item (produto_id);

CREATE TABLE importacao_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    importacao_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    acao VARCHAR(30) NOT NULL,
    status_anterior VARCHAR(14),
    status_novo VARCHAR(14) NOT NULL,
    detalhe VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_importacao_historico PRIMARY KEY (id),
    CONSTRAINT chk_importacao_historico_acao CHECK (acao IN (
        'criada', 'alterada', 'embarcada', 'desembaracada', 'cancelada'
    )),
    CONSTRAINT chk_importacao_historico_status_anterior CHECK (
        status_anterior IS NULL
        OR status_anterior IN ('aberta', 'embarcada', 'desembaracada', 'recebida', 'cancelada')
    ),
    CONSTRAINT chk_importacao_historico_status_novo CHECK (
        status_novo IN ('aberta', 'embarcada', 'desembaracada', 'recebida', 'cancelada')
    ),
    CONSTRAINT fk_importacao_historico_importacao FOREIGN KEY (importacao_id) REFERENCES importacao (id) ON DELETE RESTRICT,
    CONSTRAINT fk_importacao_historico_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_importacao_historico_importacao ON importacao_historico (importacao_id, criado_em DESC, id DESC);
CREATE INDEX idx_importacao_historico_usuario ON importacao_historico (usuario_id);
