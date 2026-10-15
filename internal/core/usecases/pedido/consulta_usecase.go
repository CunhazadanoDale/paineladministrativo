package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

var _ portsin.ConsultaUseCase = (*ConsultaUsecaseImpl)(nil)

type ConsultaUsecaseImpl struct {
	consultas portsout.ConsultaRepository
	permissoes
}

func NewConsultaUsecase(
	consultas portsout.ConsultaRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ConsultaUsecaseImpl {
	return &ConsultaUsecaseImpl{
		consultas: consultas,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
		},
	}
}

func (u *ConsultaUsecaseImpl) Pendencias(ctx context.Context, input portsin.ConsultarPendenciasInput) ([]*domainpedido.Pendencia, error) {
	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.consultas.Pendencias(ctx, portsout.PendenciaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EscopoFiltro:    escopo,
		ClienteID:       input.ClienteID,
		ProdutoID:       input.ProdutoID,
	})
}

func (u *ConsultaUsecaseImpl) Prazos(ctx context.Context, input portsin.ConsultarPrazosInput) ([]*domainpedido.Pedido, error) {
	situacao, err := domainpedido.NovaSituacaoPrazo(input.Situacao)
	if err != nil {
		return nil, err
	}
	janela, err := domainpedido.NovaJanelaDePrazo(situacao, input.Dias, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.consultas.PorPrazo(ctx, portsout.PrazoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EscopoFiltro:    escopo,
		Janela:          janela,
	})
}

func (u *ConsultaUsecaseImpl) Cancelamentos(ctx context.Context, input portsin.ConsultarCancelamentosInput) ([]*domainpedido.Pedido, error) {
	if input.De != nil && input.Ate != nil && domainpedido.DataDe(*input.De).After(domainpedido.DataDe(*input.Ate)) {
		return nil, domain.ErroValidacao("data inicial do período não pode ser depois da final")
	}

	escopo, err := u.escopoDoUsuario(ctx, input.UsuarioID, input.Empresa)
	if err != nil {
		return nil, err
	}

	return u.consultas.Cancelamentos(ctx, portsout.CancelamentoFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		EscopoFiltro:    escopo,
		ComDevolucao:    input.ComDevolucao,
		De:              input.De,
		Ate:             input.Ate,
	})
}

func (u *ConsultaUsecaseImpl) escopoDoUsuario(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) (portsout.EscopoFiltro, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return portsout.EscopoFiltro{}, err
	}

	return perfil.escopo(empresa, nil)
}
