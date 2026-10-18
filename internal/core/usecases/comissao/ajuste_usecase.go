package comissao

import (
	"context"
	"time"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
)

var _ portsin.AjusteUseCase = (*AjusteUsecaseImpl)(nil)

type AjusteUsecaseImpl struct {
	comissoes portsout.ComissaoRepository
	pedidos   portsout.PedidoRepository
	permissoes
}

func NewAjusteUsecase(
	comissoes portsout.ComissaoRepository,
	pedidos portsout.PedidoRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *AjusteUsecaseImpl {
	return &AjusteUsecaseImpl{
		comissoes: comissoes,
		pedidos:   pedidos,
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
		},
	}
}

func (u *AjusteUsecaseImpl) Criar(ctx context.Context, input portsin.CriarAjusteInput) (*domaincomissao.Ajuste, error) {
	perfil, err := u.exigeFinanceiro(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	ajuste, err := domaincomissao.NovoAjuste(input.Dados, input.UsuarioID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if !perfil.acesso.Permite(ajuste.EmpresaID) {
		return nil, erroSemAcessoAEmpresa()
	}

	representante, err := u.representantes.Obter(ctx, ajuste.RepresentanteID)
	if err != nil {
		return nil, err
	}
	if representante == nil {
		return nil, domaincomissao.ErroRepresentanteDoAjusteNaoEncontrado()
	}

	if ajuste.PedidoID != nil {
		pedido, err := u.pedidos.Obter(ctx, *ajuste.PedidoID)
		if err != nil {
			return nil, err
		}
		if pedido == nil {
			return nil, domaincomissao.ErroPedidoDoAjusteNaoEncontrado()
		}
		if pedido.EmpresaID != ajuste.EmpresaID || pedido.RepresentanteID == nil || *pedido.RepresentanteID != ajuste.RepresentanteID {
			return nil, domaincomissao.ErroPedidoDeOutraEmpresaOuRepresentante()
		}
	}

	if err := u.comissoes.CriarAjuste(ctx, ajuste); err != nil {
		return nil, err
	}

	return ajuste, nil
}

func (u *AjusteUsecaseImpl) Listar(ctx context.Context, input portsin.ListarAjustesInput) ([]*domaincomissao.Ajuste, error) {
	perfil, err := u.exigeLeituraDeComissoes(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa, input.RepresentanteID)
	if err != nil {
		return nil, err
	}

	filtro := portsout.AjusteFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.empresa,
		EmpresaIDs:      escopo.empresaIDs,
		RepresentanteID: escopo.representanteID,
		PedidoID:        input.PedidoID,
		Aberto:          input.Aberto,
	}
	if filtro.Competencia, err = competenciaOpcional(input.Competencia); err != nil {
		return nil, err
	}

	return u.comissoes.ListarAjustes(ctx, filtro)
}
