ALTER TABLE solicitacao_arquivo ADD CONSTRAINT uq_solicitacao_arquivo_arquivo UNIQUE (arquivo_id);
ALTER TABLE pagamento ADD CONSTRAINT uq_pagamento_comprovante UNIQUE (comprovante_arquivo_id);
