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
| Autenticado | Token válido de usuário ativo | **Leitura** de cargos, a própria sessão (`/api/v1/sessao`) e o próprio bolso de solicitações (`escopo=minhas`) |
| Comercial | Token de um usuário cujo cargo tem `"comercial": true` | Leads (criar, editar, mover, excluir), histórico e **leitura** de funis e etapas |
| Administrador | Token de um usuário cujo cargo tem `"administrador": true` | Tudo o que os perfis autenticado e comercial podem, **mais** a estrutura de funis e etapas, a gestão de usuários, a escrita de cargos, a designação de aprovadores e `escopo=todas` |
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

Troque essa senha no primeiro acesso (`POST /api/v1/usuarios/{id}/senha` ou
`POST /api/v1/sessao/senha`).

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

### Rastreamento de requisições

Toda resposta traz o cabeçalho `X-Request-Id`. A API grava uma linha JSON por requisição
na saída padrão (`requisicao_id`, `metodo`, `caminho`, `status`, `duracao_ms`); nas
respostas `5xx` a linha sai com nível `ERROR` e o campo `erro` com a causa real, que nunca
vai para o corpo da resposta. Para investigar um `500`, peça ao cliente o `X-Request-Id` e
procure a mesma chave nos logs.

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
    "comercial": true,
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

## Sessão — autenticado

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/sessao` | autenticado | `200` objeto único |
| `POST` | `/api/v1/sessao/senha` | autenticado | `204` vazio |
| `POST` | `/api/v1/sessao/encerrar` | autenticado | `204` vazio |

### `GET /api/v1/sessao`

Devolve o perfil do próprio usuário dono do token (mesmo objeto de usuário documentado
em `PUT /api/v1/usuarios/{id}`). Qualquer usuário ativo pode consultar a própria sessão
sem depender do perfil administrador.

### `POST /api/v1/sessao/senha`

```json
{ "senha_atual": "senhaForte123", "nova_senha": "senhaNova123" }
```

Troca a senha do próprio usuário dono do token. Exige a senha atual e aplica à nova as
mesmas regras de `POST /api/v1/usuarios/{id}/senha` (8 a 72 caracteres). Depois do `204`,
o login com a senha anterior responde `404` e **todos os tokens já emitidos para o
usuário deixam de valer**, inclusive o que fez a troca: o cliente precisa autenticar de novo.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Senha atual ausente | `400` | `erro de validação: senha atual é obrigatória` |
| Senha atual incorreta | `400` | `erro de validação: senha atual incorreta` |
| Nova senha fora da faixa | `400` | `erro de validação: senha do usuário deve ter no mínimo 8 caracteres` |
| Sem token | `401` | `token ausente ou inválido` |

### `POST /api/v1/sessao/encerrar`

Sem corpo. Encerra **todas** as sessões do usuário dono do token: qualquer token emitido
antes desta chamada passa a responder `401`. É o logout do lado do servidor.

### Revogação de tokens

Cada usuário tem uma versão de sessão gravada no banco e copiada para o token no login
(claim `ver`). O middleware compara as duas a cada requisição. A versão avança quando:

- o próprio usuário troca a senha (`POST /api/v1/sessao/senha`);
- o administrador redefine a senha (`POST /api/v1/usuarios/{id}/senha`);
- o usuário encerra as sessões (`POST /api/v1/sessao/encerrar`).

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
{ "nome": "Gerente de obra", "descricao": "Acompanha a obra", "administrador": false, "financeiro": false, "comercial": false }
```

