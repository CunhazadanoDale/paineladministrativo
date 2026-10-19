# Escopo — M7 Contas a pagar

> **Status:** escopo fechado — decisões D1 a D6 respondidas pelo usuário (outubro/2026); API
> e front concluídos (fases 1 a 6).
> **Objetivo:** tirar o "painel financeiro (contas a pagar)" da planilha: a solicitação de
> pagamento passa a ser a conta a pagar do grupo, com fornecedor (M4), categoria financeira e
> conta bancária (M5), competência, recorrência, adiamento, edição, estorno e visões de
> vencimento. Base para M6 (devolução), M9 (recompra), M10 (comissões), M12, M13, M14 e M15.
> **Nome do módulo:** `solicitacao` (evolução do pacote existente; sem módulo novo)
> **Branch:** `feat/m7-contas-a-pagar`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Módulo | **Evolução de `solicitacao`**: mesma tabela, mesmas rotas `/api/v1/solicitacoes`, fluxo solicitante → aprovador → financeiro inalterado |
| Contrato atual | **Só acréscimos**: todo campo novo do `POST` é opcional, nenhum campo muda de nome e nenhuma resposta perde campo (o front da F0a está corrigindo esse formulário agora) |
| Vencimento | **É o `prazo_pagamento` atual** (`DATE`); o nome do campo não muda |
| Classificação | Fornecedor, categoria financeira e competência **opcionais no `POST`**; o financeiro completa em `aprovado` (D1) |
| Pagar exige | Categoria lançável de natureza `despesa`, competência e **conta bancária** (D2) |
| Valor previsto | Flag `estimado`: o valor é previsão (conta de consumo, recorrência variável) até o pagamento registrar o efetivo |
| Recorrência | `despesa_recorrente` gera as solicitações de uma competência por rotina idempotente, disparada pela API (sem agendador interno) |
| Adiar | Muda o vencimento com motivo; não muda o status; vai para o histórico |
| Edição | `PUT` só em `pendente_aprovacao`, pelo solicitante ou administrador, com versão otimista |
| Estorno | Lançamento inverso do pagamento (append-only); a solicitação volta para `aprovado` |
| Histórico | Ganha a coluna `evento` para registrar o que não é troca de status (edição, adiamento, classificação, estorno) |
| Origem | `origem` + `origem_id`, no padrão de `produto_reserva` (M2): `manual` e `recorrencia` agora; M6, M9, M10 e M14 acrescentam os seus |
| Escopo por empresa | Já existe na solicitação (M1); a recorrência copia o padrão da seção 4 do `EMPRESAS.md` |
| Concorrência | `versao` na solicitação e na recorrência (contrato do M4); transições continuam com guarda de status |
| Regime | DRE (M15) por **competência**; fluxo de caixa (M13) pelo `pago_em` (D5) |
| Formas de pagamento | `pix`, `cartao`, `boleto` + **`transferencia`** e **`debito_automatico`** (D6); `forma_pagamento` passa a `VARCHAR(20)` |
| Notificação | **Contador de pendências** para o badge do front; e-mail fora do M7 (D4) |

---

## 2. Entidades

### 2.1 `solicitacao` (colunas novas)

| Campo | Tipo | Observações |
|---|---|---|
| `fornecedor_id` | UUID NULL | FK `fornecedor` `RESTRICT`; fornecedor ativo na gravação |
| `categoria_financeira_id` | UUID NULL | FK `categoria_financeira` `RESTRICT`; categoria `Lancavel()` e de natureza `despesa` na gravação |
| `competencia` | DATE | sempre o dia 1 do mês (`chk_solicitacao_competencia`); mês a que a despesa pertence na DRE; omitida no `POST` = mês do `prazo_pagamento` |
| `estimado` | BOOLEAN | default `false`; valor previsto, não confirmado |
| `origem` | VARCHAR(20) | default `manual`; `chk_solicitacao_origem` em (`manual`, `recorrencia`) |
| `origem_id` | UUID NULL | id da recorrência quando `origem = recorrencia`; `chk_solicitacao_origem_id` amarra as duas |
| `versao` | INT | começa em `1`; avança em toda gravação, inclusive nas transições |

