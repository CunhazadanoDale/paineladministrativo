# Escopo — M11 Propostas e tabela de preços

> **Status:** escopo fechado — decisões D1 a D9 respondidas pelo usuário (outubro/2026): todas as
> recomendações aceitas; API concluída (fases 1 a 3); front (fase 4) pendente.
> **Objetivo:** tirar os orçamentos do documento solto e o "guia de modelos de venda" da cabeça do
> vendedor: tabela de preço por produto e modelo de venda com vigência, proposta (o pedido em `proposta`
> do M6) com validade, condições comerciais editáveis, frete estimado, revisão numerada e um documento
> único de proposta para imprimir ou salvar em PDF. Aceitar a proposta continua sendo confirmar o pedido.
> **Nome do módulo:** `tabelapreco` (tabela de preço) + acréscimos no módulo `pedido` (M6), dono da proposta
> **Branch:** `feat/m11-propostas`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Tabela de preço (D1, D2) | `tabela_preco_item`: preço por **produto × modelo de venda**, do **grupo** (o produto já é do grupo), com **vigência por data de início**; o preço vigente num dia é o de maior `vigente_de` até aquele dia. Append-only: mudar preço é lançar nova vigência |
| Uso da tabela (D3) | **Sugestão**, não trava: o item do pedido grava o `preco_tabela_centavos` vigente no momento em que foi lançado, para a proposta mostrar o desconto e a gestão enxergar vendas abaixo da tabela. Sem preço na tabela, cai em `produto.preco` como hoje |
| Vitrine pública | Continua usando `produto.preco` (decisão do ROADMAP) |
| Proposta | É o pedido em `proposta` do M6; o M11 acrescenta colunas no `pedido` e regras no domínio do pedido, sem módulo paralelo |
| Validade (D4) | `pedido.validade` (DATE), obrigatória na proposta, padrão **hoje + 15 dias**; proposta vencida não confirma (`409`) até ganhar validade nova; "vencida" é derivado, nunca gravado |
| Condições (D5) | `entrada_bp`, `prazo_boleto_dias` e `prazo_entrega_dias` passam a ser **editáveis enquanto proposta**, com o padrão do modelo quando não informados; travam na confirmação |
| Frete (D6) | `pedido.frete_tipo` (`cif`/`fob`) e `pedido.frete_estimado_centavos` informativos na proposta; **não somam** ao `valor_total_centavos` (o frete real continua na entrega) |
| Revisão (D7) | `pedido.revisao` começa em 1 e sobe a cada alteração da proposta; o documento sai como "Proposta 57-R2" |
| Documento (D8) | `GET /pedidos/{id}/proposta` devolve **tudo o que o documento precisa** (empresa, cliente, obra, representante, itens com tabela e desconto, condições, frete, validade, revisão, observações); o **front monta a página de impressão** e o navegador salva em PDF. Sem dependência nova no backend |
| Aceite (D9) | Aceitar = `POST /pedidos/{id}/confirmar` (já existe), com a validade conferida; o documento traz linha de assinatura, sem assinatura eletrônica |
| Permissões | Tabela: **administrador** escreve, comercial e financeiro leem; proposta: as regras do M6 (comercial; representante com login só as próprias) |
| Escopo por empresa | Tabela é do grupo (sem empresa); a proposta segue o escopo do pedido |
| "Hoje" | `domainpedido.Hoje` (fuso de Brasília) para validade e vigência |
| Dinheiro | Centavos `BIGINT`; percentuais em pontos-base |

---

## 2. Entidades

### 2.1 `tabela_preco_item`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `produto_id` | UUID | FK `produto` `RESTRICT`; produto ativo ao lançar |
| `modelo` | VARCHAR(16) | `pronta_entrega` / `importacao` (`chk`) |
| `preco_centavos` | BIGINT | `> 0` |
| `vigente_de` | DATE | início da vigência; `uq_tabela_preco_item (produto_id, modelo, vigente_de)` |
| `observacao` | VARCHAR(500) | motivo do reajuste, opcional |
| `usuario_id` | UUID | quem lançou |
| `criado_em` | TIMESTAMPTZ | |

- Lançamento com `vigente_de` no passado é aceito (correção), mas não muda pedidos já lançados: o item
  guarda o preço de tabela do momento.
- Exclusão só de vigência **futura** (ainda não começou), pelo administrador; o histórico do que já
  vigorou não é apagado.

### 2.2 Colunas novas no `pedido` (M6)

| Campo | Tipo | Observações |
|---|---|---|
| `validade` | DATE NULL | obrigatória enquanto `proposta`; as propostas existentes recebem `criado_em + 15 dias` na migration |
| `revisao` | INT | `>= 1`; sobe a cada `PUT` da proposta |
| `frete_tipo` | VARCHAR(3) NULL | `cif`/`fob` |
| `frete_estimado_centavos` | BIGINT NULL | `>= 0`; só com `frete_tipo` |

