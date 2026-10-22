# Escopo — M15 DRE e painéis

> **Status:** concluído (outubro/2026) — API completa (fases 1 a 5); front pendente (fase 6).
> **Objetivo:** substituir as planilhas de DRE, Faturamento e Relatório semanal por consultas
> somente leitura sobre o que os módulos M2, M7, M8, M9, M12 e M13 já gravam.
> **Nome do módulo:** `dre` (pastas `domain/dre`, `ports/in|out/dre`, `usecases/dre`,
> `handlers/dre`, `dto/dre`; rotas em `/api/v1/dre/...`).
> Coordenação com o M14 em "Coordenação M14 × M15" (`ROADMAP.md`, seção do M14).

---

## 1. Visão geral

O M15 não tem entidade nem dado digitado. Cada painel é um `resumo` (padrão do
`resumo_usecase` do estoque e do `contas_a_pagar_resumo_repo`): o usecase confere permissão e
escopo por empresa, normaliza o período e chama consultas agregadas em SQL no `adapter/postgres`;
o domínio monta as linhas, os subtotais e os indicadores a partir dos agregados.

| Tema | Decisão |
|---|---|
| Regime | **Competência** (D5 do M7): despesa pela `competencia`, receita pela emissão da NF |
| Mês | Mês civil no fuso de Brasília; datas de fato (`DATE`) usadas como estão, instantes (`TIMESTAMPTZ`) convertidos para Brasília antes de truncar |
| "Hoje" | Dia no fuso de Brasília, como `domainpedido.Hoje` / `domaincaixa.Hoje` |
| Escopo | Por empresa (seção 4 do `EMPRESAS.md`); consolidado = soma das empresas no acesso do usuário |
| Gravação | Nenhuma; migrations só de índices |
| Fonte única | Cada número tem uma só fonte (seção 3), para nada ser contado duas vezes |

---

## 2. Linhas da DRE

Ordem de apresentação e sinal. Receita soma, despesa subtrai; os subtotais são calculados no
domínio.

| # | Linha (`chave`) | Fonte | Sinal |
|---|---|---|---|
| 1 | `receita_bruta` | NF de venda (3.1) | + |
| 2 | `outras_receitas` | `receita_operacional` (3.3, 3.4, 3.5) | + |
| 3 | `deducoes` | `deducao_venda` (3.3, 3.5) + devoluções (D2) | − |
| = | **`receita_liquida`** | 1 + 2 − 3 | |
| 4 | `cmv` | lotes consumidos (3.2) | − |
| 5 | `perdas_estoque` | lotes consumidos por perda/ajuste (D1) | − |
| = | **`lucro_bruto`** | | |
| 6 | `despesas_variaveis` | `despesa_variavel` (3.3, 3.5) | − |
| = | **`margem_contribuicao`** | | |
| 7 | `despesas_fixas` | `despesa_fixa` (3.3, 3.5) | − |
| 8 | `nao_classificado` | solicitação sem categoria (D5) | − |
| = | **`resultado_operacional`** | | |
| 9 | `resultado_financeiro` | `resultado_financeiro` (3.3, 3.4, 3.5) + juros e encargos do M9 (3.4) | ± |
| = | **`resultado`** | | |
| — | `fora_dre` | `fora_dre` (3.3, 3.4, 3.5) — entradas e saídas à parte, fora do resultado | ± |

Linhas 2, 3, 6, 7, 9 e `fora_dre` trazem o detalhamento por **categoria raiz** (o plano tem dois
níveis e a subcategoria herda a linha), com `categoria_id`, `nome` e valor; as linhas de origem
sem categoria (juros recebidos, encargos do borderô) aparecem como itens nomeados fixos.

**Indicadores do mês:**

- `margem_contribuicao_bp` = margem de contribuição ÷ receita líquida, em pontos-base;
- `ponto_equilibrio_centavos` = despesas fixas ÷ índice de margem de contribuição (D6); nulo quando
  a margem é ≤ 0 ou a receita líquida é 0;
- `quantidade_sem_custo`: unidades consumidas de lote com custo desconhecido no mês (o CMV as conta
  como zero; o painel avisa).

---

## 3. Fontes (consultas)

Toda consulta filtra `empresa_id = ANY($n)` quando o usuário não tem acesso a todas as empresas.

### 3.1 Receita bruta — `nota_fiscal`

- `natureza = 'venda' AND status = 'autorizada' AND NOT entre_empresas`;
- mês = `date_trunc('month', emitida_em AT TIME ZONE 'America/Sao_Paulo')`;
- valor = `valor_total_centavos`; a quantidade de notas não conta NF `complementar` nem `ajuste` (o valor
  delas soma, mas não são uma venda nova e distorceriam o ticket médio);
