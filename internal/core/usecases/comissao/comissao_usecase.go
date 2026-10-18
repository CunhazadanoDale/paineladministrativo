package comissao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

const limiteDaApuracao = 500

var _ portsin.ComissaoUseCase = (*ComissaoUsecaseImpl)(nil)

type ComissaoUsecaseImpl struct {
	comissoes portsout.ComissaoRepository
	permissoes
}

func NewComissaoUsecase(
	comissoes portsout.ComissaoRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ComissaoUsecaseImpl {
	return &ComissaoUsecaseImpl{
		comissoes: comissoes,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
		},
	}
}

func (u *ComissaoUsecaseImpl) Apurar(ctx context.Context, input portsin.ApurarInput) (domaincomissao.ResultadoApuracao, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return domaincomissao.ResultadoApuracao{}, err
	}

	filtro, err := filtroDeEntregas(perfil, input.Empresa, input.Competencia)
	if err != nil {
		return domaincomissao.ResultadoApuracao{}, err
	}
	filtro.PaginacaoFiltro = domain.PaginacaoFiltro{Page: 1, Size: limiteDaApuracao + 1}

	entregas, err := u.comissoes.ListarEntregasParaApurar(ctx, filtro)
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
		apuracao, err := u.apuracaoDaEntrega(ctx, entrega, representantes, input.UsuarioID)
		if err == nil {
			var criada bool
			criada, err = u.comissoes.GravarApuracao(ctx, apuracao)
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

func (u *ComissaoUsecaseImpl) Pendencias(ctx context.Context, input portsin.PendenciasInput) ([]*domaincomissao.EntregaParaApurar, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	filtro, err := filtroDeEntregas(perfil, input.Empresa, input.Competencia)
	if err != nil {
		return nil, err
	}
	filtro.PaginacaoFiltro = input.Filtro.Normalizada()

	return u.comissoes.ListarEntregasParaApurar(ctx, filtro)
}

func (u *ComissaoUsecaseImpl) Listar(ctx context.Context, input portsin.ListarComissoesInput) ([]*domaincomissao.Comissao, error) {
	perfil, err := u.exigeLeituraDeComissoes(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa, input.RepresentanteID)
	if err != nil {
		return nil, err
	}

	filtro := portsout.ComissaoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		RepresentanteID: escopo.representanteID,
		PedidoID:        input.PedidoID,
	}
	if filtro.Competencia, err = competenciaOpcional(input.Competencia); err != nil {
		return nil, err
	}

	return u.comissoes.Listar(ctx, filtro)
}

func (u *ComissaoUsecaseImpl) apuracaoDaEntrega(
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
			if representante, err = u.representantes.Obter(ctx, entrega.RepresentanteID); err != nil {
				return nil, err
			}
			representantes[entrega.RepresentanteID] = representante
		}
	}

	return domaincomissao.NovaApuracao(*entrega, representante, usuarioID, time.Now().UTC())
}

func filtroDeEntregas(perfil perfilDoUsuario, empresa *uuid.UUID, competencia string) (portsout.EntregaFiltro, error) {
	escopo, err := perfil.escopo(empresa, nil)
	if err != nil {
		return portsout.EntregaFiltro{}, err
	}

	filtro := portsout.EntregaFiltro{Empresa: escopo.empresa, EmpresaIDs: escopo.empresaIDs}
	ate, err := competenciaOpcional(competencia)
	if err != nil || ate == nil {
		return filtro, err
	}

	limite := ate.Seguinte().Inicio()
	filtro.SaidaAntesDe = &limite

	return filtro, nil
}

func competenciaOpcional(texto string) (*domaincomissao.Competencia, error) {
	if strings.TrimSpace(texto) == "" {
		return nil, nil
	}

	competencia, err := domaincomissao.NovaCompetencia(texto)
	if err != nil {
		return nil, err
	}

	return &competencia, nil
}

func erroDeNegocio(err error) bool {
	return errors.Is(err, domain.ErrValidacao) || errors.Is(err, domain.ErrConflito) || errors.Is(err, domain.ErrNotFound)
}
