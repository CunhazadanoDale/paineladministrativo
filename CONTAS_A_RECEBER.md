# Escopo — M9 Contas a receber e antecipação

> **Status:** escopo fechado — decisões D1 a D6 respondidas pelo usuário (outubro/2026); API concluída
> (fases 1 a 5); front (fase 6) pendente, depois do F0a.
> **Objetivo:** tirar "A receber no banco" e os títulos do "Controle de NFs" das planilhas: título a
> receber por empresa e cliente, com cobrança em banco, envio de boleto, baixa total ou parcial,
> antecipação em borderô com fomento, recompra e repasse de pagamento feito na conta errada. Base
> para a geração de títulos pela NF-e (M8, última fase), o caixa e a conciliação (M13) e a DRE (M15).
> **Nome do módulo:** `receber`
> **Branch:** `feat/m9-contas-a-receber`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Módulo | Pacote novo `receber`: `domain/receber`, `ports/in|out/receber`, `usecases/receber`; rotas `/api/v1/titulos-receber` e `/api/v1/antecipacoes` |
| Dono do título | **O M9** (acordo "Coordenação M8 × M9" no `ROADMAP.md`): `titulo_receber` com `origem` + `origem_id` no padrão da `solicitacao` do M7 |
| Origens | `manual`, `pedido` (entrada da importação, M6) e `nota_fiscal_duplicata` (M8), **todas** no `chk_titulo_receber_origem` desde a primeira migration |
| `origem_id` | **Sem FK**: `nota_fiscal_duplicata` nasce no M8 e a origem `pedido` não trava o pedido como dono do título |
| Idempotência | `uq_titulo_receber_origem (origem, origem_id) WHERE origem <> 'manual'` + `INSERT ... ON CONFLICT DO NOTHING` |
| Geração por origem | `TituloReceberRepository.GerarDeOrigem` no padrão de `GerarDeRecorrencia` (seção 6), com `CancelarPorOrigem` para quando a origem é cancelada |
| Estados | `carteira` → `em_banco` → `antecipado` → `recebido`, mais `cancelado`; **`vencido` é derivado** da data, nunca gravado |
| Saldo | Baixa total ou parcial; o título só vira `recebido` quando o saldo zera. Recebimento é append-only, com estorno como lançamento inverso (padrão do M7) |
| Antecipação | Borderô (`antecipacao`) com o fomento como **fornecedor do M4** (D1), valor bruto, líquido creditado na conta e encargos. Cliente com `proibe_antecipacao` (M3) bloqueia |
| Recompra e repasse | Geram conta a pagar no M7 (`origem` `recompra` e `repasse_fomento`) na **mesma transação** da escrita do M9 (seção 6.3) |
| Entrada do pedido | A baixa total do título de entrada registra a entrada paga do pedido (M6) na mesma transação, por porta mínima do M9 (seção 6.2) |
| Escopo por empresa | Padrão da seção 4 do `EMPRESAS.md` em todo título, borderô e consulta |
| Permissões | **Financeiro e administrador** leem e escrevem (D5); título não é excluído, só cancelado (histórico append-only) |
| Concorrência | `versao` no título e no borderô (contrato do M4); transições com guarda de status e versão |
| "Hoje" | Dia no fuso de Brasília (`domainpedido.Hoje`), como o M6 |
| Dinheiro | Centavos `BIGINT`; taxa do borderô em pontos-base (`INT`) |

---

## 2. Entidades

