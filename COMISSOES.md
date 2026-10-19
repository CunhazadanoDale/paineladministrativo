# Escopo — M10 Comissões

> **Status:** escopo fechado — decisões D1 a D8 respondidas pelo usuário (outubro/2026): todas as
> recomendações aceitas; API e front concluídos (fases 1 a 5).
> **Objetivo:** tirar o cálculo de comissões da mão: comissão do representante apurada pela saída das
> entregas do pedido (M6), proporcional ao entregue, com ajustes, fechamento mensal por representante
> e empresa, e a conta a pagar do fechamento gerada no M7 com vencimento no dia 15. Base para a linha
> `despesa_variavel` da DRE (M15).
> **Nome do módulo:** `comissao`
> **Branch:** `feat/m10-comissoes`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Módulo | Pacote novo `comissao`: `domain/comissao`, `ports/in|out/comissao`, `usecases/comissao`; rotas `/api/v1/comissoes` e `/api/v1/pedidos/{id}/comissao` |
| Fato gerador (D1) | A **saída** da entrega (`POST /entregas/{id}/saiu`): é onde o M6 soma `pedido_item.quantidade_entregue` e baixa o estoque. Entrega que saiu não volta (`saiu` → só `entregue`), então a comissão nasce sobre fato definitivo |
| Base (D3) | Valor dos produtos que saíram: `Σ entrega_item.quantidade × pedido_item.preco_unitario_centavos`. Sem frete (`entrega.frete_centavos`) e sem o valor da NF |
| Percentual (D2) | Percentual do pedido em tabela do M10 (`pedido_comissao`), copiado do padrão do representante (`comissao_pronta_entrega_bp` / `comissao_importacao_bp`, pelo `modelo` do pedido) e sobrescrevível pelo comercial. **Não toca na tabela nem no domínio do pedido** |
| Apuração (D4) | Rotina idempotente `POST /comissoes/apurar`, no padrão da recorrência do M7: varre as entregas que saíram e ainda não têm comissão e grava uma comissão por entrega (`uq_comissao_entrega`). Não muda o fluxo de gravação do M6 e já cobre as entregas anteriores ao M10 |
| Ajuste (D6) | `comissao_ajuste` append-only, positivo ou negativo, com motivo (devolução, desconto concedido depois, erro). Desfazer = ajuste inverso |
| Fechamento (D5) | Um por representante × empresa × competência (mês da saída), feito pelo financeiro; soma comissões e ajustes abertos até a competência e gera **uma** conta a pagar no M7 na mesma transação, vencendo no **dia 15 do mês seguinte** |
| Favorecido (D7) | O representante aponta para um `fornecedor` (M4) por `representante.fornecedor_id`; a conta a pagar do fechamento sai em nome desse fornecedor |
| Pagamento | O M10 não acompanha o pagamento: "paga" é lido da `solicitacao` do fechamento (status `pago`), sem callback do M7 para o M10 |
| Escopo por empresa | Padrão da seção 4 do `EMPRESAS.md`: a comissão herda a `empresa_id` do pedido |
| Permissões (D8) | Financeiro e administrador apuram, ajustam, fecham e veem tudo; comercial define o percentual do pedido; representante com login vê só as próprias comissões |
| Concorrência | Apuração e fechamento travam o que leem (`FOR SHARE` na entrega e no pedido, `FOR UPDATE` nas comissões abertas, ordem por `id`); `versao` no percentual do pedido |
| "Hoje" | Dia no fuso de Brasília, cópia própria em `domain/comissao/data.go`, como o M9 fez com `domainreceber.Hoje` |
| Dinheiro | Centavos `BIGINT`; percentual em pontos-base `INT` (`Percentual` do M3); arredondamento meio para cima, por entrega |

---

## 2. Entidades

### 2.1 `pedido_comissao` — percentual do pedido

| Campo | Tipo | Observações |
|---|---|---|
| `pedido_id` | UUID | PK; FK `pedido` `RESTRICT` |
| `representante_id` | UUID | FK `representante` `RESTRICT`; o representante do pedido quando o percentual foi definido |
| `percentual_bp` | INT | `0..10000` (`chk_pedido_comissao_percentual`) |
| `origem` | VARCHAR(10) | `padrao` (copiado do representante na apuração) ou `manual` (definido pelo comercial) |
| `usuario_id` | UUID | quem definiu; FK `usuario` |
| `versao` | INT | versão otimista (contrato do M4) |
| `criado_em`, `atualizado_em` | TIMESTAMPTZ | |

