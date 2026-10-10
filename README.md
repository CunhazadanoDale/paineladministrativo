# Painel MIA Construtora — API

API em Go do painel administrativo: leads e funis, solicitações de pagamento com aprovação e
financeiro, estoque com vitrine pública, usuários, cargos e perfis. O front fica no
repositório irmão `painelMiaConstrutora` (React + Vite).

## Requisitos

- Go 1.27.1 para desenvolvimento e lint (a versão do `go.mod`); a imagem Docker compila com 1.27.2, que traz correções de segurança da biblioteca padrão (o motivo da diferença está na `PENDENCIAS.md`)
- Docker com Compose, para o Postgres e as migrations
- [`migrate`](https://github.com/golang-migrate/migrate) se for rodar as migrations fora do Compose

## Configuração

Copie `.env.example` para `.env` e preencha. A API não sobe sem um `JWT_SECRET` de pelo
menos 32 caracteres:

```bash
openssl rand -hex 32
```

| Variável | Obrigatória | Padrão | Uso |
| --- | --- | --- | --- |
| `DATABASE_URL` | sim | — | conexão com o Postgres |
| `JWT_SECRET` | sim | — | assinatura dos tokens (mínimo 32 caracteres) |
| `PORT` | não | `8080` | porta HTTP |
| `CORS_ORIGINS` | não | vazio | origens liberadas, separadas por vírgula. Vazio não libera nenhuma; `*` libera todas |
| `JWT_EXPIRA_MINUTOS` | não | `480` | validade do token |
| `STORAGE_DRIVER` | não | `disco` | `disco` ou `r2` |
| `STORAGE_DIR` | não | `storage_local` | diretório do driver `disco` |
| `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET` | com `r2` | — | credenciais do Cloudflare R2 |

Em desenvolvimento o front usa o proxy do Vite, então `CORS_ORIGINS` só precisa ser
preenchida quando o front roda em outra origem.

## Subindo

Tudo pelo Compose (Postgres na porta `5444`, Adminer na `8084`, migrations e API na `8080`):

```bash
docker compose up -d
```

Ou só o banco pelo Compose e a API local:

```bash
docker compose up -d postgres migrate
go run ./cmd/api
```

As migrations criam um administrador inicial; as credenciais estão na `API.md`.

## Testes

```bash
gofmt -l .
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go test ./...
go test -tags=e2e ./test/...
```

Os testes de integração e E2E criam um banco descartável por teste no Postgres do Compose e
são pulados quando ele não está no ar. Detalhes em [`test/README.md`](test/README.md).

## Documentação

- [`API.md`](API.md) — contrato HTTP: rotas, perfis, corpos e erros
- [`ESTOQUE.md`](ESTOQUE.md) — escopo e decisões do módulo de estoque
- [`PENDENCIAS.md`](PENDENCIAS.md) — o que está em aberto
