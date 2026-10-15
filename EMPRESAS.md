# Escopo — M1 Empresas e filiais

> **Status:** concluído: API (fases 1 a 5) e frontend (fase 6; seletor de empresa no F0a, cadastro e acesso no front do M1).
> **Objetivo:** tudo o que é financeiro, fiscal ou de estoque passa a pertencer a uma empresa
> do grupo, e cada usuário só enxerga as empresas às quais tem acesso.
> **Nome do módulo:** `empresa`
> **Branch:** `feat/m1-empresas`

---

## 1. Visão geral

O grupo opera com quatro CNPJs (MIA / Gigante PB, Red Diamond matriz, Gigante CE, Gigante SP)
e hoje separa tudo à mão. O M1 cria o cadastro de empresas, o vínculo usuário × empresa e o
**padrão de escopo por empresa** que os módulos seguintes (M2 a M15) copiam. O primeiro
módulo existente a adotar o escopo é `solicitacao`.

Decisões já tomadas com o usuário:

| Tema | Decisão |
|---|---|
| Permissões | **Flags no cargo** continuam (`administrador`, `financeiro`, `comercial`); o M1 acrescenta só o escopo por empresa |
| Seed | **Nenhuma empresa** na migração; o administrador cadastra as quatro pela API/tela |
| Solicitações existentes | **Apagadas** na migração, junto com pagamentos, históricos e vínculos de anexo; `empresa_id` nasce obrigatório |
| Usuários existentes | Recebem **acesso a todas as empresas**; usuários novos nascem sem acesso e o administrador vincula |
| Administrador | Enxerga e opera todas as empresas, sempre |

---

## 2. Entidades

### 2.1 `empresa`

| Campo | Tipo | Obrigatório | Observações |
|---|---|---|---|
| `id` | UUID | sim | `gen_random_uuid()` |
| `razao_social` | VARCHAR(200) | sim | |
| `nome_fantasia` | VARCHAR(120) | sim | nome curto exibido nas telas |
| `cnpj` | CHAR(14) | sim | sem máscara (12 caracteres `0-9A-Z` + 2 dígitos verificadores), `uq_empresa_cnpj`, dígitos verificadores validados no domínio |
| `uf` | CHAR(2) | sim | enum fechado das 27 UFs |
| `regime_tributario` | VARCHAR(10) | sim | `simples`, `presumido`, `real` |
| `ativa` | BOOLEAN | sim | default `true` |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | sim | |

**Regras:**
- CNPJ aceito com ou sem máscara na entrada, numérico ou alfanumérico, gravado sem máscara, devolvido nos dois
  formatos (`cnpj` e `cnpj_formatado`). CNPJ com todos os dígitos iguais é inválido.
- CNPJ **não muda** depois de criado (é a identidade fiscal; corrigir = desativar e criar
  outra).
- Empresa **inativa** continua visível no histórico, mas não recebe registro novo
  (solicitação nova em empresa inativa → `400`).
- Exclusão só de empresa sem nenhum vínculo (FK `RESTRICT` → `400 registro em uso`); o
  caminho normal é desativar.

### 2.2 `usuario_empresa`

| Campo | Tipo | Observações |
|---|---|---|
| `usuario_id` | UUID | FK `usuario`, `ON DELETE CASCADE` |
| `empresa_id` | UUID | FK `empresa`, `ON DELETE RESTRICT` |
| `criado_em` | TIMESTAMPTZ | |

PK `(usuario_id, empresa_id)`; índice em `empresa_id`.

### 2.3 `usuario.acesso_todas_empresas`

Coluna nova `BOOLEAN NOT NULL DEFAULT FALSE`. A migração marca `TRUE` em todos os usuários
que já existem. Com `TRUE`, o usuário enxerga todas as empresas, inclusive as cadastradas
depois, sem precisar de vínculo.

### 2.4 Acesso do usuário (value object de domínio)

`empresa.Acesso` resume o que um usuário enxerga:

- `Todas()` — administrador ou `acesso_todas_empresas = true`;
- `Permite(empresaID) bool`;
- `EmpresaIDs() []uuid.UUID` — usado como filtro `IN (...)` nas consultas quando não é
  `Todas()`; lista vazia significa **nenhuma empresa** (a consulta devolve vazio, nunca
  "sem filtro").

É calculado em uma consulta só (`usuario` + `cargo` + `usuario_empresa`) pelo repositório
de empresas e entregue aos outros módulos por porta mínima (regra 6 do ROADMAP §3).

### 2.5 Adoção em `solicitacao`

- Coluna `empresa_id UUID NOT NULL`, FK `RESTRICT`, índice `(empresa_id, criado_em DESC)`.
- Criar exige `empresa_id`: a empresa precisa existir, estar ativa e estar no acesso do
  solicitante (`403` se não estiver).
- Visibilidade:
  - o **solicitante** continua vendo as próprias solicitações;
  - o **administrador** vê todas;
  - **aprovador** e **financeiro** só veem solicitações de empresas do seu acesso.
- Listagem: as filas `aprovacao` e `financeiro` passam a filtrar pelo acesso; todos os
  escopos aceitam `?empresa=<id>`; nas filas, empresa fora do acesso → `403`; em `minhas` o
  filtro vale sem checar o acesso, porque o solicitante sempre vê as próprias.
- Aprovar, rejeitar e registrar pagamento exigem acesso à empresa da solicitação.

---

## 3. Endpoints

