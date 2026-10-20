# Escopo — M12 Importações

> **Status:** escopo fechado — decisões D1 a D14 respondidas pelo usuário (outubro/2026), todas as
> recomendações aceitas; API concluída (fases 1 a 4: importação e estados, parcelas, câmbio e custos com conta a
> pagar, alocação de pedidos, recebimento com estoque e reserva, alertas); front (fase 5) pendente.
> **Objetivo:** tirar das planilhas a parte de compra das "Entregas de importação" e do painel
> financeiro: o processo de importação do pedido ao fornecedor até a mercadoria no estoque, com
> câmbio, custos de nacionalização rateados no custo do lote, pagamentos ao fornecedor e aos
> prestadores como contas a pagar e reserva automática para os pedidos de venda que a importação
> atende. Base do custo de mercadoria (CMV) importada no M15.
> **Nome do módulo:** `importacao`
> **Branch:** `feat/m12-importacoes`

---

## 1. Visão geral

| Tema | Proposta |
|---|---|
| Módulo | Pacote novo `importacao`: `domain/importacao`, `ports/in|out/importacao`, `usecases/importacao`; rotas `/api/v1/importacoes` |
| Unidade | Uma importação = **um embarque** de um fornecedor para uma empresa importadora (uma DI/DUIMP), com vários itens (D7) |
| Empresa | A importadora, ativa e no acesso; tudo herda a `empresa_id` dela (seção 4 do `EMPRESAS.md`) |
| Fornecedor | Do M4, ativo ao criar; a moeda padrão dele só sugere (`domainfornecedor.Moeda`) |
| Estados (D8) | `aberta` → `embarcada` → `desembaracada` → `recebida`; `cancelada` antes do recebimento |
| Valor na moeda | Total por item em centésimos da moeda estrangeira (D12); nunca `float` |
| Câmbio (D1) | Taxa com 6 casas decimais em inteiro (`cambio_micro`, `5,432100` → `5432100`) |
| Pagamento ao fornecedor (D2) | Parcelas na moeda; cada parcela vira conta a pagar no M7 quando o câmbio é fechado |
| Custos (D3, D4) | Frete, seguro, tributos e despesas aduaneiras em reais; cada custo vira conta a pagar no M7 e entra ou não no custo do produto |
| Rateio (D5) | Proporcional ao valor do item em reais (FOB convertido) |
| Recebimento (D7) | Único, num local da empresa importadora: entrada por item com custo unitário rateado (cria o lote do M2) e reserva das quantidades alocadas a pedidos, na mesma transação |
| Pedidos (D6) | Alocação manual de itens de pedidos de importação (M6) da mesma empresa à importação |
| Alertas (D9) | Pedido alocado cujo `prazo_limite` vence antes da chegada prevista; pedido de importação com falta a cobrir sem alocação |
| Permissões (D10) | Financeiro escreve; comercial lê e aloca pedidos; cancelar com conta a pagar gerada exige cancelá-la antes no M7 |
| "Hoje" | Dia no fuso de Brasília, cópia própria em `domain/importacao/data.go` (como `domainreceber.Hoje`) |
| Dinheiro | Reais em centavos `BIGINT`; moeda estrangeira em centésimos `BIGINT` |

---

## 2. Entidades

### 2.1 `importacao`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; importadora, ativa e no acesso |
| `numero` | INT | sequencial por empresa (`importacao_sequencia`); `uq_importacao_empresa_numero` |
| `fornecedor_id` | UUID | FK `fornecedor` `RESTRICT`; ativo ao criar ou trocar |
| `referencia` | VARCHAR(60) NULL | proforma / invoice do fornecedor |
| `moeda` | CHAR(3) | `USD`, `EUR`, `CNY` ou `BRL` (D12) |
| `cambio_micro` | BIGINT NULL | câmbio do recebimento (D1); `> 0` |
| `status` | VARCHAR(14) | `aberta`, `embarcada`, `desembaracada`, `recebida`, `cancelada` |
| `embarque_previsto`, `embarcada_em` | DATE NULL | |
| `chegada_prevista`, `chegada_em` | DATE NULL | `chegada_prevista` alimenta os alertas |
| `desembaracada_em`, `recebida_em` | DATE NULL | data do fato, sem data futura |
| `documento_embarque` | VARCHAR(60) NULL | BL / AWB |
| `declaracao` | VARCHAR(30) NULL | número da DI ou DUIMP, exigido no desembaraço |
| `local_recebimento_id` | UUID NULL | FK `estoque_local`; da empresa, ativo; gravado no recebimento |
| `observacoes` | VARCHAR(2000) | |
| `cancelada_motivo` | VARCHAR(1000) NULL | obrigatório ao cancelar |
| `usuario_id` | UUID | quem criou |
| `versao` | INT | versão otimista (contrato do M4) |
| `criado_em`, `atualizado_em` | TIMESTAMPTZ | |

