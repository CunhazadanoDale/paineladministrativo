# Escopo — M13 Caixa e conciliação

> **Status:** escopo fechado — decisões D1 a D15 respondidas pelo usuário (outubro/2026): todas as
> recomendações aceitas, com OFX como único formato (D4); fase 1 concluída (saldo inicial e importação de extrato).
> **Objetivo:** tirar o "Caixa do mês" e o "Fluxo de caixa diário" das planilhas: extrato bancário
> importado por conta (OFX), conciliado com os pagamentos do M7 e os recebimentos e borderôs do
> M9, linhas sem par classificadas no plano de contas, fechamento mensal por conta com divergência
> apontada e projeção diária de saldo por empresa e consolidada. Base do relatório semanal e dos
> saldos bancários do M15.
> **Nome do módulo:** `caixa`
> **Branch:** `feat/m13-caixa`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Módulo | Pacote novo `caixa`: `domain/caixa`, `ports/in|out/caixa`, `usecases/caixa`; rotas `/api/v1/extratos`, `/api/v1/conciliacoes` e `/api/v1/caixa` |
| Fonte do saldo (D1) | O **extrato** é a verdade do saldo: saldo da conta = abertura + Σ linhas do extrato. Os lançamentos do M7/M9 são o que se concilia contra ele |
| Abertura (D2) | Saldo inicial por conta numa **data de corte** digitada pelo financeiro; linha de extrato anterior ao corte é ignorada na importação |
| Histórico (D3) | **Sem retroativo**: o extrato começa no último mês encerrado (corte no último dia do mês anterior a ele); pagamentos e recebimentos anteriores ao corte ficam fora da conciliação |
| Importação (D4) | **Só OFX** (1.x SGML e 2.x XML; todos os bancos do grupo exportam), enviado por `POST /arquivos` e importado por `POST /extratos/importacao`, como o XML do M8. Conta identificada pelo `BANKID`/`BRANCHID`/`ACCTID`. Idempotente por identificador da transação |
| Conciliação (D5) | Uma linha do extrato ↔ **um ou mais** movimentos do sistema com soma exata. Sugestão por conta, valor e data (±3 dias); automática só no par único exato. Desfazer é append-only |
| Movimentos conciliáveis | Débito: `pagamento` do M7 (inclui recompra, repasse e comissão, que são contas a pagar). Crédito: `titulo_recebimento` tipo `conta` (valor + juros) e crédito do borderô (`antecipacao`, valor líquido). Mais os lançamentos do próprio M13 |
| Linha sem par (D6) | Classificada a partir da própria linha: vira `lancamento_caixa` com categoria lançável (natureza pelo sinal) e já nasce conciliada. Tarifa, IOF, rendimento, juros bancários |
| Transferência entre contas (D8, D9) | Registro próprio criado a partir da linha de uma das pontas; a outra ponta é conciliada contra ele quando o extrato dela chegar. Categoria `entre_contas`, fora da DRE. Entre empresas do grupo: **regra provisória** do M8 — permitida, marcada `entre_empresas`, sem título nem conta a pagar entre elas |
| Fechamento (D10) | Mensal, por conta, sequencial; exige o mês todo conciliado ou classificado e o saldo calculado igual ao do banco; reabertura só pelo administrador, com motivo |
| M7/M9 não mudam (D11, D12) | O mês fechado **não bloqueia** o M7 nem o M9: lançamento posterior com data em mês fechado vira divergência do fechamento. Estorno de item conciliado continua livre: a conciliação fica `invalidada` e trava o fechamento até ser desfeita |
| Projeção (D13) | Diária, até 90 dias, por empresa e consolidada: saldo atual + títulos a receber − contas a pagar − recorrentes ainda não geradas, vencidos acumulados em "hoje" |
| Dependências do M7/M9 | **Nenhum método novo nos ports nem nos repositórios do M7 e do M9**: o M13 lê `pagamento`, `titulo_recebimento` e `antecipacao` por consulta própria no `adapter/postgres/caixa_*`, como o M10 lê `entrega` |
| Escopo por empresa | Padrão da seção 4 do `EMPRESAS.md`: tudo herda a `empresa_id` da conta bancária |
| Permissões (D15) | Financeiro e administrador; reabrir fechamento e desfazer importação só o administrador |
| "Hoje" | Dia no fuso de Brasília, cópia própria em `domain/caixa/data.go` (como `domainreceber.Hoje`) |
| Dinheiro | Centavos `BIGINT`; valor da linha do extrato **com sinal** (crédito positivo, débito negativo) |