### 2.1 `titulo_receber`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; empresa ativa e no acesso ao criar; não muda |
| `cliente_id` | UUID | FK `cliente` `RESTRICT`; cliente ativo ao criar título manual; não muda |
| `origem` | VARCHAR(30) | `manual`, `pedido`, `nota_fiscal_duplicata` (`chk_titulo_receber_origem`); não muda |
| `origem_id` | UUID NULL | **sem FK**; `chk_titulo_receber_origem_id`: nulo só em `manual`; não muda |
| `numero` | VARCHAR(30) | número do documento (`1234` da NF, `PED-57` do pedido, livre no manual); obrigatório |
| `parcela` | SMALLINT | `1..999`; `uq_titulo_receber_numero (empresa_id, origem, lower(numero), parcela) WHERE status <> 'cancelado'` |
| `vencimento` | DATE | pode ser anterior a hoje no título gerado (duplicata atrasada); no manual, não anterior a hoje |
| `valor_centavos` | BIGINT | `> 0`; não muda depois do primeiro recebimento ou antecipação |
| `categoria_financeira_id` | UUID NULL | FK `RESTRICT`; **obrigatória em `manual`**, nula nas outras (D6); lançável e de natureza `receita` |
| `competencia` | DATE | dia 1; só lida pelo M15 no título `manual`; omitida = mês do vencimento |
| `conta_bancaria_id` | UUID NULL | conta de **cobrança**; obrigatória em `em_banco`; ativa e da mesma empresa |
| `status` | VARCHAR(12) | `carteira`, `em_banco`, `antecipado`, `recebido`, `cancelado` |
| `recebido_centavos` | BIGINT | soma dos recebimentos vigentes (valor + desconto); `0 <= recebido <= valor`; cache mantido na transação |
| `boleto_enviado_em` | TIMESTAMPTZ NULL | último envio registrado |
| `boleto_canal` | VARCHAR(10) NULL | `email`, `whatsapp`, `portal`, `correio`, `outro` |
| `origem_cancelada_em` | TIMESTAMPTZ NULL | a origem foi cancelada com o título já movimentado (seção 3.4) |
| `cancelado_motivo` | VARCHAR(1000) NULL | obrigatório em `cancelado` (`chk_titulo_receber_cancelado_motivo`) |
| `observacao` | VARCHAR(1000) | |
| `usuario_id` | UUID | quem criou (ou disparou a geração) |
| `versao` | INT | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

Índices: `uq_titulo_receber_origem (origem, origem_id) WHERE origem <> 'manual'`,
`uq_titulo_receber_numero`, `idx_titulo_receber_vencimento (empresa_id, status, vencimento, id)`,
`idx_titulo_receber_cliente (cliente_id, vencimento, id)`, `idx_titulo_receber_conta (conta_bancaria_id)`.

- **Vencido** = `vencimento < hoje`, saldo maior que zero e status `carteira`, `em_banco` ou
  `antecipado` (o antecipado vencido é o risco de recompra). A resposta traz `vencido` e `saldo_centavos`.
- Título de origem diferente de `manual`: empresa, cliente, número, parcela, valor e competência vêm da
  origem e **não são editáveis**; o financeiro só muda vencimento (prorrogação, com motivo), conta de
  cobrança e observação.
- Cliente e categoria inativos depois da gravação não invalidam o título; a regra de ativo vale para
  gravar, conferida na transação com `FOR SHARE`, como no M7.

### 2.2 `titulo_receber_historico`

Append-only: `titulo_id`, `evento` (`criado`, `gerado`, `editado`, `prorrogado`, `enviado_banco`,
`retirado_banco`, `boleto_enviado`, `antecipado`, `antecipacao_estornada`, `recomprado`,
`recebido`, `recebimento_estornado`, `cancelado`, `origem_cancelada`), `de_status`, `para_status`,
`usuario_id`, `descricao`, `criado_em`.

### 2.3 `titulo_recebimento` e `titulo_recebimento_estorno`

| Campo (`titulo_recebimento`) | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `titulo_id` | UUID | FK `titulo_receber` `RESTRICT` |
| `tipo` | VARCHAR(10) | `conta` (dinheiro entrou numa conta da empresa) ou `fomento` (liquidado pelo cliente direto no fomento; só título `antecipado`) |
| `conta_bancaria_id` | UUID NULL | obrigatória em `conta`, nula em `fomento`; ativa e da mesma empresa; pode diferir da conta de cobrança |
| `recebido_em` | DATE | data do fato; sem data futura |
| `valor_centavos` | BIGINT | `> 0`; abate o saldo e entrou no caixa |
| `juros_centavos` | BIGINT | `>= 0`; juros e multa recebidos, não abatem o saldo |
| `desconto_centavos` | BIGINT | `>= 0`; abatimento concedido, abate o saldo sem entrar no caixa |
| `status_titulo_anterior` | VARCHAR(12) | status para onde o título volta no estorno |
| `registrou_entrada_pedido` | BOOLEAN | a baixa gravou a entrada paga do pedido (seção 6.2); não é estornada |
| `estornado` | BOOLEAN | mesmo mecanismo do `pagamento` do M7 |
| `usuario_id` / `criado_em` | | |

