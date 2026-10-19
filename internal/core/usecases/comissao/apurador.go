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
	resultado := domaincomissao.ResultadoApuracao{Apuradas: []uuid.UUID{}, Falhas: []domaincomissao.FalhaApuracao{}}
	representantes := make(map[uuid.UUID]*domaincliente.Representante)
	ignorar := make([]uuid.UUID, 0)
	processadas := 0

	for processadas < limiteDaApuracao {
		if len(resultado.Falhas) >= limiteDaApuracao {
			resultado.RestamEntregas = true
			break
		}

		restantes := limiteDaApuracao - processadas
		filtro.IgnorarIDs = ignorar
		filtro.PaginacaoFiltro = domain.PaginacaoFiltro{Page: 1, Size: restantes + 1}
		entregas, err := a.comissoes.ListarEntregasParaApurar(ctx, filtro)
		if err != nil {
			return domaincomissao.ResultadoApuracao{}, err
		}

		resultado.RestamEntregas = len(entregas) > restantes
		if resultado.RestamEntregas {
			entregas = entregas[:restantes]
		}

		falhasAntes := len(resultado.Falhas)
		for _, entrega := range entregas {
			criada, comissaoID, err := a.apurarEntrega(ctx, entrega, representantes, usuarioID)
			if err != nil {
				if !erroDeNegocio(err) {
					return domaincomissao.ResultadoApuracao{}, err
				}
				resultado.Falhas = append(resultado.Falhas, domaincomissao.FalhaApuracao{
					EntregaID: entrega.EntregaID, RepresentanteID: entrega.RepresentanteID, Motivo: err.Error(),
				})
				ignorar = append(ignorar, entrega.EntregaID)
				continue
			}

			processadas++
			if criada {
				resultado.Apuradas = append(resultado.Apuradas, comissaoID)
			} else {
				resultado.JaExistentes++
			}
		}

		if len(resultado.Falhas) == falhasAntes {
			break
		}
	}

	return resultado, nil
}

func (a apurador) apurarEntrega(
	ctx context.Context,
	entrega *domaincomissao.EntregaParaApurar,
	representantes map[uuid.UUID]*domaincliente.Representante,
	usuarioID uuid.UUID,
) (bool, uuid.UUID, error) {
	var representante *domaincliente.Representante
	if !entrega.Percentual.VigentePara(entrega.RepresentanteID) {
		var carregado bool
		representante, carregado = representantes[entrega.RepresentanteID]
		if !carregado {
			var err error
			if representante, err = a.representantes.Obter(ctx, entrega.RepresentanteID); err != nil {
				return false, uuid.Nil, err
			}
			representantes[entrega.RepresentanteID] = representante
		}
	}

	apuracao, err := domaincomissao.NovaApuracao(*entrega, representante, usuarioID, time.Now().UTC())
	if err != nil {
		return false, uuid.Nil, err
	}

	criada, err := a.comissoes.GravarApuracao(ctx, apuracao)
	if err != nil {
		return false, uuid.Nil, err
	}

	return criada, apuracao.Comissao.ID, nil
}
