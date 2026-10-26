# API

Documentação das rotas HTTP do painel administrativo.

- Base URL local: `http://localhost:8080` (docker compose) ou a porta definida em `PORT`
- Todas as rotas de negócio ficam sob `/api/v1`
- Todas as respostas são JSON (`application/json; charset=utf-8`), exceto `204` e os binários de arquivo

---

## Variáveis de ambiente

| Nome | Obrigatória | Padrão | Descrição |
| --- | --- | --- | --- |
| `PORT` | não | `8080` | Porta HTTP do servidor |
| `DATABASE_URL` | sim | — | String de conexão PostgreSQL, ex.: `postgres://mia:mia@localhost:5444/mia?sslmode=disable` |
| `CORS_ORIGINS` | não | nenhuma | Origens separadas por vírgula. Ex.: `http://localhost:5173`. Vazio não libera nenhuma origem; `*` libera todas. As respostas expõem `X-Request-Id` e `Retry-After` ao navegador |
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
| Financeiro | Token de um usuário cujo cargo tem `"financeiro": true` (ou de administrador) | `escopo=financeiro`, classificar, adiar, pagar e estornar as solicitações aprovadas, as despesas recorrentes e o resumo de contas a pagar, a escrita de fornecedores e do plano de contas, e as contas bancárias das empresas do seu acesso |

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
| `405` | Método não permitido na rota: `{"erro":{"codigo":405,"mensagem":"método não permitido nesta rota"}}`, com cabeçalho `Allow` |
| `409` | Conflito: transição de status inválida, registro já preenchido (ex.: segundo pagamento na mesma solicitação, aprovador já designado) ou exclusão de anexo vinculado |
| `500` | Erro interno — mensagem fixa `erro interno do servidor` |
| `503` | `/health/db` sem conexão com o banco |

Observações:

- Campos desconhecidos no corpo rejeitam a requisição (`400` com `json: unknown field`)
- IDs que não forem UUID devolvem `400` (`parâmetro id inválido`)
- Quando uma regra do banco barra a gravação antes das validações da aplicação, a resposta
  continua sendo `400`/`409`, nunca `500`:

  | Situação no banco | Status | Mensagem |
  | --- | --- | --- |
  | Referência a registro inexistente (gravação) | `400` | `erro de validação: registro relacionado não encontrado` |
  | Registro em uso (exclusão) | `400` | `erro de validação: registro em uso por outros dados e não pode ser excluído` |
  | Chave única repetida | `409` | `já existe um registro com estes dados` |
  | Regra `CHECK` | `400` | `erro de validação: dados fora das regras do cadastro` |
  | Texto acima do tamanho da coluna | `400` | `erro de validação: texto maior que o tamanho permitido` |
  | Número acima do tipo da coluna | `400` | `erro de validação: número fora do intervalo permitido` |
  | Campo obrigatório nulo | `400` | `erro de validação: campo obrigatório não informado` |
  | Duas gravações travando os mesmos registros em ordem inversa (deadlock) | `409` | `operação concorrente no mesmo registro; recarregue e tente de novo` |
- Mensagens iniciadas com `erro de validação:` ou `registro não encontrado:` vêm das
  regras de negócio. `403` e `409` devolvem só o detalhe, **sem** o prefixo da
  sentinela (mesmo estilo do middleware: `perfil sem permissão para esta operação`)

### Rastreamento de requisições

Toda resposta traz o cabeçalho `X-Request-Id`. A API grava uma linha JSON por requisição
na saída padrão (`requisicao_id`, `metodo`, `caminho`, `status`, `duracao_ms`); nas
respostas `5xx` a linha sai com nível `ERROR` e o campo `erro` com a causa real, que nunca
vai para o corpo da resposta. Falhas que acontecem depois de a resposta começar (por exemplo,
um download interrompido pelo cliente) saem com nível `WARN` e o mesmo campo `erro`. Para
investigar um `500`, peça ao cliente o `X-Request-Id` e procure a mesma chave nos logs.

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
    "financeiro": true,
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

`administrador`, `comercial` e `financeiro` dizem quais perfis o usuário tem; os dois últimos
são cumulativos com o de administrador. O front usa esses campos para mostrar ou esconder as
telas, sem precisar sondar rotas protegidas.

Proteção contra força bruta: 5 falhas do mesmo e-mail a partir do mesmo IP, ou 50 falhas
de um mesmo IP, em 15 minutos, bloqueiam novas tentativas por 15 minutos (`429`). Um login
bem-sucedido zera o contador daquele e-mail. O tempo de resposta para e-mail inexistente é
o mesmo de senha errada, para não revelar quem tem cadastro.

| Erro | Status | Mensagem |
| --- | --- | --- |
| E-mail ou senha incorretos | `404` | `email ou senha inválidos` |
| Muitas falhas seguidas | `429` | `muitas tentativas de login; tente novamente em N minuto(s)` (com `Retry-After` em segundos) |
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

### Último administrador

O sistema nunca fica sem administrador ativo. Responde `409` com
`o sistema precisa de pelo menos um administrador ativo` qualquer operação que tire o acesso
do último deles:

- desativar (`PATCH .../desativar` ou `PUT` com `"ativo": false`) ou excluir o usuário;
- trocar o cargo dele por um cargo sem `administrador`;
- tirar o `administrador` do cargo (`PUT /api/v1/cargos/{id}`) quando todos os
  administradores ativos estão nesse cargo.

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
| Tirar `administrador` do cargo de todos os admins ativos | `409` | `o sistema precisa de pelo menos um administrador ativo` |

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

A lista precisa trazer **todas** as etapas do funil, cada uma uma vez; caso contrário a
resposta é `400` (`a reordenação precisa trazer todas as etapas do funil, sem repetição`).

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
4. O financeiro classifica a conta a pagar (fornecedor, categoria financeira e competência),
   se o solicitante não o fez.
5. O financeiro registra o pagamento — o **valor pago pode ser diferente** do estimado.

