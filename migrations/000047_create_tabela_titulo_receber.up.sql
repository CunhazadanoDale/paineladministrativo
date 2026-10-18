CREATE TABLE titulo_receber (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL,
    cliente_id UUID NOT NULL,
    origem VARCHAR(30) NOT NULL,
    origem_id UUID,
    numero VARCHAR(30) NOT NULL,
    parcela SMALLINT NOT NULL,
    vencimento DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    categoria_financeira_id UUID,
    competencia DATE NOT NULL,
    conta_bancaria_id UUID,
    status VARCHAR(12) NOT NULL,
    recebido_centavos BIGINT NOT NULL DEFAULT 0,
    boleto_enviado_em TIMESTAMPTZ,
    boleto_canal VARCHAR(10),
    origem_cancelada_em TIMESTAMPTZ,
    cancelado_motivo VARCHAR(1000),
    observacao VARCHAR(1000) NOT NULL DEFAULT '',
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_titulo_receber PRIMARY KEY (id),
    CONSTRAINT chk_titulo_receber_origem CHECK (origem IN ('manual', 'pedido', 'nota_fiscal_duplicata')),
    CONSTRAINT chk_titulo_receber_origem_id CHECK ((origem = 'manual') = (origem_id IS NULL)),
    CONSTRAINT chk_titulo_receber_numero CHECK (btrim(numero) <> ''),
    CONSTRAINT chk_titulo_receber_parcela CHECK (parcela >= 1),
    CONSTRAINT chk_titulo_receber_valor CHECK (valor_centavos > 0),
    CONSTRAINT chk_titulo_receber_categoria CHECK ((origem = 'manual') = (categoria_financeira_id IS NOT NULL)),
    CONSTRAINT chk_titulo_receber_competencia CHECK (competencia = date_trunc('month', competencia)::date),
    CONSTRAINT chk_titulo_receber_status CHECK (
        status IN ('carteira', 'em_banco', 'antecipado', 'recebido', 'cancelado')
    ),
    CONSTRAINT chk_titulo_receber_conta_em_banco CHECK (status <> 'em_banco' OR conta_bancaria_id IS NOT NULL),
    CONSTRAINT chk_titulo_receber_recebido CHECK (recebido_centavos BETWEEN 0 AND valor_centavos),
    CONSTRAINT chk_titulo_receber_quitado CHECK ((status = 'recebido') = (recebido_centavos = valor_centavos)),
    CONSTRAINT chk_titulo_receber_boleto CHECK (
        (boleto_enviado_em IS NULL) = (boleto_canal IS NULL)
        AND (boleto_canal IS NULL OR boleto_canal IN ('email', 'whatsapp', 'portal', 'correio', 'outro'))
    ),
    CONSTRAINT chk_titulo_receber_cancelado_motivo CHECK (
        (status = 'cancelado') = (cancelado_motivo IS NOT NULL)
        AND (cancelado_motivo IS NULL OR btrim(cancelado_motivo) <> '')
    ),
    CONSTRAINT chk_titulo_receber_versao CHECK (versao >= 1),
    CONSTRAINT fk_titulo_receber_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_receber_cliente FOREIGN KEY (cliente_id) REFERENCES cliente (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_receber_categoria_financeira FOREIGN KEY (categoria_financeira_id)
        REFERENCES categoria_financeira (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_receber_conta_bancaria FOREIGN KEY (conta_bancaria_id)
        REFERENCES conta_bancaria (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_receber_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX uq_titulo_receber_origem ON titulo_receber (origem, origem_id) WHERE origem <> 'manual';
CREATE UNIQUE INDEX uq_titulo_receber_numero ON titulo_receber (empresa_id, origem, lower(numero), parcela)
    WHERE status <> 'cancelado';
CREATE INDEX idx_titulo_receber_vencimento ON titulo_receber (empresa_id, status, vencimento, id);
CREATE INDEX idx_titulo_receber_cliente ON titulo_receber (cliente_id, vencimento, id);
CREATE INDEX idx_titulo_receber_conta_bancaria ON titulo_receber (conta_bancaria_id) WHERE conta_bancaria_id IS NOT NULL;
CREATE INDEX idx_titulo_receber_categoria_financeira ON titulo_receber (categoria_financeira_id)
    WHERE categoria_financeira_id IS NOT NULL;
CREATE INDEX idx_titulo_receber_usuario ON titulo_receber (usuario_id);
CREATE INDEX idx_titulo_receber_origem_cancelada ON titulo_receber (empresa_id, vencimento, id)
    WHERE origem_cancelada_em IS NOT NULL;

CREATE TABLE titulo_receber_historico (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    titulo_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    evento VARCHAR(30) NOT NULL,
    de_status VARCHAR(12),
    para_status VARCHAR(12) NOT NULL,
    descricao VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_titulo_receber_historico PRIMARY KEY (id),
    CONSTRAINT chk_titulo_receber_historico_evento CHECK (evento IN (
        'criado', 'gerado', 'editado', 'prorrogado', 'enviado_banco', 'retirado_banco', 'boleto_enviado',
        'antecipado', 'antecipacao_estornada', 'recomprado', 'recebido', 'recebimento_estornado',
        'cancelado', 'origem_cancelada'
    )),
    CONSTRAINT chk_titulo_receber_historico_de_status CHECK (
        de_status IS NULL OR de_status IN ('carteira', 'em_banco', 'antecipado', 'recebido', 'cancelado')
    ),
    CONSTRAINT chk_titulo_receber_historico_para_status CHECK (
        para_status IN ('carteira', 'em_banco', 'antecipado', 'recebido', 'cancelado')
    ),
    CONSTRAINT fk_titulo_receber_historico_titulo FOREIGN KEY (titulo_id) REFERENCES titulo_receber (id) ON DELETE RESTRICT,
    CONSTRAINT fk_titulo_receber_historico_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_titulo_receber_historico_titulo ON titulo_receber_historico (titulo_id, criado_em, id);
CREATE INDEX idx_titulo_receber_historico_usuario ON titulo_receber_historico (usuario_id);