- Solicitações já existentes ficam com `fornecedor_id` e `categoria_financeira_id` nulos;
  `competencia` é preenchida pela migration com o mês do `prazo_pagamento` e a coluna é
  `NOT NULL`. Uma `aprovado` sem categoria precisa ser classificada antes de ser paga
  (seção 3.2).
- `idx_solicitacao_categoria` saiu com o nome completo `idx_solicitacao_categoria_financeira`; o
  `uq_solicitacao_recorrencia` entra com a recorrência (fase 4).
- A criação também passou a conferir a empresa ativa na transação (`FOR SHARE`), fechando para a
  solicitação a corrida do item 10 do `PENDENCIAS.md`.
- Índices: `idx_solicitacao_vencimento (empresa_id, status, prazo_pagamento, id)` para as visões,
  `idx_solicitacao_fornecedor`, `idx_solicitacao_categoria` e
  `uq_solicitacao_recorrencia (origem_id, competencia) WHERE origem = 'recorrencia'`, que é o que
  torna a geração idempotente.
- Fornecedor e categoria inativos depois da gravação não invalidam a solicitação; a regra de
  ativo vale para gravar (criar, editar, classificar) e, para a categoria, também para **pagar**:
  categoria desativada depois da classificação exige reclassificar antes do pagamento.
- Em solicitação com `origem` diferente de `manual` a `competencia` é **fixa** (editar e classificar
  recusam outra; o `PUT` sem competência mantém a gravada), porque ela é a chave de idempotência
  da geração.
- As checagens que dependem de outra linha (empresa ativa, fornecedor ativo, categoria lançável,
  conta ativa da mesma empresa) rodam na transação da escrita com `FOR SHARE` na linha de que
  dependem, como no M5 (seção 3 do `FINANCEIRO.md`).

### 2.2 `pagamento` (mudanças) e `pagamento_estorno`

`pagamento` ganha `conta_bancaria_id UUID` (FK `conta_bancaria` `RESTRICT`; conta ativa e da
mesma empresa da solicitação, empresa ativa) e `usuario_id UUID` (quem registrou). A conta é
**obrigatória** em todo pagamento novo (D2); os pagamentos antigos ficam com a coluna nula.

`uq_pagamento_solicitacao` é trocada por `uq_pagamento_solicitacao_vigente` parcial: só um
pagamento **não estornado** por solicitação. Pagamento nunca é alterado nem apagado.

| Campo (`pagamento_estorno`) | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `pagamento_id` | UUID | FK `pagamento` `RESTRICT`; `uq_pagamento_estorno_pagamento` |
| `motivo` | VARCHAR(500) | obrigatório |
| `estornado_em` | TIMESTAMPTZ | data do fato; não pode estar no futuro nem antes do `pago_em` |
| `usuario_id` | UUID | quem registrou |
| `criado_em` | TIMESTAMPTZ | |

O "não estornado" do índice parcial é mantido por uma coluna `estornado BOOLEAN` em `pagamento`,
gravada na mesma transação do estorno (única exceção à imutabilidade da linha, com
`UPDATE ... WHERE NOT estornado`); o motivo e a data ficam só em `pagamento_estorno`. Índices
novos: `idx_pagamento_solicitacao` (substitui o índice do `UNIQUE` antigo) e
`idx_pagamento_conta_bancaria (conta_bancaria_id, pago_em)` para a conciliação do M13.

As checagens da conta (existe, mesma empresa, ativa) e da empresa ativa rodam na transação do
pagamento com `FOR SHARE`. O estorno devolve o objeto da solicitação (`200`), como as outras
transições, para o cliente ler a versão nova.

### 2.3 `solicitacao_historico.evento`

`evento VARCHAR(20) NOT NULL`: `criada`, `aprovada`, `rejeitada`, `cancelada`, `paga`,
`editada`, `classificada`, `adiada`, `estornada`. A migration preenche as linhas antigas a partir
de `para_status`. Eventos sem troca de status gravam `de_status = para_status`. A `descricao`
continua livre e traz o detalhe (vencimento antigo e novo, motivo, campos alterados).

