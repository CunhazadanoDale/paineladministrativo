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
| Validação antes do banco | limites de tamanho e de tipo da coluna validados no domínio ou no usecase (ex.: `utf8.RuneCountInString` contra o `VARCHAR`, `int32` para coluna `INT`); a tradução do banco é rede de segurança, não regra de negócio | `usecases/lead/lead_usecase.go`, `domain/estoque/quantidade.go` |
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
   cada módulo, ou por uma sessão própria depois que a API estiver `concluído`. O front de
   qualquer módulo a partir do M1 só começa depois do F0a (seção 5), que conserta as telas
   quebradas pelo M1 e pelo M2, a conversão de valores e de fuso, as permissões e os
   contratos de erro (`409` de versão); o F0b corre em paralelo.
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
Onda 4   M8 NF-e por XML                     (M2 M3 M6; fase final: M9)
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
ao mesmo tempo, regra 10 da seção 3). M8 e M9 correm juntos pelo acordo de títulos da seção 5
("Coordenação M8 × M9"): só a fase final do M8 espera o M9 estar na `main`.

A trilha **F0** (correções do front) corre em paralelo às ondas, numa sessão própria no repo
do front. O **F0a** precisa estar concluído antes do front de qualquer módulo a partir do M1;
o **F0b** pode correr junto com essas telas.

| # | Módulo | Depende de | Substitui (controle manual atual) | Status |
|---|---|---|---|---|
| T0 | Saneamento | — | — | concluído |
| F0a | Correções do front — bloqueante | — | — | concluído |
| F0b | Correções do front — qualidade | F0a | — | concluído |
| F-M1 | Front de empresas e filiais | M1 F0a | — | concluído |
| F-M4 | Front de fornecedores | M4 F0a | — | concluído |
| F-M5 | Front de contas bancárias e plano de contas | M5 F0a | — | concluído |
| F-M2 | Front de estoque por local, lote e reserva | M2 F0a | — | concluído |
| F-M3 | Front de clientes e representantes | M3 F0a | — | concluído |
| F-M6 | Front de pedidos de venda e entregas | M6 F-M2 F-M3 | — | concluído |
| F-M7 | Front de contas a pagar | M7 F-M4 F-M5 | — | concluído |
| F-M9 | Front de contas a receber e antecipação | M9 F-M3 F-M5 | — | concluído |
| F-M8 | Front de NF-e por XML | M8 F-M6 F-M9 | — | concluído |
| F-M10 | Front de comissões | M10 F-M6 F-M7 | — | em andamento (feat/front-m10-comissoes) |
| M1 | Empresas e filiais | T0 | separação por empresa feita à mão em todos os controles | concluído |
| M2 | Estoque por local, lote e reserva | M1 | Estoque geral, Pendências de envio | concluído |
| M3 | Clientes e representantes | M1 | Base de clientes | concluído |
| M4 | Fornecedores | T0 | coluna fornecedor do painel financeiro | concluído |
| M5 | Contas bancárias e plano de contas | M1 | classificação manual dos extratos | concluído |
| M6 | Pedidos de venda e entregas | M1 M2 M3 | Vendas futuras, Entregas de importação, Pendências de envio | concluído |
| M7 | Contas a pagar | M1 M4 M5 | Painel financeiro (contas a pagar) | concluído |
| M8 | NF-e por XML | M2 M3 M6 (fase final: M9) | Controle de NFs | concluído |
| M9 | Contas a receber e antecipação | M3 M5 | A receber no banco, títulos do Controle de NFs | concluído |
| M10 | Comissões | M3 M6 | comissões calculadas à mão | concluído |
| M11 | Propostas e tabela de preços | M3 M6 | orçamentos em documento, guia de modelos de venda | concluído |
| M12 | Importações | M2 M4 M6 M7 | parte de Entregas de importação e do painel financeiro | pendente |
| M13 | Caixa e conciliação | M5 M7 M9 | Caixa do mês, Fluxo de caixa diário | concluído |
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

**Entregue (outubro/2026):** comentários removidos dos testes e `doc.go` apagado; diretórios
vazios removidos; ports de `usuarios` e `lead` em português (`Criar`, `Obter`, `Listar`,
`Buscar`, `Remover`, `ContarPorFunil`, `ObterProxima`…), tipos `LeadHistory*` →
`LeadHistorico*`, `FunilRepo` → `FunilRepository`, `usecases/leadpoint` → `usecases/lead`
(importado como `leadusecases`; domínio como `domainlead`); portas mínimas
`UsuarioRepository`/`CargoRepository` (só `Obter`) em `ports/out/estoque` e
`ports/out/solicitacao`. Construtores seguem `NewX`, padrão de todos os módulos.

### F0 — Correções do front

Sessão própria no repo `D:\PROJETOS\FRONT\painelMiaConstrutora`, saída do diagnóstico de
outubro/2026, revisado depois do M5 contra o front (commit `1f542b6`) e a `API.md`. Dividida
em duas etapas:

- **F0a — bloqueante**: telas que pararam de funcionar com o M1 e o M2, bugs que gravam e
  mostram dados errados e os contratos novos de erro. Bloqueia o front de **todos** os
  módulos a partir do M1 (regra 8 da seção 3).
- **F0b — qualidade**: o restante do diagnóstico. Corre em paralelo ao front do M1–M5,
  depois que o F0a estiver concluído.

#### F0a — bloqueante

- **Solicitação sem empresa**: desde o M1 o `POST /api/v1/solicitacoes` exige `empresa_id`
  (empresa ativa e no acesso do solicitante), e `criarSolicitacao` não envia o campo: toda
  criação responde `400`. O formulário ganha o seletor de empresa; a lista e o detalhe
  mostram a empresa e a fila aceita `?empresa=`.
- **Movimento sem local**: desde o M2 o `POST /api/v1/produtos/{id}/movimentos` exige
  `local_id`, e a `entrada` exige `custo_unitario_centavos` (zero aceito); `movimentar` não
  envia nenhum dos dois: todo movimento responde `400`. O formulário ganha o seletor de local,
  o custo por tipo de movimento (tabela de `API.md`) e o tipo `implantacao`; `quantidade`
  negativa só no `ajuste`.
- **Seletores compartilhados**: seletor de empresa (`GET /api/v1/empresas?ativas=true`, só o
  acesso do usuário) e seletor de local de estoque, usados pelos dois itens acima e por todas
  as telas novas do M1–M5.
- **Valores**: `src/lib/formatar.js` → `paraCentavos("1.500")` vira 150 centavos (R$ 1,50).
  Tratar ponto como separador de milhar quando não houver vírgula e o padrão for de
  milhar; manter `"1234.56"` como decimal. O teste atual consolida o comportamento errado.
- **Datas sem fuso**: `formatarData` converte `DATE` (`2026-10-15T00:00:00Z`) para o fuso
  local e mostra 14/10 em UTC-3. Campos só-data formatam com `timeZone: 'UTC'`.
- **Permissões pelo login**: a resposta do login já traz `administrador`, `comercial` e
  `financeiro` (desde o M4), mas o `AutenticacaoProvider` só grava os dois primeiros e
  descobre `financeiro` por sondagem. `financeiro` passa a vir do login; só `aprovacao`
  continua por sondagem (aprovador é designação na tabela `aprovador`, não flag de cargo), e
  erro de rede/`500` na sondagem passa a ser **negado** (hoje é fail-open).
- **Versão otimista**: fornecedores, contas bancárias e plano de contas exigem `versao` no
  `PUT`, `ativar` e `desativar` e respondem `409` quando outra pessoa gravou antes. Tratamento
  único no front: aviso com a mensagem da API e recarga do registro, sem perder o formulário.
