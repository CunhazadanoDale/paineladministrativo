# Escopo — M2 Estoque por local, lote e reserva

> **Status:** escopo aprovado — em implementação.
> **Objetivo:** o saldo de cada produto passa a existir por local de estoque, cada local
> pertence a uma empresa, toda entrada vira lote com custo e toda saída consome lote por
> FIFO; reservas separam o que já está comprometido do que está disponível.
> **Nome do módulo:** `estoque` (evolução do módulo existente, mesmo pacote)
> **Branch:** `feat/m2-estoque-locais`

---

## 1. Visão geral

Hoje o produto tem um saldo único (`produto.quantidade_atual`) e o grupo controla à mão,
nas planilhas "Estoque geral" e "Pendências de envio", onde está cada material, de qual
empresa ele é e o que já foi vendido sem ter saído. O M2 traz isso para o sistema:

- **produto continua do grupo**: catálogo, preço de venda, vitrine e categoria não mudam
  e não ganham empresa;
- **o saldo é que tem dono**: o local de estoque pertence a uma empresa (padrão da seção 4
  do `EMPRESAS.md`), e o saldo por local é o que cada usuário enxerga conforme o acesso;
- **custo nasce no lote**: entrada informa custo unitário; saída consome os lotes mais
  antigos e guarda quanto custou, base do CMV do M15;
- **reserva**: separa quantidade comprometida; disponível = saldo − reservado. O M6
  (pedidos) passa a criar e baixar reservas pelo port; até lá, a reserva é manual.

`produto.quantidade_atual` continua existindo como **soma** de todos os locais mais o
saldo ainda não alocado (seção 7), atualizada na mesma transação de cada operação.

Decisões já tomadas por boa prática (detalhe na seção 9):

| Tema | Decisão |
|---|---|
| Concorrência | `produto.versao_estoque` com guarda otimista: toda operação de estoque do produto lê a versão, grava com `WHERE versao_estoque = :lida` e incrementa; zero linhas → `409` |
| Ordem FIFO | `entrada_em` do lote, desempate pelo `id` |
| Transferência | não muda o total do produto; leva o custo e a data de entrada do lote de origem |
| Histórico | `produto_movimento` continua append-only; consumo de lote em `produto_movimento_lote`, também append-only |
| Movimentos antigos | ficam sem local (`local_id NULL`); a constraint nova vale só para linhas novas (`NOT VALID`) |

---

## 2. Entidades

### 2.1 `estoque_local`

| Campo | Tipo | Obrigatório | Observações |
|---|---|---|---|
| `id` | UUID | sim | `gen_random_uuid()` |
| `empresa_id` | UUID | sim | FK `empresa`, `RESTRICT` |
| `nome` | VARCHAR(120) | sim | ex.: "Galpão Cabedelo", "Loja João Pessoa" |
| `ativo` | BOOLEAN | sim | default `true` |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | sim | |

**Regras:**
- `uq_estoque_local_empresa_nome` em `(empresa_id, lower(nome))`; índice
  `idx_estoque_local_empresa` em `(empresa_id, nome)`.
- Empresa do local precisa estar ativa e no acesso de quem cria (`403`/`400`).
- A empresa do local **não muda** depois de criado (o saldo e os lotes são dela).
- Local **inativo** não recebe entrada, implantação nem transferência de destino; saídas
  continuam para esvaziá-lo. Desativar local com saldo → `400`.
- Exclusão só sem nenhum vínculo (FK `RESTRICT` → `400 registro em uso`).

### 2.2 `produto_saldo_local`

| Campo | Tipo | Observações |
|---|---|---|
| `produto_id` | UUID | FK `produto`, `RESTRICT` |
| `local_id` | UUID | FK `estoque_local`, `RESTRICT` |
| `quantidade` | INT | `CHECK >= 0` |
| `reservado` | INT | `CHECK >= 0 AND reservado <= quantidade` |
| `atualizado_em` | TIMESTAMPTZ | |

