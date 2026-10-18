package notafiscal_test

import (
	"context"
	"testing"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
	notafiscalusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
	"github.com/google/uuid"
)

const (
	cnpjMia          = "11222333000181"
	cnpjRed          = "33400004000116"
	cnpjInativa      = "99888777000100"
	cnpjCliente      = "11444777000161"
	cpfClienteNovo   = "52998224725"
	cnpjDesconhecido = "11111111000191"
)

type cenario struct {
	ctx     context.Context
	m       *memoria
	notas   portsin.NotaFiscalUseCase
	eventos portsin.EventoUseCase

	mia     uuid.UUID
	red     uuid.UUID
	inativa uuid.UUID

	administrador uuid.UUID
	financeiro    uuid.UUID
	financeiroRed uuid.UUID
	comercial     uuid.UUID
	vendedor      uuid.UUID
	representante *domaincliente.Representante

	cliente *domaincliente.Cliente
	piso    *domainestoque.Produto
	rejunte *domainestoque.Produto
	pedido  *domainpedido.Pedido
	entrega *domainpedido.Entrega
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	m := novaMemoria()
	c := &cenario{
		ctx: context.Background(),
		m:   m,
		notas: notafiscalusecases.NewNotaFiscalUsecase(
			repoNotas{m}, repoArquivos{m}, storageFalso{m}, repoEmpresas{m}, repoClientes{m}, repoProdutos{m},
			repoPedidos{m}, repoEntregas{m}, repoTitulos{m}, repoRepresentantes{m}, repoUsuarios{m}, repoCargos{m},
		),
		eventos: notafiscalusecases.NewEventoUsecase(
			repoNotas{m}, repoArquivos{m}, storageFalso{m}, repoEmpresas{m}, repoClientes{m},
			repoTitulos{m}, repoRepresentantes{m}, repoUsuarios{m}, repoCargos{m},
		),
	}

	c.mia = c.novaEmpresa(cnpjMia, true)
	c.red = c.novaEmpresa(cnpjRed, true)
	c.inativa = c.novaEmpresa(cnpjInativa, false)

	acessoMia := domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa})
	c.administrador = c.novoUsuario(true, false, false, domainempresa.AcessoTotal())
	c.financeiro = c.novoUsuario(false, false, true, acessoMia)
	c.financeiroRed = c.novoUsuario(false, false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.comercial = c.novoUsuario(false, true, false, acessoMia)
	c.vendedor = c.novoUsuario(false, true, false, acessoMia)
	c.representante = &domaincliente.Representante{ID: uuid.New(), Nome: "Vendedor", UsuarioID: &c.vendedor, Ativo: true}
	m.representantes[c.representante.ID] = c.representante

	c.cliente = &domaincliente.Cliente{
		ID: uuid.New(), Documento: domaincliente.DocumentoDe(cnpjCliente), Nome: "Construtora Alfa",
		RepresentanteID: &c.representante.ID, Ativo: true,
	}
	m.clientes[c.cliente.ID] = c.cliente

	c.piso = c.novoProduto("PISO-01")
	c.rejunte = c.novoProduto("")

	c.pedido = &domainpedido.Pedido{
		ID: uuid.New(), EmpresaID: c.mia, Numero: 7, ClienteID: c.cliente.ID,
		RepresentanteID: &c.representante.ID, Status: domainpedido.StatusConfirmado, Versao: 2,
	}
	m.pedidos[c.pedido.ID] = c.pedido
	c.entrega = &domainpedido.Entrega{ID: uuid.New(), PedidoID: c.pedido.ID, Status: domainpedido.StatusEntregaSaiu, Versao: 2}
	m.entregas[c.entrega.ID] = c.entrega

	return c
}

func (c *cenario) novaEmpresa(cnpj string, ativa bool) uuid.UUID {
	empresa := &domainempresa.Empresa{ID: uuid.New(), NomeFantasia: "Empresa", CNPJ: domainempresa.CNPJDe(cnpj), Ativa: ativa}
	c.m.empresas[empresa.ID] = empresa

	return empresa.ID
}

func (c *cenario) novoUsuario(administrador, comercial, financeiro bool, acesso domainempresa.Acesso) uuid.UUID {
	cargo := &domainusuarios.Cargo{
		ID: uuid.New(), Nome: "Cargo", Ativo: true,
		Administrador: administrador, Comercial: comercial, Financeiro: financeiro,
	}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = cargo
	c.m.usuarios[usuario.ID] = usuario
	c.m.acessos[usuario.ID] = acesso

	return usuario.ID
}

func (c *cenario) novoProduto(codigo string) *domainestoque.Produto {
	produto := &domainestoque.Produto{ID: uuid.New(), Nome: "Produto " + codigo, Ativo: true}
	if codigo != "" {
		produto.Codigo = &codigo
	}
	c.m.produtos[produto.ID] = produto

	return produto
}

func (c *cenario) enviar(dono uuid.UUID, conteudo []byte) uuid.UUID {
	return c.enviarComTipo(dono, conteudo, "application/xml")
}

func (c *cenario) enviarComTipo(dono uuid.UUID, conteudo []byte, tipo string) uuid.UUID {
	id := uuid.New()
	chave := dono.String() + "/" + id.String() + "/nfe.xml"
	c.m.arquivos[id] = &domainsolicitacao.Arquivo{
		ID: id, ProprietarioID: dono, Nome: "nfe.xml", Chave: chave, ContentType: tipo,
		Tamanho: int64(len(conteudo)), CriadoEm: time.Now().UTC(),
	}
	c.m.conteudos[chave] = conteudo

	return id
}

func (c *cenario) importar(t *testing.T, usuarioID uuid.UUID, documento nfe.NFe) *portsin.Importacao {
	t.Helper()

	importacao, err := c.notas.Importar(c.ctx, portsin.ImportarInput{
		UsuarioID: usuarioID,
		ArquivoID: c.enviar(usuarioID, documento.XML()),
	})
	if err != nil {
		t.Fatalf("Importar = %v", err)
	}

	return importacao
}

func (c *cenario) notaDaMia(numero int) nfe.NFe {
	return nfe.Nova(cnpjMia, cnpjCliente, numero)
}