### 2.4 `despesa_recorrente`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; não muda |
| `fornecedor_id` | UUID NULL | FK `fornecedor` `RESTRICT` |
| `categoria_financeira_id` | UUID | obrigatória; lançável, natureza `despesa` |
| `descricao` | VARCHAR(1000) | vira a `observacao` da solicitação gerada |
| `valor_centavos` | BIGINT | `> 0` |
| `estimado` | BOOLEAN | valor variável (energia, água); a solicitação nasce `estimado = true` |
| `forma_pagamento` | VARCHAR(20) | mesmo domínio da solicitação |
| `dia_vencimento` | SMALLINT | `1..31`; mês mais curto vence no último dia |
| `vencimento_mes_seguinte` | BOOLEAN | `false` vence no mês da competência; `true` no mês seguinte |
| `inicio` / `fim` | DATE / DATE NULL | competências (dia 1); gera só dentro do intervalo |
| `ativa` | BOOLEAN | inativa não gera |
| `versao` | INT | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Geração** (`POST /api/v1/despesas-recorrentes/gerar?competencia=AAAA-MM`): para cada recorrência
ativa, dentro do intervalo e de empresa ativa no acesso do usuário, cria a solicitação da
competência com `origem = recorrencia`; a que já existe é pulada pelo `uq_solicitacao_recorrencia`
(`INSERT ... ON CONFLICT DO NOTHING`). Responde quantas foram geradas e quantas já existiam.
O solicitante da gerada é quem disparou a geração, e ela nasce em `pendente_aprovacao` (D3).
A resposta traz `geradas` (ids), `ja_existentes` e `recusadas` (recorrência e motivo): uma recorrência
com fornecedor inativo ou categoria que deixou de ser lançável é recusada sem impedir as outras. Cada
solicitação gerada é gravada numa transação própria, com a recorrência travada (`FOR SHARE`), e a
exclusão trava a recorrência (`FOR UPDATE`) antes de conferir se ela já gerou solicitações.
O vencimento calculado pode estar no passado (gerar uma competência atrasada é registrar uma
conta vencida) e, por isso, não passa pela regra "não anterior a hoje" do `POST`.

---

## 3. Fluxos novos

### 3.1 Edição (`PENDENCIAS.md` item 7)

`PUT /api/v1/solicitacoes/{id}` substitui valor, vencimento, observação, forma, fornecedor,
categoria, competência e `estimado`, com `versao` no corpo. Só em `pendente_aprovacao`, pelo
solicitante ou administrador. Anexos continuam pelo fluxo atual. Grava evento `editada` com os
campos alterados. Depois de aprovada, corrigir = cancelar e criar outra. Corpo igual ao gravado
não avança a versão nem grava histórico; campo omitido volta ao padrão do `POST` (competência =
mês do vencimento).

### 3.2 Classificação

`PATCH /api/v1/solicitacoes/{id}/classificacao` (financeiro) com `fornecedor_id`,
`categoria_financeira_id`, `competencia`, `versao`, em `aprovado`. É o caminho para classificar as
solicitações antigas e as que o solicitante deixou sem categoria. Evento `classificada`.

### 3.3 Adiar

`POST /api/v1/solicitacoes/{id}/adiar` (financeiro) com `vencimento`, `motivo`, `versao`, em
`aprovado`. O novo vencimento precisa ser posterior ao atual e não anterior a hoje. Evento
`adiada` com o vencimento antigo, o novo e o motivo. A competência não muda.

### 3.4 Pagamento e estorno

- `POST /{id}/pagamento` ganha `conta_bancaria_id` e passa a exigir categoria e competência na
  solicitação (`400 solicitação sem categoria financeira; classifique antes de pagar`).
- `POST /{id}/pagamento/estorno` (financeiro) com `motivo`, `estornado_em`, `versao`, em `pago`:
  grava o estorno, marca o pagamento como estornado e volta a solicitação para `aprovado`
  (transição nova `pago → aprovado`, só por este caminho). A partir daí ela pode ser paga de
  novo ou cancelada.
- `GET /{id}/pagamento` continua devolvendo o pagamento vigente; `GET /{id}/pagamentos` lista
  todos, com o estorno de cada um.

### 3.5 Pendências (`PENDENCIAS.md` item 6)

