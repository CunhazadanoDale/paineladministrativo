# Escopo — M14 Dívidas e empréstimos

> **Status:** escopo fechado — decisões D1 a D10 respondidas pelo usuário (outubro/2026), todas as
> recomendações aceitas; fases 1 e 2 concluídas (dívida, parcelas, geração das contas a pagar, cancelamento,
> renegociação e reabertura de parcela); fase 3 (visões) pendente.
> **Objetivo:** tirar da planilha "Dívidas do grupo" os empréstimos bancários, os aportes de investidores, os
> parcelamentos com o fisco e as dívidas parceladas com fornecedores: saldo devedor por credor e por empresa,
> cronograma de parcelas e as contas a pagar de cada parcela no M7, com juros na DRE (`resultado_financeiro`) e
> amortização fora dela (`fora_dre`).
> **Nome do módulo:** `divida`
> **Branch:** `feat/m14-dividas`

---

## 1. Visão geral

| Tema | Proposta |
|---|---|
| Módulo | Pacote novo `divida`: `domain/divida`, `ports/in|out/divida`, `usecases/divida`; rotas `/api/v1/dividas` |
| Unidade | Uma dívida = um contrato (empréstimo, aporte, parcelamento) de **uma empresa** com **um credor** |
| Empresa | A devedora, ativa e no acesso; tudo herda a `empresa_id` dela (seção 4 do `EMPRESAS.md`) |
| Credor (D1) | Tipo `banco`, `investidor`, `fisco` ou `fornecedor` + `fornecedor_id` do M4 (ativo ao criar), que é o favorecido das contas a pagar |
| Carga inicial | Saldo devedor na **data do saldo** (corte) e só as parcelas que vencem depois dela (decisão fechada no `ROADMAP.md`); valor original, taxa e contrato são informativos |
| Cronograma (D3) | Parcelas digitadas: vencimento, amortização e juros em centavos; Σ amortização = saldo na data do saldo |
| Conta a pagar (D2) | Cada parcela gera **duas** contas a pagar: `divida_amortizacao` (categoria `fora_dre`) e `divida_juros` (categoria `resultado_financeiro`), `origem_id` = id da parcela; componente zero não gera conta |
| Competência (D4) | A do mês do vencimento da parcela (padrão do `NovaSolicitacaoDeOrigem`) |
| Renegociação | Nova dívida ligada à original (`divida_origem_id`); a original vira `renegociada` e as contas a pagar ainda não pagas dela são canceladas na mesma transação |
| Crédito | Não conciliado: `conta_liberacao_id` e `liberada_em` só informativos; o M13 classifica a linha do extrato como captação (`fora_dre`) |
| Visões | Saldo devedor por tipo de credor e por empresa; vencido × a vencer |
| "Hoje" | Dia no fuso de Brasília, cópia própria em `domain/divida/data.go` (como `domainimportacao.Hoje`) |
| Dinheiro | Centavos `BIGINT`; taxa em pontos-base `INT` |
| Permissões (D8) | Só o financeiro lê e escreve |

---

## 2. Entidades

