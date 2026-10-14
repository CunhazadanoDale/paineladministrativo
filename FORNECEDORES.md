# Escopo — M4 Fornecedores

> **Status:** escopo aprovado — pronto para implementação.
> **Objetivo:** tirar a "coluna fornecedor do painel financeiro" do texto livre: cadastro único
> de fornecedores do grupo, nacionais e estrangeiros, base para contas a pagar (M7),
> importações (M12) e dívidas (M14).
> **Nome do módulo:** `fornecedor`
> **Branch:** `feat/m4-fornecedores`

---

## 1. Visão geral

Decisões já tomadas com o usuário (outubro/2026):

| Tema | Decisão |
|---|---|
| Escopo do fornecedor | **Do grupo**: cadastro único, sem `empresa_id`. A empresa entra na conta a pagar (M7) |
| Documento | **Reaproveita o M3**: `domaincliente.NovoDocumento` (CPF ou CNPJ numérico/alfanumérico) para fornecedor do Brasil; estrangeiro não tem documento brasileiro |
| Permissões | **Financeiro e administrador escrevem**; demais autenticados leem. O M4 cria o perfil `financeiro` nas rotas, que o M5 reaproveita |
| Moeda | **Lista fechada**: `BRL`, `USD`, `EUR`, `CNY`; ampliar = migration nova |
| Tamanho | **Só o CRUD**: sem dados de pagamento (pix, banco) e sem importação CSV. Dados de pagamento ficam como sugestão para o M7 |
| Concorrência | **Versão otimista** (`fornecedor.versao`): primeiro cadastro a resolver o item 10 do `PENDENCIAS.md`, modelo para os demais |
| UF e cidade | **Opcionais**, aceitas só para fornecedor do Brasil |
| Exclusão | **Só administrador**, como no M3 |
| País | **Validado só pelo formato** ISO 3166-1 alfa-2, sem lista fechada |

---

## 2. Entidade `fornecedor`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `pais` | CHAR(2) | ISO 3166-1 alfa-2 em maiúsculas (`^[A-Z]{2}$`), default `BR`; só o formato é validado |
| `documento` | VARCHAR(14) NULL | CPF ou CNPJ sem máscara; **obrigatório se `pais = 'BR'`, proibido caso contrário**; `uq_fornecedor_documento` parcial (`WHERE documento IS NOT NULL`) |
| `tipo_pessoa` | VARCHAR(2) NULL | `pf`/`pj`, derivado do documento (não vem no corpo); nulo para estrangeiro |
| `identificacao_estrangeira` | VARCHAR(40) NULL | tax id / registro do exterior, texto livre; só para `pais <> 'BR'`; único por país quando informado (`uq_fornecedor_identificacao_estrangeira` em `(pais, identificacao_estrangeira)`) |
| `nome` | VARCHAR(200) | razão social ou nome da pessoa; obrigatório |
| `nome_fantasia` | VARCHAR(120) NULL | |
| `nome_busca` | VARCHAR(200) | `domaincliente.NomeDeBusca(nome)`; indexado para busca e alerta de duplicidade |
| `moeda_padrao` | CHAR(3) | `BRL`, `USD`, `EUR`, `CNY`; default `BRL`; sugestão para M7/M12, não trava o lançamento |
| `uf` / `cidade` | CHAR(2) NULL / VARCHAR(120) NULL | UF do enum de `domainempresa.UF`; só para `pais = 'BR'` |
| `contato_nome` | VARCHAR(160) NULL | pessoa de contato |
| `email` | VARCHAR(255) NULL | validado como no M3 |
| `telefone` | VARCHAR(40) NULL | |
| `observacoes` | VARCHAR(2000) | default `''` |
| `ativo` | BOOLEAN | default `true` |
| `versao` | INT | começa em `1`; avança a cada alteração gravada |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

