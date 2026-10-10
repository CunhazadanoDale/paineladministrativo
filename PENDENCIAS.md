# Pendências

O que ficou em aberto depois da rodada de paginação, autenticação, seed, documentação e do
módulo de solicitações de pagamento (outubro/2026). Itens em ordem sugerida de ataque —
atualize a coluna **Status** conforme forem resolvidos.

---

## Próximos passos

| # | Item | Por quê | Esforço | Status |
| --- | --- | --- | --- | --- |
| 1 | **README completo** | O `README.md` tem 2 bytes. Quem clona não descobre que `JWT_SECRET` é obrigatório (a API não sobe sem), como subir banco/migrations, os comandos de teste nem que existe a `API.md` | Baixo | concluído (requisitos, variáveis, subida pelo Compose, comandos de teste e links da documentação) |
| 2 | **Rotas de sessão** | Usuário comum consegue logar, mas não vê o próprio perfil nem troca a própria senha — todas as rotas `/usuarios` são de administrador. Criar `GET /api/v1/sessao` (perfil do token) e `POST /api/v1/sessao/senha` (exigindo a senha atual) | Baixo | concluído (`GET /api/v1/sessao`, `POST /api/v1/sessao/senha` exigindo `senha_atual` e `POST /api/v1/sessao/encerrar`; trocar a senha ou encerrar a sessão revoga todos os tokens do usuário via `usuario.versao_sessao`, migração `000022`; e2e em `usuarios_test.go`; API.md atualizado) |
| 3 | **Senha temporária no 1º login** | A decisão tomada no seed foi "senha inicial fixa, trocar no primeiro login", mas não existe flag de senha temporária nem exigência de troca. Alternativa: aceitar o fluxo manual e apenas manter a orientação na `API.md` | Médio | pendente |
| 4 | **Rate limit no login** | `POST /usuarios/autenticar` não limita tentativas: é possível testar senhas em loop. A mensagem genérica `email ou senha inválidos` esconde o que existe, mas não limita a frequência. Limite simples por IP/e-mail, em memória | Médio | concluído (em memória: 5 falhas por e-mail + IP e 50 por IP em 15 minutos, `429` com `Retry-After`; e-mail inexistente compara contra hash fictício para não revelar cadastro) |
| 5 | **Total de páginas na paginação** | Hoje a resposta ecoa só `pagina`/`tamanho` (decisão registrada). Se o front for fazer navegação "página X de Y", precisa de contagem — muda o contrato das listas | Baixo | pendente |
| 6 | **Notificação de fila nova** | O módulo de solicitações só empurra trabalho para a pessoa se ela abrir a fila (`escopo=aprovacao` / `escopo=financeiro`). Sem e-mail, webhook ou ao menos badge de pendências, aprovação depende de alguém lembrar de olhar | Médio | pendente |
| 7 | **Editar e estornar solicitação** | Não existe `PUT /api/v1/solicitacoes/{id}` (para corrigir um dado é cancelar e recriar, e o histórico fica) e a tabela `pagamento` é única por solicitação, então um pagamento errado não tem estorno pela API — só correção direta no banco | Médio | pendente |
| 8 | **Expurgo de anexo órfão** | Anexo enviado e nunca vinculado a solicitação fica no storage para sempre (o `DELETE /api/v1/arquivos/{id}` existe, mas ninguém é obrigado a usá-lo). Falta rotina de varredura ou regra de retenção | Baixo | pendente |
| 9 | **Telas do módulo de solicitações** | As 17 rotas do módulo estão prontas e cobertas por e2e, mas não há tela no front: fila do aprovador, fila do financeiro, upload com progresso e histórico | Alto | concluído (front em `painelMiaConstrutora`: fila com escopos, upload com progresso, detalhe com anexos e histórico) |

---

## Decisões em aberto

### Falha de login responde `404` ou `401`?