- **Testes** para os itens acima, no padrão dos existentes (`vitest` + Testing Library).

**Entregue (outubro/2026, branch `feat/f0a` do front):** para as telas do M1–M5:

- `paraCentavos` lê ponto como milhar quando não há vírgula e o padrão é de milhar;
  `formatarData` é só para campo só-data (formata em UTC) e `formatarDataHora` para instante
  (fuso local); os testes rodam com `TZ=America/Sao_Paulo` para o erro de fuso aparecer no CI;
- `useAutenticacao()` expõe `financeiro` (herdado do administrador) lido do login; só
  `permissoes.aprovacao` vem de sondagem, negada em qualquer erro;
- `listarTodas(caminho, params)` em `lib/api.js` percorre as páginas de 100 e recusa lista
  acima de 5000 em vez de cortá-la;
- `SeletorEmpresa` e `SeletorLocal` (`app/components`) carregam o acesso do usuário, selecionam
  sozinhos a única opção (`selecionarUnica={false}` em filtros) e limpam valor que saiu da
  lista; `useNomesEmpresas` (`app/empresas`) traduz `empresa_id` em nome fantasia;
- `ehConflitoDeConcorrencia(erro)` reconhece o `409` de versão ou de operação concorrente pela
  mensagem (a API não tem código próprio: todas dizem "recarregue") e `AvisoConflito` mostra
  a mensagem com "Recarregar dados" sem perder o formulário; o movimento de estoque já usa;
- o `429` do login já aparece com o tempo de espera, porque a tela mostra a mensagem da API;
  travar o botão durante o `Retry-After` foi para o F0b.

#### F0b — qualidade

- **Limite de login**: travar o botão de entrar durante o `Retry-After` do `429`.

- **Produtos**: a tela mostra banner de erro e "Nenhum produto encontrado" ao mesmo tempo;
  falhas de categorias e resumo são engolidas; categorias e resumo são buscados de novo a
  cada página. Não mexer no layout (`Produtos.jsx`/`Produtos.css`), já corrigido à parte.
- **Upload**: `enviarArquivo` em `services/arquivos.js` engole a falha de parse do JSON:
  resposta HTML do fallback do SPA com status `2xx` resolve como `null` em vez de erro.
  Aplicar a mesma checagem de resposta não-JSON do `requisitar`.
- **Imagens de produto inativo**: o detalhe (`ProdutoDetalhe.jsx`) usa
  `/api/v1/publico/imagens/{id}`, que só serve imagem de produto visível na vitrine. O
  painel precisa baixar pela rota autenticada `/api/v1/arquivos/{id}` (via `fetch` com token
  e `URL.createObjectURL`).
- **Listas cortadas em 100**: aprovadores, categorias, cargos e o mapa de nomes da
  solicitação não paginam além do máximo da API. Para o plano de contas usar
  `GET /categorias-financeiras/arvore`, que não tem o corte.
- **Upload órfão**: se `anexarImagem` falha depois do upload, apagar o arquivo enviado
  (como o `PagamentoModal` já faz).
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
- **Testes** para os itens acima, no padrão dos existentes.

**Entregue (outubro/2026, branch `feat/f0b` do front):** valem para as telas do M1–M5:

- `ErroApi.esperaSegundos` traz o `Retry-After` em segundos; o login trava o botão com contagem
  regressiva durante o `429`;
- `listarTodas` cobre aprovadores, usuários, cargos, categorias, funis e histórico e anexos da
  solicitação (`listarTodos*` nos services); os funis também cortavam em 50 no dashboard e nos
  leads. O plano de contas ainda não tem tela no front: a `GET /categorias-financeiras/arvore`
  fica para o front do M5;
- toda leitura dos services aceita `{ sinal }`, e todo hook de dados cria um `AbortController`
  por efeito e cancela ao desmontar ou trocar de chave (`modules/cancelamento.test.jsx` cobre
  cada hook);
- `enviarArquivo` rejeita `2xx` sem JSON com a mesma `MENSAGEM_SEM_JSON` do `requisitar`;
  `obterArquivoBlob` baixa com token e `ImagemArquivo` (`app/components`) mostra o arquivo por
  `URL.createObjectURL`. O detalhe do produto usa a rota autenticada só para o administrador:
  `GET /arquivos/{id}` serve apenas o dono ou o administrador, então os demais seguem na rota
  pública e não veem imagem de produto inativo;
- `anexarImagem` apaga o arquivo enviado quando o vínculo falha; `baixarArquivo` revoga o
  object URL depois de 60 s;
- Produtos separa o erro da lista do vazio e avisa falha de categorias e resumo com "Tentar
  novamente"; categorias carregam uma vez e o resumo só depois de alteração;
- Leads: a página fica atrelada ao `q`, então a busca da TopBar abre na página 1; `recarregar`
  busca a lista uma vez e só refaz funis e etapas se foram eles que falharam; o erro do lead e o
  da estrutura no detalhe ficam presos à chave e somem no retry;
- login volta ao link de origem (`state.de` no redirecionamento para `/login`, resolvido pela
  rota `/login` do `App`); o `AutenticacaoProvider` ouve o evento `storage` e sincroniza logout
  e login entre abas;
- `DataGrid` usa `row.id` como `key` e como seleção;
- `Modal` prende o foco, devolve o foco a quem abriu, usa `aria-labelledby` e não fecha no
  clique fora depois que algum campo mudou (X e Esc continuam fechando); `index.html` em
  `pt-BR` com o título "Painel MiaConstrutora"; busca da TopBar com `role="search"` e nome
  acessível; os botões Ajuda e Notificações saíram;
- higiene: `.env` ignorado, os seis arquivos sem uso removidos, README próprio,
  `react-hooks/exhaustive-deps` como erro (o código já estava limpo) e nenhum comentário em
  `src` nem no `vite.config.js`;
- testes: `asyncUtilTimeout` de 3 s no `src/test/setup.js`, porque a suíte completa estourava
  o limite de 1 s do Testing Library em máquina carregada.

#### Front do M1–M5 depois do F0a

Ordem sugerida, cada um como sessão própria (regra 8 da seção 3):

1. **M1 Empresas**: cadastro, ativar/desativar e vínculo de usuários às empresas; base do
   seletor de empresa.
2. **M4 Fornecedores**: pequeno; modelo da tela com versão otimista.
3. **M5 Contas bancárias e plano de contas**: árvore de categorias em dois níveis.
4. **M2 Locais, saldo por local, lotes, transferência e reservas.**
5. **M3 Clientes, obras, grupos, representantes**, conversão de lead e importação por CSV.

**Entregue o front do M1 (outubro/2026, branch `feat/front-m1-empresas` do front):**

- `services/empresas.js` ganhou `criarEmpresa`, `atualizarEmpresa` (sem `cnpj`), `ativarEmpresa`,
  `desativarEmpresa`, `removerEmpresa`, `obterEmpresasDoUsuario` e `definirEmpresasDoUsuario`;
  `listarEmpresas` e `obterEmpresa` passaram a aceitar `{ sinal }`;
- tela `/empresas` (`modules/empresas`), restrita a administrador como `/usuarios` e com entrada na
  Sidebar em Sistema: lista paginada com filtro de situação, cadastro com CNPJ numérico ou
  alfanumérico (enviado sem máscara e em maiúsculas; os dígitos verificadores ficam com a API),
  edição sem CNPJ, ativar/desativar e exclusão com a mensagem de "registro em uso" da API;