- NF cancelada depois sai da DRE do mês dela (o painel é sempre recalculado).

### 3.2 CMV — `produto_movimento_lote`

- `produto_movimento_lote` ⨝ `produto_movimento` ⨝ `estoque_local` (empresa do local);
- mês = `date_trunc('month', produto_movimento.criado_em AT TIME ZONE 'America/Sao_Paulo')`;
- valor = `Σ quantidade × custo_unitario_centavos` (custo `NULL` conta zero e soma em
  `quantidade_sem_custo`);
- tipos: ver D1. `transferencia` nunca entra (o lote só muda de local);
- contas a pagar `compra_mercadoria` (inclusive as do M12) **nunca** entram.

### 3.3 Despesas — `solicitacao`

- `solicitacao` ⨝ `categoria_financeira` (`LEFT JOIN`, ver D5) por `competencia`;
- `status NOT IN ('cancelado', 'rejeitado')`, **sem filtro por `origem`** (as contas do M10, M12 e
  M14 entram pela categoria: juros do M14 em `resultado_financeiro`, amortização em `fora_dre`);
- valor: ver D3;
- `compra_mercadoria` e `entre_contas` ficam fora;
- a conta de uma `importacao_parcela` e a de um `importacao_custo` com `compoe_custo` ficam fora **qualquer
  que seja a categoria** (o M12 só sugere `compra_mercadoria`): esse valor já está no custo do lote e entra como
  CMV na saída (3.2). Custo com `compoe_custo = false` entra pela própria categoria.

### 3.4 Contas a receber — `titulo_receber`, `titulo_recebimento`, `antecipacao`

- título `origem = 'manual'`, `status <> 'cancelado'`, por `competencia` + `linha_dre` da categoria,
  valor = `valor_centavos` (títulos de NF e de pedido ficam fora: a receita vem da NF);
- **juros recebidos**: `titulo_recebimento.juros_centavos` com `NOT estornado` e `tipo = 'conta'` (no tipo
  `fomento` o cliente pagou direto ao fomento e o dinheiro não entrou na empresa), mês de
  `recebido_em`, empresa do título → `resultado_financeiro` (+);
- **encargos do borderô**: `valor_bruto_centavos − valor_liquido_centavos` de `antecipacao` com
  `status = 'ativa'`, mês de `operado_em` → `resultado_financeiro` (−);
- descontos concedidos no recebimento: ver D4 (só tipo `conta`, como os juros);
- recompra e repasse ao fomento entram pela conta a pagar do M7 (3.3), nunca daqui.

### 3.5 Caixa — `lancamento_caixa`

- `NOT estornado`, por `competencia` + `linha_dre` da categoria, valor com o sinal da linha do
  extrato (crédito +, débito −);
- `transferencia_conta` e `entre_contas` ficam fora; crédito de empréstimo (M14) cai em `fora_dre`.

### 3.6 Relatório semanal

Semana de segunda a domingo (`InicioDaSemana` do M7). Ver D7 para o que entra.

| Bloco | Fonte |
|---|---|
| NFs emitidas | `nota_fiscal` autorizada com `emitida_em` (Brasília) na semana, por empresa e natureza (quantidade e valor) |
| Pagamentos feitos | `pagamento` `NOT estornado` com `pago_em` (Brasília) na semana ⨝ `solicitacao`, por empresa e `linha_dre` |
| Recebimentos | `titulo_recebimento` `NOT estornado` com `recebido_em` na semana (D7) |
| Títulos vencidos | `titulo_receber` em `carteira`, `em_banco` ou `antecipado` com `vencimento < hoje` e saldo; contas a pagar `pendente_aprovacao`/`aprovado` com `prazo_pagamento < hoje` — foto de "hoje", por empresa |
| Estoque baixo | produtos ativos com `estoque_minimo > 0 AND quantidade_atual <= estoque_minimo` (mesma regra de `GET /produtos?estoque_baixo=true`; produto é do grupo) |
| Saldos bancários | porta mínima `Saldos(ctx, empresaIDs)` satisfeita pelo `SaldoRepository` do M13 (o mesmo de `GET /caixa/saldos`) |
| Endividamento | porta mínima `ParcelasEmAberto(ctx, empresaIDs)` satisfeita pelo `DividaConsultaRepository` do M14, resumida por `domaindivida.NovoResumo` com o "hoje" do relatório (D8) |

### 3.7 Faturamento histórico

Mesma consulta da 3.1 agrupada por mês e empresa (quantidade de notas, valor, ticket médio), sem
carga retroativa: mostra só as NFs já importadas (decisão de outubro/2026).