### 2.1 `divida`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; devedora, ativa e no acesso |
| `credor_tipo` | VARCHAR(12) | `banco`, `investidor`, `fisco`, `fornecedor` |
| `fornecedor_id` | UUID | FK `fornecedor` `RESTRICT`; credor, ativo ao criar (D1) |
| `descricao` | VARCHAR(120) | "Capital de giro Sicredi", "Parcelamento PERT" |
| `contrato` | VARCHAR(60) NULL | número do contrato / processo |
| `valor_original_centavos` | BIGINT NULL | informativo; `> 0` |
| `contratada_em` | DATE NULL | informativo; sem data futura |
| `taxa_bp` | INT NULL | informativo; pontos-base (`1,85%` → `185`), `0..100000` (D9) |
| `taxa_periodo` | VARCHAR(6) NULL | `mensal`, `anual`; junto com `taxa_bp` |
| `indexador` | VARCHAR(10) | `prefixado`, `cdi`, `selic`, `ipca`, `tr`, `outro`; padrão `prefixado` |
| `saldo_inicial_centavos` | BIGINT | saldo devedor na `data_saldo`; `> 0`; = Σ amortização das parcelas |
| `data_saldo` | DATE | data de corte (carga) ou da liberação (dívida nova); sem data futura |
| `conta_liberacao_id` | UUID NULL | FK `conta_bancaria`; da mesma empresa; informativo |
| `liberada_em` | DATE NULL | informativo; junto com a conta |
| `categoria_amortizacao_id` | UUID | FK `categoria_financeira`; lançável, `despesa`, linha `fora_dre` (D10) |
| `categoria_juros_id` | UUID | FK `categoria_financeira`; lançável, `despesa`, linha `resultado_financeiro` (D10) |
| `divida_origem_id` | UUID NULL | FK `divida`, única: a dívida renegociada que esta substitui |
| `status` | VARCHAR(12) | `ativa`, `renegociada`, `cancelada` (D6) |
| `motivo` | VARCHAR(1000) NULL | obrigatório ao renegociar e ao cancelar |
| `observacoes` | VARCHAR(2000) | |
| `usuario_id` | UUID | quem criou |
| `versao` | INT | versão otimista (contrato do M4) |
| `criado_em`, `atualizado_em` | TIMESTAMPTZ | |

### 2.2 `divida_parcela`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | `origem_id` das duas contas a pagar |
| `divida_id` | UUID | FK |
| `numero` | INT | `1..n` na dívida, pela ordem de vencimento; `uq_divida_parcela_numero` |
| `vencimento` | DATE | depois da `data_saldo`; único na dívida |
| `amortizacao_centavos` | BIGINT | `>= 0` |
| `juros_centavos` | BIGINT | `>= 0` (juros, encargos e IOF da parcela); amortização + juros `> 0` |
| `status` | VARCHAR(10) | `prevista` (sem conta), `gerada`, `cancelada` |
| `gerada_em` | TIMESTAMPTZ NULL | quando as contas a pagar nasceram |
| `substituta_id` | UUID NULL | FK `divida_parcela`: a parcela nova criada ao reabrir esta (D5) |
| `criado_em`, `atualizado_em` | TIMESTAMPTZ | |

- As contas a pagar não ficam em coluna: são lidas por `JOIN solicitacao ON origem IN ('divida_amortizacao',
  'divida_juros') AND origem_id = parcela.id` (o `uq_solicitacao_origem` garante uma de cada).
- **Situação** derivada na leitura: `prevista`/`a_vencer`/`vencida` (vencimento antes de hoje) enquanto houver
  componente não pago, `paga` quando as contas existentes estão todas `pago`, `cancelada`.
- **Saldo devedor** da dívida = Σ amortização das parcelas não canceladas cuja conta de amortização não está `pago`
  (parcela `prevista` conta inteira).

### 2.3 `divida_historico`

Append-only, como `importacao_historico`: `divida_id`, `acao` (`criada`, `alterada`, `parcela_alterada`,
`contas_geradas`, `parcela_reaberta`, `renegociada`, `cancelada`), `status_anterior`, `status_novo`,
`usuario_id`, `detalhe`, `criado_em`.

---

## 3. Fluxos

### 3.1 Estados da dívida

| Ação | De | Para | Efeito |
|---|---|---|---|
| criar | — | `ativa` | parcelas; gera as contas das parcelas com vencimento até o fim do mês corrente (D2) |
| alterar | `ativa` | igual | só campos informativos e categorias (com `versao`); categorias novas valem para as contas ainda não geradas |
| renegociar | `ativa` | `renegociada` | seção 3.4 |
| cancelar | `ativa` | `cancelada` | motivo; recusa (`409`) com alguma conta `pago`; cancela as contas abertas e as parcelas na mesma transação |

