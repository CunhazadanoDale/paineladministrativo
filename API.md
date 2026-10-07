# API

Documentação das rotas HTTP do painel administrativo.

- Base URL local: `http://localhost:8080` (docker compose) ou a porta definida em `PORT`
- Todas as rotas de negócio ficam sob `/api/v1`
- Todas as respostas são JSON (`application/json; charset=utf-8`), exceto `204` e `405`

---

## Variáveis de ambiente

| Nome | Obrigatória | Padrão | Descrição |
| --- | --- | --- | --- |
| `PORT` | não | `8080` | Porta HTTP do servidor |
| `DATABASE_URL` | sim | — | String de conexão PostgreSQL, ex.: `postgres://mia:mia@localhost:5444/mia?sslmode=disable` |
| `CORS_ORIGINS` | não | libera todas | Origens separadas por vírgula. Ex.: `http://localhost:5173`. Vazio ou `*` libera qualquer origem |
| `JWT_SECRET` | **sim** | — | Segredo usado para assinar os tokens (HS256). A API **não sobe** se estiver vazio |
| `JWT_EXPIRA_MINUTOS` | não | `480` (8 h) | Validade do token emitido no login. Valores vazios ou inválidos caem no padrão |
| `STORAGE_DRIVER` | não | `disco` | Onde os anexos são gravados: `disco`, `r2` (Cloudflare R2) |
| `STORAGE_DIR` | não | `storage_local` | Diretório do driver `disco` (relativo à raiz da aplicação) |
| `R2_ACCOUNT_ID` | com `r2` | — | Conta do Cloudflare R2 |
| `R2_ACCESS_KEY_ID` | com `r2` | — | Chave de acesso do R2 |
| `R2_SECRET_ACCESS_KEY` | com `r2` | — | Segredo da chave do R2 |
| `R2_BUCKET` | com `r2` | — | Nome do bucket |

Use `.env.example` como ponto de partida: copie para `.env` e preencha o `JWT_SECRET`
(gerar um: `openssl rand -hex 32`).

`STORAGE_DRIVER=r2` **sem** as quatro variáveis `R2_*` preenchidas derruba a subida da API
(fail-closed), do mesmo jeito que um `JWT_SECRET` vazio.

---

## Autenticação

O acesso é por token JWT (HS256). O fluxo é:

1. `POST /api/v1/usuarios/autenticar` com e-mail e senha
2. Guarde o campo `token` da resposta
3. Envie `Authorization: Bearer <token>` em todas as rotas protegidas

O token **não é renovado automaticamente**. Quando expirar, as rotas devolvem `401` e o
cliente deve autenticar de novo.

### Perfis de acesso

| Perfil | Como é definido | O que pode |
| --- | --- | --- |
| Público | — | `/health`, `/health/db`, `POST /api/v1/usuarios/autenticar` |
| Autenticado | Token válido de usuário ativo | Leads, funis, etapas, histórico, **leitura** de cargos e o próprio bolso de solicitações (`escopo=minhas`) |
| Administrador | Token de um usuário cujo cargo tem `"administrador": true` | Tudo o que o perfil autenticado pode, **mais** a gestão de usuários, a escrita de cargos, a designação de aprovadores e `escopo=todas` |
| Aprovador | Usuário **designado** na tabela `aprovador` pelo administrador | `escopo=aprovacao`, aprovar e rejeitar as solicitações pendentes |
| Financeiro | Token de um usuário cujo cargo tem `"financeiro": true` | `escopo=financeiro` e o registro de pagamento das solicitações aprovadas |

Os três papéis novos do módulo de solicitações são **cumulativos**: um administrador também
é tratado como aprovador e como financeiro.

Um token de usuário desativado ou de um usuário já excluído também é tratado como `401`.

### Conta inicial

A migration `000008` cria a conta administradora inicial:

| Campo | Valor |
| --- | --- |
| E-mail | `admin@exemplo.com` |
| Senha | `Mia@2026Admin` |

Troque essa senha no primeiro acesso (`POST /api/v1/usuarios/{id}/senha`).

---

## Formatos de resposta

**Objeto** (leitura, criação, atualização):

