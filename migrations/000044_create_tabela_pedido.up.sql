CREATE TABLE pedido_sequencia (
    empresa_id UUID NOT NULL,
    ultimo_numero INT NOT NULL,
    CONSTRAINT pk_pedido_sequencia PRIMARY KEY (empresa_id),
    CONSTRAINT chk_pedido_sequencia_ultimo_numero CHECK (ultimo_numero >= 1),
    CONSTRAINT fk_pedido_sequencia_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT
);

CREATE TABLE pedido (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    numero INT NOT NULL,
    cliente_id UUID NOT NULL,
    obra_id UUID,
    representante_id UUID,
    modelo VARCHAR(16) NOT NULL,
    entrada_bp INT NOT NULL,
    prazo_boleto_dias INT,
    prazo_entrega_dias INT,
    prazo_limite DATE,
    numero_pedido_cliente VARCHAR(60),
    observacoes VARCHAR(2000) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL,
    valor_total_centavos BIGINT NOT NULL,
    entrada_paga_em DATE,
    entrada_paga_centavos BIGINT,
    cancelado_motivo VARCHAR(1000),
    valor_a_devolver_centavos BIGINT,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pedido PRIMARY KEY (id),
    CONSTRAINT uq_pedido_empresa_numero UNIQUE (empresa_id, numero),
    CONSTRAINT chk_pedido_numero CHECK (numero >= 1),
    CONSTRAINT chk_pedido_modelo CHECK (modelo IN ('pronta_entrega', 'importacao')),
    CONSTRAINT chk_pedido_status CHECK (
        status IN ('proposta', 'confirmado', 'parcialmente_entregue', 'entregue', 'cancelado')
    ),
    CONSTRAINT chk_pedido_entrada_bp CHECK (entrada_bp BETWEEN 0 AND 10000),
    CONSTRAINT chk_pedido_prazo_boleto_dias CHECK (prazo_boleto_dias IS NULL OR prazo_boleto_dias > 0),
    CONSTRAINT chk_pedido_prazo_entrega_dias CHECK (prazo_entrega_dias IS NULL OR prazo_entrega_dias > 0),
    CONSTRAINT chk_pedido_numero_pedido_cliente CHECK (numero_pedido_cliente IS NULL OR btrim(numero_pedido_cliente) <> ''),
    CONSTRAINT chk_pedido_valor_total CHECK (valor_total_centavos >= 0),
    CONSTRAINT chk_pedido_entrada_paga CHECK ((entrada_paga_em IS NULL) = (entrada_paga_centavos IS NULL)),
    CONSTRAINT chk_pedido_entrada_paga_valor CHECK (entrada_paga_centavos IS NULL OR entrada_paga_centavos > 0),
    CONSTRAINT chk_pedido_entrada_paga_modelo CHECK (entrada_paga_em IS NULL OR modelo = 'importacao'),
    CONSTRAINT chk_pedido_cancelado_motivo CHECK (
        (status = 'cancelado') = (cancelado_motivo IS NOT NULL)
        AND (cancelado_motivo IS NULL OR btrim(cancelado_motivo) <> '')
    ),
    CONSTRAINT chk_pedido_valor_a_devolver CHECK (
        valor_a_devolver_centavos IS NULL
        OR (
            status = 'cancelado'
            AND entrada_paga_centavos IS NOT NULL
            AND valor_a_devolver_centavos BETWEEN 0 AND entrada_paga_centavos
        )
    ),
    CONSTRAINT chk_pedido_versao CHECK (versao >= 1),
    CONSTRAINT fk_pedido_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_cliente FOREIGN KEY (cliente_id) REFERENCES cliente (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_obra FOREIGN KEY (obra_id) REFERENCES cliente_obra (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_representante FOREIGN KEY (representante_id) REFERENCES representante (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_pedido_empresa ON pedido (empresa_id, criado_em DESC, id);
CREATE INDEX idx_pedido_empresa_status ON pedido (empresa_id, status);
CREATE INDEX idx_pedido_cliente ON pedido (cliente_id);
CREATE INDEX idx_pedido_obra ON pedido (obra_id) WHERE obra_id IS NOT NULL;
CREATE INDEX idx_pedido_representante ON pedido (representante_id) WHERE representante_id IS NOT NULL;
CREATE INDEX idx_pedido_usuario ON pedido (usuario_id);
CREATE INDEX idx_pedido_prazo_limite ON pedido (prazo_limite, id)
    WHERE prazo_limite IS NOT NULL AND status IN ('confirmado', 'parcialmente_entregue');

CREATE TABLE pedido_item (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pedido_id UUID NOT NULL,
    produto_id UUID NOT NULL,
    quantidade INT NOT NULL,
    preco_unitario_centavos BIGINT NOT NULL,
    quantidade_entregue INT NOT NULL DEFAULT 0,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pedido_item PRIMARY KEY (id),
    CONSTRAINT uq_pedido_item_produto UNIQUE (pedido_id, produto_id),
    CONSTRAINT chk_pedido_item_quantidade CHECK (quantidade > 0),
    CONSTRAINT chk_pedido_item_preco CHECK (preco_unitario_centavos >= 0),
    CONSTRAINT chk_pedido_item_quantidade_entregue CHECK (quantidade_entregue BETWEEN 0 AND quantidade),
    CONSTRAINT fk_pedido_item_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_item_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT
);

CREATE INDEX idx_pedido_item_produto ON pedido_item (produto_id);

CREATE TABLE pedido_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pedido_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    acao VARCHAR(30) NOT NULL,
    status_anterior VARCHAR(24),
    status_novo VARCHAR(24) NOT NULL,
    detalhe VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_pedido_historico PRIMARY KEY (id),
    CONSTRAINT chk_pedido_historico_acao CHECK (acao IN (
        'criado', 'alterado', 'confirmado', 'entrada_registrada', 'pendencias_reservadas',
        'entrega_criada', 'entrega_saiu', 'entrega_concluida', 'entrega_cancelada', 'cancelado'
    )),
    CONSTRAINT chk_pedido_historico_status_anterior CHECK (
        status_anterior IS NULL
        OR status_anterior IN ('proposta', 'confirmado', 'parcialmente_entregue', 'entregue', 'cancelado')
    ),
    CONSTRAINT chk_pedido_historico_status_novo CHECK (
        status_novo IN ('proposta', 'confirmado', 'parcialmente_entregue', 'entregue', 'cancelado')
    ),
    CONSTRAINT fk_pedido_historico_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_pedido_historico_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_pedido_historico_pedido ON pedido_historico (pedido_id, criado_em DESC, id DESC);
CREATE INDEX idx_pedido_historico_usuario ON pedido_historico (usuario_id);
