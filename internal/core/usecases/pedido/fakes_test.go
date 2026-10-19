package pedido_test

import (
	"context"
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios       map[uuid.UUID]*domainusuarios.Usuario
	cargos         map[uuid.UUID]*domainusuarios.Cargo
	empresas       map[uuid.UUID]*domainempresa.Empresa
	acessos        map[uuid.UUID]domainempresa.Acesso
	representantes map[uuid.UUID]*domaincliente.Representante
	clientes       map[uuid.UUID]*domaincliente.Cliente
	obras          map[uuid.UUID]*domaincliente.Obra
	produtos       map[uuid.UUID]*domainestoque.Produto
	locais         map[uuid.UUID]*domainestoque.Local
	estoques       map[uuid.UUID]*domainestoque.EstoqueProduto
	reservas       map[uuid.UUID]*domainestoque.Reserva
	pedidos        map[uuid.UUID]*domainpedido.Pedido
	entregas       map[uuid.UUID]*domainpedido.Entrega
	historicos     []*domainpedido.Historico
	sequencias     map[uuid.UUID]int
	gravacoes      int
	precosTabela   map[string]int64
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:       map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:         map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:       map[uuid.UUID]*domainempresa.Empresa{},
		acessos:        map[uuid.UUID]domainempresa.Acesso{},
		representantes: map[uuid.UUID]*domaincliente.Representante{},
		clientes:       map[uuid.UUID]*domaincliente.Cliente{},
		obras:          map[uuid.UUID]*domaincliente.Obra{},
		produtos:       map[uuid.UUID]*domainestoque.Produto{},
		locais:         map[uuid.UUID]*domainestoque.Local{},
		estoques:       map[uuid.UUID]*domainestoque.EstoqueProduto{},
		reservas:       map[uuid.UUID]*domainestoque.Reserva{},
		pedidos:        map[uuid.UUID]*domainpedido.Pedido{},
		entregas:       map[uuid.UUID]*domainpedido.Entrega{},
		sequencias:     map[uuid.UUID]int{},
		precosTabela:   map[string]int64{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

func copiarPedido(pedido *domainpedido.Pedido) *domainpedido.Pedido {
	copia := *pedido
	copia.Itens = make([]*domainpedido.Item, 0, len(pedido.Itens))
	for _, item := range pedido.Itens {
		copia.Itens = append(copia.Itens, copiar(item))
	}

	return &copia
}

func copiarEntrega(entrega *domainpedido.Entrega) *domainpedido.Entrega {
	copia := *entrega
	copia.Itens = make([]*domainpedido.EntregaItem, 0, len(entrega.Itens))
	for _, item := range entrega.Itens {
		copia.Itens = append(copia.Itens, copiar(item))
	}

	return &copia
}

func copiarEstoque(estoque *domainestoque.EstoqueProduto) *domainestoque.EstoqueProduto {
	return &domainestoque.EstoqueProduto{
		Produto:      copiar(estoque.Produto),
		Saldos:       maps.Clone(estoque.Saldos),
		Lotes:        slices.Clone(estoque.Lotes),
		TotalAlocado: estoque.TotalAlocado,
	}
}

type repoUsuarios struct{ *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return copiar(r.usuarios[id]), nil
}

type repoCargos struct{ *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return copiar(r.cargos[id]), nil
}

type repoEmpresas struct{ *memoria }

func (r repoEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return copiar(r.empresas[id]), nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.acessos[usuarioID], nil
}

type repoRepresentantes struct{ *memoria }

func (r repoRepresentantes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Representante, error) {
	return copiar(r.representantes[id]), nil
}

func (r repoRepresentantes) ObterPorUsuario(_ context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	for _, representante := range r.representantes {
		if representante.UsuarioID != nil && *representante.UsuarioID == usuarioID {
			return copiar(representante), nil
		}
	}

	return nil, nil
}

type repoClientes struct{ *memoria }

func (r repoClientes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	return copiar(r.clientes[id]), nil
}

type repoObras struct{ *memoria }

func (r repoObras) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Obra, error) {
	return copiar(r.obras[id]), nil
}

type repoProdutos struct{ *memoria }

func (r repoProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	return copiar(r.produtos[id]), nil
}

type repoPrecos struct{ *memoria }

func (r repoPrecos) PrecoVigente(_ context.Context, produtoID uuid.UUID, modelo domainpedido.Modelo, _ time.Time) (*int64, error) {
	preco, ok := r.precosTabela[produtoID.String()+modelo.String()]
	if !ok {
		return nil, nil
	}

	return &preco, nil
}

type repoLocais struct{ *memoria }

func (r repoLocais) ObterVarios(_ context.Context, ids []uuid.UUID) ([]*domainestoque.Local, error) {
	locais := make([]*domainestoque.Local, 0, len(ids))
	for _, id := range ids {
		if local, ok := r.locais[id]; ok {
			locais = append(locais, copiar(local))
		}
	}

	return locais, nil
}

func (r repoLocais) AtivosDaEmpresa(_ context.Context, empresaID uuid.UUID) ([]*domainestoque.Local, error) {
	locais := make([]*domainestoque.Local, 0)
	for _, local := range r.locais {
		if local.EmpresaID == empresaID && local.Ativo {
			locais = append(locais, copiar(local))
		}
	}
	sort.Slice(locais, func(i, j int) bool { return locais[i].Nome < locais[j].Nome })

	return locais, nil
}

type repoEstoque struct{ *memoria }

func (r repoEstoque) Carregar(_ context.Context, produtoID uuid.UUID, _ []uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	estoque, ok := r.estoques[produtoID]
	if !ok {
		return nil, nil
	}

	return copiarEstoque(estoque), nil
}

type repoReservas struct{ *memoria }

func (r repoReservas) AtivasPorOrigem(_ context.Context, origemIDs []uuid.UUID) ([]*domainestoque.Reserva, error) {
	reservas := make([]*domainestoque.Reserva, 0)
	for _, reserva := range r.reservas {
		if reserva.Ativa() && reserva.OrigemID != nil && slices.Contains(origemIDs, *reserva.OrigemID) {
			reservas = append(reservas, copiar(reserva))
		}
	}
	sort.Slice(reservas, func(i, j int) bool { return reservas[i].CriadoEm.Before(reservas[j].CriadoEm) })

	return reservas, nil
}

type repoEntregas struct{ *memoria }

func (r repoEntregas) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Entrega, error) {
	entrega, ok := r.entregas[id]
	if !ok {
		return nil, nil
	}

	return copiarEntrega(entrega), nil
}

