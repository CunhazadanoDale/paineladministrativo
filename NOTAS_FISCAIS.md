# Escopo — M8 NF-e por XML

> **Status:** API concluída (fases 1 a 6, com os títulos no M9); front (fase 7) pendente, depois do F0a.
> **Objetivo:** tirar o "Controle de NFs" da planilha: a NF-e emitida pelas empresas do grupo
> entra no sistema pelo XML autorizado, com itens, duplicatas e eventos, ligada ao cliente, ao
> pedido e à entrega. Base para os títulos do M9 (fase final) e para a receita do M15.
> **Nome do módulo:** `notafiscal` (rotas `/api/v1/notas-fiscais`)
> **Branch:** `feat/m8-notas-fiscais`

---

## 1. Visão geral

O sistema **não emite** NF-e: a nota continua sendo emitida no emissor atual e o XML autorizado
(`nfeProc`, com o protocolo da SEFAZ) é enviado ao painel. O M8 lê o XML, confere a chave e o
protocolo, resolve a empresa emitente, o cliente e os produtos e grava o documento fiscal como
ele é. Valor da nota é o do documento; diferença para o pedido é assunto da contabilidade
(ROADMAP §5, fora de escopo deliberado).

Decisões que já vêm do ROADMAP:

| Tema | Decisão |
|---|---|
| Entrada do XML | upload pelo fluxo de `arquivos` (`POST /api/v1/arquivos` passa a aceitar XML) e importação por `arquivo_id`; o XML original fica guardado e vinculado à nota |
| Chave repetida | `409 já existe uma nota fiscal com esta chave de acesso` |
| Empresa emitente | resolvida pelo CNPJ do emitente (`EmpresaRepository.ObterPorCNPJ`); emitente fora do cadastro → `400`; empresa fora do acesso de quem importa → `403` |
| Cliente | resolvido pelo documento do destinatário (`ClienteRepository.ObterPorDocumento`); criado se não existir |
| Produto | resolvido pelo `codigo` do produto (`cProd` do item) |
| Entre empresas do grupo | regra provisória do ROADMAP §6: destinatário com CNPJ de empresa cadastrada → nota marcada como **entre empresas do grupo**, sem título, conta a pagar nem movimento de estoque; o M15 a deixa fora da receita. O efeito definitivo vira fase própria quando a contabilidade responder |
| Títulos | as fases 1 a 5 gravam `nota_fiscal_duplicata` **sem gerar título**; a geração no M9 (origem `nota_fiscal_duplicata`) e o efeito do cancelamento nos títulos são a **fase 6**, que só começa com o M9 na `main` e esta branch atualizada, pela assinatura fechada no `CONTAS_A_RECEBER.md` |
| Pedido | vínculo NF ↔ pedido ↔ entrega por **porta mínima própria** (`ports/out/notafiscal/`), com o `Obter` que o `PedidoRepository` e o `EntregaRepository` já têm; nenhum método novo no port do M6 |
| Estoque | a NF **não move estoque**: a saída da venda já acontece na entrega do M6 (`saiu`) |

---

## 2. Entidades

### 2.1 `nota_fiscal`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `chave` | CHAR(44) | `uq_nota_fiscal_chave`; validada no domínio (seção 3.1) |
| `empresa_id` | UUID | FK `empresa` `RESTRICT`; emitente |
| `numero` | INT | `nNF`, `1..999999999` |
| `serie` | INT | `0..999` |
| `emitida_em` | TIMESTAMPTZ | `dhEmi` |
| `tipo_operacao` | VARCHAR(7) | `saida` / `entrada` (`tpNF`) |
| `finalidade` | VARCHAR(12) | `normal`, `complementar`, `ajuste`, `devolucao` (`finNFe`) |
| `natureza` | VARCHAR(13) | `venda`, `remessa`, `transferencia`, `devolucao` (seção 3.3) |
| `natureza_operacao` | VARCHAR(60) | `natOp`, texto do emissor |
| `entre_empresas` | BOOLEAN | destinatário é empresa cadastrada (regra provisória) |
| `cliente_id` | UUID NULL | FK `cliente`; `NULL` quando `entre_empresas` |
| `empresa_destinataria_id` | UUID NULL | FK `empresa`; preenchido só quando `entre_empresas` |
| `destinatario_documento` | VARCHAR(14) | CPF/CNPJ do destinatário como veio no XML |
| `destinatario_nome` | VARCHAR(200) | `xNome` do destinatário, congelado |
| `cfop_predominante` | CHAR(4) | CFOP do item de maior `valor_centavos` (desempate pelo menor `numero_item`) |
| `valor_total_centavos` | BIGINT | `vNF`, `>= 0` |
| `valor_produtos_centavos` | BIGINT | `vProd` do total |
| `valor_frete_centavos` / `valor_desconto_centavos` | BIGINT | `vFrete` / `vDesc` |
| `numero_pedido_cliente` | VARCHAR(60) NULL | `compra/xPed` (ou o `xPed` do primeiro item que tiver) |
| `protocolo` | VARCHAR(15) | `nProt` da autorização |
| `autorizada_em` | TIMESTAMPTZ | `dhRecbto` da autorização |
| `status` | VARCHAR(10) | `autorizada` → `cancelada` |
| `pedido_id` | UUID NULL | FK `pedido` |
| `entrega_id` | UUID NULL | FK `entrega`; exige `pedido_id` e pertence a ele |
| `arquivo_id` | UUID | FK `arquivo` `RESTRICT`; o XML original, único |
| `usuario_id` | UUID | quem importou |
| `versao` | INT | versão otimista (vínculos e eventos) |
| `criado_em` / `atualizado_em` | TIMESTAMPTZ | |