PK `(produto_id, local_id)`; índice `idx_produto_saldo_local_local` em `(local_id)`.
A linha nasce na primeira operação do produto no local e nunca é apagada (fica zerada).

### 2.3 Colunas novas em `produto`

| Campo | Tipo | Observações |
|---|---|---|
| `quantidade_reservada` | INT | default 0, `CHECK >= 0 AND quantidade_reservada <= quantidade_atual`; soma das reservas ativas |
| `versao_estoque` | BIGINT | default 0; guarda otimista de toda operação de estoque |

Disponível do produto = `quantidade_atual − quantidade_reservada`. Não alocado =
`quantidade_atual − SUM(produto_saldo_local.quantidade)`.

### 2.4 `produto_lote`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `produto_id` | UUID | FK `produto`, `RESTRICT` |
| `local_id` | UUID | FK `estoque_local`, `RESTRICT` |
| `movimento_id` | UUID | FK `produto_movimento`: o movimento que criou o lote |
| `lote_origem_id` | UUID NULL | FK `produto_lote`: preenchido quando o lote nasceu de transferência |
| `custo_unitario_centavos` | BIGINT NULL | `CHECK >= 0`; `NULL` = custo desconhecido |
| `quantidade_inicial` | INT | `CHECK > 0` |
| `quantidade_atual` | INT | `CHECK >= 0 AND quantidade_atual <= quantidade_inicial` |
| `entrada_em` | TIMESTAMPTZ | idade do lote para o FIFO; na transferência, herda a do lote de origem |
| `criado_em` | TIMESTAMPTZ | |

Índice `idx_produto_lote_fifo` em `(produto_id, local_id, entrada_em, id)
WHERE quantidade_atual > 0`.

**Regras:**
- Lote só diminui por consumo de movimento; nunca é editado nem apagado.
- `custo_unitario_centavos` em centavos, nunca `float`. Custo zero é permitido
  (bonificação); custo desconhecido é `NULL` e aparece como tal no CMV.

### 2.5 `produto_movimento` (colunas novas)

| Campo | Tipo | Observações |
|---|---|---|
| `local_id` | UUID NULL | FK `estoque_local`; obrigatório em linha nova (`chk_produto_movimento_local ... NOT VALID`) |
| `local_destino_id` | UUID NULL | só na `transferencia`; diferente de `local_id` |
| `saldo_local_apos` | INT NULL | saldo do local (de origem, na transferência) depois do movimento |
| `custo_unitario_centavos` | BIGINT NULL | custo informado nos tipos que criam lote |
| `reserva_id` | UUID NULL | saída que baixa uma reserva (fase 5) |

`saldo_apos` continua sendo o total do produto. Tipos novos no `CHECK`:

| Tipo | Efeito | Lote |
|---|---|---|
| `entrada` | + no local | cria lote com o custo informado (**obrigatório**) |
| `devolucao` | + no local | cria lote; custo opcional (`NULL` = desconhecido) |
| `ajuste` positivo | + no local | cria lote; custo opcional |
| `ajuste` negativo | − no local | consome FIFO |
| `saida` / `perda` | − no local | consome FIFO; limitado ao **disponível** do local, salvo `saida` com `reserva_id` |
| `implantacao` | + no local, total não muda | tira do não alocado e cria lote; custo opcional |
| `transferencia` | − origem, + destino, total não muda | consome FIFO na origem e cria lotes no destino com o mesmo custo e `entrada_em` |

### 2.6 `produto_movimento_lote`

| Campo | Tipo | Observações |
|---|---|---|
| `movimento_id` | UUID | FK `produto_movimento` |
| `lote_id` | UUID | FK `produto_lote` |
| `quantidade` | INT | `CHECK > 0` |
| `custo_unitario_centavos` | BIGINT NULL | cópia do custo do lote no momento do consumo |

PK `(movimento_id, lote_id)`. É de onde o M15 tira o CMV:
`SUM(quantidade × custo_unitario_centavos)` das saídas do mês.

