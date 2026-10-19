package comissao

import (
	"context"
	"errors"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

var _ portsin.ComissaoUseCase = (*ComissaoUsecaseImpl)(nil)

type ComissaoUsecaseImpl struct {
	comissoes portsout.ComissaoRepository
	apuracao  apurador
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
		apuracao:  apurador{comissoes: comissoes, representantes: representantes},
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

	return u.apuracao.apurar(ctx, filtro, input.UsuarioID)
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
		Aberta:          input.Aberta,
	}
	if filtro.Competencia, err = competenciaOpcional(input.Competencia); err != nil {
		return nil, err
	}

	return u.comissoes.Listar(ctx, filtro)
}

func (u *ComissaoUsecaseImpl) Resumo(ctx context.Context, input portsin.ResumoInput) (domaincomissao.ResumoComissoes, error) {
	perfil, err := u.exigeLeituraDeComissoes(ctx, input.UsuarioID)
	if err != nil {
		return domaincomissao.ResumoComissoes{}, err
	}

	escopo, err := perfil.escopo(input.Empresa, input.RepresentanteID)
	if err != nil {
		return domaincomissao.ResumoComissoes{}, err
	}

	competencia, err := domaincomissao.NovaCompetencia(input.Competencia)
	if err != nil {
		return domaincomissao.ResumoComissoes{}, err
	}

	linhas, err := u.comissoes.Resumo(ctx, portsout.ResumoFiltro{
		Competencia:     competencia,
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		RepresentanteID: escopo.representanteID,
	})
	if err != nil {
		return domaincomissao.ResumoComissoes{}, err
	}

	return domaincomissao.ResumoComissoes{Competencia: competencia, Linhas: linhas}, nil
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
