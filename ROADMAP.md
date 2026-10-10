# Roadmap — Painel do Grupo (MIA / Red Diamond / Gigante)

Plano de evolução do backend para tirar a operação comercial e financeira dos controles
manuais (planilhas e páginas publicadas à mão) e trazê-la para dentro do sistema. Este
arquivo é o **ponto de coordenação** entre sessões: antes de começar qualquer módulo, leia
as seções 1 a 3 inteiras.

Documentos irmãos: `API.md` (contrato), `ESTOQUE.md` (modelo de escopo de módulo),
`PENDENCIAS.md` (pendências pontuais fora deste roadmap).

---

## 1. Regras inegociáveis

1. **A estrutura do projeto não muda.** Todo módulo novo segue exatamente o caminho
   `core/domain` → `core/ports/in|out` → `core/usecases` → `adapter/postgres` →
   `adapter/http/dto` + `adapter/http/handlers` → registro em `rotas.go` e
   `cmd/api/main.go`. Nada de pacote `utils`, `common` ou `helpers` dentro de `internal`,
   nem camada nova.
2. **Boas práticas de Go**: `gofmt`, `go vet`, `staticcheck@v0.8.1` limpos; erros sempre
   tratados e embrulhados com `%w`; `context.Context` como primeiro parâmetro de tudo
   que toca I/O; interfaces declaradas do lado de quem consome (ports); sem estado global;
   sem `panic` fora do `main`; dependências injetadas pelo construtor.
3. **Nenhum comentário no código** — nem em linha, nem em bloco, nem doc comment. O
   código se explica por nomes de pacotes, tipos, funções e variáveis. Vale também para
   testes e SQL de migration.
4. **Commits pequenos, feitos pelo usuário.** A sessão nunca roda `git commit`,
   `git push`, `merge` nem `rebase`. Ao concluir cada etapa de mudança, ela para e
   informa o commit sugerido: arquivos a adicionar e mensagem no padrão
   `tipo: descricao em minusculas` (`feat`, `fix`, `refactor`, `test`, `docs`, `build`),
   uma etapa por commit.

## 2. Padrão de código (herdado — não reinventar)

