# Escopo — Módulo de Estoque (Catálogo de Produtos)

> **Status:** escopo aprovado — pronto para implementação.
> **Objetivo:** produtos disponibilizados em estilo ecommerce, exibidos no site posteriormente.
> **Nome do módulo:** `estoque`

---

## 1. Visão geral

O módulo funciona como um **catálogo de produtos com controle de saldo**, não como
almoxarifado de obra. Os produtos cadastrados no painel serão exibidos no site
(frontend público) futuramente.

Decisões já tomadas:

| Tema | Decisão |
|---|---|
| Nome do módulo | **`estoque`** (pastas de domínio, rotas e telas) |
| Categorias | Árvore de **2 níveis** (categoria → subcategoria), **obrigatória** para o produto |
| Apontamento do produto | Sempre para **subcategoria** (nível 2) — categoria raiz só existe para agrupar |
| Escopo de estoque | **Centralizado** — sem centro de custo/obra (tabela de obras não existe) |
| Estilo | Ecommerce — produto com imagem, preço, descrição e disponibilidade |
| Preço | **Opcional** — aceita "sob consulta" (exibido como tal no site) |
| Saldo negativo | **Bloqueado sempre** — só entra no site quem tem saldo real |
| Primeira versão | Categoria + Produto + Movimentação de estoque |

---

## 2. Entidades

### 2.1 `categoria`

Árvore de 2 níveis. A categoria raiz não tem pai; a subcategoria tem `categoria_pai_id`.

| Campo | Tipo | Obrigatório | Observações |
|---|---|---|---|
| `id` | UUID | sim | `gen_random_uuid()` |
| `nome` | VARCHAR(120) | sim | |
| `categoria_pai_id` | UUID NULL | não | `NULL` = categoria raiz; preenchido = subcategoria |
| `slug` | VARCHAR(140) | sim | único, usado na URL pública do site |
| `ordem` | INT | não | ordem de exibição no menu |
| `icone` | VARCHAR(60) | não | nome do ícone (lucide) no site |
| `ativo` | BOOLEAN | sim | default `true` |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | sim | |

**Regras:**
- Máximo de 2 níveis: se `categoria_pai_id` for informado, o pai não pode ter pai.
- Não pode ser pai de si mesma.
- **Produto só pode apontar para subcategoria** (categoria com pai) — regra
  validada no usecase ao criar/editar produto.
- Único por `(categoria_pai_id, nome)` dentro do mesmo nível.
- Excluir categoria com produtos vinculados é proibido → usar `ativo = false`.

### 2.2 `produto` (item de estoque)

| Campo | Tipo | Obrigatório | Observações |
|---|---|---|---|
| `id` | UUID | sim | |
| `categoria_id` | UUID | **sim** | FK para subcategoria (nível 2) — validação no usecase |
| `nome` | VARCHAR(200) | sim | |
| `slug` | VARCHAR(220) | sim | único, URL pública |
| `descricao` | TEXT | não | descrição exibida no site |
| `codigo` | VARCHAR(60) | não | SKU/código interno (único quando informado) |
| `unidade_medida` | VARCHAR(10) | sim | enum: `un`, `m`, `m2`, `m3`, `kg`, `t`, `cx`, `sc`, `pct`, `lt` |
| `preco` | BIGINT NULL | não | em centavos (mesma convenção de `valor_estimado`); `NULL` = sob consulta |
| `preco_promocional` | BIGINT NULL | não | em centavos; `CHECK >= 0`; só faz sentido se `preco` informado |
| `quantidade_atual` | INT | sim | default 0, `CHECK >= 0` — **cache** das movimentações |
| `estoque_minimo` | INT | não | alimenta o alerta de reposição |
| `peso_kg` | NUMERIC(10,3) | não | opcional, útil p/ frete futuro |
| `destaque` | BOOLEAN | sim | default `false` — aparece em destaque no site |
| `ativo` | BOOLEAN | sim | default `true` — visível no site |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | sim | |