- `valor + desconto <= saldo`; saldo zerado → título `recebido`.
- Entrada no caixa (M13) = `valor + juros`, por `conta_bancaria_id` + `recebido_em`
  (`idx_titulo_recebimento_conta`). O tipo `fomento` não entra no caixa.
- **Pagamento na conta errada**: título `antecipado` recebido com `tipo = conta` gera o repasse ao
  fomento (seção 2.6) na mesma transação. Título `em_banco` recebido em outra conta da empresa é só
  uma baixa com outra conta (o front compara `conta_bancaria_id` do recebimento com a de cobrança).
- Estorno (`titulo_recebimento_estorno`: `recebimento_id` único, `motivo`, `estornado_em`,
  `usuario_id`, `criado_em`): devolve o saldo, e o título volta para `status_titulo_anterior` se
  estava `recebido`. Não estorna recebimento que gerou repasse, nem a baixa que registrou a entrada do
  pedido (`409`: corrigir pelo M7 ou pelo M6).

### 2.4 `antecipacao` (borderô) e `antecipacao_titulo`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK; todos os títulos são desta empresa |
| `fornecedor_id` | UUID | o fomento (D1); fornecedor ativo |
| `numero` | VARCHAR(30) NULL | número do borderô no fomento |
| `operado_em` | DATE | data do fato |
| `taxa_bp` | INT | taxa ao mês em pontos-base, informativa (`0..10000`) |
| `valor_bruto_centavos` | BIGINT | soma dos saldos dos títulos, calculada |
| `valor_liquido_centavos` | BIGINT | creditado na conta; `0 < liquido <= bruto` |
| `conta_bancaria_id` | UUID | conta do crédito; ativa e da mesma empresa |
| `status` | VARCHAR(10) | `ativa`, `estornada` |
| `usuario_id`, `versao`, `criado_em`, `atualizado_em` | | |

`antecipacao_titulo (antecipacao_id, titulo_id, valor_centavos, status_anterior, vigente)`, PK
composta, `uq_antecipacao_titulo_vigente (titulo_id) WHERE vigente`; `vigente` é a única coluna que
muda (recompra e estorno do borderô), com `UPDATE ... WHERE vigente`.

- Só título `carteira` ou `em_banco`, com saldo, sem `origem_cancelada_em`, de cliente sem
  `proibe_antecipacao` (`400 cliente não permite antecipação: <nome>`). Os títulos passam a `antecipado`.
- **Encargos** = bruto − líquido; o M15 os lê como `resultado_financeiro` sem categoria própria (D6).
- **Estorno do borderô** (lançamento errado): só se nenhum título dele foi recebido nem recomprado; os
  títulos voltam ao status anterior gravado em `antecipacao_titulo.status_anterior`.

### 2.5 `recompra`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `titulo_id` | UUID | título `antecipado` sem recebimento vigente |
| `antecipacao_id` | UUID | o borderô vigente do título |
| `recomprado_em` | DATE | data do fato |
| `encargos_centavos` | BIGINT | `>= 0`; juros cobrados pelo fomento |
| `valor_centavos` | BIGINT | saldo do título + encargos, calculado |
| `vencimento` | DATE | vencimento da conta a pagar ao fomento |
| `categoria_financeira_id` | UUID | categoria da conta a pagar (lançável, `despesa`) |
| `solicitacao_id` | UUID | FK `solicitacao`; a conta a pagar gerada |
| `motivo` | VARCHAR(500) | |
| `usuario_id`, `criado_em` | | |

O vínculo com o borderô sai do vigente (`antecipacao_titulo.vigente = false`) e o título volta para
`carteira`: a empresa cobra o cliente de novo.