**Constraints:** `chk_fornecedor_pais`, `chk_fornecedor_documento` (formato por `tipo_pessoa`,
como em `cliente`), `chk_fornecedor_documento_pais` (`(pais = 'BR') = (documento IS NOT NULL)`),
`chk_fornecedor_tipo_pessoa` (`(documento IS NULL) = (tipo_pessoa IS NULL)`),
`chk_fornecedor_identificacao_estrangeira` (nula quando `pais = 'BR'`),
`chk_fornecedor_endereco_pais` (`uf` e `cidade` nulas quando `pais <> 'BR'`),
`chk_fornecedor_moeda_padrao`, `chk_fornecedor_uf`, `chk_fornecedor_nome`,
`chk_fornecedor_nome_busca`, `chk_fornecedor_email`, `chk_fornecedor_versao` (`versao >= 1`).
Índices: `idx_fornecedor_nome_busca (nome_busca, id)` e `idx_fornecedor_ativo (ativo, nome_busca, id)`.

**Regras:**
- Documento aceito com ou sem máscara, gravado sem máscara, devolvido nos dois formatos
  (`documento`, `documento_formatado`).
- **País e documento não mudam** depois de criado (corrigir = desativar e criar outro, como em
  empresa e cliente). `identificacao_estrangeira` pode ser alterada.
- Duplicidade: documento repetido → `409 já existe um fornecedor com este documento`;
  identificação estrangeira repetida no mesmo país → `409 já existe um fornecedor com esta
  identificação neste país`. Nome parecido não bloqueia; a busca por `nome_busca` serve para
  achar o cadastro antes de criar outro.
- Fornecedor inativo continua no histórico e não recebe lançamento novo (regra aplicada pelo
  M7, M12 e M14 via porta mínima).

---

## 3. Versão otimista

Primeiro cadastro do sistema com controle de versão; o padrão abaixo é o modelo para resolver o
item 10 do `PENDENCIAS.md` nos outros cadastros.

- Toda resposta de fornecedor traz `versao`.
- `PUT`, `PATCH .../ativar` e `PATCH .../desativar` exigem `versao` no corpo (inteiro ≥ 1);
  ausente ou inválida → `400 versão do fornecedor é obrigatória`.
- O repositório grava com
  `UPDATE ... SET ..., versao = versao + 1 WHERE id = $1 AND versao = $2`
  (mesma ideia do `versao_estoque` em `postgres/estoque_repo.go`). Zero linhas afetadas →
  `409 o fornecedor foi alterado por outra pessoa; recarregue e tente de novo` se o registro
  existe, `404` se não (padrão "Transição de estado" da seção 2 do `ROADMAP.md`).
- Versão divergente é conferida também no usecase, antes de aplicar a regra de negócio, para que
  a mensagem seja a mesma com ou sem corrida.
- `ativar` em fornecedor já ativo (e `desativar` em já inativo) responde `200` com o registro sem
  avançar a versão.

---

## 4. Endpoints

```
POST   /api/v1/fornecedores                       criar (financeiro)
GET    /api/v1/fornecedores                       ?busca= ?pais= ?moeda= ?ativo= + paginação (autenticado)
GET    /api/v1/fornecedores/{id}                  detalhe (autenticado)
PUT    /api/v1/fornecedores/{id}                  alterar, sem país e documento; exige versao (financeiro)
PATCH  /api/v1/fornecedores/{id}/ativar           { "versao": n } (financeiro)
PATCH  /api/v1/fornecedores/{id}/desativar        { "versao": n } (financeiro)
DELETE /api/v1/fornecedores/{id}                  só sem vínculos (administrador)
```

- `busca` procura em `nome_busca` (termo normalizado com `NomeDeBusca`), em `documento` (termo
  sem máscara, `domaincliente.TermoDeDocumento`) e em `identificacao_estrangeira`.
- Listagem ordenada por `nome_busca, id`.
- `DELETE` passa por `tratarErro`: hoje nada aponta para fornecedor; quando M7, M12 e M14
  criarem as FKs `RESTRICT`, fornecedor usado → `400 registro em uso`.

Exemplo de corpo do `POST`:
```json
{
  "pais": "CN",
  "identificacao_estrangeira": "91330200MA2XXXXX",
  "nome": "Foshan Ceramics Co., Ltd.",
  "moeda_padrao": "USD",
  "contato_nome": "Li Wei",
  "email": "sales@exemplo.cn",
  "observacoes": ""
}
```

---