**Regras:**
- **Categoria é obrigatória** e deve ser de nível 2 (subcategoria).
- **`preco` opcional**: quando `NULL`, o produto é exibido no site como **"sob consulta"**.
- `quantidade_atual` **nunca é editada direto** — só via movimentação.
- `preco_promocional` (se informado) menor que `preco`; exige `preco` preenchido.
- Exclusão lógica (`ativo = false`), como no restante do sistema.

### 2.3 `produto_imagem`

Múltiplas imagens por produto (padrão ecommerce), reutilizando o módulo `arquivo`
que já existe (`tabela arquivo` + storage R2/disco).

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `produto_id` | UUID | FK |
| `arquivo_id` | UUID | FK para `arquivo` (já existe, migration 10) |
| `ordem` | INT | ordem de exibição |
| `alt` | VARCHAR(160) | texto alternativo (SEO/acessibilidade) |

### 2.4 `produto_movimento` — o coração do controle

Nada de editar saldo na mão: toda alteração gera movimento com trilha de auditoria
(mesmo conceito do `solicitacao_historico`).

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `produto_id` | UUID | FK |
| `tipo` | VARCHAR(12) | enum abaixo |
| `quantidade` | INT | `> 0` |
| `saldo_apos` | INT | saldo resultante, congelado no registro |
| `usuario_id` | UUID | quem registrou |
| `documento_ref` | VARCHAR(80) | nº da nota/pedido (opcional) |
| `observacao` | TEXT | |
| `criado_em` | TIMESTAMPTZ | |

**Tipos de movimento:**

| Tipo | Efeito no saldo |
|---|---|
| `entrada` | + (compra/recebimento) |
| `saida` | − (venda/consumo) |
| `ajuste` | +/− (inventário, correção; `quantidade` é o delta com sinal) |
| `perda` | − (avaria/vencimento) |
| `devolucao` | + (devolução de venda) |

**Regras:**
- `saida` e `perda` **nunca podem deixar o saldo negativo** — erro `domain.ErroValidacao`.
  Regra de ouro: só aparece no site quem tem saldo real.
- Movimento é **append-only**: não edita, não deleta.
- Atualização do saldo + gravação do movimento na **mesma transação** (`BeginTxx`).
- Estoque zerado não impede o produto de continuar ativo no painel, mas o site
  exibe `esgotado` (ou oculta, a decidir na tela pública).

---

## 3. Endpoints (padrão `API.md`)

Base: `/api/v1` — envelope `{ "dados": ... }` / `{ "erro": { codigo, mensagem } }`,
paginação `pagina`/`tamanho`. Nomes em português, como o módulo `solicitacao`.

### Categorias (admin)
```
POST   /api/v1/categorias              criar (raiz ou sub)
GET    /api/v1/categorias              listar árvore (autenticado)
GET    /api/v1/categorias/{id}
PUT    /api/v1/categorias/{id}
DELETE /api/v1/categorias/{id}         inativar se houver produtos
```

### Produtos
```
POST   /api/v1/produtos                criar (admin)
GET    /api/v1/produtos                lista paginada + filtros
GET    /api/v1/produtos/{id}
PUT    /api/v1/produtos/{id}           editar (admin)
DELETE /api/v1/produtos/{id}           inativar (admin)
POST   /api/v1/produtos/{id}/imagens   anexar imagem (reusa POST /api/v1/arquivos)
DELETE /api/v1/produtos/{id}/imagens/{imagemId}
```
Filtros de `GET /produtos`: `categoria_id`, `busca` (nome/código), `ativo`,
`estoque_baixo=true`, `destaque=true`, paginação.

### Movimentações
```
POST   /api/v1/produtos/{id}/movimentos   entrada | saida | ajuste | perda | devolucao
GET    /api/v1/produtos/{id}/movimentos   histórico paginado
GET    /api/v1/produtos/{id}/saldo        quantidade atual
```

### Painel / dashboard
```
GET    /api/v1/estoque/resumo    → total de produtos, valor em estoque,
                                   produtos abaixo do mínimo, últimos movimentos
```