### 2.6 `repasse_fomento`

`id`, `recebimento_id` (único), `antecipacao_id`, `valor_centavos` (valor recebido, sem juros),
`vencimento`, `categoria_financeira_id`, `solicitacao_id`, `usuario_id`, `criado_em`. Nasce na
transação do recebimento (seção 2.3) e gera a conta a pagar ao fomento.

---

## 3. Fluxos

### 3.1 Estados do título

| Ação | De | Para |
|---|---|---|
| criar (manual) / gerar (origem) | — | `carteira`, ou `em_banco` se vier com conta de cobrança |
| enviar ao banco | `carteira` | `em_banco` (conta de cobrança obrigatória) |
| retirar do banco | `em_banco` | `carteira` |
| antecipar (borderô) | `carteira`, `em_banco` | `antecipado` |
| estornar borderô | `antecipado` | status anterior |
| recomprar | `antecipado` | `carteira` |
| receber (saldo zerado) | `carteira`, `em_banco`, `antecipado` | `recebido` |
| estornar recebimento | `recebido` ou igual | status anterior ao recebimento |
| cancelar | `carteira`, `em_banco` sem recebimento vigente | `cancelado` |

Toda transição: `UPDATE` condicionado ao status e à `versao`, histórico na mesma transação.

### 3.2 Boleto

`POST /titulos-receber/{id}/boleto-enviado { canal, enviado_em, versao }` registra o envio (evento
`boleto_enviado`, sem trocar status). O alerta de boleto não enviado lista os títulos `em_banco` sem
envio com vencimento até `hoje + dias` (padrão 5).

### 3.3 Título de entrada do pedido (D3)

`POST /api/v1/pedidos/{id}/titulo-entrada { vencimento, conta_bancaria_id? }` (financeiro) gera, pela
mesma `GerarDeOrigem`, o título `origem = pedido`, `origem_id = pedido.id`, número `PED-<numero>`,
parcela 1, valor = `valor_total × entrada_bp / 10000` (arredondado para baixo). Só pedido de
`importacao`, `confirmado` ou `parcialmente_entregue` e sem entrada registrada; repetido responde o
título existente (`200`). A baixa que zera o saldo registra no pedido a entrada paga (`recebido_em` do
último recebimento, valor = soma dos `valor_centavos`) na mesma transação; se o pedido já tinha
entrada registrada à mão pelo M6, a baixa segue sem efeito no pedido e o histórico registra isso.

### 3.4 Origem cancelada (contrato com o M8)

`CancelarPorOrigem` recebe a origem e os `origem_id` cancelados (as duplicatas da NF) e, numa
transação:

- título `carteira` ou `em_banco` **sem recebimento vigente** → `cancelado` (evento `cancelado`, com
  o motivo vindo da origem);
- título `antecipado`, `recebido` ou com recebimento parcial → **não muda de status**: ganha
  `origem_cancelada_em` e o evento `origem_cancelada`, e aparece no filtro `?origem_cancelada=true`
  para o financeiro resolver (estornar recebimento, recomprar, devolver ao cliente). O cancelamento da
  origem **nunca é recusado** pelo M9 (D2);
- título já `cancelado` ou `origem_id` sem título → ignorado (idempotente).

A resposta conta `cancelados`, `marcados` e `ignorados`. Título com `origem_cancelada_em` não recebe
antecipação nem envio ao banco.

---

## 4. Endpoints