Decisão D4: `GET /api/v1/solicitacoes/pendencias` devolve os contadores do usuário
(`aprovacao`, `financeiro`, `vencidas`), cada um só quando o perfil permite (`null` fora do
perfil, para o front não mostrar badge de fila que o usuário não vê), contados com o mesmo filtro
da listagem (`SolicitacaoRepository.Contar`). E-mail fica fora.

---

## 4. Endpoints

### Solicitações (acréscimos)
```
PUT    /api/v1/solicitacoes/{id}                      edição em pendente_aprovacao (solicitante ou administrador)
PATCH  /api/v1/solicitacoes/{id}/classificacao        classificação em aprovado (financeiro)
POST   /api/v1/solicitacoes/{id}/adiar                novo vencimento + motivo (financeiro)
POST   /api/v1/solicitacoes/{id}/pagamento/estorno    estorno do pagamento vigente (financeiro)
GET    /api/v1/solicitacoes/{id}/pagamentos           pagamentos e estornos (quem pode ver)
GET    /api/v1/solicitacoes/pendencias                contadores do usuário (autenticado)
GET    /api/v1/solicitacoes                           novos filtros: ?fornecedor= ?categoria= ?competencia=
                                                      ?vencimento_de= ?vencimento_ate= ?vencidas=true ?origem=
```

- `vencidas=true` = `prazo_pagamento < hoje` e status `pendente_aprovacao` ou `aprovado`;
  "vencido" é derivado, nunca gravado.
- Objeto de solicitação ganha `fornecedor_id`, `categoria_financeira_id`, `competencia`
  (`AAAA-MM`), `estimado`, `origem`, `origem_id`, `versao` e `vencida`.

### Visões do financeiro
```
GET    /api/v1/contas-a-pagar/resumo   ?de= ?ate= ?empresa= (financeiro)
```

Totais e quantidades por **semana** (segunda a domingo, pelo vencimento) e por **empresa**, em
três colunas sem sobreposição: `previsto` (`pendente_aprovacao` a vencer), `a_pagar` (`aprovado` a
vencer) e `vencido` (as duas, vencidas); mais o total de `estimado`. As vencidas antes do início
do período saem à parte em `atrasadas`, por empresa, e entram no total de vencido. Só agregado: o
financeiro continua sem ver o detalhe de `pendente_aprovacao`. Consulta em SQL no repositório
(`ContasAPagarResumoRepository`, `date_trunc('week')`), no padrão do `resumo_usecase` do estoque.
Período padrão de 8 semanas a partir da segunda-feira atual; intervalo máximo de 1 ano.

### Despesas recorrentes
```
POST   /api/v1/despesas-recorrentes                 criar (financeiro)
GET    /api/v1/despesas-recorrentes                 ?empresa= ?ativa= ?fornecedor= + paginação (financeiro)
GET    /api/v1/despesas-recorrentes/{id}            detalhe (financeiro)
PUT    /api/v1/despesas-recorrentes/{id}            campos editáveis + versao (financeiro)
PATCH  /api/v1/despesas-recorrentes/{id}/ativar     { "versao": n } (financeiro)
PATCH  /api/v1/despesas-recorrentes/{id}/desativar  { "versao": n } (financeiro)
DELETE /api/v1/despesas-recorrentes/{id}            só sem solicitação gerada (administrador)
POST   /api/v1/despesas-recorrentes/gerar           ?competencia=AAAA-MM ?empresa= (financeiro)
```

Editar a recorrência não altera solicitações já geradas.

---

## 5. Permissões

- Perfis existentes, sem flag nova: solicitante (qualquer autenticado), aprovador designado,
  financeiro (`cargo.financeiro`, criado no M4) e administrador.
- `rotas.go`: rotas novas de solicitação seguem `protegido(...)` com a regra fina no usecase, como
  as atuais; `despesas-recorrentes` e `contas-a-pagar` usam `financeiro(...)`; `DELETE` de
  recorrência usa `administrador(...)`.
- Regras finas em `usecases/solicitacao/permissoes.go` (já existe).
- Escopo por empresa (seção 4 do `EMPRESAS.md`): toda rota nova confere o acesso; recorrência com
  teste de isolamento no usecase e no e2e; a geração só cria para empresas do acesso.
- Segregação mantida: quem dispara a geração não aprova a solicitação gerada (é o solicitante).

---

## 6. Contrato com os módulos seguintes