Índices: `idx_nota_fiscal_empresa_emissao (empresa_id, emitida_em DESC, id)`,
`idx_nota_fiscal_cliente`, `idx_nota_fiscal_pedido`, `uq_nota_fiscal_entrega_vigente (entrega_id)
WHERE status = 'autorizada'` (uma entrega tem no máximo uma NF autorizada).
`chk_nota_fiscal_destinatario`: `entre_empresas` ⇔ `empresa_destinataria_id IS NOT NULL` ⇔
`cliente_id IS NULL`.

### 2.2 `nota_fiscal_item`

| Campo | Tipo | Observações |
|---|---|---|
| `nota_fiscal_id` / `numero_item` | UUID / INT | PK composta; `nItem` |
| `codigo` | VARCHAR(60) | `cProd` |
| `descricao` | VARCHAR(120) | `xProd` |
| `produto_id` | UUID NULL | FK `produto`; resolvido pelo `codigo`; `NULL` = item pendente de produto, vinculável depois |
| `ncm` | CHAR(8) | |
| `cfop` | CHAR(4) | |
| `unidade` | VARCHAR(6) | `uCom` como veio (não é o enum do produto) |
| `quantidade_decimilesimos` | BIGINT | `qCom` × 10⁴ (o XML tem até 4 casas) |
| `valor_unitario` | VARCHAR(22) | `vUnCom` como veio (até 10 casas; só exibição) |
| `valor_centavos` | BIGINT | `vProd` do item |
| `desconto_centavos` | BIGINT | `vDesc` do item, `0` se ausente |
| `pedido_cliente` | VARCHAR(60) NULL | `xPed` do item |

### 2.3 `nota_fiscal_duplicata`

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | é o `origem_id` do título no M9 (fase 6) |
| `nota_fiscal_id` | UUID | FK `nota_fiscal` |
| `numero` | VARCHAR(60) | `nDup`; único por nota |
| `vencimento` | DATE | `dVenc` |
| `valor_centavos` | BIGINT | `vDup`, `> 0` |

Gravada na importação, inclusive em nota entre empresas (sem título, pela regra provisória).

### 2.4 `nota_fiscal_evento`

Append-only, como `solicitacao_historico`.

| Campo | Tipo | Observações |
|---|---|---|
| `id` | UUID | |
| `nota_fiscal_id` | UUID | FK `nota_fiscal` |
| `tipo` | VARCHAR(16) | `cancelamento` (`110111`), `carta_correcao` (`110110`) |
| `sequencia` | INT | `nSeqEvento`; `uq_nota_fiscal_evento (nota_fiscal_id, tipo, sequencia)` |
| `ocorrido_em` | TIMESTAMPTZ | `dhEvento` |
| `protocolo` | VARCHAR(15) | `nProt` do retorno do evento |
| `justificativa` | VARCHAR(1000) | `xJust` (cancelamento) ou `xCorrecao` (CC-e) |
| `arquivo_id` | UUID | FK `arquivo`; o XML do evento, único |
| `usuario_id` | UUID | |
| `criado_em` | TIMESTAMPTZ | |

---

## 3. Regras

### 3.1 Validação do XML