Toda transição usa o `UPDATE` condicionado ao status e à `versao` e grava o histórico na mesma transação.

### 3.2 Geração das contas a pagar (D2)

- `POST /dividas/gerar?competencia=AAAA-MM` (idempotente, como `POST /despesas-recorrentes/gerar`): para cada
  parcela `prevista` de dívida `ativa` que vence até o fim do mês, numa transação por dívida, gera as contas por
  `NovaSolicitacaoDeOrigem` + `gerarSolicitacaoDeOrigem` (fornecedor = credor, `pendente_aprovacao`, forma
  `transferencia`, observação "dívida <descricao> — parcela n/total — amortização|juros") e marca a parcela
  `gerada`. Resposta com geradas e recusadas (empresa ou credor inativo, categoria não lançável) por dívida.
- Criar e renegociar geram na hora as parcelas que vencem até o fim do mês corrente (Brasília).
- Parcela `prevista` aceita alterar vencimento (dentro do mesmo cronograma, sem repetir) e juros (`PUT
  .../parcelas/{pid}`): cobre parcelamento do fisco corrigido pela Selic. Amortização só muda por renegociação.

### 3.3 Reabrir parcela (D5)

`POST /dividas/{id}/parcelas/{pid}/reabrir { versao }`: com a parcela `gerada` e as contas dela todas `cancelado`
ou `rejeitado` no M7 (travadas com `FOR SHARE`), a parcela vira `cancelada` com `substituta_id` e nasce outra
`prevista` com o mesmo número, vencimento e valores, editável e gerável de novo.

### 3.4 Renegociação e parcelamento

`POST /dividas/{id}/renegociar { versao, motivo, nova: { credor_tipo, fornecedor_id, descricao, contrato, ...,
saldo_inicial_centavos, data_saldo, parcelas } }`, numa transação:

1. trava a original (`versao`, status `ativa`) e as contas a pagar dela com `FOR UPDATE`;
2. cancela as contas `pendente_aprovacao` e `aprovado` (status `cancelado`, `cancelado_em`, `versao + 1`,
   `solicitacao_historico` `cancelada` com "dívida renegociada"), sem passar pelo port do M7;
3. marca `cancelada` as parcelas sem nenhuma conta `pago` e mantém as pagas; parcela com uma conta paga e outra
   aberta fica com a paga e tem a aberta cancelada;
4. a original vira `renegociada` com o motivo; nasce a nova `ativa` com `divida_origem_id`, da **mesma empresa**
   (D7), `data_saldo` sem data futura e não anterior à `data_saldo` da original, e gera as contas do mês corrente.

O saldo da nova é informado (pode incluir multa, juros e encargos incorporados); o histórico das duas dívidas
registra o saldo devedor que a original tinha, para conferência. A resposta é a dívida nova (`201`).

### 3.5 Visões

`GET /dividas/resumo?empresa=`:

- saldo devedor por tipo de credor e por empresa (e o total), só dívidas `ativa`;
- vencido × a vencer: amortização e juros não pagos das parcelas não canceladas, separados por vencimento antes
  de hoje ou a partir de hoje (Brasília), com a quantidade de parcelas; a vencer também nos próximos 30, 90 e
  365 dias.

---

## 4. Endpoints

```
POST   /api/v1/dividas                                   (financeiro)
GET    /api/v1/dividas                                   ?empresa= ?credor_tipo= ?fornecedor= ?status= ?quitada= ?vencidas=
GET    /api/v1/dividas/{id}                              parcelas com as contas a pagar e a situação, saldo devedor
PUT    /api/v1/dividas/{id}                              { versao, campos informativos e categorias }
PUT    /api/v1/dividas/{id}/parcelas/{parcela_id}        { versao, vencimento, juros_centavos } parcela prevista
POST   /api/v1/dividas/{id}/parcelas/{parcela_id}/reabrir   { versao }
POST   /api/v1/dividas/{id}/renegociar                   { versao, motivo, nova }
POST   /api/v1/dividas/{id}/cancelar                     { versao, motivo }
GET    /api/v1/dividas/{id}/historico
POST   /api/v1/dividas/gerar                             ?competencia=AAAA-MM
GET    /api/v1/dividas/resumo                            ?empresa=
```

