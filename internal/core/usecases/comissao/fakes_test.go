package comissao_test

import (
	"context"
	"errors"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

var (
	_ portsout.ComissaoRepository            = repoComissoes{}
	_ portsout.PedidoRepository              = repoPedidos{}
	_ portsout.RepresentanteRepository       = repoRepresentantes{}
	_ portsout.EmpresaRepository             = repoEmpresas{}
	_ portsout.UsuarioRepository             = repoUsuarios{}
	_ portsout.CargoRepository               = repoCargos{}
	_ portsout.CategoriaFinanceiraRepository = repoCategorias{}
)

var errBancoFora = errors.New("banco fora do ar")

type memoria struct {
	usuarios       map[uuid.UUID]*domainusuarios.Usuario
	cargos         map[uuid.UUID]*domainusuarios.Cargo
	acessos        map[uuid.UUID]domainempresa.Acesso
	representantes map[uuid.UUID]*domaincliente.Representante
	pedidos        map[uuid.UUID]*domainpedido.Pedido
	percentuais    map[uuid.UUID]*domaincomissao.PercentualPedido
	entregas       []*domaincomissao.EntregaParaApurar
	comissoes      []*domaincomissao.Comissao
	ajustes        []*domaincomissao.Ajuste
	empresas       map[uuid.UUID]*domainempresa.Empresa
	categorias     map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	saldos         []domaincomissao.SaldoAberto
	fechamentos    []*portsout.GravacaoFechamento
	falhaNoFechar  map[uuid.UUID]error
	saidas         map[uuid.UUID]bool

	ultimoFiltroEntregas portsout.EntregaFiltro
	ultimoFiltro         portsout.ComissaoFiltro
	ultimoFiltroAjustes  portsout.AjusteFiltro
	ultimoFiltroSaldos   portsout.SaldoFiltro
	ultimoFiltroFechados portsout.FechamentoFiltro
	ultimoFiltroResumo   portsout.ResumoFiltro
	resumo               []domaincomissao.LinhaResumo
	versaoLida           *int
	falhaNaGravacao      map[uuid.UUID]error
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:        make(map[uuid.UUID]*domainusuarios.Usuario),
		cargos:          make(map[uuid.UUID]*domainusuarios.Cargo),
		acessos:         make(map[uuid.UUID]domainempresa.Acesso),
		representantes:  make(map[uuid.UUID]*domaincliente.Representante),
		pedidos:         make(map[uuid.UUID]*domainpedido.Pedido),
		percentuais:     make(map[uuid.UUID]*domaincomissao.PercentualPedido),
		saidas:          make(map[uuid.UUID]bool),
		falhaNaGravacao: make(map[uuid.UUID]error),
		empresas:        make(map[uuid.UUID]*domainempresa.Empresa),
		categorias:      make(map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira),
		falhaNoFechar:   make(map[uuid.UUID]error),
	}
}

type repoComissoes struct{ m *memoria }

func (r repoComissoes) ObterPercentual(_ context.Context, pedidoID uuid.UUID) (*domaincomissao.PercentualPedido, error) {
	percentual, ok := r.m.percentuais[pedidoID]
	if !ok {
		return nil, nil
	}

	copia := *percentual

	return &copia, nil
}

func (r repoComissoes) DefinirPercentual(_ context.Context, percentual *domaincomissao.PercentualPedido, versaoLida *int) error {
	if r.m.saidas[percentual.PedidoID] {
		return domaincomissao.ErroPercentualTravado()
	}
	r.m.versaoLida = versaoLida
	if versaoLida != nil {
		percentual.Versao = *versaoLida + 1
	}

	copia := *percentual
	r.m.percentuais[percentual.PedidoID] = &copia

	return nil
}

func (r repoComissoes) ListarEntregasParaApurar(_ context.Context, filtro portsout.EntregaFiltro) ([]*domaincomissao.EntregaParaApurar, error) {
	r.m.ultimoFiltroEntregas = filtro

	apuradas := make(map[uuid.UUID]bool, len(r.m.comissoes))
	for _, comissao := range r.m.comissoes {
		apuradas[comissao.EntregaID] = true
	}

	for _, id := range filtro.IgnorarIDs {
		apuradas[id] = true
	}

	resultado := make([]*domaincomissao.EntregaParaApurar, 0)
	for _, entrega := range r.m.entregas {
		if apuradas[entrega.EntregaID] || len(resultado) >= filtro.Size {
			continue
		}
		resultado = append(resultado, entrega)
	}

	return resultado, nil
}

