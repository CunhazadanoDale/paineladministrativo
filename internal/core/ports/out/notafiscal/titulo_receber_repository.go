package notafiscal

import (
	"context"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
)

type TituloReceberRepository interface {
	GerarDeOrigem(ctx context.Context, geracao *domainreceber.Geracao) (domainreceber.ResultadoGeracao, error)
	CancelarPorOrigem(ctx context.Context, cancelamento domainreceber.CancelamentoDeOrigem) (domainreceber.ResultadoCancelamento, error)
}
