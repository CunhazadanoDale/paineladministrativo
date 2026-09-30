# Testes

## Regra da casa

**Teste unitário fica ao lado do código que ele testa. Teste com infraestrutura
vem para cá.**

```
internal/adapter/http/handslead/lead/
├── lead_handler.go
├── lead_handler_test.go   ← unitário: mesmo pacote, sem banco, sem rede
└── resposta_test.go

test/
├── helpers/                      ← suporte compartilhado
├── integration/                  ← precisa de Postgres
│   └── adapter/postgres/lead_repo_test.go
└── e2e/                          ← precisa de Postgres + HTTP
    └── adapter/http/api_test.go
```

A árvore de `test/` espelha `internal/`: o teste de
`internal/adapter/postgres` vive em `test/integration/adapter/postgres`.

Por que não jogar tudo em `test/`?

- arquivos `_test.go` não entram no binário de qualquer forma, então não há
  "poluição" em manter o unitário junto;
- o unitário no mesmo pacote alcança o que não é exportado (`mapearErro`,
  `corpoJSON`...) sem abrir a API pública só para teste;
- abrir `lead_handler.go` já mostra o `lead_handler_test.go` do lado — o
  teste some do contexto quando fica em outra árvore.

O que **não** dá para testar sem infraestrutura é justamente o que vem para
cá: queries contra Postgres, transações, constraints e o caminho inteiro
requisição → resposta.

## Comandos

```bash
go test ./...                     # unitários + integração (pula se não houver Postgres)
go test ./internal/...            # só os unitários
go test -tags=e2e ./test/...      # integração + E2E (exige Postgres)
go test ./test/... -count=1       # só os testes desta árvore
go test -run TestMover -v ./test/integration/...   # um teste específico
```

Sem Postgres os testes são **pulados** com instrução de como subir o banco —
`go test ./...` continua verde em máquina sem Docker.

## Banco de teste

Cada teste recebe **um banco novo e descartável**, criado com um nome
`mia_teste_<aleatório>` e derrubado no fim do teste. Não é luxo: o
`go test ./...` roda os pacotes em paralelo e um banco em comum faria os
testes apagarem os dados uns dos outros — o clássico teste de banco que falha
só de vez em quando.

O usuário precisa ter permissão `CREATEDB` (o `mia` do `docker-compose.yml`
é superuser, então já funciona).

| Variável de ambiente | O que faz | Padrão |
|---|---|---|
| `TEST_DATABASE_URL` | servidor e credenciais usados para criar os bancos de teste | `postgres://mia:mia@localhost:5444/mia?sslmode=disable` |

Com `TEST_DATABASE_URL` definida o banco vira **obrigatório**: se não
conectar, o teste falha em vez de pular. É o que o CI usa para não deixar a
suíte passar "verde" sem rodar nada.

As migrations de `migrations/*.up.sql` são aplicadas em uma transação única
a cada teste, então o schema sempre nasce inteiro.

## Helpers

| Função | Para que serve |
|---|---|
| `helpers.BancoDoTeste(t)` | banco novo + migrations; devolve `*sqlx.DB` |
| `helpers.CriarFunil(t, db, nome)` | fixture de funil, devolve o id |
| `helpers.CriarEtapas(t, db, funil, "A", "B")` | etapas na ordem 1..n, devolve os ids |
| `helpers.CriarLead(t, db, etapa, nome)` | fixture de lead ativo |
| `helpers.DesativarLead(t, db, id)` | marca o lead como inativo |
| `helpers.NovoServidor(t, db)` | sobe a aplicação inteira em `httptest.Server` (tag `e2e`) |

As fixtures escrevem direto no banco, sem passar pelos usecases: elas montam
o cenário do teste e não devem quebrar quando a regra de negócio mudar.

## Convenções

- **Em português**, como o resto do projeto: `TestNomeAcao_Resultado`.
- Cenário organizado em **tabela de casos** quando há variação
  (inválido/vazio/inexistente); `t.Run` para subcasos.
- `t.Fatalf` para o que impede o teste de rodar, `t.Errorf` para o que apenas
  divergiu — assim um caso ruim não esconde os demais.
- Nada de `t.Parallel()` nos testes de banco: cada teste já tem o seu banco,
  mas os pacotes compartilham o servidor e paralelizar demais só deixa a
  falha mais difícil de ler.
- Todo helper começa com `t.Helper()` para o apontar para a linha certa.

## E2E fica atrás da tag `e2e`

`test/e2e/` depende da camada HTTP e do banco, então fica atrás da tag para
`go test ./...` não precisar de servidor nenhum:

```bash
go test -tags=e2e ./test/...
```

## CI

`.github/workflows/testes.yml` roda dois jobs: os unitários (sem serviços) e
os de infraestrutura com um Postgres de serviço do GitHub Actions.