### Empresas
```
POST   /api/v1/empresas                 criar (administrador)
GET    /api/v1/empresas                 empresas do acesso do usuário; ?ativas=true (autenticado)
GET    /api/v1/empresas/{id}            detalhe, se estiver no acesso (autenticado)
PUT    /api/v1/empresas/{id}            alterar razão social, nome fantasia, UF, regime (administrador)
PATCH  /api/v1/empresas/{id}/ativar     (administrador)
PATCH  /api/v1/empresas/{id}/desativar  (administrador)
DELETE /api/v1/empresas/{id}            só sem vínculos (administrador)
```

### Acesso do usuário às empresas (administrador)
```
GET    /api/v1/usuarios/{id}/empresas   { "acesso_todas": bool, "empresa_ids": [...] }
PUT    /api/v1/usuarios/{id}/empresas   substitui o conjunto inteiro (mesmo corpo)
```

### Solicitações (contrato alterado)
```
POST   /api/v1/solicitacoes             corpo ganha "empresa_id" (obrigatório)
GET    /api/v1/solicitacoes             ?empresa=<id> opcional em todos os escopos
```
Todas as respostas de solicitação passam a trazer `empresa_id`.

---

## 4. Padrão para os próximos módulos

O que M2 a M15 copiam para escopo por empresa:

1. A tabela do módulo tem `empresa_id UUID NOT NULL` com FK `RESTRICT` e índice que começa
   por `empresa_id`.
2. O usecase recebe por porta mínima (`ports/out/<modulo>/empresa_repository.go`, com
   `Obter` e `AcessoDoUsuario`) o `empresa.Acesso` do usuário; no `main.go` ela é
   satisfeita pelo `postgres.EmpresaRepository`. Referência: módulo `solicitacao`.
3. Criar/alterar: `acesso.Permite(empresaID)` senão `403`; empresa inativa → `400`.
4. Ler um registro: fora do acesso → `403` (mesmo tratamento de "quem pode ver").
5. Listar: o filtro do repositório recebe `EmpresaIDs` quando o acesso não é `Todas()`;
   lista vazia devolve vazio.
6. Teste obrigatório de isolamento: usecase com dois usuários em empresas diferentes, um
   não vê nem altera o registro do outro; e2e com o mesmo cenário pela API.

---

## 5. Estrutura de código

```
migrations/000025_create_tabela_empresa.up.sql|.down.sql
migrations/000026_create_tabela_usuario_empresa.up.sql|.down.sql   (+ usuario.acesso_todas_empresas)
migrations/000027_add_empresa_solicitacao.up.sql|.down.sql         (apaga solicitações; coluna NOT NULL)

internal/core/domain/empresa/            cnpj.go, uf.go, regime_tributario.go, empresa.go, acesso.go, vinculos.go
internal/core/ports/out/empresa/         empresa_repository.go, usuario_repository.go, cargo_repository.go
internal/core/ports/in/empresa/          empresa_usecase.go
internal/core/usecases/empresa/          empresa_usecase.go, permissoes.go (+ testes/fakes)
internal/adapter/postgres/               empresa_repo.go
internal/adapter/http/dto/empresa/       empresa.go, vinculos.go
internal/adapter/http/handlers/empresa/  empresa_handler.go
internal/core/ports/out/solicitacao/     empresa_repository.go (porta mínima: Obter + AcessoDoUsuario)
cmd/api/main.go + internal/adapter/http/rotas.go   (registro)
```

---

## 6. Permissões

- Escrita de empresas e de vínculos: `administrador(...)` em `rotas.go` e checagem no
  usecase (`permissoes.go`), como no estoque.
- Leitura de empresas: `protegido(...)`; o usecase filtra pelo acesso.
- Escopo por empresa nas solicitações: regras finas em `usecases/solicitacao/permissoes.go`.

---

## 7. Migração e deploy

A `000027` **apaga de forma irreversível**, no ambiente onde rodar:

- todos os `pagamento`;
- todo o `solicitacao_historico` e os vínculos `solicitacao_arquivo`;
- todas as `solicitacao`.

Os arquivos anexados continuam no storage e na tabela `arquivo`, sem vínculo (o dono pode
excluí-los). O `down` recria o schema anterior, mas **não restaura os dados**. Decisão
explícita do usuário (outubro/2026): o histórico de solicitações atual não precisa ser
preservado. Antes de rodar num ambiente com dados reais, confirmar que ainda vale.

Depois do deploy, o administrador cadastra as quatro empresas antes de qualquer solicitação
nova (sem empresa ativa no acesso, `POST /solicitacoes` responde `400`/`403`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000025–000026 + domínio (`CNPJ`, `UF`, `RegimeTributario`, `Empresa`, `Acesso`) + testes | concluída |
| **2** | Repositório de empresas e acesso + testes de integração | concluída |
| **3** | Usecases de empresa e de acesso do usuário + permissões + testes com fakes | concluída |
| **4** | Handlers/DTOs + rotas + e2e + `API.md` | concluída |
| **5** | Adoção em `solicitacao`: migration 000027, domínio, usecase, filtro, testes de isolamento, e2e, `API.md` | concluída |
| **6** | Frontend: cadastro de empresas, acesso por usuário, seletor de empresa na solicitação (depois da F0) | concluída |

---

## 9. Decisões fechadas

- [x] **Modelo de permissões**: flags no cargo + escopo por empresa (ROADMAP §6).
- [x] **Seed**: nenhum; cadastro pela API/tela.
- [x] **Solicitações antigas**: apagadas na migração.
- [x] **Usuários atuais**: acesso a todas as empresas; novos sem acesso.
- [x] **Administrador**: acesso total implícito, sem vínculo.

- [x] **Filiais**: não existe entidade `filial`; cada CNPJ, inclusive o de filial, é uma
      `empresa`. Se for preciso agrupar CNPJs, um campo `matriz_id` opcional entra depois.

## 10. Decisões pendentes

Sem abertos.