Portas mínimas declaradas **neste** módulo (`ports/out/solicitacao/`), satisfeitas no `main.go`
pelos repositórios do M4 e do M5:

- `FornecedorRepository.Obter` (precisa de `Ativo`), `CategoriaFinanceiraRepository.Obter`
  (`Lancavel()`, `Natureza`), `ContaBancariaRepository.Obter` (`EmpresaID`, `Ativa`). A trava
  `FOR SHARE` das checagens fica no próprio `SolicitacaoRepository`, dentro da transação da escrita.

Para M6 (devolução), M9 (recompra), M10 (comissão), M12 e M14 (parcela):

- Cada um acrescenta o próprio valor em `chk_solicitacao_origem` (migration do módulo) e grava
  `origem_id` com o id do seu registro; a idempotência é um índice único parcial do módulo, como
  `uq_solicitacao_recorrencia`.
- A criação passa por uma porta mínima declarada no consumidor, satisfeita por um método novo do
  `SolicitacaoRepository` que grava solicitação + histórico na mesma transação. **Fechada pelo M9**
  (outubro/2026): `domainsolicitacao.NovaSolicitacaoDeOrigem(DadosSolicitacaoDeOrigem, agora)` monta a
  solicitação em `pendente_aprovacao`, já classificada, e o histórico `criada`;
  `SolicitacaoRepository.GerarDeOrigem(ctx, solicitacao, historico) (bool, error)` grava numa
  transação própria, idempotente por `uq_solicitacao_origem (origem, origem_id) WHERE origem NOT IN
  ('manual', 'recorrencia')`; quem precisa gravar junto com o próprio registro usa a função interna
  `gerarSolicitacaoDeOrigem(ctx, tx, ...)` do pacote `postgres`, como o M9. Cada consumidor
  acrescenta o seu valor em `chk_solicitacao_origem` e em `Origem.Sistemica()`.
- Solicitação de origem sistêmica nasce com categoria e competência, então pode ser paga sem
  classificação manual.
- O M13 concilia o extrato com `pagamento` pela `conta_bancaria_id` e pelo `pago_em`, e o estorno
  com `pagamento_estorno`. O M15 usa `competencia` + `categoria_financeira.linha_dre` (regime de
  competência, decisão D5) e ignora pagamento estornado.

---

## 7. Estrutura de código

