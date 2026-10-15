package solicitacao

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type Classificacao struct {
	FornecedorID          *uuid.UUID
	CategoriaFinanceiraID *uuid.UUID
	Competencia           *Competencia
}

func ErroFornecedorNaoEncontrado() error {
	return domain.ErroValidacao("fornecedor não encontrado")
}

func ErroFornecedorInativo() error {
	return domain.ErroValidacao("fornecedor inativo não recebe conta a pagar")
}

func ErroCategoriaFinanceiraNaoEncontrada() error {
	return domain.ErroValidacao("categoria financeira não encontrada")
}

func ErroCategoriaFinanceiraNaoLancavel() error {
	return domain.ErroValidacao("categoria financeira deve ser uma subcategoria ativa")
}

func ErroCategoriaFinanceiraNaoEhDespesa() error {
	return domain.ErroValidacao("categoria financeira deve ser de despesa")
}