func (r repoComissoes) GravarApuracao(_ context.Context, apuracao *domaincomissao.Apuracao) (bool, error) {
	if err := r.m.falhaNaGravacao[apuracao.Comissao.EntregaID]; err != nil {
		return false, err
	}
	for _, comissao := range r.m.comissoes {
		if comissao.EntregaID == apuracao.Comissao.EntregaID {
			return false, nil
		}
	}
	if apuracao.PercentualNovo {
		r.m.percentuais[apuracao.Percentual.PedidoID] = apuracao.Percentual
	}
	r.m.comissoes = append(r.m.comissoes, apuracao.Comissao)

	return true, nil
}

func (r repoComissoes) Listar(_ context.Context, filtro portsout.ComissaoFiltro) ([]*domaincomissao.Comissao, error) {
	r.m.ultimoFiltro = filtro

	return r.m.comissoes, nil
}

func (r repoComissoes) CriarAjuste(_ context.Context, ajuste *domaincomissao.Ajuste) error {
	r.m.ajustes = append(r.m.ajustes, ajuste)

	return nil
}

func (r repoComissoes) ListarAjustes(_ context.Context, filtro portsout.AjusteFiltro) ([]*domaincomissao.Ajuste, error) {
	r.m.ultimoFiltroAjustes = filtro

	return r.m.ajustes, nil
}

func (r repoComissoes) SaldosAbertos(_ context.Context, filtro portsout.SaldoFiltro) ([]domaincomissao.SaldoAberto, error) {
	r.m.ultimoFiltroSaldos = filtro

	saldos := make([]domaincomissao.SaldoAberto, 0, len(r.m.saldos))
	for _, saldo := range r.m.saldos {
		if filtro.RepresentanteID == nil || *filtro.RepresentanteID == saldo.RepresentanteID {
			saldos = append(saldos, saldo)
		}
	}

	return saldos, nil
}

func (r repoComissoes) Fechar(_ context.Context, gravacao *portsout.GravacaoFechamento) error {
	if err := r.m.falhaNoFechar[gravacao.Fechamento.RepresentanteID]; err != nil {
		return err
	}
	r.m.fechamentos = append(r.m.fechamentos, gravacao)

	return nil
}

func (r repoComissoes) ObterFechamento(_ context.Context, id uuid.UUID) (*domaincomissao.Fechamento, error) {
	for _, gravacao := range r.m.fechamentos {
		if gravacao.Fechamento.ID == id {
			return gravacao.Fechamento, nil
		}
	}

	return nil, nil
}

func (r repoComissoes) ListarFechamentos(_ context.Context, filtro portsout.FechamentoFiltro) ([]*domaincomissao.Fechamento, error) {
	r.m.ultimoFiltroFechados = filtro

	fechamentos := make([]*domaincomissao.Fechamento, 0, len(r.m.fechamentos))
	for _, gravacao := range r.m.fechamentos {
		fechamentos = append(fechamentos, gravacao.Fechamento)
	}

	return fechamentos, nil
}

func (r repoComissoes) ItensDoFechamento(context.Context, uuid.UUID) ([]*domaincomissao.Comissao, []*domaincomissao.Ajuste, error) {
	return r.m.comissoes, r.m.ajustes, nil
}

func (r repoComissoes) Resumo(_ context.Context, filtro portsout.ResumoFiltro) ([]domaincomissao.LinhaResumo, error) {
	r.m.ultimoFiltroResumo = filtro

	return r.m.resumo, nil
}

type repoCategorias struct{ m *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return r.m.categorias[id], nil
}

type repoPedidos struct{ m *memoria }

func (r repoPedidos) Obter(_ context.Context, id uuid.UUID) (*domainpedido.Pedido, error) {
	return r.m.pedidos[id], nil
}

type repoRepresentantes struct{ m *memoria }

func (r repoRepresentantes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Representante, error) {
	return r.m.representantes[id], nil
}

func (r repoRepresentantes) ObterPorUsuario(_ context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	for _, representante := range r.m.representantes {
		if representante.UsuarioID != nil && *representante.UsuarioID == usuarioID {
			return representante, nil
		}
	}

	return nil, nil
}

type repoEmpresas struct{ m *memoria }

func (r repoEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return r.m.empresas[id], nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.m.acessos[usuarioID], nil
}

type repoUsuarios struct{ m *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return r.m.usuarios[id], nil
}

type repoCargos struct{ m *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return r.m.cargos[id], nil
}
