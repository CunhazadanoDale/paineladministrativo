CREATE TABLE nota_fiscal (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    chave CHAR(44) NOT NULL,
    empresa_id UUID NOT NULL,
    numero INT NOT NULL,
    serie INT NOT NULL,
    emitida_em TIMESTAMPTZ NOT NULL,
    tipo_operacao VARCHAR(7) NOT NULL,
    finalidade VARCHAR(12) NOT NULL,
    natureza VARCHAR(13) NOT NULL,
    natureza_operacao VARCHAR(60) NOT NULL,
    entre_empresas BOOLEAN NOT NULL,
    cliente_id UUID,
    empresa_destinataria_id UUID,
    destinatario_documento VARCHAR(14) NOT NULL,
    destinatario_nome VARCHAR(200) NOT NULL,
    cfop_predominante CHAR(4) NOT NULL,
    valor_total_centavos BIGINT NOT NULL,
    valor_produtos_centavos BIGINT NOT NULL,
    valor_frete_centavos BIGINT NOT NULL,
    valor_desconto_centavos BIGINT NOT NULL,
    numero_pedido_cliente VARCHAR(60),
    protocolo VARCHAR(15) NOT NULL,
    autorizada_em TIMESTAMPTZ NOT NULL,
    status VARCHAR(10) NOT NULL,
    pedido_id UUID,
    entrega_id UUID,
    arquivo_id UUID NOT NULL,
    usuario_id UUID NOT NULL,
    versao INT NOT NULL DEFAULT 1,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_nota_fiscal PRIMARY KEY (id),
    CONSTRAINT uq_nota_fiscal_chave UNIQUE (chave),
    CONSTRAINT uq_nota_fiscal_arquivo UNIQUE (arquivo_id),
    CONSTRAINT chk_nota_fiscal_chave CHECK (chave ~ '^[0-9]{6}[0-9A-Z]{12}[0-9]{26}$'),
    CONSTRAINT chk_nota_fiscal_numero CHECK (numero BETWEEN 1 AND 999999999),
    CONSTRAINT chk_nota_fiscal_serie CHECK (serie BETWEEN 0 AND 999),
    CONSTRAINT chk_nota_fiscal_tipo_operacao CHECK (tipo_operacao IN ('saida', 'entrada')),
    CONSTRAINT chk_nota_fiscal_finalidade CHECK (finalidade IN ('normal', 'complementar', 'ajuste', 'devolucao')),
    CONSTRAINT chk_nota_fiscal_natureza CHECK (natureza IN ('venda', 'remessa', 'transferencia', 'devolucao')),
    CONSTRAINT chk_nota_fiscal_natureza_operacao CHECK (btrim(natureza_operacao) <> ''),
    CONSTRAINT chk_nota_fiscal_destinatario CHECK (
        (entre_empresas AND empresa_destinataria_id IS NOT NULL AND cliente_id IS NULL)
        OR (NOT entre_empresas AND empresa_destinataria_id IS NULL AND cliente_id IS NOT NULL)
    ),
    CONSTRAINT chk_nota_fiscal_transferencia CHECK (entre_empresas = (natureza = 'transferencia')),
    CONSTRAINT chk_nota_fiscal_destinatario_documento CHECK (destinatario_documento ~ '^([0-9]{11}|[0-9A-Z]{12}[0-9]{2})$'),
    CONSTRAINT chk_nota_fiscal_destinatario_nome CHECK (btrim(destinatario_nome) <> ''),
    CONSTRAINT chk_nota_fiscal_cfop_predominante CHECK (cfop_predominante ~ '^[0-9]{4}$'),
    CONSTRAINT chk_nota_fiscal_valor_total CHECK (valor_total_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_valor_produtos CHECK (valor_produtos_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_valor_frete CHECK (valor_frete_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_valor_desconto CHECK (valor_desconto_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_numero_pedido_cliente CHECK (numero_pedido_cliente IS NULL OR btrim(numero_pedido_cliente) <> ''),
    CONSTRAINT chk_nota_fiscal_protocolo CHECK (protocolo ~ '^[0-9]{15}$'),
    CONSTRAINT chk_nota_fiscal_status CHECK (status IN ('autorizada', 'cancelada')),
    CONSTRAINT chk_nota_fiscal_entrega_pedido CHECK (entrega_id IS NULL OR pedido_id IS NOT NULL),
    CONSTRAINT chk_nota_fiscal_pedido_entre_empresas CHECK (NOT entre_empresas OR pedido_id IS NULL),
    CONSTRAINT chk_nota_fiscal_versao CHECK (versao >= 1),
    CONSTRAINT fk_nota_fiscal_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_cliente FOREIGN KEY (cliente_id) REFERENCES cliente (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_empresa_destinataria FOREIGN KEY (empresa_destinataria_id) REFERENCES empresa (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_pedido FOREIGN KEY (pedido_id) REFERENCES pedido (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_entrega FOREIGN KEY (entrega_id) REFERENCES entrega (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_arquivo FOREIGN KEY (arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT
);

CREATE INDEX idx_nota_fiscal_empresa_emissao ON nota_fiscal (empresa_id, emitida_em DESC, id);
CREATE INDEX idx_nota_fiscal_cliente ON nota_fiscal (cliente_id, emitida_em DESC, id);
CREATE INDEX idx_nota_fiscal_empresa_destinataria ON nota_fiscal (empresa_destinataria_id);
CREATE INDEX idx_nota_fiscal_pedido ON nota_fiscal (pedido_id);
CREATE INDEX idx_nota_fiscal_usuario ON nota_fiscal (usuario_id);
CREATE UNIQUE INDEX uq_nota_fiscal_entrega_vigente ON nota_fiscal (entrega_id)
    WHERE entrega_id IS NOT NULL AND status = 'autorizada';

CREATE TABLE nota_fiscal_item (
    nota_fiscal_id UUID NOT NULL,
    numero_item INT NOT NULL,
    codigo VARCHAR(60) NOT NULL,
    descricao VARCHAR(120) NOT NULL,
    produto_id UUID,
    ncm VARCHAR(8) NOT NULL,
    cfop CHAR(4) NOT NULL,
    unidade VARCHAR(6) NOT NULL,
    quantidade_decimilesimos BIGINT NOT NULL,
    valor_unitario VARCHAR(22) NOT NULL,
    valor_centavos BIGINT NOT NULL,
    desconto_centavos BIGINT NOT NULL,
    pedido_cliente VARCHAR(60),
    CONSTRAINT pk_nota_fiscal_item PRIMARY KEY (nota_fiscal_id, numero_item),
    CONSTRAINT chk_nota_fiscal_item_numero CHECK (numero_item BETWEEN 1 AND 990),
    CONSTRAINT chk_nota_fiscal_item_codigo CHECK (btrim(codigo) <> ''),
    CONSTRAINT chk_nota_fiscal_item_descricao CHECK (btrim(descricao) <> ''),
    CONSTRAINT chk_nota_fiscal_item_ncm CHECK (ncm ~ '^([0-9]{2}|[0-9]{8})$'),
    CONSTRAINT chk_nota_fiscal_item_cfop CHECK (cfop ~ '^[0-9]{4}$'),
    CONSTRAINT chk_nota_fiscal_item_unidade CHECK (btrim(unidade) <> ''),
    CONSTRAINT chk_nota_fiscal_item_quantidade CHECK (quantidade_decimilesimos > 0),
    CONSTRAINT chk_nota_fiscal_item_valor_unitario CHECK (valor_unitario ~ '^[0-9]{1,11}(\.[0-9]{1,10})?$'),
    CONSTRAINT chk_nota_fiscal_item_valor CHECK (valor_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_item_desconto CHECK (desconto_centavos >= 0),
    CONSTRAINT chk_nota_fiscal_item_pedido_cliente CHECK (pedido_cliente IS NULL OR btrim(pedido_cliente) <> ''),
    CONSTRAINT fk_nota_fiscal_item_nota_fiscal FOREIGN KEY (nota_fiscal_id) REFERENCES nota_fiscal (id) ON DELETE RESTRICT,
    CONSTRAINT fk_nota_fiscal_item_produto FOREIGN KEY (produto_id) REFERENCES produto (id) ON DELETE RESTRICT
);

CREATE INDEX idx_nota_fiscal_item_produto ON nota_fiscal_item (produto_id);
CREATE INDEX idx_nota_fiscal_item_sem_produto ON nota_fiscal_item (nota_fiscal_id, numero_item) WHERE produto_id IS NULL;

CREATE TABLE nota_fiscal_duplicata (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nota_fiscal_id UUID NOT NULL,
    numero VARCHAR(60) NOT NULL,
    vencimento DATE NOT NULL,
    valor_centavos BIGINT NOT NULL,
    CONSTRAINT pk_nota_fiscal_duplicata PRIMARY KEY (id),
    CONSTRAINT uq_nota_fiscal_duplicata_numero UNIQUE (nota_fiscal_id, numero),
    CONSTRAINT chk_nota_fiscal_duplicata_numero CHECK (btrim(numero) <> ''),
    CONSTRAINT chk_nota_fiscal_duplicata_valor CHECK (valor_centavos > 0),
    CONSTRAINT fk_nota_fiscal_duplicata_nota_fiscal FOREIGN KEY (nota_fiscal_id) REFERENCES nota_fiscal (id) ON DELETE RESTRICT
);

CREATE INDEX idx_nota_fiscal_duplicata_vencimento ON nota_fiscal_duplicata (vencimento, id);