| Tema | Padrão | Referência |
|---|---|---|
| Value objects | `type X struct{ campo }` + `NovoX(...) (X, error)` valida, `XDe(...) X` reconstrói do banco, getters sem prefixo `Get` | `domain/solicitacao/valor.go`, `domain/estoque/preco.go` |
| Enums | `type Status string` + constantes + `NovoStatus` / `StatusDe` / `Validado()` | `domain/solicitacao/status.go`, `domain/estoque/tipo_movimento.go` |
| Erros | só `domain.ErroValidacao`, `ErroNaoEncontrado`, `ErroPermissao`, `ErroConflito` | `domain/errors.go` |
| Erros HTTP | `400`/`404` com prefixo da sentinela, `403`/`409` sem prefixo, `500` fixo; `404`/`405` de rota também no envelope JSON | `handlers/resposta`, `middleware/roteamento.go` |
| Erros do banco | todo retorno de erro de função de escrita do repositório passa por `tratarErroDeGravacao` (FK → `registro relacionado não encontrado`); exclusões passam por `tratarErro` (FK → `registro em uso`). Os dois traduzem unique → `409` e CHECK, NOT NULL, texto longo, número fora do intervalo e formato inválido → `400`. Inclui o `tx.Commit()`, onde estouram as constraints adiadas | `postgres/erros.go`, `postgres/lead_repo.go` |
| Validação antes do banco | limites de tamanho e de tipo da coluna validados no domínio ou no usecase (ex.: `utf8.RuneCountInString` contra o `VARCHAR`, `int32` para coluna `INT`); a tradução do banco é rede de segurança, não regra de negócio | `usecases/leadpoint/lead_usecase.go`, `domain/estoque/quantidade.go` |
| Observabilidade | handler nunca engole erro: causa de `500` vai por `resposta.ResponderErro` ou `middleware.AnotarErro(w, err)`, e o `middleware.Registrar` grava a linha JSON com `requisicao_id`; falha depois de a resposta começar (ex.: `io.Copy` interrompido) também é anotada e sai como `WARN` | `middleware/registro.go`, `handlers/solicitacao/arquivo_handler.go` |
| Nomes | português em tudo, inclusive ports (`Criar`, `Obter`, `Listar`, `Atualizar`, `Remover`) | módulos `solicitacao` e `estoque` |
| Permissões | wrappers `protegido` / `administrador` / `comercial` em `rotas.go`; perfil novo = flag no `cargo` + método `TemAcesso<Perfil>` no usecase de usuários + `middleware.Exigir<Perfil>` + campo na resposta do login; regras finas em `usecases/<modulo>/permissoes.go` | `middleware/autenticacao.go`, `usecases/solicitacao/permissoes.go` |
| Sessão | troca de senha (própria ou pelo administrador) e logout avançam `usuario.versao_sessao`, que derruba os tokens emitidos antes; usuário desativado é barrado pelo middleware; o sistema nunca fica sem administrador ativo | `usecases/usuarios/usuario_usecase.go` |
| Dinheiro | centavos `BIGINT`, nunca `float` | `solicitacao.valor_estimado` |
| Banco | UUID `gen_random_uuid()`, `TIMESTAMPTZ`, `DATE` para vencimentos, constraints nomeadas `pk_/fk_/chk_/uq_`, índices `idx_*`, `ON DELETE RESTRICT` | `migrations/000019`, `000021` |
| Histórico | append-only; desfazer = lançamento inverso, nunca `UPDATE`/`DELETE` | `produto_movimento`, `solicitacao_historico` |
| Transição de estado | `UPDATE` condicionado ao status anterior, só colunas de transição; zero linhas afetadas → `409` se o registro existe, `404` se não | `atualizarStatusComGuarda`, `LeadRepository.MoverParaEtapa` |
| Datas | data do fato informada pelo usuário (`pago_em`, `vencimento`) separada do momento do registro (`criado_em`, `atualizado_em` sempre `time.Now()`); data futura onde não faz sentido → `400` | `domain/solicitacao/pagamento.go` |
| Paginação | `domain/paginacao_filtro.go` + `dto/paginacao.go`; todo `ORDER BY` de listagem termina na chave primária (`id`, ou `funil_id`/`etapa_id` nas tabelas antigas) para a paginação ser estável | listas de usuário/estoque |
| Arquivos | upload em `POST /api/v1/arquivos`, vínculo em tabela própria; rota que envia ou devolve arquivo usa `middleware.EstenderPrazo(prazoDeArquivo, ...)` e responde com `X-Content-Type-Options: nosniff`; anexo de solicitação nunca vira conteúdo público | `solicitacao_arquivo`, `produto_imagem`, `rotas.go` |
| Vitrine pública | produto só aparece se ativo, com saldo e de categoria visível (ativa e com pai ativo); a mesma regra vale para lista, detalhe e imagem | `usecases/estoque/vitrine.go` |
| Ports entre módulos | o consumidor declara a **porta mínima** que usa (só os métodos de que precisa) em vez de depender do repositório inteiro do outro módulo; cada método novo num port grande quebra os fakes de todos os pacotes que o implementam | `usecases/*/fakes_test.go` |
| Testes | domínio e usecase unitários com fakes; integração em `test/integration` via `test/helpers/banco.go`; e2e com `//go:build e2e` em `test/e2e` | `usecases/estoque/fakes_test.go` |

## 3. Protocolo de trabalho entre sessões

1. **Reservar antes de começar.** Escolha um módulo com todas as dependências em
   `concluído`, mude o status na tabela da seção 4 para `em andamento (<branch>)` e peça
   ao usuário o commit só dessa linha na `main` antes de escrever código. Módulo já
   reservado não é tocado por outra sessão.
2. **Uma branch por módulo**, `feat/<modulo>`, preferencialmente em worktree próprio.
3. **Escopo primeiro.** A primeira entrega do módulo é `<MODULO>.md` na raiz, no formato
   do `ESTOQUE.md` (entidades, endpoints, permissões, fases, decisões fechadas e
   abertas). Decisões abertas são levadas ao usuário antes de codar.
4. **Numeração de migration é definida no merge.** Durante o desenvolvimento use o
   próximo número livre; antes do merge, o usuário atualiza a branch com a `main` e a
   sessão renomeia a migration para o próximo número livre de novo. Nunca mergear migration com número menor que a última
   já existente na `main`.
