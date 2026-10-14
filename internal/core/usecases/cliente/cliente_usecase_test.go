package cliente_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

func TestCriarClienteValidaPerfilDocumentoEVinculos(t *testing.T) {
	c := novoCenario(t)
	grupo, err := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Grupo Inativo"})
	if err != nil {
		t.Fatalf("criação do grupo falhou: %v", err)
	}
	if _, err := c.grupos.Desativar(c.ctx, grupo.ID, c.comercial); err != nil {
		t.Fatalf("desativar grupo falhou: %v", err)
	}
	representante := c.novoRepresentante(t, "Inativo", nil)
	if _, err := c.representantes.Desativar(c.ctx, representante.ID, c.administrador); err != nil {
		t.Fatalf("desativar representante falhou: %v", err)
	}
	empresaInativa := c.novaEmpresa(false)
	c.novoCliente(t, "11.222.333/0001-81", "Construtora Águia", nil)
	inexistente := uuid.New()

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		documento string
		alterar   func(*portsin.DadosClienteInput)
		sentinela error
		mensagem  string
	}{
		{"sem perfil comercial", c.operador, "33.400.004/0001-16", func(*portsin.DadosClienteInput) {}, domain.ErrPermissao, ""},
		{"documento inválido", c.comercial, "33.400.004/0001-17", func(*portsin.DadosClienteInput) {}, domain.ErrValidacao, ""},
		{"documento repetido", c.comercial, "11222333000181", func(*portsin.DadosClienteInput) {}, domain.ErrConflito, "já existe um cliente com este documento"},
		{"tipo inválido", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.Tipo = "revenda" }, domain.ErrValidacao, ""},
		{"uf inválida", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.UF = "XX" }, domain.ErrValidacao, ""},
		{"grupo inexistente", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.GrupoID = &inexistente }, domain.ErrValidacao, "grupo de clientes não encontrado"},
		{"grupo inativo", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.GrupoID = &grupo.ID }, domain.ErrValidacao, "grupo de clientes inativo não recebe cliente"},
		{"representante inativo", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.RepresentanteID = &representante.ID }, domain.ErrValidacao, ""},
		{"empresa inexistente", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.EmpresaRelacionamentoID = &inexistente }, domain.ErrValidacao, ""},
		{"empresa inativa", c.comercial, "33.400.004/0001-16", func(d *portsin.DadosClienteInput) { d.EmpresaRelacionamentoID = &empresaInativa }, domain.ErrValidacao, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeCliente("Loja Concreção", "loja", nil)
			caso.alterar(&dados)
			_, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{UsuarioID: caso.usuario, Documento: caso.documento, Dados: dados})
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.mensagem != "" && !mesmaMensagem(err, caso.mensagem) {
				t.Errorf("mensagem %q, esperado %q", err.Error(), caso.mensagem)
			}
		})
	}

	empresa := c.novaEmpresa(true)
	dados := dadosDeCliente("Loja Concreção", "loja", nil)
	dados.EmpresaRelacionamentoID = &empresa
	cliente, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{UsuarioID: c.comercial, Documento: "33.400.004/0001-16", Dados: dados})
	if err != nil {
		t.Fatalf("criar pelo comercial falhou: %v", err)
	}
	if cliente.UF != "PB" || cliente.Documento.Valor() != "33400004000116" || *cliente.EmpresaRelacionamentoID != empresa {
		t.Errorf("cliente criado = %+v", cliente)
	}
}

func mesmaMensagem(err error, mensagem string) bool {
	for _, sentinela := range []func(string) error{domain.ErroValidacao, domain.ErroConflito, domain.ErroPermissao, domain.ErroNaoEncontrado} {
		if err.Error() == sentinela(mensagem).Error() {
			return true
		}
	}

	return false
}

