package receber_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	receberusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/receber"
	"github.com/google/uuid"
)

type cenario struct {
	ctx      context.Context
	m        *memoria
	titulos  portsin.TituloUseCase
	entradas portsin.EntradaDoPedidoUseCase
	baixas   portsin.RecebimentoUseCase
	borderos portsin.AntecipacaoUseCase

	mia     uuid.UUID
	red     uuid.UUID
	inativa uuid.UUID

	administrador uuid.UUID
	financeiro    uuid.UUID
	financeiroRed uuid.UUID
	comercial     uuid.UUID

	cliente        uuid.UUID
	clienteInativo uuid.UUID
	receita        uuid.UUID
	despesa        uuid.UUID
	raiz           uuid.UUID
	contaMia       uuid.UUID
	contaRed       uuid.UUID
	contaInativa   uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	m := novaMemoria()
	c := &cenario{
		ctx: context.Background(),
		m:   m,
		titulos: receberusecases.NewTituloUsecase(
			repoTitulos{m}, repoClientes{m}, repoContas{m}, repoCategorias{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		entradas: receberusecases.NewEntradaDoPedidoUsecase(
			repoTitulos{m}, repoPedidos{m}, repoContas{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		baixas: receberusecases.NewRecebimentoUsecase(
			repoTitulos{m}, repoAntecipacoes{m}, repoPedidos{m}, repoContas{m}, repoCategorias{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		borderos: receberusecases.NewAntecipacaoUsecase(
			repoAntecipacoes{m}, repoTitulos{m}, repoClientes{m}, repoFornecedores{m}, repoContas{m}, repoCategorias{m},
			repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
	}

	c.mia = c.novaEmpresa(true)
	c.red = c.novaEmpresa(true)
	c.inativa = c.novaEmpresa(false)

	c.administrador = c.novoUsuario(true, false, false, domainempresa.AcessoTotal())
	c.financeiro = c.novoUsuario(false, true, false, domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa}))
	c.financeiroRed = c.novoUsuario(false, true, false, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.comercial = c.novoUsuario(false, false, true, domainempresa.AcessoTotal())

	c.cliente = c.novoCliente(true)
	c.clienteInativo = c.novoCliente(false)

	raiz := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), Natureza: domainfinanceiro.NaturezaReceita, Ativa: true}
	m.categorias[raiz.ID] = raiz
	c.raiz = raiz.ID
	c.receita = c.novaCategoria(raiz.ID, domainfinanceiro.NaturezaReceita)
	c.despesa = c.novaCategoria(raiz.ID, domainfinanceiro.NaturezaDespesa)

	c.contaMia = c.novaConta(c.mia, true)
	c.contaRed = c.novaConta(c.red, true)
	c.contaInativa = c.novaConta(c.mia, false)

	return c
}

func (c *cenario) novaEmpresa(ativa bool) uuid.UUID {
	empresa := &domainempresa.Empresa{ID: uuid.New(), NomeFantasia: "Empresa", Ativa: ativa}
	c.m.empresas[empresa.ID] = empresa

	return empresa.ID
}

func (c *cenario) novoUsuario(administrador, financeiro, comercial bool, acesso domainempresa.Acesso) uuid.UUID {
	cargo := &domainusuarios.Cargo{
		ID: uuid.New(), Nome: "Cargo", Ativo: true,
		Administrador: administrador, Financeiro: financeiro, Comercial: comercial,
	}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = cargo
	c.m.usuarios[usuario.ID] = usuario
	c.m.acessos[usuario.ID] = acesso

	return usuario.ID
}

func (c *cenario) novoCliente(ativo bool) uuid.UUID {
	cliente := &domaincliente.Cliente{ID: uuid.New(), Nome: "Construtora Alfa", Ativo: ativo}
	c.m.clientes[cliente.ID] = cliente

	return cliente.ID
}

func (c *cenario) novaCategoria(paiID uuid.UUID, natureza domainfinanceiro.Natureza) uuid.UUID {
	categoria := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), CategoriaPaiID: &paiID, Natureza: natureza, Ativa: true}
	c.m.categorias[categoria.ID] = categoria

	return categoria.ID
}

func (c *cenario) novaConta(empresaID uuid.UUID, ativa bool) uuid.UUID {
	conta := &domainfinanceiro.ContaBancaria{ID: uuid.New(), EmpresaID: empresaID, Ativa: ativa}
	c.m.contas[conta.ID] = conta

	return conta.ID
}

func (c *cenario) dados() portsin.DadosEdicaoInput {
	return portsin.DadosEdicaoInput{
		Numero:                "ALG-10",
		Parcela:               1,
		Vencimento:            domainreceber.Hoje(time.Now().UTC()).AddDate(0, 1, 0),
		ValorCentavos:         150000,
		CategoriaFinanceiraID: &c.receita,
	}
}

func (c *cenario) criarInput(usuarioID, empresaID uuid.UUID) portsin.CriarTituloInput {
	return portsin.CriarTituloInput{UsuarioID: usuarioID, EmpresaID: empresaID, ClienteID: c.cliente, Dados: c.dados()}
}

func (c *cenario) novoTitulo(t *testing.T) *domainreceber.Titulo {
	t.Helper()

	titulo, err := c.titulos.Criar(c.ctx, c.criarInput(c.financeiro, c.mia))
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	return titulo
}

func (c *cenario) emBanco(t *testing.T) *domainreceber.Titulo {
	t.Helper()

	titulo := c.novoTitulo(t)
	alterado, err := c.titulos.EnviarAoBanco(c.ctx, portsin.EnviarAoBancoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, ContaBancariaID: c.contaMia,
	})
	if err != nil {
		t.Fatalf("EnviarAoBanco falhou: %v", err)
	}

	return alterado
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}