```json
{ "dados": { "id": "00000000-0000-0000-0000-0000000000ad", "nome": "Administrador" } }
```

**Lista paginada**:

```json
{
  "dados": [ { "nome": "Administrador" } ],
  "pagina": 1,
  "tamanho": 20
}
```

**Sem conteúdo** (`204`): corpo vazio.

**Erro**:

```json
{ "erro": { "codigo": 400, "mensagem": "erro de validação: email já cadastrado" } }
```

---

## Paginação

Aceitam paginação: usuários, cargos, leads, funis, histórico de movimentação e
solicitações de pagamento.

| Parâmetro | Padrão | Limite |
| --- | --- | --- |
| `pagina` | `1` | mínimo `1` |
| `tamanho` | `20` | máximo `100` |

Valores fora da faixa são normalizados (`?pagina=0&tamanho=9999` vira `pagina=1&tamanho=100`).
A resposta ecoa os valores efetivamente usados. Não há campo de total de registros.

---

## Códigos de status

| Status | Quando |
| --- | --- |
| `200` | Leitura, atualização ou listagem com sucesso |
| `201` | Recurso criado |
| `204` | Exclusão ou alteração sem conteúdo de resposta |
| `400` | Erro de validação: corpo inválido, campo ausente, regra de negócio violada, registro em uso |
| `401` | Token ausente, inválido, expirado, ou de usuário inexistente/inativo. Sempre com `WWW-Authenticate: Bearer` |
| `403` | Usuário autenticado sem o perfil necessário: sem `administrador`, ou tentando um `escopo` que não é seu (ex.: `aprovacao` de quem não foi designado) |
| `404` | Recurso não encontrado ou credenciais de login inválidas |
| `405` | Método não permitido na rota (resposta em texto puro do roteador, com cabeçalho `Allow`) |
| `409` | Conflito: transição de status inválida, registro já preenchido (ex.: segundo pagamento na mesma solicitação, aprovador já designado) ou exclusão de anexo vinculado |
| `500` | Erro interno — mensagem fixa `erro interno do servidor` |
| `503` | `/health/db` sem conexão com o banco |

Observações:

- Campos desconhecidos no corpo rejeitam a requisição (`400` com `json: unknown field`)
- IDs que não forem UUID devolvem `400` (`parâmetro id inválido`)
- Mensagens iniciadas com `erro de validação:` ou `registro não encontrado:` vêm das
  regras de negócio. `403` e `409` devolvem só o detalhe, **sem** o prefixo da
  sentinela (mesmo estilo do middleware: `perfil sem permissão para esta operação`)

---

## Rotas de saúde — público

| Método | Rota | Sucesso |
| --- | --- | --- |
| `GET` | `/health` | `200` `{"dados":{"status":"ok"}}` |
| `GET` | `/health/db` | `200` `{"dados":{"status":"ok"}}` ou `503` `{"erro":{"codigo":503,"mensagem":"banco de dados indisponível"}}` |

Qualquer rota desconhecida responde `404` com `{"erro":{"codigo":404,"mensagem":"rota não encontrada"}}`.

---

## Autenticação — público

### `POST /api/v1/usuarios/autenticar`

Requisição:

```json
{ "email": "admin@exemplo.com", "senha": "Mia@2026Admin" }
```

Resposta `200`:

```json
{
  "dados": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expira_em": "2026-10-07T15:29:42.0670651Z",
    "administrador": true,
    "usuario": {
      "id": "00000000-0000-0000-0000-0000000000ad",
      "nome": "Administrador",
      "email": "admin@exemplo.com",
      "ativo": true,
      "cargo_id": "00000000-0000-0000-0000-0000000000ad",
      "ultimo_login": "2026-10-07T14:29:42.0617007Z",
      "criado_em": "2026-10-07T13:58:00.301364Z",
      "atualizado_em": "2026-10-07T13:58:00.301364Z"
    }
  }
}
```

O login registra o `ultimo_login` e a resposta nunca contém a senha.

