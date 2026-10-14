package cliente_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type cenarioDeRepresentantes struct {
	*cenario
	usuarioAna uuid.UUID
	ana        *cliente.Representante
	bia        *cliente.Representante
	daAna      *cliente.Cliente
	daBia      *cliente.Cliente
	obraDaBia  *cliente.Obra
}

func novoCenarioDeRepresentantes(t *testing.T) *cenarioDeRepresentantes {
	t.Helper()

	c := novoCenario(t)
	usuarioAna := c.novoUsuario("Ana", false, true)
	usuarioBia := c.novoUsuario("Bia", false, true)
	ana := c.novoRepresentante(t, "Ana Representações", &usuarioAna)
	bia := c.novoRepresentante(t, "Bia Vendas", &usuarioBia)
	daBia := c.novoCliente(t, "33400004000116", "Construtora Ômega", &bia.ID)

	obra, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: c.administrador, ClienteID: daBia.ID, Dados: dadosDeObra("Obra da Bia")})
	if err != nil {
		t.Fatalf("criar obra falhou: %v", err)
	}

	return &cenarioDeRepresentantes{
		cenario:    c,
		usuarioAna: usuarioAna,
		ana:        ana,
		bia:        bia,
		daAna:      c.novoCliente(t, "11222333000181", "Construtora Águia", &ana.ID),
		daBia:      daBia,
		obraDaBia:  obra,
	}
}

func TestRepresentanteNaoMexeNoClienteNemNaObraDeOutroPorNenhumaRota(t *testing.T) {
	c := novoCenarioDeRepresentantes(t)

	tentativas := map[string]func() error{
		"ativar cliente": func() error {
			_, err := c.clientes.Ativar(c.ctx, c.daBia.ID, c.usuarioAna)
			return err
		},
		"desativar cliente": func() error {
			_, err := c.clientes.Desativar(c.ctx, c.daBia.ID, c.usuarioAna)
			return err
		},
		"obter obra": func() error {
			_, err := c.obras.Obter(c.ctx, c.obraDaBia.ID, c.usuarioAna)
			return err
		},
		"alterar obra": func() error {
			_, err := c.obras.Alterar(c.ctx, portsin.AlterarObraInput{UsuarioID: c.usuarioAna, ObraID: c.obraDaBia.ID, Dados: dadosDeObra("Tomada")})
			return err
		},
		"ativar obra": func() error {
			_, err := c.obras.Ativar(c.ctx, c.obraDaBia.ID, c.usuarioAna)
			return err
		},
		"desativar obra": func() error {
			_, err := c.obras.Desativar(c.ctx, c.obraDaBia.ID, c.usuarioAna)
			return err
		},
	}

	for nome, tentar := range tentativas {
		t.Run(nome, func(t *testing.T) {
			if err := tentar(); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("erro %v, esperado erro de permissão", err)
			}
		})
	}

	if obra, _ := c.obras.Obter(c.ctx, c.obraDaBia.ID, c.administrador); obra.Nome != "Obra da Bia" || !obra.Ativo {
		t.Errorf("obra da Bia foi alterada: %+v", obra)
	}
}

func TestConversaoDeLeadPorRepresentanteFicaComEle(t *testing.T) {
	c := novoCenarioDeRepresentantes(t)
	leadID := c.novoLead("Loja Nova", "", "")
	dados := portsin.DadosClienteInput{Tipo: "loja", UF: "PB", Cidade: "JP"}

	comOutro := dados
	comOutro.RepresentanteID = &c.bia.ID
	if _, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.usuarioAna, LeadID: leadID, Documento: "55666777000181", Dados: comOutro}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("converter para outro representante = %v, esperado erro de permissão", err)
	}

	convertido, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.usuarioAna, LeadID: leadID, Documento: "55666777000181", Dados: dados})
	if err != nil || convertido.RepresentanteID == nil || *convertido.RepresentanteID != c.ana.ID {
		t.Errorf("lead convertido pela Ana = %+v (%v), esperado com ela como representante", convertido, err)
	}
}

func TestRepresentanteSemPerfilComercialSoLeOsProprios(t *testing.T) {
	c := novoCenarioDeRepresentantes(t)
	usuarioLeitor := c.novoUsuario("Leitor", false, false)
	leitor := c.novoRepresentante(t, "Leitor Representações", &usuarioLeitor)
	proprio := c.novoCliente(t, "55666777000181", "Cliente do Leitor", &leitor.ID)

	lista, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: usuarioLeitor})
	if err != nil || len(lista) != 1 || lista[0].ID != proprio.ID {
		t.Errorf("clientes do representante sem perfil comercial = %+v (%v)", lista, err)
	}
	if _, err := c.clientes.Obter(c.ctx, c.daAna.ID, usuarioLeitor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("obter cliente alheio = %v, esperado erro de permissão", err)
	}
	if _, err := c.clientes.Desativar(c.ctx, proprio.ID, usuarioLeitor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("desativar sem perfil comercial = %v, esperado erro de permissão", err)
	}
}

func TestListagemDoRepresentanteRespeitaAPaginacao(t *testing.T) {
	c := novoCenarioDeRepresentantes(t)

	primeira, err := c.representantes.Listar(c.ctx, portsin.ListarRepresentantesInput{UsuarioID: c.usuarioAna, Filtro: domain.PaginacaoFiltro{Page: 1, Size: 10}})
	if err != nil || len(primeira) != 1 {
		t.Errorf("primeira página = %+v (%v), esperado o próprio cadastro", primeira, err)
	}
	segunda, err := c.representantes.Listar(c.ctx, portsin.ListarRepresentantesInput{UsuarioID: c.usuarioAna, Filtro: domain.PaginacaoFiltro{Page: 2, Size: 10}})
	if err != nil || len(segunda) != 0 {
		t.Errorf("segunda página = %+v (%v), esperado vazia", segunda, err)
	}
}

func TestBuscaSoComPontuacaoNaoDevolveTodos(t *testing.T) {
	c := novoCenario(t)
	c.novoCliente(t, "11222333000181", "Construtora Águia", nil)
	c.novoCliente(t, "33400004000116", "Construtora Ômega", nil)

	for _, busca := range []string{".", "/", "-"} {
		encontrados, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.administrador, Busca: busca})
		if err != nil || len(encontrados) != 0 {
			t.Errorf("busca %q = %d clientes (%v), esperado nenhum", busca, len(encontrados), err)
		}
	}
}