func (r repoEntregas) ListarDoPedido(_ context.Context, pedidoID uuid.UUID) ([]*domainpedido.Entrega, error) {
	entregas := make([]*domainpedido.Entrega, 0)
	for _, entrega := range r.entregas {
		if entrega.PedidoID == pedidoID {
			entregas = append(entregas, copiarEntrega(entrega))
		}
	}
	sort.Slice(entregas, func(i, j int) bool { return entregas[i].CriadoEm.Before(entregas[j].CriadoEm) })

	return entregas, nil
}

func (r repoEntregas) Listar(_ context.Context, filtro portsout.EntregaFiltro) ([]*domainpedido.Entrega, error) {
	entregas := make([]*domainpedido.Entrega, 0)
	for _, entrega := range r.entregas {
		pedido := r.pedidos[entrega.PedidoID]
		switch {
		case filtro.Empresa != nil && pedido.EmpresaID != *filtro.Empresa:
			continue
		case filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, pedido.EmpresaID):
			continue
		case filtro.RepresentanteID != nil && (pedido.RepresentanteID == nil || *pedido.RepresentanteID != *filtro.RepresentanteID):
			continue
		case filtro.Status != "" && entrega.Status != filtro.Status:
			continue
		}
		entregas = append(entregas, copiarEntrega(entrega))
	}

	return entregas, nil
}

type repoPedidos struct{ *memoria }

func (r repoPedidos) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Pedido, error) {
	pedido, ok := r.pedidos[id]
	if !ok {
		return nil, nil
	}

	return copiarPedido(pedido), nil
}

func (r repoPedidos) Listar(_ context.Context, filtro portsout.PedidoFiltro) ([]*domainpedido.Pedido, error) {
	pedidos := make([]*domainpedido.Pedido, 0)
	for _, pedido := range r.pedidos {
		switch {
		case filtro.Empresa != nil && pedido.EmpresaID != *filtro.Empresa:
			continue
		case filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, pedido.EmpresaID):
			continue
		case filtro.RepresentanteID != nil && (pedido.RepresentanteID == nil || *pedido.RepresentanteID != *filtro.RepresentanteID):
			continue
		case filtro.ClienteID != nil && pedido.ClienteID != *filtro.ClienteID:
			continue
		case filtro.Status != "" && pedido.Status != filtro.Status:
			continue
		case filtro.Modelo != "" && pedido.Modelo != filtro.Modelo:
			continue
		}
		pedidos = append(pedidos, copiarPedido(pedido))
	}

	return pedidos, nil
}

