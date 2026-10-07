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

Use `.env.example` como ponto de partida: copie para `.env` e preencha o `JWT_SECRET`
(gerar um: `openssl rand -hex 32`).

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
| Autenticado | Token válido de usuário ativo | Leads, funis, etapas, histórico e **leitura** de cargos |
| Administrador | Token de um usuário cujo cargo tem `"administrador": true` | Tudo o que o perfil autenticado pode, **mais** a gestão de usuários e a escrita de cargos |

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

Aceitam paginação: usuários, cargos, leads, funis e histórico de movimentação.

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
| `403` | Usuário autenticado sem o perfil administrador |
| `404` | Recurso não encontrado ou credenciais de login inválidas |
| `405` | Método não permitido na rota (resposta em texto puro do roteador, com cabeçalho `Allow`) |
| `500` | Erro interno — mensagem fixa `erro interno do servidor` |
| `503` | `/health/db` sem conexão com o banco |

Observações:

- Campos desconhecidos no corpo rejeitam a requisição (`400` com `json: unknown field`)
- IDs que não forem UUID devolvem `400` (`parâmetro id inválido`)
- Mensagens iniciadas com `erro de validação:` ou `registro não encontrado:` vêm das
  regras de negócio

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