Desde o M7 a solicitação é a **conta a pagar** do grupo. Os campos de classificação são
opcionais na criação; o `prazo_pagamento` é o vencimento. As despesas fixas entram pela seção
"Despesas recorrentes", que gera as solicitações com `origem = | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`. O M9 gera as
contas a pagar da | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` | de título antecipado (`origem = recompra`) e do repasse ao fomento
(`origem = | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`): elas nascem em `pendente_aprovacao`, já com fornecedor (o fomento),
categoria e competência, e seguem o fluxo de aprovação normal.

### Estados

| De | Para | Rota |
| --- | --- | --- |
| `pendente_aprovacao` | `aprovado` | `POST /{id}/aprovar` |
| `pendente_aprovacao` | `rejeitado` | `POST /{id}/rejeitar` |
| `pendente_aprovacao`, `aprovado` | `cancelado` | `POST /{id}/cancelar` |
| `pendente_aprovacao` | `pendente_aprovacao` (edição) | `PUT /{id}` |
| `aprovado` | `aprovado` (classificação ou adiamento) | `PATCH /{id}/classificacao`, `POST /{id}/adiar` |
| `aprovado` | `pago` | `POST /{id}/pagamento` |
| `pago` | `aprovado` (estorno) | `POST /{id}/pagamento/estorno` |

`rejeitado` e `cancelado` são estados terminais; `pago` só sai pelo estorno, que volta para
`aprovado`. Toda transição grava um registro no histórico **na mesma transação**, e o `UPDATE` só
acontece se o status e a versão lidos ainda baterem: se outra pessoa mudar primeiro (inclusive uma
edição), a segunda recebe `409` (`solicitação alterada por outra operação, recarregue e tente
novamente`).

"Hoje" (prazo não anterior a hoje, `vencida`, `vencidas=true`, resumo) é o dia em **UTC**: entre
21h e 24h no horário de Brasília já vale o dia seguinte.

### Rotas

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/solicitacoes` | autenticado | `201` |
| `GET` | `/api/v1/solicitacoes` | autenticado | `200` lista paginada |
| `GET` | `/api/v1/solicitacoes/{id}` | quem pode ver | `200` objeto único |
| `POST` | `/api/v1/solicitacoes/{id}/aprovar` | aprovador ou administrador, exceto o próprio solicitante | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/rejeitar` | aprovador ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/cancelar` | solicitante ou administrador | `200` |
| `PATCH` | `/api/v1/solicitacoes/{id}/classificacao` | financeiro ou administrador | `200` |
| `PUT` | `/api/v1/solicitacoes/{id}` | solicitante ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/adiar` | financeiro ou administrador | `200` |
| `POST` | `/api/v1/solicitacoes/{id}/pagamento` | financeiro ou administrador | `201` |
| `GET` | `/api/v1/solicitacoes/{id}/pagamento` | quem pode ver | `200` objeto único (pagamento vigente) |
| `GET` | `/api/v1/solicitacoes/{id}/pagamentos` | quem pode ver | `200` lista, com os estornados |
| `GET` | `/api/v1/solicitacoes/pendencias` | autenticado | `200` contadores do usuário |
| `POST` | `/api/v1/solicitacoes/{id}/pagamento/estorno` | financeiro ou administrador | `200` |
| `GET` | `/api/v1/solicitacoes/{id}/historico` | quem pode ver | `200` lista |
| `GET` | `/api/v1/solicitacoes/{id}/arquivos` | quem pode ver | `200` lista |

**Quem pode ver** uma solicitação (roteiros `Obter`, `pagamento`, `pagamentos`, `historico`, `arquivos`):
o próprio solicitante, um administrador, o aprovador designado e o financeiro (que só enxerga
solicitações fora de `pendente_aprovacao`). Aprovador e financeiro só enxergam solicitações
de empresas do seu **acesso** (seção "Empresas"); o solicitante sempre vê as próprias. Quem
não pode ver recebe `403` (`perfil sem permissão para consultar esta solicitação`).
Aprovar, rejeitar e registrar pagamento de solicitação de empresa fora do acesso também
respondem `403` (`perfil sem acesso a esta empresa`).

### Escopos de listagem

`GET /api/v1/solicitacoes` aceita `pagina`, `tamanho`, `escopo`, `status`, `empresa` e os filtros
de conta a pagar da tabela no fim desta seção:

| `escopo` | Quem pode | O que lista |
| --- | --- | --- |
| vazio ou `minhas` | autenticado | as que ele mesmo criou |
| `aprovacao` | aprovador designado ou administrador | as `pendente_aprovacao` das empresas do seu acesso |
| `financeiro` | cargo com `"financeiro": true` ou administrador | as `aprovado` das empresas do seu acesso |
| `todas` | administrador | todas |

`empresa=<uuid>` restringe qualquer escopo a uma empresa. Nas filas (`aprovacao`,
`financeiro`) e em `todas`, empresa fora do acesso devolve `403` (`perfil sem acesso a esta
empresa`); em `minhas` o filtro vale mesmo sem acesso, porque o solicitante sempre vê as
próprias solicitações.

`status=` (ex.: `?escopo=todas&status=pago`) filtra dentro do que o escopo já deixa
enxergar — não serve para furar a definição do escopo. `escopo=aprovacao` só aceita
`status=pendente_aprovacao` e `escopo=financeiro` aceita qualquer status **exceto**
`pendente_aprovacao`; a combinação fora da regra devolve `400`
(`status não permitido para o escopo de listagem informado`). `escopo` fora da lista
devolve `400` (`escopo de listagem inválido`); escopo de outro perfil devolve `403`.

Filtros de conta a pagar, combináveis com qualquer escopo (valor inválido → `400`):

| Consulta | Filtra |
| --- | --- |
| `fornecedor=<uuid>` | fornecedor da solicitação |
| `categoria=<uuid>` | categoria financeira |
| `competencia=AAAA-MM` | competência (`erro de validação: competência deve estar no formato AAAA-MM`) |
| `origem=manual\|recorrencia\|recompra\|repasse_fomento\|comissao\|importacao_parcela\|importacao_custo\|divida_amortizacao\|divida_juros` | origem da solicitação |
| `vencimento_de=AAAA-MM-DD` / `vencimento_ate=AAAA-MM-DD` | intervalo do `prazo_pagamento`, inclusivo (`erro de validação: consulta vencimento_de deve estar no formato AAAA-MM-DD`; final antes do inicial → `400 erro de validação: vencimento final não pode ser anterior ao inicial`) |
| `vencidas=true` | `pendente_aprovacao` ou `aprovado` com vencimento antes de hoje (UTC); aceita `true`/`false`/`1`/`0`, outro valor → `400 erro de validação: consulta vencidas deve ser true ou false` |

### `POST /api/v1/solicitacoes`

```json
{
  "empresa_id": "uuid-da-empresa",
  "valor_centavos": 150000,
  "prazo_pagamento": "2026-10-20",
  "observacao": "Compra de cimento para a fase 2",
  "forma_pagamento": "pix",
  "fornecedor_id": "uuid-do-fornecedor",
  "categoria_financeira_id": "uuid-da-subcategoria",
  "competencia": "2026-10",
  "estimado": false,
  "arquivo_ids": ["3f1c2a4e-0000-4000-8000-000000000001"]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; a empresa precisa existir, estar ativa e estar no acesso do solicitante |
| `valor_centavos` | inteiro em centavos, maior que zero (`150000` = R$ 1.500,00) |
| `prazo_pagamento` | `AAAA-MM-DD` (ou RFC3339). Obrigatório e não pode ser anterior a hoje |
| `observacao` | obrigatória, no máximo 1000 caracteres |
| `forma_pagamento` | `pix`, `cartao`, `boleto`, `transferencia` ou `debito_automatico` |
| `fornecedor_id` | opcional. Fornecedor existente e ativo |
| `categoria_financeira_id` | opcional. Subcategoria ativa de natureza `despesa` (plano de contas) |
| `competencia` | opcional, `AAAA-MM` (de `2000-01` a `2099-12`). Padrão: o mês do `prazo_pagamento` |
| `estimado` | opcional, padrão `false`. `true` quando o valor é previsão (conta de consumo) |
| `arquivo_ids` | opcional. Cada anexo tem que existir, pertencer ao solicitante, aparecer uma vez só, não estar em outra solicitação e não ser XML (o XML é exclusivo da importação de NF-e) |

**Objeto de solicitação:**

```json
{
  "id": "00000000-0000-4000-8000-000000000001",
  "solicitante_id": "00000000-0000-4000-8000-000000000002",
  "empresa_id": "00000000-0000-4000-8000-000000000004",
  "aprovador_id": null,
  "valor_centavos": 150000,
  "prazo_pagamento": "2026-10-20T00:00:00Z",
  "observacao": "Compra de cimento para a fase 2",
  "forma_pagamento": "pix",
  "fornecedor_id": null,
  "categoria_financeira_id": null,
  "competencia": "2026-10",
  "estimado": false,
  "origem": "manual",
  "origem_id": null,
  "vencida": false,
  "versao": 1,
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
| Forma fora da lista | `400` | `erro de validação: forma de pagamento deve ser pix, cartao, boleto, transferencia ou debito_automatico` |
| Fornecedor inexistente / inativo | `400` | `erro de validação: fornecedor não encontrado` / `...fornecedor inativo não recebe conta a pagar` |
| Categoria inexistente | `400` | `erro de validação: categoria financeira não encontrada` |
| Categoria raiz ou inativa | `400` | `erro de validação: categoria financeira deve ser uma subcategoria ativa` |
| Categoria de receita ou transferência | `400` | `erro de validação: categoria financeira deve ser de despesa` |
| Competência fora do formato | `400` | `erro de validação: competência deve estar no formato AAAA-MM` |
| Anexo de outra pessoa | `403` | `arquivo não pertence ao solicitante` |
| Anexo já em uso | `400` | `erro de validação: arquivo já vinculado a outra solicitação` |
| Mesmo anexo duas vezes | `400` | `erro de validação: arquivo repetido na solicitação` |
| Sem `empresa_id` | `400` | `erro de validação: empresa da solicitação não informada` |
| Empresa inexistente (solicitante com acesso a todas) | `400` | `erro de validação: empresa da solicitação não encontrada` |
| Empresa fora do acesso, inclusive inexistente | `403` | `perfil sem acesso a esta empresa` |
| Empresa inativa | `400` | `erro de validação: empresa inativa não recebe solicitação nova` |

Campos de leitura do objeto:

| Campo | Significado |
| --- | --- |
| `competencia` | mês da despesa na DRE, `AAAA-MM` |
| `origem` / `origem_id` | `manual` (sem id) ou o módulo que gerou a solicitação e o id do registro de origem |
| `vencida` | `pendente_aprovacao` ou `aprovado` com `prazo_pagamento` antes de hoje (UTC); derivado, nunca gravado |
| `versao` | começa em `1` e avança em toda gravação, inclusive aprovação, rejeição, cancelamento e pagamento |

O histórico (`GET /{id}/historico`) traz `evento` em cada registro: `criada`, `aprovada`,
`rejeitada`, `cancelada`, `paga`, `editada`, `classificada`, `adiada` ou `estornada`. Eventos que
não mudam o status (como `classificada`) gravam `de_status` igual a `para_status`.

### `PATCH /api/v1/solicitacoes/{id}/classificacao`

Financeiro ou administrador, com a empresa da solicitação no acesso. Só em `aprovado`. Substitui a
classificação inteira: `fornecedor_id` omitido vira `null`. Em solicitação gerada por recorrência
(`origem` diferente de `manual`) a `competencia` precisa ser a mesma já gravada
(`400 erro de validação: competência de solicitação gerada por outro módulo não pode mudar`), porque
é ela que impede a geração duplicada.

```json
{
  "fornecedor_id": "uuid-do-fornecedor",
  "categoria_financeira_id": "uuid-da-subcategoria",
  "competencia": "2026-09",
  "versao": 2
}
```

| Campo | Regra |
| --- | --- |
| `fornecedor_id` | opcional; mesmas regras da criação |
| `categoria_financeira_id` | **obrigatório**; mesmas regras da criação |
| `competencia` | **obrigatória**, `AAAA-MM` |
| `versao` | **obrigatória**: a `versao` lida da solicitação |

Responde `200` com o objeto da solicitação (`versao` avançada) e grava o evento `classificada` no
histórico.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para classificar solicitação` |
| Empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| `versao` ausente, zero ou negativa | `400` | `erro de validação: versão da solicitação é obrigatória` |
| `versao` diferente da atual | `409` | `solicitação alterada por outra operação, recarregue e tente novamente` |
| Fora de `aprovado` | `409` | `só é possível classificar solicitação aprovada` |
| Sem categoria / sem competência | `400` | `erro de validação: categoria financeira é obrigatória na classificação` / `...competência é obrigatória na classificação` |

### `PUT /api/v1/solicitacoes/{id}`

Edição pelo solicitante ou pelo administrador, só em `pendente_aprovacao`. Substitui todos os
campos editáveis: o que for omitido volta ao padrão (`fornecedor_id` e `categoria_financeira_id`
viram `null`, `competencia` vira o mês do `prazo_pagamento`, `estimado` vira `false`). Empresa e
anexos não mudam por esta rota; depois de aprovada, corrigir é cancelar e criar outra.

```json
{
  "valor_centavos": 64000,
  "prazo_pagamento": "2026-10-24",
  "observacao": "Compra de cimento e areia",
  "forma_pagamento": "transferencia",
  "fornecedor_id": null,
  "categoria_financeira_id": null,
  "competencia": "2026-10",
  "estimado": true,
  "versao": 1
}
```

Campos com as mesmas regras do `POST`, mais `versao` obrigatória, com duas diferenças:

- a regra "prazo não anterior a hoje" só vale quando o vencimento **muda**: uma solicitação já
  vencida pode ser editada reenviando o mesmo `prazo_pagamento`;
- em solicitação gerada por recorrência a `competencia` não muda: omitida, fica a gravada (não
  é recalculada pelo vencimento); diferente, responde `400` como na classificação.

Fornecedor e categoria informados precisam continuar ativos e lançáveis, mesmo que não tenham
mudado. Responde `200` com o objeto atualizado e grava o evento `editada`, com os campos alterados
na descrição (`solicitação editada: valor, vencimento`). Corpo igual ao gravado responde `200` sem
avançar a versão nem gravar histórico. Fora de `pendente_aprovacao` responde `409` antes de validar
o corpo.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Nem solicitante nem administrador | `403` | `perfil sem permissão para editar esta solicitação` |
| `versao` ausente, zero ou negativa | `400` | `erro de validação: versão da solicitação é obrigatória` |
| `versao` diferente da atual | `409` | `solicitação alterada por outra operação, recarregue e tente novamente` |
| Fora de `pendente_aprovacao` | `409` | `só é possível editar solicitação pendente de aprovação` |
| Campo inválido | `400` | as mensagens do `POST` |

### `POST /api/v1/solicitacoes/{id}/adiar`

Financeiro ou administrador, com a empresa da solicitação no acesso. Só em `aprovado`. Muda o
`prazo_pagamento`; a competência e o status não mudam.

```json
{ "vencimento": "2026-11-13", "motivo": "fornecedor aceitou o novo prazo", "versao": 3 }
```

| Campo | Regra |
| --- | --- |
| `vencimento` | obrigatório, `AAAA-MM-DD`; posterior ao vencimento atual e não anterior a hoje |
| `motivo` | obrigatório, no máximo 500 caracteres |
| `versao` | obrigatória |

Responde `200` com o objeto atualizado e grava o evento `adiada` com a descrição
`vencimento adiado de 2026-10-24 para 2026-11-13: fornecedor aceitou o novo prazo`.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para adiar solicitação` |
| Empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| `versao` diferente da atual | `409` | `solicitação alterada por outra operação, recarregue e tente novamente` |
| Fora de `aprovado` | `409` | `só é possível adiar solicitação aprovada` |
| Sem vencimento / formato errado | `400` | `erro de validação: vencimento é obrigatório` / `...vencimento deve estar no formato AAAA-MM-DD` |
| Vencimento no mesmo dia ou antes do atual | `400` | `erro de validação: novo vencimento deve ser posterior ao atual` |
| Vencimento antes de hoje | `400` | `erro de validação: prazo de pagamento não pode ser anterior a hoje` |
| Sem motivo / motivo longo | `400` | `erro de validação: motivo do adiamento é obrigatório` / `...deve ter no máximo 500 caracteres` |

### `POST /api/v1/solicitacoes/{id}/aprovar`

Corpo opcional: `{ "versao": n }`. Com `versao`, a aprovação só acontece se a solicitação ainda
estiver na versão que o aprovador viu; uma edição feita depois responde `409`
(`solicitação alterada por outra operação, recarregue e tente novamente`), em vez de aprovar um
valor que ele não conferiu. O front deve sempre mandar a `versao`; sem corpo, vale o comportamento
anterior. Ninguém aprova a própria solicitação, nem o administrador: a segregação entre quem pede e
quem aprova vale para todos os perfis.

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
  "conta_bancaria_id": "00000000-0000-4000-8000-000000000006",
  "valor_centavos": 148500,
  "comprovante_arquivo_id": "00000000-0000-4000-8000-000000000004",
  "pago_em": "2026-10-15T14:00:00Z"
}
```

A solicitação precisa estar **classificada** (com `categoria_financeira_id`) antes de ser paga;
as antigas e as que o solicitante deixou sem categoria passam antes pelo
`PATCH /{id}/classificacao`.

| Campo | Regra |
| --- | --- |
| `conta_bancaria_id` | **obrigatório desde o M7**: conta de onde o dinheiro saiu, ativa e da mesma empresa da solicitação; a empresa precisa estar ativa |
| `valor_centavos` | valor **efetivo** pago, maior que zero — pode diferir do estimado |
| `comprovante_arquivo_id` | opcional. Quando presente tem que ser um PDF **de quem está registrando** e ainda não vinculado a outra solicitação. O comprovante de um pagamento estornado pode ser usado de novo no pagamento seguinte da mesma solicitação |
| `pago_em` | opcional, RFC3339. Padrão: agora (UTC). Não pode estar no futuro (`400 erro de validação: data do pagamento não pode estar no futuro`). É a data em que o dinheiro saiu; `criado_em` do pagamento e `atualizado_em` da solicitação registram o momento do lançamento |

**Objeto de pagamento:**

```json
{
  "id": "00000000-0000-4000-8000-000000000005",
  "solicitacao_id": "00000000-0000-4000-8000-000000000001",
  "conta_bancaria_id": "00000000-0000-4000-8000-000000000006",
  "usuario_id": "00000000-0000-4000-8000-000000000007",
  "comprovante_arquivo_id": "00000000-0000-4000-8000-000000000004",
  "valor_centavos": 148500,
  "pago_em": "2026-10-15T14:00:00Z",
  "estornado": false,
  "estorno": null,
  "criado_em": "2026-10-15T14:02:10Z"
}
```

`usuario_id` é quem registrou o pagamento. Pagamentos anteriores ao M7 têm `conta_bancaria_id` e
`usuario_id` nulos. Em pagamento estornado, `estorno` traz
`{ "id", "motivo", "estornado_em", "usuario_id", "criado_em" }`.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para registrar pagamento` |
| Sem `conta_bancaria_id` | `400` | `erro de validação: conta bancária do pagamento é obrigatória` |
| Conta inexistente | `400` | `erro de validação: conta bancária não encontrada` |
| Conta de outra empresa | `400` | `erro de validação: conta bancária deve ser da empresa da solicitação` |
| Conta inativa | `400` | `erro de validação: conta bancária inativa não recebe pagamento` |
| Empresa inativa | `400` | `erro de validação: empresa inativa não recebe pagamento` |
| Solicitação sem categoria | `400` | `erro de validação: solicitação sem categoria financeira; classifique antes de pagar` |
| Categoria desativada depois da classificação | `400` | `erro de validação: categoria financeira deve ser uma subcategoria ativa` (reclassifique) |
| Status diferente de `aprovado` | `409` | `não é possível mudar a solicitação de "<status atual>" para "pago"` |
| Comprovante não PDF | `400` | `comprovante deve ser um arquivo PDF` |
| Comprovante de outro usuário | `403` | `comprovante não pertence ao usuário` |
| Comprovante já usado em outra solicitação | `400` | `comprovante já vinculado a outra solicitação` |
| Sem pagamento vigente | `404` | `registro não encontrado: pagamento não encontrado` (em `GET /{id}/pagamento`, inclusive depois de um estorno) |

A resposta de `aprovar`, `rejeitar` e `cancelar` devolve o objeto da solicitação já com
o status novo; a de `POST /{id}/pagamento` devolve o objeto do pagamento.

### `POST /api/v1/solicitacoes/{id}/pagamento/estorno`

Financeiro ou administrador, com a empresa da solicitação no acesso. Só em `pago`. Estorna o
pagamento vigente como lançamento inverso: o pagamento continua gravado, marcado como
`estornado`, o estorno fica registrado ao lado e a solicitação **volta para `aprovado`**, pronta
para um novo pagamento ou para ser cancelada.

```json
{ "motivo": "pago em duplicidade", "estornado_em": "2026-10-16T09:00:00Z", "versao": 4 }
```

| Campo | Regra |
| --- | --- |
| `motivo` | obrigatório, no máximo 500 caracteres |
| `estornado_em` | opcional, RFC3339. Padrão: agora. Não pode estar no futuro nem antes do `pago_em` |
| `versao` | obrigatória |

Responde `200` com o objeto da solicitação (status `aprovado`, `versao` avançada) e grava o
evento `estornada` (`pagamento estornado: <motivo>`).

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para estornar pagamento` |
| Empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| `versao` diferente da atual | `409` | `solicitação alterada por outra operação, recarregue e tente novamente` |
| Fora de `pago` | `409` | `só é possível estornar solicitação paga` |
| Sem motivo | `400` | `erro de validação: motivo do estorno é obrigatório` |
| Data no futuro / antes do pagamento | `400` | `erro de validação: data do estorno não pode estar no futuro` / `...não pode ser anterior ao pagamento` |
| Data fora do formato | `400` | `erro de validação: data do estorno deve estar no formato AAAA-MM-DDTHH:MM:SSZ` |

### `GET /api/v1/solicitacoes/{id}/pagamentos`

Todos os pagamentos da solicitação, do mais antigo ao mais novo, inclusive os estornados com o
`estorno` preenchido. Mesma regra de "quem pode ver" das outras consultas.

### `GET /api/v1/solicitacoes/pendencias`

Contadores para o badge do front, nas empresas do acesso do usuário. Cada contador só vem
preenchido quando o perfil enxerga aquela fila; fora do perfil vem `null` (não `0`).

```json
{ "dados": { "aprovacao": 4, "financeiro": 7, "vencidas": 2 } }
```

| Campo | Quem recebe | Conta |
| --- | --- | --- |
| `aprovacao` | aprovador designado ou administrador | `pendente_aprovacao` (a mesma fila de `escopo=aprovacao`) |
| `financeiro` | financeiro ou administrador | `aprovado` (a mesma fila de `escopo=financeiro`) |
| `vencidas` | financeiro ou administrador | `pendente_aprovacao` ou `aprovado` com vencimento antes de hoje (UTC) |

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
| Arquivo ausente no armazenamento | `404` `registro não encontrado: arquivo não encontrado no armazenamento` (registro existe, mas o conteúdo sumiu do disco ou do bucket) |
| Tempo de transferência | até 5 minutos no envio (`POST /api/v1/arquivos`) e no download (`GET /api/v1/arquivos/{id}` e `GET /api/v1/publico/imagens/{id}`); as demais rotas têm 15 segundos para ler a requisição e escrever a resposta |
| Tipos aceitos | `application/pdf`, `image/png`, `image/jpeg`, `image/webp`, `application/xml` (`text/xml` é gravado como `application/xml`; só para a importação de NF-e, não vira anexo de solicitação), `application/x-ofx` (também `application/ofx`, `text/ofx`, ou `application/octet-stream`, `text/plain` e tipo vazio quando o nome termina em `.ofx`; só para a importação de extrato, não vira anexo de solicitação) |
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
| `GET` | `/api/v1/produtos/{id}/saldo` | autenticado | `200` saldo total e por local (ver **Estoque — locais**) |
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
| `GET` | `/api/v1/produtos/{id}/movimentos` | autenticado | `200` lista paginada (`?tipo=`, `?local=`) |

```json
{
  "local_id": "uuid-do-galpao",
  "tipo": "entrada",
  "quantidade": 10,
  "custo_unitario_centavos": 2590,
  "documento_ref": "NF-100",
  "observacao": "reposição do mês"
}
```

Todo movimento acontece num **local de estoque** (`local_id` obrigatório; ver
**Estoque — locais**). O local precisa estar no acesso do usuário (`403`).

| Tipo | Efeito no local | Custo (`custo_unitario_centavos`) | Lote |
| --- | --- | --- | --- |
| `entrada` | `+` | **obrigatório** (zero é aceito) | cria lote |
| `devolucao` | `+` | opcional (`null` = desconhecido) | cria lote |
| `ajuste` positivo | `+` | opcional | cria lote |
| `implantacao` | `+`, sem mudar o total do produto | opcional | cria lote; limitado ao saldo **não alocado** |
| `ajuste` negativo | `−` (`quantidade` é o delta, aceita negativo) | não aceita | consome FIFO |
| `saida` / `perda` | `−` | não aceita | consome FIFO |

`quantidade` não pode ser `0`. Movimento é append-only: não edita, não apaga, e o
`saldo_apos` (total do produto) e o `saldo_local_apos` ficam congelados no registro.
Saídas só consomem o **disponível** do local (saldo − reservado). `transferencia` tem rota
própria.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem `local_id` ou local inexistente | `400` | `erro de validação: local do movimento não encontrado` |
| Entrada sem custo | `400` | `erro de validação: custo unitário é obrigatório na entrada` |
| Custo em saída, perda ou ajuste negativo | `400` | `erro de validação: custo unitário só é informado em movimento que cria lote` |
| Acima do disponível no local | `400` | `erro de validação: saldo disponível insuficiente no local` |
| Implantação acima do não alocado | `400` | `erro de validação: quantidade maior que o saldo ainda não alocado do produto` |
| Entrada em local inativo | `400` | `erro de validação: local inativo não recebe entrada de estoque` |
| `tipo` = `transferencia` | `400` | `erro de validação: transferência usa a rota de transferências` |
| Local fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Estoque alterado por outra operação | `409` | `estoque do produto alterado por outra operação, recarregue e tente novamente` |

Para desfazer uma movimentação registra-se a inversa (`entrada` ↔ `saida`,
`devolucao` ↔ `perda`); o histórico permanece completo.

O objeto de movimento traz também `local_id`, `local_destino_id` (só na `transferencia`) e
`saldo_local_apos` (saldo do local, de origem na transferência, depois do movimento), além de
`reserva_id` na saída que baixou uma reserva; movimentos anteriores aos locais vêm com os três
primeiros em `null`. Corpo do `POST` aceita também `reserva_id` (ver **Reservas**). O custo **não** aparece aqui:
ele fica na rota de lotes, restrita a administrador e financeiro.

A listagem mostra só os movimentos de locais das empresas do acesso do usuário, mais os
anteriores aos locais. `?local=<uuid>` filtra pela origem ou pelo destino; local
inexistente responde `400` e local fora do acesso, `403`.

---

## Estoque — resumo

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/estoque/resumo` | autenticado | `200` (`?empresa=` opcional) |

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
| `ultimos_movimentos` | os 5 mais recentes dos locais das empresas do acesso (mais os anteriores aos locais), com o nome do produto (não expõe `usuario_id` nem observações); `?empresa=` restringe a uma empresa do acesso, fora dele `403` |

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
| `arquivo_id` | obrigatório, precisa existir, ser **imagem** (`400 arquivo deve ser uma imagem`) e não pertencer a uma solicitação (`400 arquivo de uma solicitação não pode virar imagem pública de produto`) |
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
edição busca o detalhe). O produto não tem exclusão — só desativação. Ao remover a imagem,
o arquivo também é apagado do cadastro e do storage quando nenhum outro produto o usa.

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
| `/publico/categorias` | só ativas, montadas em árvore (`filhas[]`), casamento por `categoria_id`; subcategoria de pai inativo não aparece |
| `/publico/produtos` | `?categoria=<slug>` aceita raiz (inclui as subcategorias), `?pagina` e `?tamanho` |
| `/publico/produtos/{slug}` | slug desconhecido ou produto oculto → `404` |
| `/publico/destaques` | ativos com destaque **e** estoque |
| `/publico/imagens/{id}` | `Content-Type` original, `X-Content-Type-Options: nosniff` e `Cache-Control: public, max-age=300`; imagem de produto fora da vitrine → `404` |

O objeto público **não expõe** `saldo`, `codigo`, `ativo`, `estoque_minimo`, `criado_em`
nem `atualizado_em`. Sem estoque **disponível** (saldo − reservado) o produto some da vitrine
inteira (lista, destaques e detalhe) — não há selo "esgotado"; produto todo reservado também
some. O mesmo vale para produto de categoria inativa ou de
subcategoria cujo pai está inativo. Categoria desconhecida ou oculta em `?categoria=` devolve `404`.

---

## Empresas — administrador escreve, autenticado lê o próprio acesso

Cada CNPJ do grupo é uma empresa (inclusive filiais). Usuários enxergam só as empresas do
seu **acesso**: o administrador vê todas; os demais veem as vinculadas a eles ou todas,
quando `acesso_todas` está ligado.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/empresas` | **administrador** | `201` objeto único |
| `GET` | `/api/v1/empresas` | autenticado | `200` lista paginada do acesso (`?ativas=true` só ativas, `?ativas=false` só inativas) |
| `GET` | `/api/v1/empresas/{id}` | autenticado | `200` se estiver no acesso, senão `403` (inclusive para id inexistente); `404` só para quem tem acesso a todas |
| `PUT` | `/api/v1/empresas/{id}` | **administrador** | `200` |
| `PATCH` | `/api/v1/empresas/{id}/ativar` | **administrador** | `200` |
| `PATCH` | `/api/v1/empresas/{id}/desativar` | **administrador** | `200` |
| `DELETE` | `/api/v1/empresas/{id}` | **administrador** | `204` |
| `GET` | `/api/v1/usuarios/{id}/empresas` | **administrador** | `200` vínculos do usuário |
| `PUT` | `/api/v1/usuarios/{id}/empresas` | **administrador** | `200` vínculos gravados |

### `POST /api/v1/empresas`

```json
{
  "cnpj": "11.222.333/0001-81",
  "razao_social": "MIA Construtora LTDA",
  "nome_fantasia": "MIA",
  "uf": "PB",
  "regime_tributario": "presumido"
}
```

| Campo | Regra |
| --- | --- |
| `cnpj` | obrigatório, com ou sem máscara; dígitos verificadores conferidos; aceita o formato alfanumérico (12 caracteres `0-9A-Z` + 2 dígitos); não muda depois de criado |
| `razao_social` | obrigatório, até 200 caracteres |
| `nome_fantasia` | obrigatório, até 120 caracteres |
| `uf` | sigla de um dos 27 estados, maiúscula ou minúscula |
| `regime_tributario` | `simples`, `presumido` ou `real` |

`PUT /api/v1/empresas/{id}` usa o mesmo corpo **sem** `cnpj`; enviar `cnpj` responde `400`.

| Erro | Status | Mensagem |
| --- | --- | --- |
| CNPJ inválido | `400` | `erro de validação: cnpj inválido` |
| CNPJ já cadastrado | `409` | `já existe uma empresa com este cnpj` |
| UF fora da lista | `400` | `erro de validação: uf deve ser a sigla de um estado brasileiro` |
| Regime fora da lista | `400` | `erro de validação: regime tributário deve ser simples, presumido ou real` |
| Empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Excluir empresa com vínculos | `400` | `erro de validação: registro em uso por outros dados e não pode ser excluído` |

**Objeto de empresa:**

```json
{
  "id": "uuid",
  "razao_social": "MIA Construtora LTDA",
  "nome_fantasia": "MIA",
  "cnpj": "11222333000181",
  "cnpj_formatado": "11.222.333/0001-81",
  "uf": "PB",
  "regime_tributario": "presumido",
  "ativa": true,
  "criado_em": "2026-10-10T14:26:19.854486Z",
  "atualizado_em": "2026-10-10T14:26:19.854486Z"
}
```

O caminho normal para tirar uma empresa de uso é desativar: ela continua no histórico,
mas não recebe registro novo. A exclusão só funciona para empresa sem nenhum vínculo.

### `PUT /api/v1/usuarios/{id}/empresas`

Substitui o conjunto inteiro de vínculos do usuário:

```json
{ "acesso_todas": false, "empresa_ids": ["uuid-da-mia", "uuid-da-gigante-ce"] }
```

Com `acesso_todas: true` o usuário enxerga todas as empresas, inclusive as criadas depois,
e a lista é guardada para quando o acesso total for desligado. Empresas repetidas são
ignoradas; empresa inexistente responde `400`. A resposta (e o `GET` da mesma rota) devolve
o mesmo formato. Usuários criados depois do M1 nascem **sem** acesso; os que existiam antes
receberam `acesso_todas: true` na migração.

A rota mostra os vínculos **gravados**, não o acesso efetivo: um usuário com cargo
`administrador` enxerga todas as empresas mesmo que a rota devolva
`acesso_todas: false` e `empresa_ids: []`.

---

## Estoque — locais, saldo por local, lotes e transferência

Cada local de estoque pertence a uma empresa; o saldo de cada produto existe por local, e
o usuário só enxerga os locais das empresas do seu acesso (mesmo escopo das solicitações).
O produto continua do grupo: `saldo` do produto é a soma de todos os locais mais o saldo
**não alocado** (o que existia antes dos locais e ainda não foi implantado num local).

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/estoque/locais` | **administrador** | `201` objeto único |
| `GET` | `/api/v1/estoque/locais` | autenticado | `200` lista paginada do acesso (`?empresa=`, `?ativos=true\|false`) |
| `GET` | `/api/v1/estoque/locais/{id}` | autenticado | `200` se a empresa do local estiver no acesso, senão `403` |
| `PUT` | `/api/v1/estoque/locais/{id}` | **administrador** | `200` (só `nome`) |
| `PATCH` | `/api/v1/estoque/locais/{id}/ativar` | **administrador** | `200` |
| `PATCH` | `/api/v1/estoque/locais/{id}/desativar` | **administrador** | `200`; `400` se houver saldo |
| `DELETE` | `/api/v1/estoque/locais/{id}` | **administrador** | `204`; `400` se tiver saldo, lote ou movimento |
| `GET` | `/api/v1/estoque/locais/{id}/saldos` | autenticado | `200` produtos do local, paginado (`?com_saldo=true`) |
| `GET` | `/api/v1/produtos/{id}/lotes` | administrador ou financeiro | `200` lotes do acesso, paginado (`?local=`, `?com_saldo=true`) |
| `POST` | `/api/v1/produtos/{id}/transferencias` | **administrador** | `201` movimento `transferencia` |
| `POST` | `/api/v1/produtos/{id}/reservas` | **administrador** | `201` reserva |
| `GET` | `/api/v1/produtos/{id}/reservas` | autenticado | `200` reservas dos locais do acesso, mais recentes primeiro (`?status=`, `?local=`) |
| `POST` | `/api/v1/reservas/{id}/liberar` | **administrador** | `200` reserva liberada |

### `POST /api/v1/estoque/locais`

```json
{ "empresa_id": "uuid-da-mia", "nome": "Galpão Cabedelo" }
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; precisa estar no acesso (`403`, inclusive para id inexistente), existir e estar ativa (`400`); não muda depois |
| `nome` | obrigatório, até 120 caracteres, único por empresa sem diferenciar maiúsculas (`409 já existe um local com este nome na empresa`) |

Local inativo não recebe entrada nem transferência de destino; saídas continuam para
esvaziá-lo. Objeto de local: `id`, `empresa_id`, `nome`, `ativo`, `criado_em`,
`atualizado_em`.

### `GET /api/v1/produtos/{id}/saldo`

```json
{
  "dados": {
    "produto_id": "uuid",
    "saldo": 14,
    "reservado": 0,
    "disponivel": 14,
    "nao_alocado": 4,
    "locais": [
      {
        "produto_id": "uuid",
        "produto_nome": "Cimento CP II 50",
        "produto_codigo": "SKU-001",
        "unidade_medida": "sc",
        "local_id": "uuid",
        "local_nome": "Galpão Cabedelo",
        "empresa_id": "uuid",
        "quantidade": 10,
        "reservado": 0,
        "disponivel": 10,
        "atualizado_em": "2026-10-11T12:00:00Z"
      }
    ]
  }
}
```

`saldo`, `reservado`, `disponivel` e `nao_alocado` são do grupo; `locais` traz só os locais
das empresas do acesso. `GET /api/v1/estoque/locais/{id}/saldos` devolve itens no mesmo
formato de `locais`, um por produto.

### `GET /api/v1/produtos/{id}/lotes`

Cada entrada cria um lote com o custo informado; saídas consomem os lotes mais antigos
primeiro (FIFO). Lista em ordem de entrada:

```json
{
  "id": "uuid",
  "produto_id": "uuid",
  "local_id": "uuid",
  "movimento_id": "uuid",
  "lote_origem_id": null,
  "custo_unitario_centavos": 2590,
  "quantidade_inicial": 10,
  "quantidade_atual": 4,
  "entrada_em": "2026-10-11T12:00:00Z"
}
```

`custo_unitario_centavos: null` é custo desconhecido. `lote_origem_id` aparece no lote que
nasceu de transferência e aponta para o lote de origem (mesmo custo e mesma data de
entrada). Perfil sem administrador nem financeiro responde `403`.

### `POST /api/v1/produtos/{id}/transferencias`

```json
{
  "local_origem_id": "uuid-do-galpao",
  "local_destino_id": "uuid-da-loja",
  "quantidade": 4,
  "documento_ref": "ROM-12",
  "observacao": "abastecer a loja"
}
```

| Erro | Status | Mensagem |
| --- | --- | --- |
| Local inexistente | `400` | `erro de validação: local de origem não encontrado` / `...de destino...` |
| Mesmo local | `400` | `erro de validação: local de destino deve ser diferente do de origem` |
| Locais de empresas diferentes | `400` | `erro de validação: transferência só entre locais da mesma empresa` |
| Destino inativo | `400` | `erro de validação: local inativo não recebe entrada de estoque` |
| Acima do disponível na origem | `400` | `erro de validação: saldo disponível insuficiente no local de origem` |
| Local fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Produto inexistente | `404` | `registro não encontrado: produto não encontrado` |
| Estoque alterado por outra operação | `409` | `estoque do produto alterado por outra operação, recarregue e tente novamente` |

A transferência não muda o saldo total do produto: consome os lotes da origem por FIFO e
cria no destino lotes com o mesmo custo e a mesma data de entrada.

### Reservas

Reserva separa uma quantidade do **disponível** de um local (disponível = saldo −
reservado) sem tirar do estoque. Saída, perda, ajuste negativo, transferência e reserva nova
só usam o disponível; a vitrine pública também.

```json
{ "local_id": "uuid-do-galpao", "quantidade": 3, "observacao": "pedido 123 da Construtora Alfa" }
```

Objeto de reserva: `id`, `produto_id`, `local_id`, `quantidade`, `origem` (`manual`; os pedidos
do M6 usarão `pedido_item` com `origem_id`), `origem_id`, `status`, `usuario_id`,
`observacao`, `criado_em`, `encerrada_em`.

| Status | Como chega |
| --- | --- |
| `ativa` | criada |
| `baixada` | `POST /produtos/{id}/movimentos` com `tipo: "saida"`, `reserva_id`, o mesmo `local_id` e a mesma `quantidade` da reserva: a saída consome a reserva e os lotes juntos |
| `liberada` | `POST /reservas/{id}/liberar`: a quantidade volta ao disponível |

| Erro | Status | Mensagem |
| --- | --- | --- |
| Acima do disponível no local | `400` | `erro de validação: saldo disponível insuficiente no local para a reserva` |
| `reserva_id` inexistente na saída | `400` | `erro de validação: reserva não encontrada` |
| Baixa com outro tipo, local ou quantidade | `400` | `só a saída baixa reserva` / `reserva de outro produto ou local` / `saída que baixa reserva deve ter a quantidade da reserva` |
| Reserva já baixada ou liberada | `409` | `reserva já encerrada` |
| Reserva inexistente ao liberar | `404` | `registro não encontrado: reserva não encontrada` |
| Local fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Liberar ou baixar reserva de item de pedido (`origem = pedido_item`) | `409` | `reserva de item de pedido só é liberada ou baixada pelo próprio pedido` |

O movimento da baixa traz `reserva_id`.

---

## Clientes, obras, grupos e representantes

Cadastro único de clientes do **grupo** (sem escopo por empresa). Perfis: **comercial** (ou
administrador) escreve clientes, obras e grupos; **administrador** gerencia representantes e
exclui clientes; qualquer autenticado lê clientes, obras e grupos; representantes são lidos pelo
comercial.

**Representante com login**: usuário vinculado a um representante (e que não é administrador)
só vê e altera os clientes em que é o representante — cliente, obras ou filtro
`?representante=` de outro → `403` (`cliente de outro representante`); cliente que ele cria ou
altera fica sempre com ele como representante (informar outro → `403`); em representantes, só
vê o próprio cadastro.

### Clientes

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/clientes` | **comercial** | `201` |
| `GET` | `/api/v1/clientes` | autenticado | `200` lista paginada (`?busca=`, `?grupo=`, `?representante=`, `?tipo=`, `?uf=`, `?ativo=`, `?lead=`) |
| `GET` | `/api/v1/clientes/{id}` | autenticado | `200` |
| `PUT` | `/api/v1/clientes/{id}` | **comercial** | `200` (mesmo corpo, **sem** `documento`) |
| `PATCH` | `/api/v1/clientes/{id}/ativar` / `desativar` | **comercial** | `200` |
| `DELETE` | `/api/v1/clientes/{id}` | **administrador** | `204`; `400 registro em uso` com obra ou outro vínculo |

```json
{
  "documento": "11.222.333/0001-81",
  "nome": "Construtora Águia LTDA",
  "nome_fantasia": "Águia",
  "tipo": "construtora",
  "grupo_id": "uuid-do-grupo",
  "uf": "PB",
  "cidade": "João Pessoa",
  "empresa_relacionamento_id": "uuid-da-empresa",
  "representante_id": "uuid-do-representante",
  "email_nf": "nf@aguia.com.br",
  "telefone": "83 99999-0000",
  "observacoes": "",
  "exige_numero_pedido_na_nf": true,
  "proibe_antecipacao": false
}
```

| Campo | Regra |
| --- | --- |
| `documento` | obrigatório; CPF ou CNPJ (numérico ou alfanumérico), com ou sem máscara, dígitos conferidos; único (`409 já existe um cliente com este documento`); não muda depois de criado |
| `tipo` | `construtora`, `loja`, `pf` ou `outro`; `pf` exige CPF e CPF exige `pf` |
| `nome` / `cidade` / `uf` | obrigatórios (`nome` até 200, `cidade` até 120, `uf` sigla) |
| `grupo_id`, `representante_id`, `empresa_relacionamento_id` | opcionais; precisam existir (`400`); ao atribuir ou trocar, precisam estar ativos (o vínculo inativo já gravado é mantido) |
| `email_nf` | um e-mail só, destinatário da NF-e |
| `exige_numero_pedido_na_nf`, `proibe_antecipacao` | regras comerciais tipadas, usadas pela NF-e e pela antecipação |

`?busca=` procura pelo nome sem diferenciar acento e maiúscula e pelo começo do documento sem
máscara (`aguia` acha "Águia"; `529.982` acha o CPF). A resposta traz `documento` (sem
máscara), `documento_formatado`, `tipo_pessoa` (`pf`/`pj`), os ids dos vínculos, `lead_id`,
`ativo`, `criado_em` e `atualizado_em`.

### Conversão de lead em cliente

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/leads/{id}/converter` | **comercial** | `201` cliente criado |

Corpo igual ao de `POST /api/v1/clientes`; `nome`, `email_nf` e `telefone` vazios são
preenchidos com os dados do lead. O cliente guarda `lead_id` (`GET /api/v1/clientes?lead=<id>`
encontra o cliente de um lead). O lead não muda.

| Erro | Status | Mensagem |
| --- | --- | --- |
| Lead inexistente | `404` | `registro não encontrado: lead não encontrado` |
| Lead já convertido | `409` | `lead já convertido em cliente` |
| Documento já cadastrado | `409` | `já existe um cliente com este documento` |

Lead convertido não pode ser excluído (`DELETE /api/v1/leads/{id}` → `400 registro em uso`).

### Importação de clientes por CSV

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/clientes/importacao` | **administrador** | `200` relatório (`?simular=true` valida sem gravar) |

Corpo: o arquivo CSV cru (`Content-Type: text/csv`), em **UTF-8** (com ou sem BOM) ou em
**Windows-1252** (o "CSV (separado por vírgulas)" padrão do Excel em português; o arquivo que não
é UTF-8 válido é lido como Windows-1252), separador `;`,
campos com `;` entre aspas, até 5 MB (`413` acima) e 5.000 linhas (`400` acima). Cabeçalho
fixo, nesta ordem:

```
documento;nome;nome_fantasia;tipo;grupo;uf;cidade;email_nf;telefone;representante_documento;exige_numero_pedido_na_nf;proibe_antecipacao;observacoes
```

- Cada linha segue as mesmas regras de `POST /api/v1/clientes`; linha válida é gravada, linha
  inválida volta no relatório com o número da linha no arquivo e o motivo. Nada é sobrescrito:
  documento já cadastrado ou repetido no arquivo rejeita a linha.
- `grupo` pelo nome (sem diferenciar acento e maiúscula), criado se não existir; grupo
  inativo rejeita a linha.
- `representante_documento`: CPF ou CNPJ de um representante cadastrado e ativo.
- `?simular=` aceita `true` ou `false`; outro valor (inclusive `sim`) → `400`.
- Flags `exige_numero_pedido_na_nf` e `proibe_antecipacao`: `sim`/`nao` (vazio = `nao`).
- Cabeçalho diferente, número de colunas errado ou CSV malformado responde `400` sem gravar
  nada.

```json
{
  "dados": {
    "simulacao": false,
    "total": 4,
    "importadas": 2,
    "rejeitadas": [
      { "linha": 4, "documento": "11222333000181", "motivo": "documento repetido no arquivo (linha 2)" },
      { "linha": 5, "documento": "111.444.777-35", "motivo": "cliente do tipo pf exige CPF, e CPF exige o tipo pf" }
    ]
  }
}
```

### Obras do cliente

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/clientes/{id}/obras` | **comercial** | `201` |
| `GET` | `/api/v1/clientes/{id}/obras` | autenticado | `200` lista paginada (`?ativas=`) |
| `GET` | `/api/v1/obras/{id}` | autenticado | `200` |
| `PUT` | `/api/v1/obras/{id}` | **comercial** | `200` |
| `PATCH` | `/api/v1/obras/{id}/ativar` / `desativar` | **comercial** | `200` |

```json
{
  "nome": "Residencial Mar Azul",
  "cnpj": "33.400.004/0001-16",
  "logradouro": "Av. Beira Mar",
  "numero": "100",
  "complemento": null,
  "bairro": "Intermares",
  "cidade": "Cabedelo",
  "uf": "PB",
  "cep": "58310-000"
}
```

`nome`, `logradouro`, `cidade` e `uf` são obrigatórios; nome único dentro do cliente sem
diferenciar maiúscula (`409 já existe uma obra com este nome no cliente`); `cnpj` (da SPE) e
`cep` (8 dígitos) opcionais, devolvidos sem máscara (`cnpj_formatado` também). Cliente inativo
não recebe obra nova (`400`).

### Grupos de clientes

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/cliente-grupos` | **comercial** | `201` `{ "nome": "Grupo Águia" }` |
| `GET` | `/api/v1/cliente-grupos` | autenticado | `200` lista paginada (`?busca=`, `?ativos=`) |
| `GET` | `/api/v1/cliente-grupos/{id}` | autenticado | `200` |
| `PUT` | `/api/v1/cliente-grupos/{id}` | **comercial** | `200` (renomeia) |
| `PATCH` | `/api/v1/cliente-grupos/{id}/ativar` / `desativar` | **comercial** | `200` |

Nome único sem diferenciar acento e maiúscula (`409 já existe um grupo de clientes com este nome`).

### Representantes

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/representantes` | **administrador** | `201` |
| `GET` | `/api/v1/representantes` | comercial | `200` lista paginada (`?busca=` nome ou documento, `?ativos=`) |
| `GET` | `/api/v1/representantes/{id}` | comercial | `200` |
| `PUT` | `/api/v1/representantes/{id}` | **administrador** | `200` |
| `PATCH` | `/api/v1/representantes/{id}/ativar` / `desativar` | **administrador** | `200` |
| `PUT` | `/api/v1/representantes/{id}/fornecedor` | **administrador** | `200`; corpo `{ "fornecedor_id": "…" }` ou `{ "fornecedor_id": null }` para tirar; o campo é obrigatório |

```json
{
  "nome": "José Vendas",
  "documento": "529.982.247-25",
  "usuario_id": "uuid-do-usuario",
  "email": "jose@vendas.com",
  "telefone": null,
  "comissao_pronta_entrega_bp": 300,
  "comissao_importacao_bp": 150
}
```

| Campo | Regra |
| --- | --- |
| `documento` | opcional; CPF ou CNPJ; único (`409 já existe um representante com este documento`) |
| `usuario_id` | opcional; usuário precisa existir (`400`) e não estar ligado a outro representante (`409 usuário já vinculado a outro representante`) |
| `comissao_*_bp` | percentual padrão em pontos-base (`300` = 3,00%), de `0` a `10000` |

Representante inativo não recebe cliente novo. O `PUT` substitui o cadastro inteiro, inclusive o
vínculo: `PUT` sem `usuario_id` desfaz o vínculo, e esse usuário (se tiver perfil comercial)
passa a ver todos os clientes. A paginação vale também para o representante com login, que vê
só o próprio cadastro na primeira página.

`fornecedor_id` (na resposta) é o fornecedor do M4 que recebe a conta a pagar do fechamento de
comissões; sem ele o representante não fecha (seção Comissões). Ele é definido só pela rota própria
`PUT /representantes/{id}/fornecedor`: o `PUT` do cadastro não o altera (e as duas rotas não se
desfazem uma à outra). Corpo sem `fornecedor_id` → `400`; fornecedor inexistente → `400 erro de
validação: fornecedor do representante não encontrado`; fornecedor inativo → `400 erro de validação:
fornecedor inativo não pode receber as comissões do representante`.

---

## Fornecedores

Cadastro único de fornecedores do **grupo** (sem escopo por empresa), nacionais e estrangeiros.
Perfis: **financeiro** (ou administrador) cria, altera, ativa e desativa; **administrador**
exclui; qualquer autenticado lê.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/fornecedores` | **financeiro** | `201` |
| `GET` | `/api/v1/fornecedores` | autenticado | `200` lista paginada (`?busca=`, `?pais=`, `?moeda=`, `?ativo=`) |
| `GET` | `/api/v1/fornecedores/{id}` | autenticado | `200` |
| `PUT` | `/api/v1/fornecedores/{id}` | **financeiro** | `200`; corpo com `versao`, **sem** `pais` e `documento` |
| `PATCH` | `/api/v1/fornecedores/{id}/ativar` / `desativar` | **financeiro** | `200`; corpo `{ "versao": n }` |
| `DELETE` | `/api/v1/fornecedores/{id}` | **administrador** | `204`; `400 registro em uso` quando contas a pagar, importações ou dívidas apontarem para ele |

Corpo do `POST` (fornecedor nacional):

```json
{
  "documento": "11.222.333/0001-81",
  "nome": "Cerâmica Paraíba LTDA",
  "nome_fantasia": "Cerâmica PB",
  "moeda_padrao": "BRL",
  "uf": "PB",
  "cidade": "João Pessoa",
  "contato_nome": "Maria",
  "email": "vendas@ceramicapb.com.br",
  "telefone": "83 3333-0000",
  "observacoes": ""
}
```

Fornecedor estrangeiro:

```json
{
  "pais": "CN",
  "identificacao_estrangeira": "91330200MA2XXXXX",
  "nome": "Foshan Ceramics Co., Ltd.",
  "moeda_padrao": "USD",
  "contato_nome": "Li Wei",
  "email": "sales@exemplo.cn"
}
```

| Campo | Regra |
| --- | --- |
| `pais` | código ISO de duas letras (`BR`, `CN`, `US`…); vazio = `BR`; só o formato é conferido; não muda depois de criado |
| `documento` | CPF ou CNPJ (numérico ou alfanumérico), com ou sem máscara; **obrigatório** para `BR` e **proibido** para outro país; único (`409 já existe um fornecedor com este documento`); não muda depois de criado |
| `identificacao_estrangeira` | tax id ou registro do exterior, até 40 caracteres, gravado em maiúsculas; só fora do Brasil; único por país (`409 já existe um fornecedor com esta identificação neste país`) |
| `nome` | obrigatório, até 200 |
| `moeda_padrao` | `BRL`, `USD`, `EUR` ou `CNY`; vazio = `BRL` |
| `uf` / `cidade` | opcionais, só para `BR` (`uf` sigla, `cidade` até 120) |
| `contato_nome` / `email` / `telefone` / `observacoes` | opcionais (160, um e-mail, 40 e 2000 caracteres) |

A resposta traz `documento` (sem máscara), `documento_formatado` e `tipo_pessoa` (`pf`/`pj`) —
os três `null` no estrangeiro —, os demais campos, `ativo`, `versao`, `criado_em` e
`atualizado_em`.

`?busca=` procura pelo nome sem diferenciar acento e maiúscula, pelo começo do documento sem
máscara e pelo começo da identificação estrangeira. `?pais=` e `?moeda=` inválidos → `400`.

### Versão (edição concorrente)

Todo fornecedor tem `versao`, que começa em `1` e avança a cada alteração gravada. `PUT`,
`ativar` e `desativar` exigem no corpo a `versao` que o cliente leu:

| Situação | Status | Mensagem |
| --- | --- | --- |
| `versao` ausente ou menor que 1 | `400` | `erro de validação: versão do fornecedor é obrigatória` |
| `versao` diferente da atual (outra pessoa gravou antes) | `409` | `o fornecedor foi alterado por outra pessoa; recarregue e tente de novo` |
| `pais` ou `documento` no corpo do `PUT` | `400` | `erro de validação: corpo da requisição inválido: json: unknown field ...` |

Ativar um fornecedor já ativo (ou desativar um já inativo) responde `200` sem mudar a versão. O
`PUT` substitui o cadastro inteiro: campo opcional omitido fica vazio e `moeda_padrao` omitida
volta a `BRL`.

---

## Contas bancárias

Contas de cada **empresa** do grupo, com escopo por empresa. Perfis: **financeiro** (ou
administrador) cria, lê, altera, ativa e desativa as contas das empresas do seu acesso;
**administrador** exclui. Quem não é financeiro não lê contas (`403`).

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/contas-bancarias` | **financeiro** | `201` |
| `GET` | `/api/v1/contas-bancarias` | **financeiro** | `200` lista paginada das empresas do acesso (`?empresa=`, `?banco=`, `?tipo=`, `?ativa=`) |
| `GET` | `/api/v1/contas-bancarias/{id}` | **financeiro** | `200` |
| `PUT` | `/api/v1/contas-bancarias/{id}` | **financeiro** | `200`; corpo `{ "apelido": "...", "versao": n }` |
| `PATCH` | `/api/v1/contas-bancarias/{id}/ativar` / `desativar` | **financeiro** | `200`; corpo `{ "versao": n }` |
| `DELETE` | `/api/v1/contas-bancarias/{id}` | **administrador** | `204`; `400 registro em uso` quando lançamentos apontarem para ela |
| `GET` | `/api/v1/bancos` | autenticado | `200` catálogo `[{ "codigo": "001", "nome": "Banco do Brasil" }, ...]` |

Corpo do `POST`:

```json
{
  "empresa_id": "…",
  "banco": "748",
  "tipo": "corrente",
  "agencia": "0812",
  "conta": "54321-0",
  "apelido": "Sicredi movimento"
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; empresa do acesso (`403 perfil sem acesso a esta empresa`) e ativa (`400 empresa inativa não recebe conta bancária`); não muda |
| `banco` | código COMPE do catálogo: `001` Banco do Brasil, `033` Santander, `077` Inter, `237` Bradesco, `341` Itaú, `403` Cora, `748` Sicredi; outro → `400 banco não suportado`; não muda |
| `tipo` | `corrente`, `poupanca`, `pagamento` ou `aplicacao`; não muda |
| `agencia` / `agencia_digito` | até 5 dígitos, não só zeros; dígito `0-9`, `X` ou `P`, junto (`1234-5`) **ou** no campo próprio, nunca nos dois; não muda |
| `conta` / `conta_digito` | até 20 dígitos, não só zeros; dígito com 1 ou 2 caracteres `0-9`/`X`/`P`, junto (`54321-0`) **ou** no campo próprio; pontos e espaços são ignorados; não muda |
| `apelido` | obrigatório, até 60; único na empresa sem diferenciar maiúsculas (`409 já existe uma conta com este apelido na empresa`); único campo editável |

O dígito verificador não é conferido. A mesma conta não existe duas vezes no grupo, nem em
empresas diferentes → `409 já existe uma conta com este banco, agência e número`. É a mesma conta
quando banco, `tipo` e agência coincidem (zeros à esquerda ignorados) e o número coincide sem o
dígito (`54321-0` × `54321`) ou com o dígito colado (`54321-0` × `543210`). Contas de tipos
diferentes no mesmo número (corrente e poupança do Banco do Brasil, por exemplo) são aceitas.
Corrigir banco, tipo, agência ou conta = desativar e criar outra.

A resposta traz `banco` e `banco_nome`, `tipo`, `agencia`, `agencia_digito` e
`agencia_formatada` (`1234-5`), `conta`, `conta_digito` e `conta_formatada`, `apelido`, `ativa`,
`versao`, `criado_em` e `atualizado_em`. Dígito ausente vem `null`.

Na listagem, usuário sem empresa no acesso recebe lista vazia; `?empresa=` fora do acesso → `403`;
`?banco=` ou `?tipo=` inválidos → `400`. Ordem: apelido, sem diferenciar maiúsculas.

Reativar conta de empresa inativa → `400 empresa inativa não recebe conta bancária`.

## Plano de contas

Categorias financeiras do **grupo** em dois níveis (raiz e subcategoria), usadas para classificar
contas a pagar, títulos e extratos. Perfis: **financeiro** (ou administrador) cria, altera, ativa e
desativa; **administrador** exclui; qualquer autenticado lê.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/categorias-financeiras` | **financeiro** | `201` |
| `GET` | `/api/v1/categorias-financeiras` | autenticado | `200` lista paginada (`?pai=`, `?raiz=true`, `?natureza=`, `?linha_dre=`, `?ativa=`, `?busca=`) |
| `GET` | `/api/v1/categorias-financeiras/arvore` | autenticado | `200` plano inteiro em dois níveis, **sem paginação**; `?ativas=true` só as ativas |
| `GET` | `/api/v1/categorias-financeiras/{id}` | autenticado | `200` |
| `PUT` | `/api/v1/categorias-financeiras/{id}` | **financeiro** | `200`; corpo `{ "nome", "codigo", "ordem", "versao" }` |
| `PATCH` | `/api/v1/categorias-financeiras/{id}/ativar` / `desativar` | **financeiro** | `200`; corpo `{ "versao": n }` |
| `DELETE` | `/api/v1/categorias-financeiras/{id}` | **administrador** | `204`; `400 registro em uso` com subcategorias ou lançamentos |

Corpo do `POST` (raiz e subcategoria):

```json
{ "nome": "Despesas administrativas", "natureza": "despesa", "linha_dre": "despesa_fixa", "codigo": "4.2", "ordem": 3 }
{ "nome": "Aluguel", "categoria_pai_id": "…", "codigo": "4.2.01" }
```

| Campo | Regra |
| --- | --- |
| `nome` | obrigatório, até 120; único entre as raízes e entre as filhas do mesmo pai, sem diferenciar maiúsculas (`409 já existe uma categoria com este nome neste nível`) |
| `categoria_pai_id` | opcional; só uma raiz ativa pode ser pai (`400 categoria pai deve ser uma categoria raiz ativa`); não muda |
| `natureza` / `linha_dre` | **obrigatórias na raiz** (`400 natureza e linha da DRE são obrigatórias na categoria raiz`) e **proibidas na subcategoria**, que herda as do pai (`400 subcategoria herda natureza e linha da DRE da categoria pai`); não mudam |
| `codigo` | opcional, números separados por ponto (`4.1.03`), até 20; único (`409 já existe uma categoria com este código`) |
| `ordem` | inteiro `>= 0`, default `0`; ordena a exibição |

Natureza e linha da DRE aceitas juntas:

| `linha_dre` | `natureza` | Uso |
| --- | --- | --- |
| `receita_operacional` | `receita` | receita que não vem de NF |
| `deducao_venda` | `despesa` | impostos sobre a venda, devoluções |
| `compra_mercadoria` | `despesa` | compra de estoque e importação; **fora do resultado** (o custo entra pelo estoque) |
| `despesa_variavel` | `despesa` | frete, comissões, taxas de venda |
| `despesa_fixa` | `despesa` | custos fixos |
| `resultado_financeiro` | `receita` ou `despesa` | juros, tarifas, rendimentos |
| `fora_dre` | `receita` ou `despesa` | empréstimo, aporte, distribuição de lucro |
| `entre_contas` | `transferencia` | transferência entre contas próprias |

Outra combinação → `400 linha da DRE <linha> não aceita natureza <natureza>`.

A resposta traz `categoria_pai_id` (`null` na raiz), `natureza`, `linha_dre`, `codigo`, `ordem`,
`ativa`, `lancavel`, `versao`, `criado_em` e `atualizado_em`. **`lancavel`** é `true` só em
subcategoria ativa: é onde contas a pagar, títulos e extratos podem ser classificados.

Regras de situação:

- desativar raiz com subcategoria ativa → `409 categoria possui subcategorias ativas e não pode ser desativada`;
- ativar subcategoria com o pai inativo → `400 categoria pai está inativa`.

`GET /arvore` devolve as raízes, cada uma com `subcategorias` (lista, vazia quando não houver),
na ordem `ordem`, nome (sem diferenciar maiúsculas). `?busca=` procura no nome (sem diferenciar
maiúsculas) e no começo do código. Listagem ordenada por `ordem`, nome (sem diferenciar
maiúsculas).

O `PUT` substitui os campos editáveis inteiros: `codigo` omitido vira `null` e `ordem` omitida
volta a `0`. Para mudar só o nome, reenvie o `codigo` e a `ordem` lidos.

### Versão nas contas bancárias e no plano de contas

Mesmo contrato dos fornecedores: toda resposta traz `versao`; `PUT`, `ativar` e `desativar`
exigem a `versao` lida.

| Situação | Status | Mensagem |
| --- | --- | --- |
| `versao` ausente, `null` ou menor que 1 | `400` | `erro de validação: versão da conta bancária é obrigatória` / `... da categoria financeira ...` |
| corpo vazio ou `versao` que não é inteiro (`"3"`, `1.5`) | `400` | `erro de validação: corpo da requisição inválido: ...` |
| `versao` diferente da atual | `409` | `a conta bancária foi alterada por outra pessoa; recarregue e tente de novo` / `a categoria financeira foi alterada ...` |
| campo que não muda no corpo do `PUT` (`banco`, `conta`, `natureza`…) | `400` | `erro de validação: corpo da requisição inválido: json: unknown field ...` |

Ativar o que já está ativo (ou desativar o inativo) responde `200` sem mudar a versão.

## Pedidos de venda e entregas

Pedidos de cada **empresa** vendedora, com escopo por empresa (padrão da seção 4 do
`EMPRESAS.md`). Perfis: **comercial** (ou administrador) cria, altera, confirma, reserva
pendências, cancela e opera entregas; **financeiro** (ou administrador) registra a entrada paga;
qualquer autenticado lê os pedidos das empresas do seu acesso. **Representante com login** (não
administrador) vê e altera só os pedidos em que é o representante, cria sempre em nome próprio e
só para clientes da própria carteira (`403 cliente de outro representante`), e cancela só a própria
proposta.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/pedidos` | **comercial** | `201` proposta |
| `GET` | `/api/v1/pedidos` | autenticado | `200` lista paginada sem itens (`?empresa=`, `?cliente=`, `?representante=`, `?status=`, `?modelo=`, `?numero=`) |
| `GET` | `/api/v1/pedidos/{id}` | autenticado | `200` detalhe com itens, reservado e falta cobrir por item e reservas ativas |
| `PUT` | `/api/v1/pedidos/{id}` | **comercial** | `200`; só em `proposta`; corpo do `POST` sem `empresa_id` e com `versao` |
| `POST` | `/api/v1/pedidos/{id}/confirmar` | **comercial** | `200`; corpo `{ "versao": n }`; proposta vencida → `409` |
| `POST` | `/api/v1/pedidos/{id}/entrada` | **financeiro** | `200`; corpo `{ "versao", "pago_em", "valor_centavos" }` |
| `POST` | `/api/v1/pedidos/{id}/reservar-pendencias` | **comercial** | `200`; corpo `{ "versao": n }` |
| `POST` | `/api/v1/pedidos/{id}/cancelar` | **comercial** | `200`; corpo `{ "versao", "motivo", "valor_a_devolver_centavos" }` |
| `GET` | `/api/v1/pedidos/{id}/historico` | autenticado | `200` lista (`{ "dados": [...] }`), mais antigo primeiro |
| `GET` | `/api/v1/pedidos/{id}/proposta` | autenticado | `200` documento da proposta, pronto para a página de impressão (seção "Documento da proposta") |
| `GET` | `/api/v1/modelos-venda` | autenticado | `200` lista com a condição padrão de cada modelo e o texto por extenso |
| `POST` | `/api/v1/pedidos/{id}/entregas` | **comercial** | `201` entrega `pendente` |
| `GET` | `/api/v1/pedidos/{id}/entregas` | autenticado | `200` lista (`{ "dados": [...] }`) |
| `GET` | `/api/v1/entregas` | autenticado | `200` lista paginada (`?empresa=`, `?status=`, `?previsao_ate=`) |
| `GET` | `/api/v1/entregas/{id}` | autenticado | `200` |
| `POST` | `/api/v1/entregas/{id}/saiu` | **comercial** | `200`; corpo `{ "versao", "saiu_em" }`; baixa o estoque |
| `POST` | `/api/v1/entregas/{id}/entregue` | **comercial** | `200`; corpo `{ "versao", "entregue_em" }` |
| `POST` | `/api/v1/entregas/{id}/cancelar` | **comercial** | `200`; corpo `{ "versao": n }`; só `pendente` |
| `GET` | `/api/v1/pedidos/pendencias` | autenticado | `200` lista paginada por item (`?empresa=`, `?cliente=`, `?produto=`) |
| `GET` | `/api/v1/pedidos/prazos` | autenticado | `200` lista paginada (`?empresa=`, `?situacao=vencido\|a_vencer`, `?dias=30`) |
| `GET` | `/api/v1/pedidos/cancelamentos` | autenticado | `200` lista paginada (`?empresa=`, `?com_devolucao=true`, `?de=`, `?ate=`) |

Corpo do `POST /pedidos`:

```json
{
  "empresa_id": "…",
  "cliente_id": "…",
  "obra_id": "…",
  "representante_id": "…",
  "modelo": "pronta_entrega",
  "entrada_bp": 0,
  "prazo_boleto_dias": 28,
  "validade": "2026-11-02",
  "frete_tipo": "cif",
  "frete_estimado_centavos": 18000,
  "prazo_limite": "2026-12-20",
  "numero_pedido_cliente": "OC-778",
  "observacoes": "",
  "itens": [{ "produto_id": "…", "quantidade": 10, "preco_unitario_centavos": 1500 }]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; empresa do acesso (`403 perfil sem acesso a esta empresa`) e ativa (`400 empresa inativa não recebe pedido novo`); não muda |
| `cliente_id` | obrigatório; cliente ativo ao criar ou trocar (`400 cliente inativo não recebe pedido novo`) |
| `obra_id` | opcional; obra do mesmo cliente e ativa ao escolher (`400 obra não encontrada para o cliente do pedido`) |
| `representante_id` | opcional; sem ele vale o representante do cliente; ativo ao atribuir; representante com login só em nome próprio (`403`) |
| `modelo` | `pronta_entrega` ou `importacao`; define a condição padrão gravada no pedido |
| `entrada_bp`, `prazo_boleto_dias`, `prazo_entrega_dias` | opcionais; sobrescrevem a condição padrão do modelo enquanto o pedido é proposta (o que não vier fica com o padrão). Entrada `0..10000` e prazo de entrega `1..730` **só na importação** (`400` na pronta entrega, que é paga no boleto); prazo do boleto `1..365`. Importação com entrada `0`: o prazo de entrega conta da confirmação (o `confirmar` grava o `prazo_limite`) e não há entrada a registrar (`400`) |
| `validade` | data `AAAA-MM-DD` da proposta, não anterior a hoje (o `PUT` aceita reenviar a validade atual, mesmo vencida); sem ela, **hoje + 15 dias** (também no `PUT`, que assim renova a proposta) |
| `frete_tipo`, `frete_estimado_centavos` | opcionais; frete **estimado** da proposta (`cif` exige valor). Informativo: não soma a `valor_total_centavos`; o frete real continua na entrega |
| `prazo_limite` | data `AAAA-MM-DD`, opcional, só na pronta entrega; na importação é calculado pela entrada paga (`400`) |
| `numero_pedido_cliente` | opcional, até 60 (número de compra do cliente) |
| `itens` | pelo menos um; produto ativo ao incluir e sem repetir no pedido; `quantidade > 0`; `preco_unitario_centavos >= 0` (o front sugere o preço de `GET /tabela-precos/vigentes`). A API grava em cada item o `preco_tabela_centavos` vigente hoje (tabela de preço do modelo ou, sem ela, `produto.preco`) |

Condição padrão de cada modelo, devolvida em `condicao` quando a proposta não a sobrescreve:

| `modelo` | `entrada_bp` | `prazo_boleto_dias` | `prazo_entrega_dias` |
| --- | --- | --- | --- |
| `pronta_entrega` | `0` | `30` | `null` |
| `importacao` | `3000` (30%) | `null` | `120`, contados da entrada paga |

A resposta traz `numero` (sequencial por empresa), `status`, `valor_total_centavos`, `condicao`,
`validade`, `vencida` (proposta com validade anterior a hoje; sempre `false` fora de `proposta`),
`revisao` (começa em 1 e sobe a cada `PUT` da proposta), `frete_tipo`, `frete_estimado_centavos`,
`prazo_limite`, `entrada_paga_em`, `entrada_paga_centavos`, `cancelado_motivo`,
`valor_a_devolver_centavos`, `versao` e `itens` (`preco_tabela_centavos`, `desconto_unitario_centavos`
= tabela − preço, negativo quando vendido acima da tabela, `subtotal_centavos`, `quantidade_entregue`
e `restante`). Proposta vencida não confirma (`409 proposta vencida em DD/MM/AAAA; renove a validade
antes de confirmar`): renove com um `PUT` que traga validade nova ou nenhuma. No `GET /pedidos/{id}` cada item traz também `reservado` e `falta_cobrir`, e o pedido
traz `reservas` (mesmo formato de `GET /produtos/{id}/reservas`, com `origem = pedido_item`).
Datas só-data (`prazo_limite`, `entrada_paga_em`, `previsao`, `saiu_em`, `entregue_em`) chegam como
meia-noite UTC e são aceitas como `AAAA-MM-DD`. "Hoje" (data padrão, data no futuro, prazos e
período de cancelamento) é o dia no fuso de Brasília (UTC−3).

### Documento da proposta

`GET /api/v1/pedidos/{id}/proposta` (mesmas regras de leitura do pedido: escopo por empresa e
representante com login só os próprios) junta tudo o que o documento impresso precisa; o front monta a
página e o navegador salva em PDF (decisão D8 do `PROPOSTAS.md`). Funciona em qualquer status, para
reimprimir o pedido confirmado.

```json
{
  "dados": {
    "pedido_id": "…",
    "numero": 57,
    "revisao": 2,
    "identificacao": "Proposta 57-R2",
    "status": "proposta",
    "modelo": "importacao",
    "emitido_em": "2026-10-19",
    "validade": "2026-11-03",
    "vencida": false,
    "empresa": { "id": "…", "razao_social": "MIA Materiais Ltda", "nome_fantasia": "MIA", "cnpj": "11.222.333/0001-81", "uf": "PB" },
    "cliente": { "id": "…", "nome": "Construtora Alfa", "documento": "11.444.777/0001-61", "cidade": "João Pessoa", "uf": "PB", "telefone": null, "email": "nf@alfa.com" },
    "obra": { "id": "…", "nome": "Residencial Alfa", "endereco": "Rua das Flores, 120 — Centro — João Pessoa/PB — CEP 58000-000" },
    "representante": { "id": "…", "nome": "Ana Representações", "email": null, "telefone": null },
    "itens": [
      {
        "item_id": "…", "produto_id": "…", "codigo": "POR-60", "nome": "Porcelanato 60x60", "unidade_medida": "m2",
        "quantidade": 10, "preco_unitario_centavos": 1500, "preco_tabela_centavos": 1800,
        "desconto_unitario_centavos": 300, "subtotal_centavos": 15000
      }
    ],
    "total_centavos": 15000,
    "desconto_centavos": 3000,
    "condicoes": [
      "Entrada de 30% na confirmação do pedido.",
      "Saldo de 70% na chegada da mercadoria.",
      "Prazo de entrega de 120 dias contados do pagamento da entrada.",
      "Frete CIF (por conta do vendedor), estimado em R$ 180,00.",
      "Proposta válida até 03/11/2026."
    ],
    "frete_tipo": "cif",
    "frete_estimado_centavos": 18000,
    "numero_pedido_cliente": "OC-778",
    "observacoes": ""
  }
}
```

- `identificacao`: número do pedido e revisão — `Proposta 57-R2` enquanto proposta, `Pedido 57-R2` depois
  de confirmado; `emitido_em`: hoje em Brasília. Empresa ou cliente do pedido ausentes → `404`.
- `desconto_centavos`: soma dos descontos positivos sobre a tabela (`(tabela − preço) × quantidade`);
  venda acima da tabela aparece no item com desconto negativo e não entra na soma.
- `condicoes`: as frases do documento, já montadas a partir da condição, do prazo-limite, do frete e da
  validade (a validade só aparece enquanto o pedido é proposta).

`GET /api/v1/modelos-venda` devolve, para o front montar o seletor de modelo e mostrar a condição padrão:

```json
{
  "dados": [
    { "modelo": "pronta_entrega", "condicao": { "entrada_bp": 0, "prazo_boleto_dias": 30, "prazo_entrega_dias": null },
      "condicoes": ["Pagamento em boleto com vencimento em 30 dias após a entrega."] },
    { "modelo": "importacao", "condicao": { "entrada_bp": 3000, "prazo_boleto_dias": null, "prazo_entrega_dias": 120 },
      "condicoes": ["Entrada de 30% na confirmação do pedido.", "Saldo de 70% na chegada da mercadoria.",
                    "Prazo de entrega de 120 dias contados do pagamento da entrada."] }
  ]
}
```

### Estados e efeitos

| Ação | De | Para | Efeito |
| --- | --- | --- | --- |
| confirmar | `proposta` | `confirmado` | reserva o disponível de cada item nos locais ativos da empresa, começando pelo local com mais disponível; o que faltar fica a cobrir |
| entrada | `confirmado`, `parcialmente_entregue` | igual | só importação, uma vez; `pago_em` não pode estar no futuro; valor entre 1 e o total; grava `prazo_limite` |
| reservar pendências | `confirmado`, `parcialmente_entregue` | igual | reserva o que ficou a cobrir; sem nada novo para reservar devolve o pedido sem gravar |
| saída de entrega | `confirmado`, `parcialmente_entregue` | `parcialmente_entregue` ou `entregue` | baixa o estoque e soma o entregue |
| cancelar | `proposta`, `confirmado`, `parcialmente_entregue` | `cancelado` | libera as reservas e cancela as entregas `pendente`; entrega `saiu` bloqueia (`409 há entrega em trânsito...`) |

Cancelamento: `motivo` obrigatório (até 1000). Com entrada paga, `valor_a_devolver_centavos` vale a
entrada quando omitido e fica entre `0` e a entrada; sem entrada paga, informar valor diferente de
zero → `400`. O valor a devolver só fica registrado: a conta a pagar sai depois, no M7.

### Entregas

Corpo do `POST /pedidos/{id}/entregas`:

```json
{
  "transportadora": "Rodonaves",
  "frete_tipo": "cif",
  "frete_centavos": 18000,
  "previsao": "2026-10-20",
  "itens": [{ "pedido_item_id": "…", "local_id": "…", "quantidade": 5 }]
}
```

| Campo | Regra |
| --- | --- |
| `frete_tipo` | `cif` ou `fob`; `frete_centavos` obrigatório no `cif`, opcional no `fob`, `>= 0` |
| `itens` | pelo menos um; item do pedido sem repetir; local **da empresa do pedido**; quantidade até o restante ainda não programado em outras entregas pendentes |
| `transportadora` | opcional, até 160 |

`saiu` (data padrão: hoje) baixa o estoque na mesma gravação, com **um movimento de saída por
item** (`itens[].movimento_id`): se o item tem uma única reserva no local com a quantidade exata, a
saída baixa essa reserva; senão, as reservas do item no local são liberadas, a saída sai do
disponível e o que sobrar das reservas é reservado de novo. Reservas do item em **outros locais**
são reduzidas (liberadas e, se for o caso, reservadas de novo com menos) para que o item nunca fique
com mais reservado que o que falta entregar. Sem disponível no local → `400 saldo disponível
insuficiente no local`. `entregue` (data padrão: hoje, não antes de `saiu_em`) só
registra a data.

### Versão (edição concorrente)

Pedido e entrega têm `versao` própria, com o mesmo contrato dos fornecedores (`400` sem versão,
`409 o pedido foi alterado por outra pessoa; recarregue e tente de novo` ou `409 a entrega foi
alterada por outra pessoa; ...`). **Toda operação de entrega também avança a `versao` do pedido**:
recarregue o pedido depois de criar, dar saída, concluir ou cancelar uma entrega. Mudança de
estoque concorrente (outra reserva ou movimento no mesmo produto) responde `409 estoque do produto
alterado por outra operação, recarregue e tente novamente`, e nada é gravado.

### Consultas

Todas respeitam o acesso por empresa (`?empresa=` fora do acesso → `403`) e, para o representante
com login, só os pedidos dele.

**`GET /pedidos/pendencias`** — uma linha por item ainda não entregue de pedido `confirmado` ou
`parcialmente_entregue`, ordenada pelo prazo-limite (sem prazo por último):

| Campo | Significado |
| --- | --- |
| `pedido_id`, `numero`, `empresa_id`, `cliente_id`, `representante_id`, `status`, `prazo_limite` | do pedido |
| `pedido_item_id`, `produto_id`, `produto_nome` | do item |
| `vendido` / `entregue` / `restante` | quantidade do item, já saída e o que falta entregar |
| `programado` | em entregas `pendente` |
| `reservado` | em reservas ativas do item |
| `falta_cobrir` | restante sem reserva |
| `disponivel_na_empresa` | disponível do produto nos locais ativos da empresa do pedido |
| `pode_cobrir_agora` | menor entre `falta_cobrir` e `disponivel_na_empresa`: o que `reservar-pendencias` reservaria agora |

**`GET /pedidos/prazos`** — pedidos `confirmado` ou `parcialmente_entregue` com `prazo_limite`,
ordenados pelo prazo. `situacao=vencido`: prazo antes de hoje; `situacao=a_vencer`: de hoje até
`hoje + dias`; sem `situacao`: vencidos e a vencer até `hoje + dias`. `dias` de `0` a `365`, padrão
`30`. Cada pedido traz `dias_para_o_prazo` (negativo quando vencido). Importação sem entrada paga
ainda não tem prazo e não aparece.

**`GET /pedidos/cancelamentos`** — pedidos `cancelado`, mais recentes primeiro. `com_devolucao=true`
traz só os com `valor_a_devolver_centavos > 0`. `de` e `ate` (`AAAA-MM-DD`, inclusivos) filtram pela
data do cancelamento no fuso de Brasília (`atualizado_em`, que não muda depois do cancelamento); `de`
depois de `ate` → `400`.

---

## Contas a pagar — resumo (financeiro)

`GET /api/v1/contas-a-pagar/resumo?de=AAAA-MM-DD&ate=AAAA-MM-DD&empresa=<uuid>`, financeiro ou
administrador, nas empresas do acesso (`empresa` fora do acesso → `403`). Sem `de`, começa na
segunda-feira da semana atual; sem `ate`, cobre 8 semanas. Intervalo máximo de 1 ano
(`400 erro de validação: intervalo do resumo deve ter no máximo 1 ano`); `ate` antes do início (o
`de` informado ou a segunda-feira atual) → `400 erro de validação: fim do resumo não pode ser
anterior ao início`. Sem perfil financeiro, o middleware responde `403 perfil sem permissão para
esta operação`.

Considera só `pendente_aprovacao` e `aprovado`, pelo `prazo_pagamento` e pelo `valor_centavos` da
solicitação. As três colunas não se sobrepõem:

| Coluna | Conta |
| --- | --- |
| `previsto` | `pendente_aprovacao` vencendo hoje ou depois |
| `a_pagar` | `aprovado` vencendo hoje ou depois |
| `vencido` | qualquer das duas com vencimento antes de hoje (UTC) |

`estimado_centavos` soma o valor das que têm `estimado = true`, em qualquer coluna.

```json
{
  "dados": {
    "de": "2026-10-12",
    "ate": "2026-12-06",
    "semanas": [
      {
        "semana": "2026-10-12",
        "empresa_id": "uuid",
        "previsto": { "quantidade": 2, "valor_centavos": 100000 },
        "a_pagar": { "quantidade": 1, "valor_centavos": 50000 },
        "vencido": { "quantidade": 1, "valor_centavos": 50000 },
        "estimado_centavos": 0
      }
    ],
    "atrasadas": [
      { "empresa_id": "uuid", "vencido": { "quantidade": 3, "valor_centavos": 90000 } }
    ],
    "totais": {
      "previsto": { "quantidade": 2, "valor_centavos": 100000 },
      "a_pagar": { "quantidade": 1, "valor_centavos": 50000 },
      "vencido": { "quantidade": 4, "valor_centavos": 140000 },
      "estimado_centavos": 0
    }
  }
}
```

- `semanas`: uma linha por semana (segunda-feira em `semana`) e empresa que tenha algo no período;
  semana sem conta não aparece.
- `atrasadas`: vencidas **antes** do `de`, por empresa, para o atraso acumulado não sumir do resumo.
- `totais`: soma das semanas; o `vencido` inclui as `atrasadas`.
- Só agregado: o financeiro continua sem ver o detalhe de `pendente_aprovacao`.

---

## Despesas recorrentes

Despesas fixas do mês (aluguel, energia, internet, impostos) que geram as solicitações de pagamento
de cada competência. Escrita e leitura pelo **financeiro** (ou administrador), com escopo por
empresa; exclusão só pelo **administrador**.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/despesas-recorrentes` | financeiro | `201` |
| `GET` | `/api/v1/despesas-recorrentes` | financeiro | `200` lista paginada (`?empresa=`, `?ativa=`, `?fornecedor=`) |
| `GET` | `/api/v1/despesas-recorrentes/{id}` | financeiro | `200` |
| `PUT` | `/api/v1/despesas-recorrentes/{id}` | financeiro | `200` |
| `PATCH` | `/api/v1/despesas-recorrentes/{id}/ativar` | financeiro | `200` |
| `PATCH` | `/api/v1/despesas-recorrentes/{id}/desativar` | financeiro | `200` |
| `DELETE` | `/api/v1/despesas-recorrentes/{id}` | administrador | `204` |
| `POST` | `/api/v1/despesas-recorrentes/gerar?competencia=AAAA-MM` | financeiro | `200` |

### `POST /api/v1/despesas-recorrentes`

```json
{
  "empresa_id": "uuid-da-empresa",
  "fornecedor_id": "uuid-do-fornecedor",
  "categoria_financeira_id": "uuid-da-subcategoria",
  "descricao": "Aluguel do galpão",
  "valor_centavos": 420000,
  "estimado": false,
  "forma_pagamento": "transferencia",
  "dia_vencimento": 10,
  "vencimento_mes_seguinte": false,
  "inicio": "2026-01",
  "fim": null
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; empresa ativa e no acesso do usuário. Não muda depois |
| `fornecedor_id` | opcional; fornecedor ativo |
| `categoria_financeira_id` | obrigatório; subcategoria ativa de natureza `despesa` |
| `descricao` | obrigatória, até 1000 caracteres; vira a `observacao` da solicitação gerada |
| `valor_centavos` | maior que zero |
| `estimado` | `true` para valor variável (energia, água); a solicitação nasce `estimado` |
| `forma_pagamento` | mesmas formas da solicitação |
| `dia_vencimento` | `1` a `31`; em mês mais curto vence no último dia |
| `vencimento_mes_seguinte` | `false`: vence no mês da competência; `true`: no mês seguinte |
| `inicio` / `fim` | competências `AAAA-MM`; `fim` opcional e não anterior ao `inicio` |

O `PUT` recebe os mesmos campos, **sem** `empresa_id` (campo desconhecido → `400`), mais `versao`;
substitui tudo (omitidos voltam ao padrão). `ativar`/`desativar` recebem `{ "versao": n }`; já na
situação pedida responde `200` sem avançar a versão. Reativar exige a empresa ativa. Editar a
recorrência não muda solicitações já geradas.

Objeto: os campos acima mais `id`, `ativa`, `versao`, `criado_em` e `atualizado_em` (`fim` `null`
quando não há).

| Erro | Status | Mensagem |
| --- | --- | --- |
| Sem perfil de financeiro | `403` | `perfil sem permissão para esta operação` (middleware das rotas `financeiro`) |
| Empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Empresa inativa | `400` | `erro de validação: empresa inativa não recebe despesa recorrente` |
| Sem categoria / sem início | `400` | `erro de validação: categoria financeira é obrigatória na despesa recorrente` / `...início da despesa recorrente é obrigatório` |
| Fim antes do início | `400` | `erro de validação: fim da despesa recorrente não pode ser anterior ao início` |
| Dia fora de 1 a 31 | `400` | `erro de validação: dia de vencimento deve estar entre 1 e 31` |
| `versao` divergente | `409` | `a despesa recorrente foi alterada por outra pessoa; recarregue e tente de novo` |
| Excluir com solicitação gerada | `400` | `erro de validação: despesa recorrente já gerou solicitações e não pode ser excluída; desative-a` |
| Excluir sem ser administrador | `403` | `perfil sem permissão para esta operação` (middleware das rotas `administrador`) |

### `POST /api/v1/despesas-recorrentes/gerar`

`?competencia=AAAA-MM` (obrigatória) e `?empresa=` (opcional, precisa estar no acesso). Para cada
recorrência ativa, vigente na competência (`inicio <= competência <= fim`) e de empresa ativa no
acesso do usuário, cria a solicitação daquela competência com `origem = | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`, em
`pendente_aprovacao`, com quem disparou como solicitante. É **idempotente**: rodar de novo a mesma
competência não duplica (índice único por recorrência e competência). O vencimento calculado pode
estar no passado (competência atrasada nasce vencida). Sem agendador: o front ou uma rotina externa
chama a rota. Uma competência já gerada conta em `ja_existentes` mesmo que a solicitação tenha sido
rejeitada ou cancelada depois: a geração não recria a conta (para recriar, lance manualmente).

```json
{
  "dados": {
    "competencia": "2026-11",
    "geradas": ["uuid-da-solicitacao"],
    "ja_existentes": 3,
    "recusadas": [
      { "despesa_recorrente_id": "uuid", "motivo": "fornecedor inativo não recebe conta a pagar" }
    ]
  }
}
```

`recusadas` lista as recorrências que não puderam gerar (fornecedor inativo, categoria que deixou
de ser lançável, recorrência desativada no meio da geração), sem impedir as outras.

---

## Contas a receber — títulos (financeiro)

Título a receber de cada **empresa**, com escopo por empresa (padrão da seção 4 do `EMPRESAS.md`).
Só **financeiro** e administrador leem e escrevem (`403 perfil sem permissão para contas a receber`
no usecase; o middleware das rotas `financeiro` responde antes para os demais perfis). O título nasce
`manual` por esta API ou é gerado por outro módulo (`origem` `pedido` e `nota_fiscal_duplicata`,
fases seguintes do M9 e do M8).

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/titulos-receber` | **financeiro** | `201` título manual em `carteira` (ou `em_banco` com `conta_bancaria_id`) |
| `GET` | `/api/v1/titulos-receber` | **financeiro** | `200` lista paginada por vencimento (`?empresa=`, `?cliente=`, `?status=`, `?origem=`, `?origem_id=`, `?conta=`, `?vencimento_de=`, `?vencimento_ate=`, `?vencidos=true`, `?origem_cancelada=true`) |
| `GET` | `/api/v1/titulos-receber/{id}` | **financeiro** | `200` |
| `GET` | `/api/v1/titulos-receber/{id}/historico` | **financeiro** | `200` lista (`{ "dados": [...] }`), mais antigo primeiro |
| `PUT` | `/api/v1/titulos-receber/{id}` | **financeiro** | `200`; só título `manual` em `carteira` sem recebimento; corpo do `POST` sem `empresa_id`, `cliente_id` e `conta_bancaria_id`, com `versao` |
| `POST` | `/api/v1/titulos-receber/{id}/prorrogar` | **financeiro** | `200`; corpo `{ "versao", "vencimento", "motivo" }`; `carteira` ou `em_banco` |
| `POST` | `/api/v1/titulos-receber/{id}/enviar-banco` | **financeiro** | `200`; corpo `{ "versao", "conta_bancaria_id" }`; `carteira` → `em_banco` |
| `POST` | `/api/v1/titulos-receber/{id}/retirar-banco` | **financeiro** | `200`; corpo `{ "versao": n }`; `em_banco` → `carteira`, limpa a conta de cobrança |
| `POST` | `/api/v1/titulos-receber/{id}/boleto-enviado` | **financeiro** | `200`; corpo `{ "versao", "canal", "enviado_em" }`; só `em_banco`; não troca o status |
| `POST` | `/api/v1/titulos-receber/{id}/cancelar` | **financeiro** | `200`; corpo `{ "versao", "motivo" }`; `carteira` ou `em_banco` sem recebimento |
| `POST` | `/api/v1/pedidos/{id}/titulo-entrada` | **financeiro** | `201` título de entrada gerado; `200` com o título que já existia; corpo `{ "vencimento", "conta_bancaria_id" }` |
| `POST` | `/api/v1/titulos-receber/{id}/recebimentos` | **financeiro** | `201` `{ "titulo", "recebimento" }`; baixa total ou parcial |
| `GET` | `/api/v1/titulos-receber/{id}/recebimentos` | **financeiro** | `200` lista (`{ "dados": [...] }`), com o estorno de cada um |
| `POST` | `/api/v1/titulos-receber/{id}/recebimentos/{recebimento_id}/estorno` | **financeiro** | `200` título; corpo `{ "versao", "motivo", "estornado_em" }` |

Corpo do `POST /titulos-receber`:

```json
{
  "empresa_id": "…",
  "cliente_id": "…",
  "numero": "ALG-10",
  "parcela": 1,
  "vencimento": "2026-11-10",
  "valor_centavos": 150000,
  "categoria_financeira_id": "…",
  "competencia": "2026-11",
  "conta_bancaria_id": "…",
  "observacao": "aluguel do galpão"
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; empresa do acesso (`403 perfil sem acesso a esta empresa`) e ativa (`400 empresa inativa não recebe título`); não muda |
| `cliente_id` | obrigatório; cliente ativo (`400 cliente inativo não recebe título novo`); não muda |
| `numero` / `parcela` | número obrigatório até 30; parcela de 1 a 999; repetidos na empresa e na origem (sem diferenciar maiúsculas, fora os cancelados) → `409 já existe um título com este número e parcela na empresa` |
| `vencimento` | `AAAA-MM-DD`, não anterior a hoje (Brasília); editar sem mudar o vencimento é aceito mesmo vencido |
| `valor_centavos` | `> 0` |
| `categoria_financeira_id` | obrigatória no título manual; subcategoria ativa (`400 categoria financeira deve ser uma subcategoria ativa`) de natureza `receita` (`400 categoria financeira do título deve ser de receita`) |
| `competencia` | `AAAA-MM`, opcional; omitida = mês do vencimento (DRE do título manual no M15) |
| `conta_bancaria_id` | opcional; conta de cobrança ativa e da mesma empresa (`400 conta bancária deve ser da mesma empresa do título`); com ela o título nasce `em_banco` |
| `observacao` | opcional, até 1000 |

Resposta:

```json
{
  "dados": {
    "id": "…",
    "empresa_id": "…",
    "cliente_id": "…",
    "origem": "manual",
    "origem_id": null,
    "numero": "ALG-10",
    "parcela": 1,
    "vencimento": "2026-11-10",
    "valor_centavos": 150000,
    "recebido_centavos": 0,
    "saldo_centavos": 150000,
    "vencido": false,
    "categoria_financeira_id": "…",
    "competencia": "2026-11",
    "conta_bancaria_id": null,
    "status": "carteira",
    "boleto_enviado_em": null,
    "boleto_canal": null,
    "origem_cancelada_em": null,
    "cancelado_motivo": null,
    "observacao": "aluguel do galpão",
    "usuario_id": "…",
    "versao": 1,
    "criado_em": "…",
    "atualizado_em": "…"
  }
}
```

- `status`: `carteira`, `em_banco`, `antecipado`, `recebido`, `cancelado`. **`vencido` é derivado**
  (`vencimento < hoje`, saldo maior que zero e status `carteira`, `em_banco` ou `antecipado`), nunca
  gravado: `?status=vencido` → `400`; use `?vencidos=true`.
- `vencimento` sai como `AAAA-MM-DD` e `competencia` como `AAAA-MM`; `boleto_enviado_em` é um
  instante (RFC 3339).
- `PUT` com os mesmos dados devolve o título sem avançar a versão nem gravar histórico.
- Prorrogar: novo vencimento posterior ao atual e não anterior a hoje (`400`); `motivo` obrigatório
  até 500; a competência não muda.
- Boleto: `canal` em `email`, `whatsapp`, `portal`, `correio`, `outro`; `enviado_em` em RFC 3339
  (`2026-10-15T09:30:00-03:00`), obrigatório e não no futuro. Registra o último envio.
- Título com a origem cancelada (`origem_cancelada_em` preenchido) não volta ao banco (`409`).
- Histórico (`evento`): `criado`, `gerado`, `editado`, `prorrogado`, `enviado_banco`,
  `retirado_banco`, `boleto_enviado`, `antecipado`, `antecipacao_estornada`, `| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |do`,
  `recebido`, `recebimento_estornado`, `cancelado`, `origem_cancelada`; cada item traz `de_status`,
  `para_status` e `descricao`.

### Título de entrada do pedido

`POST /api/v1/pedidos/{id}/titulo-entrada` gera o título da entrada de um pedido de **importação**
`confirmado` ou `parcialmente_entregue`, ainda sem entrada registrada: `origem = pedido`,
`origem_id` = id do pedido, `numero = PED-<numero do pedido>`, parcela 1, empresa e cliente do pedido,
valor = `valor_total_centavos × entrada_bp / 10000` (arredondado para baixo), sem categoria (a
receita vem da NF). `vencimento` (`AAAA-MM-DD`) é obrigatório e pode estar no passado;
`conta_bancaria_id` é opcional (com ela o título nasce `em_banco`). A geração é idempotente: se o
pedido já tem título de entrada, a resposta é `200` com ele, inclusive cancelado (o título de entrada
não é gerado de novo). Cancelar o pedido não cancela o título; o financeiro o cancela.

| Situação | Status | Mensagem |
| --- | --- | --- |
| Pedido de pronta entrega | `400` | `erro de validação: só o pedido de importação tem título de entrada` |
| Pedido em `proposta`, `entregue` ou `cancelado` | `409` | `o título de entrada só é gerado para pedido confirmado e não entregue` |
| Entrada já registrada no pedido | `409` | `a entrada do pedido já foi registrada` |
| Pedido de empresa fora do acesso | `403` | `perfil sem acesso a esta empresa` |
| Pedido cancelado durante a geração | `409` | `origem cancelada não gera título` |

### Recebimento (baixa) e estorno

Corpo do `POST /titulos-receber/{id}/recebimentos`:

```json
{
  "versao": 1,
  "tipo": "conta",
  "conta_bancaria_id": "…",
  "recebido_em": "2026-10-15",
  "valor_centavos": 95000,
  "juros_centavos": 800,
  "desconto_centavos": 5000
}
```

| Campo | Regra |
| --- | --- |
| `tipo` | `conta` (padrão, dinheiro numa conta da empresa) ou `fomento` (título antecipado liquidado pelo cliente direto no fomento, sem conta) |
| `conta_bancaria_id` | obrigatória em `conta`; conta ativa da mesma empresa, mesmo que diferente da conta de cobrança; proibida em `fomento` |
| `recebido_em` | `AAAA-MM-DD`, obrigatória, não no futuro (Brasília) |
| `valor_centavos` | `> 0`; abate o saldo e entrou no caixa |
| `juros_centavos` | `>= 0`; juros e multa recebidos, não abatem o saldo |
| `desconto_centavos` | `>= 0`; abate o saldo sem entrar no caixa |

- `valor + desconto` não pode passar do saldo (`400 valor mais desconto passa do saldo do título (<saldo> centavos)`).
  Quando o saldo zera, o título vira `recebido`; antes disso, o status não muda.
- Só título `carteira`, `em_banco` ou `antecipado` recebe baixa (`409`).
- **Pagamento na conta errada**: título `antecipado` recebido com `tipo = conta` exige
  `"repasse": { "vencimento", "categoria_financeira_id" }` no corpo (`400` sem ele). Na mesma
  transação da baixa, o M9 grava o repasse (valor = `valor_centavos`, sem juros) e gera a conta a pagar
  ao fomento no M7 (`origem = | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`, categoria lançável de `despesa`, competência do mês
  do recebimento). O recebimento traz `repasse` com o `solicitacao_id`; o que gerou repasse não é
  estornado (`409`). `repasse` em título que não está antecipado → `400`.
- O recebimento traz `entrada_no_caixa_centavos` (`valor + juros` em `conta`, `0` em `fomento`), que
  é o que o M13 concilia com o extrato pela `conta_bancaria_id` e pela `recebido_em`.
- **Título de entrada do pedido**: a baixa que zera o saldo registra a entrada paga no pedido (M6) na
  mesma transação, com `recebido_em` como data e a soma dos `valor_centavos` como valor; o pedido ganha
  `entrada_paga_em`, `entrada_paga_centavos` e o `prazo_limite`, e o histórico do pedido registra
  `entrada_registrada`. O recebimento sai com `registrou_entrada_pedido = true`. Se o pedido já tinha a
  entrada registrada pela rota do M6, ou foi cancelado, a baixa é aceita sem mexer no pedido e o
  histórico do título avisa. Pedido alterado por outra pessoa no meio → `409` e nada é gravado.
- Estorno: lançamento inverso (o recebimento fica `estornado = true` com o `estorno`); o saldo volta e o
  título `recebido` volta ao status anterior à baixa. `motivo` obrigatório (até 500); `estornado_em`
  não no futuro nem antes do recebimento. Recebimento já estornado → `409 o recebimento já foi
  estornado`; o que registrou a entrada do pedido não é estornado (`409`).

### Erros do título

| Situação | Status | Mensagem |
| --- | --- | --- |
| `versao` ausente | `400` | `erro de validação: versão do título é obrigatória` |
| `versao` divergente | `409` | `o título foi alterado por outra pessoa; recarregue e tente de novo` |
| Editar título gerado / fora de `carteira` / com recebimento | `409` | `só o título manual pode ser editado; ...` / `só o título em carteira pode ser editado` / `título com recebimento não pode ser editado` |
| Transição fora do estado | `409` | `não é possível mudar o título de "<atual>" para "<novo>"` |
| Boleto fora do banco | `409` | `só o título em banco tem boleto para enviar` |
| Cancelar com recebimento | `409` | `título com recebimento não pode ser cancelado; estorne o recebimento antes` |

---

## Antecipações (borderô, | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` | e repasse)

Antecipação de títulos com uma empresa de fomento, que é um **fornecedor** do M4. Só **financeiro** e
administrador, com escopo por empresa.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/antecipacoes` | **financeiro** | `201` borderô `ativa`; os títulos passam a `antecipado` |
| `GET` | `/api/v1/antecipacoes` | **financeiro** | `200` lista paginada, mais recente primeiro (`?empresa=`, `?fornecedor=`, `?status=ativa\|estornada`, `?de=`, `?ate=`) |
| `GET` | `/api/v1/antecipacoes/{id}` | **financeiro** | `200` com os títulos |
| `POST` | `/api/v1/antecipacoes/{id}/estorno` | **financeiro** | `200`; corpo `{ "versao", "motivo" }` |
| `POST` | `/api/v1/titulos-receber/{id}/| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |` | **financeiro** | `201` `{ "titulo", "recompra" }` |

Corpo do `POST /antecipacoes`:

```json
{
  "empresa_id": "…",
  "fornecedor_id": "…",
  "numero": "B-77",
  "operado_em": "2026-10-15",
  "taxa_bp": 300,
  "valor_liquido_centavos": 294000,
  "conta_bancaria_id": "…",
  "titulo_ids": ["…", "…"]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | empresa do acesso e ativa; todos os títulos são dela |
| `fornecedor_id` | o fomento; fornecedor ativo (`400 fomento inativo não recebe antecipação`) |
| `numero` | opcional, até 30 (número do borderô no fomento) |
| `operado_em` | `AAAA-MM-DD`, não no futuro |
| `taxa_bp` | taxa ao mês em pontos-base, informativa, `0..10000` |
| `valor_liquido_centavos` | creditado na conta; `> 0` e no máximo o bruto |
| `conta_bancaria_id` | conta ativa da empresa onde o líquido entrou (o M13 concilia por ela e por `operado_em`) |
| `titulo_ids` | de 1 a 200 títulos `carteira` ou `em_banco`, com saldo, sem origem cancelada, sem repetir |

- `valor_bruto_centavos` é a soma dos saldos dos títulos; `encargos_centavos` = bruto − líquido (o M15
  os lê como resultado financeiro). Cada título guarda o valor antecipado e o status anterior em
  `titulos[]`.
- Cliente com `proibe_antecipacao` (M3) bloqueia o borderô inteiro:
  `400 cliente não permite antecipação: <nome do cliente>`, conferido também na transação.
- Título já antecipado em outro borderô → `409`.
- **Estorno do borderô** (lançamento errado): só se nenhum título foi recebido nem | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |do (`409`);
  os títulos voltam ao status anterior e a antecipação fica `estornada` com `estorno_motivo` (até
  500) e `estornado_em`. `versao` divergente → `409 a antecipação foi alterada por outra pessoa; ...`.

### Recompra

Corpo do `POST /titulos-receber/{id}/| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`:

```json
{
  "versao": 2,
  "| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |do_em": "2026-10-15",
  "encargos_centavos": 4500,
  "vencimento": "2026-10-20",
  "categoria_financeira_id": "…",
  "motivo": "cliente não pagou"
}
```

Só título `antecipado` numa antecipação vigente (`409`). Na mesma transação: o título volta para
`carteira` (a empresa cobra o cliente de novo), sai do borderô, e o M7 ganha a conta a pagar ao
fomento com `origem = | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |`, valor = saldo do título + encargos, a `categoria_financeira_id`
informada (subcategoria ativa de `despesa`, senão `400`), vencimento informado e competência do mês da
| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |, em `pendente_aprovacao`. A resposta traz `recompra.solicitacao_id`. `motivo` obrigatório
(até 500); `| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` |do_em` não no futuro.

---

## Contas a receber — resumo e alerta de boleto (financeiro)

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/contas-a-receber/resumo` | **financeiro** | `200` (`?de=`, `?ate=`, `?empresa=`) |
| `GET` | `/api/v1/titulos-receber/boletos-pendentes` | **financeiro** | `200` lista paginada de títulos (`?dias=5`, `?empresa=`) |

**Resumo**: saldos (`valor − recebido`) dos títulos em aberto por **semana** (segunda a domingo, pelo
vencimento) e por **empresa**, em três colunas sem sobreposição: `a_receber` (`carteira` e `em_banco` a
vencer), `antecipado` (a vencer, só informativo: o dinheiro já entrou pelo borderô) e `vencido`
(as três situações, vencidas). Os vencidos antes do início do período saem à parte em `atrasados`, por
empresa, e entram no total de vencido. Período padrão de 8 semanas a partir da segunda-feira atual
(Brasília); intervalo máximo de 1 ano; fim antes do início → `400`. Só as empresas do acesso;
`?empresa=` fora do acesso → `403`.

```json
{
  "dados": {
    "de": "2026-10-12",
    "ate": "2026-12-06",
    "semanas": [
      {
        "semana": "2026-10-19",
        "empresa_id": "…",
        "a_receber": { "quantidade": 2, "saldo_centavos": 270000 },
        "antecipado": { "quantidade": 1, "saldo_centavos": 150000 },
        "vencido": { "quantidade": 0, "saldo_centavos": 0 }
      }
    ],
    "atrasados": [{ "empresa_id": "…", "vencido": { "quantidade": 1, "saldo_centavos": 150000 } }],
    "totais": {
      "a_receber": { "quantidade": 2, "saldo_centavos": 270000 },
      "antecipado": { "quantidade": 1, "saldo_centavos": 150000 },
      "vencido": { "quantidade": 2, "saldo_centavos": 300000 }
    }
  }
}
```

**Boletos pendentes**: títulos `em_banco` sem envio de boleto registrado e com vencimento até
`hoje + dias` (padrão `5`, de `0` a `90`, senão `400`), inclusive os já vencidos, ordenados por
vencimento. Registrar o envio (`POST /titulos-receber/{id}/boleto-enviado`) tira o título da lista.

---

## Notas fiscais (NF-e por XML)

O sistema **não emite** NF-e: importa o XML autorizado (`nfeProc`) das notas emitidas pelas empresas
do grupo. Escopo por empresa emitente (padrão da seção 4 do `EMPRESAS.md`). Perfis: **financeiro**
(ou administrador) importa notas e eventos, vincula pedido e entrega e liga item a produto; qualquer
autenticado lê as notas das empresas do seu acesso. **Representante com login** (não administrador)
vê só notas de clientes da própria carteira (nota entre empresas não aparece) e não consulta itens
sem produto (`403`).

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/notas-fiscais/importacao` | **financeiro** | `201` nota com itens, duplicatas e `avisos` |
| `GET` | `/api/v1/notas-fiscais` | autenticado | `200` lista paginada sem itens (`?empresa=`, `?cliente=`, `?pedido=`, `?natureza=`, `?status=`, `?entre_empresas=`, `?sem_pedido=true`, `?emitida_de=`, `?emitida_ate=`, `?numero=`, `?chave=`) |
| `GET` | `/api/v1/notas-fiscais/{id}` | autenticado | `200` detalhe com itens e duplicatas |
| `GET` | `/api/v1/notas-fiscais/{id}/xml` | autenticado | `200` XML original (`attachment`, `nosniff`) |
| `PUT` | `/api/v1/notas-fiscais/{id}/vinculo` | **financeiro** | `200`; corpo `{ "versao", "pedido_id", "entrega_id" }` (`null` desfaz) |
| `PUT` | `/api/v1/notas-fiscais/{id}/itens/{numero}/produto` | **financeiro** | `200`; corpo `{ "versao", "produto_id" }` |
| `GET` | `/api/v1/notas-fiscais/itens-sem-produto` | autenticado | `200` lista paginada (`?empresa=`) |
| `POST` | `/api/v1/notas-fiscais/eventos` | **financeiro** | `201` `{ "evento": {...}, "nota_fiscal": {...} }` |
| `GET` | `/api/v1/notas-fiscais/{id}/eventos` | autenticado | `200` lista (`{ "dados": [...] }`), mais antigo primeiro |
| `GET` | `/api/v1/entregas/sem-nota-fiscal` | autenticado | `200` lista paginada (`?empresa=`, `?cliente=`) |
| `POST` | `/api/v1/notas-fiscais/{id}/titulos` | **financeiro** | `200` sincroniza os títulos a receber da nota (sem corpo) |

### Importação

1. Envie o XML por `POST /api/v1/arquivos` (`application/xml` ou `text/xml`); o arquivo fica
   guardado e vira o XML original da nota.
2. Chame a importação com o `arquivo_id`:

```json
{ "arquivo_id": "…", "pedido_id": "…", "entrega_id": "…" }
```

`pedido_id` e `entrega_id` são opcionais (`entrega_id` exige `pedido_id`).

| Regra | Resposta |
| --- | --- |
| arquivo de outro usuário | `403 o arquivo XML precisa ter sido enviado por você` |
| arquivo que não é XML ou de uma solicitação | `400` |
| arquivo já importado em outra nota ou evento | `409` |
| XML sem protocolo (`NFe` sem `nfeProc`), malformado, NFC-e (modelo 65), homologação (`tpAmb = 2`), não autorizada (`cStat` diferente de `100`/`150`) | `400` |
| chave inválida (44 caracteres, dígito módulo 11, CNPJ alfanumérico aceito) ou incoerente com a nota (UF, ano e mês, CNPJ, série, número) ou com o protocolo | `400` |
| emitente que não é empresa cadastrada / empresa inativa | `400` |
| emitente fora do acesso | `403 perfil sem acesso a esta empresa` |
| chave já importada | `409 já existe uma nota fiscal com esta chave de acesso` |
| destinatário estrangeiro (`idEstrangeiro`) | `400` |
| nota de entrada (`tpNF = 0`) que não é devolução (`finNFe` diferente de `4`) | `400` |

A assinatura digital do XML não é conferida; a autenticidade vem da chave e do protocolo da SEFAZ.

**Destinatário**: CNPJ de empresa cadastrada → nota **entre empresas do grupo** (`entre_empresas =
true`, `natureza = transferencia`, `empresa_destinataria_id`, sem cliente). Regra provisória até a
contabilidade responder: sem título, conta a pagar nem movimento de estoque. Senão o cliente é achado
pelo documento e, se não existir, **criado** com nome, UF e cidade do XML (`tipo` `pf` para CPF e
`outro` para CNPJ, sem representante; e-mail inválido é ignorado). Cliente novo sem UF ou cidade no
XML → `400`.

**Natureza**: `transferencia` (entre empresas) › `devolucao` (`finNFe = 4`) › `remessa` (CFOP
predominante de remessa: `5116/5117/6116/6117`, `5905/6905`, `5910`–`5917`, `6910`–`6917`,
`5923/6923`, `5934/6934`, `5949/6949`) › `venda` (inclusive o simples faturamento `5922/6922`). O CFOP
predominante é o do item de maior valor. Só `venda` gera título no contas a receber (seção
"Títulos a receber da nota"); as duplicatas são guardadas em todas as naturezas.

**Itens**: o produto é resolvido pelo `codigo` (`cProd`); item sem produto entra com `produto_id =
null` e aparece em `itens-sem-produto` enquanto a nota estiver autorizada. A NF **não move estoque**
(a saída é a da entrega).

`avisos` da importação:

| Aviso | Quando |
| --- | --- |
| `itens_sem_produto` | algum item não achou produto pelo código |
| `numero_pedido_ausente` | o cliente exige número de pedido na NF e o XML não traz `xPed` |
| `cliente_criado` | o cliente foi criado a partir do destinatário |
| `titulos_pendentes` | a nota foi gravada, mas a geração dos títulos falhou; refaça com `POST /notas-fiscais/{id}/titulos` |

Resposta (resumo):

```json
{
  "dados": {
    "id": "…", "chave": "2526…", "empresa_id": "…", "numero": 123, "serie": 1,
    "emitida_em": "2026-10-10T12:30:00Z", "tipo_operacao": "saida", "finalidade": "normal",
    "natureza": "venda", "natureza_operacao": "VENDA DE MERCADORIA", "entre_empresas": false,
    "cliente_id": "…", "empresa_destinataria_id": null, "destinatario_documento": "11444777000161",
    "destinatario_nome": "…", "cfop_predominante": "5102", "valor_total_centavos": 109000,
    "valor_produtos_centavos": 109000, "valor_frete_centavos": 0, "valor_desconto_centavos": 0,
    "numero_pedido_cliente": "PC-77", "protocolo": "325260000012345",
    "autorizada_em": "2026-10-10T12:31:02Z", "status": "autorizada", "pedido_id": null,
    "entrega_id": null, "arquivo_id": "…", "usuario_id": "…", "versao": 1,
    "itens": [{
      "numero_item": 1, "codigo": "PISO-01", "descricao": "…", "produto_id": "…", "ncm": "69072100",
      "cfop": "5102", "unidade": "M2", "quantidade": "10.5000", "quantidade_decimilesimos": 105000,
      "valor_unitario": "100.0000000000", "valor_centavos": 105000, "desconto_centavos": 0,
      "pedido_cliente": null
    }],
    "duplicatas": [{ "id": "…", "numero": "001", "vencimento": "2026-11-09T00:00:00Z", "valor_centavos": 54500 }],
    "avisos": ["itens_sem_produto"],
    "titulos_gerados": ["…", "…"]
  }
}
```

`quantidade` vem como texto com 4 casas (o XML tem até 4); `valor_unitario` é o `vUnCom` como veio.
`vencimento` é só-data (meia-noite UTC). Filtros `emitida_de`/`emitida_ate` (`AAAA-MM-DD`,
inclusivos) usam o dia no fuso de Brasília; `de` depois de `ate` → `400`.

### Vínculo com pedido e entrega

Pedido da mesma empresa e do mesmo cliente da nota, fora de `proposta` e `cancelado`; entrega do
pedido e não cancelada (`400`). Uma entrega tem no máximo uma NF autorizada (`409 a entrega já tem
uma nota fiscal autorizada`). Nota entre empresas não se vincula a pedido (`400`); nota cancelada não
muda de vínculo (`409`). A situação do pedido e da entrega é conferida de novo na gravação (`409` se
mudou). `versao` segue o contrato dos fornecedores (`400` sem versão, `409 a nota fiscal foi alterada
por outra pessoa; ...`). Ligar item a produto (`PUT .../itens/{numero}/produto`) usa a mesma versão;
produto inexistente → `404`.

### Eventos (cancelamento e carta de correção)

Envie o XML do evento (`procEventoNFe`) por `POST /api/v1/arquivos` e importe com
`{ "arquivo_id": "…" }`. Aceita `110111` (cancelamento) e `110110` (carta de correção), com retorno
`cStat` `135`, `136` ou `155` e ambiente de produção. A nota é achada pela chave (`404` se ainda não
foi importada) e o autor do evento precisa ser o CNPJ emitente (`400`).

- **Cancelamento**: nota `autorizada` → `cancelada` (avança a `versao`); a entrega volta a aparecer
  em `entregas/sem-nota-fiscal` e os títulos das duplicatas são cancelados no contas a receber
  (seção abaixo). Cancelar de novo → `409`. A resposta traz também `avisos`,
  `titulos_cancelados` e `titulos_marcados`.
- **Carta de correção**: só registra o texto (`justificativa`); mesma sequência repetida → `409`;
  nota cancelada não recebe (`409`).

`GET /entregas/sem-nota-fiscal` lista entregas `saiu` ou `entregue` sem NF autorizada, mais antigas
primeiro, com `entrega_id`, `pedido_id`, `pedido_numero`, `empresa_id`, `cliente_id`, `status`,
`saiu_em`, `entregue_em` e `transportadora`; o representante com login vê só as dos próprios pedidos.

### Títulos a receber da nota

Nota de `venda`, autorizada, com cliente e com duplicatas gera **um título por duplicata** no contas a
receber (`origem = nota_fiscal_duplicata`, `origem_id` = id da duplicata), na própria importação:
número `NF <série>-<número>`, parcela pela ordem de vencimento, valor e vencimento da duplicata,
situação `carteira`, sem conta de cobrança. Os títulos aparecem em
`GET /api/v1/titulos-receber?origem=nota_fiscal_duplicata&origem_id=<duplicata>` e seguem as regras
do contas a receber (empresa, cliente, número, valor e competência vêm da nota e não são editáveis).
Remessa, devolução e nota entre empresas do grupo não geram título.

Nota e títulos são gravados em transações separadas: se a geração falhar, a nota fica gravada, a
importação responde `201` com o aviso `titulos_pendentes` e a causa vai para o log da requisição.

O **cancelamento** da nota (evento `110111`) cancela os títulos ainda em `carteira` ou `em_banco` sem
recebimento; título antecipado, recebido ou com recebimento parcial não muda de situação e ganha
`origem_cancelada_em` para o financeiro resolver. O cancelamento da nota nunca é recusado por causa dos
títulos; se a parte dos títulos falhar, o evento responde `201` com o aviso `titulos_pendentes`.

`POST /api/v1/notas-fiscais/{id}/titulos` (financeiro, sem corpo) refaz o que faltou e pode ser
chamado quantas vezes for preciso: nota autorizada de venda gera os títulos que ainda não existem;
nota cancelada cancela os títulos das duplicatas; outra nota → `400`.

```json
{ "dados": { "gerados": [], "ja_existentes": ["…", "…"], "cancelados": [], "marcados": [], "ignorados": 0 } }
```

---

## Comissões

Usuário vinculado a um representante (`representante.usuario_id`, fora o administrador) vê só as
próprias comissões, ajustes, fechamentos e resumo, mesmo que o cargo seja financeiro, e não apura,
ajusta nem fecha.

Comissão do representante por **entrega** que saiu (`saiu` ou `entregue`), proporcional ao que saiu:
base = soma de `quantidade × preco_unitario_centavos` dos itens da entrega (sem frete), valor = base ×
percentual / 10000, arredondado meio centavo para cima. Pedido sem representante não gera comissão.
Item bonificado (preço zero) entra com subtotal zero; entrega só de brindes gera comissão com
`base_centavos` e `valor_centavos` zero, para não voltar à apuração.
Escopo por empresa (padrão da seção 4 do `EMPRESAS.md`); escopo e decisões em `COMISSOES.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/pedidos/{id}/| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` | comercial, financeiro ou o representante do pedido | `200` percentual vigente; sem definição, o padrão do representante com `definido: false` |
| `PUT` | `/api/v1/pedidos/{id}/| `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` | **comercial** (representante com login não define) | `200`; corpo `{ "percentual_bp", "versao" }`; só antes da primeira saída de entrega |
| `POST` | `/api/v1/comissoes/apurar` | **financeiro** | `200` resultado; corpo opcional `{ "empresa_id", "competencia" }` |
| `GET` | `/api/v1/comissoes/pendencias` | **financeiro** | `200` lista paginada das entregas que saíram e ainda não foram apuradas (`?empresa=`, `?competencia=`) |
| `GET` | `/api/v1/comissoes` | **financeiro** ou representante com login (só as próprias) | `200` lista paginada (`?empresa=`, `?representante=`, `?pedido=`, `?competencia=AAAA-MM`, `?aberta=`), mais recente primeiro |
| `POST` | `/api/v1/comissoes/ajustes` | **financeiro** | `201` ajuste |
| `GET` | `/api/v1/comissoes/ajustes` | **financeiro** ou representante com login (só os próprios) | `200` lista paginada (`?empresa=`, `?representante=`, `?pedido=`, `?competencia=AAAA-MM`, `?aberto=`), mais recente primeiro |
| `POST` | `/api/v1/comissoes/fechamentos` | **financeiro** | `200` resultado; corpo `{ "empresa_id", "competencia", "categoria_financeira_id", "representante_id" }` |
| `GET` | `/api/v1/comissoes/fechamentos` | **financeiro** ou representante com login (só os próprios) | `200` lista paginada (`?empresa=`, `?representante=`, `?competencia=`), mais recente primeiro |
| `GET` | `/api/v1/comissoes/fechamentos/{id}` | **financeiro** ou o representante do fechamento | `200` fechamento com `comissoes` e `ajustes` |
| `GET` | `/api/v1/comissoes/resumo` | **financeiro** ou representante com login (só o próprio) | `200` resumo da competência por empresa e representante (`?competencia=AAAA-MM` obrigatório, `?empresa=`, `?representante=`) |

### Percentual do pedido

O percentual padrão vem do representante pelo modelo do pedido (`comissao_pronta_entrega_bp` ou
`comissao_importacao_bp`). O comercial pode sobrescrevê-lo enquanto nenhuma entrega do pedido saiu; a
primeira definição dispensa `versao`, as seguintes exigem a `versao` lida (`400` sem ela, `409` se outra
pessoa gravou antes). Sem definição manual, a primeira apuração grava o padrão daquele momento:
mudar o padrão do representante depois não muda pedido já apurado. Definição feita para outro
representante (o representante do pedido mudou na proposta) deixa de valer e é substituída.

```json
{
  "dados": {
    "pedido_id": "…",
    "representante_id": "…",
    "percentual_bp": 450,
    "origem": "manual",
    "definido": true,
    "versao": 2,
    "usuario_id": "…",
    "atualizado_em": "2026-10-18T21:10:00Z"
  }
}
```

### Apuração

Grava uma comissão por entrega, idempotente (`uq_comissao_entrega`): rodar de novo só apura o que
falta. Sem `competencia`, apura todas as saídas pendentes do acesso do usuário; com ela, as saídas até o
fim do mês informado. Cada chamada apura até 500 entregas (`restam_entregas: true` quando sobrou).
Entrega que não apura entra em `falhas` com o motivo e o `representante_id`, não conta no limite e as
demais seguem.

```json
{ "dados": { "apuradas": ["…"], "ja_existentes": 0, "falhas": [], "restam_entregas": false } }
```

Item da lista de comissões:

```json
{
  "id": "…", "empresa_id": "…", "representante_id": "…", "pedido_id": "…", "entrega_id": "…",
  "competencia": "2026-10", "base_centavos": 6000, "percentual_bp": 450, "valor_centavos": 270,
  "usuario_id": "…", "criado_em": "2026-10-18T21:15:00Z"
}
```

### Ajustes

Lançamento manual, positivo ou negativo, para o que a apuração não cobre: devolução de mercadoria,
desconto concedido depois da saída, prêmio, correção de erro. É append-only: não há edição nem
exclusão, e desfazer é lançar o ajuste inverso. Entra no primeiro fechamento que cobrir a competência
informada: o da própria competência ou, se ela já foi fechada para o representante, o do mês seguinte
(o resumo o mostra `em_aberto` até lá).

```json
{
  "empresa_id": "…",
  "representante_id": "…",
  "pedido_id": "…",
  "competencia": "2026-10",
  "valor_centavos": -375,
  "motivo": "devolução de uma peça"
}
```

- `valor_centavos` diferente de zero; `motivo` obrigatório, até 1000 caracteres;
- `competencia` no formato `AAAA-MM`, nunca posterior ao mês corrente em Brasília;
- `pedido_id` opcional; quando vier, o pedido precisa ser da mesma empresa e do mesmo representante;
- o representante pode estar inativo (a comissão continua devida).

### Fechamento

Fecha o mês de uma empresa: um fechamento por representante, com as comissões e os ajustes **em
aberto** de competência até a informada (o que ficou de meses anteriores entra junto), e uma conta a
pagar no M7 gerada na mesma transação.

- `competencia` precisa ser de mês já encerrado em Brasília (`400` no mês corrente);
- `categoria_financeira_id`: subcategoria ativa de `despesa` (sugestão: debaixo de `despesa_variavel`);
- antes de somar, a rota apura as saídas pendentes da empresa até a competência (`apuracao` na
  resposta); se sobrarem mais de 500 entregas a apurar, responde `409` e pede para apurar antes;
- representante com entrega que não apurou (aparece em `apuracao.falhas`), com total ≤ 0, sem
  `fornecedor_id` ou que já tenha fechamento na competência vai para `falhas` com o motivo, e os demais
  seguem; com `representante_id`, o motivo volta como erro (`400` sem saldo ou com total ≤ 0, `409` já
  fechado ou com entrega sem apurar);
- a conta a pagar nasce `pendente_aprovacao`, `origem = comissao`, `origem_id` = fechamento, em nome do
  fornecedor do representante, com a categoria e a competência do fechamento e vencimento no **dia 15
  do mês seguinte** à competência; segue o fluxo normal do M7;
- fechamento não é desfeito: correção depois vira ajuste num mês seguinte. `status_conta_a_pagar` mostra
  em que pé está a conta a pagar (`pago` quando paga).

```json
{
  "dados": {
    "competencia": "2026-09",
    "apuracao": { "apuradas": ["…"], "ja_existentes": 0, "falhas": [], "restam_entregas": false },
    "fechados": [
      {
        "id": "…", "empresa_id": "…", "representante_id": "…", "competencia": "2026-09",
        "comissoes_centavos": 180, "ajustes_centavos": -100, "total_centavos": 80, "vencimento": "2026-10-15",
        "categoria_financeira_id": "…", "solicitacao_id": "…", "status_conta_a_pagar": "pendente_aprovacao",
        "usuario_id": "…", "criado_em": "2026-10-02T12:00:00Z"
      }
    ],
    "falhas": [{ "representante_id": "…", "motivo": "erro de validação: representante sem fornecedor; …" }]
  }
}
```

Comissões e ajustes ganham `fechamento_id` (nulo enquanto abertos) e os filtros `?aberta=` / `?aberto=`.

### Resumo

Uma linha por empresa e representante com movimento na competência (comissões e ajustes **daquela**
competência, pela data da saída ou pela competência do ajuste), ordenada pelo nome do representante:

```json
{
  "dados": {
    "competencia": "2026-09",
    "total_centavos": 80,
    "linhas": [
      {
        "empresa_id": "…", "representante_id": "…", "representante_nome": "Ana Representações",
        "comissoes_centavos": 180, "ajustes_centavos": -100, "total_centavos": 80,
        "em_aberto_centavos": 0, "fechado_centavos": 80, "pago_centavos": 0
      }
    ]
  }
}
```

- `em_aberto_centavos`: o que ainda não entrou em fechamento;
- `fechado_centavos`: o que já está num fechamento (conta a pagar gerada);
- `pago_centavos`: a parte fechada cuja conta a pagar está `pago` no M7.

### Erros das comissões

| Situação | Status | Mensagem |
| --- | --- | --- |
| Pedido sem representante | `400` | `erro de validação: pedido sem representante não tem comissão` |
| Pedido cancelado | `409` | `pedido cancelado não muda a comissão` |
| Depois da primeira saída | `409` | `a comissão do pedido não muda depois da primeira saída de entrega` |
| Versão divergente | `409` | `a comissão do pedido foi alterada por outra pessoa; recarregue e tente de novo` |
| Representante com login no `PUT` | `403` | `representante não define a própria comissão` |
| Comissão de outro representante | `403` | `comissão de outro representante` |
| Comercial sem financeiro na lista | `403` | `perfil sem permissão para ver comissões` |
| Usuário vinculado a representante apurando, ajustando ou fechando | `403` | `representante não apura, ajusta nem fecha comissões` |
| Fechamento de representante com entrega sem apurar | `409` | `há entregas do representante que não foram apuradas na competência; veja as falhas da apuração e feche de novo` |
| Ajuste com valor zero | `400` | `erro de validação: valor do ajuste não pode ser zero` |
| Ajuste em competência futura | `400` | `erro de validação: competência do ajuste não pode ser posterior ao mês corrente` |
| Pedido do ajuste de outra empresa ou representante | `400` | `erro de validação: pedido do ajuste deve ser da mesma empresa e do mesmo representante` |
| Representante do ajuste inexistente | `400` | `erro de validação: representante do ajuste não encontrado` |
| Fechamento do mês corrente | `400` | `erro de validação: só é possível fechar competência de mês já encerrado` |
| Fechamento de representante sem saldo | `400` | `erro de validação: nada a pagar ao representante na competência` |
| Mais de 500 entregas a apurar | `409` | `há mais entregas a apurar na competência; apure e feche de novo` |
| Fechamento inexistente | `404` | `registro não encontrado: fechamento não encontrado` |

---

## Tabela de preços

Preço de cada produto por **modelo de venda** (`pronta_entrega` ou `importacao`), do grupo inteiro (sem
empresa), com **vigência por data de início**: o preço de um dia é o lançamento de maior `vigente_de` até
aquele dia; um lançamento novo encerra o anterior. Mudar preço é lançar uma nova vigência; o histórico não
é editado. A tabela só **sugere** o preço do pedido; a vitrine pública continua com `produto.preco`.
Escopo e decisões em `PROPOSTAS.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/tabela-precos` | **administrador** | `201` lançamento |
| `GET` | `/api/v1/tabela-precos` | comercial, financeiro ou administrador | `200` histórico paginado (`?produto=`, `?modelo=`), vigência mais recente primeiro |
| `GET` | `/api/v1/tabela-precos/vigentes` | comercial, financeiro ou administrador | `200` preço do dia por produto ativo (`?modelo=` obrigatório, `?data=AAAA-MM-DD` padrão hoje em Brasília, `?produto=`, `?busca=` nome ou código), ordem por nome |
| `DELETE` | `/api/v1/tabela-precos/{id}` | **administrador** | `204`; só lançamento cuja vigência ainda não começou |

```json
{
  "produto_id": "…",
  "modelo": "pronta_entrega",
  "preco_centavos": 12000,
  "vigente_de": "2026-11-01",
  "observacao": "reajuste do fornecedor"
}
```

- `preco_centavos > 0`; `vigente_de` obrigatório (data no passado é aceita, para corrigir); `observacao`
  até 500 caracteres; produto precisa existir e estar ativo.
- Item de `vigentes`: `preco_centavos` e `origem` — `tabela` (com `item_id` e `vigente_de`), `produto`
  (sem tabela, cai em `produto.preco`) ou `sem_preco` (`preco_centavos` nulo).

```json
{
  "dados": [
    {
      "produto_id": "…", "produto_nome": "Porcelanato 60x60", "produto_codigo": "POR-60", "unidade_medida": "m2",
      "modelo": "pronta_entrega", "data": "2026-10-18", "preco_centavos": 12000, "origem": "tabela",
      "item_id": "…", "vigente_de": "2026-10-01"
    }
  ],
  "pagina": 1,
  "tamanho": 20
}
```

| Situação | Status | Mensagem |
| --- | --- | --- |
| Perfil sem comercial, financeiro ou administrador | `403` | `perfil sem permissão para ver a tabela de preços` |
| Mesmo produto, modelo e início | `409` | `já existe preço deste produto e modelo com início nesta data` |
| Produto inativo | `400` | `erro de validação: produto inativo não recebe preço novo` |
| Exclusão de vigência que já começou | `409` | `só é possível excluir preço cuja vigência ainda não começou` |

---

## Caixa — saldo inicial, extrato, conciliação, fechamento e projeção (financeiro)

Extrato bancário importado por conta, em **OFX** (1.x SGML ou 2.x XML), sobre um **saldo inicial** por
conta. O extrato é a fonte do saldo: saldo da conta = saldo inicial + linhas do extrato depois da data de
corte. Escopo por empresa pela conta bancária (padrão da seção 4 do `EMPRESAS.md`); tudo é do
**financeiro** (ou administrador), e só o administrador desfaz importação. Escopo e decisões em `CAIXA.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `PUT` | `/api/v1/caixa/contas/{id}/abertura` | **financeiro** | `200` saldo inicial criado ou alterado |
| `GET` | `/api/v1/caixa/contas/{id}/abertura` | **financeiro** | `200` saldo inicial da conta |
| `GET` | `/api/v1/caixa/saldos` | **financeiro** | `200` lista (`{ "dados": [...] }`) por conta das empresas do acesso (`?empresa=`) |
| `POST` | `/api/v1/extratos/importacao` | **financeiro** | `201` relatório da importação (`200` na simulação) |
| `GET` | `/api/v1/extratos/importacoes` | **financeiro** | `200` lista paginada (`?empresa=`, `?conta=`), mais recente primeiro |
| `DELETE` | `/api/v1/extratos/importacoes/{id}` | **administrador** | `204`; só a importação mais recente da conta |
| `GET` | `/api/v1/extratos/lancamentos` | **financeiro** | `200` lista paginada (`?conta=` obrigatório, `?de=`, `?ate=` em `AAAA-MM-DD`, `?situacao=pendente`, `conciliado` ou `invalidado`), por data |
| `GET` | `/api/v1/extratos/lancamentos/{id}/sugestoes` | **financeiro** | `200` lista de movimentos do sistema que podem ser o par do lançamento |
| `POST` | `/api/v1/extratos/lancamentos/{id}/conciliacao` | **financeiro** | `201` conciliação manual |
| `POST` | `/api/v1/extratos/conciliar-automatico` | **financeiro** | `200` resultado da conciliação automática |
| `GET` | `/api/v1/conciliacoes/{id}` | **financeiro** | `200` conciliação com os movimentos |
| `POST` | `/api/v1/conciliacoes/{id}/desfazer` | **financeiro** | `200` conciliação desfeita |
| `GET` | `/api/v1/caixa/pendencias-sistema` | **financeiro** | `200` lista paginada dos movimentos do sistema sem par no extrato (`?conta=` obrigatório, `?de=`, `?ate=`) |
| `GET` | `/api/v1/caixa/pagamentos-sem-conta` | **financeiro** | `200` lista paginada dos pagamentos sem conta bancária (`?empresa=`, `?de=`, `?ate=`) |
| `POST` | `/api/v1/extratos/lancamentos/{id}/classificacao` | **financeiro** | `201` lançamento de caixa e a conciliação |
| `POST` | `/api/v1/extratos/lancamentos/{id}/transferencia` | **financeiro** | `201` transferência entre contas e a conciliação |
| `GET` | `/api/v1/caixa/lancamentos` | **financeiro** | `200` lista paginada dos lançamentos de caixa (`?empresa=`, `?conta=`, `?de=`, `?ate=`) |
| `GET` | `/api/v1/caixa/transferencias` | **financeiro** | `200` lista paginada das transferências entre contas (`?empresa=`, `?conta=`, `?de=`, `?ate=`) |
| `GET` | `/api/v1/caixa/fechamentos/previa` | **financeiro** | `200` prévia do fechamento (`?conta=` e `?competencia=AAAA-MM` obrigatórios) |
| `POST` | `/api/v1/caixa/fechamentos` | **financeiro** | `201` fechamento do mês da conta |
| `GET` | `/api/v1/caixa/fechamentos` | **financeiro** | `200` lista paginada (`?empresa=`, `?conta=`, `?competencia=AAAA-MM`), mais recente primeiro |
| `GET` | `/api/v1/caixa/fechamentos/{id}` | **financeiro** | `200` fechamento com as divergências |
| `POST` | `/api/v1/caixa/fechamentos/{id}/reabrir` | **administrador** | `200` fechamento reaberto |
| `GET` | `/api/v1/caixa/projecao` | **financeiro** | `200` saldo diário projetado de hoje até `?ate=` (`?empresa=`; sem ela, todas as empresas do acesso) |
| `GET` | `/api/v1/caixa/movimento-diario` | **financeiro** | `200` lista (`{ "dados": [...] }`) de entradas, saídas e saldo por conta e dia (`?empresa=`, `?conta=`, `?de=`, `?ate=`) |

### Saldo inicial

```json
{ "data_corte": "2026-09-30", "saldo_centavos": 1500000, "versao": 1 }
```

- `data_corte` é o dia cujo **fim** tem o saldo informado; precisa ser anterior a hoje em Brasília. O
  extrato vale do dia seguinte em diante: linha com data até o corte é ignorada na importação e fica fora
  do saldo. `saldo_centavos` é obrigatório e aceita negativo (cheque especial).
- Sem saldo inicial, o `PUT` cria (sem `versao`). Com saldo inicial, o `PUT` exige a `versao` lida
  (`409 a conta já tem saldo inicial; envie a versão para alterar` sem ela; `409 o saldo inicial da conta
  foi alterado por outra pessoa; recarregue e tente de novo` divergente).
- Conta inativa ou de empresa inativa → `400`; conta fora do acesso → `403`; `GET` sem saldo inicial →
  `404 registro não encontrado: conta sem saldo inicial`.
- Resposta do `PUT` e do `GET`:

```json
{
  "dados": {
    "conta_bancaria_id": "…", "data_corte": "2026-09-30", "saldo_centavos": 1500000, "usuario_id": "…",
    "versao": 1, "criado_em": "2026-10-19T13:00:00Z", "atualizado_em": "2026-10-19T13:00:00Z"
  }
}
```

### Importação do extrato

1. Envie o arquivo por `POST /api/v1/arquivos` (gravado como `application/x-ofx`).
2. Chame a importação:

```json
{ "arquivo_id": "…", "conta_bancaria_id": "…", "simular": false }
```

- Sem `conta_bancaria_id`, a conta é achada pelo `BANKID` (código COMPE), `BRANCHID` e `ACCTID` do OFX,
  entre as contas **ativas no acesso do usuário**. O número igual (sem zeros à esquerda, com ou sem o
  dígito depois do hífen) vale mais que o dígito colado ao número ou a agência colada na frente, e dígito
  diferente do cadastrado não corresponde. Com `conta_bancaria_id`, o OFX precisa ser dessa conta e
  nenhuma outra conta do banco pode corresponder melhor a ele.
- Um arquivo com vários blocos da mesma conta é aceito (período = união dos blocos; saldo do banco = o
  `LEDGERBAL` mais recente); contas diferentes no mesmo arquivo → `400`.
- Data com fuso (`DTPOSTED` com hora e `[-3:BRT]`, `[0:GMT]`…) vira o dia em Brasília; sem fuso, vale o dia
  escrito.
- `simular: true` devolve o relatório sem gravar (`200`, `importacao_id` nulo).
- Cada linha tem um identificador (`FITID` + data + valor + ocorrência no arquivo; sem `FITID`, a
  descrição entra no lugar): linha já importada por outro arquivo conta como `repetidas` e não é gravada
  de novo. Linha com valor zero, data até o corte ou data futura vai em `ignoradas` com o `motivo`
  (`valor_zero`, `anterior_ao_saldo_inicial`, `mes_fechado`, `data_futura`).
- Arquivo em Windows-1252 é convertido para UTF-8; descrição vem do `MEMO` (senão `NAME`), documento do
  `CHECKNUM` (senão `REFNUM`). O saldo do `LEDGERBAL` fica guardado como saldo do banco.

| Regra | Resposta |
| --- | --- |
| arquivo de outro usuário | `403 o arquivo OFX precisa ter sido enviado por você` |
| arquivo que não é OFX ou de uma solicitação | `400` |
| arquivo já importado | `409 arquivo já importado em outro extrato` |
| OFX inválido, de cartão (`CCACCTFROM`), com mais de uma conta, data ou valor ilegível, mais de 5000 lançamentos ou acima de 5MB | `400` |
| OFX sem conta e sem `conta_bancaria_id` | `400 erro de validação: o extrato não identifica a conta; informe conta_bancaria_id` |
| nenhuma ou mais de uma conta cadastrada correspondem | `400` (informe `conta_bancaria_id`) |
| `conta_bancaria_id` de outra conta | `400 erro de validação: o extrato é de outra conta (banco …, agência …, conta …)` |
| conta fora do acesso | `403 perfil sem acesso a esta empresa` |
| conta sem saldo inicial | `400 erro de validação: conta sem saldo inicial; informe a abertura antes de importar o extrato` |
| conta ou empresa inativa | `400` |
| saldo inicial alterado durante a importação | `409` |

Resposta:

```json
{
  "dados": {
    "simulacao": false, "importacao_id": "…", "conta_bancaria_id": "…", "arquivo_id": "…",
    "periodo_inicio": "2026-10-01", "periodo_fim": "2026-10-15",
    "saldo_banco_centavos": 1084567, "saldo_banco_em": "2026-10-15",
    "total": 3, "novas": 1, "repetidas": 1,
    "ignoradas": [
      { "sequencia": 1, "data": "2026-09-30", "valor_centavos": -100, "descricao": "TARIFA", "motivo": "anterior_ao_saldo_inicial" }
    ]
  }
}
```

Item de `GET /extratos/importacoes`:

```json
{
  "id": "…", "conta_bancaria_id": "…", "arquivo_id": "…", "periodo_inicio": "2026-10-01", "periodo_fim": "2026-10-15",
  "saldo_banco_centavos": 1084567, "saldo_banco_em": "2026-10-15", "linhas_novas": 1, "linhas_repetidas": 1,
  "linhas_ignoradas": 1, "usuario_id": "…", "criado_em": "2026-10-19T13:00:00Z"
}
```

**Desfazer** (`DELETE /extratos/importacoes/{id}`) apaga a importação e as linhas que ela trouxe; só vale
para a mais recente da conta (`409 só a importação mais recente da conta pode ser desfeita`). O arquivo
continua guardado e pode ser importado de novo; enquanto estiver numa importação, ele não pode ser
excluído (`400 registro em uso`).

Lançamento (`GET /extratos/lancamentos`): `id`, `conta_bancaria_id`, `importacao_id`, `data`,
`valor_centavos` (crédito positivo, débito negativo), `tipo` (`credito` ou `debito`), `descricao`,
`documento`, `situacao` (`pendente`, `conciliado` ou `invalidado`) e `conciliacao_id` (a conciliação
vigente, ou `null`). `ate` antes de `de` → `400`.

Importação com lançamento que já passou por conciliação, mesmo desfeita, não pode ser desfeita
(`409 importação com lançamento que já passou por conciliação não pode ser desfeita`).

### Conciliação

Liga um lançamento do extrato aos movimentos que o sistema já registrou na **mesma conta**:

| Movimento (`tipo`) | Origem | Valor no caixa | Data |
| --- | --- | --- | --- |
| `pagamento` | pagamento da conta a pagar (M7), inclusive comissão, | `origem=manual|recorrencia|recompra|repasse_fomento|comissao|importacao_parcela|importacao_custo` | e repasse | `- valor` | dia do `pago_em` em Brasília |
| `transferencia_saida` / `transferencia_entrada` | a outra ponta de uma transferência entre contas (abaixo) | `- valor` / `valor` | data da transferência |
| `recebimento` | baixa do título a receber do tipo `conta` (M9) | `valor + juros` | `recebido_em` |
| `antecipacao` | crédito do borderô (M9) | `valor_liquido_centavos` | `operado_em` |

Movimento estornado, de data até o corte do saldo inicial ou já conciliado não entra. Movimento
(`GET .../sugestoes`, `pendencias-sistema`, `pagamentos-sem-conta`):

```json
{
  "tipo": "pagamento", "id": "…", "conta_bancaria_id": "…", "empresa_id": "…", "data": "2026-10-15",
  "valor_centavos": -50000, "descricao": "Cimentos do Vale - Compra de cimento", "origem_id": "…",
  "estornado": false, "conciliacao_id": null
}
```

`origem_id` é a solicitação (pagamento), o título (recebimento) ou o próprio borderô (antecipação).

**Sugestões**: movimentos pendentes com o mesmo valor do lançamento e data até 3 dias antes ou depois,
o mais próximo primeiro.

**Manual** (`POST /extratos/lancamentos/{id}/conciliacao`): um lançamento com um ou mais movimentos
(até 50), cuja soma precisa ser exatamente o valor do lançamento:

```json
{ "movimentos": [{ "tipo": "pagamento", "id": "…" }, { "tipo": "pagamento", "id": "…" }] }
```

| Regra | Resposta |
| --- | --- |
| lista vazia, tipo desconhecido, movimento repetido | `400` |
| movimento que não é da conta ou é de data até o corte | `400 erro de validação: movimento não encontrado nesta conta ou anterior ao saldo inicial` |
| crédito com movimento de saída ou débito com movimento de entrada | `400` |
| movimento a mais de 3 dias do lançamento | `400 erro de validação: movimento fora da janela de 3 dias do lançamento` |
| soma diferente | `400 erro de validação: a soma dos movimentos (-499,00) é diferente do valor do lançamento (-500,00)` |
| lançamento já conciliado | `409 lançamento do extrato já conciliado; desfaça a conciliação antes` |
| movimento já conciliado em outro lançamento | `409 movimento já conciliado com outro lançamento do extrato` |
| movimento estornado ou alterado durante a gravação | `409 um dos movimentos foi estornado ou alterado; recarregue e tente de novo` |

**Automática** (`POST /extratos/conciliar-automatico`): `{ "conta_bancaria_id", "de", "ate" }`
(obrigatórios, até 92 dias). Concilia só os pares exatos e únicos nos dois sentidos: mesmo dia e mesmo
valor, com um só movimento para o lançamento e um só lançamento para o movimento. Processa até 500
lançamentos por chamada, sem dividir um dia: os dias que não couberem ficam para a próxima chamada
(`restam_linhas`); mais de 500 pendentes num mesmo dia ou mais de 5000 movimentos no período → `400`:

```json
{ "dados": { "conciliadas": 12, "ambiguos": 1, "sem_par": 3, "falhas": 0, "restam_linhas": false } }
```

`falhas` conta os pares que outra pessoa conciliou no meio da rotina; rodar de novo é seguro.

**Conciliação** (`GET /conciliacoes/{id}`, resposta da manual e do desfazer):

```json
{
  "dados": {
    "id": "…", "extrato_lancamento_id": "…", "conta_bancaria_id": "…", "modo": "manual",
    "vigente": true, "invalidada": false,
    "movimentos": [{ "tipo": "pagamento", "id": "…", "valor_centavos": -50000, "estornado": false }],
    "usuario_id": "…", "criado_em": "2026-10-19T13:00:00Z",
    "desfeita_em": null, "desfeita_por": null, "motivo_desfazer": null
  }
}
```

**Estorno depois de conciliar**: o M7 e o M9 continuam estornando normalmente. A conciliação fica
`invalidada` e o lançamento aparece como `invalidado` até alguém desfazer a conciliação.

**Desfazer** (`POST /conciliacoes/{id}/desfazer`): `{ "motivo": "…" }` (obrigatório, até 500
caracteres). A conciliação fica guardada com `vigente = false`; o lançamento e os movimentos voltam a
pendentes. Desfazer de novo → `409 conciliação já desfeita`.

**Pagamentos sem conta** (`GET /caixa/pagamentos-sem-conta`): pagamentos não estornados registrados
antes do M7, sem `conta_bancaria_id`. Ficam fora da conciliação e só aparecem nesta lista.

### Linhas sem par: classificação

Tarifa, IOF, juros e rendimento não têm movimento no M7 nem no M9: a linha é classificada no plano de
contas e vira um **lançamento de caixa**, já conciliado a ela (`modo = classificacao`).

```json
{ "categoria_financeira_id": "…", "descricao": "Tarifa do pacote" }
```

- Categoria lançável (subcategoria ativa): `despesa` em débito, `receita` em crédito; `transferencia` → `400`
  (use a transferência). A natureza é conferida de novo na transação.
- `descricao` é opcional (padrão: a da linha). `competencia` é o mês da data da linha, fixo.
- Resposta `201`: `{ "lancamento_caixa": {...}, "conciliacao": {...} }`, com o lançamento:

```json
{
  "id": "…", "empresa_id": "…", "conta_bancaria_id": "…", "data": "2026-10-05", "valor_centavos": -1290,
  "categoria_financeira_id": "…", "competencia": "2026-10", "descricao": "Tarifa do pacote",
  "estornado": false, "usuario_id": "…", "criado_em": "2026-10-19T13:00:00Z"
}
```

- Desfazer a conciliação (`POST /conciliacoes/{id}/desfazer`) estorna o lançamento (`estornado = true`) e
  a linha volta a pendente. O M15 lê os lançamentos não estornados por `competencia` e `linha_dre`.

### Transferência entre contas

A linha de uma das pontas cria a transferência, já conciliada a ela (`modo = transferencia`):

```json
{ "conta_contraparte_id": "…", "categoria_financeira_id": "…", "descricao": "Aplicação" }
```

- Débito → origem é a conta da linha; crédito → destino é a conta da linha. `valor_centavos` da
  transferência é sempre positivo.
- Categoria lançável de natureza `transferencia` (linha `entre_contas`), fora da DRE. Conta da outra ponta
  ativa, diferente da linha e no acesso do usuário (`403` sem acesso). Contas de empresas diferentes →
  `entre_empresas = true` (regra provisória: sem título nem conta a pagar entre elas).
- A outra ponta aparece em `pendencias-sistema` e nas sugestões da conta dela como `transferencia_entrada`
  (no destino, valor positivo) ou `transferencia_saida` (na origem, valor negativo), e é conciliada pela
  conciliação manual ou automática.
- Desfazer a conciliação que criou a transferência a estorna (`estornado = true`); se a outra ponta já
  estava conciliada, a conciliação dela fica `invalidada`. Desfazer exige acesso às duas contas (`403`) e
  mês aberto nas duas (`409`).
- Criar a transferência também recusa mês fechado na conta da outra ponta (`409`).
- Resposta `201`: `{ "transferencia": {...}, "conciliacao": {...} }`, com a transferência:

```json
{
  "id": "…", "conta_origem_id": "…", "conta_destino_id": "…", "data": "2026-10-05", "valor_centavos": 100000,
  "entre_empresas": false, "categoria_financeira_id": "…", "descricao": "Aplicação", "estornado": false,
  "usuario_id": "…", "criado_em": "2026-10-19T13:00:00Z"
}
```

`GET /caixa/lancamentos` e `GET /caixa/transferencias` listam os registros das contas no acesso (a
transferência aparece para quem vê a origem ou o destino), incluindo os estornados.

### Fechamento mensal

Um por conta e mês (`competencia` em `AAAA-MM`), em sequência: o primeiro é o mês do dia seguinte ao
corte do saldo inicial, e cada um depois é o mês seguinte ao último fechamento vigente. Só fecha mês
encerrado em Brasília.

**Prévia** (`GET /caixa/fechamentos/previa?conta=&competencia=`), também para o mês corrente:

```json
{
  "dados": {
    "conta_bancaria_id": "…", "competencia": "2026-09", "saldo_inicial_centavos": 100000,
    "entradas_centavos": 10000, "saidas_centavos": 1290, "saldo_final_centavos": 108710,
    "saldo_banco_centavos": 108710, "lancamentos_pendentes": 0, "conciliacoes_invalidadas": 0,
    "mes_encerrado": true, "pode_fechar": true, "movimentos_sem_linha": []
  }
}
```

`saldo_inicial_centavos` é o saldo inicial da conta (primeiro mês) ou o saldo final do fechamento anterior.
`saldo_banco_centavos` é o `LEDGERBAL` da importação mais recente com data no último dia do mês (`null` se
não houver). `movimentos_sem_linha` são os pagamentos, recebimentos, borderôs e pontas de transferência do
mês sem linha no extrato: aparecem como divergência, mas não impedem o fechamento. `pode_fechar` é `true`
só com o mês encerrado, sem pendência nem conciliação invalidada e com o `LEDGERBAL` do último dia igual ao
saldo final; sem esse `LEDGERBAL`, ainda é possível fechar informando `saldo_banco_centavos` no corpo.

**Fechar** (`POST /caixa/fechamentos`):

```json
{ "conta_bancaria_id": "…", "competencia": "2026-09", "saldo_banco_centavos": 108710 }
```

| Regra | Resposta |
| --- | --- |
| competência antes do saldo inicial, mês não encerrado, formato inválido | `400` |
| conta sem saldo inicial | `400 erro de validação: conta sem saldo inicial; informe a abertura antes de importar o extrato` |
| mês já fechado / mês anterior ainda aberto | `409 mês já fechado nesta conta` / `409 feche antes o mês de MM/AAAA` |
| linha do mês pendente ou conciliação invalidada | `409 o mês tem N lançamento(s) pendente(s) e M conciliação(ões) invalidada(s); …` |
| sem `saldo_banco_centavos` e sem `LEDGERBAL` do último dia | `400 erro de validação: informe o saldo do banco no fim do mês; …` |
| saldo calculado diferente do saldo do banco | `409 saldo calculado (1087,10) diferente do saldo do banco (1087,60); diferença de 0,50` |
| extrato, conciliações, saldo do banco ou saldo inicial mudaram entre a leitura e a gravação | `409` (recarregue a prévia) |

Resposta `201` e detalhe (`GET /caixa/fechamentos/{id}`): os campos do fechamento (`id`,
`conta_bancaria_id`, `competencia`, `saldo_inicial_centavos`, `entradas_centavos`, `saidas_centavos`,
`saldo_final_centavos`, `saldo_banco_centavos`, `vigente`, `reaberto_em`, `reaberto_por`,
`motivo_reabertura`, `usuario_id`, `criado_em`) mais as divergências recalculadas na leitura:
`conciliacoes_invalidadas` (estorno feito no M7 ou no M9 depois do fechamento) e `movimentos_sem_linha`.

**Mês fechado** trava o caixa daquela conta naquele mês: conciliar, classificar, transferir ou desfazer
conciliação de linha do mês → `409 mês fechado nesta conta; reabra o fechamento para alterar`; a
importação ignora as linhas do mês (motivo `mes_fechado`); o saldo inicial não muda enquanto houver
fechamento vigente (`409 a conta já tem mês fechado; reabra os fechamentos para mudar o saldo inicial`).
O M7 e o M9 não são bloqueados: o que mudar lá aparece como divergência no detalhe.

**Reabrir** (`POST /caixa/fechamentos/{id}/reabrir`, administrador): `{ "motivo": "…" }`. Só o último mês
fechado da conta (`409 reabra antes os meses seguintes desta conta`); reabrir de novo → `409 fechamento já
reaberto`. O registro fica com `vigente = false` e o mês volta a aceitar alterações.

### Projeção diária

`GET /caixa/projecao?empresa=&ate=` projeta o saldo dia a dia, de **hoje** (Brasília) até `ate` (padrão
hoje + 30; máximo hoje + 90; antes de hoje → `400`). Sem `empresa`, soma todas as empresas do acesso.

- **Saldo de partida**: contas ativas com saldo inicial (abertura + linhas do extrato depois do corte) mais
  o **em trânsito**: pagamentos, recebimentos, borderôs e pontas de transferência não conciliados e não
  estornados com data depois do fim do último extrato importado da conta. Conta sem saldo inicial fica fora
  (`contas_sem_saldo_inicial`).
- **Entradas**: saldo dos títulos `carteira` e `em_banco` pelo vencimento (`antecipado` fica fora: o
  dinheiro já entrou pelo borderô).
- **Saídas**: contas a pagar `pendente_aprovacao` e `aprovado` pelo `prazo_pagamento`; as `estimado` vão em
  `saidas_estimadas_centavos`.
- **Recorrentes**: despesas recorrentes vigentes cuja solicitação da competência ainda não foi gerada,
  pelo dia de vencimento.
- Entradas e saídas vencidas (antes de hoje) entram no primeiro dia, em colunas próprias.

```json
{
  "dados": {
    "de": "2026-10-19", "ate": "2026-11-18",
    "saldo_de_partida": {
      "extrato_centavos": 101500, "em_transito_centavos": -50000, "total_centavos": 51500,
      "contas_consideradas": 2, "contas_sem_saldo_inicial": 0
    },
    "saldo_final_centavos": 150000,
    "dias": [{
      "data": "2026-10-19", "entradas_centavos": 0, "entradas_vencidas_centavos": 20000,
      "saidas_centavos": 5000, "saidas_estimadas_centavos": 0, "saidas_vencidas_centavos": 0,
      "recorrentes_centavos": 0, "saldo_centavos": 66500
    }]
  }
}
```

### Movimento diário realizado

`GET /caixa/movimento-diario?empresa=&conta=&de=&ate=` substitui o "Fluxo de caixa diário": um item por
conta e dia com linha no extrato (depois do corte), com o saldo acumulado da conta no fim do dia. Padrão:
do dia 1 do mês até hoje; no máximo 92 dias (`400`).

```json
{ "dados": [{ "data": "2026-10-14", "conta_bancaria_id": "…", "entradas_centavos": 2000, "saidas_centavos": 500, "saldo_centavos": 101500 }] }
```

### Saldos por conta

```json
{
  "dados": [
    {
      "conta_bancaria_id": "…", "empresa_id": "…", "apelido": "Sicredi", "banco": "748", "banco_nome": "Sicredi",
      "ativa": true, "data_corte": "2026-09-30", "saldo_inicial_centavos": 100000, "saldo_atual_centavos": 193710,
      "ultimo_lancamento_em": "2026-10-14", "saldo_banco_centavos": 194950, "saldo_banco_em": "2026-10-12",
      "diferenca_banco_centavos": -50, "lancamentos_pendentes": 4
    }
  ]
}
```

Conta sem saldo inicial aparece com `data_corte`, `saldo_inicial_centavos` e `saldo_atual_centavos` nulos.
`lancamentos_pendentes` conta as linhas depois do corte ainda sem conciliação vigente.
`diferenca_banco_centavos` = saldo do banco (último `LEDGERBAL` importado) − saldo calculado até a data
dele; nulo sem saldo do banco ou com saldo do banco anterior ao corte.

---

## Importações (financeiro escreve; comercial e financeiro leem)

Compra de mercadoria de um fornecedor (M4) por uma empresa importadora, do pedido ao fornecedor até o
desembaraço. Uma importação é **um embarque** (uma DI/DUIMP). Escopo por empresa pela importadora (padrão
da seção 4 do `EMPRESAS.md`). Escopo e decisões em `IMPORTACOES.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/importacoes` | **financeiro** | `201` importação com itens |
| `GET` | `/api/v1/importacoes` | comercial ou financeiro | `200` lista paginada (`?empresa=`, `?fornecedor=`, `?status=`, `?numero=`, `?chegada_ate=AAAA-MM-DD`), mais recente primeiro |
| `GET` | `/api/v1/importacoes/{id}` | comercial ou financeiro | `200` importação com itens |
| `PUT` | `/api/v1/importacoes/{id}` | **financeiro** | `200` importação alterada |
| `POST` | `/api/v1/importacoes/{id}/embarcar` | **financeiro** | `200` importação `embarcada` |
| `POST` | `/api/v1/importacoes/{id}/desembaracar` | **financeiro** | `200` importação `desembaracada` |
| `POST` | `/api/v1/importacoes/{id}/cancelar` | **financeiro** | `200` importação `cancelada` |
| `GET` | `/api/v1/importacoes/{id}/historico` | comercial ou financeiro | `200` lista (`{ "dados": [...] }`) em ordem cronológica |
| `POST` | `/api/v1/importacoes/{id}/parcelas` | **financeiro** | `201` importação com a parcela nova |
| `PUT` | `/api/v1/importacoes/{id}/parcelas/{parcela_id}` | **financeiro** | `200`; só parcela `aberta` |
| `DELETE` | `/api/v1/importacoes/{id}/parcelas/{parcela_id}?versao=` | **financeiro** | `200` importação sem a parcela; só parcela `aberta` |
| `POST` | `/api/v1/importacoes/{id}/parcelas/{parcela_id}/cambio` | **financeiro** | `200`; fecha o câmbio e gera a conta a pagar |
| `POST` | `/api/v1/importacoes/{id}/parcelas/{parcela_id}/reabrir` | **financeiro** | `200`; com a conta a pagar cancelada ou rejeitada |
| `POST` | `/api/v1/importacoes/{id}/custos` | **financeiro** | `201`; lança o custo e gera a conta a pagar |
| `POST` | `/api/v1/importacoes/{id}/custos/{custo_id}/cancelar` | **financeiro** | `200`; com a conta a pagar cancelada ou rejeitada |
| `GET` | `/api/v1/importacoes/{id}/custo-previsto` | **financeiro** | `200` prévia do rateio (`?cambio_micro=`), sem gravar |
| `PUT` | `/api/v1/importacoes/{id}/alocacoes` | comercial ou financeiro | `200` importação com as alocações (substitui todas) |
| `POST` | `/api/v1/importacoes/{id}/receber` | **financeiro** | `200` importação recebida: entrada no estoque e reservas para os pedidos |
| `GET` | `/api/v1/importacoes/alertas` | comercial ou financeiro | `200` prazos em risco, alocações sem previsão e pedidos sem importação (`?empresa=`, `?folga=`) |
| `GET` | `/api/v1/pedidos/{id}/importacoes` | comercial ou financeiro | `200` lista (`{ "dados": [...] }`) das importações que atendem o pedido |

Leitura sem perfil comercial nem financeiro → `403 perfil sem permissão para consultar importações`;
escrita sem financeiro → `403` do middleware. Importação de empresa fora do acesso → `403 perfil sem acesso
a esta empresa`. **Os valores só aparecem para o financeiro**: para o comercial,
`valor_total_moeda_centesimos`, `cambio_micro`, `itens[].valor_moeda_centesimos`, `itens[].custo_unitario_centavos`,
`parcelas` e `custos` saem do JSON; itens recebidos e `alocacoes` aparecem para os dois.

### Estados

| Ação | De | Para | Exige |
| --- | --- | --- | --- |
| criar | — | `aberta` | empresa ativa no acesso, fornecedor ativo, itens |
| alterar | `aberta`, `embarcada` | igual | `versao`; fornecedor e moeda só em `aberta` (`409`); depois do embarque a chegada prevista é obrigatória |
| embarcar | `aberta` | `embarcada` | `embarcada_em` (não futura), `chegada_prevista` (não antes do embarque), `documento_embarque` opcional (BL/AWB) |
| desembaraçar | `embarcada` | `desembaracada` | `chegada_em` (não futura, não antes do embarque), `desembaracada_em` (não futura, não antes da chegada), `declaracao` (número da DI/DUIMP, até 30) |
| receber | `desembaracada` | `recebida` | seção "Recebimento"; depois disso nada muda, salvo custo posterior |
| cancelar | `aberta`, `embarcada`, `desembaracada` | `cancelada` | `motivo` (até 1000); toda conta a pagar gerada (parcela com câmbio fechado ou custo não cancelado) precisa estar `cancelado` ou `rejeitado` no M7 (`409 cancele ou rejeite antes as contas a pagar geradas pela importação`) |

Toda ação exige a `versao` lida (`400 erro de validação: versão da importação é obrigatória` sem ela; `409 a
importação foi alterada por outra pessoa; recarregue e tente de novo` divergente) e responde `409` quando o
estado não permite (`não é possível mudar a importação de "aberta" para "desembaracada"`). "Hoje" é o dia em
Brasília.

### Criar e alterar

```json
{
  "empresa_id": "…",
  "fornecedor_id": "…",
  "moeda": "USD",
  "referencia": "PI-2026-001",
  "embarque_previsto": "2026-11-05",
  "chegada_prevista": "2026-12-20",
  "observacoes": "",
  "itens": [{ "produto_id": "…", "quantidade": 1000, "valor_moeda_centesimos": 450000 }]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | só na criação; empresa ativa e no acesso (`403` fora dele, `400` inativa) |
| `fornecedor_id` | obrigatório; ativo ao criar ou trocar (`400 erro de validação: fornecedor "X" está inativo e não recebe importação nova`) |
| `moeda` | `BRL`, `USD`, `EUR` ou `CNY`; vazia na criação usa a `moeda_padrao` do fornecedor, vazia na alteração mantém a atual |
| `referencia` | opcional, até 60 (proforma ou invoice do fornecedor) |
| `embarque_previsto`, `chegada_prevista` | opcionais, `AAAA-MM-DD`; chegada não antes do embarque previsto (nem do embarque real, depois de embarcada) |
| `itens` | pelo menos um; produto ativo ao incluir e sem repetir; `quantidade` de 1 a 2147483647; `valor_moeda_centesimos` é o **total do item** na moeda, em centésimos (`>= 0`; zero é bonificação); cada valor e o total da importação vão até `1000000000000000` (`400 erro de validação: valor acima do limite permitido`) |

O `PUT` recebe o mesmo corpo sem `empresa_id` e com `versao`, e **substitui** os itens; o item de um produto
que continua na importação mantém o `id`. Produto e fornecedor já gravados não são conferidos de novo (podem
ter sido desativados depois). Com parcelas lançadas, a **moeda** não muda (`409`; remova as abertas e reabra as
de câmbio fechado antes) e, com parcela de câmbio fechado, o **fornecedor** também não (`409`). Tirar um item
alocado a pedido ou reduzi-lo abaixo do alocado, ou deixar os itens abaixo da soma das parcelas → `400`.

Resposta (criar, detalhe e ações):

```json
{
  "dados": {
    "id": "…", "empresa_id": "…", "numero": 1, "fornecedor_id": "…", "referencia": "PI-2026-001",
    "moeda": "USD", "status": "embarcada", "valor_total_moeda_centesimos": 450000,
    "embarque_previsto": null, "embarcada_em": "2026-10-09", "chegada_prevista": "2026-11-18",
    "chegada_em": null, "desembaracada_em": null, "documento_embarque": "BL-778", "declaracao": null,
    "observacoes": "", "cancelada_motivo": null, "usuario_id": "…", "versao": 3,
    "criado_em": "2026-10-19T13:00:00Z", "atualizado_em": "2026-10-19T13:05:00Z",
    "recebida_em": null, "local_recebimento_id": null,
    "itens": [{
      "id": "…", "produto_id": "…", "quantidade": 1000, "valor_moeda_centesimos": 450000,
      "quantidade_recebida": null, "movimento_id": null
    }],
    "parcelas": [], "custos": [], "alocacoes": []
  }
}
```

`numero` é sequencial por empresa. A lista traz os mesmos campos sem `itens`, `parcelas`, `custos` e
`alocacoes`. Depois do recebimento aparecem também `cambio_micro` e `itens[].custo_unitario_centavos` (só para
o financeiro; ver "Recebimento").

### Ações

```json
{ "versao": 2, "embarcada_em": "2026-10-09", "chegada_prevista": "2026-11-18", "documento_embarque": "BL-778" }
{ "versao": 3, "chegada_em": "2026-11-17", "desembaracada_em": "2026-11-19", "declaracao": "26/0001234-5" }
{ "versao": 4, "motivo": "fornecedor desistiu" }
```

Histórico: `id`, `usuario_id`, `acao`, `status_anterior`, `status_novo`, `detalhe` e `criado_em`, em ordem
cronológica. Ações: `criada`, `alterada`, `embarcada` (detalhe: chegada prevista), `desembaracada` (declaração),
`cancelada` (motivo), `alocacao_alterada` (quantas alocações e unidades), `parcela_incluida`,
`parcela_alterada`, `parcela_removida`, `parcela_cambio_fechado` (câmbio e valor em reais), `parcela_reaberta`,
`custo_lancado`, `custo_cancelado` e `recebida` (data, local, mercadoria, custos e unidades reservadas). **Para
quem não é financeiro, o `detalhe` das ações de parcela, custo e recebimento vem vazio**, porque traz valores.
Toda operação de parcela ou custo exige a `versao` da importação e a avança.

Fornecedor, produto e empresa usados numa importação passam a responder `400 erro de validação: registro em
uso por outros dados e não pode ser excluído` na exclusão.

### Parcelas ao fornecedor e câmbio

A parcela é o pagamento ao fornecedor **na moeda** da importação (sinal, saldo). Ela só vira conta a pagar
quando o câmbio é fechado, com o valor em reais exato; antes disso não aparece no M7 nem na projeção do M13.

```json
{ "versao": 1, "descricao": "Sinal 30%", "valor_moeda_centesimos": 135000, "vencimento_previsto": "2026-10-20" }
```

- `descricao` obrigatória (até 120); `valor_moeda_centesimos > 0`; `vencimento_previsto` obrigatório. A soma
  das parcelas ativas (abertas e com câmbio fechado) não passa do total dos itens (`400 erro de validação: a
  soma das parcelas não pode passar do valor dos itens da importação`), e o `PUT` da importação também não
  pode reduzir os itens abaixo dela. Importação cancelada não recebe parcela nem custo (`409`).
- `PUT` substitui descrição, valor e vencimento previsto da parcela `aberta`; `DELETE` a apaga (só `aberta`,
  `?versao=` com a versão da importação). Parcela com câmbio fechado → `409`.

**Fechar o câmbio** (`POST .../parcelas/{parcela_id}/cambio`):

```json
{ "versao": 2, "cambio_micro": 5432100, "vencimento": "2026-10-20", "categoria_financeira_id": "…" }
```

| Campo | Regra |
| --- | --- |
| `cambio_micro` | reais por unidade da moeda com 6 casas, em inteiro (`5,432100` → `5432100`); de 1 a 1000000000. Importação em `BRL` dispensa (vale `1000000`; outro valor → `400`) |
| `vencimento` | obrigatório; vira o `prazo_pagamento` da conta a pagar e a competência dela |
| `categoria_financeira_id` | subcategoria ativa de **despesa** (sugerida: `compra_mercadoria`, fora do resultado; o custo chega à DRE pelo CMV dos lotes) |

`valor_centavos = valor_moeda_centesimos × cambio_micro ÷ 1000000`, arredondado para o centavo mais próximo
(meio para cima). Na mesma gravação nasce a conta a pagar: `origem = importacao_parcela`, `origem_id` = parcela,
fornecedor da importação, empresa importadora, `pendente_aprovacao`, já classificada, observação "importação
N — descrição". A parcela passa a `cambio_fechado` com `cambio_micro`, `valor_centavos`, `vencimento`,
`solicitacao_id` e `status_conta_a_pagar`.

**Reabrir** (`POST .../reabrir { "versao": n }`): só com a conta a pagar `cancelado` ou `rejeitado` no M7
(senão `409 cancele ou rejeite antes a conta a pagar da parcela`). A parcela vira `cancelada` (guarda a conta
antiga) e nasce uma parcela `aberta` igual, que pode ter o câmbio fechado de novo.

### Custos de nacionalização

```json
{
  "versao": 3, "tipo": "despachante", "descricao": "", "fornecedor_id": "…", "valor_centavos": 250000,
  "vencimento": "2026-11-10", "compoe_custo": true, "categoria_financeira_id": "…"
}
```

| Campo | Regra |
| --- | --- |
| `tipo` | `frete_internacional`, `seguro`, `imposto_importacao`, `ipi`, `pis`, `cofins`, `icms`, `taxa_siscomex`, `despachante`, `armazenagem`, `frete_nacional` ou `outros` |
| `descricao` | opcional (até 255); vazia usa o nome do tipo ("Despachante aduaneiro") |
| `fornecedor_id` | obrigatório e ativo: a quem se paga (transportadora, despachante, Receita) |
| `valor_centavos` | em reais, `> 0` |
| `compoe_custo` | padrão `true`; `false` para tributo que a empresa credita (ICMS, IPI, PIS/COFINS conforme o regime, conferido com a contabilidade) |
| `categoria_financeira_id` | subcategoria ativa de despesa |

O custo gera a conta a pagar na mesma gravação (`origem = importacao_custo`, `origem_id` = custo, fornecedor do
custo). Custo lançado depois do recebimento sai com `posterior = true` e `compoe_custo = false` (o lote já foi
criado). **Cancelar** (`POST .../custos/{custo_id}/cancelar { "versao": n }`) exige a conta a pagar cancelada ou
rejeitada e marca `cancelado` (`cancelado_em`, `cancelado_por`); nada é apagado. Depois do recebimento, custo que
entrou no lote (`compoe_custo = true`) não é cancelado (`409 o custo já entrou no custo dos lotes recebidos e não
pode ser cancelado`). Valor do custo até `1000000000000000` centavos.

Resposta do detalhe para o financeiro (além dos campos da importação):

```json
{
  "parcelas": [{
    "id": "…", "descricao": "Sinal 30%", "valor_moeda_centesimos": 135000, "vencimento_previsto": "2026-10-20",
    "status": "cambio_fechado", "cambio_micro": 5432100, "valor_centavos": 733334, "vencimento": "2026-10-20",
    "solicitacao_id": "…", "status_conta_a_pagar": "pendente_aprovacao", "usuario_id": "…",
    "criado_em": "…", "atualizado_em": "…"
  }],
  "custos": [{
    "id": "…", "tipo": "despachante", "descricao": "Despachante aduaneiro", "fornecedor_id": "…",
    "valor_centavos": 250000, "vencimento": "2026-11-10", "compoe_custo": true, "posterior": false,
    "categoria_financeira_id": "…", "solicitacao_id": "…", "status_conta_a_pagar": "pendente_aprovacao",
    "cancelado": false, "cancelado_em": null, "cancelado_por": null, "usuario_id": "…", "criado_em": "…"
  }]
}
```

### Custo previsto

`GET /importacoes/{id}/custo-previsto?cambio_micro=5500000` calcula o custo que o recebimento gravaria, sem
gravar:

1. Mercadoria em reais = valor em reais das parcelas com câmbio fechado + a parte dos itens ainda sem parcela
   fechada convertida pelo `cambio_micro` informado (obrigatório só quando essa parte existe: `400 erro de
   validação: informe o câmbio para os N centésimos de USD ainda sem câmbio fechado`; em `BRL` vale 1). Câmbio
   enviado é sempre validado (de 1 a 1000000000; em `BRL` só `1000000`), mesmo quando não é usado.
2. A mercadoria é dividida entre os itens pelo valor na moeda (o resto do arredondamento vai para o último item).
3. Os custos com `compoe_custo` (não cancelados) são divididos pelo valor em reais de cada item.
4. `custo_unitario_centavos = (mercadoria + custos do item) ÷ quantidade`, arredondado.

```json
{
  "dados": {
    "valor_moeda_centesimos": 450000, "parcelas_fechadas_moeda_centesimos": 135000,
    "parcelas_fechadas_centavos": 733334, "sem_cambio_fechado_moeda_centesimos": 315000,
    "cambio_informado_micro": 5500000, "cambio_medio_micro": 5479631, "valor_mercadoria_centavos": 2465834,
    "custos_centavos": 250000, "custos_fora_do_rateio_centavos": 0,
    "itens": [{
      "item_id": "…", "produto_id": "…", "quantidade": 1000, "valor_moeda_centesimos": 450000,
      "valor_mercadoria_centavos": 2465834, "custos_centavos": 250000, "custo_total_centavos": 2715834,
      "custo_unitario_centavos": 2716
    }]
  }
}
```

### Alocação de pedidos

`PUT /importacoes/{id}/alocacoes` (comercial ou financeiro) diz quais itens de pedidos de venda a importação
atende. O corpo **substitui** todas as alocações da importação (lista vazia limpa):

```json
{ "versao": 2, "alocacoes": [{ "importacao_item_id": "…", "pedido_item_id": "…", "quantidade": 600 }] }
```

| Regra | Erro |
| --- | --- |
| item da importação e item de pedido existem; par sem repetir | `400` |
| pedido de **importação**, `confirmado` ou `parcialmente_entregue`, da **mesma empresa** da importação (estoque não passa entre CNPJs) e do mesmo produto do item | `400` |
| Σ alocações do item da importação ≤ quantidade dele | `400 erro de validação: as alocações passam da quantidade do item da importação` |
| Σ alocações do item do pedido, somando as outras importações não canceladas e não recebidas, ≤ o que falta entregar | `400 erro de validação: as alocações do item do pedido N passam do que falta entregar (…)`; corrida com outra importação → `409` |
| importação `aberta`, `embarcada` ou `desembaracada` | `409` |

O `PUT` da importação recusa (`400`) tirar um item alocado ou reduzi-lo abaixo do alocado. Importação cancelada
deixa de contar nas somas. Histórico: `alocacao_alterada`.

### Recebimento

`POST /importacoes/{id}/receber` (financeiro), com a importação `desembaracada`:

```json
{
  "versao": 4, "recebida_em": "2026-11-20", "local_id": "…", "cambio_micro": 5500000,
  "itens": [{ "item_id": "…", "quantidade_recebida": 950 }]
}
```

| Campo | Regra |
| --- | --- |
| `recebida_em` | obrigatória, não futura, não antes do desembaraço |
| `local_id` | local **ativo da empresa importadora** (`400` de outra empresa ou inativo) |
| `cambio_micro` | o mesmo do custo previsto: obrigatório só quando há valor sem parcela com câmbio fechado |
| `itens` | opcional; item que não vier é recebido inteiro. `quantidade_recebida` de `0` à quantidade do item (falta e avaria); item repetido → `400`; nada recebido → `400` (cancele a importação) |

Numa transação só:

1. o custo é calculado como no custo previsto, com as quantidades recebidas: o valor em reais de cada item é o
   mesmo, dividido pela quantidade recebida (a falta encarece a unidade; item com zero não entra no estoque e
   não recebe custos);
2. cada item com quantidade recebida gera uma **entrada** no local, com o custo unitário e `documento_ref =
   IMP-<numero>`: é o lote do estoque (e o CMV do M15). `itens[].movimento_id` aponta para o movimento;
3. cada alocação vira uma **reserva** do pedido (`origem = pedido_item`) no mesmo local, do menor entre a
   alocação, o que falta cobrir no item do pedido (restante − reservado) e o recebido ainda não reservado, por
   ordem de prazo-limite do pedido (sem prazo por último). `alocacoes[].reservada` e `reserva_id` guardam o
   resultado; o pedido ganha `pendencias_reservadas` no histórico. Pedido que saiu de atendimento no meio →
   `409` e nada é gravado;
4. o que sobra do recebido fica disponível no estoque.

Parcela com câmbio fechado cuja conta a pagar foi cancelada ou rejeitada no M7, ou custo que compõe o custo com a
conta nessa situação, bloqueia o recebimento (`409`): reabra a parcela ou cancele o custo antes, para o lote não
receber valor que não será pago. Pedido que mudou durante o recebimento a ponto de ficar com reserva acima do que
falta entregar → `409` e nada é gravado.

Estoque alterado por outra operação no mesmo produto → `409 estoque do produto alterado por outra operação,
recarregue e tente novamente`. Depois de recebida, a importação só aceita custo novo, gravado com `posterior =
true` (não muda o lote), e parcela.

Campos que o recebimento preenche no detalhe: `recebida_em`, `local_recebimento_id`, `cambio_micro`
(financeiro) e, nos itens, `quantidade_recebida`, `custo_unitario_centavos` (financeiro) e `movimento_id`;
`alocacoes` vem sempre:

```json
{
  "alocacoes": [{
    "importacao_item_id": "…", "pedido_item_id": "…", "quantidade": 600, "reservada": 600, "reserva_id": "…",
    "usuario_id": "…", "criado_em": "…", "atualizado_em": "…"
  }]
}
```

### Alertas

`GET /importacoes/alertas?empresa=&folga=` (comercial ou financeiro; sem `empresa`, todas do acesso). `folga` é
o tempo, em dias, entre a chegada e a entrega ao cliente (desembaraço e frete): padrão `10`, de `0` a `90`
(`400` fora disso). Só entram importações `aberta`, `embarcada` ou `desembaracada` e pedidos de importação
`confirmado` ou `parcialmente_entregue`.

| Lista | O que mostra |
| --- | --- |
| `prazo_em_risco` | alocação cuja chegada (`chegada_em` quando já chegou, senão `chegada_prevista`) `+ folga` passa do `prazo_limite` do pedido, com `chegada_com_folga` e `dias_de_atraso`; pedido sem prazo-limite (importação sem entrada paga) não entra |
| `sem_previsao` | alocação de importação ainda sem `chegada_prevista` |
| `sem_importacao` | item de pedido de importação com `restante − reservado − alocado > 0` (`sem_cobertura`), somando as importações abertas |

Ordenadas pelo prazo-limite (sem prazo por último).

```json
{
  "dados": {
    "folga_dias": 10,
    "prazo_em_risco": [{
      "importacao_id": "…", "importacao_numero": 1, "importacao_status": "embarcada", "empresa_id": "…",
      "chegada_prevista": "2027-05-07", "chegada_em": null, "pedido_id": "…", "pedido_numero": 3, "pedido_item_id": "…",
      "produto_id": "…", "quantidade": 300, "prazo_limite": "2027-02-16",
      "chegada_com_folga": "2027-05-17", "dias_de_atraso": 90
    }],
    "sem_previsao": [],
    "sem_importacao": [{
      "pedido_id": "…", "pedido_numero": 3, "empresa_id": "…", "pedido_item_id": "…", "produto_id": "…",
      "prazo_limite": "2027-02-16", "restante": 600, "reservado": 0, "alocado": 400, "sem_cobertura": 200
    }]
  }
}
```

### Importações do pedido

`GET /pedidos/{id}/importacoes` (comercial ou financeiro, pedido de empresa no acesso; `404` sem pedido) lista
as alocações do pedido em importações não canceladas: `importacao_id`, `importacao_numero`, `importacao_status`,
`chegada_prevista`, `recebida_em`, `pedido_item_id`, `produto_id`, `quantidade` e `reservada` (preenchida no
recebimento), em ordem de número da importação.

---

## Dívidas e empréstimos (financeiro)

Empréstimos bancários, aportes de investidores, parcelamentos com o fisco e dívidas parceladas com fornecedores.
Cada parcela vira **duas contas a pagar** no M7: a amortização (`origem = divida_amortizacao`, categoria em
`fora_dre`) e os juros (`origem = divida_juros`, categoria em `resultado_financeiro`), as duas com `origem_id` =
id da parcela. Escopo por empresa pela devedora (seção 4 do `EMPRESAS.md`). Escopo e decisões em `DIVIDAS.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/dividas` | **financeiro** | `201` dívida com parcelas; gera as contas do mês corrente |
| `GET` | `/api/v1/dividas` | **financeiro** | `200` lista paginada (`?empresa=`, `?fornecedor=`, `?credor_tipo=`, `?status=`, `?quitada=`, `?vencidas=`), mais recente primeiro, com saldo e vencido |
| `GET` | `/api/v1/dividas/{id}` | **financeiro** | `200` dívida com parcelas, contas a pagar, situação e saldo devedor |
| `PUT` | `/api/v1/dividas/{id}` | **financeiro** | `200`; só os campos informativos e as categorias |
| `PUT` | `/api/v1/dividas/{id}/parcelas/{parcela_id}` | **financeiro** | `200`; vencimento e juros da parcela `prevista` |
| `POST` | `/api/v1/dividas/{id}/cancelar` | **financeiro** | `200` dívida `cancelada`; cancela as contas abertas |
| `POST` | `/api/v1/dividas/{id}/renegociar` | **financeiro** | `201` dívida nova ligada à original; a original fica `renegociada` e as contas abertas dela são canceladas |
| `POST` | `/api/v1/dividas/{id}/parcelas/{parcela_id}/reabrir` | **financeiro** | `200`; parcela com as contas canceladas ou rejeitadas no M7 volta a `prevista` |
| `GET` | `/api/v1/dividas/{id}/historico` | **financeiro** | `200` lista (`{ "dados": [...] }`) em ordem cronológica |
| `GET` | `/api/v1/dividas/resumo` | **financeiro** | `200` saldo devedor por tipo de credor e por empresa, vencido × a vencer (`?empresa=`) |
| `POST` | `/api/v1/dividas/gerar?competencia=AAAA-MM` | **financeiro** | `200` contas a pagar geradas das parcelas que vencem até o fim da competência (`?empresa=` opcional) |

Sem perfil financeiro → `403` do middleware. Dívida de empresa fora do acesso → `403 perfil sem acesso a esta
empresa`; inexistente → `404 registro não encontrado: dívida não encontrada`. "Hoje" é o dia em Brasília.

### Criar

A dívida entra pelo **saldo devedor na data do saldo** (data de corte da carga, ou da liberação numa dívida
nova) e só com as parcelas que vencem **depois** dela; parcelas pagas ou vencidas antes ficam fora e não geram
conta a pagar. Valor original, taxa, contrato e liberação são informativos.

```json
{
  "empresa_id": "…",
  "fornecedor_id": "…",
  "credor_tipo": "banco",
  "descricao": "Capital de giro Sicredi",
  "contrato": "CG-2026-77",
  "valor_original_centavos": 500000,
  "contratada_em": "2026-03-10",
  "taxa_bp": 185,
  "taxa_periodo": "mensal",
  "indexador": "cdi",
  "conta_liberacao_id": "…",
  "liberada_em": "2026-03-10",
  "saldo_inicial_centavos": 300000,
  "data_saldo": "2026-09-30",
  "categoria_amortizacao_id": "…",
  "categoria_juros_id": "…",
  "observacoes": "",
  "parcelas": [
    { "vencimento": "2026-10-10", "amortizacao_centavos": 100000, "juros_centavos": 5550 },
    { "vencimento": "2026-11-10", "amortizacao_centavos": 100000, "juros_centavos": 3700 },
    { "vencimento": "2026-12-10", "amortizacao_centavos": 100000, "juros_centavos": 1850 }
  ]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | só na criação; empresa ativa e no acesso (`403` fora dele, `400` inativa) |
| `fornecedor_id` | o credor, cadastrado no M4 (banco, investidor e Receita também); ativo ao criar (`400 erro de validação: credor "X" está inativo e não recebe dívida nova`); não muda depois |
| `credor_tipo` | `banco`, `investidor`, `fisco` ou `fornecedor` |
| `descricao` | obrigatória, até 120 |
| `contrato` | opcional, até 60 |
| `valor_original_centavos` | opcional, `> 0` |
| `contratada_em`, `liberada_em` | opcionais, `AAAA-MM-DD`, não futuras |
| `taxa_bp`, `taxa_periodo` | opcionais e juntos; taxa em pontos-base (`1,85%` → `185`, de 0 a 100000), período `mensal` ou `anual` |
| `indexador` | `prefixado` (padrão), `cdi`, `selic`, `ipca`, `tr` ou `outro` |
| `conta_liberacao_id` | opcional, junto com `liberada_em`; conta bancária da mesma empresa. Só informativo: o crédito é classificado no caixa (M13) como captação em `fora_dre` |
| `saldo_inicial_centavos` | obrigatório, `> 0`; igual à **soma das amortizações** (`400 erro de validação: a soma das amortizações (R$ 3000,00) deve ser igual ao saldo devedor na data do saldo (R$ 3000,01)`) |
| `data_saldo` | obrigatória, não futura |
| `categoria_amortizacao_id` | subcategoria ativa de despesa na linha `fora_dre` |
| `categoria_juros_id` | subcategoria ativa de despesa na linha `resultado_financeiro` (`400 erro de validação: categoria financeira de juros deve ser de despesa na linha resultado_financeiro`) |
| `parcelas` | de 1 a 600; `vencimento` depois da `data_saldo` e sem repetir; `amortizacao_centavos` e `juros_centavos` `>= 0` (juros inclui encargos e IOF da parcela), não os dois zero, até 10^15 |

As parcelas são numeradas pela ordem de vencimento. Na criação já nascem as contas a pagar das parcelas que
vencem **até o fim do mês corrente**; as demais ficam `prevista` até a rotina de geração.

Resposta (criar, detalhe e ações):

```json
{
  "dados": {
    "id": "…", "empresa_id": "…", "credor_tipo": "banco", "fornecedor_id": "…",
    "descricao": "Capital de giro Sicredi", "contrato": "CG-2026-77", "valor_original_centavos": 500000,
    "contratada_em": "2026-03-10", "taxa_bp": 185, "taxa_periodo": "mensal", "indexador": "cdi",
    "saldo_inicial_centavos": 300000, "data_saldo": "2026-09-30",
    "conta_liberacao_id": "…", "liberada_em": "2026-03-10",
    "categoria_amortizacao_id": "…", "categoria_juros_id": "…", "divida_origem_id": null,
    "status": "ativa", "motivo": null, "observacoes": "", "usuario_id": "…", "versao": 1,
    "criado_em": "2026-10-20T13:00:00Z", "atualizado_em": "2026-10-20T13:00:00Z",
    "saldo_devedor_centavos": 300000,
    "quitada": false,
    "parcelas": [{
      "id": "…", "numero": 1, "vencimento": "2026-10-10", "amortizacao_centavos": 100000, "juros_centavos": 5550,
      "status": "gerada", "situacao": "vencida", "gerada_em": "2026-10-20T13:00:00Z", "substituta_id": null,
      "conta_amortizacao": { "solicitacao_id": "…", "status": "pendente_aprovacao" },
      "conta_juros": { "solicitacao_id": "…", "status": "pendente_aprovacao" }
    }]
  }
}
```

- `parcelas[].status`: `prevista` (sem conta a pagar), `gerada` ou `cancelada`. Componente zero não gera conta
  (`conta_juros: null` numa parcela só de amortização).
- `parcelas[].situacao`, calculada na leitura: `cancelada`; `conta_cancelada` (alguma conta da parcela foi
  cancelada ou rejeitada no contas a pagar); `paga` (todas as contas geradas estão `pago`); `vencida`
  (vencimento antes de hoje); `a_vencer`.
- `saldo_devedor_centavos`: soma das amortizações não pagas das parcelas não canceladas (`0` em dívida
  `renegociada` ou `cancelada`). `quitada`: dívida ativa sem nenhum componente em aberto.
- A lista traz os campos da dívida com `saldo_devedor_centavos`, `vencido_centavos` (amortização e juros em aberto
  das parcelas vencidas antes de hoje) e `quitada`, sem `parcelas`. `?quitada=true|false` e `?vencidas=true|false`
  (dívida `ativa` com parcela vencida em aberto) aceitam `true`/`false`/`1`/`0`; outro valor → `400`.

### Alterar e editar parcela

`PUT /api/v1/dividas/{id}` recebe os campos informativos com a `versao` (`credor_tipo`, `descricao`, `contrato`,
`valor_original_centavos`, `contratada_em`, `taxa_bp`, `taxa_periodo`, `indexador`, `conta_liberacao_id`,
`liberada_em`, `categoria_amortizacao_id`, `categoria_juros_id`, `observacoes`) e **substitui** todos eles
(campo omitido fica vazio). Empresa, credor, saldo, data do saldo e cronograma não mudam. Categorias novas
valem para as contas ainda não geradas.

`PUT /api/v1/dividas/{id}/parcelas/{parcela_id}` com `{ "versao": 1, "vencimento": "2026-11-15",
"juros_centavos": 4100 }` ajusta a parcela `prevista` (ex.: juros do parcelamento corrigido pela Selic); o
vencimento continua depois da data do saldo e entre o da parcela anterior e o da seguinte (`400`). Parcela
gerada → `409 só a parcela prevista, sem contas a pagar geradas, pode ser alterada`; a amortização só muda por
renegociação.

### Cancelar

`POST /api/v1/dividas/{id}/cancelar` com `{ "versao": 1, "motivo": "lançada em duplicidade" }` (motivo até
1000) é para cadastro errado: só com nenhuma conta da dívida `pago` (`409 dívida com parcela paga não pode ser
cancelada; renegocie o saldo ou estorne o pagamento antes`). Na mesma transação as contas `pendente_aprovacao`
e `aprovado` viram `cancelado` (com `solicitacao_historico` `cancelada`, "dívida cancelada: motivo") e as
parcelas viram `cancelada`. Se alguma conta da dívida mudou no M7 depois da leitura → `409 uma conta a pagar da
dívida mudou no contas a pagar; recarregue a dívida e tente de novo`. Dívida que não está `ativa` → `409
dívida cancelada não pode ser alterada` (ou `renegociada`).

### Renegociar e parcelar

`POST /api/v1/dividas/{id}/renegociar` troca o saldo em aberto da dívida `ativa` por uma dívida nova (acordo com o
banco, parcelamento com o fisco, dívida com fornecedor paga com empréstimo):

```json
{
  "versao": 3,
  "motivo": "acordo com o banco",
  "nova": {
    "fornecedor_id": "…", "credor_tipo": "banco", "descricao": "Renegociação Sicredi",
    "saldo_inicial_centavos": 210000, "data_saldo": "2026-10-20",
    "categoria_amortizacao_id": "…", "categoria_juros_id": "…",
    "parcelas": [{ "vencimento": "2026-11-20", "amortizacao_centavos": 105000, "juros_centavos": 900 }]
  }
}
```

- `nova` tem os mesmos campos da criação, **sem** `empresa_id`: a dívida nova é da mesma empresa (que precisa
  estar ativa). O credor pode mudar; o saldo é o informado (pode incluir multa e encargos incorporados) e o
  histórico registra o saldo em aberto da original para conferência. `data_saldo` não pode ser futura nem
  anterior à da original (`400`); motivo obrigatório, até 1000.
- Numa transação: a original vira `renegociada` com o motivo; as contas `pendente_aprovacao` e `aprovado` dela
  viram `cancelado` (histórico "dívida renegociada: motivo"); as parcelas sem nenhuma conta paga viram
  `cancelada` e as que têm conta paga ficam `gerada` (a conta aberta delas é cancelada); a nova nasce `ativa` com
  `divida_origem_id` e já gera as contas das parcelas que vencem até o fim do mês. Conta da original alterada no
  M7 desde a leitura → `409`.
- Resposta `201` com a dívida nova. Uma dívida só é renegociada uma vez (`409` depois); a nova pode ser
  renegociada de novo.

### Reabrir parcela

Quando uma conta de uma parcela `gerada` é cancelada ou rejeitada no contas a pagar (situação
`conta_cancelada`), `POST /api/v1/dividas/{id}/parcelas/{parcela_id}/reabrir` com `{ "versao": 4 }` marca a parcela
`cancelada` (com `substituta_id`) e cria outra `prevista` com o mesmo número e vencimento e **só com o que não foi
pago**: se a amortização já está `pago` e os juros foram rejeitados, a nova parcela tem amortização zero e os juros
originais. Ela pode ser editada e gerada de novo. Com alguma conta da parcela ainda aberta → `409 cancele ou
rejeite antes as contas a pagar da parcela no contas a pagar`; sem nenhuma conta cancelada ou rejeitada → `409 a
parcela não tem conta a pagar cancelada ou rejeitada para reabrir`; parcela que não está `gerada` → `409`.
A conta paga continua ligada à parcela substituída e impede cancelar a dívida.

Toda escrita exige a `versao` lida (`400 erro de validação: versão da dívida é obrigatória`; `409 a dívida foi
alterada por outra pessoa; recarregue e tente de novo`) e a avança.

### Geração das contas a pagar

`POST /api/v1/dividas/gerar?competencia=2026-11` (idempotente, ao lado de `POST /despesas-recorrentes/gerar`):
para cada dívida `ativa` do acesso (ou só da `?empresa=`) com parcela `prevista` que vence **até o fim da
competência**, gera as contas a pagar dessas parcelas numa transação por dívida e avança a `versao` dela.

```json
{
  "dados": {
    "competencia": "2026-11",
    "geradas": [{ "divida_id": "…", "parcelas": 1, "contas": 2 }],
    "recusadas": [{ "divida_id": "…", "motivo": "fornecedor inativo não recebe conta a pagar" }]
  }
}
```

As contas nascem `pendente_aprovacao`, com o credor como fornecedor, a categoria da dívida, forma
`transferencia`, vencimento e **competência do mês do vencimento** da parcela, solicitante = quem gerou e
observação "dívida <descrição> — parcela n/total — amortização|juros". Dívida com empresa ou credor inativo,
categoria que deixou de ser lançável ou alterada ao mesmo tempo vai para `recusadas` e não grava nada. Sem
`competencia` → `400 erro de validação: competência da geração é obrigatória`; empresa fora do acesso → `403`.

### Resumo

`GET /api/v1/dividas/resumo?empresa=` (empresa opcional; sem ela, todas do acesso) considera só dívidas `ativa` e
parcelas não canceladas; componente com conta `pago` não conta, parcela `prevista` conta inteira.

```json
{
  "dados": {
    "hoje": "2026-10-20",
    "saldo_devedor_centavos": 290000,
    "dividas": 2,
    "por_credor": [
      { "credor_tipo": "banco", "saldo_devedor_centavos": 200000, "dividas": 1 },
      { "credor_tipo": "investidor", "saldo_devedor_centavos": 0, "dividas": 0 },
      { "credor_tipo": "fisco", "saldo_devedor_centavos": 90000, "dividas": 1 },
      { "credor_tipo": "fornecedor", "saldo_devedor_centavos": 0, "dividas": 0 }
    ],
    "por_empresa": [{ "empresa_id": "…", "saldo_devedor_centavos": 290000, "dividas": 2 }],
    "vencido": { "amortizacao_centavos": 30000, "juros_centavos": 600, "parcelas": 1 },
    "a_vencer": { "amortizacao_centavos": 260000, "juros_centavos": 11700, "parcelas": 5 },
    "a_vencer_30_dias": { "amortizacao_centavos": 30000, "juros_centavos": 5950, "parcelas": 2 },
    "a_vencer_90_dias": { "amortizacao_centavos": 130000, "juros_centavos": 9650, "parcelas": 3 },
    "a_vencer_365_dias": { "amortizacao_centavos": 260000, "juros_centavos": 11700, "parcelas": 5 }
  }
}
```

- `saldo_devedor_centavos`: soma das amortizações em aberto (o principal); os juros aparecem só nas faixas.
- `dividas`: dívidas com algum componente em aberto. `por_credor` traz sempre os quatro tipos; `por_empresa` só as
  empresas com saldo ou juros em aberto, do maior saldo para o menor.
- `vencido`: vencimento antes de hoje (Brasília); `a_vencer`: de hoje em diante; as faixas contam de hoje até
  hoje + 30, 90 e 365 dias, inclusive.
- Empresa fora do acesso → `403`.

Histórico: `criada` (saldo, data e número de parcelas), `alterada`, `parcela_alterada` (vencimento e juros antes
e depois), `contas_geradas` (quantas contas e de quais parcelas), `parcela_reaberta`, `renegociada` (saldo em aberto,
saldo novo e motivo; a dívida nova começa com `criada` citando a original) e `cancelada` (motivo).

Credor, empresa, conta bancária e categorias usados numa dívida passam a responder `400 erro de validação:
registro em uso por outros dados e não pode ser excluído` na exclusão.

---

## DRE e painéis (financeiro)

Somente leitura: nada é digitado aqui. A DRE é por **competência** e soma o que os outros módulos já
gravam, cada número de uma única fonte (NF de venda, lotes do estoque, contas a pagar, títulos manuais,
recebimentos, borderôs e lançamentos de caixa). Escopo por empresa (padrão da seção 4 do `EMPRESAS.md`);
sem `?empresa=`, só as empresas do acesso do usuário. Escopo e decisões em `DRE.md`.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `GET` | `/api/v1/dre/resumo` | **financeiro** | `200` DRE por mês e empresa, consolidado e total (`?de=AAAA-MM`, `?ate=AAAA-MM`, `?empresa=`) |
| `GET` | `/api/v1/dre/faturamento` | **financeiro** | `200` faturamento por mês e empresa (`?de=AAAA-MM`, `?ate=AAAA-MM`, `?empresa=`) |
| `GET` | `/api/v1/dre/relatorio-semanal` | **financeiro** | `200` relatório da semana (`?semana=AAAA-MM-DD`, `?empresa=`) |

Sem financeiro → `403` do middleware. `?empresa=` fora do acesso → `403 perfil sem acesso a esta empresa`.
Mês fora do formato → `400 erro de validação: de deve estar no formato AAAA-MM` (ou `ate`); `ate` antes de
`de` → `400 erro de validação: mês final não pode ser anterior ao mês inicial`. Sem `de`, o período começa
11 meses antes de `ate`; sem `ate`, termina no mês corrente (Brasília). Máximo de 24 meses na DRE e de 60 no
faturamento (`400 erro de validação: período deve ter no máximo N meses`).

### Fontes e linhas

| `linha` | De onde vem | Sinal |
| --- | --- | --- |
| `receita_bruta` | NF `venda` autorizada, fora `entre_empresas`, `valor_total_centavos`, mês de `emitida_em` em Brasília (item `nf_venda`) | + |
| `outras_receitas` | categorias `receita_operacional` (títulos `manual`, lançamentos de caixa) | + |
| `deducoes` | categorias `deducao_venda` + NF `devolucao` autorizada de entrada (item `nf_devolucao`) | − |
| `cmv` | lotes consumidos por movimento `saida` (`Σ quantidade × custo do lote`), mês do movimento (item `cmv`) | − |
| `perdas_estoque` | lotes consumidos por `perda` e `ajuste` negativo (item `perdas_estoque`) | − |
| `despesas_variaveis` | categorias `despesa_variavel` | − |
| `despesas_fixas` | categorias `despesa_fixa` | − |
| `nao_classificado` | contas a pagar sem categoria (item `sem_categoria`) | − |
| `resultado_financeiro` | categorias `resultado_financeiro` + juros recebidos (`juros_recebidos`), descontos concedidos no recebimento (`descontos_concedidos`) e encargos do borderô ativo, bruto − líquido (`encargos_antecipacao`) | ± |
| `fora_dre` | categorias `fora_dre` (empréstimo, aporte, distribuição); fica **fora** do resultado | ± |

- **Contas a pagar** (`solicitacao`): por `competencia`, todos os status menos `cancelado` e `rejeitado`,
  qualquer `origem` (M10, M12 e M14 entram pela categoria). Conta `pago` usa o valor do pagamento vigente;
  as demais, `valor_estimado`. A conta de parcela de importação e a de custo de importação que compõe o custo
  do lote ficam fora em qualquer categoria: esse valor já entra como CMV.
- **Títulos**: só `manual`, por `competencia`; títulos de NF e de pedido ficam fora (a receita vem da NF).
  Juros e descontos pelo `recebido_em` dos recebimentos não estornados do tipo `conta` (no tipo `fomento` o
  cliente pagou direto ao fomento).
- **Caixa**: `lancamento_caixa` não estornado, por `competencia`, com o sinal da linha do extrato.
- **Nunca entram**: categorias `compra_mercadoria` (o custo vem dos lotes) e `entre_contas`,
  `transferencia_conta`, NF `remessa`, `transferencia` e devolução de saída, movimento `transferencia`.
- O detalhamento (`itens`) agrupa pela **categoria raiz** (`chave = "categoria"`, `categoria_id`, `nome`); as
  fontes sem categoria aparecem com a `chave` fixa e `categoria_id` nulo, antes das categorias.

### Resumo da DRE

```json
{
  "dados": {
    "de": "2026-09",
    "ate": "2026-10",
    "meses": [
      {
        "mes": "2026-09",
        "empresas": [{ "empresa_id": "…", "linhas": [], "subtotais": {}, "indicadores": {}, "quantidade_sem_custo": 0 }],
        "consolidado": { "linhas": [], "subtotais": {}, "indicadores": {}, "quantidade_sem_custo": 0 }
      }
    ],
    "total": {
      "empresas": [{ "empresa_id": "…", "linhas": [], "subtotais": {}, "indicadores": {}, "quantidade_sem_custo": 0 }],
      "consolidado": {
        "linhas": [
          {
            "linha": "receita_bruta", "valor_centavos": 10000000, "quantidade": 12,
            "itens": [{ "chave": "nf_venda", "categoria_id": null, "nome": "Notas fiscais de venda", "valor_centavos": 10000000, "quantidade": 12 }]
          },
          {
            "linha": "despesas_fixas", "valor_centavos": -2000000, "quantidade": 3,
            "itens": [{ "chave": "categoria", "categoria_id": "…", "nome": "Custos fixos", "valor_centavos": -2000000, "quantidade": 3 }]
          }
        ],
        "subtotais": {
          "receita_liquida_centavos": 9500000, "lucro_bruto_centavos": 5400000, "margem_contribuicao_centavos": 5000000,
          "resultado_operacional_centavos": 2800000, "resultado_centavos": 2700000
        },
        "indicadores": { "margem_contribuicao_bp": 5263, "ponto_equilibrio_centavos": 3800000 },
        "quantidade_sem_custo": 2
      }
    }
  }
}
```

- `linhas` traz sempre as dez linhas, na ordem da tabela acima, com valor já com sinal (despesa negativa).
- `empresas` lista, em todos os meses, as empresas com algum valor no período (e a de `?empresa=`, mesmo
  zerada), na mesma ordem.
- Subtotais: `receita_liquida` = receita bruta + outras receitas + deduções; `lucro_bruto` = receita líquida +
  CMV + perdas; `margem_contribuicao` = lucro bruto + despesas variáveis; `resultado_operacional` = margem +
  despesas fixas + não classificado; `resultado` = operacional + resultado financeiro (`fora_dre` não entra).
- `margem_contribuicao_bp` = margem ÷ receita líquida em pontos-base; `ponto_equilibrio_centavos` = despesas
  fixas ÷ (margem ÷ receita líquida). Os dois são nulos sem receita líquida positiva; o ponto de equilíbrio
  também com margem ≤ 0.
- `quantidade_sem_custo`: unidades saídas de lote com custo desconhecido (contadas como zero no CMV).

### Faturamento

```json
{
  "dados": {
    "de": "2025-11",
    "ate": "2026-10",
    "meses": [
      {
        "mes": "2026-10",
        "empresas": [{ "empresa_id": "…", "quantidade": 4, "valor_centavos": 436000, "ticket_medio_centavos": 109000 }],
        "consolidado": { "quantidade": 4, "valor_centavos": 436000, "ticket_medio_centavos": 109000 }
      }
    ],
    "empresas": [{ "empresa_id": "…", "quantidade": 30, "valor_centavos": 3270000, "ticket_medio_centavos": 109000 }],
    "consolidado": { "quantidade": 30, "valor_centavos": 3270000, "ticket_medio_centavos": 109000 }
  }
}
```

Só NF `venda` autorizada fora `entre_empresas`, pelo mês de `emitida_em` em Brasília; NF `complementar` e de
`ajuste` somam o valor, mas não contam em `quantidade`. Mostra apenas as notas
já importadas (sem carga retroativa); meses sem nota aparecem zerados. `ticket_medio_centavos` arredondado,
`0` sem notas.

### Relatório semanal

`GET /dre/relatorio-semanal?semana=AAAA-MM-DD` mostra a semana de **segunda a domingo** que contém `semana`
(qualquer dia dela); sem `semana`, a última semana encerrada. Semana que começa depois da corrente → `400
erro de validação: semana não pode estar no futuro`; data fora do formato → `400 erro de validação: semana deve
estar no formato AAAA-MM-DD`. "Hoje" é o dia de Brasília.

```json
{
  "dados": {
    "inicio": "2026-10-12", "fim": "2026-10-18", "hoje": "2026-10-20",
    "notas_emitidas": {
      "total_venda": { "quantidade": 3, "valor_centavos": 327000 },
      "por_empresa": [{ "empresa_id": "…", "natureza": "venda", "quantidade": 3, "valor_centavos": 327000 }]
    },
    "pagamentos": {
      "total": { "quantidade": 2, "valor_centavos": 150000 },
      "por_empresa": [{ "empresa_id": "…", "linha_dre": "despesa_fixa", "quantidade": 2, "valor_centavos": 150000 }]
    },
    "recebimentos": {
      "total": { "quantidade": 1, "valor_centavos": 54500 },
      "por_empresa": [{ "empresa_id": "…", "tipo": "conta", "quantidade": 1, "valor_centavos": 54500, "juros_centavos": 0 }]
    },
    "titulos_vencidos": {
      "total": { "quantidade": 2, "valor_centavos": 109000 },
      "por_empresa": [{ "empresa_id": "…", "quantidade": 2, "valor_centavos": 109000 }],
      "itens": [{
        "id": "…", "empresa_id": "…", "cliente_nome": "Construtora Exemplo SPE", "numero": "123", "parcela": 1,
        "status": "carteira", "vencimento": "2026-10-01", "saldo_centavos": 54500
      }]
    },
    "contas_vencidas": {
      "total": { "quantidade": 1, "valor_centavos": 50000 },
      "por_empresa": [{ "empresa_id": "…", "quantidade": 1, "valor_centavos": 50000 }],
      "itens": [{
        "id": "…", "empresa_id": "…", "fornecedor_nome": null, "observacao": "Compra de cimento",
        "status": "aprovado", "vencimento": "2026-10-01", "valor_centavos": 50000
      }]
    },
    "estoque_baixo": {
      "total": 1,
      "itens": [{ "produto_id": "…", "nome": "Cimento CP II", "codigo": null, "quantidade_atual": 0, "estoque_minimo": 5 }]
    },
    "saldos_bancarios": {
      "saldo_total_centavos": 193710, "contas_sem_abertura": 0,
      "contas": [{
        "conta_bancaria_id": "…", "empresa_id": "…", "apelido": "Sicredi", "banco": "748", "banco_nome": "Sicredi",
        "ativa": true, "data_corte": "2026-09-30", "saldo_atual_centavos": 193710, "ultimo_lancamento_em": "2026-10-14"
      }]
    },
    "endividamento": {
      "saldo_devedor_centavos": 300000, "dividas": 1,
      "por_credor": [
        { "credor_tipo": "banco", "saldo_devedor_centavos": 300000, "dividas": 1 },
        { "credor_tipo": "investidor", "saldo_devedor_centavos": 0, "dividas": 0 },
        { "credor_tipo": "fisco", "saldo_devedor_centavos": 0, "dividas": 0 },
        { "credor_tipo": "fornecedor", "saldo_devedor_centavos": 0, "dividas": 0 }
      ],
      "por_empresa": [{ "empresa_id": "…", "saldo_devedor_centavos": 300000, "dividas": 1 }],
      "vencido": { "amortizacao_centavos": 0, "juros_centavos": 0, "parcelas": 0 },
      "a_vencer": { "amortizacao_centavos": 300000, "juros_centavos": 11100, "parcelas": 3 },
      "a_vencer_30": { "amortizacao_centavos": 100000, "juros_centavos": 5550, "parcelas": 1 },
      "a_vencer_90": { "amortizacao_centavos": 200000, "juros_centavos": 9250, "parcelas": 2 },
      "a_vencer_365": { "amortizacao_centavos": 300000, "juros_centavos": 11100, "parcelas": 3 }
    }
  }
}
```

| Bloco | O que entra |
| --- | --- |
| `notas_emitidas` | NF autorizada com `emitida_em` (Brasília) na semana, por empresa e natureza; `total_venda` soma só `venda` |
| `pagamentos` | pagamentos não estornados com `pago_em` (Brasília) na semana, por empresa e `linha_dre` da categoria (`null` sem categoria) |
| `recebimentos` | recebimentos de título não estornados com `recebido_em` na semana, por empresa e `tipo` (`conta`, `fomento`), com os juros; `total` soma só o tipo `conta` (o `fomento` foi pago direto ao fomento) |
| `titulos_vencidos` | foto de hoje: títulos `carteira`, `em_banco` ou `antecipado` com vencimento antes de hoje e saldo a receber |
| `contas_vencidas` | foto de hoje: contas a pagar `pendente_aprovacao` ou `aprovado` com vencimento antes de hoje (`valor_estimado`) |
| `estoque_baixo` | produtos ativos com `estoque_minimo > 0` e `quantidade_atual <= estoque_minimo`, do grupo (sem filtro de empresa), por nome |
| `saldos_bancarios` | os mesmos saldos de `GET /caixa/saldos` (M13); `saldo_total_centavos` soma as contas com saldo inicial |
| `endividamento` | o mesmo cálculo de `GET /dividas/resumo` (M14): saldo devedor das dívidas `ativa` por tipo de credor e por empresa, vencido × a vencer (30, 90 e 365 dias), pelo "hoje" do relatório |

As listas `itens` trazem no máximo **50** linhas (vencidos mais antigos primeiro); `total` e `por_empresa`
contam todas.

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
