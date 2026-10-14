# Escopo — M5 Contas bancárias e plano de contas

> **Status:** escopo fechado — pronto para implementação. As decisões da seção 1 foram delegadas
> pelo usuário à sessão (outubro/2026), seguindo a boa prática.
> **Objetivo:** tirar a "classificação manual dos extratos" da planilha: cadastro das contas
> bancárias de cada empresa e plano de contas único do grupo, base para contas a pagar (M7),
> contas a receber (M9), caixa e conciliação (M13), dívidas (M14) e DRE (M15).
> **Nome do módulo:** `financeiro`
> **Branch:** `feat/m5-financeiro`

---

## 1. Visão geral

| Tema | Decisão |
|---|---|
| Conta bancária | **Por empresa** (`empresa_id`), com o padrão de escopo da seção 4 do `EMPRESAS.md` |
| Plano de contas | **Do grupo**: um plano único; a DRE por empresa (M15) sai do mesmo plano |
| Classificação da DRE | **`linha_dre`** substitui o `grupo_dre` do `ROADMAP.md`, que não separava frete, comissões, impostos e custos fixos (seção 2.2) |
| Natureza | `receita`, `despesa` e **`transferencia`** (movimento entre contas próprias não é receita nem despesa) |
| Dois níveis | Natureza e linha DRE nascem na raiz; a subcategoria **herda**. Lançamento (M7, M9, M13) só em **subcategoria ativa** |
| Banco | **Código COMPE** com catálogo fechado dos bancos em uso; ampliar = código + migration |
| Imutabilidade | Na conta, só o apelido muda; na categoria, natureza, linha DRE e pai não mudam (corrigir = desativar e criar outra) |
| Leitura | Contas bancárias só para **financeiro e administrador**, filtradas pelo acesso à empresa; plano de contas para qualquer autenticado |
| Escrita | **Financeiro e administrador** criam, alteram, ativam e desativam; **só o administrador** exclui |
| Perfil financeiro | **Reaproveitado do M4** (`financeiro(...)` em `rotas.go`); o M5 não cria middleware nem campo de login |
| Concorrência | **Versão otimista** no contrato do M4 (seção 3 do `FORNECEDORES.md`), mais guardas atômicas nas regras que dependem de outra linha |
| Saldo inicial | **Fica no M13** (abertura da conta no fechamento do caixa); o M5 não tem dinheiro nem data de fato |
| Seed | **Nenhum**: contas e plano de contas são cadastrados pela API/tela, como as empresas no M1 |

---

## 2. Entidades

### 2.1 `conta_bancaria`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; não muda |
| `banco` | CHAR(3) | código COMPE do catálogo (seção 2.3); não muda |
| `tipo` | VARCHAR(10) | `corrente`, `poupanca`, `pagamento`, `aplicacao`; não muda |
| `agencia` | VARCHAR(5) | só dígitos (`^[0-9]{1,5}$`), como digitado (zeros à esquerda preservados); não muda |
| `agencia_digito` | VARCHAR(1) NULL | `^[0-9X]$`; não muda |
| `conta` | VARCHAR(20) | só dígitos (`^[0-9]{1,20}$`); não muda |
| `conta_digito` | VARCHAR(2) NULL | `^[0-9X]{1,2}$`; não muda |
| `apelido` | VARCHAR(60) | obrigatório; único por empresa sem diferenciar maiúsculas; único campo editável |
| `ativa` | BOOLEAN | default `true` |
| `versao` | INT | começa em `1`; avança a cada alteração gravada |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Constraints:** `chk_conta_bancaria_banco` (catálogo), `chk_conta_bancaria_tipo`,
`chk_conta_bancaria_agencia`, `chk_conta_bancaria_agencia_digito`, `chk_conta_bancaria_conta`,
`chk_conta_bancaria_conta_digito`, `chk_conta_bancaria_apelido`, `chk_conta_bancaria_versao`
(`versao >= 1`), `fk_conta_bancaria_empresa`.
Índices: `uq_conta_bancaria_numero` em `(banco, ltrim(agencia, '0'), ltrim(conta, '0'))`,
`uq_conta_bancaria_empresa_apelido` em `(empresa_id, lower(apelido))` e
`idx_conta_bancaria_empresa (empresa_id, apelido, id)`.