func TestAlterarClienteMantemVinculoInativoJaAtribuido(t *testing.T) {
	c := novoCenario(t)
	grupo, _ := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Grupo Ômega"})
	outro, _ := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Grupo Alfa"})
	dados := dadosDeCliente("Construtora Ômega", "construtora", nil)
	dados.GrupoID = &grupo.ID
	cliente, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{UsuarioID: c.comercial, Documento: "11222333000181", Dados: dados})
	if err != nil {
		t.Fatalf("criar falhou: %v", err)
	}
	for _, id := range []uuid.UUID{grupo.ID, outro.ID} {
		if _, err := c.grupos.Desativar(c.ctx, id, c.comercial); err != nil {
			t.Fatalf("desativar grupo falhou: %v", err)
		}
	}

	dados.Nome = "Ômega Engenharia"
	alterado, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{UsuarioID: c.comercial, ClienteID: cliente.ID, Dados: dados})
	if err != nil || alterado.Nome != "Ômega Engenharia" || *alterado.GrupoID != grupo.ID {
		t.Fatalf("alterar mantendo grupo inativo = %+v (%v)", alterado, err)
	}

	dados.GrupoID = &outro.ID
	if _, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{UsuarioID: c.comercial, ClienteID: cliente.ID, Dados: dados}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("trocar para grupo inativo = %v, esperado erro de validação", err)
	}
	if _, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{UsuarioID: c.comercial, ClienteID: uuid.New(), Dados: dados}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("alterar cliente inexistente = %v, esperado não encontrado", err)
	}
	if _, err := c.clientes.Desativar(c.ctx, cliente.ID, c.operador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("desativar sem perfil = %v, esperado erro de permissão", err)
	}
	if desativado, err := c.clientes.Desativar(c.ctx, cliente.ID, c.comercial); err != nil || desativado.Ativo {
		t.Errorf("desativar = %+v (%v)", desativado, err)
	}
}

func TestRemoverClienteExigeAdministradorESemVinculos(t *testing.T) {
	c := novoCenario(t)
	comObra := c.novoCliente(t, "11222333000181", "Construtora Águia", nil)
	semObra := c.novoCliente(t, "33400004000116", "Loja Avulsa", nil)
	if _, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: c.comercial, ClienteID: comObra.ID, Dados: dadosDeObra("Mar Azul")}); err != nil {
		t.Fatalf("criar obra falhou: %v", err)
	}

	if err := c.clientes.Remover(c.ctx, semObra.ID, c.comercial); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("remover pelo comercial = %v, esperado erro de permissão", err)
	}
	if err := c.clientes.Remover(c.ctx, comObra.ID, c.administrador); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remover com obra = %v, esperado erro de validação", err)
	}
	if err := c.clientes.Remover(c.ctx, semObra.ID, c.administrador); err != nil {
		t.Errorf("remover sem vínculo = %v", err)
	}
	if err := c.clientes.Remover(c.ctx, semObra.ID, c.administrador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("remover de novo = %v, esperado não encontrado", err)
	}
}

func TestListarClientesValidaFiltros(t *testing.T) {
	c := novoCenario(t)
	c.novoCliente(t, "11222333000181", "Construtora Águia", nil)
	pessoa, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{
		UsuarioID: c.comercial, Documento: "529.982.247-25", Dados: dadosDeCliente("José da Silva", "pf", nil),
	})
	if err != nil {
		t.Fatalf("criar pessoa física falhou: %v", err)
	}

	pf, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.operador, Tipo: "PF", UF: "pb"})
	if err != nil || len(pf) != 1 || pf[0].ID != pessoa.ID {
		t.Errorf("filtro pf/pb = %+v (%v)", pf, err)
	}
	busca, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.operador, Busca: "AGUIA"})
	if err != nil || len(busca) != 1 {
		t.Errorf("busca sem acento = %d (%v), esperado 1", len(busca), err)
	}
	if _, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.operador, Tipo: "revenda"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tipo inválido = %v, esperado erro de validação", err)
	}
	if _, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.operador, UF: "XX"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("uf inválida = %v, esperado erro de validação", err)
	}
}

