CREATE INDEX IF NOT EXISTS idx_solicitacao_aprovador ON solicitacao (aprovador_id, status) WHERE aprovador_id IS NOT NULL;
