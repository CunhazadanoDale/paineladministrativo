# Escopo — M6 Pedidos de venda e entregas

> **Status:** escopo fechado; nenhuma fase iniciada.
> **Objetivo:** tirar "Vendas futuras", "Entregas de importação" e "Pendências de envio" das
> planilhas: pedido de venda por empresa, com reserva de estoque, entregas parciais que baixam o
> estoque e consultas de pendência, prazo e cancelamento. Base para NF-e (M8), comissões (M10),
> propostas (M11) e importações (M12).
> **Nome do módulo:** `pedido`
> **Branch:** `feat/m6-pedidos`

---

## 1. Visão geral

Decisões já tomadas com o usuário (outubro/2026):

| Tema | Decisão |
|---|---|
| Empresa faturadora | **Escolhida no pedido**, entre as empresas ativas no acesso do usuário; sem regra automática |
| Gravação da entrega | **Uma transação só**: o repositório de pedidos abre a `tx` e grava a entrega e as operações de estoque nela, por função interna do pacote `postgres` extraída do `EstoqueRepository.Gravar` (que mantém o comportamento) |
| Entrega parcial | **Libera e reserva de novo**: na mesma transação, libera a reserva do item, dá saída do entregue e cria reserva nova com o restante. A reserva integral do M2 não muda |
| Local de saída | **Só locais da empresa vendedora**; estoque de outra empresa exige transferência antes (aguarda a decisão fiscal entre CNPJs) |
| Entrada paga (importação) | **Registrada no pedido** (data e valor), com histórico; o prazo de 120 dias conta dessa data. No M9 a baixa do título de entrada passa a fazer esse registro pelo port |
| Cancelamento com entrada paga | **Só registra** o valor a devolver no pedido; a conta a pagar no M7 fica como pendência para depois do M7 |
| Número do pedido | **Sequencial por empresa**, único dentro da empresa |
| Preço | **Digitado**; o front sugere `produto.preco` (o M11 troca a sugestão pela tabela de preços) |
| Edição | **Itens editáveis só em `proposta`**; confirmado só cancela |
| Itens sem estoque | **Ação manual** "reservar pendências"; o M12 automatiza no recebimento da importação |
| Frete | **Tipo e valor** na entrega: `cif`/`fob`, valor em centavos (opcional no FOB) e transportadora |
| Prazo de pronta entrega | **Opcional, digitado**; na importação é calculado |
| Cancelamento | **Comercial e administrador**, com motivo obrigatório; representante com login cancela só a própria proposta |

---

## 2. Entidades

### 2.1 `pedido`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; empresa vendedora, ativa e no acesso ao criar |
| `numero` | INT | sequencial por empresa; `uq_pedido_empresa_numero` |
| `cliente_id` | UUID | FK `cliente`; ativo ao criar ou trocar |
| `obra_id` | UUID NULL | FK `cliente_obra`; da mesma obra do cliente e ativa; endereço de entrega |
| `representante_id` | UUID NULL | FK `representante`; padrão é o do cliente; ativo ao atribuir |
| `modelo` | VARCHAR(16) | `pronta_entrega` / `importacao` |
| `entrada_bp` | INT | percentual de entrada em pontos-base (`3000` na importação, `0` na pronta entrega) |
| `prazo_boleto_dias` | INT | `30` na pronta entrega |
| `prazo_entrega_dias` | INT NULL | `120` na importação, contado de `entrada_paga_em` |
| `prazo_limite` | DATE NULL | pronta entrega: digitado e opcional; importação: `entrada_paga_em + prazo_entrega_dias` |
| `numero_pedido_cliente` | VARCHAR(60) NULL | número de compra do cliente (o M8 cobra quando `exige_numero_pedido_na_nf`) |
| `observacoes` | VARCHAR(2000) | |
| `status` | VARCHAR(24) | `proposta` → `confirmado` → `parcialmente_entregue` → `entregue`; `cancelado` |
| `valor_total_centavos` | BIGINT | soma dos itens, recalculada no domínio |
| `entrada_paga_em` | DATE NULL | só importação; data do fato, sem data futura |
| `entrada_paga_centavos` | BIGINT NULL | |
| `cancelado_motivo` | VARCHAR(1000) NULL | obrigatório ao cancelar |
| `valor_a_devolver_centavos` | BIGINT NULL | só no cancelamento com entrada paga |
| `usuario_id` | UUID | quem criou |
| `versao` | BIGINT | versão otimista (padrão do M4) |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Condições por modelo:** value object `Condicao` (modelo, entrada em pontos-base, prazo do
boleto, prazo de entrega) com os padrões do modelo; os valores ficam **gravados no pedido** para
uma mudança futura de padrão não alterar pedidos antigos.