- Definido manualmente pelo `PUT /pedidos/{id}/comissao` enquanto nenhuma entrega do pedido saiu;
  depois da primeira saída fica travado (`409`), e diferenças viram ajuste.
- Sem linha manual, a primeira apuração grava a linha `padrao` com o percentual do representante
  naquele momento: mudar o padrão do representante depois não muda pedido já apurado.
- Se o representante do pedido mudar (só é possível em `proposta`) depois de uma linha `manual`, a
  linha deixa de valer: a leitura sugere o padrão do representante novo, a próxima definição a
  substitui e, se nenhuma definição vier, a apuração a troca pelo padrão do representante do pedido.

### 2.2 `comissao` — uma por entrega

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa`; a do pedido |
| `representante_id` | UUID | FK `representante` |
| `pedido_id` | UUID | FK `pedido` |
| `entrega_id` | UUID | FK `entrega`; `uq_comissao_entrega` — idempotência da apuração |
| `competencia` | DATE | dia 1 do mês do `entrega.saiu_em` |
| `base_centavos` | BIGINT | `>= 0` (zero quando a entrega só tem itens bonificados) |
| `percentual_bp` | INT | cópia do `pedido_comissao` na apuração |
| `valor_centavos` | BIGINT | `round(base × percentual / 10000)`; `0..base` (percentual zero gera comissão zero, para a entrega não voltar à apuração) |
| `fechamento_id` | UUID NULL | FK `comissao_fechamento` (fase 3); nulo = aberta |
| `usuario_id` | UUID | quem apurou |
| `criado_em` | TIMESTAMPTZ | |

- Imutável depois de gravada. Pedido sem representante não gera comissão.
- Pedido cancelado depois de saídas parciais mantém a comissão do que saiu (a mercadoria saiu).

### 2.3 `comissao_ajuste`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id`, `representante_id` | UUID | FKs |
| `pedido_id` | UUID NULL | referência opcional |
| `competencia` | DATE | dia 1; não posterior ao mês corrente |
| `valor_centavos` | BIGINT | `<> 0`; negativo reduz |
| `motivo` | VARCHAR(1000) | obrigatório |
| `fechamento_id` | UUID NULL | nulo = aberto (fase 3) |
| `usuario_id`, `criado_em` | | |

### 2.4 `comissao_fechamento`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | é o `origem_id` da conta a pagar |
| `empresa_id`, `representante_id` | UUID | FKs |
| `competencia` | DATE | dia 1; `uq_comissao_fechamento (empresa_id, representante_id, competencia)` |
| `comissoes_centavos`, `ajustes_centavos`, `total_centavos` | BIGINT | `total = comissões + ajustes`, `> 0` |
| `vencimento` | DATE | dia 15 do mês seguinte à competência |
| `categoria_financeira_id` | UUID | despesa lançável (sugestão: subcategoria de `despesa_variavel`) |
| `solicitacao_id` | UUID | FK `solicitacao`; gerada na mesma transação |
| `usuario_id`, `criado_em` | | |

- Fechamento não é desfeito: erro descoberto depois vira ajuste no mês seguinte, e a conta a pagar
  segue o fluxo do M7 (rejeitar/cancelar lá não reabre as comissões).
- Total `<= 0` não fecha: o representante vai para `falhas` (ou `400 nada a pagar` quando o fechamento
  pede só ele) e os ajustes negativos esperam o próximo mês.

---

## 3. Fluxos

### 3.1 Apuração

1. Seleciona entregas com `status IN ('saiu','entregue')`, sem `comissao`, cujo pedido tem representante,
   até a competência informada (padrão: todas), no acesso do usuário.
2. Por entrega, numa transação própria: trava entrega e pedido (`FOR SHARE`) e confere o representante,
   grava o percentual padrão quando o pedido não tem um vigente, trava o `pedido_comissao` (`FOR SHARE`)
   e confere que é o usado no cálculo, e grava com `INSERT ... ON CONFLICT (entrega_id) DO NOTHING`.
3. Até 500 entregas por chamada. Responde `{ apuradas, ja_existentes, falhas, restam_entregas }`: erro de
   negócio de uma entrega vira `falha` com o motivo e as demais seguem; erro de infraestrutura
   interrompe com `500`. Rodar de novo é seguro.

### 3.2 Fechamento

1. Financeiro informa `empresa_id`, `competencia`, `categoria_financeira_id` e, opcionalmente,
   `representante_id` (sem ele, fecha todos os representantes com total positivo).