5. **Arquivos compartilhados** (`rotas.go`, `cmd/api/main.go`, `API.md`,
   `test/helpers/fixtures.go`, `test/helpers/servidor.go`): só **acrescentar** um bloco
   do módulo ao final da seção correspondente; não reordenar nem reformatar o que já
   existe. Pedir ao usuário para atualizar a branch com a `main` com frequência.
6. **Contrato entre módulos é o port.** Módulo que consome outro depende apenas de uma
   interface — nunca do repositório ou do handler alheio. Prefira a **porta mínima**
   declarada no próprio módulo consumidor (`core/ports/out/<consumidor>/`), com só os
   métodos que ele usa, satisfeita pelo repositório do dono no `main.go`. Assim um
   método novo no port do dono não quebra os fakes de quem não o usa.
7. **Pronto significa**: `gofmt -l .` vazio, `go vet ./...`, `staticcheck ./...`,
   `go test ./...`, `go test -tags=e2e ./test/...` limpos; `API.md` atualizado;
   `<MODULO>.md` com fases marcadas; status `concluído` nesta tabela; cada etapa
   entregue com o commit sugerido ao usuário (regra 4 da seção 1).
8. **Frontend** fica no repo irmão `D:\PROJETOS\FRONT\painelMiaConstrutora`
   (`src/services/<modulo>.js`, `src/modules/<modulo>/`) e é entregue como fase final de
   cada módulo, ou por uma sessão própria depois que a API estiver `concluído`. Telas com
   dinheiro ou data só depois da trilha F0 (seção 5), que corrige a conversão de valores
   e de fuso.
9. **Banco de dev compartilhado é intocável.** Nenhuma sessão roda `migrate` no banco
   `mia` (porta `5444`): cada uma cria a própria `000NN` e duas aplicações cruzadas deixam
   a `schema_migrations` inconsistente. Para testar a API ou o front ao vivo, use um banco
   descartável no mesmo Postgres (`CREATE DATABASE mia_<modulo>`, `migrate up`, API em
   outra porta com `DATABASE_URL` apontando para ele, `DROP DATABASE` no fim). Os testes de
   integração e E2E já fazem isso sozinhos.
10. **Paralelismo limitado.** No máximo três sessões de módulo ao mesmo tempo: `rotas.go`,
    `cmd/api/main.go` e `test/helpers/servidor.go` são pontos únicos de contato e cada
    sessão a mais vira conflito de atualização com a `main`.
11. **Decisão em aberto bloqueia.** Módulo afetado por um item da seção 6 só é reservado
    depois que o usuário responder o item. A sessão não decide por conta própria.
12. **As regras vivem neste arquivo.** Worktrees nascem do que está commitado: um
    `CLAUDE.md` ignorado pelo git ou um arquivo não commitado não chega às sessões. Regra
    que vale para todas entra aqui, e este arquivo é commitado antes de abrir sessões.

## 4. Mapa de módulos e dependências

```
Onda 0   T0 Saneamento                       (sozinho, antes de qualquer módulo)
Onda 1   M1 Empresas e filiais
         M4 Fornecedores                     (não depende de empresa)
            │
Onda 2      ├── M2 Estoque por local, lote e reserva
            ├── M3 Clientes e representantes
            └── M5 Contas bancárias e plano de contas
                 │
Onda 3   M6 Pedidos de venda e entregas      (M1 M2 M3)
         M7 Contas a pagar                   (M1 M4 M5)
                 │
Onda 4   M8 NF-e por XML                     (M2 M3 M6)
         M9 Contas a receber e antecipação   (M3 M5)
         M10 Comissões                       (M3 M6)
         M11 Propostas e tabela de preços    (M3 M6)
                 │
Onda 5   M12 Importações                     (M2 M4 M6 M7)
         M13 Caixa e conciliação             (M5 M7 M9)
         M14 Dívidas e empréstimos           (M4 M5 M7)
                 │
Onda 6   M15 DRE e painéis                   (M2 M7 M8 M9 M13)
```

Módulos da mesma onda podem ser feitos em paralelo por sessões diferentes (no máximo três
ao mesmo tempo, regra 10 da seção 3).

