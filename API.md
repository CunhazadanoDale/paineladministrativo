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
| Financeiro | Token de um usuário cujo cargo tem `"financeiro": true` (ou de administrador) | `escopo=financeiro`, o registro de pagamento das solicitações aprovadas, a escrita de fornecedores e do plano de contas, e as contas bancárias das empresas do seu acesso |

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
solicitações fora de `pendente_aprovacao`). Aprovador e financeiro só enxergam solicitações
de empresas do seu **acesso** (seção "Empresas"); o solicitante sempre vê as próprias. Quem
não pode ver recebe `403` (`perfil sem permissão para consultar esta solicitação`).
Aprovar, rejeitar e registrar pagamento de solicitação de empresa fora do acesso também
respondem `403` (`perfil sem acesso a esta empresa`).

### Escopos de listagem

`GET /api/v1/solicitacoes` aceita `pagina`, `tamanho`, `escopo`, `status` e `empresa`:

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

### `POST /api/v1/solicitacoes`

```json
{
  "empresa_id": "uuid-da-empresa",
  "valor_centavos": 150000,
  "prazo_pagamento": "2026-10-20",
  "observacao": "Compra de cimento para a fase 2",
  "forma_pagamento": "pix",
  "arquivo_ids": ["3f1c2a4e-0000-4000-8000-000000000001"]
}
```

| Campo | Regra |
| --- | --- |
| `empresa_id` | obrigatório; a empresa precisa existir, estar ativa e estar no acesso do solicitante |
| `valor_centavos` | inteiro em centavos, maior que zero (`150000` = R$ 1.500,00) |
| `prazo_pagamento` | `AAAA-MM-DD` (ou RFC3339). Obrigatório e não pode ser anterior a hoje |
| `observacao` | obrigatória, no máximo 1000 caracteres |
| `forma_pagamento` | `pix`, `cartao` ou `boleto` |
| `arquivo_ids` | opcional. Cada anexo tem que existir, pertencer ao solicitante, aparecer uma vez só e não estar em outra solicitação |

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
| Mesmo anexo duas vezes | `400` | `erro de validação: arquivo repetido na solicitação` |
| Sem `empresa_id` | `400` | `erro de validação: empresa da solicitação não informada` |
| Empresa inexistente (solicitante com acesso a todas) | `400` | `erro de validação: empresa da solicitação não encontrada` |
| Empresa fora do acesso, inclusive inexistente | `403` | `perfil sem acesso a esta empresa` |
| Empresa inativa | `400` | `erro de validação: empresa inativa não recebe solicitação nova` |

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
| `pago_em` | opcional, RFC3339. Padrão: agora (UTC). Não pode estar no futuro (`400 erro de validação: data do pagamento não pode estar no futuro`). É a data em que o dinheiro saiu; `criado_em` do pagamento e `atualizado_em` da solicitação registram o momento do lançamento |

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
| Arquivo ausente no armazenamento | `404` `registro não encontrado: arquivo não encontrado no armazenamento` (registro existe, mas o conteúdo sumiu do disco ou do bucket) |
| Tempo de transferência | até 5 minutos no envio (`POST /api/v1/arquivos`) e no download (`GET /api/v1/arquivos/{id}` e `GET /api/v1/publico/imagens/{id}`); as demais rotas têm 15 segundos para ler a requisição e escrever a resposta |
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

Corpo: o arquivo CSV cru (`Content-Type: text/csv`), **UTF-8** (com ou sem BOM; arquivo em outra
codificação, como o CSV padrão do Excel em Windows-1252, é recusado inteiro com
`400 linha N não está em UTF-8; salve o arquivo como CSV UTF-8`), separador `;`,
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
cancela só a própria proposta.