## 5. Permissões

- **Perfil financeiro nas rotas** (receita da seção 2 do `ROADMAP.md`; a flag `cargo.financeiro`
  já existe):
  - `UsuarioUseCase.TemAcessoFinanceiro` (financeiro **ou** administrador, cumulativo como no
    módulo de solicitações);
  - `middleware.ExigirFinanceiro`;
  - wrapper `financeiro(...)` em `rotas.go`, ao lado de `protegido`/`administrador`/`comercial`;
  - campo `financeiro` na resposta do login (`POST /api/v1/usuarios/autenticar`), como
    `administrador` e `comercial` — acréscimo, não quebra contrato; destrava o item
    "Permissões fail-open" da F0.
- Rotas: `financeiro(...)` para criar, alterar, ativar e desativar; `administrador(...)` para
  excluir; `protegido(...)` para leitura.
- Regras finas em `usecases/fornecedor/permissoes.go` (a escrita confere o perfil também no
  usecase, como em `empresa` e `cliente`), com portas mínimas `UsuarioRepository` e
  `CargoRepository` (só `Obter`).
- Sem escopo por empresa (decisão da seção 1).

---

## 6. Estrutura de código

```
migrations/000037_create_tabela_fornecedor

internal/core/domain/fornecedor/      fornecedor.go (entidade, Alterar, Ativar, Desativar, versão),
                                      pais.go, moeda.go, texto.go (textos e e-mail)
internal/core/ports/out/fornecedor/   fornecedor_repository.go, usuario_repository.go,
                                      cargo_repository.go
internal/core/ports/in/fornecedor/    fornecedor_usecase.go
internal/core/usecases/fornecedor/    fornecedor_usecase.go, permissoes.go (+ testes e fakes)
internal/core/usecases/usuarios/      TemAcessoFinanceiro (+ teste)
internal/adapter/postgres/            fornecedor_repo.go
internal/adapter/http/middleware/     ExigirFinanceiro (+ teste)
internal/adapter/http/dto/fornecedor/ + handlers/fornecedor/
internal/adapter/http/dto/usuarios/   campo financeiro na resposta do login
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/helpers/servidor.go + test/integration + test/e2e
```

- O domínio importa `domaincliente` (documento e nome de busca) e `domainempresa` (UF), no mesmo
  sentido em que `cliente` já importa `empresa`.
- `Moeda` fica em `domain/fornecedor/moeda.go`; o M12 a reaproveita importando
  `domainfornecedor`.

---

## 7. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migration 000037 + domínio (país, moeda, regras de documento por país, versão) + testes | pendente |
| **2** | Repositório com guarda de versão + testes de integração (unicidade, busca, filtros, corrida de versão) | pendente |
| **3** | Perfil financeiro (`TemAcessoFinanceiro`, `ExigirFinanceiro`, campo no login) + testes | pendente |
| **4** | Usecase + permissões + testes com fakes | pendente |
| **5** | Handlers/DTOs + rotas + e2e + `API.md` | pendente |
| **6** | Frontend: lista, cadastro e edição de fornecedores | pendente |

O M4 não tem dinheiro nem data de fato (moeda é só um código), então a fase 6 não depende da
F0.

---

## 8. Decisões fechadas

- [x] **Fornecedor do grupo**, sem escopo por empresa.
- [x] **Documento pelo M3** (`domaincliente.NovoDocumento`), obrigatório no Brasil e proibido
      fora dele; estrangeiro usa `identificacao_estrangeira`.
- [x] **Financeiro e administrador escrevem**; o M4 cria o perfil financeiro nas rotas.
- [x] **Moeda em lista fechada** (`BRL`, `USD`, `EUR`, `CNY`).
- [x] **Só o CRUD**: sem dados de pagamento e sem importação CSV.
- [x] **Versão otimista** em `fornecedor.versao`.
- [x] **UF e cidade opcionais**, só para fornecedor do Brasil.

- [x] **Exclusão só pelo administrador**, como no M3.
- [x] **País validado só pelo formato** `^[A-Z]{2}$`, sem a lista completa da ISO 3166.

## 9. Decisões pendentes

Sem abertos.