| Erro | Status | Mensagem |
| --- | --- | --- |
| E-mail ou senha incorretos | `404` | `email ou senha inválidos` |
| Usuário desativado | `400` | `erro de validação: usuário desativado` |
| Corpo ausente/inválido | `400` | `erro de validação: corpo da requisição inválido: ...` |

---

## Usuários — administrador

Todas as rotas abaixo exigem perfil administrador, exceto `POST /usuarios/autenticar`
(documentada acima).

| Método | Rota | Sucesso | Descrição |
| --- | --- | --- | --- |
| `POST` | `/api/v1/usuarios` | `201` | Cria usuário |
| `GET` | `/api/v1/usuarios` | `200` | Lista paginada |
| `GET` | `/api/v1/usuarios/{id}` | `200` | Obtém um usuário |
| `PUT` | `/api/v1/usuarios/{id}` | `200` | Atualiza nome, e-mail, cargo e status |
| `DELETE` | `/api/v1/usuarios/{id}` | `204` | Exclui usuário |
| `POST` | `/api/v1/usuarios/{id}/senha` | `204` | Troca a senha |
| `PATCH` | `/api/v1/usuarios/{id}/ativar` | `200` | Ativa e devolve o usuário |
| `PATCH` | `/api/v1/usuarios/{id}/desativar` | `200` | Desativa e devolve o usuário |

### `POST /api/v1/usuarios`

```json
{ "nome": "Ana Souza", "email": "ana@exemplo.com", "senha": "senhaForte123", "cargo_id": "uuid-do-cargo" }
```

| Erro | Status | Mensagem |
| --- | --- | --- |
| Nome/e-mail/senha ausentes | `400` | `erro de validação: nome do usuário é obrigatório`, `email do usuário é inválido` |
| Senha fora da faixa | `400` | `erro de validação: senha do usuário deve ter no mínimo 8 caracteres` (máx. 72) |
| Cargo inexistente | `400` | `erro de validação: cargo do usuário não encontrado` |
| E-mail já usado | `400` | `erro de validação: email já cadastrado` |

### `GET /api/v1/usuarios`

| Parâmetro | Descrição |
| --- | --- |
| `pagina`, `tamanho` | Paginação |
| `q` | Busca por nome ou e-mail, sem distinção de maiúsculas |
| `ativos=true` | Somente ativos — só entra se `q` estiver vazio |

### `PUT /api/v1/usuarios/{id}`

```json
{ "nome": "Ana Alterada", "email": "ana@exemplo.com", "cargo_id": "uuid-do-cargo", "ativo": true }
```

A senha não é alterada por aqui.

### `POST /api/v1/usuarios/{id}/senha`

```json
{ "nova_senha": "senhaNova123" }
```

A troca de senha invalida a senha anterior: depois do `204`, o login com a senha velha
responde `404`.

**Objeto de usuário:**

```json
{
  "id": "uuid",
  "nome": "Ana Souza",
  "email": "ana@exemplo.com",
  "ativo": true,
  "cargo_id": "uuid",
  "ultimo_login": "2026-10-07T14:29:42.0617007Z",
  "criado_em": "2026-10-07T14:26:19.854486Z",
  "atualizado_em": "2026-10-07T14:26:19.854486Z"
}
```

---

