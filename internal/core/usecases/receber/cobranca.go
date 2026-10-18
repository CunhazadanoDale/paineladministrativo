package receber

import (
	"context"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

type cobranca struct {
	contas portsout.ContaBancariaRepository
}

func (c *cobranca) exigirContaDeCobranca(ctx context.Context, contaID, empresaID uuid.UUID) error {
	conta, err := c.contas.Obter(ctx, contaID)
	if err != nil {
		return err
	}
	if conta == nil {
		return domainreceber.ErroContaBancariaNaoEncontrada()
	}
	if conta.EmpresaID != empresaID {
		return domainreceber.ErroContaBancariaDeOutraEmpresa()
	}
	if !conta.Ativa {
		return domainreceber.ErroContaBancariaInativa()
	}

	return nil
}