Hoje: `404` com `email ou senha inválidos` (status herdado da decisão da rodada 2; a
mensagem foi limpa — vinha como `registro não encontrado: email ou senha inválidos`).

`401` é o padrão HTTP para credenciais inválidas. Se mudar, é uma linha no handler de
`Autenticar` — mas é uma mudança de contrato para quem já consome a API.

- [ ] Definir o status final
- [ ] Se mudar: atualizar `API.md`, teste do handler e teste e2e

### Rate limit: em memória ou no proxy?

Decidido: na aplicação, em memória (zera ao reiniciar e não é compartilhado entre réplicas).
A origem é o `RemoteAddr` da conexão. Atrás de um proxy reverso todo mundo chega com o IP do
proxy: aí o limite por IP (50) vira um limite global e o por e-mail + IP passa a valer por
e-mail. Se a API for para trás de nginx/Caddy, ou o limite migra para o proxy ou a aplicação
passa a ler um cabeçalho de IP confiável configurado explicitamente.

---

## Fora do repositório

- [x] **Frontend precisa enviar `Bearer`** — resolvido: o cliente HTTP do front
  (`src/lib/api.js`) injeta `Authorization` em toda requisição, guarda o token em
  `localStorage` e encerra a sessão no `401`. O fluxo e os exemplos estão na `API.md`.
- [ ] **Snippet de client** — gerar um exemplo pronto (fetch/axios) de login + chamada
  autenticada para colar no front.
- [ ] **`JWT_SECRET` no ambiente de produção** — o `docker compose up api` falha de
  propósito se não houver `.env` com o segredo (fail-closed). Garantir que o ambiente
  tenha um valor gerado (`openssl rand -hex 32`), nunca o de desenvolvimento.