- vínculo de usuários às empresas pelo botão "Empresas do usuário" em `/usuarios`: o modal grava o
  conjunto inteiro (`acesso_todas` + `empresa_ids`), guarda a lista quando o acesso total está
  ligado, preserva vínculo com empresa que não veio na lista, marca empresa inativa e avisa quando
  o usuário fica sem acesso ou quando o cargo é de administrador (a rota mostra o gravado, não o
  efetivo);
- `useEmpresas` e `useAcessoEmpresas` com `AbortController`, ambos em
  `modules/cancelamento.test.jsx`;
- empresas não têm `versao` na API, então o `AvisoConflito` não entra aqui: o único `409` é o de
  CNPJ repetido, mostrado como erro do formulário sem perder o que foi digitado.

**Entregue o front do M4 (outubro/2026, branch `feat/front-m4-fornecedores` do front):**

- `services/fornecedores.js` com listagem (`busca`, `pais`, `moeda`, `ativo`), detalhe, criação,
  alteração, ativar/desativar e exclusão; `atualizarFornecedor` tira `pais` e `documento` do corpo
  (a API responde `400` se vierem) e ativar/desativar enviam `{ versao }`; as leituras aceitam
  `{ sinal }`;
- tela `/fornecedores` (`modules/fornecedores`) aberta a qualquer autenticado, com entrada na
  Sidebar em Financeiro: o financeiro (ou administrador) cria, edita, ativa e desativa; só o
  administrador exclui; os demais só leem. Lista paginada com busca, filtro de moeda e de situação;
- formulário por país: `BR` exige CPF ou CNPJ (enviado sem máscara e em maiúsculas) e aceita UF e
  cidade; outro país usa `identificacao_estrangeira` em maiúsculas; país e documento travados na
  edição; campo opcional vazio não vai no corpo, e o `PUT` leva a `versao` lida;
- **modelo da versão otimista para o M5**: o `409` de versão no `PUT` mostra o `AvisoConflito` no
  formulário, e "Recarregar dados" busca o registro (`GET /{id}` com `AbortController`), troca só a
  `versao` e mantém o que foi digitado, avisando para conferir e salvar de novo; o `409` de versão
  no ativar/desativar mostra o mesmo aviso na lista, que recarrega a página com as versões novas.
  Os outros `409` (documento ou identificação repetidos) seguem como erro comum do formulário;
- `useFornecedores` com `AbortController`, em `modules/cancelamento.test.jsx`; a recarga do
  formulário é cancelada quando o modal fecha (coberta em `Fornecedores.test.jsx`).

**Entregue o front do M5 (outubro/2026, branch `feat/front-m5-financeiro` do front):** as duas telas
ficam em `modules/financeiro` (nome do módulo na API), com um `Financeiro.css` próprio. Para os
próximos fronts:

- `services/contasBancarias.js` (lista com `empresa`, `banco`, `tipo`, `ativa`; detalhe; criação;
  `atualizarContaBancaria` manda só `{ apelido, versao }`; ativar/desativar com `{ versao }`; exclusão;
  `listarBancos` para o catálogo de `GET /bancos`) e `services/categoriasFinanceiras.js`
  (`obterArvoreFinanceira({ ativas })` para `GET /categorias-financeiras/arvore`; criação que tira
  `natureza` e `linha_dre` da subcategoria; `atualizarCategoriaFinanceira` manda só
  `{ nome, codigo, ordem, versao }`); todas as leituras aceitam `{ sinal }`;
- **seletores prontos para M7, M9, M13 e M14**: `SeletorBanco` (`modules/financeiro`, sobre o
  `SeletorRemoto`, catálogo pela API, sem lista fixa no front); para escolher categoria de lançamento
  use `obterArvoreFinanceira({ ativas: true })` e ofereça só as subcategorias com `lancavel`, filtrando
  pela natureza do lançamento; para escolher conta use `listarTodas('/api/v1/contas-bancarias',
  { empresa, ativa: true })` com a empresa do lançamento (a leitura é só do financeiro);
- `planoContasUtils.js` traz `NATUREZAS`, `LINHAS_DRE` (com as naturezas aceitas por linha, a mesma
  tabela do `chk_categoria_financeira_natureza_linha`), `rotuloDaNatureza`, `rotuloDaLinha` e
  `linhasDaNatureza`; o M15 pode reaproveitar os rótulos das linhas da DRE.
  `contasBancariasUtils.js` traz `TIPOS_CONTA`, `rotuloDoTipo` e a validação de agência, conta e
  dígitos;
- tela `/contas-bancarias` só para financeiro ou administrador (rota e entrada na Sidebar; a Sidebar
  ganhou o filtro `financeiro: true` ao lado de `admin` e `comercial`): lista paginada com filtros de
  empresa (`SeletorEmpresa` com inativas), banco, tipo e situação; empresa pelo `useNomesEmpresas`;
  criação com empresa ativa do acesso, banco, tipo, agência, conta e dígitos em campos próprios
  (pontos e espaços tirados, dígito em maiúsculas, dígito vazio não vai no corpo); a edição mostra os
  dados imutáveis e muda só o apelido;
- tela `/plano-de-contas` para qualquer autenticado, com entrada na Sidebar: árvore em tabela (raiz
  e subcategorias recuadas) com código, natureza, linha da DRE herdada, selo "Lançável" e filtro "Só
  ativas"; o financeiro cria raiz (linha da DRE limitada à natureza escolhida, aviso de
  `compra_mercadoria` fora do resultado), cria subcategoria pela raiz (mostra o que herda), edita
  nome, código e ordem (código apagado vai `null`, porque o `PUT` substitui os editáveis) e
  ativa/desativa; só o administrador exclui;
- versão otimista no padrão do M4 nas duas telas: `AvisoConflito` no formulário com recarga do
  registro por `GET /{id}` cancelada ao fechar, e na lista para o ativar/desativar. Os `409` sem
  "recarregue" (conta ou apelido repetidos, nome ou código repetidos, raiz com subcategorias ativas)
  ficam como erro comum; os `400` de empresa inativa e de pai inativo aparecem com "Fechar";
- `useContasBancarias` e `usePlanoContas` com `AbortController`, em `modules/cancelamento.test.jsx`;
- a suíte completa às vezes estoura o tempo em testes antigos que passam sozinhos (Login,
  ProdutoDetalhe, SolicitacaoFormModal, Usuarios), e o bundle passou de 500 kB (aviso do Vite, sem
  falhar o build): candidatos a code-splitting das rotas numa sessão própria.

**Entregue o front do M2 (outubro/2026, branch `feat/front-m2-estoque` do front, empilhada sobre a
`feat/front-m5-financeiro`):** para os próximos fronts (M3, M6, M8, M12):

- `services/locais.js` ganhou criação, renomear (só `{ nome }`: local não tem `versao`), ativar,
  desativar, exclusão e `listarSaldosDoLocal` (`?com_saldo=true`); `services/produtos.js` ganhou
  `local` em `listarMovimentos`, `listarLotes`, `transferir`, `reservar`, `listarReservas` e
  `liberarReserva` (`POST /reservas/{id}/liberar`); todas as leituras aceitam `{ sinal }`;
- `useNomesLocais` (`app/estoque`) traduz `local_id` em nome, inclusive de local inativo, no padrão do
  `useNomesEmpresas`; o `SeletorLocal` do F0a aceita `empresaId` para limitar o destino à empresa;
- tela `/locais-estoque` (`modules/locais`) aberta a qualquer autenticado, com entrada na Sidebar em
  Site & Catálogo: lista com filtros de empresa e situação; só o administrador cria (empresa ativa do
  acesso), renomeia (empresa travada), ativa, desativa (o `400` de local com saldo aparece com
  "Fechar") e exclui; "Ver saldos" abre os produtos do local com saldo, reservado e disponível;
