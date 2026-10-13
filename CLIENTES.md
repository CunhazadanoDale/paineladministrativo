# Escopo — M3 Clientes e representantes

> **Status:** escopo aprovado — em implementação.
> **Objetivo:** tirar a "Base de clientes" da planilha: cadastro único de clientes do grupo,
> com obras, grupo econômico, regras comerciais tipadas e representante, base para pedidos
> (M6), NF-e (M8), contas a receber (M9) e comissões (M10).
> **Nome do módulo:** `cliente`
> **Branch:** `feat/m3-clientes`

---

## 1. Visão geral

Decisões já tomadas com o usuário (outubro/2026):

| Tema | Decisão |
|---|---|
| Representante | **Vínculo opcional a usuário**: cadastro próprio que pode apontar para um `usuario`; o representante com login vê só os próprios clientes (e, no M6/M10, os próprios pedidos e comissões) |
| Escopo do cliente | **Do grupo**: cadastro único, sem escopo por empresa; `empresa_relacionamento_id` é informativo. O escopo por empresa entra em pedidos (M6) e títulos (M9) |
| Permissões | **Comercial escreve** clientes, grupos e obras e converte lead; **administrador** gerencia representantes, importação e exclusões; demais autenticados leem |
| Importação | **Formato próprio** de CSV, documentado na `API.md` |

---

## 2. Entidades

### 2.1 `cliente_grupo`

Grupo econômico ou construtora que reúne vários CNPJs (ex.: as SPEs de uma construtora).

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `nome` | VARCHAR(120) | único sem diferenciar maiúsculas e acentos (`uq_cliente_grupo_nome`) |
| `ativo` | BOOLEAN | default `true` |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

### 2.2 `representante`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `nome` | VARCHAR(160) | obrigatório |
| `documento` | VARCHAR(14) NULL | CPF ou CNPJ sem máscara, dígitos conferidos; único quando informado |
| `usuario_id` | UUID NULL | FK `usuario` `RESTRICT`, único: um usuário é no máximo um representante |
| `email` / `telefone` | VARCHAR(255) / VARCHAR(40) | opcionais |
| `comissao_pronta_entrega_bp` | INT | percentual padrão em pontos-base (`250` = 2,50%), `0..10000` |
| `comissao_importacao_bp` | INT | idem, para venda por importação |
| `ativo` | BOOLEAN | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

Percentual nunca é `float`: value object `Percentual` em pontos-base (o M10 calcula a
comissão em centavos a partir dele). Representante inativo não recebe cliente novo.

### 2.3 `cliente`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `documento` | VARCHAR(14) | **obrigatório**; CPF (11 dígitos) ou CNPJ (numérico ou alfanumérico, como em `empresa`), sem máscara; `uq_cliente_documento` |
| `tipo_pessoa` | VARCHAR(2) | `pf`/`pj`, derivado do documento (não vem no corpo) |
| `nome` | VARCHAR(200) | razão social ou nome da pessoa |
| `nome_fantasia` | VARCHAR(120) NULL | |
| `nome_busca` | VARCHAR(200) | nome normalizado (minúsculas, sem acento, espaços colapsados), gerado no domínio; indexado para busca e alerta de duplicidade |
| `tipo` | VARCHAR(12) | `construtora`, `loja`, `pf`, `outro`; `pf` exige CPF e CPF exige `pf` |
| `grupo_id` | UUID NULL | FK `cliente_grupo` |
| `uf` / `cidade` | CHAR(2) / VARCHAR(120) | UF do enum de `empresa`; obrigatórios |
| `empresa_relacionamento_id` | UUID NULL | FK `empresa`; informativo |
| `representante_id` | UUID NULL | FK `representante`; precisa estar ativo ao atribuir |
| `email_nf` | VARCHAR(255) NULL | destinatário da NF-e (M8) |
| `telefone` | VARCHAR(40) NULL | |
| `observacoes` | VARCHAR(2000) | |
| `exige_numero_pedido_na_nf` | BOOLEAN | regra comercial tipada (M8) |
| `proibe_antecipacao` | BOOLEAN | regra comercial tipada (M9) |
| `lead_id` | UUID NULL | FK `lead` `RESTRICT`, único: lead convertido |
| `ativo` | BOOLEAN | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Regras:**
- Documento aceito com ou sem máscara, gravado sem máscara, devolvido nos dois formatos
  (`documento`, `documento_formatado`); **não muda** depois de criado (corrigir = desativar e
  criar outro, como na empresa).
