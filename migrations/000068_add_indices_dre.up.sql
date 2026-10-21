CREATE INDEX idx_nota_fiscal_emissao_autorizada ON nota_fiscal (emitida_em, empresa_id) WHERE status = 'autorizada';
CREATE INDEX idx_produto_movimento_consumo ON produto_movimento (criado_em, local_id) WHERE tipo IN ('saida', 'perda', 'ajuste');
CREATE INDEX idx_solicitacao_competencia ON solicitacao (competencia, empresa_id) WHERE status NOT IN ('cancelado', 'rejeitado');
CREATE INDEX idx_titulo_receber_competencia_manual ON titulo_receber (competencia, empresa_id) WHERE origem = 'manual' AND status <> 'cancelado';
CREATE INDEX idx_titulo_recebimento_recebido_em ON titulo_recebimento (recebido_em) WHERE NOT estornado;
CREATE INDEX idx_antecipacao_operado_em ON antecipacao (operado_em, empresa_id) WHERE status = 'ativa';
CREATE INDEX idx_pagamento_pago_em ON pagamento (pago_em) WHERE NOT estornado;
