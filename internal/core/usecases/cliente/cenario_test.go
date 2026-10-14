package cliente_test

import (
	"context"
	"testing"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	clienteusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/cliente"
	"github.com/google/uuid"
)

type cenario struct {
	ctx            context.Context
	clientes       portsin.ClienteUseCase
	obras          portsin.ObraUseCase
	grupos         portsin.GrupoUseCase
	representantes portsin.RepresentanteUseCase
	importacao     portsin.ImportacaoUseCase
	leadsRepo      *repositorioLeads
	gruposRepo     *repositorioGrupos
	usuariosRepo   *repositorioUsuarios
	cargosRepo     *repositorioCargos
	empresasRepo   *repositorioEmpresas
	clientesRepo   *repositorioClientes
	administrador  uuid.UUID
	comercial      uuid.UUID
	operador       uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	usuarios := &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
	cargos := &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
	empresas := &repositorioEmpresas{itens: map[uuid.UUID]*domainempresa.Empresa{}}
	grupos := &repositorioGrupos{itens: map[uuid.UUID]*domaincliente.Grupo{}}
	representantes := &repositorioRepresentantes{itens: map[uuid.UUID]*domaincliente.Representante{}}
	obras := &repositorioObras{itens: map[uuid.UUID]*domaincliente.Obra{}}
	clientes := &repositorioClientes{itens: map[uuid.UUID]*domaincliente.Cliente{}, obras: obras}
	leads := &repositorioLeads{itens: map[uuid.UUID]*domainlead.Lead{}}
	usecaseDeClientes := clienteusecases.NewClienteUsecase(clientes, grupos, representantes, empresas, leads, usuarios, cargos)

	c := &cenario{
		ctx:            context.Background(),
		clientes:       usecaseDeClientes,
		importacao:     usecaseDeClientes,
		leadsRepo:      leads,
		gruposRepo:     grupos,
		obras:          clienteusecases.NewObraUsecase(obras, clientes, representantes, usuarios, cargos),
		grupos:         clienteusecases.NewGrupoUsecase(grupos, representantes, usuarios, cargos),
		representantes: clienteusecases.NewRepresentanteUsecase(representantes, usuarios, cargos),
		usuariosRepo:   usuarios,
		cargosRepo:     cargos,
		empresasRepo:   empresas,
		clientesRepo:   clientes,
	}
	c.administrador = c.novoUsuario("Diretoria", true, false)
	c.comercial = c.novoUsuario("Comercial", false, true)
	c.operador = c.novoUsuario("Operador", false, false)

	return c
}

func (c *cenario) novoUsuario(nome string, administrador, comercial bool) uuid.UUID {
	cargoID := uuid.New()
	c.cargosRepo.itens[cargoID] = &domainusuarios.Cargo{
		ID: cargoID, Nome: "Cargo " + nome, Ativo: true, Administrador: administrador, Comercial: comercial,
	}

	usuarioID := uuid.New()
	c.usuariosRepo.itens[usuarioID] = &domainusuarios.Usuario{ID: usuarioID, Nome: nome, CargoID: cargoID, Ativo: true}

	return usuarioID
}

func (c *cenario) novaEmpresa(ativa bool) uuid.UUID {
	id := uuid.New()
	c.empresasRepo.itens[id] = &domainempresa.Empresa{ID: id, NomeFantasia: "Empresa", Ativa: ativa}

	return id
}

func (c *cenario) novoRepresentante(t *testing.T, nome string, usuarioID *uuid.UUID) *domaincliente.Representante {
	t.Helper()

	representante, err := c.representantes.Criar(c.ctx, portsin.CriarRepresentanteInput{
		UsuarioID: c.administrador,
		Dados: portsin.DadosRepresentanteInput{
			Nome:                    nome,
			UsuarioVinculadoID:      usuarioID,
			ComissaoProntaEntregaBP: 300,
			ComissaoImportacaoBP:    150,
		},
	})
	if err != nil {
		t.Fatalf("criação do representante falhou: %v", err)
	}

	return representante
}

func dadosDeCliente(nome, tipo string, representanteID *uuid.UUID) portsin.DadosClienteInput {
	return portsin.DadosClienteInput{
		Nome:            nome,
		Tipo:            tipo,
		UF:              "pb",
		Cidade:          "João Pessoa",
		RepresentanteID: representanteID,
	}
}

func (c *cenario) novoCliente(t *testing.T, documento, nome string, representanteID *uuid.UUID) *domaincliente.Cliente {
	t.Helper()

	cliente, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{
		UsuarioID: c.administrador,
		Documento: documento,
		Dados:     dadosDeCliente(nome, "construtora", representanteID),
	})
	if err != nil {
		t.Fatalf("criação do cliente falhou: %v", err)
	}

	return cliente
}

func dadosDeObra(nome string) portsin.DadosObraInput {
	return portsin.DadosObraInput{Nome: nome, Logradouro: "Av. Beira Mar", Cidade: "Cabedelo", UF: "PB"}
}