**Regras:**
- Entrada aceita com máscara (`1234-5`, `12.345-6`): o domínio separa número e dígito quando o
  corpo traz o dígito junto e recusa qualquer caractere além de dígitos, `X`, `-`, `.` e espaço.
- Dígito verificador **não é conferido**: cada banco usa um algoritmo, e um DV errado é corrigido
  desativando e criando outra conta.
- A unicidade ignora zeros à esquerda (`0001`/`1`) e o dígito: a mesma conta não existe duas
  vezes no grupo, nem em empresas diferentes → `409 já existe uma conta com este banco, agência
  e número`. Apelido repetido na empresa → `409 já existe uma conta com este apelido na empresa`.
- Agência e conta só com dígitos permitem ao M13 casar o extrato OFX (`BANKID`, `BRANCHID`,
  `ACCTID`) com a conta sem intervenção.
- Empresa inativa não recebe conta nova e não tem conta reativada → `400 empresa inativa não
  recebe conta bancária`. A checagem é feita na própria escrita (`INSERT ... SELECT ... FROM
  empresa WHERE ativa` e `UPDATE` condicionado), não só antes dela.
- Conta inativa continua no histórico e não recebe lançamento novo (regra aplicada pelos
  consumidores via porta mínima, seção 6).

### 2.2 `categoria_financeira`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `nome` | VARCHAR(120) | obrigatório; único entre as raízes e entre as filhas do mesmo pai, sem diferenciar maiúsculas |
| `categoria_pai_id` | UUID NULL | FK composta para `categoria_financeira` `RESTRICT` (seção abaixo); só raiz pode ser pai (dois níveis); não muda |
| `natureza` | VARCHAR(13) | `receita`, `despesa`, `transferencia`; na filha, copiada do pai; não muda |
| `linha_dre` | VARCHAR(20) | ver tabela abaixo; na filha, copiada do pai; não muda |
| `codigo` | VARCHAR(20) NULL | código contábil opcional (`^[0-9]+(\.[0-9]+)*$`, ex.: `4.1.03`); único quando informado |
| `ordem` | INT | `>= 0`, default `0`; ordena a exibição |
| `ativa` | BOOLEAN | default `true` |
| `versao` | INT | começa em `1` |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Linhas da DRE** (o que o M15 faz com cada uma):

| `linha_dre` | Natureza aceita | Uso no M15 |
|---|---|---|
| `receita_operacional` | `receita` | receita que não vem de NF (a receita de venda vem do M8) |
| `deducao_venda` | `despesa` | impostos sobre a venda, devoluções |
| `compra_mercadoria` | `despesa` | **fora do resultado**: compra de estoque e importação; o custo entra como CMV pelos lotes do M2, nunca pelo pagamento |
| `despesa_variavel` | `despesa` | frete, comissões, taxas de venda |
| `despesa_fixa` | `despesa` | custos fixos por categoria; base do ponto de equilíbrio |
| `resultado_financeiro` | `receita` ou `despesa` | juros, tarifas, rendimentos, encargos de antecipação |
| `fora_dre` | `receita` ou `despesa` | empréstimo (principal), aporte, distribuição de lucro |
| `entre_contas` | `transferencia` | transferência entre contas próprias |

`chk_categoria_financeira_natureza_linha` amarra as combinações da tabela.

**Constraints e índices:** `chk_categoria_financeira_nome`, `chk_categoria_financeira_natureza`,
`chk_categoria_financeira_linha_dre`, `chk_categoria_financeira_codigo`,
`chk_categoria_financeira_ordem`, `chk_categoria_financeira_versao`,
`chk_categoria_financeira_pai` (não aponta para si mesma), `uq_categoria_financeira_classificacao`
em `(id, natureza, linha_dre)` e `fk_categoria_financeira_pai` **composta**
`(categoria_pai_id, natureza, linha_dre)` → `(id, natureza, linha_dre)`: o próprio banco garante
que a subcategoria tem a mesma classificação do pai; `uq_categoria_financeira_raiz_nome` em `lower(nome)` parcial
(`WHERE categoria_pai_id IS NULL`), `uq_categoria_financeira_sub_nome` em
`(categoria_pai_id, lower(nome))` parcial, `uq_categoria_financeira_codigo` parcial
(`WHERE codigo IS NOT NULL`), `idx_categoria_financeira_pai (categoria_pai_id, ordem, nome, id)`.