- detalhe do produto: painel de saldo por local (`useSaldoProduto`, isolado: a falha não derruba o
  detalhe) com reservado, disponível e não alocado do grupo; movimentos com local, "origem → destino"
  na transferência, "Anterior aos locais" nos antigos, saldo no local e filtro por local;
  `useProdutoDetalhe` expõe `geracao`, que os painéis usam para recarregar depois de cada operação;
- transferência (`TransferenciaModal`, administrador): origem entre os locais com disponível, destino
  só entre os ativos da mesma empresa; o `409` de estoque recarrega o saldo sem perder o formulário;
- reservas (`ReservasProduto`): filtro de status (ativas por padrão); o administrador cria (limitada
  ao disponível do local), libera e baixa a reserva manual — a baixa é `POST /movimentos` com `tipo:
  "saida"`, o mesmo local e quantidade e o `reserva_id`. Reserva de `pedido_item` aparece "Pelo pedido",
  sem ações: **o front do M6 cria e baixa as reservas pelo pedido**;
- lotes (`LotesProduto`) só para administrador ou financeiro: custo unitário ("Desconhecido" no
  `null`), quantidades, valor em estoque e origem por transferência;
- `useLocais`, `useSaldosDoLocal`, `useSaldoProduto`, `useReservasProduto` e `useLotesProduto` com
  `AbortController`, em `modules/cancelamento.test.jsx`;
- as falhas intermitentes de tempo da suíte completa ficaram mais frequentes com o volume de testes
  (passam sozinhas); vale uma sessão para o `asyncUtilTimeout` ou para o pool do vitest.

**Entregue o front do M3 (outubro/2026, branch `feat/front-m3-clientes` do front):** telas em
`modules/clientes`. Para os próximos fronts (M6, M8, M9, M10):

- `requisitar` aceita `tipoConteudo` para mandar corpo cru (usado no CSV); `services/clientes.js`
  (clientes, obras, `converterLead`, `importarClientes(arquivo, { simular })`),
  `services/clienteGrupos.js` e `services/representantes.js`, com `listarTodosGrupos` e
  `listarTodosRepresentantes` para seletor; o `PUT` do cliente tira o `documento`;
- **para o M6**: `ClienteFormModal` reaproveitável (cria, edita e converte lead pela prop `inicial`);
  `useCadastrosCliente(comercial)` carrega grupos e, só para o comercial, representantes, com
  `nomeGrupo`/`nomeRepresentante`; vínculo inativo já gravado aparece "(inativo)" e é mantido; as obras
  (`listarObras(clienteId, { ativas: 'true' })`) são o endereço de entrega, montado por `enderecoDaObra`;
- `clientesUtils.js`: percentual em pontos-base (`formatarPercentual`, `paraPontosBase`,
  `pontosBaseParaTexto`, `erroDoPercentual` — **base do M10**), `TIPOS_CLIENTE`,
  `erroDoTipoComDocumento` (pf ⇔ CPF), CEP e o cabeçalho fixo da importação;
- `/clientes` aberta a qualquer autenticado (o comercial escreve, só o administrador exclui e importa;
  quem não é comercial vê "Com representante", porque não lê representantes); `/clientes/:id` com dados,
  regras comerciais, link para o lead de origem e obras (o comercial cria, edita e ativa/desativa; cliente
  inativo não recebe obra); `/grupos-clientes` aberta a todos; `/representantes` só para o comercial, com
  escrita do administrador (vínculo a usuário pela lista de usuários e aviso de que tirar o login desfaz o
  vínculo); entradas na Sidebar em Comercial;
- conversão de lead (`ConversaoLead`, no detalhe do lead): procura o cliente pelo `?lead=` e mostra o link;
  sem cliente, abre o cadastro preenchido com nome, e-mail e telefone do lead;
- importação (`ImportacaoClientesModal`, administrador): simular sem gravar, importar, relatório das linhas
  rejeitadas e recusa local de arquivo acima de 5 MB;
- M3 não tem `versao`: os `409` (documento, nome, usuário vinculado) são erro comum do formulário;
- `useGruposClientes`, `useRepresentantes`, `useClientes`, `useCadastrosCliente`, `useClienteDetalhe`,
  `useObrasCliente` e `useClienteDoLead` com `AbortController`, em `modules/cancelamento.test.jsx`;
- as falhas de tempo intermitentes da suíte completa continuam (passam sozinhas); segue valendo a sessão
  proposta para estabilizar a configuração do vitest.

**Entregue o front do M6 (outubro/2026, branch `feat/front-m6-pedidos` do front):** telas em
`modules/pedidos`. Para os próximos fronts (M8, M9, M10, M11, M12):

- `services/pedidos.js` (pedidos, entregas e as três consultas); `atualizarPedido` tira `empresa_id`;
  `obterObra` em `services/clientes.js`;
- peças genéricas novas: `BuscaRemota` (`app/components`: busca com debounce, cancelamento e o valor
  atual mantido fora dos resultados — use para escolher cliente, produto ou fornecedor em catálogo
  grande) e `useNomesPorId` (`app/nomes`: traduz ids em nomes buscando cada id uma vez, com
  `nomeDe` para outro campo);
- `PropostaFormModal` cria e edita a proposta (cliente por busca, obra ativa do cliente, representante
  padrão do cliente, modelo, prazo só na pronta entrega, itens com preço sugerido de `produto.preco`);
  **o M11 pode reaproveitar** trocando a sugestão de preço pela tabela;
- `/pedidos` (lista com filtros), `/pedidos/:id` (dados, itens com reservado e falta cobrir, histórico,
  editar, confirmar, reservar pendências, entrada paga do financeiro com "hoje" no fuso de Brasília,
  cancelar com valor a devolver), entregas no detalhe (programar por item e local da empresa, frete
  CIF/FOB, saída com baixa, entregue, cancelar; toda ação recarrega o pedido porque avança a `versao`
  dele), `/entregas` (consulta geral; as ações ficam no pedido) e `/pedidos/consultas` (abas de
  pendências, prazos e cancelamentos, com o hook genérico `useConsulta`);
- `AcaoPedidoModal` concentra validação (texto ou lista), conflito de versão (recarrega o pedido) e
  erro da API; `pedidosUtils` traz status, modelos, condição, `hojeEmBrasilia`, `entradaSugerida`,
  rótulos do histórico e das entregas;
- o título de entrada (M9) entrou depois, no F-M9; o F-M6 só registrava a entrada manual;
- `vite.config.js` com `testTimeout: 15000`: os estouros intermitentes eram de 5 s por teste.

**Entregue o front do M7 (outubro/2026, branch `feat/front-m7-contas-a-pagar` do front):** evolução de
`modules/solicitacoes` mais telas novas em `modules/financeiro`. Para os próximos fronts (M9, M10, M12–M15):

- `services/solicitacoes.js` só com acréscimos: filtros de conta a pagar, `atualizarSolicitacao` (sem empresa
  nem anexos), `aprovar(id, versao)`, `classificar`, `adiar`, `estornarPagamento`, `listarPagamentos` e
  `obterPendencias`; `services/despesasRecorrentes.js` e `services/contasAPagar.js` (resumo);
  `listarTodasContasBancarias` em `services/contasBancarias.js`;
- **peças reaproveitáveis**: `AcaoModal` (`app/components`, validação em texto ou lista, conflito de versão
  que fecha e recarrega, erro da API — movido de pedidos) e `SeletorCategoriaDespesa` (`modules/financeiro`,
  só subcategorias lançáveis de despesa agrupadas pela raiz, mantendo a gravada que deixou de ser lançável;
  **o M9 pode copiar o padrão para receita**);
