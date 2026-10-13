package cliente

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type LinhaImportacao struct {
	Numero                 int
	Documento              string
	Nome                   string
	NomeFantasia           string
	Tipo                   string
	Grupo                  string
	UF                     string
	Cidade                 string
	EmailNF                string
	Telefone               string
	RepresentanteDocumento string
	ExigeNumeroPedidoNaNF  string
	ProibeAntecipacao      string
	Observacoes            string
}

type ImportarClientesInput struct {
	UsuarioID uuid.UUID
	Linhas    []LinhaImportacao
	Simular   bool
}

type ImportacaoUseCase interface {
	Importar(ctx context.Context, input ImportarClientesInput) (*domaincliente.RelatorioImportacao, error)
}