---

## 4. Endpoints

Base `/api/v1`, envelope `{ "dados": ... }`. Mês no formato `AAAA-MM`, datas `AAAA-MM-DD`.

```
GET /api/v1/dre/resumo?de=AAAA-MM&ate=AAAA-MM&empresa=<uuid>
    meses × empresas com as linhas da seção 2, detalhamento por categoria e indicadores,
    mais o consolidado por mês e o total do período.
    Padrão: os últimos 12 meses até o mês corrente; máximo 24 meses.

GET /api/v1/dre/faturamento?de=AAAA-MM&ate=AAAA-MM&empresa=<uuid>
    meses × empresas: quantidade de NFs de venda, valor, ticket médio; total por empresa
    e consolidado. Padrão: últimos 12 meses; máximo 60 meses.

GET /api/v1/dre/relatorio-semanal?semana=AAAA-MM-DD&empresa=<uuid>
    blocos da seção 3.6. `semana` é qualquer dia da semana desejada; padrão: a última
    semana encerrada. Listas com corte de 50 itens e total completo.

O endividamento é o bloco `endividamento` do relatório semanal; o painel completo de dívidas é o
`GET /api/v1/dividas/resumo` do M14 (D8).
```

Erros: `400` para mês/data inválidos, `de > ate` ou período acima do máximo; `403` para perfil sem
financeiro ou empresa fora do acesso.

---

## 5. Permissões

- Tudo em `financeiro(...)`, como os outros resumos (`/contas-a-pagar/resumo`, `/caixa/saldos`).
- Regra fina em `usecases/dre/permissoes.go`: `exigeFinanceiro` + `empresasVisiveis` (padrão do
  `usecases/caixa/permissoes.go`), com portas mínimas de usuário, cargo e empresa.
- `?empresa=` fora do acesso → `403`; sem `?empresa=`, só as empresas do acesso. Teste de isolamento
  no usecase e no e2e.

---

## 6. Contrato com os outros módulos

- **Nenhum port nem repositório de outro módulo muda.** As consultas da seção 3 vivem em
  repositórios do M15 no `adapter/postgres` (como o M13 lê `pagamento` e `titulo_recebimento`).
- Única dependência de código alheio: a porta mínima de saldos (`ports/out/dre`), satisfeita no
  `main.go` pelo `SaldoRepository` do M13.
- **M14**: nenhuma leitura de `divida`/`divida_parcela` antes da fase 5; até lá as contas do M14
  entram na DRE pela 3.3. A fase 5 só começa com o M14 na `main` e esta branch atualizada, lendo por
  porta mínima declarada em `ports/out/dre`.

---

## 7. Estrutura de código

```
migrations/000068_add_indices_dre.up.sql|.down.sql     só índices (renumerada depois da 000067 do M14)

internal/core/domain/dre/            data.go, mes.go, periodo.go, semana.go, intervalo.go, linha.go (chaves e itens),
                                     demonstrativo.go (montagem, subtotais, indicadores),
                                     faturamento.go, relatorio_semanal.go (+ testes)
internal/core/ports/in/dre/          resumo_usecase.go, faturamento_usecase.go,
                                     relatorio_semanal_usecase.go
internal/core/ports/out/dre/         resumo_repository.go (Lancamentos, ConsumosSemCusto, Faturamento), relatorio_semanal_repository.go,
                                     saldo_repository.go, usuario_repository.go,
                                     cargo_repository.go, empresa_repository.go
internal/core/usecases/dre/          resumo_usecase.go, faturamento_usecase.go,
                                     relatorio_semanal_usecase.go, permissoes.go (+ testes e fakes)
internal/adapter/postgres/           dre_resumo_repo.go, dre_relatorio_semanal_repo.go
internal/adapter/http/dto/dre/       resumo.go, faturamento.go, relatorio_semanal.go, consulta.go
internal/adapter/http/handlers/dre/  resumo_handler.go, faturamento_handler.go,
                                     relatorio_semanal_handler.go
test/integration/dre_*_test.go, test/e2e/dre_test.go
rotas.go, cmd/api/main.go, test/helpers/servidor.go       bloco do M15 no fim
```

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Domínio: período mensal/semanal, linhas e sinais, montagem do demonstrativo, subtotais, indicadores e ponto de equilíbrio + testes | concluída |
| **2** | `dre_resumo_repo.go` (receita, CMV, solicitação, títulos, caixa, faturamento) + índices + testes de integração com os cenários de fonte única | concluída |
| **3** | Usecases de resumo e faturamento + permissões + testes com fakes; DTOs, handlers, rotas, `main.go`, e2e e `API.md` | concluída |
| **4** | Relatório semanal: repositório, porta de saldos do M13, usecase, handler, e2e e `API.md` | concluída |
| **5** | Endividamento (depois do M14 na `main`): porta mínima de dívidas, bloco no relatório semanal, e2e e `API.md` | concluída |
| **6** | Front (repo irmão ou sessão própria F-M15) | pendente |