### 2.2 `importacao_item`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `importacao_id` | UUID | FK |
| `produto_id` | UUID | FK `produto`; ativo ao incluir; único por importação |
| `quantidade` | INT | `> 0`, `int32`; editável só em `aberta` e `embarcada` |
| `valor_moeda_centesimos` | BIGINT | total do item na moeda (`>= 0`; zero = bonificação) |
| `quantidade_recebida` | INT NULL | gravada no recebimento; `0..quantidade` (falta e avaria, D7) |
| `custo_unitario_centavos` | BIGINT NULL | custo rateado gravado no recebimento |
| `movimento_id` | UUID NULL | FK `produto_movimento`: a entrada no estoque |
| `criado_em` | TIMESTAMPTZ | |

### 2.3 `importacao_parcela` — pagamento ao fornecedor (D2)

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `importacao_id` | UUID | FK |
| `descricao` | VARCHAR(120) | "Sinal 30%", "Saldo 70%" |
| `valor_moeda_centesimos` | BIGINT | `> 0` |
| `vencimento_previsto` | DATE | |
| `status` | VARCHAR(14) | `aberta`, `cambio_fechado`, `cancelada` |
| `cambio_micro` | BIGINT NULL | preenchido ao fechar o câmbio |
| `valor_centavos` | BIGINT NULL | `round(valor_moeda × câmbio)`, gravado ao fechar o câmbio |
| `vencimento` | DATE NULL | vencimento da conta a pagar, informado ao fechar o câmbio |
| `solicitacao_id` | UUID NULL | FK `solicitacao`, única: conta a pagar gerada ao fechar o câmbio |
| `usuario_id`, `criado_em`, `atualizado_em` | | |

- Σ parcelas ativas na moeda ≤ Σ itens na moeda (conferido no domínio, inclusive ao alterar os itens).
- Parcela com câmbio fechado é imutável; desfazer = cancelar a conta a pagar no M7 e reabrir a parcela
  (só se a solicitação estiver `cancelado` ou `rejeitado`): a parcela vira `cancelada` e nasce outra `aberta`.

### 2.4 `importacao_custo` — nacionalização (D3, D4)

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `importacao_id` | UUID | FK |
| `tipo` | VARCHAR(30) | `frete_internacional`, `seguro`, `imposto_importacao`, `ipi`, `pis`, `cofins`, `icms`, `taxa_siscomex`, `despachante`, `armazenagem`, `frete_nacional`, `outros` |
| `descricao` | VARCHAR(255) | |
| `fornecedor_id` | UUID | FK `fornecedor`: a quem se paga (transportadora, despachante, Receita) |
| `valor_centavos` | BIGINT | `> 0`, em reais |
| `vencimento` | DATE | |
| `compoe_custo` | BOOLEAN | entra no rateio do custo do produto (D4) |
| `categoria_financeira_id` | UUID | lançável, natureza `despesa` (sugerida: `compra_mercadoria`) |
| `solicitacao_id` | UUID | FK `solicitacao`, única: conta a pagar gerada na mesma transação |
| `posterior` | BOOLEAN | lançado depois do recebimento; nunca compõe o custo |
| `cancelado_em`, `cancelado_por` | | preenchidos juntos ao cancelar |
| `usuario_id`, `criado_em` | | |