2. Roda a apuração da empresa até a competência (`409` se sobrarem mais de 500 entregas) e soma os
   saldos abertos por representante.
3. Para cada representante (precisa ter `fornecedor_id` — D7), numa transação própria: trava as comissões
   e os ajustes abertos com competência `<=` a informada (`FOR UPDATE`, ordem por `id`) e confere a soma
   lida antes; gera a conta a pagar pela função interna `gerarSolicitacaoDeOrigem` (que trava empresa,
   fornecedor e categoria com `FOR SHARE`; origem `comissao`, `origem_id` = fechamento,
   `pendente_aprovacao`); grava o fechamento e marca `fechamento_id`. Falha de negócio de um
   representante vai para `falhas` e os demais seguem.

---

## 4. Endpoints

```
GET    /api/v1/pedidos/{id}/comissao              percentual vigente ou o que seria aplicado (comercial, financeiro, representante dono)
PUT    /api/v1/pedidos/{id}/comissao              { percentual_bp, versao? } (comercial; antes da primeira saída)

POST   /api/v1/comissoes/apurar                   { empresa_id?, competencia? } (financeiro)
GET    /api/v1/comissoes                          ?empresa= ?representante= ?pedido= ?competencia= ?aberta= (financeiro; representante: só as suas)
GET    /api/v1/comissoes/pendencias               ?empresa= ?competencia= — entregas que saíram e ainda não foram apuradas (financeiro)

POST   /api/v1/comissoes/ajustes                  { empresa_id, representante_id, pedido_id?, competencia, valor_centavos, motivo } (financeiro)
GET    /api/v1/comissoes/ajustes                  ?empresa= ?representante= ?pedido= ?competencia= ?aberto= (financeiro; representante: só os seus)

POST   /api/v1/comissoes/fechamentos              { empresa_id, competencia, categoria_financeira_id, representante_id? } (financeiro)
GET    /api/v1/comissoes/fechamentos              ?empresa= ?representante= ?competencia= (financeiro; representante: só os seus)
GET    /api/v1/comissoes/fechamentos/{id}         com comissões, ajustes e status da conta a pagar

GET    /api/v1/comissoes/resumo                   ?competencia= (obrigatória) ?empresa= ?representante= — por empresa e representante: comissões, ajustes, em aberto, fechado, pago

PUT    /api/v1/representantes/{id}/fornecedor     { fornecedor_id } (administrador)
```

---

## 5. Permissões

- `financeiro(...)` apura, ajusta e fecha; `comercial(...)` define o percentual do pedido; leitura em
  `protegido(...)` com regra fina em `usecases/comissao/permissoes.go`.
- Representante com login (não administrador) vê só as próprias comissões, ajustes, fechamentos e
  resumo, **mesmo com cargo financeiro**, e não apura, ajusta nem fecha (ninguém lança nem paga a
  própria comissão); regra de `usecases/pedido/permissoes.go` com `ObterPorUsuario`.
- Escopo por empresa em tudo, com teste de isolamento no usecase e no e2e.

---

## 6. Contrato com os outros módulos

### 6.1 Pedido (M6)

Porta mínima em `ports/out/comissao/` só de leitura (`Obter` do pedido com itens e entregas). A
apuração lê `entrega`, `entrega_item` e `pedido_item` por consulta própria no
`adapter/postgres/comissao_repo.go`. **Nenhum método novo no port nem no repositório do M6 e nenhuma
coluna nova no `pedido`**, o que deixa o M11 livre para mexer no pedido em paralelo.

### 6.2 Representante (M3)

Porta mínima `Obter` e `ObterPorUsuario`. Pela D7, a migration 000055 acrescenta
`representante.fornecedor_id UUID NULL` (FK `fornecedor` `RESTRICT`), definido pelo administrador em
`PUT /representantes/{id}/fornecedor` — o único acréscimo ao M3 (domínio, repositório, usecase, DTO,
rota e `API.md` do representante).

### 6.3 Conta a pagar (M7)

- Migration acrescenta `comissao` em `chk_solicitacao_origem`; o `uq_solicitacao_origem` existente
  garante a idempotência.
- `domainsolicitacao.OrigemComissao` em `NovaOrigem` e em `Origem.Sistemica()`; a mensagem fixa
  "recompra ou repasse_fomento" de `NovaSolicitacaoDeOrigem` passa a listar as origens sistêmicas.
