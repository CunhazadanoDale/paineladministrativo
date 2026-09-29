CREATE TABLE funil (
    funil_id UUID NOT NULL DEFAULT gen_random_uuid(),
    nome VARCHAR(150) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT pk_funil PRIMARY KEY (funil_id),
    CONSTRAINT chk_funil_nome CHECK (btrim(nome) <> '')
);
