ALTER TABLE usuario ADD COLUMN acesso_todas_empresas BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE usuario SET acesso_todas_empresas = TRUE;

CREATE TABLE usuario_empresa (
    usuario_id UUID NOT NULL,
    empresa_id UUID NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_usuario_empresa PRIMARY KEY (usuario_id, empresa_id),
    CONSTRAINT fk_usuario_empresa_usuario FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE CASCADE,
    CONSTRAINT fk_usuario_empresa_empresa FOREIGN KEY (empresa_id) REFERENCES empresa (id) ON DELETE RESTRICT
);

CREATE INDEX idx_usuario_empresa_empresa ON usuario_empresa (empresa_id);