Os três perfis são independentes e opcionais (ausente vale `false`). Um cargo criado antes
da migração `000023` nasce com `"comercial": false`: quem já usava leads precisa ter o cargo
marcado como comercial pelo administrador.

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
  "administrador": true,
  "financeiro": false,
  "comercial": false
}
```

---

## Leads — comercial

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

Limites: `nome` obrigatório até 200 caracteres, `email` vazio ou com `@` até 255,
`telefone` até 40 e `origem` até 80. A etapa precisa existir e estar ativa.

`PUT` edita só o cadastro e exige `"ativo"` explícito (`true` ou `false`). O `etapa_id` é
opcional: se vier, precisa ser a etapa atual — a troca de etapa só acontece pelo `PATCH`,
que é quem grava o histórico.

`PATCH /etapa`:

```json
{ "etapa_id": "uuid-da-etapa" }
```

| Erro | Status | Mensagem |
| --- | --- | --- |
| `PUT` sem `ativo` | `400` | `erro de validação: campo ativo é obrigatório` |
| `PUT` com outra etapa | `400` | `erro de validação: a etapa do lead só muda pela movimentação de etapa` |
| Etapa inexistente | `400` | `erro de validação: etapa de destino não encontrada` |
| Etapa inativa | `400` | `erro de validação: etapa de destino está inativa` |
| Etapa de outro funil | `400` | `erro de validação: a etapa de destino pertence a outro funil` |
| Lead movido por outra pessoa no meio da operação | `409` | `o lead mudou de etapa durante a operação; recarregue e tente de novo` |

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

## Funis — comercial lê, administrador escreve

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/funils` | **administrador** | `201` |
| `GET` | `/api/v1/funils` | comercial | `200` lista paginada (`ativos=true` filtra) |
| `GET` | `/api/v1/funils/{funil_id}` | comercial | `200` |
| `PUT` | `/api/v1/funils/{funil_id}` | **administrador** | `200` |
| `DELETE` | `/api/v1/funils/{funil_id}` | **administrador** | `204` |

```json
{ "nome": "Vendas" }
```

`PUT` usa `{"nome": "Vendas", "ativo": true}`.

```json
{ "funil_id": "uuid", "nome": "Vendas", "ativo": true }
```

---

## Etapas — comercial lê, administrador escreve

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/etapas` | **administrador** | `201` |
| `GET` | `/api/v1/funils/{funil_id}/etapas` | comercial | `200` lista (sem paginação) |
| `PUT` | `/api/v1/funils/{funil_id}/etapas/ordem` | **administrador** | `200` lista reordenada |
| `GET` | `/api/v1/etapas/{etapa_id}` | comercial | `200` |
| `PUT` | `/api/v1/etapas/{etapa_id}` | **administrador** | `200` |
| `DELETE` | `/api/v1/etapas/{etapa_id}` | **administrador** | `204` |
| `GET` | `/api/v1/etapas/{etapa_id}/proxima` | comercial | `200` |
| `GET` | `/api/v1/etapas/{etapa_id}/anterior` | comercial | `200` |

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

## Histórico de movimentação — comercial

| Método | Rota | Sucesso |
| --- | --- | --- |
| `GET` | `/api/v1/leads/{lead_id}/historico` | `200` lista paginada |

O histórico é só leitura: cada registro nasce do `PATCH /api/v1/leads/{id}/etapa`, na mesma
transação que move o lead. Não existe gravação manual (o antigo `POST` responde `405`).

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
| `POST` | `/api/v1/solicitacoes/{id}/aprovar` | aprovador ou administrador, exceto o próprio solicitante | `200` |
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

### `POST /api/v1/solicitacoes/{id}/aprovar`

Sem corpo. Ninguém aprova a própria solicitação, nem o administrador: a segregação entre quem
pede e quem aprova vale para todos os perfis.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de aprovador | `403` | `perfil sem permissão para aprovar ou rejeitar solicitações` |
| Solicitante aprovando a própria | `403` | `o solicitante não pode aprovar a própria solicitação` |
| Fora de `pendente_aprovacao` | `409` | `não é possível mudar a solicitação de "..." para "aprovado"` |

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

## Estoque — categorias

Escrita exige perfil **administrador**; leitura é liberada para qualquer perfil
autenticado.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/categorias` | **administrador** | `201` |
| `GET` | `/api/v1/categorias` | autenticado | `200` lista paginada |
| `GET` | `/api/v1/categorias/{id}` | autenticado | `200` objeto único |
| `PUT` | `/api/v1/categorias/{id}` | **administrador** | `200` |
| `PATCH` | `/api/v1/categorias/{id}/ativar` | **administrador** | `200` |
| `PATCH` | `/api/v1/categorias/{id}/desativar` | **administrador** | `200` |

`GET /api/v1/categorias` aceita `ativo=true|false`, `pagina` e `tamanho`. A árvore
tem dois níveis: `categoria_pai_id` preenchido marca a subcategoria, e a categoria
raiz só agrupa — produto nunca aponta para ela.

```json
{ "nome": "Cimento", "categoria_pai_id": "00000000-0000-4000-8000-000000000010", "ordem": 1, "icone": "package" }
```