---

## 2. Entidades

### 2.1 `caixa_abertura` — saldo inicial da conta

| Campo | Tipo | Observações |
|---|---|---|
| `conta_bancaria_id` | UUID | PK; FK `conta_bancaria` `RESTRICT` |
| `data_corte` | DATE | saldo no **fim** deste dia; o extrato vale a partir do dia seguinte |
| `saldo_centavos` | BIGINT | aceita negativo (cheque especial) |
| `usuario_id` | UUID | FK `usuario` |
| `versao` | INT | versão otimista (contrato do M4) |
| `criado_em`, `atualizado_em` | TIMESTAMPTZ | |

- Editável enquanto a conta não tem fechamento; depois disso, `409`.
- Conta sem abertura não importa extrato (`400 conta sem saldo inicial; informe a abertura`).
- Conta inativa não recebe abertura nem importação; a empresa precisa estar ativa.

### 2.2 `extrato_importacao`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `conta_bancaria_id` | UUID | FK `conta_bancaria` |
| `arquivo_id` | UUID | FK `arquivo`; `uq_extrato_importacao_arquivo` |
| `periodo_inicio`, `periodo_fim` | DATE | `DTSTART`/`DTEND` do OFX |
| `saldo_banco_centavos` | BIGINT NULL | `LEDGERBAL` do OFX; usado no fechamento |
| `saldo_banco_em` | DATE NULL | data do `LEDGERBAL` |
| `linhas_novas`, `linhas_repetidas`, `linhas_ignoradas` | INT | relatório gravado |
| `usuario_id`, `criado_em` | | |

### 2.3 `extrato_lancamento`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `conta_bancaria_id` | UUID | FK `conta_bancaria` |
| `importacao_id` | UUID | FK `extrato_importacao` (a primeira que trouxe a linha) |
| `data` | DATE | `DTPOSTED` (dia, sem fuso) |
| `valor_centavos` | BIGINT | `<> 0`; crédito positivo, débito negativo |
| `descricao` | VARCHAR(255) | `MEMO`/`NAME` normalizado |
| `documento` | VARCHAR(60) NULL | `CHECKNUM`/`REFNUM` |
| `identificador` | VARCHAR(128) | chave de idempotência (seção 3.1); `uq_extrato_lancamento_identificador (conta_bancaria_id, identificador)` |
| `criado_em` | TIMESTAMPTZ | |

- Imutável. Índice `idx_extrato_lancamento_conta (conta_bancaria_id, data, id)`.
- "Pendente" ou "conciliado" é derivado da `conciliacao` vigente, não gravado na linha.

### 2.4 `conciliacao` e `conciliacao_item`

| Campo (`conciliacao`) | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `extrato_lancamento_id` | UUID | FK; `uq_conciliacao_lancamento_vigente (extrato_lancamento_id) WHERE vigente` |
| `modo` | VARCHAR(10) | `manual`, `automatica`, `classificacao`, `transferencia` |
| `vigente` | BOOLEAN | `false` depois de desfeita |
| `desfeita_em`, `desfeita_por`, `motivo_desfazer` | | preenchidos juntos (`chk_conciliacao_desfeita`) |
| `usuario_id`, `criado_em` | | |

| Campo (`conciliacao_item`) | Tipo | Observações |
|---|---|---|
| `conciliacao_id` | UUID | FK |
| `alvo_tipo` | VARCHAR(20) | `pagamento`, `recebimento`, `antecipacao`, `lancamento_caixa`, `transferencia` |
| `alvo_id` | UUID | sem FK (aponta para tabelas diferentes), conferido com trava na transação |
| `valor_centavos` | BIGINT | com sinal, igual ao efeito do alvo no caixa |
| `vigente` | BOOLEAN | cópia do da conciliação, para o índice; `uq_conciliacao_item_alvo_vigente (alvo_tipo, alvo_id) WHERE vigente` |

- Σ `conciliacao_item.valor_centavos` = `extrato_lancamento.valor_centavos` (conferido no usecase).
- Desfazer = `UPDATE ... SET vigente = false WHERE vigente` na conciliação e nos itens, na mesma
  transação, no padrão do `antecipacao_titulo`. Nada é apagado.
