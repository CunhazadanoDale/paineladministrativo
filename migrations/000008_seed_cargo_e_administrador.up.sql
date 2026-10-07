INSERT INTO cargo (id, nome, descricao, administrador)
VALUES ('00000000-0000-0000-0000-0000000000ad', 'Administrador', 'Perfil com acesso total ao painel', TRUE);

INSERT INTO usuario (id, nome, email, senha, cargo_id, ativo)
VALUES (
    '00000000-0000-0000-0000-0000000000ad',
    'Administrador',
    'admin@exemplo.com',
    '$2a$10$Ac9mgZotONfqqhetwlN7D.adW9uobbZc9uKL3t6fsrWBIyem0jMCO',
    '00000000-0000-0000-0000-0000000000ad',
    TRUE
);
