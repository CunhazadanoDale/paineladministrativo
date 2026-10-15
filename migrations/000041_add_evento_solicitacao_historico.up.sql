ALTER TABLE solicitacao_historico ADD COLUMN evento VARCHAR(20);

UPDATE solicitacao_historico
SET evento = CASE para_status
    WHEN 'pendente_aprovacao' THEN 'criada'
    WHEN 'aprovado' THEN 'aprovada'
    WHEN 'rejeitado' THEN 'rejeitada'
    WHEN 'cancelado' THEN 'cancelada'
    WHEN 'pago' THEN 'paga'
END;

ALTER TABLE solicitacao_historico
    ALTER COLUMN evento SET NOT NULL,
    ADD CONSTRAINT chk_sh_evento CHECK (
        evento IN (
            'criada', 'aprovada', 'rejeitada', 'cancelada', 'paga',
            'editada', 'classificada', 'adiada', 'estornada'
        )
    );