- Raiz `nfeProc` com `NFe/infNFe` e `protNFe/infProt`; `NFe` sem protocolo → `400 XML sem
  protocolo de autorização`. Parser `encoding/xml` da biblioteca padrão (não resolve entidades
  externas); limite de 10 MB do fluxo de arquivos.
- Modelo `55` (NFC-e `65` → `400`); `infProt/cStat` `100` ou `150` (autorizada), senão `400`;
  ambiente de produção (`tpAmb = 1`) na nota, no protocolo e no evento: homologação → `400`.
- Chave: 44 caracteres, `Id` do `infNFe` = `NFe` + chave = `infProt/chNFe`; dígito verificador
  módulo 11; campos embutidos coerentes com o XML (UF, ano e mês da emissão, CNPJ do emitente,
  modelo, série, número). O CNPJ da chave aceita o formato **alfanumérico** (como em `empresa`):
  cada caractere vale seu código ASCII − 48 no cálculo do dígito.
- Destinatário com `idEstrangeiro` → `400` (o cliente exige CPF/CNPJ).
- O arquivo precisa ser do usuário que importa (`403`), ter tipo XML (`400`), não estar vinculado a
  solicitação (`400`) nem a outra nota ou evento (`409`).
- Empresa emitente inativa → `400` (padrão do `EMPRESAS.md`).

### 3.2 Cliente

- Destinatário com documento de empresa cadastrada (ativa ou não) → `entre_empresas = true`,
  sem cliente (regra provisória).
- Senão, `ObterPorDocumento`; se não existir, cria pelo domínio do M3: documento, `nome` =
  `xNome`, `uf`/`cidade` do `enderDest`, `email_nf` = `email` (se válido), `tipo` = `pf` com CPF
  e `outro` com CNPJ (o comercial reclassifica), sem representante. Cliente inativo é aceito (a
  nota já foi emitida) e não é reativado.

### 3.3 Natureza

1. `entre_empresas` → `transferencia`;
2. `finNFe = 4` (devolução) → `devolucao`;
3. `cfop_predominante` de remessa → `remessa`: `5116`, `5117`, `6116`, `6117` (remessa da venda
   para entrega futura), `5905`, `6905` (depósito), `5910` a `5917` e `6910` a `6917`
   (bonificação, amostra, demonstração, consignação, conserto), `5923`, `6923` (conta e ordem),
   `5934`, `6934` e `5949`, `6949` (outras saídas). A lista vive no domínio (`natureza.go`);
4. demais → `venda`, inclusive o simples faturamento da entrega futura (`5922`, `6922`).

Só `venda` entra como receita no M15 e gera título na fase 6; `remessa`, `transferencia` e
`devolucao` guardam as duplicatas, se houver, sem título.

### 3.3.1 Itens sem produto

Item cujo `codigo` não bate com nenhum `produto.codigo` é gravado com `produto_id NULL` e
aparece em `GET /notas-fiscais/itens-sem-produto`. O financeiro liga depois
(`PUT /notas-fiscais/{id}/itens/{numero}/produto`); a nota nunca é recusada por isso.

### 3.3.2 Devolução

Nota de devolução (`finNFe = 4`) só é registrada: o retorno ao estoque e o acerto de títulos
ficam manuais nesta versão.

### 3.4 Vínculo com pedido e entrega

- Opcional na importação (`pedido_id`, `entrega_id`) e alterável depois
  (`PUT /notas-fiscais/{id}/vinculo`, com `versao`), enquanto a nota estiver autorizada.
- Pedido da mesma empresa e do mesmo cliente da nota, fora de `proposta` e `cancelado`; entrega
  do pedido e não cancelada; uma entrega com no máximo uma NF autorizada (`409`).
- Nota entre empresas não se vincula a pedido (`400`).
- Cliente com `exige_numero_pedido_na_nf` e nota sem `xPed` → a importação grava e devolve o
  aviso `numero_pedido_ausente` (a nota já está autorizada; recusar não a desfaz).

### 3.5 Eventos

- XML de evento (`procEventoNFe`) importado pelo mesmo fluxo: `POST /notas-fiscais/eventos` com
  `arquivo_id`. A nota é achada pela `chNFe`; `cStat` `135`, `136` ou `155` exigido; o autor do
  evento precisa ser o CNPJ emitente da chave.
- Cancelamento: nota `autorizada` → `cancelada` com guarda de status e versão; libera o vínculo
  da entrega (o índice parcial deixa outra NF ser ligada). Nas fases 1 a 5 não há título a
  tratar; a fase 6 aplica o efeito no M9.