### Títulos
```
POST   /api/v1/titulos-receber                          criar manual (financeiro)
GET    /api/v1/titulos-receber                          ?empresa= ?cliente= ?status= ?origem= ?origem_id= ?vencimento_de=
                                                        ?vencimento_ate= ?vencidos=true ?origem_cancelada=true ?conta= + paginação (financeiro)
GET    /api/v1/titulos-receber/{id}                     detalhe com saldo e vencido (financeiro)
GET    /api/v1/titulos-receber/{id}/historico           (financeiro)
PUT    /api/v1/titulos-receber/{id}                     manual em carteira sem movimento: valor, vencimento, número, parcela,
                                                        categoria, competência, observação + versao (financeiro)
POST   /api/v1/titulos-receber/{id}/prorrogar           { vencimento, motivo, versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/enviar-banco        { conta_bancaria_id, versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/retirar-banco       { versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/boleto-enviado      { canal, enviado_em, versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/cancelar            { motivo, versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/recebimentos        { tipo, conta_bancaria_id, recebido_em, valor_centavos, juros_centavos,
                                                          desconto_centavos, versao, repasse: { vencimento, categoria_financeira_id } } (financeiro)
GET    /api/v1/titulos-receber/{id}/recebimentos        com estornos e repasse (financeiro)
POST   /api/v1/titulos-receber/{id}/recebimentos/{recebimento_id}/estorno  { motivo, estornado_em, versao } (financeiro)
POST   /api/v1/titulos-receber/{id}/recompra            { recomprado_em, encargos_centavos, vencimento, categoria_financeira_id,
                                                          motivo, versao } (financeiro)
GET    /api/v1/titulos-receber/boletos-pendentes        ?dias=5 ?empresa= (financeiro)
POST   /api/v1/pedidos/{id}/titulo-entrada              { vencimento, conta_bancaria_id } (financeiro)
```

### Antecipações
```
POST   /api/v1/antecipacoes                             { empresa_id, fornecedor_id, numero, operado_em, taxa_bp,
                                                          valor_liquido_centavos, conta_bancaria_id, titulo_ids } (financeiro)
GET    /api/v1/antecipacoes                             ?empresa= ?fornecedor= ?status= ?de= ?ate= + paginação (financeiro)
GET    /api/v1/antecipacoes/{id}                        com os títulos (financeiro)
POST   /api/v1/antecipacoes/{id}/estorno                { motivo, versao } (financeiro)
```

### Visão
```
GET    /api/v1/contas-a-receber/resumo                  ?de= ?ate= ?empresa= (financeiro)
```

Por semana (segunda a domingo, pelo vencimento) e por empresa, sem sobreposição: `a_receber`
(`carteira` + `em_banco` a vencer), `antecipado` (a vencer, só informativo) e `vencido`; atrasados
antes do período à parte, como no resumo do M7.

---

## 5. Permissões

- Rotas em `financeiro(...)`. Sem perfil novo. Não há `DELETE`: título lançado errado é cancelado,
  e o histórico fica.
- Regras finas em `usecases/receber/permissoes.go` (financeiro ou administrador), portas mínimas de
  usuário e cargo (só `Obter`).
- Escopo por empresa (itens 1 a 6 da seção 4 do `EMPRESAS.md`) em título, borderô, resumo e alerta;
  teste de isolamento no usecase e no e2e. O borderô só aceita títulos da própria empresa.

---

## 6. Contrato com os outros módulos

### 6.1 Geração por origem (M8 e o próprio M9)

Port de saída do M9 (`ports/out/receber/titulo_receber_repository.go`), satisfeito pelo
`postgres.TituloReceberRepository`. O M8 declara a porta mínima com só estes dois métodos em
`ports/out/notafiscal/`:

```go
GerarDeOrigem(ctx context.Context, geracao *domainreceber.Geracao) (domainreceber.ResultadoGeracao, error)
CancelarPorOrigem(ctx context.Context, cancelamento domainreceber.CancelamentoDeOrigem) (domainreceber.ResultadoCancelamento, error)
```

Domínio (`domain/receber/geracao.go`):

```go
type ParcelaDeOrigem struct {
	OrigemID      uuid.UUID
	Numero        string
	Parcela       int
	Vencimento    time.Time
	ValorCentavos int64
}

func NovaGeracao(origem Origem, empresaID, clienteID uuid.UUID, contaBancariaID *uuid.UUID,
	parcelas []ParcelaDeOrigem, usuarioID uuid.UUID, agora time.Time) (*Geracao, error)

type ResultadoGeracao struct {
	Gerados      []uuid.UUID
	JaExistentes []uuid.UUID
}

func NovoCancelamentoDeOrigem(origem Origem, origemIDs []uuid.UUID, motivo string,
	usuarioID uuid.UUID, agora time.Time) (CancelamentoDeOrigem, error)

type ResultadoCancelamento struct {
	Cancelados []uuid.UUID
	Marcados   []uuid.UUID
	Ignorados  int
}
```