- O alvo precisa ser da **mesma conta**, não estornado e com data em até ±N dias da linha (D5).

### 2.5 `lancamento_caixa` — linha sem par no sistema

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id`, `conta_bancaria_id` | UUID | FKs; empresa da conta |
| `data` | DATE | a da linha do extrato |
| `valor_centavos` | BIGINT | com sinal, igual ao da linha |
| `categoria_financeira_id` | UUID | lançável; `receita` em crédito, `despesa` em débito; `transferencia` só pela transferência |
| `competencia` | DATE | dia 1 do mês da `data` (D7) |
| `descricao` | VARCHAR(255) | sugerida pela linha, editável na criação |
| `usuario_id`, `criado_em` | | |

- Nasce só a partir de uma linha do extrato (D6), conciliado a ela na mesma transação.
- Desfazer a conciliação de classificação desfaz o lançamento (fica registrado como `estornado`, com
  `lancamento_caixa_estorno` no padrão do `pagamento_estorno`).

### 2.6 `transferencia_conta`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `conta_origem_id`, `conta_destino_id` | UUID | FKs; diferentes |
| `data` | DATE | |
| `valor_centavos` | BIGINT | `> 0` |
| `entre_empresas` | BOOLEAN | contas de empresas diferentes (D8) |
| `categoria_financeira_id` | UUID | natureza `transferencia` (linha `entre_contas`) |
| `usuario_id`, `criado_em` | | |

- Criada a partir da linha de uma das pontas (débito na origem ou crédito no destino), que já é
  conciliada a ela. A outra ponta aparece como pendência do sistema até a linha correspondente chegar
  e ser conciliada (`alvo_tipo = transferencia`, com o sinal da ponta).

### 2.7 `caixa_fechamento`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `conta_bancaria_id` | UUID | FK |
| `competencia` | DATE | dia 1; `uq_caixa_fechamento_vigente (conta_bancaria_id, competencia) WHERE vigente` |
| `saldo_inicial_centavos` | BIGINT | abertura (primeiro mês) ou saldo final do fechamento anterior |
| `entradas_centavos`, `saidas_centavos` | BIGINT | Σ créditos e Σ débitos do extrato no mês |
| `saldo_final_centavos` | BIGINT | `inicial + entradas − saídas` |
| `saldo_banco_centavos` | BIGINT | `LEDGERBAL` do fim do mês ou digitado |
| `vigente` | BOOLEAN | `false` depois de reaberto |
| `reaberto_em`, `reaberto_por`, `motivo_reabertura` | | |
| `usuario_id`, `criado_em` | | |

---

## 3. Fluxos

### 3.1 Importação do extrato

1. O financeiro envia o arquivo por `POST /arquivos` (o upload passa a aceitar `.ofx`) e chama
   `POST /extratos/importacao { arquivo_id, conta_bancaria_id?, simular? }`.
2. O usecase lê o arquivo como o `leitorDeXML` do M8: dono = quem importa, tipo aceito, não vinculado
   a solicitação, não importado antes (`409`); limite de 5 MB e 5000 linhas.
3. OFX: `BANKID` + `BRANCHID` + `ACCTID` resolvem a conta (banco COMPE, agência e conta sem zeros à
   esquerda nem dígito); `conta_bancaria_id` informada precisa bater (`400`). Charset `1252`/`ISO-8859-1`
   convertido para UTF-8.
4. Identificador da linha: `FITID` + data + valor (alguns bancos repetem o `FITID` entre exportações ou
   entre linhas do mesmo dia).
5. Linhas antes da data de corte ou em mês fechado são ignoradas e entram no relatório; repetidas são
   contadas. Tudo numa transação, com `INSERT ... ON CONFLICT DO NOTHING`. `simular` devolve o relatório
   sem gravar.

### 3.2 Conciliação

1. `GET /extratos/lancamentos/{id}/sugestoes` lista alvos pendentes da mesma conta com o mesmo valor
   absoluto e data a até ±N dias (`pago_em` convertido para o dia de Brasília), por distância de data.
2. `POST /extratos/lancamentos/{id}/conciliacao { itens }` trava os alvos (`FOR SHARE`), confere conta,
   não estornado, soma e janela de data, e grava com os índices parciais garantindo exclusividade
   (`409` se outro usuário conciliou antes).
3. `POST /extratos/conciliar-automatico { conta_bancaria_id, de, ate }` concilia só os pares únicos nos
   dois sentidos (uma linha, um alvo, mesmo valor, mesma data) e responde `{ conciliadas, ambiguas,
   sem_par }`.
4. Alvo estornado depois de conciliado (D12): a conciliação continua gravada, mas aparece como
   `invalidada` nas consultas e bloqueia o fechamento até ser desfeita.

### 3.3 Fechamento

1. `POST /caixa/fechamentos { conta_bancaria_id, competencia, saldo_banco_centavos? }`. Só mês encerrado,
   sequencial (o anterior precisa estar fechado, salvo o mês da data de corte).
2. Trava a conta e as linhas do mês; recusa (`409`, com a lista) se houver linha pendente ou conciliação
   invalidada (D10).
3. Calcula os saldos; `saldo_banco_centavos` vem do corpo ou do `LEDGERBAL` de uma importação com
   `saldo_banco_em` no último dia do mês; diferente do calculado → `409` com a diferença.
4. Pendências do lado do sistema (pagamento ou recebimento do mês sem linha no extrato) não bloqueiam;
   vão no corpo da resposta e no detalhe do fechamento como divergência.

### 3.4 Projeção

`GET /caixa/projecao?empresa=&de=&ate=` (máximo 90 dias, `de` a partir de hoje), por dia:

- **saldo de partida**: saldo das contas ativas da empresa pelo extrato, mais os movimentos do sistema
  já registrados e ainda sem linha no extrato (pagos e recebidos "em trânsito");
- **entradas previstas**: saldo dos títulos `carteira` e `em_banco` por vencimento (`antecipado` fica
  fora: o dinheiro já entrou no borderô);
- **saídas previstas**: contas a pagar `pendente_aprovacao` e `aprovado` por `prazo_pagamento`, com
  as `estimado` em coluna própria;
- **recorrentes previstas**: despesas recorrentes vigentes cujas solicitações ainda não foram geradas
  para as competências do horizonte, pelo dia de vencimento, em coluna própria (D13);
- vencidos de ambos os lados acumulados no dia de hoje, em coluna própria.

Consolidado = soma das empresas no acesso. `GET /caixa/movimento-diario` dá o realizado (extrato por
dia, por empresa e conta), que substitui o "Fluxo de caixa diário".

---

## 4. Endpoints

```
PUT    /api/v1/caixa/contas/{conta_id}/abertura      { data_corte, saldo_centavos, versao? } (financeiro)
GET    /api/v1/caixa/contas/{conta_id}/abertura
GET    /api/v1/caixa/saldos                          ?empresa= — saldo por conta, data do último extrato, pendentes

