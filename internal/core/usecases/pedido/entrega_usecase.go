package pedido

import (
	"context"
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

var _ portsin.EntregaUseCase = (*EntregaUsecaseImpl)(nil)

type EntregaUsecaseImpl struct {
	entregas portsout.EntregaRepository
	estoqueDoPedido
	permissoes
}

func NewEntregaUsecase(
	pedidos portsout.PedidoRepository,
	entregas portsout.EntregaRepository,
	representantes portsout.RepresentanteRepository,
	empresas portsout.EmpresaRepository,
	estoque portsout.EstoqueRepository,
	reservas portsout.ReservaRepository,
	locais portsout.LocalRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *EntregaUsecaseImpl {
	return &EntregaUsecaseImpl{
		entregas: entregas,
		estoqueDoPedido: estoqueDoPedido{
			estoque:  estoque,
			reservas: reservas,
			locais:   locais,
		},
		permissoes: permissoes{
			usuarios:       usuarios,
			cargos:         cargos,
			representantes: representantes,
			empresas:       empresas,
			pedidos:        pedidos,
		},
	}
}

func (u *EntregaUsecaseImpl) Criar(ctx context.Context, input portsin.CriarEntregaInput) (*domainpedido.Entrega, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, input.PedidoID)
	if err != nil {
		return nil, err
	}

	tipo, err := domainpedido.NovoTipoFrete(input.FreteTipo)
	if err != nil {
		return nil, err
	}
	frete, err := domainpedido.NovoFrete(tipo, input.FreteCentavos)
	if err != nil {
		return nil, err
	}

	itens, err := u.itensDaEntrega(ctx, input.Itens)
	if err != nil {
		return nil, err
	}

	existentes, err := u.entregas.ListarDoPedido(ctx, pedido.ID)
	if err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	entrega, err := pedido.NovaEntrega(input.UsuarioID, domainpedido.DadosEntrega{
		Transportadora: input.Transportadora,
		Frete:          frete,
		Previsao:       input.Previsao,
		Itens:          itens,
	}, domainpedido.ProgramadoPorItem(existentes), agora)
	if err != nil {
		return nil, err
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:         pedido,
		VersaoLida:     pedido.Versao,
		StatusAnterior: pedido.Status,
		EntregasNovas:  []*domainpedido.Entrega{entrega},
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, domainpedido.AcaoEntregaCriada, nil, pedido.Status,
				detalheDaEntrega(entrega), agora),
		},
	}); err != nil {
		return nil, err
	}

	return entrega, nil
}

func (u *EntregaUsecaseImpl) Obter(ctx context.Context, entregaID, usuarioID uuid.UUID) (*domainpedido.Entrega, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	entrega, _, err := u.entregaVisivel(ctx, perfil, entregaID)

	return entrega, err
}

func (u *EntregaUsecaseImpl) ListarDoPedido(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]*domainpedido.Entrega, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if _, err := u.pedidoVisivel(ctx, perfil, pedidoID); err != nil {
		return nil, err
	}

	return u.entregas.ListarDoPedido(ctx, pedidoID)
}

func (u *EntregaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarEntregasInput) ([]*domainpedido.Entrega, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	escopo, err := perfil.escopo(input.Empresa, nil)
	if err != nil {
		return nil, err
	}

	filtro := portsout.EntregaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Empresa:         escopo.Empresa,
		EmpresaIDs:      escopo.EmpresaIDs,
		RepresentanteID: escopo.RepresentanteID,
		PrevisaoAte:     input.PrevisaoAte,
	}
	if input.Status != "" {
		if filtro.Status, err = domainpedido.NovoStatusEntrega(input.Status); err != nil {
			return nil, err
		}
	}

	return u.entregas.Listar(ctx, filtro)
}

func (u *EntregaUsecaseImpl) RegistrarSaida(ctx context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	return u.transicionar(ctx, input, domainpedido.AcaoEntregaSaiu,
		func(pedido *domainpedido.Pedido, entrega *domainpedido.Entrega, data, agora time.Time) ([]*domainestoque.OperacaoEstoque, error) {
			if entrega.Status != domainpedido.StatusEntregaPendente {
				return nil, domain.ErroConflito(fmt.Sprintf("não é possível mudar a entrega de %q para %q", entrega.Status, domainpedido.StatusEntregaSaiu))
			}

			operacoes, err := u.baixarEntrega(ctx, pedido, entrega, input.UsuarioID, agora)
			if err != nil {
				return nil, err
			}

			return operacoes, pedido.RegistrarSaida(entrega, data, agora)
		})
}