- Custo lançado depois do recebimento não muda o lote (lote é imutável, M2): é gravado com
  `compoe_custo = false` e sinalizado como "posterior" na consulta (D3).

### 2.5 `importacao_alocacao` — pedidos atendidos (D6)

| Campo | Tipo | Observações |
|---|---|---|
| `importacao_item_id` | UUID | FK; PK composta |
| `pedido_item_id` | UUID | FK `pedido_item`; PK composta |
| `quantidade` | INT | `> 0` |
| `reservada` | INT NULL | quanto a reserva do recebimento cobriu (0 quando nada) |
| `reserva_id` | UUID NULL | FK `produto_reserva`, única: reserva criada no recebimento |
| `usuario_id`, `criado_em`, `atualizado_em` | | |

- O pedido é de **importação**, `confirmado` ou `parcialmente_entregue`, da **mesma empresa** e do mesmo
  produto do item.
- Σ alocações do item da importação ≤ quantidade dele; Σ alocações do item do pedido (em importações não
  canceladas e ainda não recebidas) ≤ restante do item do pedido.
- Editável até o recebimento; importação cancelada solta as alocações.

### 2.6 `importacao_historico`

Append-only, como `pedido_historico`: `importacao_id`, `acao` (`criada`, `alterada`, `embarcada`,
`desembaracada`, `recebida`, `cancelada`, `parcela_incluida`, `parcela_alterada`, `parcela_removida`,
`parcela_cambio_fechado`, `parcela_reaberta`, `custo_lancado`, `custo_cancelado`, `alocacao_alterada`),
`status_anterior`, `status_novo`, `usuario_id`, `detalhe`, `criado_em`.

### 2.7 `importacao_sequencia`

`empresa_id` (PK, FK `empresa`), `ultimo_numero` INT; mesmo mecanismo da `pedido_sequencia`.

---

## 3. Fluxos

### 3.1 Estados

| Ação | De | Para | Efeito |
|---|---|---|---|
| criar | — | `aberta` | número da empresa; itens |
| alterar | `aberta`, `embarcada` | igual | cabeçalho e itens (com `versao`); fornecedor e moeda só em `aberta` |
| embarcar | `aberta` | `embarcada` | `embarcada_em`, `chegada_prevista`, `documento_embarque` |
| desembaraçar | `embarcada` | `desembaracada` | `chegada_em`, `desembaracada_em`, `declaracao` |
| receber | `desembaracada` | `recebida` | seção 3.4 |
| cancelar | `aberta`, `embarcada`, `desembaracada` | `cancelada` | motivo; recusa (`409`) com conta a pagar gerada que não esteja `cancelada`/`rejeitada` |

Toda transição usa o `UPDATE` condicionado ao status e à `versao` e grava o histórico na mesma
transação.

### 3.2 Câmbio da parcela (D2)

`POST /importacoes/{id}/parcelas/{parcela_id}/cambio { cambio_micro, vencimento, categoria_financeira_id,
versao }`: grava câmbio e valor em reais e gera a conta a pagar (`origem = importacao_parcela`,
`origem_id` = parcela, fornecedor da importação, `pendente_aprovacao`, competência do vencimento, não
estimada) na mesma transação, por `gerarSolicitacaoDeOrigem`.

### 3.3 Custos (D3)

`POST /importacoes/{id}/custos` grava o custo e a conta a pagar (`origem = importacao_custo`) na mesma
transação. Cancelar o custo (marca `cancelado_em`, não apaga) só com a conta a pagar `cancelado`/`rejeitado`;
depois do recebimento, só o custo que não entrou no lote.

### 3.4 Recebimento (D1, D5, D7)

`POST /importacoes/{id}/receber { versao, recebida_em, local_id, cambio_micro, itens: [{ item_id,
quantidade_recebida }] }`:

1. Câmbio do custo (D1): média ponderada das parcelas com câmbio fechado; a parte do valor dos itens
   ainda sem parcela fechada usa o `cambio_micro` informado (obrigatório se houver essa parte).