### 2.2 `pedido_item`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `pedido_id` | UUID | FK `pedido` |
| `produto_id` | UUID | FK `produto`; ativo ao incluir; único por pedido |
| `quantidade` | INT | `> 0`, `int32` |
| `preco_unitario_centavos` | BIGINT | `>= 0` |
| `quantidade_entregue` | INT | mantida na transação da entrega; `<= quantidade` |
| `criado_em` | TIMESTAMPTZ | |

As reservas do item ficam em `produto_reserva` com `origem = 'pedido_item'` e
`origem_id = pedido_item.id`; um item pode ter reservas em mais de um local da empresa.

### 2.3 `pedido_historico`

Append-only, como `solicitacao_historico`: `pedido_id`, `acao` (`criado`, `alterado`,
`confirmado`, `entrada_registrada`, `pendencias_reservadas`, `entrega_criada`, `entrega_saiu`,
`entrega_concluida`, `entrega_cancelada`, `cancelado`), `status_anterior`, `status_novo`,
`usuario_id`, `detalhe` (texto), `criado_em`.

### 2.4 `pedido_sequencia`

`empresa_id` (PK, FK `empresa`), `ultimo_numero` INT. O número sai de
`INSERT ... ON CONFLICT DO UPDATE SET ultimo_numero = ultimo_numero + 1 RETURNING` na mesma
transação da criação.

### 2.5 `entrega` e `entrega_item`

| Campo (`entrega`) | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `pedido_id` | UUID | FK `pedido`; pedido `confirmado` ou `parcialmente_entregue` |
| `status` | VARCHAR(10) | `pendente` → `saiu` → `entregue`; `cancelada` só a partir de `pendente` |
| `transportadora` | VARCHAR(160) NULL | |
| `frete_tipo` | VARCHAR(3) | `cif` / `fob` |
| `frete_centavos` | BIGINT NULL | obrigatório no CIF, opcional no FOB, `>= 0` |
| `previsao` | DATE NULL | |
| `saiu_em` / `entregue_em` | DATE NULL | data do fato, sem data futura |
| `usuario_id` | UUID | |
| `versao` | BIGINT | |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

| Campo (`entrega_item`) | Tipo | Observações |
|---|---|---|
| `entrega_id` / `pedido_item_id` | UUID | PK composta |
| `local_id` | UUID | FK `estoque_local`; da empresa do pedido |
| `quantidade` | INT | `> 0`; soma das entregas não canceladas `<=` quantidade do item |
| `movimento_id` | UUID NULL | FK `produto_movimento`; preenchido na baixa |

---

## 3. Regras de estado e de estoque

### 3.1 Pedido

| Ação | De | Para | Efeito |
|---|---|---|---|
| criar | — | `proposta` | número da empresa; condições do modelo |
| alterar | `proposta` | `proposta` | cabeçalho e itens; exige `versao` |
| confirmar | `proposta` | `confirmado` | reserva o disponível de cada item nos locais da empresa (ver 3.2); o restante fica pendente de cobertura |
| registrar entrada | `confirmado`, `parcialmente_entregue` | igual | só importação, uma vez; grava data, valor e `prazo_limite` |
| reservar pendências | `confirmado`, `parcialmente_entregue` | igual | reserva o que estiver disponível para o que falta cobrir |
| entrega sai | `confirmado`, `parcialmente_entregue` | `parcialmente_entregue` ou `entregue` | quando todo item tem `quantidade_entregue = quantidade` → `entregue` |
| cancelar | qualquer não final | `cancelado` | libera as reservas ativas; entregas `pendente` são canceladas; entregas `entregue` permanecem; entrega em trânsito (`saiu`) impede o cancelamento (`409`) até ser concluída |

Toda transição usa o `UPDATE` condicionado ao status anterior e à `versao` (zero linhas → `409`
se existe, `404` se não) e grava o histórico na mesma transação.