A trilha **F0** (correções do front) corre em paralelo às ondas, numa sessão própria no repo
do front, e precisa estar concluída antes de qualquer tela de dinheiro ou data dos módulos
M6, M7, M9 e M13.

| # | Módulo | Depende de | Substitui (controle manual atual) | Status |
|---|---|---|---|---|
| T0 | Saneamento | — | — | pendente (comentários do `lead_repo.go` já removidos) |
| F0 | Correções do front | — | — | pendente |
| M1 | Empresas e filiais | T0 | separação por empresa feita à mão em todos os controles | pendente |
| M2 | Estoque por local, lote e reserva | M1 | Estoque geral, Pendências de envio | pendente |
| M3 | Clientes e representantes | M1 | Base de clientes | pendente |
| M4 | Fornecedores | T0 | coluna fornecedor do painel financeiro | pendente |
| M5 | Contas bancárias e plano de contas | M1 | classificação manual dos extratos | pendente |
| M6 | Pedidos de venda e entregas | M1 M2 M3 | Vendas futuras, Entregas de importação, Pendências de envio | pendente |
| M7 | Contas a pagar | M1 M4 M5 | Painel financeiro (contas a pagar) | pendente |
| M8 | NF-e por XML | M2 M3 M6 | Controle de NFs | pendente |
| M9 | Contas a receber e antecipação | M3 M5 | A receber no banco, títulos do Controle de NFs | pendente |
| M10 | Comissões | M3 M6 | comissões calculadas à mão | pendente |
| M11 | Propostas e tabela de preços | M3 M6 | orçamentos em documento, guia de modelos de venda | pendente |
| M12 | Importações | M2 M4 M6 M7 | parte de Entregas de importação e do painel financeiro | pendente |
| M13 | Caixa e conciliação | M5 M7 M9 | Caixa do mês, Fluxo de caixa diário | pendente |
| M14 | Dívidas e empréstimos | M4 M5 M7 | Dívidas do grupo | pendente |
| M15 | DRE e painéis | M2 M7 M8 M9 M13 | DRE, Faturamento, Relatório semanal | pendente |

---

## 5. Escopo por módulo

Cada item abaixo é o ponto de partida do `<MODULO>.md`; o detalhamento final (colunas,
mensagens de erro, códigos de status) é fechado no escopo do próprio módulo.

### T0 — Saneamento

Roda **sozinho**, antes de qualquer módulo: renomeia arquivos e ports que todo o resto
importa, e qualquer sessão em paralelo entraria em conflito.

- Remover os comentários que restam: `test/helpers/banco.go`, `test/helpers/doc.go`,
  `test/helpers/fixtures.go`, `test/helpers/servidor.go`,
  `test/e2e/adapter/http/api_test.go` e `test/integration/adapter/postgres/lead_repo_test.go`
  (apagar `doc.go` se ficar vazio). O `internal/adapter/postgres/lead_repo.go` já está limpo.
- Remover diretórios vazios legados: `internal/core/domain/payment`,
  `internal/core/domain/stock`, `internal/core/domain/fale-conosco`.
- Decidir com o usuário a nomenclatura dos ports de `lead`/`usuarios` (pendência já
  registrada em `PENDENCIAS.md`); se aprovado, traduzir para português e renomear
  `usecases/leadpoint` para `usecases/lead`. O pacote já tem testes de usecase
  (`lead_usecase_test.go`, `etapa_usecase_test.go`): eles são renomeados junto e precisam
  continuar passando sem mudança de comportamento.
- Ao traduzir `UsuarioRepository`, aproveitar para quebrar a dependência dos pacotes
  `estoque` e `solicitacao` no port inteiro: cada um declara a porta mínima de usuários e
  cargos que usa (regra 6 da seção 3), e os fakes encolhem.
- Sem migration. Não altera contrato HTTP.

### F0 — Correções do front

Sessão própria no repo `D:\PROJETOS\FRONT\painelMiaConstrutora`, saída do diagnóstico de
outubro/2026. Os dois primeiros itens são bugs que gravam e mostram dados errados hoje e
bloqueiam as telas de dinheiro e data dos módulos novos.

- **Valores**: `src/lib/formatar.js` → `paraCentavos("1.500")` vira 150 centavos (R$ 1,50).
  Tratar ponto como separador de milhar quando não houver vírgula e o padrão for de
  milhar; manter `"1234.56"` como decimal. O teste atual consolida o comportamento errado.