func (u *EntregaUsecaseImpl) Concluir(ctx context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	return u.transicionar(ctx, input, domainpedido.AcaoEntregaConcluida,
		func(_ *domainpedido.Pedido, entrega *domainpedido.Entrega, data, agora time.Time) ([]*domainestoque.OperacaoEstoque, error) {
			return nil, entrega.Concluir(data, agora)
		})
}

func (u *EntregaUsecaseImpl) Cancelar(ctx context.Context, input portsin.TransicaoEntregaInput) (*domainpedido.Entrega, error) {
	return u.transicionar(ctx, input, domainpedido.AcaoEntregaCancelada,
		func(_ *domainpedido.Pedido, entrega *domainpedido.Entrega, _, agora time.Time) ([]*domainestoque.OperacaoEstoque, error) {
			return nil, entrega.Cancelar(agora)
		})
}

type mudancaDeEntrega func(pedido *domainpedido.Pedido, entrega *domainpedido.Entrega, data, agora time.Time) ([]*domainestoque.OperacaoEstoque, error)

func (u *EntregaUsecaseImpl) transicionar(
	ctx context.Context, input portsin.TransicaoEntregaInput, acao domainpedido.Acao, mudar mudancaDeEntrega,
) (*domainpedido.Entrega, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	entrega, pedido, err := u.entregaVisivel(ctx, perfil, input.EntregaID)
	if err != nil {
		return nil, err
	}
	if err := entrega.ConferirVersao(input.Versao); err != nil {
		return nil, err
	}

	agora := time.Now().UTC()
	data := input.Data
	if data.IsZero() {
		data = agora
	}

	versaoDoPedido, statusDoPedido := pedido.Versao, pedido.Status
	statusDaEntrega := entrega.Status
	operacoes, err := mudar(pedido, entrega, data, agora)
	if err != nil {
		return nil, err
	}

	var anterior *domainpedido.Status
	if pedido.Status != statusDoPedido {
		anterior = &statusDoPedido
	}

	if err := u.pedidos.Gravar(ctx, &portsout.Gravacao{
		Pedido:         pedido,
		VersaoLida:     versaoDoPedido,
		StatusAnterior: statusDoPedido,
		EntregasAlteradas: []portsout.EntregaAlterada{
			{Entrega: entrega, VersaoLida: input.Versao, StatusAnterior: statusDaEntrega},
		},
		OperacoesEstoque: operacoes,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, input.UsuarioID, acao, anterior, pedido.Status, detalheDaEntrega(entrega), agora),
		},
	}); err != nil {
		return nil, err
	}

	return entrega, nil
}

func (u *EntregaUsecaseImpl) entregaVisivel(ctx context.Context, perfil perfilDoUsuario, entregaID uuid.UUID) (*domainpedido.Entrega, *domainpedido.Pedido, error) {
	entrega, err := u.entregas.Obter(ctx, entregaID)
	if err != nil {
		return nil, nil, err
	}
	if entrega == nil {
		return nil, nil, domain.ErroNaoEncontrado("entrega não encontrada")
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, entrega.PedidoID)
	if err != nil {
		return nil, nil, err
	}

	return entrega, pedido, nil
}

func (u *EntregaUsecaseImpl) itensDaEntrega(ctx context.Context, itens []portsin.EntregaItemInput) ([]domainpedido.EntregaItemInput, error) {
	ids := make([]uuid.UUID, 0, len(itens))
	for _, item := range itens {
		ids = append(ids, item.LocalID)
	}

	locais, err := u.locais.ObterVarios(ctx, ids)
	if err != nil {
		return nil, err
	}
	porID := make(map[uuid.UUID]*domainestoque.Local, len(locais))
	for _, local := range locais {
		porID[local.ID] = local
	}

	convertidos := make([]domainpedido.EntregaItemInput, 0, len(itens))
	for _, item := range itens {
		local, ok := porID[item.LocalID]
		if !ok {
			return nil, domain.ErroValidacao("local de saída não encontrado")
		}
		convertidos = append(convertidos, domainpedido.EntregaItemInput{
			PedidoItemID: item.PedidoItemID,
			Local:        local,
			Quantidade:   item.Quantidade,
		})
	}

	return convertidos, nil
}

func detalheDaEntrega(entrega *domainpedido.Entrega) string {
	total := 0
	for _, item := range entrega.Itens {
		total += item.Quantidade
	}

	return fmt.Sprintf("entrega %s com %d unidades", entrega.ID, total)
}