### Público (sem JWT — exibir no site)
```
GET    /api/v1/publico/categorias        árvore de categorias ativas
GET    /api/v1/publico/produtos          só `ativo = true` E `quantidade_atual > 0`,
                                         com imagens; `?categoria=<slug>` aceita raiz
GET    /api/v1/publico/produtos/{slug}   detalhe por slug
GET    /api/v1/publico/destaques         destaque = true + ativo + estoque
GET    /api/v1/publico/imagens/{id}      binário da imagem (Content-Type original)
```
> **Filtro de saldo no site:** `quantidade_atual > 0` — quem não tem estoque real
> não aparece publicamente (regra 4), nem em detalhe (`404`): sem selo "esgotado".
> Registradas com `mux.HandleFunc` direto em `rotas.go`, fora dos wrappers
> `protegido(...)`/`administrador(...)` — o CORS continua valendo.

---

## 4. Estrutura de código (seguindo o padrão do módulo solicitação)

```
migrations/000018_create_tabela_categoria.up.sql|.down.sql
migrations/000019_create_tabela_produto.up.sql|.down.sql
migrations/000020_create_tabela_produto_imagem.up.sql|.down.sql
migrations/000021_create_tabela_produto_movimento.up.sql|.down.sql

internal/core/domain/estoque/          categoria.go, produto.go, unidade.go,
                                       preco.go, movimento.go, saldo.go
internal/core/ports/out/estoque/       categoria_repository.go,
                                       produto_repository.go, movimento_repository.go
internal/core/ports/in/estoque/        categoria_usecase.go, produto_usecase.go,
                                       movimento_usecase.go
internal/core/usecases/estoque/        categoria_usecase.go, produto_usecase.go,
                                       movimento_usecase.go, permissoes.go (+ testes/fakes)
internal/adapter/postgres/             categoria_repo.go, produto_repo.go,
                                       movimento_repo.go
internal/adapter/http/handlers/estoque/  categoria_handler.go, produto_handler.go,
                                       movimento_handler.go
internal/adapter/http/dto/estoque/     categoria.go, produto.go, movimento.go
cmd/api/main.go + internal/adapter/http/rotas.go   (registro)
```

Convenções herdadas: UUID `gen_random_uuid()`, `TIMESTAMPTZ`, valores em centavos
`BIGINT` com `CHECK > 0`, constraints nomeadas `pk_/fk_/chk_/uq_`, índices `idx_*`,
erros `domain.ErroValidacao` / `ErroNaoEncontrado`, FK `23503` mapeada em
`postgres/erros.go`.

---

## 5. Frontend (repo irmão `D:\PROJETOS\FRONT\painelMiaConstrutora`)

Já existe slot pronto: rota `/produtos` (stub `<div>Produtos Module</div>` em
`src/App.jsx`) e item "Produtos" no `src/components/Sidebar.jsx`.

Telas previstas:

| Tela | Arquivo sugerido | Conteúdo |
|---|---|---|
| Lista de produtos | `src/modules/produtos/Produtos.jsx` | tabela + filtros + badge de estoque baixo |
| Formulário | `src/modules/produtos/ProdutoFormModal.jsx` | dados, categoria (2 níveis), imagens |
| Detalhe | `src/modules/produtos/ProdutoDetalhe.jsx` | info + saldo + histórico de movimentos |
| Categorias | `src/modules/produtos/Categorias.jsx` | árvore com pai/filho |
| Movimentar estoque | `src/modules/produtos/MovimentoModal.jsx` | tipo + quantidade + observação |
| Serviço | `src/services/produtos.js` | espelha `src/services/solicitacoes.js` |

Menu: renomear "Produtos" para "Estoque" ou manter — decidir na implementação.

---

## 6. Permissões (inicial)

Usar wrappers existentes do `rotas.go`:

- `protegido(...)` — leitura (lista, detalhe, saldo, movimentos).
- `administrador(...)` — escrita (criar/editar produto, categoria, movimentar).
- Regras finas no usecase, estilo `usecases/solicitacao/permissoes.go`.

*Nota:* se for público no site, os endpoints `publico/*` ficam **sem JWT**.

---