```
migrations/000040_add_classificacao_solicitacao        fornecedor, categoria, competência, estimado, origem, versão
migrations/000041_add_evento_solicitacao_historico
migrations/000042_add_conta_estorno_pagamento          conta, usuário, estornado + pagamento_estorno
migrations/000043_create_tabela_despesa_recorrente

internal/core/domain/solicitacao/    competencia.go, origem.go, evento.go, estorno.go, adiamento.go,
                                     despesa_recorrente.go, dia_vencimento.go, versao.go (+ testes);
                                     solicitacao.go ganha Editar, Classificar, Adiar, Estornar
internal/core/ports/out/solicitacao/ fornecedor_repository.go, categoria_financeira_repository.go,
                                     conta_bancaria_repository.go, despesa_recorrente_repository.go
internal/core/ports/in/solicitacao/  solicitacao_usecase.go (métodos novos), despesa_recorrente_usecase.go,
                                     resumo_usecase.go
internal/core/usecases/solicitacao/  idem (+ testes, fakes e cenário)
internal/adapter/postgres/           solicitacao_repo.go, despesa_recorrente_repo.go, contas_a_pagar_resumo_repo.go
internal/adapter/http/dto/solicitacao/ + handlers/solicitacao/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

- Numeração a partir da `000040` (a `main` está na `000039`); renumerar antes do merge se outra
  sessão (M6) mergear primeiro (regra 4 da seção 3 do `ROADMAP.md`).
- O domínio continua importando só `domain`; fornecedor, categoria e conta entram pelo usecase.

---

## 8. Fases de implementação

Cada fase atravessa todas as camadas (domínio → e2e + `API.md`) e termina com os gates verdes,
para a branch estar sempre mergeável.

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Classificação: migrations 000040–000041, fornecedor/categoria/competência/estimado/origem/versão, formas `transferencia` e `debito_automatico`, evento no histórico, portas mínimas do M4 e do M5, `PATCH /classificacao`, novos filtros | concluída |
| **2** | Edição (`PUT`) e adiamento, com versão otimista e eventos | concluída |
| **3** | Pagamento com conta bancária obrigatória e exigência de classificação; estorno e `GET /pagamentos` (migration 000042). Quebra o modal de pagamento do front atual até a fase 6 | concluída |
| **4** | Despesas recorrentes: CRUD + geração idempotente (migration 000043), isolamento por empresa | concluída |
| **5** | Visões: resumo por semana/empresa e contador de pendências | concluída |
| **6** | Frontend (depois do F0a): campos novos no formulário, classificação, adiar, estorno, recorrências, resumo e badge | concluída |

---

## 9. Decisões fechadas

Revisão de outubro/2026 (depois da fase 5), corrigida na própria branch:

- [x] **Comprovante reaproveitável depois do estorno**: `uq_pagamento_comprovante` virou
      `uq_pagamento_comprovante_vigente` (só pagamentos não estornados).
- [x] **Competência fixa nas geradas** (seção 2.1) e regeração que confere a existência antes de
      validar a classificação: competência já gerada sempre conta como `ja_existentes`.
- [x] **Editar solicitação vencida** sem mudar o vencimento.
- [x] **Pagar exige categoria lançável**, conferida também na transação.
- [x] **Ordem dos erros**: estado errado (`409`) antes de validar o corpo em editar, classificar,
      adiar e pagar.
- [x] **Transições guardam status e versão** (`versao = lida`), e o aprovar aceita `versao`
      opcional no corpo para não aprovar um valor editado depois que o aprovador abriu a tela.
- [x] **Rollbacks seguros**: os `down` da 000040 e da 000042 recusam com mensagem quando há dados
      que eles corromperiam (formas novas, pagamentos estornados).
- [x] **Travas na mesma ordem**: a alteração da recorrência trava a recorrência antes de fornecedor
      e categoria, como a geração; o vínculo do comprovante usa `ON CONFLICT (solicitacao_id,
      arquivo_id)` e não engole conflito com outra solicitação.
- [x] `idx_solicitacao_em_aberto` para o resumo e o contador de quem vê todas as empresas.

Ficaram registrados, sem mudança: "hoje" em UTC (convenção anterior ao M7, item 11 do
`PENDENCIAS.md`); editar e cancelar pelo solicitante não reconferem o acesso à empresa (mesma regra
do cancelar desde o M1); a geração que falha no meio por erro de banco devolve só o erro (rodar de
novo é seguro, por ser idempotente).

- [x] **Evolução de `solicitacao`**, sem módulo novo nem rota nova para a conta a pagar em si.
- [x] **Contrato só com acréscimos** no `POST` de solicitação.
- [x] **`prazo_pagamento` é o vencimento**; sem renomear.
- [x] **Competência no dia 1 do mês**, preenchida para as antigas a partir do vencimento.
- [x] **Estorno append-only** com retorno a `aprovado`; pagamento nunca é apagado.
- [x] **`evento` no histórico** para mudanças sem troca de status.
- [x] **Origem no padrão do M2** (`origem` + `origem_id`), estendida por cada módulo consumidor.
- [x] **Geração disparada pela API**, sem agendador interno; idempotente por índice único.
- [x] **Resumo só agregado** para o financeiro, sem expor o detalhe de `pendente_aprovacao`.
- [x] **D1 — Classificação pelo financeiro**: fornecedor, categoria e competência opcionais no
      `POST`; o financeiro completa em `aprovado`; o pagamento exige categoria e competência.
- [x] **D2 — Conta bancária obrigatória no pagamento** já na fase 3; o modal de pagamento do front
      atual quebra até a fase 6 (sessão da F0a avisada pelo usuário).
- [x] **D3 — Recorrência gera em `pendente_aprovacao`**, aprovada todo mês.
- [x] **D4 — Contador de pendências** para badge; e-mail fica fora do M7 (segue no item 6 do
      `PENDENCIAS.md` como evolução).
- [x] **D5 — DRE por competência**; fluxo de caixa pelo `pago_em`.
- [x] **D6 — Formas `transferencia` e `debito_automatico`** acrescentadas.

## 10. Decisões pendentes

Sem abertos.
