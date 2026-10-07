CREATE TABLE solicitacao_arquivo (
    solicitacao_id UUID NOT NULL,
    arquivo_id UUID NOT NULL,
    CONSTRAINT pk_solicitacao_arquivo PRIMARY KEY (solicitacao_id, arquivo_id),
    CONSTRAINT fk_sa_solicitacao FOREIGN KEY (solicitacao_id) REFERENCES solicitacao (id) ON DELETE CASCADE,
    CONSTRAINT fk_sa_arquivo FOREIGN KEY (arquivo_id) REFERENCES arquivo (id) ON DELETE RESTRICT
);

CREATE INDEX idx_solicitacao_arquivo_arquivo ON solicitacao_arquivo (arquivo_id);
