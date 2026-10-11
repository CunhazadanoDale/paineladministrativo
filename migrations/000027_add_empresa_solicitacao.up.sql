DELETE FROM pagamento;

DELETE FROM solicitacao_historico;

DELETE FROM solicitacao_arquivo;

DELETE FROM solicitacao;

ALTER TABLE solicitacao ADD COLUMN empresa_id UUID NOT NULL;

ALTER TABLE solicitacao
    ADD CONSTRAINT fk_solicitacao_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT;

CREATE INDEX idx_solicitacao_empresa ON solicitacao (empresa_id, criado_em DESC);
