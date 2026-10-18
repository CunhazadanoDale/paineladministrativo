package comissao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

const limiteDaApuracao = 500

type apurador struct {
	comissoes      portsout.ComissaoRepository
	representantes portsout.RepresentanteRepository
}

func (a apurador) apurar(ctx context.Context, filtro portsout.EntregaFiltro, usuarioID uuid.UUID) (domaincomissao.ResultadoApuracao, error) {
	filtro.PaginacaoFiltro = domain.PaginacaoFiltro{Page: 1, Size: limiteDaApuracao + 1}

	entregas, err := a.comissoes.ListarEntregasParaApurar(ctx, filtro)
	if err != nil {
		return domaincomissao.ResultadoApuracao{}, err
	}

	resultado := domaincomissao.ResultadoApuracao{Apuradas: []uuid.UUID{}, Falhas: []domaincomissao.FalhaApuracao{}}
	if len(entregas) > limiteDaApuracao {
		entregas = entregas[:limiteDaApuracao]
		resultado.RestamEntregas = true
	}

	representantes := make(map[uuid.UUID]*domaincliente.Representante)
	for _, entrega := range entregas {
		apuracao, err := a.apuracaoDaEntrega(ctx, entrega, representantes, usuarioID)
		if err == nil {
			var criada bool
			criada, err = a.comissoes.GravarApuracao(ctx, apuracao)
			if err == nil && criada {
				resultado.Apuradas = append(resultado.Apuradas, apuracao.Comissao.ID)
			}
			if err == nil && !criada {
				resultado.JaExistentes++
			}
		}
		if err == nil {
			continue
		}
		if !erroDeNegocio(err) {
			return domaincomissao.ResultadoApuracao{}, err
		}
		resultado.Falhas = append(resultado.Falhas, domaincomissao.FalhaApuracao{EntregaID: entrega.EntregaID, Motivo: err.Error()})
	}

	return resultado, nil
}

func (a apurador) apuracaoDaEntrega(
	ctx context.Context,
	entrega *domaincomissao.EntregaParaApurar,
	representantes map[uuid.UUID]*domaincliente.Representante,
	usuarioID uuid.UUID,
) (*domaincomissao.Apuracao, error) {
	var representante *domaincliente.Representante
	if !entrega.Percentual.VigentePara(entrega.RepresentanteID) {
		var carregado bool
		representante, carregado = representantes[entrega.RepresentanteID]
		if !carregado {
			var err error
			if representante, err = a.representantes.Obter(ctx, entrega.RepresentanteID); err != nil {
				return nil, err
			}
			representantes[entrega.RepresentanteID] = representante
		}
	}

	return domaincomissao.NovaApuracao(*entrega, representante, usuarioID, time.Now().UTC())
}
