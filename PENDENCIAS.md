# Pendências

O que ficou em aberto depois da rodada de paginação, autenticação, seed e documentação
(outubro/2026). Itens em ordem sugerida de ataque — atualize a coluna **Status** conforme
forem resolvidos.

---

## Próximos passos

| # | Item | Por quê | Esforço | Status |
| --- | --- | --- | --- | --- |
| 1 | **README completo** | O `README.md` tem 2 bytes. Quem clona não descobre que `JWT_SECRET` é obrigatório (a API não sobe sem), como subir banco/migrations, os comandos de teste nem que existe a `API.md` | Baixo | pendente |
| 2 | **Rotas de sessão** | Usuário comum consegue logar, mas não vê o próprio perfil nem troca a própria senha — todas as rotas `/usuarios` são de administrador. Criar `GET /api/v1/sessao` (perfil do token) e `POST /api/v1/sessao/senha` (exigindo a senha atual) | Baixo | pendente |
| 3 | **Senha temporária no 1º login** | A decisão tomada no seed foi "senha inicial fixa, trocar no primeiro login", mas não existe flag de senha temporária nem exigência de troca. Alternativa: aceitar o fluxo manual e apenas manter a orientação na `API.md` | Médio | pendente |
| 4 | **Rate limit no login** | `POST /usuarios/autenticar` não limita tentativas: é possível testar senhas em loop. A mensagem genérica `email ou senha inválidos` esconde o que existe, mas não limita a frequência. Limite simples por IP/e-mail, em memória | Médio | pendente |
| 5 | **Total de páginas na paginação** | Hoje a resposta ecoa só `pagina`/`tamanho` (decisão registrada). Se o front for fazer navegação "página X de Y", precisa de contagem — muda o contrato das listas | Baixo | pendente |

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

Se o item 4 for feito, decidir se fica na aplicação (simples, zera ao reiniciar) ou no
nginx/Caddy da frente (sobrevive a restarts, mas exige mudar o deploy).

---

## Fora do repositório

- [ ] **Frontend precisa enviar `Bearer`** — todas as rotas de negócio passaram a exigir
  token. Sem isso o painel inteiro começa a devolver `401`. O fluxo e os exemplos estão
  na `API.md`.
- [ ] **Snippet de client** — gerar um exemplo pronto (fetch/axios) de login + chamada
  autenticada para colar no front.
- [ ] **`JWT_SECRET` no ambiente de produção** — o `docker compose up api` falha de
  propósito se não houver `.env` com o segredo (fail-closed). Garantir que o ambiente
  tenha um valor gerado (`openssl rand -hex 32`), nunca o de desenvolvimento.
- [ ] **`CORS_ORIGINS` explícito em produção** — vazio ou `*` libera qualquer origem.

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