- formulário da solicitação com fornecedor, categoria, competência, estimado, formas `transferencia` e
  `debito_automatico` e modo edição com versão; detalhe com editar, aprovar com versão, classificar, adiar,
  **pagamento com conta bancária obrigatória** (consertou o modal quebrado desde a fase 3 da API), estorno e a
  lista de pagamentos com os estornados; histórico com o rótulo do `evento`;
- lista com "Mais filtros" (fornecedor, categoria, competência, vencimento, origem, vencidas) e selo de
  vencida; contador de pendências no item Solicitações da Sidebar (`usePendenciasSolicitacao`, soma aprovação
  e financeiro, atualiza a cada troca de tela);
- `/despesas-recorrentes` (financeiro: cadastro com versão, ativar/desativar, exclusão do administrador e
  "Gerar competência" com geradas, já existentes e recusadas) e `/contas-a-pagar/resumo` (financeiro: totais,
  semana × empresa e vencidas antes do período);
- "hoje" das solicitações segue em UTC na API (item 11 do `PENDENCIAS.md`); o front usa a data local nos
  formulários.

**Entregue o front do M9 (outubro/2026, branch `feat/front-m9-contas-a-receber` do front):** módulo novo
`modules/receber`, só para o financeiro. Para os próximos fronts (M8, M10, M12–M15):

- `services/titulosReceber.js` (títulos, transições, recebimentos, estorno, recompra, boletos pendentes, resumo
  e `listarTitulosAntecipaveis`, que junta as listas completas de carteira e em banco sem origem cancelada) e
  `services/antecipacoes.js`; `gerarTituloEntrada` em `services/pedidos.js`;
- **peças reaproveitáveis**: `SeletorCategoriaLancavel` (`modules/financeiro`, com `SeletorCategoriaDespesa` e
  `SeletorCategoriaReceita`), `agoraEmBrasilia`/`instanteDeBrasilia` em `receberUtils` (datetime-local em
  Brasília para RFC 3339 com `-03:00`) e `numeroDoTitulo` (número/parcela) — **o M8 pode usar para listar as
  duplicatas da NF** pelo `?origem=nota_fiscal_duplicata&origem_id=`;
- `/titulos-receber`: lista com filtros (empresa, status, origem, cliente, vencimento, vencidos, origem
  cancelada) e o título manual (conta de cobrança opcional, "nasce em banco"); detalhe com histórico, edição do
  manual com versão, prorrogar, enviar/retirar do banco, boleto enviado, cancelar, recebimentos (conta ou
  fomento, juros, desconto, repasse obrigatório no antecipado recebido em conta, aviso de conta diferente da
  cobrança), estorno de recebimento e recompra;
- `/antecipacoes`: lista, borderô com seleção dos títulos antecipáveis, bruto/encargos calculados e taxa em %
  convertida para pontos-base, detalhe com os títulos vigentes ou fora e estorno do borderô;
- `/contas-a-receber/resumo` (a receber, antecipado e vencido por semana e empresa, atrasados à parte) e
  `/contas-a-receber/boletos` (prazo de 0 a 90 dias, registro do envio pela lista);
- no detalhe do pedido de importação, o financeiro vê o título de entrada (link e status) e o gera com
  vencimento e conta de cobrança opcional.

**Entregue o front do M8 (outubro/2026, branch `feat/front-m8-notas-fiscais` do front):** módulo novo
`modules/notas`. Para os próximos fronts (M10, M12–M15):

- `services/notasFiscais.js` (lista com todos os filtros, detalhe, importação da nota e do evento por
  `arquivo_id`, vínculo, ligar item a produto, sincronizar títulos, itens sem produto, entregas sem NF e o XML)
  e `enviarXml`, que manda `application/xml` quando o navegador não informa o tipo (`.xml` no Windows);
  `baixarDoCaminho(caminho, nome)` em `services/arquivos.js` para baixar qualquer rota autenticada;
- `/notas-fiscais` (qualquer autenticado lê; o financeiro importa a NF-e e o evento, com os avisos explicados
  e o arquivo apagado quando a importação é recusada): filtros de empresa, natureza, status, entre empresas,
  sem pedido, número, emissão (dia em Brasília) e chave;
- `/notas-fiscais/:id`: dados com chave em grupos e documento formatado, itens (ligar produto com versão),
  duplicatas com o título a receber de cada uma, eventos, baixar XML, sincronizar títulos (venda autorizada
  fora do grupo ou nota cancelada) e vínculo com pedido e entrega (pedidos confirmados, em entrega ou entregues
  do mesmo cliente; nulo desfaz). O vínculo fica só no detalhe: empresa e cliente só são conhecidos depois
  de ler o XML;
- `/notas-fiscais/itens-sem-produto` (o item pendente não traz a versão da nota: o front lê a nota antes de
  ligar) e `/notas-fiscais/entregas-sem-nota`; as duas entradas da Sidebar só para o financeiro;
- `notasUtils`: `dataEmBrasilia`/`dataHoraEmBrasilia`, `formatarDocumento`, `formatarChave`,
  `formatarQuantidade`, `descricaoDaEntrega`.

### M1 — Empresas e filiais

Fundação: tudo o que é financeiro, fiscal ou de estoque passa a pertencer a uma empresa.
O escopo fechado e o detalhamento estão em `EMPRESAS.md`; este resumo só aponta o que os
outros módulos precisam saber.

- **Entidades**: `empresa` (razão social, nome fantasia, CNPJ numérico ou alfanumérico,
  UF, regime tributário, ativa), `usuario_empresa` e `usuario.acesso_todas_empresas`. Sem
  seed: o administrador cadastra as empresas pela API.
- **Adoção em `solicitacao`**: a migration `000027` apaga as solicitações existentes
  (decisão do usuário) e cria `empresa_id` já `NOT NULL`. Filas, aprovação, pagamento e
  `?empresa=` respeitam o acesso; o solicitante sempre vê as próprias.
- **Modelo de permissões** (decidido): flags no cargo continuam; o escopo por empresa é o
  padrão da seção 4 do `EMPRESAS.md`, que todo módulo seguinte copia, inclusive o teste de
  isolamento entre empresas.

### M2 — Estoque por local, lote e reserva

Evolução do módulo `estoque` (mesmo pacote, sem módulo novo). O escopo fechado e o detalhamento estão
em `ESTOQUE_LOCAIS.md`; este resumo só aponta o que os outros módulos precisam saber.

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

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `ESTOQUE_LOCAIS.md`); front no F-M2.
Para os próximos módulos:

- o saldo antigo **não** foi para um local padrão: virou "não alocado" e o administrador o
  distribui com movimentos `implantacao` (decisão do usuário, independe das empresas
  cadastradas no deploy);
- todo movimento novo exige `local_id` (migration `000032`); entrada exige custo; saídas
  consomem lotes por FIFO e gravam o custo consumido em `produto_movimento_lote` — é daí que
  o M15 tira o CMV;
- operações de estoque são calculadas no domínio (`EstoqueProduto`) e gravadas por
  `EstoqueRepository.Gravar` com guarda otimista em `produto.versao_estoque`;
- reserva: `produto_reserva` com `origem` (`manual` hoje; o M6 acrescenta `pedido_item`) e
  baixa por saída com `reserva_id`; a vitrine usa o disponível (saldo − reservado);
- transferência só entre locais da mesma empresa; entre CNPJs aguarda a decisão aberta da
  seção 6.

### M3 — Clientes e representantes

O escopo fechado e o detalhamento estão em `CLIENTES.md`; este resumo só aponta o que os
outros módulos precisam saber.

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

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `CLIENTES.md`); front no F-M3. Para
os próximos módulos:

- cliente é do **grupo** (sem `empresa_id`), com documento obrigatório e único: o M8 resolve o
  cliente da NF-e por `ClienteRepository.ObterPorDocumento`;
- regras comerciais tipadas: `cliente.exige_numero_pedido_na_nf` (M8) e
  `cliente.proibe_antecipacao` (M9);
- representante com login (`representante.usuario_id`, não administrador) vê só os próprios
  clientes; M6 e M10 repetem a regra com `RepresentanteRepository.ObterPorUsuario`;
- comissão padrão do representante em pontos-base (`comissao_pronta_entrega_bp`,
  `comissao_importacao_bp`), base do M10;
- obras (`cliente_obra`) são o endereço de entrega do M6.

### M4 — Fornecedores

O escopo fechado e o detalhamento estão em `FORNECEDORES.md`; este resumo só aponta o que os
outros módulos precisam saber.

- **Entidades**: `fornecedor` (documento opcional para estrangeiro, nome, país,
  moeda padrão, contato, observações).
- **Endpoints**: CRUD em `/api/v1/fornecedores`.
- Pequeno de propósito: destrava M7, M12 e M14.
- Fornecedor é do grupo, não de uma empresa: por isso não depende do M1 e roda na onda 1.
  Se alguma regra precisar de empresa, ela entra no M7 (a conta a pagar tem empresa).

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `FORNECEDORES.md`); front no F-M4. Para
os próximos módulos:

- fornecedor do Brasil tem documento obrigatório (CPF ou CNPJ, validado por
  `domaincliente.NovoDocumento`); estrangeiro não tem documento e usa
  `identificacao_estrangeira`, única por país;
- `domainfornecedor.Moeda` (`BRL`, `USD`, `EUR`, `CNY`) é a moeda que o M12 reaproveita;
  `moeda_padrao` só sugere, não trava o lançamento;
- M7, M12 e M14 apontam para `fornecedor.id` com FK `RESTRICT` (a exclusão passa a responder
  `400 registro em uso`) e recusam fornecedor inativo em lançamento novo, por porta mínima com
  `Obter`;
- **perfil financeiro nas rotas**: `UsuarioUseCase.TemAcessoFinanceiro`,
  `middleware.ExigirFinanceiro`, wrapper `financeiro(...)` em `rotas.go` e campo `financeiro` na
  resposta do login; o M5 reaproveita;
- **versão otimista**: `fornecedor.versao` + `UPDATE ... WHERE versao = $n RETURNING versao`, versão
  exigida no corpo do `PUT`, `ativar` e `desativar` (`409` se divergir). É o modelo para o item 10
  do `PENDENCIAS.md` nos outros cadastros.

### M5 — Contas bancárias e plano de contas

O escopo fechado e o detalhamento estão em `FINANCEIRO.md`; este resumo só aponta o que os
outros módulos precisam saber.

- **Conta bancária**: `conta_bancaria` (empresa, banco pelo código COMPE, tipo, agência,
  conta, apelido, ativa). Bancos atuais: Sicredi, Itaú, BB, Santander, Bradesco, Cora, Inter.
  Escopo por empresa; leitura só para financeiro/administrador.
- **Plano de contas**: `categoria_financeira` em dois níveis (como `categoria` do
  estoque), do grupo, com `natureza` (`receita`, `despesa`, `transferencia`) e `linha_dre`
  no lugar do `grupo_dre` previsto antes, que não separava frete, comissões, impostos e
  custos fixos para o M15. A subcategoria herda natureza e linha; lançamento (M7, M9, M13)
  só em subcategoria ativa. `compra_mercadoria` fica fora do resultado: o CMV vem dos lotes.
- **Endpoints**: CRUD de ambos; escrita restrita a financeiro/administrador (perfil
  `financeiro(...)` criado pelo M4); exclusão só pelo administrador; versão otimista.
- **Saldo inicial** das contas fica no M13.

**Entregue (outubro/2026):** API completa (fases 1 a 4 do `FINANCEIRO.md`); front no F-M5. Para
os próximos módulos:

- M7, M9 e M13 apontam para `conta_bancaria.id` e `categoria_financeira.id` com FK `RESTRICT` e
  declaram porta mínima com `Obter`; a conta precisa estar ativa e ser **da mesma empresa** do
  lançamento, e essa empresa precisa estar ativa (desativar a empresa não desativa as contas);
  a categoria precisa ser `Lancavel()` (subcategoria ativa) e ter a natureza do lançamento
  (`despesa` no M7, `receita` no M9, as três no M13);
- o M15 agrupa a DRE por `linha_dre` e deixa de fora `compra_mercadoria` (o CMV vem de
  `produto_movimento_lote`), `fora_dre` (à parte) e `entre_contas`;
- leitura de contas bancárias é só do financeiro/administrador e respeita o acesso por empresa; o
  plano de contas é lido por qualquer autenticado (`GET /categorias-financeiras/arvore` para
  listas de seleção, sem o corte de 100 da paginação);
- regras que dependem de outra linha (empresa ativa, pai ativo, filhas ativas) travam essa linha
  na transação da escrita (`FOR SHARE`/`FOR UPDATE`); `postgres/guarda_versao.go` concentra
  versão otimista, motivo da falha e tradução de unicidade pelo nome da restrição.

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

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `PEDIDOS.md`); front no F-M6. Para os próximos módulos:

- empresa faturadora escolhida no pedido (decisão da seção 6); `pedido.numero` é sequencial por
  empresa (`pedido_sequencia`); condições do modelo gravadas no pedido (`entrada_bp`,
  `prazo_boleto_dias`, `prazo_entrega_dias`) e não editáveis no M6 — o M11 pode abrir a edição;
- proposta é o pedido em `proposta`: itens só mudam nesse estado; aceitar proposta (M11) =
  `POST /pedidos/{id}/confirmar`, que reserva o disponível nos locais ativos da empresa;
- reservas de pedido são `produto_reserva` com `origem = 'pedido_item'` e `origem_id` = item; só o
  pedido as libera ou baixa (o estoque responde `409`); o M12, ao receber importação, cria reservas
  pela mesma origem ou chama `reservar-pendencias`;
- toda gravação do módulo passa por `PedidoRepository.Gravar` (`ports/out/pedido.Gravacao`): pedido,
  entregas, operações de estoque e histórico na mesma transação, com guarda de versão do pedido, da
  entrega e do estoque. Operação de entrega avança a `versao` do pedido;
- saída de entrega gera um movimento `saida` por item (`entrega_item.movimento_id`), com o custo
  FIFO em `produto_movimento_lote`: é a base do CMV do M15 e da comissão do M10 (proporcional ao
  entregue em `pedido_item.quantidade_entregue`);
- entrada paga da importação fica em `pedido.entrada_paga_em/centavos` e define `prazo_limite`
  (120 dias); no M9 a baixa do título de entrada passa a registrá-la;
- cancelamento grava `valor_a_devolver_centavos`; a conta a pagar da devolução **ainda não é
  gerada** — fica para depois do M7 (consulta `GET /pedidos/cancelamentos?com_devolucao=true`);
- representante com login vê só os próprios pedidos e só usa clientes da própria carteira (M10
  reaproveita a regra);
- "hoje" do módulo é o dia no fuso de Brasília (`domainpedido.Hoje`); os próximos módulos com datas
  de fato devem usar a mesma referência.

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

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `CONTAS_A_PAGAR.md`); front no F-M7. Para os próximos módulos:

- a conta a pagar **é a `solicitacao`**: `fornecedor_id` e `categoria_financeira_id` opcionais na
  criação, `competencia` (`DATE`, dia 1) sempre preenchida, `estimado`, `versao`, e
  `origem`/`origem_id` (`manual`, `recorrencia`). O `prazo_pagamento` é o vencimento;
