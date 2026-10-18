# Escopo — M10 Comissões

> **Status:** escopo em revisão — decisões D1 a D8 (seção 10) aguardando o usuário; nenhuma linha de
> código antes das respostas (regra 11 da seção 3 do `ROADMAP.md`).
> **Objetivo:** tirar o cálculo de comissões da mão: comissão do representante apurada pela saída das
> entregas do pedido (M6), proporcional ao entregue, com ajustes, fechamento mensal por representante
> e empresa, e a conta a pagar do fechamento gerada no M7 com vencimento no dia 15. Base para a linha
> `despesa_variavel` da DRE (M15).
> **Nome do módulo:** `comissao`
> **Branch:** `feat/m10-comissoes`

---

## 1. Visão geral

| Tema | Decisão (proposta; as marcadas com Dn dependem da seção 10) |
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
  linha é descartada na próxima definição; a apuração recusa a entrega cujo pedido tenha linha de
  outro representante (`409`, listado em `GET /comissoes/apurar` como pendência).

### 2.2 `comissao` — uma por entrega

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa`; a do pedido |
| `representante_id` | UUID | FK `representante` |
| `pedido_id` | UUID | FK `pedido` |
| `entrega_id` | UUID | FK `entrega`; `uq_comissao_entrega` — idempotência da apuração |
| `competencia` | DATE | dia 1 do mês do `entrega.saiu_em` |
| `base_centavos` | BIGINT | `> 0` |
| `percentual_bp` | INT | cópia do `pedido_comissao` na apuração |
| `valor_centavos` | BIGINT | `round(base × percentual / 10000)`; `>= 0` (percentual zero gera comissão zero, para a entrega não voltar à apuração) |
| `fechamento_id` | UUID NULL | FK `comissao_fechamento`; nulo = aberta |
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
| `fechamento_id` | UUID NULL | nulo = aberto |
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
- Total `<= 0` não fecha (`400 nada a pagar na competência`); os ajustes negativos esperam o próximo mês.

---

## 3. Fluxos

### 3.1 Apuração

1. Seleciona entregas com `status IN ('saiu','entregue')`, sem `comissao`, cujo pedido tem representante,
   até a competência informada (padrão: todas), no acesso do usuário.
2. Por entrega, trava entrega e pedido (`FOR SHARE`), resolve o percentual (2.1), calcula a base pelos
   itens da entrega e grava com `INSERT ... ON CONFLICT (entrega_id) DO NOTHING`.
3. Responde `{ apuradas, ja_existentes, pendencias }`; rodar de novo é seguro.

### 3.2 Fechamento

1. Financeiro informa `empresa_id`, `competencia`, `categoria_financeira_id` e, opcionalmente,
   `representante_id` (sem ele, fecha todos os representantes com total positivo).
2. Roda a apuração da empresa até a competência, trava as comissões e ajustes abertos com competência
   `<=` a informada (`FOR UPDATE`, ordem por `id`) e o representante (`FOR SHARE`, precisa ter
   `fornecedor_id` de fornecedor ativo — D7).
3. Grava o fechamento, marca `fechamento_id` e gera a conta a pagar pela função interna
   `gerarSolicitacaoDeOrigem` (origem `comissao`, `origem_id` = fechamento, `pendente_aprovacao`).

---

## 4. Endpoints

```
GET    /api/v1/pedidos/{id}/comissao              percentual vigente ou o que seria aplicado (comercial, financeiro, representante dono)
PUT    /api/v1/pedidos/{id}/comissao              { percentual_bp, versao? } (comercial; antes da primeira saída)

POST   /api/v1/comissoes/apurar                   { empresa_id?, competencia? } (financeiro)
GET    /api/v1/comissoes                          ?empresa= ?representante= ?pedido= ?competencia= ?aberta= (financeiro; representante: só as suas)
GET    /api/v1/comissoes/pendencias               entregas que não apuram e o motivo (financeiro)

POST   /api/v1/comissoes/ajustes                  { empresa_id, representante_id, pedido_id?, competencia, valor_centavos, motivo } (financeiro)
GET    /api/v1/comissoes/ajustes                  ?empresa= ?representante= ?competencia= ?aberto= (financeiro; representante: só os seus)

POST   /api/v1/comissoes/fechamentos              { empresa_id, competencia, categoria_financeira_id, representante_id? } (financeiro)
GET    /api/v1/comissoes/fechamentos              ?empresa= ?representante= ?competencia= (financeiro; representante: só os seus)
GET    /api/v1/comissoes/fechamentos/{id}         com comissões, ajustes e status da conta a pagar