`entrada_bp`, `prazo_boleto_dias` e `prazo_entrega_dias` já existem; o que muda é que o `PUT` da proposta
passa a aceitá-los.

### 2.3 Coluna nova no `pedido_item` (M6)

| Campo | Tipo | Observações |
|---|---|---|
| `preco_tabela_centavos` | BIGINT NULL | preço de tabela vigente quando o item foi lançado ou alterado; nulo sem tabela nem `produto.preco` |

Desconto do item = `preco_tabela − preco_unitario` (pode ser negativo: venda acima da tabela).

---

## 3. Fluxos

### 3.1 Preço vigente

`GET /tabela-precos/vigentes?modelo=&produto=&data=` responde o preço de cada produto naquele dia (padrão:
hoje), com a origem (`tabela` ou `produto`). O front do pedido usa essa rota para sugerir o preço, e o
usecase do pedido usa a mesma regra (porta mínima) para gravar `preco_tabela_centavos`.

### 3.2 Proposta

1. `POST /pedidos` e `PUT /pedidos/{id}` (já existem) aceitam `validade`, `entrada_bp`,
   `prazo_boleto_dias`, `prazo_entrega_dias`, `frete_tipo` e `frete_estimado_centavos`; sem eles, valem
   os padrões (validade hoje + 15, condições do modelo, sem frete).
2. Cada `PUT` em proposta sobe a `revisao` e regrava `preco_tabela_centavos` dos itens.
3. Validade não pode ser anterior a hoje ao criar ou alterar.
4. `POST /pedidos/{id}/confirmar` recusa proposta vencida (`409 proposta vencida em …; renove a
   validade antes de confirmar`).

### 3.3 Documento

`GET /pedidos/{id}/proposta` (mesmas permissões de leitura do pedido) devolve o documento montado:
cabeçalho da empresa (razão social, CNPJ, UF), cliente (nome, documento, cidade/UF), obra (endereço de
entrega), representante, número e revisão, data, validade e se está vencida, itens (código, nome,
unidade, quantidade, preço, preço de tabela, desconto, subtotal), total, condições por extenso (ex.:
"entrada de 30% e saldo na chegada; prazo de entrega de 120 dias após a entrada"), frete e
observações. Funciona em qualquer status (reimpressão do pedido confirmado).

---

## 4. Endpoints

```
POST   /api/v1/tabela-precos                      { produto_id, modelo, preco_centavos, vigente_de, observacao } (administrador)
GET    /api/v1/tabela-precos                      ?produto= ?modelo= — histórico paginado (comercial, financeiro)
GET    /api/v1/tabela-precos/vigentes             ?modelo= ?produto= ?data= ?busca= — preço do dia por produto (comercial, financeiro)
DELETE /api/v1/tabela-precos/{id}                 só vigência futura (administrador)

POST   /api/v1/pedidos                            + validade, condições e frete (comercial)
PUT    /api/v1/pedidos/{id}                       + validade, condições e frete; sobe a revisão (comercial)
GET    /api/v1/pedidos/{id}/proposta              documento da proposta (leitura do pedido)
GET    /api/v1/modelos-venda                      condições padrão de cada modelo, para o front (autenticado)
```

---

## 5. Permissões

- Tabela: `administrador(...)` lança e exclui; leitura em `comercial(...)` e `financeiro(...)` (a vitrine
  pública não lê a tabela).
- Proposta e documento: as do M6, inclusive o representante com login (só os próprios pedidos) e o
  escopo por empresa.

---

## 6. Contrato com os outros módulos

- **Pedido (M6)**: o M11 mexe no domínio, no repositório e no DTO do pedido (colunas novas), e o usecase
  do pedido ganha uma porta mínima `PrecoVigente` (`ports/out/pedido/`) satisfeita pelo repositório da
  tabela. Sem conflito com o M10, que só lê pedido, entrega e itens.
- **Produto (M2)**: porta mínima `Obter` (ativo, `preco` como fallback).
- **Empresa, cliente, obra, representante**: leitura para o documento, por portas mínimas do pedido.
- **M15**: pode usar `preco_tabela_centavos` para relatório de descontos concedidos.

---

## 7. Estrutura de código