- **Datas sem fuso**: `formatarData` converte `DATE` (`2026-10-15T00:00:00Z`) para o fuso
  local e mostra 14/10 em UTC-3. Campos só-data formatam com `timeZone: 'UTC'`.
- **Produtos**: a tela mostra banner de erro e "Nenhum produto encontrado" ao mesmo tempo;
  falhas de categorias e resumo são engolidas; categorias e resumo são buscados de novo a
  cada página. Não mexer no layout (`Produtos.jsx`/`Produtos.css`), já corrigido à parte.
- **Upload**: `services/arquivos.js` não aplica a checagem de resposta não-JSON que o
  `requisitar` tem; resposta HTML do fallback do SPA quebra com `TypeError`.
- **Imagens de produto inativo**: o detalhe usa `/api/v1/publico/imagens/{id}`, que agora
  só serve imagem de produto visível na vitrine. O painel precisa baixar pela rota
  autenticada `/api/v1/arquivos/{id}` (via `fetch` com token e `URL.createObjectURL`).
- **Listas cortadas em 100**: aprovadores, categorias, cargos e o mapa de nomes da
  solicitação não paginam além do máximo da API.
- **Upload órfão**: se `anexarImagem` falha depois do upload, apagar o arquivo enviado
  (como o `PagamentoModal` já faz).
- **Permissões fail-open**: `AutenticacaoProvider` trata erro de rede/500 na sondagem de
  `aprovacao`/`financeiro` como permitido. Inverter para negado, ou trazer os perfis na
  resposta do login como já acontece com `administrador` e `comercial`.
- **Requisições**: nenhum hook passa `AbortController`; busca global em `/leads?q=` não
  volta para a página 1; "recarregar" dos leads busca duas vezes.
- **Sessão**: login sempre volta para `/` (perde o link de origem); logout não sincroniza
  entre abas (falta ouvir o evento `storage`).
- **Detalhes**: `falhaLead` não é limpa no retry; `baixarArquivo` revoga o object URL logo
  após o `click()`; `DataGrid` usa o índice como `key`.
- **Acessibilidade**: `index.html` com `lang="en"` e título "painelmiaconstrutora"; busca
  da TopBar sem label; botões Ajuda e Notificações sem ação; `Modal` sem foco preso e
  fechando formulário preenchido no clique fora.
- **Higiene**: `.gitignore` sem `.env`; arquivos sem uso (`App.css`, `index.css`,
  `assets/hero.png`, `assets/react.svg`, `assets/vite.svg`, `public/icons.svg`); README
  ainda é o template do Vite; `.oxlintrc.json` sem `exhaustive-deps`; comentários no
  código (`lib/api.js`, `vite.config.js`, `services/leads.js`, telas de leads e funis).
- **Testes** para os itens acima, no padrão dos existentes (`vitest` + Testing Library).

### M1 — Empresas e filiais

Fundação: tudo o que é financeiro, fiscal ou de estoque passa a pertencer a uma empresa.

- **Entidades**: `empresa` (razão social, nome fantasia, CNPJ, UF, regime tributário,
  ativa). Seed: MIA / Gigante PB, Red Diamond matriz, Gigante CE, Gigante SP.
- **Value objects**: `CNPJ` (dígitos verificadores), `UF` (enum fechado das 27),
  `RegimeTributario` (`simples`, `presumido`, `real`).
- **Adoção em módulos existentes**: `solicitacao.empresa_id`. Migration em três passos
  seguros: coluna nula → backfill com a matriz → `NOT NULL`. Filtro `?empresa=` nas
  listagens de solicitação.
- **Usuário x empresa**: tabela `usuario_empresa` (quais empresas o usuário enxerga);
  administrador enxerga todas. Regra aplicada no `permissoes.go` de cada módulo.
- **Endpoints**: CRUD em `/api/v1/empresas` (escrita administrador, leitura protegido).
- **Modelo de permissões** (decisão da seção 6, fechada no escopo do M1): o cargo já tem
  três flags (`administrador`, `financeiro`, `comercial`) e os módulos seguintes vão pedir
  mais (fiscal, cobrança, estoque). Junto com o escopo por empresa, decidir entre continuar
  com flags no cargo ou migrar para uma tabela de permissões, e fixar o padrão que todos
  os módulos seguem: onde o filtro por empresa entra na consulta e como o teste de
  isolamento entre empresas é escrito.

