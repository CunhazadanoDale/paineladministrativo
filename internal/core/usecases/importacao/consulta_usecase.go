package importacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

var _ portsin.ConsultaUseCase = (*ConsultaUsecaseImpl)(nil)

type ConsultaUsecaseImpl struct {
	consultas portsout.ConsultaRepository
	permissoes
}

func NewConsultaUsecase(
	consultas portsout.ConsultaRepository,
	importacoes portsout.ImportacaoRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *ConsultaUsecaseImpl {
	return &ConsultaUsecaseImpl{
		consultas: consultas,
		permissoes: permissoes{
			usuarios:    usuarios,
			cargos:      cargos,
			empresas:    empresas,
			importacoes: importacoes,
		},
	}
}

func (u *ConsultaUsecaseImpl) Alertas(ctx context.Context, input portsin.AlertasInput) (*domainimportacao.Alertas, error) {
	perfil, err := u.exigeLeitura(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	folga, err := domainimportacao.NovaFolga(input.FolgaDias)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa)
	if err != nil {
		return nil, err
	}
	filtro := portsout.AlertasFiltro{Empresa: escopo.empresa, EmpresaIDs: escopo.empresaIDs}

	alocacoes, err := u.consultas.AlocacoesAbertas(ctx, filtro)
	if err != nil {
		return nil, err
	}
	pedidos, err := u.consultas.PedidosComFalta(ctx, filtro)
	if err != nil {
		return nil, err
	}

	return domainimportacao.ClassificarAlertas(folga, alocacoes, pedidos), nil
}

func (u *ConsultaUsecaseImpl) ImportacoesDoPedido(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]domainimportacao.AtendimentoDoPedido, error) {
	perfil, err := u.exigeLeitura(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	empresaID, err := u.consultas.EmpresaDoPedido(ctx, pedidoID)
	if err != nil {
		return nil, err
	}
	if empresaID == nil {
		return nil, domain.ErroNaoEncontrado("pedido não encontrado")
	}
	if !perfil.acesso.Permite(*empresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	return u.consultas.AtendimentosDoPedido(ctx, pedidoID)
}