- `NovaGeracao` recusa origem `manual`, lista vazia, `origem_id` repetido, número vazio, parcela < 1
  e valor ≤ 0 (`400`); monta os títulos (`carteira`, ou `em_banco` com conta) e o histórico `gerado`.
  Vencimento no passado é aceito.
- `GerarDeOrigem` grava **todas as parcelas numa transação**: trava a empresa (`FOR SHARE`, ativa),
  o cliente (`FOR SHARE`, existe; inativo é aceito, a NF já foi emitida), a conta de cobrança quando
  houver e a **origem** (switch `travarOrigem` por origem: `pedido` → `SELECT status FROM pedido ...
  FOR SHARE`, recusa cancelado). Cada parcela entra com `INSERT ... ON CONFLICT (origem, origem_id)
  WHERE origem <> 'manual' DO NOTHING RETURNING id`; a que já existia vai para `JaExistentes` (com o
  id do título existente) e não grava histórico. Rodar de novo é seguro.
- **A trava da origem `nota_fiscal_duplicata` é do M8** (entregue na fase 6 do `NOTAS_FISCAIS.md`):
  o `case` em `travarOrigem` chama `travarNotaFiscalDeOrigem` (no fim de `titulo_receber_repo.go`),
  que acha as notas das duplicatas, trava a `nota_fiscal` com `FOR SHARE` e recusa duplicata
  inexistente (`400 origem do título não encontrada`), NF cancelada (`409 origem cancelada não gera
  título`), NF entre empresas do grupo e NF que não é venda (`400`).
- `CancelarPorOrigem` trava os títulos das origens (`FOR UPDATE`, ordem por `id`) e aplica a seção
  3.4. O M8 chama depois de gravar o evento de cancelamento da NF; a corrida com a geração é fechada
  pela trava da `nota_fiscal` (o cancelamento a trava com `FOR UPDATE` no M8).
- Os erros seguem `tratarErroDeGravacao`; empresa inativa → `400 empresa inativa não recebe título`.

### 6.2 Pedido (M6)

Porta mínima do M9 em `ports/out/receber/pedido_repository.go` com só `Obter` (satisfeita pelo
`postgres.PedidoRepository`). O usecase valida e registra a entrada no **domínio do pedido**
(`Pedido.RegistrarEntrada`) e passa o pedido alterado na gravação do recebimento; o
`TituloReceberRepository` grava o pedido na mesma transação pela função interna `atualizarPedido`
do pacote `postgres` (com guarda de versão e status) e o histórico `entrada_registrada` do pedido.
Nenhum método novo no port nem no repositório do M6.

### 6.3 Conta a pagar (M7)

O M9 é a primeira sessão consumidora e fecha a assinatura prevista na seção 6 do
`CONTAS_A_PAGAR.md`:

- migration do M9 acrescenta `recompra` e `repasse_fomento` em `chk_solicitacao_origem` e cria
  `uq_solicitacao_origem (origem, origem_id) WHERE origem NOT IN ('manual', 'recorrencia')`;
- domínio do M7 ganha `OrigemRecompra`, `OrigemRepasseFomento` e
  `NovaSolicitacaoDeOrigem(DadosSolicitacaoDeOrigem{EmpresaID, SolicitanteID, FornecedorID,
  CategoriaFinanceiraID, Origem, OrigemID, ValorCentavos, Vencimento, Competencia, Observacao,
  FormaPagamento}, agora)`, que nasce em `pendente_aprovacao` (D4) com categoria e competência;
- `postgres` ganha `gerarSolicitacaoDeOrigem(ctx, tx, solicitacao, historico) (bool, error)` no fim
  de `solicitacao_repo.go`: confere empresa ativa e classificação gravável (as funções do M7),
  `INSERT ... ON CONFLICT (origem, origem_id) WHERE origem NOT IN ('manual','recorrencia') DO NOTHING`
  e histórico `criada`. `SolicitacaoRepository.GerarDeOrigem(ctx, solicitacao, historico) (bool,
  error)` a envolve numa transação própria, para M10, M12 e M14 declararem a porta mínima;