- quem gera conta a pagar (M6 devolução, M9 recompra, M10 comissão, M12, M14 parcela) acrescenta
  o próprio valor em `chk_solicitacao_origem`, grava `origem_id` com o id do seu registro e cria um
  índice único parcial para ser idempotente, como `uq_solicitacao_recorrencia (origem_id,
  competencia) WHERE origem = 'recorrencia'`. A criação passa por uma porta mínima do consumidor,
  satisfeita por um método novo do `SolicitacaoRepository` no padrão de `GerarDeRecorrencia`
  (trava a origem e a classificação com `FOR SHARE`, `INSERT ... ON CONFLICT DO NOTHING`, histórico
  na mesma transação). A assinatura é fechada pela primeira sessão consumidora;
- solicitação gerada por outro módulo deve nascer **com categoria** lançável: pagar exige categoria
  ativa (`solicitação sem categoria financeira; classifique antes de pagar`). A `competencia` de
  solicitação com `origem` diferente de `manual` é fixa (é a chave de idempotência);
- transições da solicitação guardam status **e versão**; o aprovar aceita `{ "versao": n }` e o
  front deve mandá-la;
- pagamento tem `conta_bancaria_id` obrigatória (conta ativa da mesma empresa) e `usuario_id`;
  estorno é lançamento inverso em `pagamento_estorno`, o pagamento fica `estornado = true` e a
  solicitação volta para `aprovado`. Só um pagamento não estornado por solicitação
  (`uq_pagamento_solicitacao_vigente`). O M13 concilia por `conta_bancaria_id` + `pago_em`
  (`idx_pagamento_conta_bancaria`);
- o M15 usa `competencia` + `categoria_financeira.linha_dre` (DRE por competência, decisão D5) e
  ignora pagamento estornado; o fluxo de caixa (M13) usa o `pago_em`;
- `solicitacao_historico.evento` (`criada`, `aprovada`, `rejeitada`, `cancelada`, `paga`,
  `editada`, `classificada`, `adiada`, `estornada`) registra também o que não troca status;
- `forma_pagamento` aceita também `transferencia` e `debito_automatico`.

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

#### Coordenação M8 × M9

Acordo fechado (outubro/2026) para as duas sessões correrem em paralelo:

- **O M9 é dono do título.** `titulo_receber` tem `origem` + `origem_id` no padrão da
  `solicitacao` do M7: valores `manual`, `pedido` e `nota_fiscal_duplicata`, todos já no
  `chk_titulo_receber_origem` desde a primeira migration do M9; `origem_id` **sem FK**, porque
  `nota_fiscal_duplicata` nasce no M8; idempotência por índice único parcial em
  `(origem, origem_id) WHERE origem <> 'manual'`.
- **O M9 expõe a geração por origem.** Um método do `TituloReceberRepository` no padrão de
  `GerarDeRecorrencia` (trava o que precisa com `FOR SHARE`, `INSERT ... ON CONFLICT DO
  NOTHING`, histórico na mesma transação), para o consumidor declarar a porta mínima. A
  assinatura, e o que acontece com o título quando a origem é cancelada, ficam fechadas no
  escopo do M9 (primeira entrega da sessão) e o M8 as lê de lá; mudança depois disso é
  combinada com o usuário.
- **O M8 só guarda as duplicatas até o M9 chegar.** As fases do M8 gravam
  `nota_fiscal_duplicata` sem gerar títulos. A geração (e o efeito do cancelamento da NF nos
  títulos) é a **última fase do M8**, que só começa depois que o M9 estiver mergeado na
  `main` e a branch do M8 for atualizada com ela.
- **O pedido (M6) é tocado pelos dois, por porta mínima própria.** O M8 vincula NF ↔ pedido
  ↔ entrega e o M9 registra a entrada paga do pedido na baixa do título de entrada. Cada um
  declara a sua porta em `core/ports/out/<consumidor>/`, sem acrescentar métodos ao port do
  M6; métodos novos no `adapter/postgres` do pedido entram só no fim do arquivo (regra 5 da
  seção 3).
- **Sem outros cruzamentos.** A recompra do M9 gera conta a pagar no M7 (o M8 não toca na
  `solicitacao`); criar cliente a partir da NF é só do M8.

**Entregue (outubro/2026):** API completa (fases 1 a 6 do `NOTAS_FISCAIS.md`); front no F-M8. Para os próximos módulos:

- a NF entra pelo XML autorizado (`nfeProc`) enviado em `POST /arquivos` e importado em
  `POST /notas-fiscais/importacao`; o sistema não emite NF-e e não confere a assinatura XMLDSig
  (autenticidade pela chave e pelo protocolo da SEFAZ). Só NF emitida por empresa do grupo;
- `nota_fiscal.natureza` é `venda`, `remessa`, `transferencia` ou `devolucao` (seção 3.3 do
  `NOTAS_FISCAIS.md`). **O M15 usa só `venda` autorizada como receita**, por `emitida_em` (fuso de
  Brasília) e `valor_total_centavos`; `entre_empresas = true` fica fora da receita pela regra
  provisória da seção 6;
- a NF **não move estoque** (a saída é a da entrega do M6); o vínculo NF ↔ pedido ↔ entrega é
  manual (`pedido_id`, `entrega_id`), com no máximo uma NF autorizada por entrega;
- itens guardam `quantidade_decimilesimos` (`qCom` × 10⁴) e `produto_id` resolvido pelo `codigo`
  (nulo quando não acha, listado em `itens-sem-produto`); o M10 pode cruzar NF e pedido por eles;
- só `venda` gera títulos (um por duplicata, origem `nota_fiscal_duplicata`); cancelar a NF por
  evento cancela os títulos pelo `CancelarPorOrigem`, e `POST /notas-fiscais/{id}/titulos` refaz
  o que tiver falhado (idempotente);
- migrations `000051` e `000052` (renumeradas depois do M9).

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
- **Coordenação com o M8**: o título, a origem `nota_fiscal_duplicata` e a geração por origem
  são do M9 e seguem o acordo em "Coordenação M8 × M9" (seção do M8).

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `CONTAS_A_RECEBER.md`); front no F-M9. Para os próximos módulos:

- **M8 (última fase)**: a porta mínima tem só `GerarDeOrigem` e `CancelarPorOrigem` do
  `TituloReceberRepository` (seção 6.1 do `CONTAS_A_RECEBER.md`); o M8 acrescenta o `case
  OrigemNotaFiscalDuplicata` em `travarOrigem` (`titulo_receber_repo.go`), travando a `nota_fiscal`
  com `FOR SHARE` e recusando NF cancelada ou entre empresas do grupo. Cancelar a NF chama
  `CancelarPorOrigem`, que cancela os títulos sem movimento e marca `origem_cancelada_em` nos
  movimentados, sem nunca recusar;
- a conta a pagar gerada por outro módulo está fechada no M7 (`NovaSolicitacaoDeOrigem`,
  `SolicitacaoRepository.GerarDeOrigem` e `gerarSolicitacaoDeOrigem` para gravar na mesma
  transação, `uq_solicitacao_origem`): M10, M12 e M14 acrescentam a própria origem em
  `chk_solicitacao_origem` e em `Origem.Sistemica()`;
- **M13**: concilia `titulo_recebimento` (tipo `conta`, por `conta_bancaria_id` + `recebido_em`,
  entrada no caixa = valor + juros) e o crédito do borderô (`antecipacao.conta_bancaria_id` +
  `operado_em`, valor líquido); a projeção usa o saldo dos títulos `carteira` e `em_banco`;