### 3.2 Reserva na confirmação

Para cada item, percorre os locais ativos da empresa (ordem: maior disponível primeiro, depois
`id`) e reserva o disponível até cobrir a quantidade. Coberto = soma das reservas ativas do item +
entregue. Item sem nada disponível não bloqueia a confirmação.

### 3.3 Entrega

- **Criar** (`pendente`): valida item, local da empresa e quantidade restante; não mexe no estoque.
- **Saiu**: baixa o estoque na mesma transação, por item:
  - reserva ativa do item no local cobre exatamente a quantidade → saída com `reserva_id`;
  - reserva maior que a quantidade → libera a reserva, saída sem reserva, reserva nova com o
    restante (`origem = 'pedido_item'`);
  - sem reserva suficiente no local → usa a reserva que houver e completa pelo disponível do
    local; sem disponível → `400`.
  - grava `entrega_item.movimento_id`, soma `pedido_item.quantidade_entregue` e avança o pedido.
- **Entregue**: só registra `entregue_em` (o estoque já saiu em `saiu`).
- **Cancelar**: só `pendente`; não mexe no estoque.

### 3.4 Encadeamento de operações de estoque

Uma `OperacaoEstoque` carrega uma reserva criada, uma encerrada e um movimento, com uma
`VersaoLida`. Liberar, dar saída e reservar de novo o mesmo produto são três operações
encadeadas: o domínio de estoque passa a expor a aplicação da operação no `EstoqueProduto` (hoje
só existe no teste), para a operação seguinte ler a versão já avançada. No `postgres`, o corpo
de `EstoqueRepository.Gravar` vira `gravarOperacaoEstoque(ctx, tx, operacao)`, chamada por
`Gravar` (sem mudança de comportamento) e pelo repositório de pedidos dentro da própria `tx`.

---

## 4. Endpoints

### Pedidos
```
POST   /api/v1/pedidos                          criar proposta (comercial)
GET    /api/v1/pedidos                          ?empresa= ?cliente= ?representante= ?status= ?modelo= ?numero= (autenticado, escopo por empresa)
GET    /api/v1/pedidos/{id}                     detalhe com itens, coberto, entregue e reservas (autenticado)
PUT    /api/v1/pedidos/{id}                     alterar proposta, com versao (comercial)
POST   /api/v1/pedidos/{id}/confirmar           { versao } (comercial)
POST   /api/v1/pedidos/{id}/entrada             { versao, pago_em, valor_centavos } (financeiro)
POST   /api/v1/pedidos/{id}/reservar-pendencias { versao } (comercial)
POST   /api/v1/pedidos/{id}/cancelar            { versao, motivo, valor_a_devolver_centavos } (comercial)
GET    /api/v1/pedidos/{id}/historico           (autenticado)
```

### Entregas
```
POST   /api/v1/pedidos/{id}/entregas            { transportadora, frete_tipo, frete_centavos, previsao, itens: [{ pedido_item_id, local_id, quantidade }] } (comercial)
GET    /api/v1/pedidos/{id}/entregas            (autenticado)
GET    /api/v1/entregas                         ?empresa= ?status= ?previsao_ate= (autenticado, escopo por empresa)
POST   /api/v1/entregas/{id}/saiu               { versao, saiu_em } (comercial)
POST   /api/v1/entregas/{id}/entregue           { versao, entregue_em } (comercial)
POST   /api/v1/entregas/{id}/cancelar           { versao } (comercial)
```

### Consultas
```
GET    /api/v1/pedidos/pendencias               por item: vendido, entregue, reservado, falta cobrir, disponível na empresa; ?empresa= ?cliente= ?produto=
GET    /api/v1/pedidos/prazos                   ?situacao=vencido|a_vencer ?dias=30 ?empresa=
GET    /api/v1/pedidos/cancelamentos            ?com_devolucao=true ?empresa= ?de= ?ate=
```

---

## 5. Permissões

- Perfis existentes, sem flag nova: `comercial(...)` cria, altera, confirma, reserva pendências,
  cancela e opera entregas; `financeiro(...)` registra a entrada paga; `protegido(...)` lê.