2. Valor do item em reais = `valor_moeda × câmbio`; custos com `compoe_custo` rateados pelo valor em reais
   (D5), resto do arredondamento no último item; custo unitário = (valor + rateio) ÷ quantidade recebida,
   arredondado. Item com recebida zero não gera entrada e seu valor some do rateio.
3. Uma operação `entrada` por item com quantidade recebida, no local informado (da empresa, ativo), com o
   custo unitário e `documento_ref = IMP-<numero>`: cria o lote do M2.
4. Para cada alocação, reserva (`origem = pedido_item`) o menor entre a alocação, a falta a cobrir do item
   do pedido e o recebido ainda não reservado, encadeada depois da entrada do mesmo produto.
5. Tudo numa transação, pela função interna `gravarOperacaoEstoque` (guarda de `versao_estoque`); mais o
   `pedido_historico` `pendencias_reservadas` dos pedidos atendidos.

### 3.5 Alertas (D9)

`GET /importacoes/alertas?empresa=`:

- `prazo_em_risco`: alocação de importação não recebida cujo pedido tem `prazo_limite` antes de
  `chegada_prevista + folga` (folga padrão 10 dias para desembaraço e entrega, `?folga=`);
- `sem_previsao`: alocação de importação não embarcada sem `chegada_prevista`;
- `sem_importacao`: item de pedido de importação com falta a cobrir maior que o alocado em importações
  abertas.

---

## 4. Endpoints

```
POST   /api/v1/importacoes                                  (financeiro)
GET    /api/v1/importacoes                                  ?empresa= ?fornecedor= ?status= ?numero= ?chegada_ate= (comercial ou financeiro)
GET    /api/v1/importacoes/{id}                             itens, parcelas, custos, alocações e prévia do custo
PUT    /api/v1/importacoes/{id}                             { versao, ... } (financeiro)
POST   /api/v1/importacoes/{id}/embarcar                    { versao, embarcada_em, chegada_prevista, documento_embarque }
POST   /api/v1/importacoes/{id}/desembaracar                { versao, chegada_em, desembaracada_em, declaracao }
POST   /api/v1/importacoes/{id}/receber                     { versao, recebida_em, local_id, cambio_micro?, itens }
POST   /api/v1/importacoes/{id}/cancelar                    { versao, motivo }
GET    /api/v1/importacoes/{id}/historico

POST   /api/v1/importacoes/{id}/parcelas                    { versao, descricao, valor_moeda_centesimos, vencimento_previsto }
PUT    /api/v1/importacoes/{id}/parcelas/{parcela_id}       antes do câmbio
DELETE /api/v1/importacoes/{id}/parcelas/{parcela_id}?versao=   antes do câmbio
POST   /api/v1/importacoes/{id}/parcelas/{parcela_id}/cambio    { versao, cambio_micro, vencimento, categoria_financeira_id }
POST   /api/v1/importacoes/{id}/parcelas/{parcela_id}/reabrir   { versao } com a conta a pagar cancelada ou rejeitada

POST   /api/v1/importacoes/{id}/custos                      { versao, tipo, descricao, fornecedor_id, valor_centavos, vencimento, compoe_custo, categoria_financeira_id }
POST   /api/v1/importacoes/{id}/custos/{custo_id}/cancelar  { versao } com a conta a pagar cancelada ou rejeitada

PUT    /api/v1/importacoes/{id}/alocacoes                   { versao, alocacoes: [{ importacao_item_id, pedido_item_id, quantidade }] } (comercial ou financeiro)
GET    /api/v1/importacoes/{id}/custo-previsto              ?cambio_micro= — prévia do rateio sem gravar
GET    /api/v1/importacoes/alertas                          ?empresa= ?folga=
GET    /api/v1/pedidos/{id}/importacoes                     importações que atendem o pedido
```

---

## 5. Permissões (D10)

- Escrita (criar, alterar, transições, parcelas, câmbio, custos, recebimento) em `financeiro(...)`;
  alocação de pedidos em `comercial(...)` **ou** financeiro; leitura em comercial ou financeiro. Regra
  fina em `usecases/importacao/permissoes.go` (`exigeFinanceiro`, `escopoPorEmpresa`).