---

## 9. Decisões fechadas

- [x] **Receita**: só NF `venda` autorizada, por `emitida_em` (Brasília) e `valor_total_centavos`,
      fora `entre_empresas` (regra provisória da seção 6 do `ROADMAP.md`).
- [x] **CMV** pelos lotes (`produto_movimento_lote`), nunca por conta a pagar de importação nem
      `compra_mercadoria`.
- [x] **Despesas** pela `solicitacao` por `competencia` + `linha_dre`, ignorando `cancelado` e
      `rejeitado`, sem filtro por `origem` (acordo M14 × M15).
- [x] **Títulos `manual`** do M9 por competência; juros recebidos e encargos do borderô como
      resultado financeiro; recompra e repasse pela conta a pagar.
- [x] **`lancamento_caixa`** não estornado por competência; `transferencia_conta` e `entre_contas`
      fora; `fora_dre` à parte.
- [x] **Faturamento histórico** só com as NFs já importadas, sem carga retroativa.
- [x] **Saldos bancários** do relatório pela porta mínima satisfeita pelo M13.
- [x] **Endividamento** é a última fase, por porta mínima, depois do M14 na `main`.
- [x] **Permissão**: `financeiro(...)` com escopo por empresa.
- [x] **Lote de custo desconhecido** conta zero no CMV e é informado em `quantidade_sem_custo`.
- [x] **Estoque baixo** no nível do grupo (produto não tem empresa), regra de `estoque_baixo=true`.

## 10. Decisões do escopo (fechadas com o usuário, outubro/2026)

- [x] **D1 — Saídas de estoque.** `saida` (entrega do M6 e saída manual) = `cmv`; `perda` e `ajuste`
      negativo = `perdas_estoque`, logo abaixo do CMV; `transferencia`, `ajuste` positivo, `devolucao`,
      `entrada` e `implantacao` não mexem no CMV (o lote criado vira CMV quando sair).
- [x] **D2 — NF de devolução.** NF `devolucao` autorizada com `tipo_operacao = 'entrada'` (devolução do
      cliente) soma em `deducoes` pelo `valor_total_centavos`, no mês de `emitida_em` (Brasília); a de
      saída (devolução ao fornecedor) fica fora. Aparece no detalhamento de `deducoes` como item fixo
      `nf_devolucao`.
- [x] **D3 — Valor da conta a pagar.** `pago` usa o `valor` do pagamento vigente (`NOT estornado`); os
      demais status usam `valor_estimado`.
- [x] **D4 — Desconto concedido no recebimento.** `titulo_recebimento.desconto_centavos` não estornado e do
      tipo `conta` (no `fomento` o cliente pagou direto ao fomento; o prejuízo dele volta pela recompra),
      mês de `recebido_em`, entra em `resultado_financeiro` (−) como item fixo `descontos_concedidos`.
- [x] **D5 — Conta a pagar sem categoria.** Linha `nao_classificado`, dentro do resultado operacional,
      com quantidade, para o financeiro classificar.
- [x] **D6 — Ponto de equilíbrio.** Despesas fixas ÷ (margem de contribuição ÷ receita líquida), sem o
      resultado financeiro e sem `nao_classificado`; por mês e no total do período; nulo quando a margem
      é ≤ 0 ou a receita líquida é 0.
- [x] **D7 — Relatório semanal.** Blocos da 3.6 com recebimentos da semana, títulos vencidos a receber e
      contas a pagar vencidas; padrão = última semana encerrada (segunda a domingo).
- [x] **D8 — Endividamento sem rota própria** (outubro/2026, depois do M14 na `main`): o M14 já entrega
      `GET /dividas/resumo` com saldo devedor por credor e por empresa e vencido × a vencer; o M15 só acrescenta o
      bloco `endividamento` ao relatório semanal, com o mesmo cálculo (`ParcelasEmAberto` + `NovoResumo`), e não
      cria `GET /dre/endividamento`.

## 11. Decisões pendentes

Sem abertos.

---

## 12. Regras de código

As da seção 1 do `ROADMAP.md`: estrutura e padrão sem mudança, nenhum comentário, dinheiro em
centavos `int64`, erros com `%w`, `context.Context` primeiro, portas mínimas do lado do M15,
testes unitários, de integração e e2e.