- **Escopo por empresa** (padrão da seção 4 do `EMPRESAS.md`): toda leitura e escrita respeita
  `empresa.Acesso`; empresa inativa → `400` ao criar; teste de isolamento obrigatório no usecase
  e no e2e.
- **Representante com login** (não administrador): vê e altera só os pedidos em que é o
  representante; pedido que ele cria fica sempre com ele (informar outro → `403`); cancela só a
  própria proposta; regra repetida de `usecases/cliente/permissoes.go` com
  `RepresentanteRepository.ObterPorUsuario`.

---

## 6. Estrutura de código

```
migrations/0000NN_create_tabela_pedido                 pedido, pedido_sequencia, pedido_item, pedido_historico
migrations/0000NN_create_tabela_entrega                entrega, entrega_item
migrations/0000NN_alter_produto_reserva_origem_pedido  CHECK de origem com pedido_item

internal/core/domain/pedido/          modelo.go, condicao.go, status.go, pedido.go, item.go,
                                      historico.go, entrega.go, frete.go, cobertura.go
internal/core/ports/out/pedido/       pedido_repository.go, entrega_repository.go, consulta_repository.go,
                                      estoque_repository.go (porta mínima: Carregar),
                                      empresa_repository.go, cliente_repository.go, obra_repository.go,
                                      representante_repository.go, produto_repository.go, local_repository.go
internal/core/ports/in/pedido/        pedido_usecase.go, entrega_usecase.go, consulta_usecase.go
internal/core/usecases/pedido/        (+ permissoes.go, testes e fakes)
internal/adapter/postgres/            pedido_repo.go, entrega_repo.go, pedido_consulta_repo.go,
                                      estoque_repo.go (extração de gravarOperacaoEstoque)
internal/core/domain/estoque/         OrigemReservaPedidoItem; aplicação de operação exportada
internal/adapter/http/dto/pedido/ + handlers/pedido/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
```

Numeração de migration definida no merge (regra 4 da seção 3 do `ROADMAP.md`): a `main` está em
`000039` e o M7 corre em paralelo.

---

## 7. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations + domínio (condição, status, pedido, item, entrega, frete, cobertura) + ajustes no domínio de estoque (origem `pedido_item`, encadeamento) + testes | pendente |
| **2** | Repositórios (sequência, transições com guarda, entrega com estoque na mesma `tx`, consultas) + testes de integração | pendente |
| **3** | Usecases + permissões (escopo por empresa, representante com login) + testes com fakes e de isolamento | pendente |
| **4** | Handlers/DTOs + rotas + e2e + `API.md` | pendente |
| **5** | Consultas de pendências, prazos e cancelamentos + testes + `API.md` | pendente |
| **6** | Frontend: pedidos, confirmação, entregas e consultas (depois do F0a) | pendente |

---

## 8. Decisões fechadas

- [x] **Empresa faturadora** escolhida no pedido.
- [x] **Entrega e estoque na mesma transação**, por função interna do `postgres`.
- [x] **Entrega parcial**: libera, dá saída e reserva de novo o restante.
- [x] **Saída só de local da empresa vendedora.**
- [x] **Entrada paga registrada no pedido**; prazo de 120 dias conta dela.
- [x] **Cancelamento só registra a devolução**; conta a pagar fica para depois do M7.
- [x] **Número sequencial por empresa.**
- [x] **Preço digitado**, sugerido de `produto.preco`.
- [x] **Itens editáveis só em proposta.**
- [x] **Reservar pendências por ação manual.**
- [x] **Frete com tipo e valor.**
- [x] **Prazo-limite opcional na pronta entrega.**
- [x] **Cancelamento por comercial e administrador**; representante só a própria proposta.
- [x] **Momento da baixa**: o estoque sai quando a entrega passa a `saiu`; `entregue` só
      registra a data.
- [x] **Condições fixas por modelo**: entrada, prazo do boleto e prazo de entrega gravados no
      pedido com o padrão do modelo e **não editáveis** no M6; o M11 pode abrir a edição.
- [x] **Valor a devolver**: informado no cancelamento, padrão igual à entrada paga e nunca maior
      que ela, mesmo com parte já entregue; o acerto do que foi entregue fica com o financeiro.
- [x] **Ordem dos locais na confirmação**: maior disponível primeiro, em vez de um local padrão
      por empresa.

## 9. Decisões pendentes

Sem abertos.