- [ ] **Go 1.27.2 no `go.mod`** — o Dockerfile e o `govulncheck` já usam 1.27.2, que corrige
  9 vulnerabilidades da biblioteca padrão presentes na 1.27.1 (`net/http`, `http2`,
  `crypto/tls`, `net/textproto`). O `go.mod` e os workflows de lint e testes ficam em 1.27.1
  de propósito: o `staticcheck@v0.8.1` não lê o formato de pacote do 1.27.2 ("export data
  version 5"). Subir o `go.mod` assim que sair um staticcheck compatível.
- [ ] **`CORS_ORIGINS` explícito em produção** — vazio não libera nenhuma origem (o front em outro domínio deixa de funcionar até a variável ser preenchida); `*` libera todas.
- [ ] **Storage dos anexos** — com `STORAGE_DRIVER=r2` (padrão de produção sugerido) as
  quatro variáveis `R2_*` são obrigatórias: a API **não sobe** sem elas. Com o driver
  `disco`, o volume `storage_data` do `docker-compose.yml` monta `/storage_local` e é o
  que mantém os arquivos entre restarts — não removê-lo nem trocar o diretório sem migrar
  o conteúdo.

---

## Padronização do módulo de solicitações (rodada de outubro/2026)

O módulo foi alinhado ao padrão dos módulos `lead`/`usuarios` + miolo transversal.

**Concluído**

- Pacote único de resposta HTTP: `adapter/http/handlers/resposta` (as três cópias de
  `resposta.go`/`resposta_test.go` foram apagadas) e helpers de teste em
  `adapter/http/apoioteste`.
- Rota unificada: `POST` **e** `GET` `/api/v1/solicitacoes/{id}/pagamento` (antes POST era
  `/pagamentos` e GET `/pagamento`).
- Ports do módulo reescritos em português (`Criar`, `Obter`, `Listar`,
  `AtualizarStatus`, `CriarPagamento`, `ObterPagamento`, `ListarPorProprietario`,
  `Remover`, `ObterPorUsuarioID`); `SolicitacaoFiltro.Aprovador` removido.
- Código morto removido: `Status.Validado()`, `domain.PaginacaoResponse`, pacote
  `adapter/storage/memoria` e índice `idx_solicitacao_aprovador` (migração `000017`).
- `tratarErro` (FK `23503` → `400`) aplicado a todos os `Delete`, inclusive os de
  `lead`/`funil`/`etapa`, que devolviam `500`.
- `escopo` + `status`: o filtro de status de query agora tem que caber na definição do
  escopo (`aprovacao` só aceita `pendente_aprovacao`, `financeiro` aceita qualquer exceto
  pendente) — antes um aprovador conseguia enxergar pagamentos por query string.
- Comprovante vira vínculo da solicitação em `CriarPagamento`, o que uniformiza o `DELETE
  /arquivos/{id}` em `409`, permite baixar o recibo por quem vê a solicitação e impede
  reuso em outra solicitação (`UNIQUE (arquivo_id)`, migração `000016`).
- `validarComprovante` passou a comparar a solicitação dona do vínculo: o comprovante da
  **mesma** solicitação não é mais rejeitado como "já vinculado".
- `atualizarStatusComGuarda` só escreve colunas de transição (valor, prazo, observação e
  forma de pagamento ficaram fora do `UPDATE` condicionado ao status anterior).
- `API.md` corrigido: valor aceito `cartao` (sem acento), `403` sem prefixo, mensagem do
  comprovante "a outra solicitação", comportamento do `status=` por escopo e vínculo do
  comprovante nos anexos.
- Testes de handler novos (`solicitacao_handler_test.go`, `aprovador_handler_test.go`)
  e e2e ampliado (rejeição, remoção de aprovador, paginação, `400` de validação, `409` do
  comprovante, filtro de status por escopo).

**Decisões registradas**

- Rota de pagamento **singular nos dois verbos**, alinhada ao `id` único de `pagamento`.
- Nomenclatura de ports **em português em todo o módulo** — ver "Em aberto" abaixo.
- Regra única de erro: `400`/`404` **com** prefixo da sentinela (`erro de validação: X`,
  `registro não encontrado: X`), `403`/`409` **sem** prefixo, `500` fixo.

**Em aberto**

- [x] **Nomenclatura dos ports divergente entre módulos** — resolvido no T0 do
  `ROADMAP.md`: `lead`/`usuarios` traduzidos para português, alinhados a `solicitacao` e
  `estoque`.

---

## Sem pendência (verificado nesta rodada)

- **CI**: os 4 workflows cobrem `gofmt`+`staticcheck`, `go vet`, testes unitários,
  integração+e2e com Postgres e migrations, segurança e docker. Os testes e2e usam o
  próprio segredo do helper, então não dependem de `JWT_SECRET` externa.
- **Gates locais**: `gofmt -l .`, `go vet ./...`, `staticcheck@v0.8.1`, `go test ./...`,
  `go test -tags=e2e ./test/...` — todos limpos.
- **Smoke na API real**: login, `401` sem token, `403` sem perfil, `400` de validação e de
  FK em uso, `204` de exclusão, CORS preflight, `405` com `Allow`, normalização de
  paginação e fail-closed do `JWT_SECRET`.
- **Bug corrigido no smoke**: token de usuário já excluído devolvia `500`; agora `401`
  (commit `f7aa20f`).
- **Sem `TODO`/`FIXME`** no código.

---

## Commits desta rodada

```
167897e docs: documenta as rotas da api e as variaveis de ambiente
ad89943 build: expoe as variaveis de token no docker compose
f7aa20f fix: token de usuario excluido devolve 401
236e8fd test: cobre login e rotas de usuario nos testes e2e
505aece feat: autentica as rotas da api com token e perfil administrador
a13f414 feat: servico de token jwt para a api
78b7684 feat: seed de cargo administrador e usuario admin inicial
009d5ad feat: coluna administrador no cargo
06e2c01 feat: paginacao das listas de usuario e cargo
88dd74a fix: exclusao de registro em uso devolve erro de validacao
```