**Regras:**
- Raiz: `natureza` e `linha_dre` obrigatórios no corpo, combinação validada no domínio.
- Subcategoria: `natureza` e `linha_dre` **não** vêm no corpo (informados → `400 subcategoria
  herda natureza e linha da DRE da categoria pai`); o `INSERT` copia os valores do pai na mesma
  instrução e só grava se o pai for raiz e estiver ativo (`400 categoria pai deve ser uma
  categoria raiz ativa`).
- **Lançável** = subcategoria ativa. Raiz nunca recebe lançamento, mesmo sem filhas: assim
  acrescentar uma filha depois não deixa lançamentos pendurados num nível intermediário.
- Desativar raiz só sem filhas **ativas** (`409 categoria possui subcategorias ativas e não pode
  ser desativada`), conferido no próprio `UPDATE` (`AND NOT EXISTS (...)`).
- Ativar subcategoria exige o pai ativo (`400 categoria pai está inativa`), também no `UPDATE`.
- Alteráveis: `nome`, `codigo`, `ordem`. Natureza e linha DRE erradas = excluir (sem vínculos)
  ou desativar e criar outra; mudar a classificação reescreveria a DRE de meses fechados.
- Duplicidade: nome repetido no mesmo nível → `409 já existe uma categoria com este nome neste
  nível`; código repetido → `409 já existe uma categoria com este código`.

### 2.3 Catálogo de bancos (domínio, sem tabela)

| Código | Nome |
|---|---|
| `001` | Banco do Brasil |
| `033` | Santander |
| `077` | Inter |
| `237` | Bradesco |
| `341` | Itaú |
| `403` | Cora |
| `748` | Sicredi |

Banco fora do catálogo → `400 banco não suportado`. Acrescentar um banco = entrada no catálogo do
domínio + migration que amplia o `chk_conta_bancaria_banco`.

---

## 3. Versão otimista

Mesmo contrato da seção 3 do `FORNECEDORES.md`, aplicado às duas entidades:

- Toda resposta traz `versao`.
- `PUT`, `PATCH .../ativar` e `PATCH .../desativar` exigem `versao` no corpo (inteiro ≥ 1);
  ausente ou inválida → `400 versão da conta bancária é obrigatória` /
  `400 versão da categoria financeira é obrigatória`.
- Gravação com `UPDATE ... SET ..., versao = versao + 1 WHERE id = $1 AND versao = $2`. Zero
  linhas afetadas → `409 a conta bancária foi alterada por outra pessoa; recarregue e tente de
  novo` (ou `a categoria financeira ...`) se o registro existe, `404` se não.
- A versão divergente é conferida também no usecase, antes da regra de negócio, para a mensagem
  ser a mesma com ou sem corrida.
- `ativar` no que já está ativo (e `desativar` no inativo) responde `200` sem avançar a versão.
- Quando a guarda atômica de outra linha (empresa ativa, pai ativo, filhas ativas) zera o
  `UPDATE`, o repositório distingue o motivo relendo o registro, e a resposta é a da regra
  (`400`/`409` das seções 2.1 e 2.2), não a de versão.

---

## 4. Endpoints

### Contas bancárias
```
POST   /api/v1/contas-bancarias                 criar (financeiro)
GET    /api/v1/contas-bancarias                 ?empresa= ?banco= ?tipo= ?ativa= + paginação (financeiro)
GET    /api/v1/contas-bancarias/{id}            detalhe, se a empresa estiver no acesso (financeiro)
PUT    /api/v1/contas-bancarias/{id}            { "apelido", "versao" } (financeiro)
PATCH  /api/v1/contas-bancarias/{id}/ativar     { "versao": n } (financeiro)
PATCH  /api/v1/contas-bancarias/{id}/desativar  { "versao": n } (financeiro)
DELETE /api/v1/contas-bancarias/{id}            só sem vínculos (administrador)
GET    /api/v1/bancos                           catálogo da seção 2.3 (autenticado)
```

- Listagem filtrada pelo acesso (`EmpresaIDs`), ordenada por `lower(apelido), id`;
  `?empresa=` fora do acesso → `403`.