```
migrations/000057_create_tabela_preco_item
migrations/000058_alter_pedido_proposta             validade, revisao, frete, pedido_item.preco_tabela_centavos

internal/core/domain/tabelapreco/      item.go, vigencia.go (+ testes)
internal/core/domain/pedido/           proposta.go (validade, revisão, frete, condições editáveis), documento.go
internal/core/ports/out/tabelapreco/ + ports/in/tabelapreco/
internal/core/ports/out/pedido/        preco_vigente_repository.go (+ portas de leitura do documento)
internal/core/usecases/tabelapreco/    + usecases/pedido (validade, condições, documento)
internal/adapter/postgres/             tabela_preco_repo.go; pedido_repo.go (colunas novas, só acréscimos)
internal/adapter/http/dto/tabelapreco/ + handlers/tabelapreco/; dto/pedido e handlers/pedido (acréscimos)
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

- Numeração a partir da `000057`: a `main` recebeu a `000056` da correção do M10 (preço zero) antes do
  M11, e a tabela de preço foi renumerada de `000056` para `000057`.

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Tabela de preço: migration 000057, domínio, repositório, usecase, rotas, preço vigente, integração, e2e + `API.md` | concluída |
| **2** | Proposta: migration 000058, validade, condições editáveis, frete, revisão, `preco_tabela_centavos` nos itens, confirmar recusando vencida, e2e + `API.md` | concluída |
| **3** | Documento da proposta (`GET /pedidos/{id}/proposta`) e `GET /modelos-venda`, com as condições por extenso montadas no domínio do pedido, testes e `API.md` | concluída |
| **4** | Frontend: tabela de preço, preço sugerido no pedido, campos da proposta e página de impressão | pendente |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Proposta é o pedido em `proposta`** (M6); aceitar = confirmar (ROADMAP, seção 5).
- [x] **Vitrine pública continua com `produto.preco`** (ROADMAP, seção 5).
- [x] **Preço por produto e modelo de venda, com vigência** (ROADMAP, seção 5).
- [x] **`PUT` sem validade renova** (fase 2): o `PUT` substitui os editáveis da proposta, então sem
      `validade` ela volta a hoje + 15 dias; para manter, o front reenvia a validade lida.
- [x] **Prazo de entrega em dias só na importação** (fase 2): na pronta entrega o prazo é o
      `prazo_limite` digitado; `prazo_entrega_dias` lá responde `400`.
- [x] **Preço de tabela recalculado a cada `PUT`** (fase 2): o item guarda o preço vigente no dia da
      última alteração da proposta, pela mesma regra de `GET /tabela-precos/vigentes`.
- [x] **D1 — Tabela do grupo ou por empresa.** Decidido: **do grupo** (o produto e o estoque de
      catálogo já são do grupo; MIA, Red e Gigante vendem pela mesma tabela). Alternativa: tabela por
      empresa (`empresa_id` na vigência), se cada CNPJ praticar preço próprio.
- [x] **D2 — Vigência.** Decidido: **só data de início**, a próxima vigência encerra a anterior
      (histórico append-only, sem sobreposição possível). Alternativa: início e fim explícitos (permite
      buracos e promoções com data para acabar, mas exige validar sobreposição).
- [x] **D3 — A tabela trava o preço?** Decidido: **só sugere** e grava o preço de tabela no item para
      mostrar o desconto. Alternativas: piso (preço abaixo da tabela só com perfil administrador) ou
      desconto máximo em percentual por perfil.
- [x] **D4 — Validade.** Decidido: obrigatória, padrão **15 dias**, vencida não confirma até renovar.
- [x] **D5 — Condições editáveis.** Decidido: entrada, prazo do boleto e prazo de entrega editáveis
      **enquanto proposta**, por quem edita a proposta, sem limite. Alternativa: limites por modelo (ex.:
      entrada mínima de 30% na importação) ou edição só pelo administrador.
- [x] **D6 — Frete na proposta.** Decidido: tipo e valor **estimado e informativo**, fora do total do
      pedido. Alternativa: frete somado ao total (muda o valor que o M10 e o M9 usam hoje).
- [x] **D7 — Revisão.** Decidido: número de revisão que sobe a cada alteração, sem guardar as versões
      antigas. Alternativa: guardar cada revisão (itens e condições) em histórico consultável.
- [x] **D8 — Onde nasce o PDF.** Decidido: **front monta a página de impressão** a partir de
      `GET /pedidos/{id}/proposta` e o navegador salva em PDF (sem dependência nova, layout ajustado sem
      deploy do backend). Alternativa: o backend gera o PDF (biblioteca Go nova, ex. `go-pdf/fpdf`), útil
      para anexar por e-mail/WhatsApp automaticamente no futuro.
- [x] **D9 — Quem mexe na tabela.** Decidido: **administrador** lança e exclui; comercial e
      financeiro leem. Alternativa: perfil comercial também lança (sem dono único da política de preço).

## 10. Decisões pendentes

Sem abertos.