- **M15**: título `manual` entra na DRE por `competencia` + categoria; títulos de NF e de pedido ficam
  fora (a receita vem da NF); juros recebidos e encargos do borderô (bruto − líquido) são resultado
  financeiro; recompra e repasse entram pela conta a pagar do M7;
- "hoje" do módulo é o dia no fuso de Brasília (`domainreceber.Hoje`).

### M10 — Comissões

- **Regra**: percentual por modelo de venda e por representante (padrão no M3,
  sobrescrevível no pedido). Comissão nasce com a entrega (M6), proporcional ao
  entregue.
- **Entidades**: `comissao` (representante, pedido, entrega, base, percentual, valor,
  status `a_pagar`/`paga`), fechamento mensal com data de pagamento (dia 15).
- **Efeito**: fechamento gera a solicitação de pagamento no M7.

**Entregue (outubro/2026):** API completa (fases 1 a 4 do `COMISSOES.md`); front pendente (fase 5),
depois do F0a. Para os próximos módulos:

- comissão nasce na **saída** da entrega, uma por entrega (`uq_comissao_entrega`), por apuração
  idempotente (`POST /comissoes/apurar`) que não mexe no fluxo do M6; base = preço × quantidade dos
  itens que saíram, sem frete; percentual do pedido em `pedido_comissao` (tabela do M10, sem coluna nova
  no `pedido`), sobrescrevível pelo comercial até a primeira saída;
- ajuste manual append-only (`comissao_ajuste`) para devolução e correção; fechamento mensal por
  representante × empresa, só de mês encerrado, que gera a conta a pagar no M7 (`origem = comissao`,
  `pendente_aprovacao`, vencimento dia 15 do mês seguinte) na mesma transação;
- o representante aponta para um `fornecedor` (`representante.fornecedor_id`, rota
  `PUT /representantes/{id}/fornecedor`), que recebe a conta a pagar: a resposta à dúvida do favorecido
  vale também para a devolução do pedido cancelado (pendência registrada na branch do M6);
- **M15**: a despesa de comissão vem da conta a pagar (`competencia` + `linha_dre` da categoria,
  sugerida `despesa_variavel`), nunca das tabelas do M10, para não contar duas vezes;
- **M13**: concilia o pagamento da comissão como qualquer conta a pagar do M7;
- **M11**: pode mexer no `pedido` livremente; o M10 só lê pedido, entrega e itens.

### M11 — Propostas e tabela de preços

- **Tabela de preço**: preço por produto e modelo de venda (pronta entrega x
  importação), com vigência. Vitrine pública continua usando `produto.preco`.
- **Proposta**: pedido em estado `proposta` (M6) com validade; geração de PDF a partir
  de um template único (cliente, itens, condições, frete, validade, assinatura).
- **Conversão**: aceitar proposta = confirmar pedido.

**Entregue (outubro/2026):** API completa (fases 1 a 3 do `PROPOSTAS.md`); front pendente (fase 4),
depois do F0a. Para os próximos módulos:

- tabela de preço do grupo (`tabela_preco_item`) por produto × modelo, com vigência só por data de
  início; `GET /tabela-precos/vigentes` é a regra única de preço do dia (tabela, senão `produto.preco`);
- a proposta é o pedido em `proposta` com `validade` (padrão hoje + 15), `revisao`, condição editável
  (`entrada_bp`, `prazo_boleto_dias`, `prazo_entrega_dias`) e frete estimado **fora** do total; proposta
  vencida não confirma. Cada item guarda o `preco_tabela_centavos` do dia da última alteração — o M15
  pode medir desconto concedido por ele;
- o documento impresso sai de `GET /pedidos/{id}/proposta` (front monta a página; sem PDF no backend);
- migrations `000057` e `000058` (renumeradas depois da correção `000056` do M10).

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

**Entregue (outubro/2026):** API completa (fases 1 a 5 do `CAIXA.md`); front pendente (fase 6). Para os
próximos módulos:

- o extrato é a fonte do saldo: `caixa_abertura` (saldo no fim da data de corte) + `extrato_lancamento`
  depois do corte; só OFX, idempotente por linha (`uq_extrato_lancamento_identificador`);
- a conciliação lê `pagamento`, `titulo_recebimento` (tipo `conta`), `antecipacao` e as pontas de
  `transferencia_conta` por consulta própria do M13: **nenhum port nem repositório do M7 e do M9 mudou**;
  estorno depois de conciliar invalida a conciliação, sem bloquear o M7 nem o M9;
- linha sem par vira `lancamento_caixa` (categoria lançável, `competencia` = mês da linha, `estornado` quando
  a conciliação é desfeita): **o M15 lê tarifas, juros e rendimentos daqui**, por `competencia` +
  `linha_dre`, ignorando os estornados; `transferencia_conta` e `entre_contas` ficam fora da DRE;
- `caixa_fechamento` vigente por conta e mês trava o caixa daquele mês; o M15 pode usar `saldo_final_centavos`
  dos meses fechados e `GET /caixa/saldos` para os saldos do relatório semanal;
- transferência entre empresas segue a regra provisória da seção 6 (marcada `entre_empresas`);
- migrations `000059` a `000063`; renumerar antes do merge se outra sessão mergear antes.

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

- [x] **Empresa faturadora** (outubro/2026): escolhida no pedido entre as empresas ativas no
      acesso do usuário, sem regra automática; vale para M6 e M8.
- [ ] **Transferências entre empresas**: tratamento fiscal com a contabilidade (afeta
      M8 e M15). Faltam: tipo de nota emitida entre as empresas, se gera título e conta a
      pagar entre elas, se entra como receita na DRE da empresa vendedora e se move estoque
      entre os locais (e por qual custo).
      **Regra provisória** (outubro/2026, até a contabilidade responder): o M8 importa a NF
      cujo destinatário tem CNPJ de empresa cadastrada e a marca como **entre empresas do
      grupo**, sem gerar título, conta a pagar nem movimento de estoque; o M15 a deixa fora da
      receita. Com essa regra o M8 pode ser reservado; o efeito definitivo vira fase própria
      quando a decisão chegar.
      **M13** (outubro/2026, D8 do `CAIXA.md`): transferência entre contas de empresas diferentes
      segue a mesma regra provisória — permitida, marcada `entre_empresas`, categoria
      `entre_contas`, fora da DRE das duas empresas, sem título nem conta a pagar entre elas.
- [x] **Representante é usuário?** Vínculo opcional (outubro/2026): `representante` é cadastro
      próprio que pode apontar para um `usuario`; o representante com login vê só os próprios
      clientes, pedidos e comissões; sem login, existe só para cálculo (afeta M3, M6, M10).
- [ ] **Importação histórica**: quantos meses de NFs, títulos e extratos carregar.
      **Extratos decididos** (outubro/2026, D3 do `CAIXA.md`): sem retroativo; o caixa do M13
      começa no último mês encerrado e pagamentos e recebimentos anteriores ao corte ficam fora da
      conciliação. NFs e títulos seguem em aberto.
- [x] **Nomenclatura dos ports** de `lead`/`usuarios` (T0): traduzidos para português.
- [x] **Modelo de permissões** (M1): flags no cargo continuam; o escopo por empresa segue
      o padrão da seção 4 do `EMPRESAS.md`.
- [ ] **IP atrás de proxy**: o limite de tentativas de login usa o IP da conexão. Se o
      deploy ficar atrás de nginx/Caddy/load balancer, decidir entre mover o limite para o
      proxy ou ler um cabeçalho de IP confiável configurado por ambiente.
- [ ] **FK de `lead_historico.etapa_*` para `etapa`**: `RESTRICT` impede excluir etapa que
      já apareceu no histórico; `CASCADE` apaga o histórico junto. Hoje não há FK.