- Resposta traz `banco_nome`, `agencia_formatada` (`1234-5`) e `conta_formatada` (`12345-6`).

Exemplo de corpo do `POST`:
```json
{
  "empresa_id": "…",
  "banco": "748",
  "tipo": "corrente",
  "agencia": "0812",
  "conta": "54321-0",
  "apelido": "Sicredi movimento"
}
```

### Plano de contas
```
POST   /api/v1/categorias-financeiras                criar raiz ou subcategoria (financeiro)
GET    /api/v1/categorias-financeiras                ?pai= ?raiz=true ?natureza= ?linha_dre= ?ativa= ?busca= + paginação (autenticado)
GET    /api/v1/categorias-financeiras/arvore         plano inteiro em dois níveis, sem paginação; ?ativas=true (autenticado)
GET    /api/v1/categorias-financeiras/{id}           detalhe (autenticado)
PUT    /api/v1/categorias-financeiras/{id}           { "nome", "codigo", "ordem", "versao" } (financeiro)
PATCH  /api/v1/categorias-financeiras/{id}/ativar    { "versao": n } (financeiro)
PATCH  /api/v1/categorias-financeiras/{id}/desativar { "versao": n } (financeiro)
DELETE /api/v1/categorias-financeiras/{id}           só sem filhas e sem vínculos (administrador)
```

- O caminho é `categorias-financeiras` porque `/api/v1/categorias` já é do estoque.
- `busca` procura no nome (sem diferenciar maiúsculas) e no prefixo do `codigo`.
- Listagem ordenada por `ordem, lower(nome), id`.
- A `arvore` existe porque o plano é a lista de seleção do M7 e do M13 e passaria do máximo de
  100 itens da paginação (item "Listas cortadas em 100" da F0). Raízes por `ordem, nome, id`,
  cada uma com as filhas na mesma ordem, e `lancavel` em cada item.
- Resposta traz `lancavel` (subcategoria ativa).

Exemplo de corpo do `POST` (raiz e subcategoria):
```json
{ "nome": "Despesas administrativas", "natureza": "despesa", "linha_dre": "despesa_fixa", "codigo": "4.2", "ordem": 3 }
{ "nome": "Aluguel", "categoria_pai_id": "…", "codigo": "4.2.01" }
```

`DELETE` das duas entidades passa por `tratarErro`: filha ou, depois, lançamento do M7/M9/M13
apontando para o registro → `400 registro em uso`.

---

## 5. Permissões

- **Perfil financeiro**: criado pelo M4 (fase 3 do `FORNECEDORES.md`: `TemAcessoFinanceiro`,
  `ExigirFinanceiro`, wrapper `financeiro(...)`, campo `financeiro` no login). O M5 só o usa.
  Se a fase 4 do M5 chegar antes de o perfil estar na `main`, as rotas de financeiro entram com
  `protegido(...)` (o usecase já barra quem não é financeiro) e trocam para `financeiro(...)`
  num commit próprio assim que a `main` tiver o wrapper.
- Rotas: `financeiro(...)` para toda a escrita e para a leitura de contas bancárias;
  `protegido(...)` para a leitura do plano de contas e do catálogo de bancos;
  `administrador(...)` para excluir.
- Regras finas em `usecases/financeiro/permissoes.go`: a escrita e a leitura de contas conferem o
  perfil também no usecase (financeiro **ou** administrador), com portas mínimas
  `UsuarioRepository` e `CargoRepository` (só `Obter`).
- **Escopo por empresa** só nas contas bancárias, pelos itens 1 a 6 da seção 4 do `EMPRESAS.md`:
  porta mínima `EmpresaRepository` (`Obter` + `AcessoDoUsuario`); criar, ler, alterar, ativar,
  desativar e excluir conta de empresa fora do acesso → `403`; listagem com `EmpresaIDs`, lista
  vazia devolve vazio; teste de isolamento no usecase e no e2e.
- Plano de contas sem escopo por empresa (decisão da seção 1).

---

## 6. Contrato com os módulos seguintes

Cada consumidor declara a própria porta mínima (regra 6 da seção 3 do `ROADMAP.md`), satisfeita
no `main.go` pelos repositórios deste módulo:

- `ContaBancaria` expõe `EmpresaID`, `Banco`, `Agencia`, `Conta`, `Ativa`. Regra dos
  consumidores: conta ativa e **da mesma empresa** do lançamento.