### 2.7 `produto_reserva` (fase 5)

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `produto_id` / `local_id` | UUID | FKs `RESTRICT` |
| `quantidade` | INT | `CHECK > 0` |
| `origem` | VARCHAR(20) | `manual` agora; `pedido_item` entra no M6 |
| `origem_id` | UUID NULL | id do item de pedido (M6); sem FK até lá |
| `status` | VARCHAR(10) | `ativa` → `baixada` / `liberada` |
| `usuario_id` | UUID | quem reservou |
| `observacao` | VARCHAR(1000) | |
| `criado_em` / `encerrada_em` | TIMESTAMPTZ | |

**Regras:**
- Reservar exige disponível no local; a reserva soma em `produto_saldo_local.reservado` e
  `produto.quantidade_reservada` na mesma transação.
- Reserva é integral: baixa (saída com `reserva_id`, quantidade igual à reservada) ou
  liberação devolvem o reservado; transição com guarda de status (`409` se já encerrada).
- Para o M6: port `ReservaRepository` com `Reservar`, `Liberar` e `Baixar`, e consulta de
  disponível por produto e local.

---

## 3. Endpoints

### Locais
```
POST   /api/v1/estoque/locais                criar (administrador)
GET    /api/v1/estoque/locais                locais das empresas do acesso; ?empresa= ?ativos= (autenticado)
GET    /api/v1/estoque/locais/{id}           detalhe, se a empresa estiver no acesso
PUT    /api/v1/estoque/locais/{id}           alterar nome (administrador)
PATCH  /api/v1/estoque/locais/{id}/ativar    (administrador)
PATCH  /api/v1/estoque/locais/{id}/desativar (administrador; 400 se houver saldo)
DELETE /api/v1/estoque/locais/{id}           só sem vínculos (administrador)
GET    /api/v1/estoque/locais/{id}/saldos    produtos com saldo no local, paginado
```

### Saldo e lotes do produto
```
GET    /api/v1/produtos/{id}/saldo           contrato ampliado (abaixo)
GET    /api/v1/produtos/{id}/lotes           ?local= ?com_saldo=true (administrador ou financeiro)
```

`/saldo` mantém `produto_id` e `saldo` e acrescenta `reservado`, `disponivel`,
`nao_alocado` e `locais: [{local_id, local_nome, empresa_id, quantidade, reservado,
disponivel}]`, só com os locais das empresas do acesso. Os totais do produto são do grupo,
como na vitrine.

### Movimentos
```
POST   /api/v1/produtos/{id}/movimentos      corpo ganha local_id (obrigatório), custo_unitario_centavos, reserva_id
GET    /api/v1/produtos/{id}/movimentos      ?tipo= ?local= ; só movimentos de locais do acesso (e os antigos, sem local)
POST   /api/v1/produtos/{id}/transferencias  { local_origem_id, local_destino_id, quantidade, documento_ref, observacao }
```

`implantacao` usa a rota de movimentos (`tipo: "implantacao"`), limitada ao não alocado.
O movimento devolvido passa a trazer `local_id`, `local_destino_id`, `saldo_local_apos`,
`custo_unitario_centavos` e os lotes consumidos.

### Reservas (fase 5)
```
POST   /api/v1/produtos/{id}/reservas        { local_id, quantidade, observacao } (administrador)
GET    /api/v1/produtos/{id}/reservas        ?status= ?local= (autenticado, filtrado pelo acesso)
POST   /api/v1/reservas/{id}/liberar         (administrador)
```

### Resumo e vitrine
- `GET /api/v1/estoque/resumo`: `ultimos_movimentos` passa a respeitar o acesso; aceita
  `?empresa=`. Os totais de produto não mudam.
- Rotas `publico/*`: **contrato igual**; o filtro de saldo vira
  `quantidade_atual - quantidade_reservada > 0` (lista, detalhe, destaques e imagem).
- `GET /api/v1/produtos?estoque_baixo=true` e o `estoque_baixo` do objeto continuam pelo
  total do produto.