`PUT` usa o mesmo corpo sem `categoria_pai_id`. O slug é gerado do nome e **não muda**
na edição (preserva as URLs públicas).

| Erro | Status | Mensagem |
| --- | --- | --- |
| Nome repetido | `409` | `já existe uma categoria com este nome` |
| Desativar com subcategorias | `409` | `categoria possui subcategorias e não pode ser inativada` |

---

## Estoque — produtos

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/produtos` | **administrador** | `201` |
| `GET` | `/api/v1/produtos` | autenticado | `200` lista paginada |
| `GET` | `/api/v1/produtos/{id}` | autenticado | `200` objeto único |
| `GET` | `/api/v1/produtos/{id}/saldo` | autenticado | `200` `{"produto_id":"…","saldo":4}` |
| `PUT` | `/api/v1/produtos/{id}` | **administrador** | `200` |
| `PATCH` | `/api/v1/produtos/{id}/ativar` | **administrador** | `200` |
| `PATCH` | `/api/v1/produtos/{id}/desativar` | **administrador** | `200` |
| `PATCH` | `/api/v1/produtos/{id}/destaque` | **administrador** | `200` `{"destaque":true}` |

`GET /api/v1/produtos` aceita `categoria_id`, `busca` (nome ou código), `ativo`,
`destaque`, `estoque_baixo=true`, `pagina` e `tamanho`. `categoria_id` aceita também a
categoria **raiz**: o filtro inclui os produtos das subcategorias dela.

```json
{
  "categoria_id": "00000000-0000-4000-8000-000000000011",
  "nome": "Cimento CP II 50",
  "descricao": "Saco de 50 kg",
  "codigo": "SKU-001",
  "unidade_medida": "un",
  "preco_centavos": 2590,
  "preco_promocional_centavos": 2390,
  "estoque_minimo": 5,
  "peso_kg": 50,
  "destaque": false
}
```

| Campo | Regra |
| --- | --- |
| `categoria_id` | obrigatório e precisa ser **subcategoria** ativa (`400`) |
| `unidade_medida` | `un`, `m`, `m2`, `m3`, `kg`, `t`, `cx`, `sc`, `pct`, `lt` |
| `preco_centavos` | opcional — `null` exibe **"sob consulta"** no site |
| `preco_promocional_centavos` | exige preço cheio e deve ser menor (`400`) |
| `saldo` | nunca vem no corpo: `quantidade_atual` só muda via movimentação |

O objeto devolvido traz `saldo`, `estoque_baixo` (saldo ≤ `estoque_minimo`), `slug` e
`ativo`. Nome repetido devolve `409 já existe um produto com este nome`.

---

## Estoque — movimentações

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/produtos/{id}/movimentos` | **administrador** | `201` |
| `GET` | `/api/v1/produtos/{id}/movimentos` | autenticado | `200` lista paginada (`?tipo=`) |

```json
{ "tipo": "entrada", "quantidade": 10, "documento_ref": "NF-100", "observacao": "reposição do mês" }
```

| Tipo | Efeito no saldo |
| --- | --- |
| `entrada` / `devolucao` | `+` |
| `saida` / `perda` | `−` |
| `ajuste` | `+`/`−` (`quantidade` é o delta, aceita negativo) |

`quantidade` não pode ser `0`. Movimento é append-only: não edita, não apaga, e o
`saldo_apos` fica congelado no registro. Estoque que ficaria negativo devolve
`400 erro de validação: saldo insuficiente para a movimentação`.

Para desfazer uma movimentação registra-se a inversa (`entrada` ↔ `saida`,
`devolucao` ↔ `perda`); o histórico permanece completo.

---