### M2 — Estoque por local, lote e reserva

Evolução do módulo `estoque` (mesmo pacote, sem módulo novo).

- **Local**: `estoque_local` (nome, empresa). Saldo deixa de ser só
  `produto.quantidade_atual` e passa a existir por local em `produto_saldo_local`;
  `quantidade_atual` vira a soma, mantida na mesma transação do movimento.
- **Lote**: `produto_lote` (produto, local, origem, custo unitário em centavos, data de
  entrada). Entrada cria lote; saída consome lote por FIFO. É o que dá o custo da
  mercadoria vendida para o M15.
- **Transferência entre locais**: novo tipo de movimento `transferencia` (sai de um
  local, entra em outro, mesmo lote, uma transação).
- **Reserva**: `produto_reserva` ligada ao item de pedido (M6). Disponível = saldo −
  reservado. A vitrine pública passa a usar o disponível.
- **Endpoints**: locais, saldo por local, lotes por produto, transferência,
  disponibilidade. Rotas públicas mantêm o contrato atual.
- **Migração de dados**: todo saldo atual vai para um local padrão por empresa, com um
  lote de custo desconhecido marcado como tal.

### M3 — Clientes e representantes

- **Entidades**: `cliente_grupo` (grupo econômico/construtora), `cliente` (PF ou PJ,
  documento, nome, grupo, UF, cidade, tipo `construtora`/`loja`/`pf`/`outro`, empresa
  de relacionamento, representante, e-mail para NF, telefone, observações),
  `cliente_obra` (obra/SPE do cliente, endereço de entrega).
- **Regras comerciais do cliente**: flags tipadas, não texto livre —
  `exige_numero_pedido_na_nf`, `proibe_antecipacao` (consumido pelo M9).
- **Duplicidade**: documento único (`uq_cliente_documento`) e busca por nome
  normalizado para evitar o mesmo cliente com nomes diferentes.
- **Representante**: `representante` (nome, documento, vínculo opcional a `usuario`,
  percentuais padrão por modelo de venda — consumidos pelo M10).
- **Lead → cliente**: `POST /api/v1/leads/{id}/converter` cria o cliente e guarda
  `cliente.lead_id`; o lead é marcado como convertido no histórico.
- **Importação inicial**: `POST /api/v1/clientes/importacao` (CSV) para carregar a base
  atual, com relatório de linhas rejeitadas.

### M4 — Fornecedores

- **Entidades**: `fornecedor` (documento opcional para estrangeiro, nome, país,
  moeda padrão, contato, observações).
- **Endpoints**: CRUD em `/api/v1/fornecedores`.
- Pequeno de propósito: destrava M7, M12 e M14.
- Fornecedor é do grupo, não de uma empresa: por isso não depende do M1 e roda na onda 1.
  Se alguma regra precisar de empresa, ela entra no M7 (a conta a pagar tem empresa).

### M5 — Contas bancárias e plano de contas

- **Conta bancária**: `conta_bancaria` (empresa, banco, agência, conta, apelido,
  ativa). Bancos atuais: Sicredi, Itaú, BB, Santander, Bradesco, Cora, Inter.
- **Plano de contas**: `categoria_financeira` em dois níveis (como `categoria` do
  estoque) com `natureza` (`receita`, `despesa`) e `grupo_dre` (`operacional`,
  `fora_dre`, `entre_contas`). É a classificação usada pelo M7, M13 e M15.
- **Endpoints**: CRUD de ambos; escrita restrita a financeiro/administrador.

### M6 — Pedidos de venda e entregas

Coração comercial. Substitui os três controles de vendas e pendências.

- **Pedido**: `pedido` (empresa vendedora, cliente, obra, representante, modelo
  `pronta_entrega`/`importacao`, condições, prazo-limite de entrega, status).
  `pedido_item` (produto, quantidade, preço unitário em centavos).
- **Estados**: `proposta` → `confirmado` → `parcialmente_entregue` → `entregue`;
  `cancelado` a partir de qualquer estado não final. Histórico append-only como
  `solicitacao_historico`.
