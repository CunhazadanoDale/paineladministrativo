package tabelapreco_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	tabelaprecousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/tabelapreco"
	"github.com/google/uuid"
)

var (
	_ portsout.TabelaPrecoRepository = (*repoItens)(nil)
	_ portsout.ProdutoRepository     = repoProdutos{}
	_ portsout.UsuarioRepository     = repoUsuarios{}
	_ portsout.CargoRepository       = repoCargos{}
)

type memoria struct {
	usuarios map[uuid.UUID]*domainusuarios.Usuario
	cargos   map[uuid.UUID]*domainusuarios.Cargo
	produtos map[uuid.UUID]*domainestoque.Produto
}

type repoItens struct {
	criados       []*domaintabelapreco.Item
	filtro        portsout.ItemFiltro
	filtroVigente portsout.VigenteFiltro
	removido      uuid.UUID
	hojeRemocao   time.Time
}

func (r *repoItens) Criar(_ context.Context, item *domaintabelapreco.Item) error {
	r.criados = append(r.criados, item)
	return nil
}

func (r *repoItens) Listar(_ context.Context, filtro portsout.ItemFiltro) ([]*domaintabelapreco.Item, error) {
	r.filtro = filtro
	return r.criados, nil
}

func (r *repoItens) Vigentes(_ context.Context, filtro portsout.VigenteFiltro) ([]*domaintabelapreco.PrecoVigente, error) {
	r.filtroVigente = filtro
	return []*domaintabelapreco.PrecoVigente{}, nil
}

func (r *repoItens) RemoverFuturo(_ context.Context, id uuid.UUID, hoje time.Time) error {
	r.removido, r.hojeRemocao = id, hoje
	return nil
}

type repoProdutos struct{ m *memoria }

func (r repoProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	return r.m.produtos[id], nil
}

type repoUsuarios struct{ m *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return r.m.usuarios[id], nil
}

type repoCargos struct{ m *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return r.m.cargos[id], nil
}

type cenario struct {
	ctx           context.Context
	m             *memoria
	itens         *repoItens
	usecase       portsin.TabelaPrecoUseCase
	administrador uuid.UUID
	comercial     uuid.UUID
	financeiro    uuid.UUID
	semPerfil     uuid.UUID
	ativo         uuid.UUID
	inativo       uuid.UUID
}

func novoCenario() *cenario {
	m := &memoria{
		usuarios: make(map[uuid.UUID]*domainusuarios.Usuario),
		cargos:   make(map[uuid.UUID]*domainusuarios.Cargo),
		produtos: make(map[uuid.UUID]*domainestoque.Produto),
	}
	itens := &repoItens{}
	c := &cenario{
		ctx:     context.Background(),
		m:       m,
		itens:   itens,
		usecase: tabelaprecousecases.NewTabelaPrecoUsecase(itens, repoProdutos{m}, repoUsuarios{m}, repoCargos{m}),
	}
	c.administrador = c.novoUsuario(domainusuarios.Cargo{Administrador: true})
	c.comercial = c.novoUsuario(domainusuarios.Cargo{Comercial: true})
	c.financeiro = c.novoUsuario(domainusuarios.Cargo{Financeiro: true})
	c.semPerfil = c.novoUsuario(domainusuarios.Cargo{})
	c.ativo = c.novoProduto(true)
	c.inativo = c.novoProduto(false)

	return c
}