- Valores (moeda, câmbio, custos e custo unitário) só para o financeiro; o comercial vê itens, datas,
  status e alocações.
- Escopo por empresa pela importadora; teste de isolamento no usecase e no e2e.

---

## 6. Contrato com os outros módulos

### 6.1 Estoque (M2)

- Entrada pela operação do domínio (`EstoqueProduto.Movimentar` tipo `entrada` com `Custo`) e
  `EstoqueProduto.Aplicar` para encadear a reserva; gravação por `gravarOperacaoEstoque(ctx, tx, ...)`
  dentro da transação do recebimento. **Sem mudança de schema no M2**: o lote nasce do movimento, e a
  ligação fica em `importacao_item.movimento_id`.
- Porta mínima própria (`Carregar` do estado de estoque, `AtivasPorOrigem` de reservas, `Obter` de local).

### 6.2 Pedidos (M6)

- Porta mínima em `ports/out/importacao/` para ler pedido e itens (`Obter`) e a cobertura do item; nenhum
  método novo no port do M6. A reserva no recebimento usa `origem = pedido_item`, então o pedido as
  libera e baixa como as próprias.
- O histórico do pedido ganha a linha `pendencias_reservadas` com o detalhe `importação <numero>`, gravada
  no repositório do M12 na mesma transação.
- Recebimento concorre com entregas e confirmações pelo mesmo `versao_estoque` do produto (`409`).

### 6.3 Contas a pagar (M7)

- `chk_solicitacao_origem`, `NovaOrigem`, `Origem.Sistemica()` e a mensagem de
  `NovaSolicitacaoDeOrigem` ganham `importacao_parcela` e `importacao_custo` (migration do M12), com `origem_id` = id da parcela ou do custo e idempotência pelo
  `uq_solicitacao_origem` já existente.
- Criação por `NovaSolicitacaoDeOrigem` e `gerarSolicitacaoDeOrigem(ctx, tx, ...)` na transação do M12,
  como o M10 faz no fechamento.
- Leitura do status da solicitação (para cancelar a importação, reabrir parcela e remover custo) por
  consulta própria.

### 6.4 Fornecedores, contas e plano de contas (M4, M5)

Portas mínimas com `Obter` (fornecedor `Ativo`, `MoedaPadrao`; categoria `Lancavel()`, `Natureza`).

### 6.5 Caixa (M13) e DRE (M15)

- O M13 concilia os pagamentos das parcelas e dos custos como qualquer conta a pagar.
- O M15 tira o CMV dos lotes; as contas a pagar de categoria `compra_mercadoria` ficam fora do resultado
  (regra do M5). Custo com `compoe_custo = false` entra pela própria categoria (tributo recuperável,
  despesa posterior).

---

## 7. Estrutura de código

