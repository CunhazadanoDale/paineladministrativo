package notafiscal

import (
	"context"
	"time"

	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
	"github.com/google/uuid"
)

type geradorDeTitulos struct {
	titulos portsout.TituloReceberRepository
}

func (g *geradorDeTitulos) gerar(ctx context.Context, nota *domainnotafiscal.NotaFiscal, usuarioID uuid.UUID, agora time.Time) (domainreceber.ResultadoGeracao, error) {
	geracao, err := domainreceber.NovaGeracao(
		domainreceber.OrigemNotaFiscalDuplicata, nota.EmpresaID, *nota.ClienteID, nil,
		nota.ParcelasDosTitulos(), usuarioID, agora,
	)
	if err != nil {
		return domainreceber.ResultadoGeracao{}, err
	}

	return g.titulos.GerarDeOrigem(ctx, geracao)
}

func (g *geradorDeTitulos) cancelar(ctx context.Context, nota *domainnotafiscal.NotaFiscal, usuarioID uuid.UUID, agora time.Time) (domainreceber.ResultadoCancelamento, error) {
	cancelamento, err := domainreceber.NovoCancelamentoDeOrigem(
		domainreceber.OrigemNotaFiscalDuplicata, nota.DuplicataIDs(), nota.MotivoDoCancelamento(), usuarioID, agora,
	)
	if err != nil {
		return domainreceber.ResultadoCancelamento{}, err
	}

	return g.titulos.CancelarPorOrigem(ctx, cancelamento)
}