- **Condições por modelo**: pronta entrega = boleto 30 dias; importação = entrada de
  30% e 70% na chegada, prazo de 120 dias contado da entrada paga. Percentuais como
  value object, não números soltos.
- **Reserva**: confirmar o pedido reserva estoque (M2) do que houver disponível; o
  restante fica como pendente de cobertura.
- **Entrega**: `entrega` + `entrega_item` (quantidade entregue por item, transportadora,
  frete CIF/FOB, status `pendente`/`saiu`/`entregue`). Entregar baixa a reserva e gera
  o movimento de saída no estoque na mesma transação.
- **Consultas**: pendências de entrega (vendido × entregue × disponível), pedidos com
  prazo vencido ou a vencer, cancelamentos com valor a devolver.
- **Cancelamento com entrada paga**: gera obrigação de devolução no M7 via port.

### M7 — Contas a pagar

Evolução do módulo `solicitacao` (mesmo pacote; o fluxo de aprovação continua igual).

- **Novos campos**: fornecedor (M4), categoria financeira (M5), competência (mês),
  vencimento, `estimado` (valor previsto x confirmado), conta bancária de pagamento
  (M5). A empresa já vem do M1.
- **Recorrência**: `despesa_recorrente` (fornecedor, categoria, valor, dia) que gera as
  solicitações do mês seguinte por rotina idempotente
  (`POST /api/v1/despesas-recorrentes/gerar?competencia=`), sem duplicar.
- **Adiar**: transição com novo vencimento e motivo, registrada no histórico.
- **Estorno, edição e notificação**: resolver os itens 6 e 7 de `PENDENCIAS.md` aqui
  (edição antes da aprovação; estorno como lançamento inverso; aviso de fila nova).
- **Visões**: a pagar por semana, por empresa, vencidos.

### M8 — NF-e por XML

- **Importação**: `POST /api/v1/notas-fiscais/importacao` recebe o XML (upload pelo
  fluxo de `arquivos`), valida a chave de acesso e a empresa emitente pelo CNPJ, e
  rejeita chave repetida (`409`).
- **Entidades**: `nota_fiscal` (chave, número, série, emissão, empresa, cliente,
  CFOP predominante, valor total, natureza `venda`/`transferencia`/`devolucao`),
  `nota_fiscal_item`, `nota_fiscal_duplicata` (número, vencimento, valor),
  `nota_fiscal_evento` (carta de correção, cancelamento).
- **Vínculos**: NF ↔ pedido (M6) ↔ entrega; cliente resolvido pelo documento (M3),
  criado se não existir; produto resolvido pelo `codigo`.
- **Efeitos**: duplicatas viram títulos no M9 via port; NF de transferência entre
  empresas do grupo é marcada como tal e não entra como receita no M15.
- **Fora de escopo deliberado**: campo de "valor real" divergente do valor da nota. O
  sistema registra o valor do documento fiscal e o valor do pedido; diferença entre
  eles é tratada com a contabilidade, não modelada como recurso.

### M9 — Contas a receber e antecipação

- **Título**: `titulo_receber` (empresa, cliente, origem NF/pedido/manual, número,
  parcela, vencimento, valor, conta bancária de cobrança). Estados: `carteira` →
  `em_banco` → `antecipado` → `recebido`; `vencido` é derivado da data, não gravado.
- **Boleto**: registro de envio ao cliente (`enviado_em`, canal) e alerta de boleto
  não enviado.
- **Antecipação (factoring)**: `antecipacao` (borderô: empresa de fomento, data, taxa,
  títulos, valor líquido) e `recompra` (título devolvido com encargos, gera conta a
  pagar no M7). Bloqueia antecipar título de cliente com `proibe_antecipacao` (M3).
- **Pagamento na conta errada**: registro de recebimento em conta diferente da
  cobrança, que gera a obrigação de repasse ao fomento.
- **Baixa**: total ou parcial, com conta bancária e data.

### M10 — Comissões

- **Regra**: percentual por modelo de venda e por representante (padrão no M3,
  sobrescrevível no pedido). Comissão nasce com a entrega (M6), proporcional ao
  entregue.