- Carta de correção: só registra o evento (texto livre da SEFAZ); não altera a nota.

### 3.6 Datas e dinheiro

- `emitida_em`, `autorizada_em` e `ocorrido_em` vêm do XML com fuso; consultas por dia usam o
  fuso de Brasília (`domainpedido.Hoje` como referência).
- Valores do XML (`0.00`) convertidos para centavos sem `float` (parser de decimal com 2 casas;
  mais casas → `400`, salvo `vUnCom`, guardado como texto).

---

## 4. Endpoints

```
POST   /api/v1/notas-fiscais/importacao        { arquivo_id, pedido_id?, entrega_id? } → nota + avisos
GET    /api/v1/notas-fiscais                    ?empresa= ?cliente= ?pedido= ?natureza= ?status= ?entre_empresas= ?sem_pedido=true ?emitida_de= ?emitida_ate= ?numero= ?chave=
GET    /api/v1/notas-fiscais/{id}               detalhe com itens, duplicatas e eventos
GET    /api/v1/notas-fiscais/{id}/xml           XML original (nosniff, prazo de arquivo)
PUT    /api/v1/notas-fiscais/{id}/vinculo       { versao, pedido_id, entrega_id } (null desfaz)
POST   /api/v1/notas-fiscais/eventos            { arquivo_id } → evento + nota
GET    /api/v1/notas-fiscais/{id}/eventos
GET    /api/v1/notas-fiscais/itens-sem-produto  itens com produto_id NULL; ?empresa=
PUT    /api/v1/notas-fiscais/{id}/itens/{numero}/produto  { versao, produto_id }
GET    /api/v1/entregas/sem-nota-fiscal         entregas `saiu`/`entregue` sem NF autorizada; ?empresa= ?cliente=
POST   /api/v1/notas-fiscais/{id}/titulos       sincroniza os títulos a receber (gera os que faltam ou cancela)
```

`POST /api/v1/arquivos` passa a aceitar `application/xml` e `text/xml` (gravado como
`application/xml`).

---

## 5. Permissões

- Importar nota e evento, alterar vínculo e ligar item a produto: `financeiro(...)` (financeiro
  ou administrador); leitura `protegido(...)`.
- **Escopo por empresa** (seção 4 do `EMPRESAS.md`) pela empresa emitente: importar exige a
  emitente no acesso (`403`); ler, listar e baixar XML filtram pelo acesso; teste de isolamento
  no usecase e no e2e.
- **Representante com login** (não administrador): vê só notas de clientes da própria carteira;
  não importa nem vincula. Nota entre empresas não aparece para ele.

---

## 6. Estrutura de código

```
migrations/000051_create_tabela_nota_fiscal          nota_fiscal, nota_fiscal_item, nota_fiscal_duplicata
migrations/000052_create_tabela_nota_fiscal_evento

internal/core/domain/notafiscal/      chave.go, decimal.go (centavos e quantidade sem float),
                                      natureza.go, finalidade.go (+ tipo de operação), status.go
                                      (+ tipo de evento), documento.go (o que o XML diz),
                                      xml.go (leitura do nfeProc e do procEventoNFe), nota_fiscal.go
                                      (nota, item, duplicata, vínculos), evento.go
internal/core/ports/out/notafiscal/   nota_fiscal_repository.go, arquivo_repository.go, storage.go,
                                      empresa_repository.go (ObterPorCNPJ, AcessoDoUsuario),
                                      cliente_repository.go (ObterPorDocumento, Criar),
                                      produto_repository.go (Obter, IDsPorCodigos),
                                      pedido_repository.go (Obter), entrega_repository.go (Obter),
                                      representante_repository.go, usuario_repository.go, cargo_repository.go,
                                      titulo_receber_repository.go (GerarDeOrigem, CancelarPorOrigem do M9)
internal/core/ports/in/notafiscal/    nota_fiscal_usecase.go, evento_usecase.go
internal/core/usecases/notafiscal/    nota_fiscal_usecase.go (importação, vínculos, consultas),
                                      destino.go (empresa do grupo ou cliente, criado se faltar),
                                      arquivo.go (leitura do XML), evento_usecase.go, permissoes.go
                                      (+ testes e fakes)
internal/adapter/postgres/            nota_fiscal_repo.go (inclui entregas sem NF e itens sem produto,
                                      sem tocar no entrega_repo.go do M6); IDsPorCodigos no fim de
                                      produto_repo.go
internal/core/usecases/solicitacao/   arquivo_usecase.go aceita XML
internal/adapter/http/dto/notafiscal/ + handlers/notafiscal/
cmd/api/main.go + internal/adapter/http/rotas.go   (registro, só acréscimos)
test/fixtures/nfe/                    gerador de XML de NF-e e de evento para usecase e e2e (chave com
                                      dígito calculado, sem dados reais); o domínio tem o próprio
                                      construtor_test.go
```
Numeração de migration definida no merge (regra 4 da seção 3 do `ROADMAP.md`): o M9 entrou primeiro na
`main` com `000047` a `000050`, e o M8 foi renumerado para `000051` e `000052`. Renumerar de novo se outro módulo entrar antes.
`000046` e o M9 também começa em `000047`; quem mergear depois renumera.