- Fechamento chama `gerarSolicitacaoDeOrigem(ctx, tx, ...)` na transação do fechamento (padrão do M9):
  `FornecedorID` = do representante, categoria do fechamento, `Competencia` = do fechamento,
  `Vencimento` = dia 15, `FormaPagamento` = `transferencia`, observação com representante e período.

### 6.4 Demais

- Empresa (M1), categoria financeira (M5, natureza `despesa`, `Lancavel()`), fornecedor (M4, `Ativo`):
  portas mínimas com `Obter`; travas `FOR SHARE` no repositório do M10.
- M15 lê a despesa da comissão pela conta a pagar (`competencia` + `linha_dre` da categoria), não pelas
  tabelas do M10, para não contar duas vezes.

---

## 7. Estrutura de código

```
migrations/000053_create_tabela_comissao                 pedido_comissao, comissao
migrations/000054_create_tabela_comissao_ajuste
migrations/000055_create_tabela_comissao_fechamento      fechamento + chk_solicitacao_origem + representante.fornecedor_id
migrations/000056_alter_comissao_base_zero               chk_comissao_base aceita base zero (itens bonificados)

internal/core/domain/comissao/       percentual_pedido.go, origem_percentual.go, comissao.go, ajuste.go,
                                     fechamento.go, resumo.go, competencia.go, data.go, erros.go (+ testes)
internal/core/domain/solicitacao/    origem.go (+1 valor), origem_sistemica.go (mensagem)
internal/core/domain/cliente/        representante.go (+ FornecedorID e DefinirFornecedor)
internal/core/ports/out/comissao/    comissao_repository.go, pedido_repository.go, representante_repository.go,
                                     empresa_repository.go, categoria_financeira_repository.go,
                                     usuario_repository.go, cargo_repository.go
internal/core/ports/in/comissao/     percentual_usecase.go, comissao_usecase.go (apuração, pendências,
                                     lista e resumo), ajuste_usecase.go, fechamento_usecase.go
internal/core/usecases/comissao/     idem + apurador.go, permissoes.go (+ testes, fakes e cenário)
internal/adapter/postgres/           comissao_repo.go, comissao_fechamento_repo.go, comissao_resumo_repo.go;
                                     representante_repo.go (fornecedor_id)
internal/adapter/http/dto/comissao/ + handlers/comissao/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

- Numeração a partir da `000053` (a `main` está na `000052`); renumerar antes do merge se outra
  sessão mergear antes (regra 4 da seção 3 do `ROADMAP.md`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Percentual do pedido e apuração: migration 000053, domínio, repositório, usecase com escopo por empresa e representante, `GET/PUT /pedidos/{id}/comissao`, `POST /comissoes/apurar`, listagem e pendências, integração de idempotência e corrida, e2e + `API.md` | concluída |
| **2** | Ajustes (migration 000054): domínio, usecase, repositório, `POST/GET /comissoes/ajustes`, integração, e2e + `API.md` | concluída |
| **3** | Fechamento com conta a pagar no M7 na mesma transação, `representante.fornecedor_id` (migration 000055), `PUT /representantes/{id}/fornecedor`, filtros de abertas, integração, e2e + `API.md` | concluída |
| **4** | Resumo por representante (`GET /comissoes/resumo`) e visão do representante com login, integração, e2e + `API.md` | concluída |
| **5** | Frontend (percentual no pedido, apuração, ajustes, fechamento, "minhas comissões") | concluída |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Nascimento proporcional ao entregue**, uma comissão por entrega (ROADMAP, seção 5).
- [x] **Fechamento mensal com pagamento no dia 15** e conta a pagar no M7 (ROADMAP, seção 5).
- [x] **Representante com login vê só as próprias comissões** (decisão da seção 6 do `ROADMAP.md`).
- [x] **Conta a pagar por `gerarSolicitacaoDeOrigem`** na transação do fechamento (contrato do M7 fechado
      pelo M9).
- [x] **Uma transação por entrega na apuração** (fase 1): uma entrega com problema não bloqueia as
      outras e aparece em `falhas`; o limite de 500 por chamada evita requisição longa.
- [x] **Definição para outro representante não vale** (fase 1): é substituída pela próxima definição ou
      pelo padrão na apuração, em vez de virar pendência sem saída depois da primeira saída.
- [x] **Representante com login nunca define a própria comissão** (fase 1), mesmo com cargo comercial.
- [x] **Fornecedor do representante em rota própria** (fase 3): `PUT /representantes/{id}/fornecedor`, só
      administrador; o `PUT` do cadastro não mexe nele, para o front do M3 (que não conhece o campo) não
      apagá-lo ao salvar.
- [x] **Só fecha mês encerrado** (fase 3): a competência precisa ser anterior ao mês corrente em
      Brasília, para o mês não fechar com saídas ainda por vir.
- [x] **Fechamento apura antes** (fase 3) as saídas da empresa até a competência e recusa (`409`) se
      sobrarem mais de 500; representante sem fornecedor, sem saldo positivo ou já fechado vira `falha`
      e os demais seguem. A transação trava comissões e ajustes abertos (`FOR UPDATE`) e confere a soma
      lida antes, para dois fechamentos simultâneos não pagarem duas vezes.
- [x] **Revisão da branch** (outubro/2026): `PUT /representantes/{id}/fornecedor` grava só a coluna
      `fornecedor_id` (o `PUT` do cadastro não a regrava, então um não desfaz o outro), exige o campo e
      recusa fornecedor inativo; entrega que falha na apuração não conta no limite de 500 nem é repetida
      na mesma chamada; representante com entrega sem apurar não fecha; fechamento de um representante só
      devolve o motivo como erro; ajuste em competência já fechada entra no fechamento seguinte.
- [x] **Item bonificado entra na apuração** (correção pós-M10, migration 000056): o M6 aceita item
      com preço zero, que entra com subtotal zero; quantidade continua obrigatória. Entrega só de
      brindes gera comissão com base e valor zero, em vez de virar falha permanente que impediria o
      representante de fechar. `chk_comissao_base` passa a `base_centavos >= 0`.
- [x] **Ajuste aceita representante inativo** (fase 2): a comissão continua devida; o pedido, quando
      informado, é travado com `FOR SHARE` e precisa ser da mesma empresa e do mesmo representante.
- [x] **D1 — Quando a comissão nasce.** Decidido: na **saída** da entrega (onde o M6 já conta o
      entregue e baixa o estoque; entrega que saiu não é desfeita). Alternativas: na conclusão
      (`entregue`, data de recebimento pelo cliente) ou no **recebimento do título** (comissão só sobre o
      que o cliente pagou — protege contra inadimplência, mas amarra o M10 ao M9 e às baixas parciais).
- [x] **D2 — Onde fica o percentual sobrescrito.** Decidido: tabela própria `pedido_comissao` do M10,
      editável até a primeira saída, com cópia do padrão do representante na primeira apuração.
      Alternativa: coluna `comissao_bp` no `pedido`, editada no `PUT` da proposta — mais natural na tela,
      mas mexe no domínio e no repositório do M6 e conflita com o M11.
- [x] **D3 — Base de cálculo.** Decidido: preço dos produtos que saíram (sem frete, sem impostos da
      NF). Alternativa: valor da NF de venda vinculada à entrega (inclui impostos e só existe depois da
      importação do XML).
- [x] **D4 — Como a comissão é gerada.** Decidido: apuração idempotente sob demanda (e automática no
      fechamento), sem tocar no fluxo de saída do M6. Alternativa: gerar na mesma transação da saída —
      comissão aparece na hora, mas exige mudar `PedidoRepository.Gravar` e o usecase do pedido.
- [x] **D5 — Fechamento.** Decidido: um por representante × empresa × mês da saída, comissões e
      ajustes em aberto de meses anteriores entram no próximo fechamento, conta a pagar em
      `pendente_aprovacao` vencendo dia 15 do **mês seguinte** à competência, categoria escolhida no
      fechamento.
- [x] **D6 — Devolução e correção.** Decidido: ajuste manual com motivo (o M6 não tem devolução de
      mercadoria; a NF de `devolucao` do M8 não move estoque). Alternativa: estornar automaticamente a
      partir da NF de devolução — depende da decisão aberta de transferências e do vínculo NF ↔ entrega.
- [x] **D7 — Favorecido da conta a pagar.** Decidido: `representante.fornecedor_id` obrigatório para
      fechar (o representante é cadastrado também como fornecedor, PF ou PJ, no M4), o que dá documento
      e conciliação no M13. Alternativa: conta a pagar sem fornecedor, com o representante só na
      observação — a mesma dúvida da pendência 12 (devolução do pedido cancelado); a resposta vale para
      as duas.
- [x] **D8 — Quem vê comissões além do financeiro.** Decidido: o comercial **não** vê valores de
      comissão, só define o percentual do pedido; o representante com login vê as próprias.
      Alternativa: comercial lê tudo (gestor comercial acompanhando a equipe).

## 10. Decisões pendentes

Sem abertos.