GET    /api/v1/comissoes/resumo                   ?empresa= ?competencia= — por representante: apurado, ajustes, fechado, pago
```

---

## 5. Permissões

- `financeiro(...)` apura, ajusta e fecha; `comercial(...)` define o percentual do pedido; leitura em
  `protegido(...)` com regra fina em `usecases/comissao/permissoes.go`.
- Representante com login (não administrador) vê só as próprias comissões, ajustes e fechamentos, sem
  o resumo dos outros; regra de `usecases/pedido/permissoes.go` com `ObterPorUsuario`.
- Escopo por empresa em tudo, com teste de isolamento no usecase e no e2e.

---

## 6. Contrato com os outros módulos

### 6.1 Pedido (M6)

Porta mínima em `ports/out/comissao/` só de leitura (`Obter` do pedido com itens e entregas). A
apuração lê `entrega`, `entrega_item` e `pedido_item` por consulta própria no
`adapter/postgres/comissao_repo.go`. **Nenhum método novo no port nem no repositório do M6 e nenhuma
coluna nova no `pedido`**, o que deixa o M11 livre para mexer no pedido em paralelo.

### 6.2 Representante (M3)

Porta mínima `Obter` e `ObterPorUsuario`. Se D7 for aceita, migration do M10 acrescenta
`representante.fornecedor_id UUID NULL` (FK `fornecedor` `RESTRICT`), editável pelo administrador no
cadastro do representante — o único acréscimo ao M3 (domínio, DTO e `API.md` do representante).

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

internal/core/domain/comissao/       percentual_pedido.go, comissao.go, apuracao.go, ajuste.go,
                                     fechamento.go, competencia.go, data.go (+ testes)
internal/core/domain/solicitacao/    origem.go (+1 valor), origem_sistemica.go (mensagem)
internal/core/domain/cliente/        representante.go (+ FornecedorID, se D7)
internal/core/ports/out/comissao/    comissao_repository.go, fechamento_repository.go, pedido_repository.go,
                                     representante_repository.go, empresa_repository.go,
                                     categoria_financeira_repository.go, fornecedor_repository.go,
                                     usuario_repository.go, cargo_repository.go
internal/core/ports/in/comissao/     percentual_usecase.go, apuracao_usecase.go, ajuste_usecase.go,
                                     fechamento_usecase.go, resumo_usecase.go
internal/core/usecases/comissao/     idem + permissoes.go (+ testes, fakes e cenário)
internal/adapter/postgres/           comissao_repo.go, comissao_fechamento_repo.go
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
| **1** | Percentual do pedido e apuração: migration 000053, domínio, repositório, usecase com escopo por empresa e representante, `GET/PUT /pedidos/{id}/comissao`, `POST /comissoes/apurar`, listagem e pendências, integração de idempotência e corrida, e2e + `API.md` | pendente |
| **2** | Ajustes (migration 000054) | pendente |
| **3** | Fechamento com conta a pagar no M7 na mesma transação, `representante.fornecedor_id` (migration 000055) | pendente |
| **4** | Resumo por representante e visão do representante com login | pendente |
| **5** | Frontend (percentual no pedido, apuração, ajustes, fechamento, "minhas comissões") | pendente |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Nascimento proporcional ao entregue**, uma comissão por entrega (ROADMAP, seção 5).
- [x] **Fechamento mensal com pagamento no dia 15** e conta a pagar no M7 (ROADMAP, seção 5).
- [x] **Representante com login vê só as próprias comissões** (decisão da seção 6 do `ROADMAP.md`).
- [x] **Conta a pagar por `gerarSolicitacaoDeOrigem`** na transação do fechamento (contrato do M7 fechado
      pelo M9).

## 10. Decisões pendentes (levar ao usuário)

Cada item traz a recomendação e a alternativa.

- [ ] **D1 — Quando a comissão nasce.** Recomendado: na **saída** da entrega (onde o M6 já conta o
      entregue e baixa o estoque; entrega que saiu não é desfeita). Alternativas: na conclusão
      (`entregue`, data de recebimento pelo cliente) ou no **recebimento do título** (comissão só sobre o
      que o cliente pagou — protege contra inadimplência, mas amarra o M10 ao M9 e às baixas parciais).
- [ ] **D2 — Onde fica o percentual sobrescrito.** Recomendado: tabela própria `pedido_comissao` do M10,
      editável até a primeira saída, com cópia do padrão do representante na primeira apuração.
      Alternativa: coluna `comissao_bp` no `pedido`, editada no `PUT` da proposta — mais natural na tela,
      mas mexe no domínio e no repositório do M6 e conflita com o M11.
- [ ] **D3 — Base de cálculo.** Recomendado: preço dos produtos que saíram (sem frete, sem impostos da
      NF). Alternativa: valor da NF de venda vinculada à entrega (inclui impostos e só existe depois da
      importação do XML).
- [ ] **D4 — Como a comissão é gerada.** Recomendado: apuração idempotente sob demanda (e automática no
      fechamento), sem tocar no fluxo de saída do M6. Alternativa: gerar na mesma transação da saída —
      comissão aparece na hora, mas exige mudar `PedidoRepository.Gravar` e o usecase do pedido.
- [ ] **D5 — Fechamento.** Recomendado: um por representante × empresa × mês da saída, comissões e
      ajustes em aberto de meses anteriores entram no próximo fechamento, conta a pagar em
      `pendente_aprovacao` vencendo dia 15 do mês seguinte, categoria escolhida no fechamento. Confirmar
      o dia 15 do **mês seguinte** e se a conta nasce `pendente_aprovacao` ou já aprovada.
- [ ] **D6 — Devolução e correção.** Recomendado: ajuste manual com motivo (o M6 não tem devolução de
      mercadoria; a NF de `devolucao` do M8 não move estoque). Alternativa: estornar automaticamente a
      partir da NF de devolução — depende da decisão aberta de transferências e do vínculo NF ↔ entrega.
- [ ] **D7 — Favorecido da conta a pagar.** Recomendado: `representante.fornecedor_id` obrigatório para
      fechar (o representante é cadastrado também como fornecedor, PF ou PJ, no M4), o que dá documento
      e conciliação no M13. Alternativa: conta a pagar sem fornecedor, com o representante só na
      observação — a mesma dúvida da pendência 12 (devolução do pedido cancelado); a resposta vale para
      as duas.
- [ ] **D8 — Quem vê comissões além do financeiro.** Recomendado: o comercial **não** vê valores de
      comissão, só define o percentual do pedido; o representante com login vê as próprias.
      Alternativa: comercial lê tudo (gestor comercial acompanhando a equipe).