func (c *cenario) novoUsuario(cargo domainusuarios.Cargo) uuid.UUID {
	cargo.ID = uuid.New()
	usuario := &domainusuarios.Usuario{ID: uuid.New(), CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = &cargo
	c.m.usuarios[usuario.ID] = usuario

	return usuario.ID
}

func (c *cenario) novoProduto(ativo bool) uuid.UUID {
	produto := &domainestoque.Produto{ID: uuid.New(), Nome: "Porcelanato", Ativo: ativo}
	c.m.produtos[produto.ID] = produto

	return produto.ID
}

func (c *cenario) dados(produtoID uuid.UUID) domaintabelapreco.DadosItem {
	return domaintabelapreco.DadosItem{
		ProdutoID: produtoID, Modelo: "pronta_entrega", PrecoCentavos: 12990, VigenteDe: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func TestCriarPrecoSoPeloAdministradorEComProdutoAtivo(t *testing.T) {
	c := novoCenario()

	item, err := c.usecase.Criar(c.ctx, portsin.CriarItemInput{UsuarioID: c.administrador, Dados: c.dados(c.ativo)})
	if err != nil || item.UsuarioID != c.administrador || len(c.itens.criados) != 1 {
		t.Fatalf("Criar = %+v, %v", item, err)
	}

	_, err = c.usecase.Criar(c.ctx, portsin.CriarItemInput{UsuarioID: c.comercial, Dados: c.dados(c.ativo)})
	esperarErro(t, err, domain.ErrPermissao, "alterar a tabela de preços")

	_, err = c.usecase.Criar(c.ctx, portsin.CriarItemInput{UsuarioID: c.administrador, Dados: c.dados(c.inativo)})
	esperarErro(t, err, domain.ErrValidacao, "produto inativo")

	_, err = c.usecase.Criar(c.ctx, portsin.CriarItemInput{UsuarioID: c.administrador, Dados: c.dados(uuid.New())})
	esperarErro(t, err, domain.ErrValidacao, "produto do preço não encontrado")
}

func TestLeituraDaTabelaPeloComercialEFinanceiro(t *testing.T) {
	c := novoCenario()

	if _, err := c.usecase.Listar(c.ctx, portsin.ListarItensInput{UsuarioID: c.financeiro, Modelo: "importacao", ProdutoID: &c.ativo}); err != nil {
		t.Fatalf("Listar: %v", err)
	}
	if c.itens.filtro.Modelo != domainpedido.ModeloImportacao || *c.itens.filtro.ProdutoID != c.ativo || c.itens.filtro.Size != domain.TamanhoPaginaPadrao {
		t.Fatalf("filtro = %+v", c.itens.filtro)
	}

	_, err := c.usecase.Listar(c.ctx, portsin.ListarItensInput{UsuarioID: c.semPerfil})
	esperarErro(t, err, domain.ErrPermissao, "ver a tabela de preços")

	_, err = c.usecase.Listar(c.ctx, portsin.ListarItensInput{UsuarioID: c.comercial, Modelo: "x"})
	esperarErro(t, err, domain.ErrValidacao, "modelo de venda")
}

func TestVigentesUsaHojeEmBrasiliaQuandoSemData(t *testing.T) {
	c := novoCenario()

	if _, err := c.usecase.Vigentes(c.ctx, portsin.VigentesInput{UsuarioID: c.comercial, Modelo: "pronta_entrega", Busca: "  porc  "}); err != nil {
		t.Fatalf("Vigentes: %v", err)
	}
	filtro := c.itens.filtroVigente
	if !filtro.Data.Equal(domainpedido.Hoje(time.Now().UTC())) || filtro.Busca != "porc" || filtro.Modelo != domainpedido.ModeloProntaEntrega {
		t.Fatalf("filtro = %+v", filtro)
	}

	data := time.Date(2026, 12, 25, 18, 0, 0, 0, time.UTC)
	if _, err := c.usecase.Vigentes(c.ctx, portsin.VigentesInput{UsuarioID: c.administrador, Modelo: "importacao", Data: &data}); err != nil {
		t.Fatalf("Vigentes com data: %v", err)
	}
	if !c.itens.filtroVigente.Data.Equal(time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("data = %v", c.itens.filtroVigente.Data)
	}

	_, err := c.usecase.Vigentes(c.ctx, portsin.VigentesInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrValidacao, "modelo de venda")
}

func TestRemoverSoPeloAdministrador(t *testing.T) {
	c := novoCenario()
	id := uuid.New()

	esperarErro(t, c.usecase.Remover(c.ctx, id, c.comercial), domain.ErrPermissao, "alterar a tabela de preços")
	if err := c.usecase.Remover(c.ctx, id, c.administrador); err != nil || c.itens.removido != id {
		t.Fatalf("Remover = %v (%v)", err, c.itens.removido)
	}
	if !c.itens.hojeRemocao.Equal(domainpedido.Hoje(time.Now().UTC())) {
		t.Fatalf("hoje = %v", c.itens.hojeRemocao)
	}
}