## 7. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000018–000021 + domínio (categoria, produto, movimento) + testes de domínio | concluída |
| **2** | Repos Postgres (sqlx) + testes de integração (`test/helpers/banco.go`) | concluída |
| **3** | Usecases + permissões + testes unitários com fakes | concluída |
| **4** | Handlers/DTOs + registro em `rotas.go` e `main.go` + e2e (`test/e2e`) | concluída |
| **5** | Frontend: telas de categoria, produto e movimentação | pendente |
| **6** | Endpoint de resumo p/ dashboard | concluída |
| **7** | Imagens de produto + endpoints `publico/*` para exibir no site | concluída |

Backend fechado após a fase 7; o frontend (fase 5) é o próximo passo.

---

## 8. Decisões fechadas

- [x] **Nome do módulo**: `estoque`.
- [x] **Apontamento do produto**: sempre subcategoria (nível 2). Categoria raiz
      só agrupa; não recebe produto direto.
- [x] **Unidade de medida**: enum fechado (`un`, `m`, `m2`, `m3`, `kg`, `t`, `cx`,
      `sc`, `pct`, `lt`) validado no domínio.
- [x] **Preço**: opcional — `NULL` = "sob consulta".
- [x] **Estoque negativo**: bloquear sempre.
- [x] **Regra de exibição no site**: `ativo = true` **e** `quantidade_atual > 0`.
- [x] **Slug**: gerado no usecase a partir do nome, com unicidade (`409`) — e **não
      muda** na edição, preservando as URLs públicas.
- [x] **Imagens**: tabela própria `produto_imagem` (ordem + `alt`) apontando para a
      tabela `arquivo` e o storage existentes; upload reusa `POST /api/v1/arquivos`
      e o anexo vira `POST /api/v1/produtos/{id}/imagens` (admin).
- [x] **Rotas públicas**: registradas com `mux.HandleFunc` direto em `rotas.go`,
      fora dos wrappers de token; erros seguem o envelope padrão.
- [x] **Estorno**: **não** existe tipo `estorno`. O movimento é append-only com os 5
      tipos; desfazer registra o tipo inverso e o histórico permanece completo.
- [x] **Saldo zerado no site**: **ocultar** o produto (sem selo "esgotado"), inclusive
      no detalhe (`404`).
- [x] **`valor_estoque_centavos`**: preço cheio × saldo dos produtos ativos; promoção
      não conta e "sob consulta" fica de fora.
- [x] **`?categoria=<slug>` público**: slug resolve para id e **categoria raiz inclui
      suas subcategorias** (expansão em SQL, mesma regra do filtro do painel).

## 9. Decisões pendentes (abertos)

Sem abertos — todos os pontos da seção 8 foram fechados durante as fases 1 a 7.

---

## 10. Regras de código (obrigatórias)

1. **Nenhum comentário no código** — nem em linha, nem em bloco. O código deve se
   explicar por nomes claros de pacotes, tipos, funções e variáveis.
2. **Value objects** para conceitos com regra: `Preco`, `Saldo`, `Quantidade`,
   `UnidadeMedida`, `TipoMovimento`, `Slug`, seguindo o padrão já existente:
   ```go
   type Preco struct {
       centavos int64
   }

   func NovoPreco(centavos int64) (Preco, error)   // valida → domain.ErroValidacao
   func PrecoDe(centavos int64) Preco              // reconstrói (ex.: do banco)
   func (p Preco) Centavos() int64
   ```
   Referência: `internal/core/domain/solicitacao/valor.go` e `status.go`.
3. **Seguir a arquitetura atual**: `core/domain` → `core/ports` → `core/usecases`
   → `adapter/*`. Nada de lógica de negócio fora do domínio/usecase; nada de SQL
   fora do `adapter/postgres`.
4. **Erros do domínio**: apenas `domain.ErroValidacao` / `domain.ErroNaoEncontrado`.
5. **Semântica de nomes em português** no módulo novo (padrão do `solicitacao`),
   inclusive nos ports (`Criar`, `Obter`, `Listar`) — ver observação de
   nomenclatura em `PENDENCIAS.md`.
6. **Testes**: domínio e usecase unitários com fakes; integração via
   `test/helpers/banco.go`; e2e com tags `build e2e`.
7. **Formato e lint**: `gofmt` + `staticcheck` (CI já roda).