## Estoque — resumo

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/estoque/resumo` | autenticado | `200` |

```json
{
  "dados": {
    "total_produtos": 120,
    "produtos_ativos": 110,
    "valor_estoque_centavos": 1543200,
    "produtos_estoque_baixo": 7,
    "ultimos_movimentos": [
      {
        "id": "00000000-0000-4000-8000-000000000021",
        "produto_id": "00000000-0000-4000-8000-000000000011",
        "produto_nome": "Cimento CP II 50",
        "tipo": "saida",
        "quantidade": 6,
        "saldo_apos": 4,
        "criado_em": "2026-10-09T12:00:00Z"
      }
    ]
  }
}
```

| Campo | Regra |
| --- | --- |
| `total_produtos` | todos os produtos, ativos ou não |
| `produtos_ativos` | só os com `ativo = true` |
| `valor_estoque_centavos` | `SUM(preco_cheio × saldo)` dos **ativos** com preço; promoção não conta e "sob consulta" fica de fora |
| `produtos_estoque_baixo` | ativos com `estoque_minimo > 0` e saldo ≤ mínimo |
| `ultimos_movimentos` | os 5 mais recentes, com o nome do produto (não expõe `usuario_id` nem observações) |

---

## Estoque — imagens de produto

Escrita restrita ao **administrador**. O arquivo chega primeiro pelo upload comum
(`POST /api/v1/arquivos`) e depois é anexado ao produto.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/produtos/{id}/imagens` | **administrador** | `201` |
| `DELETE` | `/api/v1/produtos/{id}/imagens/{imagemId}` | **administrador** | `204` |

```json
{ "arquivo_id": "00000000-0000-4000-8000-000000000031", "ordem": 0, "alt": "Saco de cimento" }
```

| Campo | Regra |
| --- | --- |
| `arquivo_id` | obrigatório, precisa existir e ser **imagem** (`400 arquivo deve ser uma imagem`) |
| `ordem` | inteiro ≥ `0`, define a ordem de exibição no site (`400`) |
| `alt` | texto alternativo, no máximo 160 caracteres (`400`) |

| Erro | Status | Mensagem |
| --- | --- | --- |
| Produto inexistente | `404` | `produto não encontrado` |
| Arquivo inexistente | `404` | `arquivo não encontrado` |
| Mesmo arquivo de novo no produto | `409` | `arquivo já anexado a este produto` |
| Imagem de outro produto na remoção | `404` | `imagem não encontrada` |

As imagens aparecem preenchidas no **detalhe** do produto do painel e nas respostas
públicas, ordenadas por `ordem`; a **lista** do painel devolve `imagens: []` (a tela de
edição busca o detalhe). O produto não tem exclusão — só desativação — e o arquivo
original continua no storage, podendo ser reaproveitado por outro produto.

---

## Vitrine pública (site) — público

Rotas abertas, **sem token**, pensadas para o site: só mostram o que o visitante pode
comprar. Visibilidade é uniforme — `ativo = true` **e** `quantidade_atual > 0`; produto
oculto em detalhe devolve `404`.

| Método | Rota | Sucesso |
| --- | --- | --- |
| `GET` | `/api/v1/publico/categorias` | `200` árvore de categorias ativas |
| `GET` | `/api/v1/publico/produtos` | `200` lista paginada |
| `GET` | `/api/v1/publico/produtos/{slug}` | `200` objeto único |
| `GET` | `/api/v1/publico/destaques` | `200` lista paginada |
| `GET` | `/api/v1/publico/imagens/{id}` | `200` binário da imagem |

```json
{
  "id": "00000000-0000-4000-8000-000000000011",
  "categoria_id": "00000000-0000-4000-8000-000000000011",
  "nome": "Cimento CP II 50",
  "slug": "cimento-cp-ii-50",
  "descricao": "Saco de 50 kg",
  "unidade_medida": "un",
  "preco_centavos": 2590,
  "preco_promocional_centavos": null,
  "destaque": false,
  "imagens": [{ "id": "00000000-0000-4000-8000-000000000031", "ordem": 0, "alt": "Saco de cimento" }]
}
```

| Rota | Observações |
| --- | --- |
| `/publico/categorias` | só ativas, montadas em árvore (`filhas[]`), casamento por `categoria_id` |
| `/publico/produtos` | `?categoria=<slug>` aceita raiz (inclui as subcategorias), `?pagina` e `?tamanho` |
| `/publico/produtos/{slug}` | slug desconhecido ou produto oculto → `404` |
| `/publico/destaques` | ativos com destaque **e** estoque |
| `/publico/imagens/{id}` | `Content-Type` original da imagem; id de imagem de produto inativo → `404` |

O objeto público **não expõe** `saldo`, `codigo`, `ativo`, `estoque_minimo`, `criado_em`
nem `atualizado_em`. Sem estoque o produto some da vitrine inteira (lista, destaques e
detalhe) — não há selo "esgotado". Categoria desconhecida em `?categoria=` devolve `404`.

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