- `CategoriaFinanceira` expõe `Natureza`, `LinhaDRE`, `EhSubcategoria()`, `Ativa` e
  `Lancavel()`. Regras dos consumidores: só categoria lançável; M7 aceita `despesa`, M9 aceita
  `receita`, M13 aceita as três (inclusive `transferencia` para `entre_contas`).
- O M15 agrupa pela `linha_dre` e ignora `compra_mercadoria` (o CMV vem dos lotes), `fora_dre`
  (apresentado à parte) e `entre_contas`.
- O regime da DRE (competência ou caixa) é decisão do M7/M15, não do M5. A transferência entre
  contas de empresas diferentes do grupo depende da decisão aberta "Transferências entre
  empresas" (seção 6 do `ROADMAP.md`) e é tratada no M13.

---

## 7. Estrutura de código

```
migrations/000038_create_tabela_categoria_financeira
migrations/000039_create_tabela_conta_bancaria

internal/core/domain/financeiro/      banco.go (catálogo), tipo_conta.go, numero_bancario.go
                                      (agência, conta e dígitos), conta_bancaria.go, natureza.go,
                                      linha_dre.go, categoria_financeira.go (+ testes)
internal/core/ports/out/financeiro/   conta_bancaria_repository.go, categoria_financeira_repository.go,
                                      empresa_repository.go, usuario_repository.go, cargo_repository.go
internal/core/ports/in/financeiro/    conta_bancaria_usecase.go, categoria_financeira_usecase.go
internal/core/usecases/financeiro/    conta_bancaria_usecase.go, categoria_financeira_usecase.go,
                                      permissoes.go (+ testes, cenário e fakes)
internal/adapter/postgres/            conta_bancaria_repo.go, categoria_financeira_repo.go
internal/adapter/http/dto/financeiro/ + handlers/financeiro/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers/servidor.go + test/integration + test/e2e
```

- O domínio importa só `domain` (erros); a empresa entra pelo usecase, por porta mínima.
- Numeração: o M4 usa `000037`; o M5 renumera antes do merge se a `main` já tiver avançado
  (regra 4 da seção 3 do `ROADMAP.md`).

---

## 8. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000038–000039 + domínio (catálogo de bancos, tipo, números bancários, conta, natureza, linha DRE, categoria com herança e versão) + testes | concluída |
| **2** | Repositórios com guarda de versão e guardas atômicas + testes de integração (unicidade com zeros à esquerda, herança no `INSERT`, desativação com filhas ativas, empresa inativa, filtros, árvore, corrida de versão) | pendente |
| **3** | Usecases + permissões (financeiro, administrador, escopo por empresa) + testes com fakes, inclusive isolamento entre empresas | pendente |
| **4** | Handlers/DTOs + rotas + e2e + `API.md` (usa o perfil financeiro do M4; ver seção 5) | pendente |
| **5** | Frontend: contas bancárias por empresa e plano de contas em árvore | pendente |

O M5 não tem dinheiro nem data de fato (o saldo inicial fica no M13), então a fase 5 não depende
da F0.

---

## 9. Decisões fechadas

- [x] **Conta bancária por empresa**; plano de contas do grupo.
- [x] **`linha_dre`** no lugar de `grupo_dre`, com `compra_mercadoria` fora do resultado.
- [x] **Natureza com `transferencia`**, amarrada à linha DRE por `CHECK`.
- [x] **Herança na subcategoria** e lançamento só em subcategoria ativa.
- [x] **Banco por código COMPE** com catálogo fechado; sem conta do tipo "caixa".
- [x] **Agência e conta só com dígitos**, sem conferir o dígito verificador; unicidade no grupo.
- [x] **Imutabilidade**: na conta só o apelido muda; na categoria, natureza, linha DRE e pai
      não mudam.
- [x] **Leitura de contas só para financeiro e administrador**, com escopo por empresa.
- [x] **Exclusão só pelo administrador**.
- [x] **Perfil financeiro do M4**, sem duplicar middleware.
- [x] **Versão otimista** no contrato do M4.
- [x] **Saldo inicial no M13**.
- [x] **Sem seed**.

## 10. Decisões pendentes

Sem abertos.