## Cargos

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/cargos` | autenticado | `200` lista paginada |
| `GET` | `/api/v1/cargos/busca?nome=Administrador` | autenticado | `200` objeto único |
| `GET` | `/api/v1/cargos/{id}` | autenticado | `200` objeto único |
| `POST` | `/api/v1/cargos` | **administrador** | `201` |
| `PUT` | `/api/v1/cargos/{id}` | **administrador** | `200` |
| `DELETE` | `/api/v1/cargos/{id}` | **administrador** | `204` |

`GET /api/v1/cargos` aceita `pagina` e `tamanho`. A busca por nome exige `?nome=` preenchido
(`400` quando vazio).

### `POST /api/v1/cargos`

```json
{ "nome": "Gerente de obra", "descricao": "Acompanha a obra", "administrador": false }
```

`PUT /api/v1/cargos/{id}` usa o mesmo corpo mais `"ativo": true`.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Nome vazio | `400` | `erro de validação: nome do cargo é obrigatório` |
| Nome repetido | `400` | `erro de validação: já existe um cargo com esse nome` |
| Cargo com usuários | `400` | `erro de validação: registro em uso por outros dados e não pode ser excluído` |

**Objeto de cargo:**

```json
{
  "id": "00000000-0000-0000-0000-0000000000ad",
  "nome": "Administrador",
  "descricao": "Perfil com acesso total ao painel",
  "ativo": true,
  "administrador": true
}
```

---

## Leads — autenticado

| Método | Rota | Sucesso | Descrição |
| --- | --- | --- | --- |
| `POST` | `/api/v1/leads` | `201` | Cria lead |
| `GET` | `/api/v1/leads` | `200` | Lista paginada (`q` por nome/e-mail) |
| `GET` | `/api/v1/leads/{id}` | `200` | Obtém |
| `PUT` | `/api/v1/leads/{id}` | `200` | Atualiza |
| `DELETE` | `/api/v1/leads/{id}` | `204` | Exclui |
| `PATCH` | `/api/v1/leads/{id}/etapa` | `200` | Move de etapa |
| `GET` | `/api/v1/funils/{funil_id}/leads` | `200` | Lista do funil (sem paginação) |
| `GET` | `/api/v1/etapas/{etapa_id}/leads` | `200` | Lista da etapa (sem paginação) |
| `GET` | `/api/v1/funils/{funil_id}/leads/contagem` | `200` | `{"dados":{"total":12}}` |
| `GET` | `/api/v1/etapas/{etapa_id}/leads/contagem` | `200` | `{"dados":{"total":3}}` |

Corpos:

```json
{ "nome": "Ana Souza", "email": "ana@exemplo.com", "telefone": "", "origem": "site", "etapa_id": "uuid" }
```

`PUT` aceita o mesmo corpo mais `"ativo": true`. `PATCH /etapa`:

```json
{ "etapa_id": "uuid-da-etapa" }
```

**Objeto de lead:**

```json
{
  "id": "uuid",
  "nome": "Ana Souza",
  "email": "ana@exemplo.com",
  "telefone": "",
  "ativo": true,
  "origem": "site",
  "criado_em": "2026-09-30T01:41:55.248088Z",
  "atualizado_em": "2026-10-05T18:07:25.648008Z",
  "etapa_id": "uuid"
}
```

---

## Funis — autenticado

| Método | Rota | Sucesso |
| --- | --- | --- |
| `POST` | `/api/v1/funils` | `201` |
| `GET` | `/api/v1/funils` | `200` lista paginada (`ativos=true` filtra) |
| `GET` | `/api/v1/funils/{funil_id}` | `200` |
| `PUT` | `/api/v1/funils/{funil_id}` | `200` |
| `DELETE` | `/api/v1/funils/{funil_id}` | `204` |

```json
{ "nome": "Vendas" }
```

`PUT` usa `{"nome": "Vendas", "ativo": true}`.

```json
{ "funil_id": "uuid", "nome": "Vendas", "ativo": true }
```

---

## Etapas — autenticado

| Método | Rota | Sucesso |
| --- | --- | --- |
| `POST` | `/api/v1/etapas` | `201` |
| `GET` | `/api/v1/funils/{funil_id}/etapas` | `200` lista (sem paginação) |
| `PUT` | `/api/v1/funils/{funil_id}/etapas/ordem` | `200` lista reordenada |
| `GET` | `/api/v1/etapas/{etapa_id}` | `200` |
| `PUT` | `/api/v1/etapas/{etapa_id}` | `200` |
| `DELETE` | `/api/v1/etapas/{etapa_id}` | `204` |
| `GET` | `/api/v1/etapas/{etapa_id}/proxima` | `200` |
| `GET` | `/api/v1/etapas/{etapa_id}/anterior` | `200` |

```json
{ "nome": "Novo contato", "ordem": 1, "funil_id": "uuid" }
```

`PUT` da etapa aceita o mesmo corpo mais `"ativo": true`. Reordenação:

```json
{ "etapas": ["uuid-1", "uuid-2", "uuid-3"] }
```

```json
{ "etapa_id": "uuid", "nome": "Novo contato", "ordem": 1, "funil_id": "uuid", "ativo": true }
```

---

## Histórico de movimentação — autenticado

| Método | Rota | Sucesso |
| --- | --- | --- |
| `GET` | `/api/v1/leads/{lead_id}/historico` | `200` lista paginada |
| `POST` | `/api/v1/leads/{lead_id}/historico` | `204` |

```json
{ "etapa_anterior_id": "uuid", "etapa_atual_id": "uuid" }
```

```json
{
  "id": "uuid",
  "lead_id": "uuid",
  "etapa_anterior_id": "uuid",
  "etapa_atual_id": "uuid",
  "movido_em": "2026-10-07T14:26:19.854486Z"
}
```

---

## Solicitações de pagamento — autenticado

Fluxo: **solicitante → aprovador designado → financeiro**.

1. O solicitante sobe os anexos (`POST /api/v1/arquivos`) e cria a solicitação.
2. O administrador designa quem pode aprovar (`POST /api/v1/aprovadores`).
3. O aprovador aprova ou rejeita.
4. O financeiro registra o pagamento — o **valor pago pode ser diferente** do estimado.

### Estados

| De | Para | Rota |
| --- | --- | --- |
| `pendente_aprovacao` | `aprovado` | `POST /{id}/aprovar` |
| `pendente_aprovacao` | `rejeitado` | `POST /{id}/rejeitar` |
| `pendente_aprovacao`, `aprovado` | `cancelado` | `POST /{id}/cancelar` |
| `aprovado` | `pago` | `POST /{id}/pagamento` |

`pago`, `rejeitado` e `cancelado` são estados terminais. Toda transição grava um registro
no histórico **na mesma transação**, e o `UPDATE` só acontece se o status anterior bater:
se outra pessoa mudar primeiro, a segunda recebe `409`
(`solicitação alterada por outra operação, recarregue e tente novamente`).

### Rotas

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/solicitacoes` | autenticado | `201` |
| `GET` | `/api/v1/solicitacoes` | autenticado | `200` lista paginada |
| `GET` | `/api/v1/solicitacoes/{id}` | quem pode ver | `200` objeto único |
| `POST` | `/api/v1/solicitacoes/{id}/aprovar` | aprovador ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/rejeitar` | aprovador ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/cancelar` | solicitante ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/pagamento` | financeiro ou administrador | `201` |
| `GET` | `/api/v1/solicitacoes/{id}/pagamento` | quem pode ver | `200` objeto único |
| `GET` | `/api/v1/solicitacoes/{id}/historico` | quem pode ver | `200` lista |
| `GET` | `/api/v1/solicitacoes/{id}/arquivos` | quem pode ver | `200` lista |

**Quem pode ver** uma solicitação (roteiros `Obter`, `pagamento`, `historico`, `arquivos`):
o próprio solicitante, um administrador, o aprovador designado e o financeiro (que só enxerga
solicitações fora de `pendente_aprovacao`). Quem não pode ver recebe `403`
(`perfil sem permissão para consultar esta solicitação`).

### Escopos de listagem

`GET /api/v1/solicitacoes` aceita `pagina`, `tamanho`, `escopo` e `status`:

| `escopo` | Quem pode | O que lista |
| --- | --- | --- |
| vazio ou `minhas` | autenticado | as que ele mesmo criou |
| `aprovacao` | aprovador designado ou administrador | as `pendente_aprovacao` |
| `financeiro` | cargo com `"financeiro": true` ou administrador | as `aprovado` |
| `todas` | administrador | todas |

`status=` (ex.: `?escopo=todas&status=pago`) filtra dentro do que o escopo já deixa
enxergar — não serve para furar a definição do escopo. `escopo=aprovacao` só aceita
`status=pendente_aprovacao` e `escopo=financeiro` aceita qualquer status **exceto**
`pendente_aprovacao`; a combinação fora da regra devolve `400`
(`status não permitido para o escopo de listagem informado`). `escopo` fora da lista
devolve `400` (`escopo de listagem inválido`); escopo de outro perfil devolve `403`.

### `POST /api/v1/solicitacoes`

```json
{
  "valor_centavos": 150000,
  "prazo_pagamento": "2026-10-20",
  "observacao": "Compra de cimento para a fase 2",
  "forma_pagamento": "pix",
  "arquivo_ids": ["3f1c2a4e-0000-4000-8000-000000000001"]
}
```

| Campo | Regra |
| --- | --- |
| `valor_centavos` | inteiro em centavos, maior que zero (`150000` = R$ 1.500,00) |
| `prazo_pagamento` | `AAAA-MM-DD` (ou RFC3339). Obrigatório e não pode ser anterior a hoje |
| `observacao` | obrigatória, no máximo 1000 caracteres |
| `forma_pagamento` | `pix`, `cartao` ou `boleto` |
| `arquivo_ids` | opcional. Cada anexo tem que existir, pertencer ao solicitante e não estar em outra solicitação |

**Objeto de solicitação:**

```json
{
  "id": "00000000-0000-4000-8000-000000000001",
  "solicitante_id": "00000000-0000-4000-8000-000000000002",
  "aprovador_id": null,
  "valor_centavos": 150000,
  "prazo_pagamento": "2026-10-20T00:00:00Z",
  "observacao": "Compra de cimento para a fase 2",
  "forma_pagamento": "pix",
  "status": "pendente_aprovacao",
  "motivo_rejeicao": null,
  "aprovado_em": null,
  "rejeitado_em": null,
  "cancelado_em": null,
  "criado_em": "2026-10-07T14:26:19.854486Z",
  "atualizado_em": "2026-10-07T14:26:19.854486Z",
  "arquivo_ids": ["00000000-0000-4000-8000-000000000003"]
}
```

| Erro | Status | Mensagem |
| --- | --- | --- |
| Valor zero ou negativo | `400` | `erro de validação: valor deve ser maior que zero` |
| Prazo vazio ou anterior a hoje | `400` | `erro de validação: prazo de pagamento é obrigatório` / `...não pode ser anterior a hoje` |
| Forma diferente das três | `400` | `erro de validação: forma de pagamento deve ser pix, cartao ou boleto` |
| Anexo de outra pessoa | `403` | `arquivo não pertence ao solicitante` |
| Anexo já em uso | `400` | `erro de validação: arquivo já vinculado a outra solicitação` |

### `POST /api/v1/solicitacoes/{id}/rejeitar`

```json
{ "motivo": "orçamento acima do previsto para esta fase" }
```

`motivo` é obrigatório e aceita no máximo 500 caracteres (`400` quando vazio).

### `POST /api/v1/solicitacoes/{id}/pagamento`

```json
{
  "valor_centavos": 148500,
  "comprovante_arquivo_id": "00000000-0000-4000-8000-000000000004",
  "pago_em": "2026-10-15T14:00:00Z"
}
```

| Campo | Regra |
| --- | --- |
| `valor_centavos` | valor **efetivo** pago, maior que zero — pode diferir do estimado |
| `comprovante_arquivo_id` | opcional. Quando presente tem que ser um PDF **de quem está registrando** e ainda não vinculado a outra solicitação |
| `pago_em` | opcional, RFC3339. Padrão: agora (UTC) |

**Objeto de pagamento:**

```json
{
  "id": "00000000-0000-4000-8000-000000000005",
  "solicitacao_id": "00000000-0000-4000-8000-000000000001",
  "comprovante_arquivo_id": "00000000-0000-4000-8000-000000000004",
  "valor_centavos": 148500,
  "pago_em": "2026-10-15T14:00:00Z",
  "criado_em": "2026-10-15T14:02:10Z"
}
```

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para registrar pagamento` |
| Status diferente de `aprovado` | `409` | `não é possível mudar a solicitação de "<status atual>" para "pago"` |
| Comprovante não PDF | `400` | `comprovante deve ser um arquivo PDF` |
| Comprovante de outro usuário | `403` | `comprovante não pertence ao usuário` |
| Comprovante já usado em outra solicitação | `400` | `comprovante já vinculado a outra solicitação` |
| Sem pagamento registrado | `404` | `registro não encontrado: pagamento não encontrado` (em `GET /{id}/pagamento`) |