- **Entidades**: `comissao` (representante, pedido, entrega, base, percentual, valor,
  status `a_pagar`/`paga`), fechamento mensal com data de pagamento (dia 15).
- **Efeito**: fechamento gera a solicitação de pagamento no M7.

### M11 — Propostas e tabela de preços

- **Tabela de preço**: preço por produto e modelo de venda (pronta entrega x
  importação), com vigência. Vitrine pública continua usando `produto.preco`.
- **Proposta**: pedido em estado `proposta` (M6) com validade; geração de PDF a partir
  de um template único (cliente, itens, condições, frete, validade, assinatura).
- **Conversão**: aceitar proposta = confirmar pedido.

### M12 — Importações

- **Entidades**: `importacao` (fornecedor, empresa importadora, moeda, câmbio, status
  `pedido`/`embarcado`/`em_transito`/`desembaracado`/`recebido`), `importacao_item`,
  `importacao_custo` (frete, impostos, despesas aduaneiras) e datas de embarque e
  chegada prevista/real.
- **Vínculos**: itens de pedidos de importação (M6) ligados à importação que os
  atende; pagamentos ao fornecedor como contas a pagar (M7).
- **Recebimento**: cria lotes no estoque (M2) com custo unitário rateado (produto +
  custos) e libera as reservas dos pedidos atendidos.
- **Alertas**: pedidos cujo prazo de 120 dias vence antes da chegada prevista.

### M13 — Caixa e conciliação

- **Extrato**: importação OFX/CSV por conta bancária (M5), idempotente por
  identificador da transação.
- **Conciliação**: lançamento do extrato ↔ conta a pagar (M7) / título (M9) /
  transferência entre contas; lançamento sem par recebe categoria financeira.
- **Fechamento do mês**: saldo inicial + entradas − saídas = saldo final por conta,
  com divergência apontada.
- **Projeção**: saldo diário projetado = saldo atual + títulos a receber − contas a
  pagar, por empresa e consolidado.

### M14 — Dívidas e empréstimos

- **Entidades**: `divida` (credor: banco, investidor, fisco ou fornecedor; empresa;
  contrato; valor original; taxa; status), `divida_parcela`.
- **Efeito**: parcelas geram contas a pagar no M7; renegociação/parcelamento cria nova
  dívida ligada à original.
- **Visões**: saldo devedor por tipo de credor e por empresa; vencido x a vencer.

### M15 — DRE e painéis

Somente leitura; nenhum dado digitado aqui.

- **DRE mensal** por empresa e consolidada: receita (NF de venda, sem transferências),
  CMV (lotes consumidos), frete, comissões, impostos, custos fixos por categoria;
  saídas `fora_dre` à parte; ponto de equilíbrio.
- **Faturamento** histórico por empresa.
- **Relatório semanal**: NFs emitidas, pagamentos feitos, títulos vencidos, estoque
  baixo, saldos bancários.
- Consultas agregadas em SQL no `adapter/postgres`, expostas como `resumo` (padrão do
  `resumo_usecase` do estoque).

---

## 6. Decisões em aberto (levar ao usuário)

- [ ] **Empresa faturadora**: regra para decidir se a venda sai pela MIA ou pela CE
      (afeta M6 e M8).
- [ ] **Transferências entre empresas**: tratamento fiscal com a contabilidade (afeta
      M8 e M15).
- [ ] **Representante é usuário?** Se sim, o representante vê só os próprios pedidos e
      comissões (afeta M3, M6, M10).
- [ ] **Importação histórica**: quantos meses de NFs, títulos e extratos carregar.
- [ ] **Nomenclatura dos ports** de `lead`/`usuarios` (T0).
- [ ] **Modelo de permissões**: flags no cargo ou tabela de permissões, e o padrão de
      escopo por empresa (M1, antes da onda 2).
- [ ] **IP atrás de proxy**: o limite de tentativas de login usa o IP da conexão. Se o
      deploy ficar atrás de nginx/Caddy/load balancer, decidir entre mover o limite para o
      proxy ou ler um cabeçalho de IP confiável configurado por ambiente.
- [ ] **FK de `lead_historico.etapa_*` para `etapa`**: `RESTRICT` impede excluir etapa que
      já apareceu no histórico; `CASCADE` apaga o histórico junto. Hoje não há FK.