---

## 4. Permissões e escopo por empresa

Segue o padrão da seção 4 do `EMPRESAS.md`, com o local como portador da empresa:

- Escrita (locais, movimentos, transferência, implantação, reservas): `administrador(...)`
  em `rotas.go`, como hoje, e `acesso.Permite(local.EmpresaID)` no usecase.
- Leitura de locais, saldo por local, movimentos e reservas: `protegido(...)`; o usecase
  filtra pelos locais das empresas do acesso (lista vazia → vazio). Local fora do acesso
  em `?local=` ou no detalhe → `403`.
- Lotes (expõem custo): administrador ou financeiro, e só dos locais do acesso.
- Porta mínima `ports/out/estoque/empresa_repository.go` (`Obter` + `AcessoDoUsuario`),
  satisfeita pelo `postgres.EmpresaRepository` no `main.go`.
- Teste de isolamento obrigatório: dois usuários em empresas diferentes, um não vê nem
  movimenta o local do outro, no usecase e no e2e.

---

## 5. Concorrência e transação

Toda operação de estoque de um produto (movimento, transferência, implantação, reserva):

1. o usecase carrega o produto (com `versao_estoque`), o saldo dos locais envolvidos e os
   lotes com saldo, em ordem FIFO;
2. o domínio calcula o resultado (saldos novos, lotes consumidos e criados, movimento);
3. o repositório grava tudo numa transação, começando por
   `UPDATE produto ... SET versao_estoque = versao_estoque + 1 WHERE id = :id AND
   versao_estoque = :lida`; zero linhas → `409 estoque do produto alterado por outra
   operação, recarregue e tente novamente`. Erros passam por `tratarErroDeGravacao`,
   inclusive o `Commit`.

Assim operações simultâneas no mesmo produto nunca gravam sobre um estado velho, e
produtos diferentes não se bloqueiam. Detalhes:

- a leitura do passo 1 roda numa transação `REPEATABLE READ` somente leitura, para produto,
  saldos, lotes e total alocado saírem do mesmo instante;
- cada lote consumido é baixado com `WHERE quantidade_atual >= :consumo` (rede de
  segurança além da versão) e o consumo vai para `produto_movimento_lote`;
- os locais que recebem lote são lidos com `FOR SHARE` e precisam estar ativos; a
  desativação trava o local com `FOR UPDATE` e só desativa se não houver saldo, então uma
  entrada e uma desativação simultâneas nunca deixam saldo em local inativo.

---

## 6. Estrutura de código

```
migrations/000028_create_tabela_estoque_local
migrations/000029_create_tabela_produto_saldo_local        (+ produto.quantidade_reservada, produto.versao_estoque)
migrations/000030_add_local_produto_movimento               (colunas e tipos)
migrations/000031_create_tabela_produto_lote                (+ produto_movimento_lote)
migrations/000032_add_chk_produto_movimento_local           (CHECK NOT VALID, fase 3)
migrations/000033_create_tabela_produto_reserva             (fase 5)

internal/core/domain/estoque/        local.go, saldo_local.go, lote.go, custo.go, fifo.go,
                                     operacao.go (cálculo de movimento/transferência), reserva.go
internal/core/ports/out/estoque/     local_repository.go, estoque_repository.go (leitura do
                                     estado + gravação da operação), lote_repository.go,
                                     reserva_repository.go, empresa_repository.go
internal/core/ports/in/estoque/      local_usecase.go, movimento_usecase.go (ampliado),
                                     reserva_usecase.go
internal/core/usecases/estoque/      local_usecase.go, movimento_usecase.go, reserva_usecase.go,
                                     permissoes.go (acesso por empresa)
internal/adapter/postgres/           local_repo.go, estoque_repo.go, lote_repo.go, reserva_repo.go
internal/adapter/http/dto/estoque/   local.go, saldo.go, lote.go, movimento.go, reserva.go
internal/adapter/http/handlers/estoque/  local_handler.go, lote_handler.go, reserva_handler.go,
                                     movimento_handler.go (ampliado)
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
```