POST   /api/v1/extratos/importacao                   { arquivo_id, conta_bancaria_id?, simular? } (financeiro)
GET    /api/v1/extratos/importacoes                  ?conta= ?empresa=
DELETE /api/v1/extratos/importacoes/{id}             sem linha conciliada e em mês aberto (administrador)
GET    /api/v1/extratos/lancamentos                  ?conta= (obrigatória) ?de= ?ate= ?situacao=pendente|conciliado|invalidado
GET    /api/v1/extratos/lancamentos/{id}/sugestoes
POST   /api/v1/extratos/lancamentos/{id}/conciliacao     { itens: [{ tipo, id }] }
POST   /api/v1/extratos/lancamentos/{id}/classificacao   { categoria_financeira_id, descricao? }
POST   /api/v1/extratos/lancamentos/{id}/transferencia   { conta_contraparte_id }
POST   /api/v1/extratos/conciliar-automatico         { conta_bancaria_id, de, ate }
POST   /api/v1/conciliacoes/{id}/desfazer            { motivo }

GET    /api/v1/caixa/pendencias-sistema              ?conta= ?de= ?ate= — pagamentos, recebimentos, borderôs e pontas de transferência sem linha
POST   /api/v1/caixa/fechamentos                     { conta_bancaria_id, competencia, saldo_banco_centavos? }
GET    /api/v1/caixa/fechamentos                     ?conta= ?empresa= ?competencia=
GET    /api/v1/caixa/fechamentos/{id}                com divergências
POST   /api/v1/caixa/fechamentos/{id}/reabrir        { motivo } (administrador)