```
migrations/000064_create_tabela_importacao              importacao, sequencia, item, historico
migrations/000065_create_tabela_importacao_financeiro   parcela, custo, origens no chk_solicitacao_origem
migrations/000066_create_tabela_importacao_alocacao

internal/core/domain/importacao/      status.go, importacao.go, item.go, cambio.go, valor_moeda.go,
                                      parcela.go, custo.go, tipo_custo.go, rateio.go, alocacao.go,
                                      recebimento.go, historico.go, alerta.go, data.go (+ testes)
internal/core/ports/out/importacao/   importacao_repository.go, consulta_repository.go,
                                      estoque_repository.go, reserva_repository.go, local_repository.go,
                                      pedido_repository.go, fornecedor_repository.go,
                                      categoria_financeira_repository.go, empresa_repository.go,
                                      produto_repository.go, usuario_repository.go, cargo_repository.go
internal/core/ports/in/importacao/    importacao_usecase.go, financeiro_usecase.go, consulta_usecase.go
internal/core/usecases/importacao/    idem + recebimento.go, permissoes.go (+ testes e fakes)
internal/core/domain/solicitacao/     origem.go (duas origens)
internal/adapter/postgres/            importacao_repo.go, importacao_consulta_repo.go
internal/adapter/http/dto/importacao/ + handlers/importacao/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

Numeração a partir da `000064`; renumerar antes do merge se outra sessão mergear antes (regra 4 da
seção 3 do `ROADMAP.md`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Importação e itens (migration 000064): domínio, estados até `desembaracada`, cancelamento, repositório com sequência e histórico, usecase com escopo por empresa, rotas, e2e + `API.md` | concluída |
| **2** | Parcelas, câmbio e custos com conta a pagar no M7 (migration 000065), cancelamento com conta gerada, prévia do custo; integração de idempotência; e2e + `API.md` | concluída |
| **3** | Alocação de pedidos (migration 000066), recebimento com entrada, lote e reserva na mesma transação; integração de concorrência com entrega; e2e + `API.md` | concluída |
| **4** | Alertas e `GET /pedidos/{id}/importacoes`; e2e + `API.md` | concluída |
| **5** | Frontend (lista, detalhe, parcelas, custos, recebimento, alertas) | pendente |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Moeda do M4**: `domainfornecedor.Moeda` (`BRL`, `USD`, `EUR`, `CNY`), reaproveitada.
- [x] **Conta a pagar gerada por outro módulo**: `NovaSolicitacaoDeOrigem` + `gerarSolicitacaoDeOrigem`
      (fechado pelo M9, usado pelo M10).
- [x] **Reserva de pedido** por `produto_reserva.origem = pedido_item` (M6); só o pedido libera ou baixa.
- [x] **Estoque entre CNPJs** segue bloqueado (seção 6 do `ROADMAP.md`): por isso a alocação só aceita
      pedido da mesma empresa importadora (D6).
- [x] **`compra_mercadoria` fora do resultado** (M5): o custo da importação chega à DRE pelo CMV dos lotes.
- [x] **Fase 1 — colunas por fase** (implementação): a migration 000064 cria só o que a fase 1 grava; as
      colunas do recebimento (`cambio_micro`, `recebida_em`, `local_recebimento_id` na importação;
      `quantidade_recebida`, `custo_unitario_centavos`, `movimento_id` no item) e as ações novas do histórico
      entram nas migrations das fases 2 e 3. O `chk_importacao_status` já aceita `recebida`.
- [x] **Fase 1 — valores só para o financeiro** (implementação): o usecase devolve a importação com
      `ComValores` (`portsin.Visao` e `portsin.Lista`); para o comercial o JSON omite
      `valor_total_moeda_centesimos` e `itens[].valor_moeda_centesimos`. Leitura em `protegido(...)` com a
      checagem de comercial ou financeiro no usecase (não há wrapper de rota para os dois perfis).
- [x] **Fase 1 — itens estáveis** (implementação): o `PUT` substitui os itens, mas o item de um produto que
      continua mantém o `id` (upsert por `id` e exclusão só dos que saíram), para as alocações da fase 3
      apontarem para ele.
- [x] **Fase 1 — moeda na alteração** (implementação): moeda vazia mantém a da importação (ou a padrão do
      fornecedor novo, quando ele é trocado com a importação aberta); fornecedor e produto já gravados não
      são conferidos de novo, e a troca de fornecedor trava o novo com `FOR SHARE` na transação.
- [x] **Fase 4 — alertas** (implementação): consulta própria (`ImportacaoConsultaRepository`) traz as alocações de
      importações abertas e os itens de pedido de importação com falta; a classificação (prazo em risco, sem
      previsão, sem importação) é do domínio (`ClassificarAlertas`). `folga` de 0 a 90 dias, padrão 10; pedido sem
      prazo-limite (importação sem entrada paga) não gera prazo em risco. Escopo pela empresa da importação e, em
      `sem_importacao`, pela empresa do pedido.
- [x] **Fase 4 — importações do pedido** (implementação): `GET /pedidos/{id}/importacoes` fica no módulo de
      importação (registrado no bloco dele em `rotas.go`) e confere só o acesso à empresa do pedido; o filtro de
      representante com login do M6 não se aplica aqui, porque a rota não mostra valores nem cliente.
- [x] **Fase 3 — leitura do pedido** (implementação): o M12 lê pedido, item, reservas ativas e o alocado em outras
      importações por consulta própria (`ImportacaoRepository.ItensDePedido`); **nenhum método novo no port nem no
      repositório do M6**. O histórico `pendencias_reservadas` do pedido é gravado pela função interna
      `inserirHistoricoDoPedido` na transação do recebimento, sem avançar a `versao` do pedido.
- [x] **Fase 3 — corrida de alocação** (implementação): gravar alocações trava os `pedido_item` envolvidos
      (`FOR UPDATE`, em ordem de id) e reconfere no banco a soma das importações abertas contra o restante (`409`).
      O PUT de alocações fica em `protegido(...)` com a checagem de comercial ou financeiro no usecase.
- [x] **Fase 3 — recebimento** (implementação): item omitido é recebido inteiro; a ordem das reservas é pelo
      prazo-limite do pedido (sem prazo por último, depois a ordem de alocação); pedido fora de atendimento na
      leitura não recebe reserva, e se sair de atendimento até a gravação (`FOR SHARE` nos pedidos atendidos) o
      recebimento inteiro é recusado com `409`. A falta (`quantidade_recebida` menor) encarece a unidade: o valor
      em reais do item continua o mesmo; item com zero não gera entrada e não recebe custos.
- [x] **Fase 3 — travas** (implementação): ordem importação → pedidos → produtos (versão do estoque), a mesma
      direção do M6 (pedido → produto), sem ciclo; entrada e reserva do mesmo produto são encadeadas com
      `EstoqueProduto.Aplicar` e gravadas por `gravarOperacaoEstoque`.
- [x] **Fase 2 — parcelas e custos no agregado** (implementação): parcelas e custos são lidos com a importação
      (`Importacao.Parcelas`, `Importacao.Custos`, com o status da conta a pagar por `JOIN` em `solicitacao`) e
      toda operação passa pela guarda de `versao` da importação, que serializa as escritas. Nenhum método novo
      no port nem no repositório do M7: a conta a pagar nasce por `gerarSolicitacaoDeOrigem` na transação do M12.
- [x] **Fase 2 — reabrir sem apagar** (implementação): o `uq_solicitacao_origem` impede uma segunda conta com a
      mesma `origem_id`, então reabrir a parcela a marca `cancelada` (guardando a conta antiga) e cria outra
      `aberta`; o custo é cancelado por marca (`cancelado_em`), rota `POST .../custos/{custo_id}/cancelar` em vez
      do `DELETE` previsto. Só a parcela `aberta` é apagada de verdade (`DELETE ...?versao=`): não tem conta.
- [x] **Fase 2 — conferência das contas na transação** (implementação): cancelar a importação, reabrir parcela
      e cancelar custo travam as solicitações envolvidas com `FOR SHARE` e recusam (`409`) se alguma não estiver
      `cancelado` ou `rejeitado`, além da checagem do domínio.
- [x] **Fase 2 — arredondamento** (implementação): valor em reais = centésimos × `cambio_micro` ÷ 10⁶, meio
      para cima, em `math/big`; a mercadoria é dividida pelos itens com piso e o resto no último item com peso; os
      custos, pelo valor em reais dos itens com quantidade (pela quantidade quando tudo vale zero); custo
      unitário arredondado meio para cima. Importação em `BRL` usa câmbio 1.
- [x] **Fase 2 — padrões** (implementação): `compoe_custo` vem `true` quando omitido (D4); descrição vazia do
      custo usa o nome do tipo; a observação da conta a pagar é "importação N — descrição". O custo previsto é só
      do financeiro e calcula com as quantidades da importação (o recebimento usará as recebidas).
- [x] **Fase 1 — embarque** (implementação): o embarque exige a chegada prevista; se ela ficar antes do
      embarque previsto antigo, o embarque previsto é limpo (a previsão perdeu o sentido).

## 10. Decisões do escopo

Respondidas pelo usuário (outubro/2026): todas as recomendações aceitas.

- [x] **D1 — Câmbio do custo do produto.** Decidido: média ponderada das parcelas com câmbio fechado
      (o que de fato se pagou), e o câmbio informado no recebimento para o valor ainda sem parcela fechada.
      Alternativas descartadas: câmbio da DI (fiscal) para tudo; câmbio digitado no recebimento para tudo.
- [x] **D2 — Quando a parcela do fornecedor vira conta a pagar.** Decidido: ao **fechar o câmbio**
      (valor em reais exato, sem estimativa); antes disso ela só aparece na importação e fica fora da
      projeção do M13. Alternativa descartada: gerar já no cadastro como `estimado` com câmbio previsto — exigiria
      editar solicitação de origem sistêmica no M7, hoje fixa.
- [x] **D3 — Custos lançados depois do recebimento.** Decidido: não mexem no lote (o M2 não edita lote);
      viram despesa pela categoria escolhida e aparecem como "posteriores". Alternativa descartada: lançamento de
      ajuste de custo no M2 (mudança no M2 e no CMV do M15).
- [x] **D4 — Tributos recuperáveis.** Decidido: flag `compoe_custo` por custo, padrão `true` para tudo;
      o financeiro desmarca o que a empresa credita (ICMS, IPI, PIS/COFINS conforme o regime), conferido
      com a contabilidade. Alternativa descartada: padrão automático pelo regime tributário da empresa (simples →
      tudo compõe; presumido → ICMS e IPI não; real → ICMS, IPI, PIS e COFINS não).
- [x] **D5 — Critério de rateio.** Decidido: proporcional ao valor do item em reais. Alternativas descartadas: por
      quantidade; por custo atribuído a item específico (ex.: imposto por NCM).
- [x] **D6 — Vínculo com pedidos.** Decidido: alocação manual de quantidade por item de pedido de
      importação da mesma empresa, reservada no recebimento; o que sobrar fica livre no estoque.
      Alternativas descartadas: só reservar automaticamente as pendências de todos os pedidos da empresa no recebimento
      (como o `reservar-pendencias`), sem alocação; ou alocação também de pedido de pronta entrega.
- [x] **D7 — Recebimento.** Decidido: **único** por importação (uma importação = um embarque/DI), com
      quantidade recebida por item para registrar falta e avaria. Alternativa descartada: recebimentos parciais
      (vários containers), com rateio a cada recebimento.
- [x] **D8 — Estados.** Decidido: `aberta`, `embarcada`, `desembaracada`, `recebida`, `cancelada` (o
      "em trânsito" do roadmap é `embarcada`). Alternativa descartada: os cinco estados do roadmap (`pedido`,
      `embarcado`, `em_transito`, `desembaracado`, `recebido`).
- [x] **D9 — Alertas.** Decidido: prazo em risco (prazo-limite do pedido antes da chegada prevista +
      10 dias de folga), alocação sem previsão de chegada e pedido de importação com falta a cobrir sem
      importação. Alternativa descartada: só o prazo em risco, sem folga.
- [x] **D10 — Permissões.** Decidido: o financeiro escreve tudo; o comercial lê (sem valores) e aloca
      pedidos. Alternativa descartada: perfil novo de "compras" (flag no cargo).
- [x] **D11 — Cancelamento com conta a pagar.** Decidido: recusar enquanto houver conta a pagar gerada
      não cancelada nem rejeitada (o financeiro resolve no M7 antes). Alternativa descartada: cancelar as solicitações
      pendentes junto e recusar só as aprovadas ou pagas.
- [x] **D12 — Moeda e valor.** Decidido: valor **total** do item na moeda (como na invoice), custo
      unitário derivado; `BRL` aceito com câmbio 1. Alternativa descartada: preço unitário com 4 casas.
- [x] **D13 — Importações em andamento hoje.** Decidido: cadastradas à mão pelo financeiro no novo
      módulo (sem migração de dados), com as parcelas já pagas lançadas com câmbio fechado. Alternativa descartada:
      carga por CSV.
- [x] **D14 — Pedido de importação sem importação.** Decidido: nada muda no M6 (o pedido segue sendo
      vendido antes de existir a importação; o alerta `sem_importacao` cobre o furo). Alternativa descartada: exigir
      alocação para confirmar o pedido de importação.

## 11. Decisões pendentes

Sem abertos.