---

## 7. Migração e deploy

- Nenhuma empresa, local ou lote é criado pela migration. O saldo atual de cada produto
  continua em `quantidade_atual` e passa a aparecer como **não alocado**.
- Depois do deploy, o administrador cria os locais e registra uma `implantacao` por
  produto e local (com custo, se souber). Até lá, a vitrine continua mostrando o produto,
  porque ela olha o total menos o reservado.
- Saída, perda, ajuste negativo, transferência e reserva só operam em saldo alocado.
- Movimentos antigos ficam com `local_id NULL` e aparecem no histórico do produto como
  anteriores aos locais.
- `down` remove colunas e tabelas na ordem inversa. Ele **não** devolve ao produto o saldo
  movimentado depois do deploy (o total já está em `quantidade_atual`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000028–000031 + domínio (local, saldo por local, lote, FIFO, custo, operação de movimento/transferência/implantação) + testes | concluída |
| **2** | Repositórios (locais, leitura do estado, gravação transacional com guarda, lotes) + testes de integração, inclusive concorrência | concluída |
| **3** | Usecases de locais, saldo por local, lotes e transferência + escopo por empresa na listagem de movimentos e no resumo + testes com fakes e de isolamento | concluída |
| **4** | Handlers/DTOs + rotas novas + e2e + `API.md`; depois a virada da movimentação: `local_id` obrigatório (com `implantacao` e custo), migration 000032 e remoção do caminho antigo sem local | pendente |
| **5** | Reservas (migration 000033, domínio, repo, usecase, rotas, port para o M6) + vitrine pelo disponível | pendente |
| **6** | Frontend: locais, saldo por local, movimentação com local e custo, transferência, reservas (depois da F0) | pendente |

---

## 9. Decisões fechadas

- [x] **Produto continua do grupo**: sem `empresa_id` no produto; o local carrega a
      empresa.
- [x] **Guarda otimista por produto** (`versao_estoque`), no lugar de comparar cada
      saldo e lote.
- [x] **FIFO** por `entrada_em`, desempate por `id`; transferência preserva custo e idade.
- [x] **Saída manual respeita o reservado**: `saida`/`perda`/`ajuste` negativo só
      consomem o disponível do local; a saída que baixa reserva informa `reserva_id`.
- [x] **Transferência só entre locais da mesma empresa**: entre CNPJs é operação fiscal
      (decisão aberta "Transferências entre empresas" do ROADMAP §6, afeta M8); até lá → `400`.
- [x] **Quem movimenta**: administrador, como hoje, mais o acesso à empresa do local. Perfil
      próprio de estoque, se precisar, entra depois, no padrão de flag no cargo.
- [x] **Custo visível só para administrador e financeiro** (rota de lotes); saldo e
      movimentos sem custo para os demais autenticados.
- [x] **Disponibilidade** sai no próprio `/saldo` (sem rota separada) e no port para o M6.
- [x] **Movimentos antigos sem local**: `CHECK (local_id IS NOT NULL) NOT VALID`.

## 10. Decisões fechadas com o usuário (outubro/2026)

- [x] **Saldo existente**: implantação manual. A migration não cria local nem lote; o saldo
      atual vira "não alocado" e o administrador registra `implantacao` por produto e local.
- [x] **Front durante o M2**: aceita a quebra do modal de movimentação (`local_id`
      obrigatório) até a fase 6, que vem depois da F0.
- [x] **Custo por tipo**: obrigatório só na `entrada`; opcional (`NULL` = desconhecido) em
      `devolucao`, `ajuste` positivo e `implantacao`.
- [x] **Reserva manual**: entra na fase 5, com origem `manual`, rotas de reservar, listar e
      liberar, e saída que baixa reserva; o M6 reaproveita o port com origem `pedido_item`.

## 11. Decisões pendentes

Sem abertos.