- Duplicidade: documento repetido → `409 já existe um cliente com este documento`; nome
  parecido não bloqueia, mas a busca usa `nome_busca` (e o documento) para o comercial achar
  o cadastro antes de criar outro.
- Cliente inativo continua no histórico e não recebe pedido novo (M6).

### 2.4 `cliente_obra`

Obra ou SPE do cliente, com o endereço de entrega.

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `cliente_id` | UUID | FK `cliente` |
| `nome` | VARCHAR(160) | único por cliente sem diferenciar maiúsculas |
| `cnpj` | VARCHAR(14) NULL | CNPJ da SPE, quando houver |
| `logradouro`, `numero`, `complemento`, `bairro`, `cidade`, `uf`, `cep` | | endereço de entrega; `cep` com 8 dígitos; `logradouro`, `cidade` e `uf` obrigatórios |
| `ativo` | BOOLEAN | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

---

## 3. Endpoints

### Clientes
```
POST   /api/v1/clientes                       criar (comercial)
GET    /api/v1/clientes                       ?busca= (nome normalizado ou documento) ?grupo= ?representante= ?tipo= ?uf= ?ativo= ?lead= (autenticado)
GET    /api/v1/clientes/{id}                  detalhe com grupo e representante (autenticado)
PUT    /api/v1/clientes/{id}                  alterar, sem documento (comercial)
PATCH  /api/v1/clientes/{id}/ativar|desativar (comercial)
DELETE /api/v1/clientes/{id}                  só sem vínculos (administrador)
POST   /api/v1/clientes/importacao            CSV (administrador); ?simular=true valida sem gravar
```

### Obras
```
POST   /api/v1/clientes/{id}/obras            (comercial)
GET    /api/v1/clientes/{id}/obras            ?ativas= (autenticado)
PUT    /api/v1/obras/{id}                     (comercial)
PATCH  /api/v1/obras/{id}/ativar|desativar    (comercial)
```

### Grupos de clientes
```
POST   /api/v1/cliente-grupos                 (comercial)
GET    /api/v1/cliente-grupos                 ?busca= ?ativos= (autenticado)
GET    /api/v1/cliente-grupos/{id}
PUT    /api/v1/cliente-grupos/{id}            (comercial)
PATCH  /api/v1/cliente-grupos/{id}/ativar|desativar (comercial)
```

### Representantes
```
POST   /api/v1/representantes                 (administrador)
GET    /api/v1/representantes                 ?ativos= ?busca= (comercial)
GET    /api/v1/representantes/{id}            (comercial)
PUT    /api/v1/representantes/{id}            inclusive vínculo com usuário e percentuais (administrador)
PATCH  /api/v1/representantes/{id}/ativar|desativar (administrador)
```

### Lead → cliente
```
POST   /api/v1/leads/{id}/converter           (comercial)
```
Corpo igual ao de criar cliente; `nome`, `email_nf` e `telefone` vazios são preenchidos com os
dados do lead. Cria o cliente com `lead_id`; lead já convertido → `409`. O lead não muda (o
histórico de leads só registra troca de etapa); `GET /clientes?lead=<id>` mostra o cliente
de um lead. Lead convertido não pode ser excluído (`400 registro em uso`).

### Importação (`text/csv`, UTF-8, separador `;`)

Cabeçalho fixo:
```
documento;nome;nome_fantasia;tipo;grupo;uf;cidade;email_nf;telefone;representante_documento;exige_numero_pedido_na_nf;proibe_antecipacao;observacoes
```
- Linha válida é gravada; linha inválida volta no relatório com número e motivo; nada é
  sobrescrito (documento já cadastrado → linha rejeitada).
