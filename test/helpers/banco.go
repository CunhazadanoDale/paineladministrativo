package helpers

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const DSNPadrao = "postgres://mia:mia@localhost:5444/mia?sslmode=disable"

const VariavelDSN = "TEST_DATABASE_URL"

const prefixoBancoDeTeste = "mia_teste_"

func BancoDoTeste(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := os.Getenv(VariavelDSN)
	obrigatorio := dsn != ""
	if !obrigatorio {
		dsn = DSNPadrao
	}

	manutencao, err := comBanco(dsn, "postgres")
	if err != nil {
		t.Fatalf("DSN inválida em %s: %v", VariavelDSN, err)
	}

	ctx := context.Background()
	base, err := postgres.ConnectionDB(manutencao)
	if err != nil {
		if obrigatorio {
			t.Fatalf("não consegui conectar no Postgres definido em %s: %v", VariavelDSN, err)
		}
		t.Skipf("Postgres indisponível em %s — suba com `docker compose up -d postgres` ou defina %s: %v", dsn, VariavelDSN, err)
	}
	t.Cleanup(func() {
		_ = base.Close()
	})

	nome := prefixoBancoDeTeste + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := base.ExecContext(ctx, "CREATE DATABASE "+nome); err != nil {
		t.Fatalf("não consegui criar o banco de teste %q: %v (o usuário de %s precisa ter permissão CREATEDB)", nome, err, VariavelDSN)
	}

	banco, err := postgres.ConnectionDB(trocarBanco(dsn, nome))
	if err != nil {
		_, _ = base.ExecContext(ctx, "DROP DATABASE "+nome+" WITH (FORCE)")
		t.Fatalf("não consegui conectar no banco de teste %q: %v", nome, err)
	}
	t.Cleanup(func() {
		_ = banco.Close()
		_, _ = base.ExecContext(ctx, "DROP DATABASE "+nome+" WITH (FORCE)")
	})

	aplicarMigrations(t, banco)

	return banco
}

func aplicarMigrations(t *testing.T, banco *sqlx.DB) {
	t.Helper()

	ctx := context.Background()

	transacao, err := banco.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("não consegui iniciar a transação das migrations: %v", err)
	}
	sucesso := false
	defer func() {
		if !sucesso {
			_ = transacao.Rollback()
		}
	}()

	arquivos, err := filepath.Glob(filepath.Join(raizDoProjeto(t), "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("não consegui procurar as migrations: %v", err)
	}
	if len(arquivos) == 0 {
		t.Fatal("nenhum arquivo migrations/*.up.sql encontrado")
	}

	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("não consegui ler a migration %s: %v", filepath.Base(caminho), err)
		}

		if _, err := transacao.ExecContext(ctx, string(conteudo)); err != nil {
			t.Fatalf("migration %s falhou: %v", filepath.Base(caminho), err)
		}
	}

	if err := transacao.Commit(); err != nil {
		t.Fatalf("não consegui confirmar as migrations: %v", err)
	}
	sucesso = true
}

func comBanco(dsn, nome string) (string, error) {
	endereco, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	endereco.Path = "/" + nome

	return endereco.String(), nil
}

func trocarBanco(dsn, nome string) string {
	endereco, err := comBanco(dsn, nome)
	if err != nil {
		return dsn
	}

	return endereco
}

func raizDoProjeto(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("não consegui descobrir o diretório atual: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		pai := filepath.Dir(dir)
		if pai == dir {
			t.Fatalf("não encontrei o go.mod subindo a partir de %s", dir)
		}
		dir = pai
	}
}