- o M9 usa a função interna dentro da transação da recompra e do recebimento, para a conta a pagar e
  o título nunca ficarem desencontrados. O solicitante é quem registrou; a aprovação segue o fluxo do
  M7.

### 6.4 Demais

- Cliente (M3): porta mínima `Obter` (`Ativo`, `ProibeAntecipacao`, `Nome`); a trava `FOR SHARE` fica
  no repositório do M9.
- Conta bancária e categoria (M5): `Obter`; regras da seção 6 do `FINANCEIRO.md` (M9 aceita
  `receita` no título, `despesa` na recompra e no repasse).
- Fornecedor (M4): `Obter` (`Ativo`).
- M13 concilia `titulo_recebimento` (tipo `conta`) e o crédito do borderô por `conta_bancaria_id` +
  data; M15 usa título `manual` por `competencia` + categoria, juros recebidos e encargos do borderô
  como `resultado_financeiro`, e ignora os títulos de NF e de pedido (a receita vem da NF).

---

## 7. Estrutura de código

```
migrations/000047_create_tabela_titulo_receber           titulo_receber, titulo_receber_historico
migrations/000048_create_tabela_titulo_recebimento       recebimento, estorno
migrations/000049_create_tabela_antecipacao              antecipacao, antecipacao_titulo, recompra, repasse_fomento
migrations/000050_alter_solicitacao_origem_receber       chk_solicitacao_origem + uq_solicitacao_origem

internal/core/domain/receber/        origem.go, status.go, titulo.go, historico.go, geracao.go,
                                     recebimento.go, antecipacao.go, recompra.go, repasse.go,
                                     canal_boleto.go, resumo.go, versao.go (+ testes)
internal/core/domain/solicitacao/    origem.go (+2 valores), origem_sistemica.go
internal/core/ports/out/receber/     titulo_receber_repository.go, antecipacao_repository.go,
                                     resumo_repository.go, empresa_repository.go, cliente_repository.go,
                                     pedido_repository.go, conta_bancaria_repository.go,
                                     categoria_financeira_repository.go, fornecedor_repository.go,
                                     usuario_repository.go, cargo_repository.go
internal/core/ports/in/receber/      titulo_usecase.go, recebimento_usecase.go, antecipacao_usecase.go,
                                     resumo_usecase.go
internal/core/usecases/receber/      idem + permissoes.go (+ testes, fakes e cenário)
internal/adapter/postgres/           titulo_receber_repo.go, antecipacao_repo.go, contas_a_receber_resumo_repo.go;
                                     solicitacao_repo.go (só acréscimo no fim)
internal/adapter/http/dto/receber/ + handlers/receber/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers + test/integration + test/e2e
```

- Numeração a partir da `000047` (a `main` está na `000046`); o M8 também começa na `000047`, então
  quem mergear depois renumera (regra 4 da seção 3 do `ROADMAP.md`).
- O domínio `receber` importa só `domain` e `github.com/google/uuid`; pedido, cliente e contas entram
  pelo usecase.

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Título manual: migration 000047, domínio (origem, status, título, histórico, vencido derivado), repositório, usecase com escopo por empresa, criar/editar/prorrogar/enviar e retirar do banco/boleto/cancelar, listagem e histórico, e2e + `API.md` | concluída |
| **2** | Geração por origem: `Geracao`, `GerarDeOrigem`, `CancelarPorOrigem` (contrato da seção 6.1, testes de integração de idempotência e de corrida) e título de entrada do pedido | concluída |
| **3** | Baixa: recebimento total e parcial, estorno, registro da entrada paga do pedido na mesma transação (migration 000048) | concluída |
| **4** | Antecipação: borderô, estorno, recompra e repasse ao fomento com a conta a pagar do M7 na mesma transação (migrations 000049–000050) | concluída |
| **5** | Visões: resumo por semana/empresa e alerta de boleto não enviado | concluída |
| **6** | Frontend (depois do F0a): títulos, baixa, borderô, recompra, resumo e alertas | pendente |