func (r repoPedidos) ListarHistorico(_ context.Context, pedidoID uuid.UUID) ([]*domainpedido.Historico, error) {
	historicos := make([]*domainpedido.Historico, 0)
	for _, historico := range r.historicos {
		if historico.PedidoID == pedidoID {
			historicos = append(historicos, copiar(historico))
		}
	}

	return historicos, nil
}

func (r repoPedidos) Gravar(_ context.Context, gravacao *portsout.Gravacao) error {
	pedido := copiarPedido(gravacao.Pedido)
	if gravacao.Novo {
		pedido.Numero = r.sequencias[pedido.EmpresaID] + 1
	} else {
		atual, ok := r.pedidos[pedido.ID]
		if !ok {
			return domain.ErroNaoEncontrado("pedido não encontrado")
		}
		if atual.Versao != gravacao.VersaoLida || atual.Status != gravacao.StatusAnterior {
			return domainpedido.ErroVersaoDivergente()
		}
		pedido.Versao = atual.Versao + 1
	}

	entregas := make(map[uuid.UUID]*domainpedido.Entrega)
	for _, entrega := range gravacao.EntregasNovas {
		entregas[entrega.ID] = copiarEntrega(entrega)
	}
	for _, alterada := range gravacao.EntregasAlteradas {
		atual, ok := r.entregas[alterada.Entrega.ID]
		if !ok {
			return domain.ErroNaoEncontrado("entrega não encontrada")
		}
		if atual.Versao != alterada.VersaoLida || atual.Status != alterada.StatusAnterior {
			return domainpedido.ErroVersaoEntregaDivergente()
		}
		nova := copiarEntrega(alterada.Entrega)
		nova.Versao = atual.Versao + 1
		entregas[nova.ID] = nova
	}

	estoques := make(map[uuid.UUID]*domainestoque.EstoqueProduto)
	reservas := make(map[uuid.UUID]*domainestoque.Reserva)
	for _, operacao := range gravacao.OperacoesEstoque {
		estoque, ok := estoques[operacao.ProdutoID]
		if !ok {
			original, existe := r.estoques[operacao.ProdutoID]
			if !existe {
				return domain.ErroValidacao("produto sem estoque")
			}
			estoque = copiarEstoque(original)
			estoques[operacao.ProdutoID] = estoque
		}
		if err := estoque.Aplicar(operacao); err != nil {
			return domain.ErroConflito("estoque do produto alterado por outra operação, recarregue e tente novamente")
		}
		if operacao.ReservaCriada != nil {
			reservas[operacao.ReservaCriada.ID] = copiar(operacao.ReservaCriada)
		}
		if encerrada := operacao.ReservaEncerrada; encerrada != nil {
			atual := reservas[encerrada.ID]
			if atual == nil {
				atual = r.reservas[encerrada.ID]
			}
			if atual == nil || !atual.Ativa() {
				return domain.ErroConflito("reserva já encerrada")
			}
			fechada := copiar(atual)
			fechada.Status = encerrada.Status
			fechada.EncerradaEm = encerrada.EncerradaEm
			reservas[fechada.ID] = fechada
		}
	}

	if gravacao.Novo {
		r.sequencias[pedido.EmpresaID] = pedido.Numero
	}
	r.pedidos[pedido.ID] = pedido
	maps.Copy(r.entregas, entregas)
	maps.Copy(r.estoques, estoques)
	maps.Copy(r.reservas, reservas)
	for _, historico := range gravacao.Historicos {
		r.historicos = append(r.historicos, copiar(historico))
	}
	r.gravacoes++

	gravacao.Pedido.Numero = pedido.Numero
	gravacao.Pedido.Versao = pedido.Versao
	for _, alterada := range gravacao.EntregasAlteradas {
		alterada.Entrega.Versao = entregas[alterada.Entrega.ID].Versao
	}

	return nil
}