---

## 7. Fases de implementação

| Fase | Entrega | Estado |
|---|---|---|
| **1** | Migrations 000051–000052 (000047–000048 até o merge do M9) + domínio (chave, decimal, natureza, nota, item, duplicata, evento, leitura do XML) + testes | concluída |
| **2** | Repositório (gravação da nota com itens e duplicatas numa transação, eventos com guarda de status e versão, vínculo com pedido e entrega travados com `FOR SHARE`, listagens; `IDsPorCodigos` e entregas sem NF) + testes de integração | concluída |
| **3** | Usecases de importação, vínculo e eventos + XML no fluxo de arquivos + permissões (escopo por empresa, representante) + testes com fakes e de isolamento | concluída |
| **4** | Handlers/DTOs + rotas + e2e + `API.md` | concluída |
| **5** | Consultas (entregas sem NF, itens sem produto e vínculo de item) + testes + `API.md` | concluída (entregue junto com as fases 2 a 4) |
| **6** | Títulos no M9: duplicata → título (origem `nota_fiscal_duplicata`) na importação, cancelamento dos títulos com a NF, `travarNotaFiscalDeOrigem` no repositório do M9 e `POST /notas-fiscais/{id}/titulos`; migrations renumeradas para 000051–000052 | concluída |
| **7** | Frontend: importação, lista, detalhe, vínculo e entregas sem NF (depois do F0a) | pendente |

O efeito definitivo das notas entre empresas é uma fase futura, aberta quando a contabilidade
responder (ROADMAP §6).

---

## 8. Decisões fechadas

- [x] **Sistema não emite NF-e**: importa o XML autorizado.
- [x] **Entre empresas do grupo** pela regra provisória do ROADMAP §6.
- [x] **Duplicatas sem título** até a fase 6 (acordo M8 × M9).
- [x] **NF não move estoque**; a saída é da entrega do M6.
- [x] **Porta mínima própria** para pedido e entrega, sem método novo no port do M6.
- [x] **Valor do documento** gravado como está; sem "valor real" divergente.
- [x] **XML guardado** pelo fluxo de arquivos e vinculado à nota e ao evento.
- [x] **Só NF-e modelo 55** autorizada; NFC-e fora.
- [x] **CNPJ alfanumérico** aceito na chave.
- [x] **Quantidade em décimos de milésimo** (`BIGINT`) e dinheiro em centavos, sem `float`.
- [x] **D1 — Abrangência**: só NF emitida pelas empresas do grupo; NF de fornecedor fica para o
      M12/M7.
- [x] **D2 — Item sem produto**: a nota entra, o item fica pendente e é ligado depois.
- [x] **D3 — Quem escreve**: financeiro (e administrador).
- [x] **D4 — Autenticidade**: estrutura, chave e protocolo; sem verificar a assinatura XMLDSig.
- [x] **D5 — Remessas**: natureza `remessa` pelo CFOP predominante, fora da receita e sem título;
      simples faturamento (`5922`/`6922`) é `venda`.
- [x] **D6 — Devolução**: só registra.
- [x] **D7 — Vínculo com pedido**: manual, na importação ou depois.

---

## 9. Decisões pendentes

- [ ] **Carga histórica**: depende do item "Importação histórica" do ROADMAP §6; até lá, um XML
      por requisição, sem importação em lote.
- [ ] **Efeito definitivo das notas entre empresas**: espera a contabilidade (ROADMAP §6).

---

## 10. Regras de código

As do ROADMAP §1 e §2: sem comentários, value objects no padrão de `solicitacao/valor.go`,
erros só das sentinelas de `domain/errors.go`, nomes em português, testes de domínio e usecase
com fakes, integração em `test/integration`, e2e com `//go:build e2e`.