GET    /api/v1/caixa/projecao                        ?empresa= ?de= ?ate=
GET    /api/v1/caixa/movimento-diario                ?empresa= ?conta= ?de= ?ate=
```

---

## 5. Permissões

- Tudo em `financeiro(...)`; reabrir fechamento e desfazer importação em `administrador(...)`. Regra
  fina em `usecases/caixa/permissoes.go`, reaproveitando o padrão de `usecases/financeiro/permissoes.go`
  (`exigeFinanceiro`, `escopoPorEmpresa`, `empresasVisiveis`).
- Escopo por empresa pela conta: conta fora do acesso → `403`; listas filtradas por `EmpresaIDs`.
  Transferência entre empresas exige acesso às duas. Teste de isolamento no usecase e no e2e.

---

## 6. Contrato com os outros módulos

### 6.1 Contas a pagar (M7)

- O M13 lê `pagamento` (`conta_bancaria_id`, `pago_em` no dia de Brasília, `valor`, `estornado`) e
  `solicitacao` (status, `prazo_pagamento`, `valor_estimado`, `estimado`, empresa) por consulta própria.
  **Nenhuma mudança no domínio, port ou repositório do M7.**
- Pagamento sem `conta_bancaria_id` (anterior ao M7) fica fora da conciliação (D14).
- O item 11 do `PENDENCIAS.md` ("hoje" em UTC no M7) afeta os vencidos da projeção: o M13 usa o próprio
  `Hoje` e recomenda-se corrigir o M7 em sessão à parte.

### 6.2 Contas a receber (M9)

- Lê `titulo_recebimento` tipo `conta` (`recebido_em`, `valor + juros`, `estornado`), `antecipacao`
  `ativa` (`operado_em`, `valor_liquido_centavos`) e `titulo_receber` (`carteira`/`em_banco`, saldo,
  vencimento) por consulta própria. Recompra e repasse saem do caixa pelo `pagamento` do M7.
- Borderô estornado é "lançamento errado" no M9: se estava conciliado, a conciliação fica invalidada
  (D12).

### 6.3 Contas bancárias e plano de contas (M5)

Portas mínimas com `Obter` (conta: `EmpresaID`, `Ativa`, `Banco`, `Agencia`, `Conta`, `Tipo`; categoria:
`Lancavel()`, `Natureza`, `LinhaDRE`) e uma busca de conta por banco + agência + número para o OFX,
feita por consulta própria do M13. Travas `FOR SHARE` no repositório do M13.

### 6.4 Arquivos

`normalizarContentType` (`usecases/solicitacao/arquivo_usecase.go`) passa a aceitar OFX
(`application/x-ofx`, e `application/octet-stream` com extensão `.ofx`). Leitura
pelo padrão do M8 (portas `ArquivoRepository` e `Storage` próprias em `ports/out/caixa/`).

### 6.5 DRE (M15)

- `lancamento_caixa` entra na DRE por `competencia` + `linha_dre` da categoria (tarifas e rendimentos
  em `resultado_financeiro`, por exemplo).
- `transferencia_conta` e categorias `entre_contas` ficam fora.
- Saldos bancários do relatório semanal vêm de `GET /caixa/saldos`.

---

## 7. Estrutura de código

```
migrations/000059_create_tabela_caixa_abertura
migrations/000060_create_tabela_extrato                    importacao, lancamento
migrations/000061_create_tabela_conciliacao                conciliacao, item
migrations/000062_create_tabela_lancamento_caixa           lancamento, estorno, transferencia_conta
migrations/000063_create_tabela_caixa_fechamento

internal/core/domain/caixa/          abertura.go, extrato.go, linha_extrato.go, ofx.go,
                                     identificador.go, conciliacao.go, alvo.go, sugestao.go,
                                     lancamento.go, transferencia.go, fechamento.go, projecao.go,
                                     data.go, erros.go (+ testes)
internal/core/ports/out/caixa/       abertura_repository.go, extrato_repository.go,
                                     conciliacao_repository.go, lancamento_repository.go,
                                     fechamento_repository.go, movimento_sistema_repository.go,
                                     projecao_repository.go, conta_bancaria_repository.go,
                                     categoria_financeira_repository.go, empresa_repository.go,
                                     arquivo_repository.go, storage.go, usuario_repository.go,
                                     cargo_repository.go
