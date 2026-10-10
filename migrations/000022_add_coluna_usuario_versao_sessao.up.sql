ALTER TABLE usuario ADD COLUMN versao_sessao INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usuario ADD CONSTRAINT chk_usuario_versao_sessao CHECK (versao_sessao >= 0);
