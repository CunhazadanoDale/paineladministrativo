package fornecedor

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type LinhaImportacao struct {
	Numero                   int
	Pais                     string
	Documento                string
	IdentificacaoEstrangeira string
	Nome                     string
	NomeFantasia             string
	MoedaPadrao              string
	UF                       string
	Cidade                   string
	ContatoNome              string
	Email                    string
	Telefone                 string
	Observacoes              string
}

type ImportarFornecedoresInput struct {
	UsuarioID uuid.UUID
	Linhas    []LinhaImportacao
	Simular   bool
}

type ImportacaoUseCase interface {
	Importar(ctx context.Context, input ImportarFornecedoresInput) (*domain.RelatorioImportacao, error)
}