Todas em `financeiro(...)`. `versao` de toda escrita é a da dívida (as parcelas não têm versão própria).

---

## 5. Permissões (D8)

- Leitura e escrita só do financeiro (ou administrador), wrapper `financeiro(...)` em `rotas.go`; regra fina em
  `usecases/divida/permissoes.go` (`exigeFinanceiro`, escopo por empresa).
- Escopo por empresa pela devedora; teste de isolamento no usecase e no e2e.
- Sem exclusão física: cadastro errado é `cancelar`.

---

## 6. Contrato com os outros módulos

### 6.1 Contas a pagar (M7)

- `chk_solicitacao_origem`, `NovaOrigem`, `Origem.Sistemica()` e as mensagens de `NovaOrigem` e
  `NovaSolicitacaoDeOrigem` ganham `divida_amortizacao` e `divida_juros` (migration do M14), com `origem_id` =
  id da parcela e idempotência pelo `uq_solicitacao_origem` já existente. **Nenhum port nem repositório do M7
  muda.**
- Criação por `NovaSolicitacaoDeOrigem` e `gerarSolicitacaoDeOrigem(ctx, tx, ...)` na transação do M14, como o M12.
- Leitura do status das contas e o cancelamento na renegociação por consulta e `UPDATE` próprios do repositório do
  M14 (mesma guarda de status e `versao` do `atualizarStatusComGuarda`, histórico `cancelada`).

### 6.2 Fornecedores, contas e plano de contas (M4, M5)

Portas mínimas com `Obter` (fornecedor `Ativo`; conta bancária ativa da mesma empresa; categoria `Lancavel()`,
`Natureza` `despesa` e `LinhaDRE` `fora_dre` ou `resultado_financeiro`).

### 6.3 Caixa (M13) e DRE (M15)

- O M13 concilia os pagamentos das contas da dívida como qualquer conta a pagar e as vê na projeção; o crédito do
  empréstimo é classificado lá como `lancamento_caixa` em receita `fora_dre`. **Nenhum port nem repositório do M13
  muda.**
- O M15 lê os juros pela `solicitacao` (`competencia` + `linha_dre`, ignorando `cancelado` e `rejeitado`), sem
  filtrar por `origem`; a amortização cai em `fora_dre`. O painel de endividamento do M15 (última fase dele) lê
  `divida` e `divida_parcela` por porta mínima própria; a definição de saldo devedor é a da seção 2.2.

---

## 7. Estrutura de código