A resposta de `aprovar`, `rejeitar` e `cancelar` devolve o objeto da solicitação já com
o status novo; a de `POST /{id}/pagamento` devolve o objeto do pagamento.

---

## Anexos (arquivos) — autenticado

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/arquivos` | autenticado | `201` (multipart) |
| `GET` | `/api/v1/arquivos` | autenticado | `200` lista paginada dos **próprios** arquivos |
| `GET` | `/api/v1/arquivos/{id}` | dono ou quem pode ver a solicitação | `200` binário |
| `DELETE` | `/api/v1/arquivos/{id}` | dono ou administrador | `204` |

O upload é `multipart/form-data` com **um campo só**: `arquivo`.

| Regra | Valor |
| --- | --- |
| Tamanho máximo | 10MB por arquivo (`413 arquivo excede o limite de 10MB`) |
| Tipos aceitos | `application/pdf`, `image/png`, `image/jpeg`, `image/webp` |
| Nome | obrigatório, no máximo 255 caracteres |
| Chave no storage | `{proprietario_id}/{uuid}/{nome}` |

`GET /api/v1/arquivos/{id}` devolve o binário com o `content-type` original e
`Content-Disposition: attachment; filename="nome.pdf"`. Quem não é dono só baixa se puder
ver a solicitação dona do anexo (`403 perfil sem permissão para baixar este arquivo`).

Anexo já vinculado a solicitação não pode ser excluído (`409 arquivo vinculado a uma
solicitação não pode ser removido`).

O comprovante registrado em `POST /{id}/pagamento` **também vira vínculo** da solicitação:
ele aparece em `GET /{id}/arquivos`, é baixável por quem enxerga a solicitação e não pode
ser excluído (mesmo `409`). Um arquivo só pode estar vinculado a uma solicitação por vez
(`UNIQUE (arquivo_id)` em `solicitacao_arquivo`).

---

## Aprovadores — administrador

| Método | Rota | Sucesso |
| --- | --- | --- |
| `POST` | `/api/v1/aprovadores` | `201` |
| `GET` | `/api/v1/aprovadores` | `200` lista paginada |
| `DELETE` | `/api/v1/aprovadores/{id}` | `204` |

```json
{ "usuario_id": "00000000-0000-4000-8000-000000000002" }
```

```json
{
  "id": "00000000-0000-4000-8000-000000000006",
  "usuario_id": "00000000-0000-4000-8000-000000000002",
  "criado_em": "2026-10-07T14:26:19.854486Z"
}
```

A designação é única por usuário: repetir devolve `409 usuário já está designado como
aprovador`, e usuário inativo devolve `400 usuário inativo não pode ser designado como
aprovador`. Remover a designação não apaga as aprovações já registradas no histórico.

---

## Exemplos

### PowerShell

```powershell
$base = "http://localhost:8080"

$login = Invoke-RestMethod -Uri "$base/api/v1/usuarios/autenticar" -Method Post `
    -ContentType "application/json" `
    -Body '{"email":"admin@exemplo.com","senha":"Mia@2026Admin"}'

$token = $login.dados.token
$auth = @{ Authorization = "Bearer $token" }

$usuarios = Invoke-RestMethod -Uri "$base/api/v1/usuarios?pagina=1&tamanho=20" -Headers $auth
$usuarios.dados

Invoke-RestMethod -Uri "$base/api/v1/leads" -Headers $auth
```

### curl

```bash
base=http://localhost:8080

token=$(curl -s -X POST "$base/api/v1/usuarios/autenticar" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@exemplo.com","senha":"Mia@2026Admin"}' | jq -r .dados.token)

curl -s "$base/api/v1/usuarios?pagina=1&tamanho=20" -H "Authorization: Bearer $token"
```

### Erro de permissão

```bash
curl -i "$base/api/v1/usuarios" -H "Authorization: Bearer $token_de_usuario_comum"
```

```http
HTTP/1.1 403 Forbidden

{"erro":{"codigo":403,"mensagem":"perfil sem permissão para esta operação"}}
```