Cada fase atravessa as camadas até o e2e e o `API.md` e termina com os gates verdes.

---

## 9. Decisões fechadas

- [x] **O M9 é dono do título**, com `origem` + `origem_id` sem FK e as três origens no CHECK desde a
      primeira migration (acordo do `ROADMAP.md`).
- [x] **Idempotência por índice único parcial** `(origem, origem_id) WHERE origem <> 'manual'`.
- [x] **Geração em lote por documento de origem** numa transação, com trava da origem; a trava da NF é
      acrescentada pelo M8.
- [x] **`vencido` derivado** da data.
- [x] **Recebimento append-only** com estorno, no padrão do pagamento do M7.
- [x] **Conta a pagar e entrada do pedido na mesma transação** da escrita do M9, por função interna
      do `postgres`, sem método novo nos ports do M6 e do M7.
- [x] **"Hoje" em Brasília**, como o M6.
- [x] **Sem exclusão de título** (fase 1): o cancelamento com motivo substitui o `DELETE`, para o
      histórico nunca ser apagado.
- [x] **Título de entrada não é regerado** (fase 2): o índice de idempotência vale também para o
      título cancelado; gerar de novo devolve o existente (`200`).
- [x] **`travarOrigem` antes das demais travas** (fase 2): a origem é travada primeiro e cada `case`
      recusa origem cancelada; o `case` de `pedido` está pronto e o de `nota_fiscal_duplicata` é do M8.
- [x] **`registrou_entrada_pedido` no recebimento** (fase 3): marca a baixa que gravou a entrada no
      pedido e bloqueia o estorno dela no próprio `UPDATE`.
- [x] **Conta a pagar gerada pelo M9** (fase 4): `domainsolicitacao.NovaSolicitacaoDeOrigem` e
      `gerarSolicitacaoDeOrigem` (fim de `solicitacao_repo.go`) com `uq_solicitacao_origem`; o M9
      chama a função interna na transação da recompra e do recebimento, e
      `SolicitacaoRepository.GerarDeOrigem` fica para M10, M12 e M14.
- [x] **Repasse sem os juros** (fase 4): o repasse ao fomento é o `valor_centavos` recebido; os juros
      pagos pelo cliente ficam com a empresa até o financeiro decidir outro acerto com o fomento.
- [x] **Número único por empresa e origem** (fase 1), sem diferenciar maiúsculas: o número da NF não
      colide com um título manual de mesmo número.
- [x] **D1 — Fomento é fornecedor do M4.** Sim; o borderô aponta para `fornecedor`, e a
      recompra e o repasse viram conta a pagar desse fornecedor.
- [x] **D2 — Origem cancelada com título movimentado.** O título antecipado, recebido ou com
      baixa parcial não é cancelado; é marcado (`origem_cancelada_em`) e listado para o financeiro
      resolver, e o cancelamento da NF nunca é recusado pelo M9.
- [x] **D3 — Título de entrada do pedido.** Gerado pelo financeiro (`POST
      /pedidos/{id}/titulo-entrada`), não na confirmação do M6; cancelar o pedido **não** cancela o
      título sozinho (o financeiro cancela; a baixa de título de pedido cancelado é aceita, mas não
      registra entrada, e o histórico avisa). Ligar o cancelamento do pedido ao
      `CancelarPorOrigem` mexe no usecase do M6 e fica fora.
- [x] **D4 — Conta a pagar da recompra e do repasse** nasce em `pendente_aprovacao`, como a
      recorrência (D3 do M7).
- [x] **D5 — Quem vê títulos.** Só financeiro e administrador; comercial e representante
      ficam fora até o M10/M11 pedirem.
- [x] **D6 — Categoria.** Obrigatória (receita) só no título manual; títulos de NF e de
      pedido sem categoria (a receita vem da NF no M15); juros recebidos e encargos do borderô sem
      categoria, lidos pelo M15 como `resultado_financeiro`.

## 10. Decisões pendentes

Sem abertos.