```
migrations/000067_create_tabela_divida                  divida, parcela, historico, origens no chk_solicitacao_origem

internal/core/domain/divida/          credor.go, status.go, divida.go, parcela.go, cronograma.go, taxa.go,
                                      renegociacao.go, geracao.go, resumo.go, historico.go, data.go, texto.go,
                                      erros.go (+ testes)
internal/core/ports/out/divida/       divida_repository.go, consulta_repository.go, fornecedor_repository.go,
                                      conta_bancaria_repository.go, categoria_financeira_repository.go,
                                      empresa_repository.go, usuario_repository.go, cargo_repository.go
internal/core/ports/in/divida/        divida_usecase.go, consulta_usecase.go
internal/core/usecases/divida/        os mesmos usecases + permissoes.go (+ testes, fakes)
internal/core/domain/solicitacao/     origem.go, origem_sistemica.go (duas origens)
internal/adapter/postgres/            divida_repo.go, divida_consulta_repo.go
internal/adapter/http/dto/divida/ + handlers/divida/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

Numeração a partir da `000067`; renumerar antes do merge se outra sessão mergear antes (regra 4 da seção 3 do
`ROADMAP.md`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Dívida e parcelas (migration 000067): domínio, criar com carga pelo saldo, alterar, editar parcela prevista, cancelar, histórico; geração das duas contas a pagar (na criação e `POST /dividas/gerar`); integração de idempotência; e2e + `API.md` | concluída |
| **2** | Renegociação com cancelamento das contas abertas e reabertura de parcela; integração da transação; e2e + `API.md` | concluída |
| **3** | Visões: lista com saldo e situação, `GET /dividas/resumo` (saldo por tipo de credor e empresa, vencido × a vencer); e2e + `API.md` | pendente |
| **4** | Frontend (lista, cadastro com cronograma, detalhe, renegociar, resumo) | pendente (sessão própria) |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Carga inicial sem retroativo** (`ROADMAP.md`, outubro/2026): saldo devedor na data de corte e só as
      parcelas que vencem depois; nada antes do corte gera conta a pagar.
- [x] **Crédito não conciliado contra a dívida** (`ROADMAP.md`): conta e data da liberação só informativas.
- [x] **Duas contas a pagar por parcela** (acordo M14 × M15): `divida_amortizacao` em `fora_dre` e `divida_juros`
      em `resultado_financeiro`, por `gerarSolicitacaoDeOrigem` na transação do M14.
- [x] **Renegociação cancela o que ficou aberto** (acordo M14 × M15), na mesma transação da nova dívida.
- [x] **"Hoje" no fuso de Brasília** (acordo M14 × M15).
- [x] **Conta a pagar gerada** (M9, M10, M12): `NovaSolicitacaoDeOrigem` + `gerarSolicitacaoDeOrigem`; competência
      de origem sistêmica é fixa no M7.
- [x] **Fase 1 — geração até o fim da competência** (implementação): `POST /dividas/gerar?competencia=` gera as
      parcelas `prevista` que vencem **até** o fim do mês, não só dentro dele, para um mês esquecido não ficar para
      trás; a resposta traz `geradas` (contas e parcelas por dívida) e `recusadas` (motivo do `400`/`409` da
      gravação: empresa ou credor inativo, categoria não lançável, dívida alterada ao mesmo tempo). Não há "já
      existentes": a parcela gerada sai da seleção.
- [x] **Fase 1 — conferência das contas na transação** (implementação): cancelar trava com `FOR UPDATE` todas as
      contas a pagar da dívida e recusa (`409`) se o status de alguma mudou desde a leitura; cada conta aberta é
      cancelada por `UPDATE` condicionado ao status lido (`versao + 1`, `cancelado_em`) com
      `solicitacao_historico` `cancelada` na mesma transação, sem passar pelo port do M7. O texto do histórico da
      conta é cortado em 500 caracteres (tamanho da coluna).
- [x] **Fase 1 — índices parciais** (implementação): número e vencimento da parcela são únicos só entre as não
      canceladas (`uq_divida_parcela_numero`, `uq_divida_parcela_vencimento`), para a parcela reaberta (fase 2)
      nascer com o mesmo número e vencimento da cancelada.
- [x] **Fase 1 — alteração** (implementação): o `PUT` substitui todos os campos informativos (omitido fica vazio);
      empresa, credor, saldo, data do saldo e cronograma não mudam. Categoria e conta de liberação só são
      conferidas quando mudam; a conta de liberação precisa ser da empresa, mas não precisa estar ativa (é
      informativa).
- [x] **Fase 1 — leitura** (implementação): `Obter` lê dívida, parcelas e o status das duas contas de cada parcela
      (`LEFT JOIN solicitacao` por `origem` + `origem_id`) numa transação somente leitura `REPEATABLE READ`; a
      lista não traz parcelas nem saldo (a fase 3 acrescenta o saldo por consulta agregada).
- [x] **Fase 2 — renegociação numa gravação** (implementação): a `Gravacao` da original leva a dívida nova em
      `Substituta`; o repositório atualiza a original com a guarda de versão, insere a nova (empresa ativa e credor
      ativo travados com `FOR SHARE`), confere e cancela as contas abertas da original, marca as parcelas e gera as
      contas do mês da nova, tudo na mesma transação. A `data_saldo` da nova não pode ser anterior à da original.
      Empresa inativa recusa a renegociação (`400`), como recusa dívida nova.
- [x] **Fase 2 — parcela com conta paga na renegociação** (implementação): fica `gerada` (é histórico pago) e só a
      conta aberta dela é cancelada; a original renegociada tem saldo devedor zero, então a conta cancelada não
      volta a contar.
- [x] **Fase 2 — reabrir** (implementação): exige a parcela `gerada` com **todas** as contas `cancelado` ou
      `rejeitado` (conferidas com `FOR UPDATE` na transação); a substituta é ligada por `substituta_id`, com a FK
      `DEFERRABLE INITIALLY DEFERRED`, porque a antiga precisa sair do índice parcial de número antes de a nova
      entrar.

## 10. Decisões do escopo

Respondidas pelo usuário (outubro/2026): todas as recomendações aceitas.

- [x] **D1 — Credor.** Decidido: `credor_tipo` + `fornecedor_id` obrigatório (banco, investidor e Receita
      cadastrados no M4, como o M12 já faz com a Receita), para a conta a pagar sair com favorecido.
      Alternativa descartada: nome livre do credor para banco, investidor e fisco, e fornecedor só no tipo `fornecedor`.
- [x] **D2 — Quando a parcela vira conta a pagar.** Decidido: por competência, em rotina idempotente
      (`POST /dividas/gerar?competencia=`, junto da das despesas recorrentes), mais as do mês corrente na criação;
      a fila de aprovação não recebe 2 × 60 contas de uma vez e os juros do mês podem ser ajustados antes.
      Alternativa descartada: todas as parcelas na criação (projeção e DRE futura completas, fila cheia, ajuste só no M7).
- [x] **D3 — Cronograma.** Decidido: parcelas digitadas (vencimento, amortização, juros) conferidas contra o
      saldo, porque contratos reais têm carência, IOF e datas irregulares. Alternativa descartada: gerar Price/SAC pela taxa
      e prazo (com ajuste manual depois).
- [x] **D4 — Competência dos juros.** Decidido: mês do vencimento da parcela (padrão do M7). Alternativa:
      competência do mês anterior ao vencimento (juros do período decorrido).
- [x] **D5 — Conta da parcela cancelada ou rejeitada no M7.** Decidido: "reabrir" a parcela como no M12 (vira
      `cancelada`, nasce outra `prevista` igual). Alternativa descartada: só resolver por renegociação.
- [x] **D6 — Estados.** Decidido: `ativa`, `renegociada`, `cancelada` gravados; "quitada" derivada (todas as
      parcelas pagas), porque o pagamento acontece no M7 sem avisar o M14; cancelar só sem conta paga.
      Alternativa descartada: status `quitada` gravado por ação manual "quitar".
- [x] **D7 — Renegociação.** Decidido: mesma empresa, credor pode mudar (dívida com fornecedor paga com
      empréstimo do banco), saldo novo informado. Alternativa descartada: mesmo credor e saldo novo igual ao saldo devedor
      aberto.
- [x] **D8 — Permissões.** Decidido: só o financeiro lê e escreve. Alternativa descartada: leitura do resumo para
      qualquer autenticado.
- [x] **D9 — Taxa.** Decidido: `taxa_bp` + `taxa_periodo` (mensal/anual) + `indexador`, só informativos.
      Alternativa descartada: texto livre ("CDI + 0,8% a.m.").
- [x] **D10 — Categorias.** Decidido: as duas categorias escolhidas por dívida e validadas pela linha da DRE
      (`fora_dre` e `resultado_financeiro`). Alternativa descartada: um par fixo para o sistema todo.

## 11. Decisões pendentes

Sem abertos.