func TestRepresentanteComLoginSoEnxergaOsProprios(t *testing.T) {
	c := novoCenario(t)
	usuarioAna := c.novoUsuario("Ana", false, true)
	usuarioBia := c.novoUsuario("Bia", false, true)
	ana := c.novoRepresentante(t, "Ana Representações", &usuarioAna)
	bia := c.novoRepresentante(t, "Bia Vendas", &usuarioBia)
	daAna := c.novoCliente(t, "11222333000181", "Construtora Águia", &ana.ID)
	daBia := c.novoCliente(t, "33400004000116", "Construtora Ômega", &bia.ID)
	c.novoCliente(t, "55666777000181", "Construtora Sem Representante", nil)
	if _, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: c.administrador, ClienteID: daBia.ID, Dados: dadosDeObra("Obra da Bia")}); err != nil {
		t.Fatalf("criar obra falhou: %v", err)
	}

	casos := []struct {
		nome    string
		usuario uuid.UUID
		proprio uuid.UUID
		cliente uuid.UUID
		alheio  uuid.UUID
		outroRe uuid.UUID
	}{
		{"Ana", usuarioAna, ana.ID, daAna.ID, daBia.ID, bia.ID},
		{"Bia", usuarioBia, bia.ID, daBia.ID, daAna.ID, ana.ID},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			lista, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: caso.usuario})
			if err != nil || len(lista) != 1 || lista[0].ID != caso.cliente {
				t.Errorf("clientes listados = %+v (%v), esperado só o próprio", lista, err)
			}
			if _, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: caso.usuario, RepresentanteID: &caso.outroRe}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("filtrar pelo outro representante = %v, esperado erro de permissão", err)
			}
			if _, err := c.clientes.Obter(c.ctx, caso.alheio, caso.usuario); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("obter cliente alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{
				UsuarioID: caso.usuario, ClienteID: caso.alheio, Dados: dadosDeCliente("Tomado", "construtora", nil),
			}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("alterar cliente alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{
				UsuarioID: caso.usuario, ClienteID: caso.cliente, Dados: dadosDeCliente("Repassado", "construtora", &caso.outroRe),
			}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("passar o próprio cliente para outro representante = %v, esperado erro de permissão", err)
			}
			if _, err := c.obras.Listar(c.ctx, portsin.ListarObrasInput{UsuarioID: caso.usuario, ClienteID: caso.alheio}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("obras do cliente alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: caso.usuario, ClienteID: caso.alheio, Dados: dadosDeObra("Invasão")}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("obra no cliente alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.representantes.Obter(c.ctx, caso.outroRe, caso.usuario); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("obter outro representante = %v, esperado erro de permissão", err)
			}
			representantes, err := c.representantes.Listar(c.ctx, portsin.ListarRepresentantesInput{UsuarioID: caso.usuario})
			if err != nil || len(representantes) != 1 || representantes[0].ID != caso.proprio {
				t.Errorf("representantes listados = %+v (%v), esperado só o próprio", representantes, err)
			}

			alterado, err := c.clientes.Alterar(c.ctx, portsin.AlterarClienteInput{
				UsuarioID: caso.usuario, ClienteID: caso.cliente, Dados: dadosDeCliente("Nome novo "+caso.nome, "construtora", nil),
			})
			if err != nil || alterado.RepresentanteID == nil || *alterado.RepresentanteID != caso.proprio {
				t.Errorf("alterar o próprio sem informar representante = %+v (%v), esperado manter o representante", alterado, err)
			}
		})
	}

	novo, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{
		UsuarioID: usuarioAna, Documento: "111.444.777-35", Dados: dadosDeCliente("Maria", "pf", nil),
	})
	if err != nil || novo.RepresentanteID == nil || *novo.RepresentanteID != ana.ID {
		t.Errorf("cliente criado pela representante = %+v (%v), esperado com ela como representante", novo, err)
	}
	if _, err := c.clientes.Criar(c.ctx, portsin.CriarClienteInput{
		UsuarioID: usuarioAna, Documento: "123.456.789-09", Dados: dadosDeCliente("Pedro", "pf", &bia.ID),
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("criar cliente para outro representante = %v, esperado erro de permissão", err)
	}

	administradorRepresentante := c.novoUsuario("Diretor que vende", true, false)
	c.novoRepresentante(t, "Diretor", &administradorRepresentante)
	if todos, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: administradorRepresentante}); err != nil || len(todos) != 4 {
		t.Errorf("administrador que é representante listou %d (%v), esperado todos os 4", len(todos), err)
	}
	if todos, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.comercial}); err != nil || len(todos) != 4 {
		t.Errorf("comercial sem vínculo listou %d (%v), esperado todos os 4", len(todos), err)
	}
}