- `grupo` pelo nome (criado se não existir); representante pelo documento (precisa existir e
  estar ativo); flags `sim`/`nao`.
- Resposta: `{ "total": n, "importadas": n, "rejeitadas": [{ "linha": 7, "documento": "…", "motivo": "…" }] }`.
- Limite de 5 MB e 5.000 linhas por envio; `?simular=true` devolve o mesmo relatório sem gravar.

---

## 4. Permissões

- Perfis existentes, sem flag nova: `comercial(...)` (comercial ou administrador) para escrita
  de clientes, grupos e obras e conversão de lead; `administrador(...)` para representantes,
  importação e exclusão; `protegido(...)` para leitura de clientes, grupos e obras; leitura de
  representantes em `comercial(...)` (traz contato e percentuais).
- **Representante com login**: usuário vinculado a um representante ativo, que não seja
  administrador, só vê e altera os clientes em que é o representante (listagem filtrada,
  detalhe e obras de outro → `403`) e não troca o representante do cliente. Regras finas em
  `usecases/cliente/permissoes.go`.
- Sem escopo por empresa (decisão da seção 1).

---

## 5. Estrutura de código

```
migrations/000034_create_tabela_representante_cliente_grupo
migrations/000035_create_tabela_cliente
migrations/000036_create_tabela_cliente_obra

internal/core/domain/cliente/         documento.go (CPF + CNPJ de domain/empresa), percentual.go,
                                      tipo_cliente.go, texto.go (nome de busca, textos e e-mail),
                                      grupo.go, representante.go, cliente.go, obra.go,
                                      importacao.go (linha e relatório, fase 5)
internal/core/ports/out/cliente/      cliente_repository.go, grupo_repository.go, obra_repository.go,
                                      representante_repository.go, lead_repository.go (porta
                                      mínima: Obter), usuario_repository.go, cargo_repository.go
internal/core/ports/in/cliente/       cliente_usecase.go, grupo_usecase.go, obra_usecase.go,
                                      representante_usecase.go, importacao_usecase.go
internal/core/usecases/cliente/       (+ permissoes.go, testes e fakes)
internal/adapter/postgres/            cliente_repo.go, cliente_grupo_repo.go, cliente_obra_repo.go,
                                      representante_repo.go
internal/adapter/http/dto/cliente/ + handlers/cliente/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
```

A conversão de lead fica no usecase de clientes, com porta mínima de leitura de lead; o
pacote `lead` não muda.

---

## 6. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000034–000036 + domínio (documento, percentual, tipo, nome de busca, grupo, representante, cliente, obra) + testes | concluída |
| **2** | Repositórios + testes de integração (unicidade, busca normalizada, filtros) | pendente |
| **3** | Usecases + permissões (comercial, administrador, representante com login) + testes com fakes | pendente |
| **4** | Handlers/DTOs + rotas + e2e + `API.md` | pendente |
| **5** | Conversão de lead + importação CSV (com simulação) + testes + `API.md` | pendente |
| **6** | Frontend: clientes, obras, grupos, representantes, conversão e importação | pendente |

O M3 não tem dinheiro nem data de fato (percentual é pontos-base), então a fase 6 não depende
da F0.

---

## 7. Decisões fechadas

- [x] **Representante é usuário?** Vínculo opcional; com login vê só os próprios clientes.
- [x] **Cliente do grupo**, sem escopo por empresa.
- [x] **Comercial escreve**; administrador gerencia representantes, importação e exclusões.
- [x] **Importação em formato próprio**, com relatório e simulação.
- [x] **Documento obrigatório** no cliente: é a chave para resolver o cliente da NF-e (M8) e
      evitar duplicidade.
- [x] **Percentual em pontos-base** (`INT`), nunca `float`.
- [x] **Conversão de lead** por `cliente.lead_id` único; o pacote `lead` não muda.

## 8. Decisões pendentes

Sem abertos.