| Método | Rota | Perfil | Sucesso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/pedidos` | **comercial** | `201` proposta |
| `GET` | `/api/v1/pedidos` | autenticado | `200` lista paginada sem itens (`?empresa=`, `?cliente=`, `?representante=`, `?status=`, `?modelo=`, `?numero=`) |
| `GET` | `/api/v1/pedidos/{id}` | autenticado | `200` detalhe com itens, reservado e falta cobrir por item e reservas ativas |
| `PUT` | `/api/v1/pedidos/{id}` | **comercial** | `200`; só em `proposta`; corpo do `POST` sem `empresa_id` e com `versao` |
| `POST` | `/api/v1/pedidos/{id}/confirmar` | **comercial** | `200`; corpo `{ "versao": n }` |
| `POST` | `/api/v1/pedidos/{id}/entrada` | **financeiro** | `200`; corpo `{ "versao", "pago_em", "valor_centavos" }` |
| `POST` | `/api/v1/pedidos/{id}/reservar-pendencias` | **comercial** | `200`; corpo `{ "versao": n }` |
| `POST` | `/api/v1/pedidos/{id}/cancelar` | **comercial** | `200`; corpo `{ "versao", "motivo", "valor_a_devolver_centavos" }` |
| `GET` | `/api/v1/pedidos/{id}/historico` | autenticado | `200` lista (`{ "dados": [...] }`), mais antigo primeiro |
| `POST` | `/api/v1/pedidos/{id}/entregas` | **comercial** | `201` entrega `pendente` |
| `GET` | `/api/v1/pedidos/{id}/entregas` | autenticado | `200` lista (`{ "dados": [...] }`) |
| `GET` | `/api/v1/entregas` | autenticado | `200` lista paginada (`?empresa=`, `?status=`, `?previsao_ate=`) |
| `GET` | `/api/v1/entregas/{id}` | autenticado | `200` |
| `POST` | `/api/v1/entregas/{id}/saiu` | **comercial** | `200`; corpo `{ "versao", "saiu_em" }`; baixa o estoque |
| `POST` | `/api/v1/entregas/{id}/entregue` | **comercial** | `200`; corpo `{ "versao", "entregue_em" }` |
| `POST` | `/api/v1/entregas/{id}/cancelar` | **comercial** | `200`; corpo `{ "versao": n }`; só `pendente` |

Corpo do `POST /pedidos`:

```json
{
  "empresa_id": "…",
  "cliente_id": "…",
  "obra_id": "…",
  "representante_id": "…",
  "modelo": "pronta_entrega",
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
| `modelo` | `pronta_entrega` ou `importacao`; define a condição gravada no pedido |
| `prazo_limite` | data `AAAA-MM-DD`, opcional, só na pronta entrega; na importação é calculado pela entrada paga (`400`) |
| `numero_pedido_cliente` | opcional, até 60 (número de compra do cliente) |
| `itens` | pelo menos um; produto ativo ao incluir e sem repetir no pedido; `quantidade > 0`; `preco_unitario_centavos >= 0` (o front sugere `produto.preco`) |

Condição gravada pelo modelo, devolvida em `condicao`:

| `modelo` | `entrada_bp` | `prazo_boleto_dias` | `prazo_entrega_dias` |
| --- | --- | --- | --- |
| `pronta_entrega` | `0` | `30` | `null` |
| `importacao` | `3000` (30%) | `null` | `120`, contados da entrada paga |

A resposta traz `numero` (sequencial por empresa), `status`, `valor_total_centavos`, `condicao`,
`prazo_limite`, `entrada_paga_em`, `entrada_paga_centavos`, `cancelado_motivo`,
`valor_a_devolver_centavos`, `versao` e `itens` (`subtotal_centavos`, `quantidade_entregue` e
`restante`). No `GET /pedidos/{id}` cada item traz também `reservado` e `falta_cobrir`, e o pedido
traz `reservas` (mesmo formato de `GET /produtos/{id}/reservas`, com `origem = pedido_item`).
Datas só-data (`prazo_limite`, `entrada_paga_em`, `previsao`, `saiu_em`, `entregue_em`) chegam como
meia-noite UTC e são aceitas como `AAAA-MM-DD`.

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
disponível e o que sobrar das reservas é reservado de novo. Sem disponível no local → `400 saldo
disponível insuficiente no local`. `entregue` (data padrão: hoje, não antes de `saiu_em`) só
registra a data.

### Versão (edição concorrente)

Pedido e entrega têm `versao` própria, com o mesmo contrato dos fornecedores (`400` sem versão,
`409 o pedido foi alterado por outra pessoa; recarregue e tente de novo` ou `409 a entrega foi
alterada por outra pessoa; ...`). **Toda operação de entrega também avança a `versao` do pedido**:
recarregue o pedido depois de criar, dar saída, concluir ou cancelar uma entrega. Mudança de
estoque concorrente (outra reserva ou movimento no mesmo produto) responde `409 estoque do produto
alterado por outra operação, recarregue e tente novamente`, e nada é gravado.

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