internal/core/ports/in/caixa/        abertura_usecase.go, extrato_usecase.go, conciliacao_usecase.go,
                                     lancamento_usecase.go, fechamento_usecase.go, projecao_usecase.go
internal/core/usecases/caixa/        idem + leitor_extrato.go, permissoes.go (+ testes e fakes)
internal/core/usecases/solicitacao/  arquivo_usecase.go (tipo OFX)
internal/adapter/postgres/           caixa_abertura_repo.go, extrato_repo.go, conciliacao_repo.go,
                                     lancamento_caixa_repo.go, caixa_fechamento_repo.go,
                                     caixa_movimento_repo.go, caixa_projecao_repo.go
internal/adapter/http/dto/caixa/ + handlers/caixa/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

- O parser de OFX fica no domínio, como `domain/notafiscal/xml.go`; sem biblioteca nova.
- Numeração a partir da `000059`; renumerar antes do merge se outra sessão mergear antes (regra 4 da
  seção 3 do `ROADMAP.md`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Abertura e importação de extrato OFX (migrations 000059–000060): domínio com parser, tipos no upload, repositório, usecase com escopo por empresa, rotas de abertura, importação, lista e saldos; integração de idempotência; e2e + `API.md` | concluída |
| **2** | Conciliação com M7 e M9 (migration 000061): consultas de movimentos do sistema, sugestões, conciliação manual e automática, desfazer, invalidação por estorno, pendências do sistema; integração de corrida; e2e + `API.md` | pendente |
| **3** | Classificação e transferência entre contas (migration 000062): `lancamento_caixa`, estorno pelo desfazer, `transferencia_conta` com as duas pontas; e2e + `API.md` | pendente |
| **4** | Fechamento mensal e reabertura (migration 000063), bloqueio de importação em mês fechado; e2e + `API.md` | pendente |
| **5** | Projeção diária e movimento diário realizado; e2e + `API.md` | pendente |
| **6** | Frontend (abertura, importação, tela de conciliação, fechamento, projeção) | pendente |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **Saldo inicial fica no M13** (`FINANCEIRO.md`, seção 9).
- [x] **Fluxo de caixa pelo `pago_em`** do M7 e pelo `recebido_em`/`operado_em` do M9; DRE por
      competência (decisão D5 do M7).
- [x] **Entrada no caixa do recebimento = valor + juros**; tipo `fomento` não entra no caixa
      (`CONTAS_A_RECEBER.md`, seção 2.3).
- [x] **M13 aceita as três naturezas** do plano de contas, `transferencia` só em `entre_contas`
      (`FINANCEIRO.md`, seção 6).
- [x] **D3 — Importação histórica.** Decidido: sem retroativo; o extrato começa no último mês
      encerrado (corte no último dia do mês anterior a ele), e pagamentos e recebimentos antigos ficam
      fora da conciliação. Alternativa descartada: carregar meses de extrato para trás e conciliar o
      histórico.
- [x] **D4 — Formatos.** Decidido: **só OFX**; todos os bancos do grupo (Sicredi, Itaú, BB, Santander,
      Bradesco, Cora, Inter) exportam. Sem leitor de CSV. Alternativas descartadas: CSV de layout fixo do
      sistema; leitor de CSV por banco.
- [x] **D8 — Transferência entre contas de empresas diferentes.** Decidido, como regra provisória igual
      à do M8: permitida, marcada `entre_empresas`, categoria `entre_contas`, fora da DRE das duas
      empresas, sem gerar título nem conta a pagar entre elas; o efeito definitivo vira fase própria
      quando a contabilidade responder. Alternativa descartada: bloquear até a resposta.
- [x] **D11 — Efeito do mês fechado nos outros módulos.** Decidido: o M13 não bloqueia o M7 nem o M9;
      pagamento ou recebimento lançado depois numa data de mês fechado aparece como divergência no
      fechamento. Alternativa descartada: M7 e M9 recusarem data em mês fechado.
- [x] **D12 — Estorno de item já conciliado.** Decidido: o M7/M9 continuam estornando livremente; a
      conciliação fica `invalidada`, a linha volta a pendente e o fechamento trava até alguém desfazer.
      Alternativa descartada: M7 e M9 recusarem estornar item conciliado.

- [x] **D1 — Fonte de verdade do saldo.** Decidido: o **extrato** (abertura + Σ linhas importadas); o
      que está no M7/M9 é conciliado contra ele, e o que só existe no sistema aparece como pendência.
      Alternativa descartada: saldo pelo sistema com o extrato só para conferência.
- [x] **D2 — Abertura.** Decidido: uma data de corte por conta, digitada com o saldo do banco no fim
      daquele dia; nada anterior ao corte entra. Alternativa descartada: abertura única do grupo numa
      data comum a todas as contas.
- [x] **D5 — Regra de conciliação.** Decidido: N:1 (uma linha cobre vários pagamentos ou recebimentos,
      como pagamento em lote ou crédito de vários boletos), janela de ±3 dias, sugestão com confirmação
      e automática só no par único exato do mesmo dia. Alternativas descartadas: só 1:1; só manual.
- [x] **D6 — Linha sem par.** Decidido: vira `lancamento_caixa` só a partir da linha do extrato (tarifa,
      IOF, rendimento, juros), nunca digitado solto. Alternativa descartada: lançamento manual sem
      extrato.
- [x] **D7 — Competência do lançamento avulso.** Decidido: mês da data do extrato, fixo. Alternativa
      descartada: competência editável na classificação.
- [x] **D9 — Aplicação e resgate automáticos.** Decidido: a aplicação é cadastrada como conta do tipo
      `aplicacao` no M5, e aplicação/resgate são transferência entre contas; rendimento é
      `lancamento_caixa` de receita. Alternativa descartada: avulsos `entre_contas` sem conta de
      aplicação.
- [x] **D10 — O que impede o fechamento.** Decidido: linha do mês pendente ou conciliação invalidada, e
      saldo calculado diferente do saldo do banco; pendências do lado do sistema só são apontadas.
      Alternativa descartada: fechar com pendências.
- [x] **D13 — O que entra na projeção.** Decidido: títulos `carteira`/`em_banco`, contas a pagar
      `pendente_aprovacao` e `aprovado` (estimadas à parte), despesas recorrentes ainda não geradas no
      horizonte (pelo `ListarVigentes` do M7, em coluna própria), vencidos em "hoje", horizonte de até
      90 dias.
- [x] **D14 — Pagamentos sem conta bancária** (anteriores ao M7, com `conta_bancaria_id` nulo).
      Decidido: ficam fora da conciliação e da projeção, listados à parte. Alternativa descartada: rota
      no M7 para completar a conta do pagamento antigo.
- [x] **D15 — Permissões.** Decidido: o financeiro faz tudo; reabrir fechamento e desfazer importação só
      o administrador. Alternativa descartada: reabrir também pelo financeiro.
- [x] **Fase 1 — saldo inicial** (implementação): rota `PUT /caixa/contas/{id}/abertura` no próprio
      módulo, sem tocar no M5; o mesmo `PUT` cria (sem `versao`) e altera (com `versao`); a data de corte
      precisa ser anterior a hoje em Brasília, porque o saldo é o do fim do dia.
- [x] **Fase 1 — conta do extrato** (implementação): sem `conta_bancaria_id`, a conta é achada entre as
      contas do banco do OFX comparando agência e conta sem zeros à esquerda, com ou sem dígito, e com a
      agência colada no número; nenhuma ou mais de uma conta → `400` pedindo a conta. Com a conta
      informada, OFX de outra conta → `400`; OFX sem `ACCTID` é aceito.
- [x] **Fase 1 — identificador da linha** (implementação): hash de `FITID` + data + valor + ocorrência no
      arquivo (sem `FITID`, a descrição entra na chave), porque há bancos que repetem o `FITID`; linhas de
      valor zero, até o corte ou com data futura são ignoradas e listadas no relatório, não gravadas.
- [x] **Fase 1 — desfazer importação** (implementação): só a mais recente da conta, para não apagar linhas
      que uma importação posterior contou como repetidas; o arquivo pode ser importado de novo depois.
- [x] **Fase 1 — OFX não é anexo** (implementação): como o XML da NF-e, arquivo OFX não pode ser anexado a
      solicitação; o upload aceita `.ofx` mesmo quando o navegador manda `application/octet-stream`.

## 10. Decisões pendentes

Sem abertos.
