package cliente_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

func texto(valor string) *string {
	return &valor
}

func TestRepresentanteSoAdministradorEscreveEValidaVinculos(t *testing.T) {
	c := novoCenario(t)
	usuarioJose := c.novoUsuario("José", false, true)
	jose := c.novoRepresentante(t, "José Vendas", &usuarioJose)
	inexistente := uuid.New()

	dados := func(nome string, documento *string, usuario *uuid.UUID, pontos int) portsin.DadosRepresentanteInput {
		return portsin.DadosRepresentanteInput{Nome: nome, Documento: documento, UsuarioVinculadoID: usuario, ComissaoProntaEntregaBP: pontos}
	}
	if _, err := c.representantes.Criar(c.ctx, portsin.CriarRepresentanteInput{
		UsuarioID: c.administrador, Dados: dados("Maria", texto("529.982.247-25"), nil, 200),
	}); err != nil {
		t.Fatalf("criar com documento falhou: %v", err)
	}

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		dados     portsin.DadosRepresentanteInput
		sentinela error
		mensagem  string
	}{
		{"comercial não cria", c.comercial, dados("Outro", nil, nil, 0), domain.ErrPermissao, ""},
		{"documento repetido", c.administrador, dados("Outro", texto("52998224725"), nil, 0), domain.ErrConflito, "já existe um representante com este documento"},
		{"documento inválido", c.administrador, dados("Outro", texto("123"), nil, 0), domain.ErrValidacao, ""},
		{"usuário inexistente", c.administrador, dados("Outro", nil, &inexistente, 0), domain.ErrValidacao, "usuário vinculado não encontrado"},
		{"usuário já vinculado", c.administrador, dados("Outro", nil, &usuarioJose, 0), domain.ErrConflito, "usuário já vinculado a outro representante"},
		{"percentual acima de 100%", c.administrador, dados("Outro", nil, nil, 10001), domain.ErrValidacao, ""},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.representantes.Criar(c.ctx, portsin.CriarRepresentanteInput{UsuarioID: caso.usuario, Dados: caso.dados})
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.mensagem != "" && !mesmaMensagem(err, caso.mensagem) {
				t.Errorf("mensagem %q, esperado %q", err.Error(), caso.mensagem)
			}
		})
	}

	alterado, err := c.representantes.Alterar(c.ctx, portsin.AlterarRepresentanteInput{
		UsuarioID: c.administrador, RepresentanteID: jose.ID, Dados: dados("José Vendas", nil, &usuarioJose, 450),
	})
	if err != nil || alterado.ComissaoProntaEntrega.PontosBase() != 450 || alterado.UsuarioID == nil {
		t.Errorf("alterar mantendo o próprio usuário = %+v (%v)", alterado, err)
	}
	if _, err := c.representantes.Alterar(c.ctx, portsin.AlterarRepresentanteInput{
		UsuarioID: c.comercial, RepresentanteID: jose.ID, Dados: dados("José", nil, nil, 0),
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("alterar pelo comercial = %v, esperado erro de permissão", err)
	}
	if _, err := c.representantes.Obter(c.ctx, uuid.New(), c.comercial); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("obter inexistente = %v, esperado não encontrado", err)
	}
	todos, err := c.representantes.Listar(c.ctx, portsin.ListarRepresentantesInput{UsuarioID: c.comercial, Busca: "VENDAS"})
	if err != nil || len(todos) != 1 || todos[0].ID != jose.ID {
		t.Errorf("busca de representantes pelo comercial = %+v (%v)", todos, err)
	}
}

func TestGruposDeClientes(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.operador, Nome: "Grupo"}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("criar sem perfil comercial = %v, esperado erro de permissão", err)
	}
	omega, err := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Grupo Ômega"})
	if err != nil {
		t.Fatalf("criar grupo falhou: %v", err)
	}
	_, err = c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "grupo omega"})
	if !errors.Is(err, domain.ErrConflito) || !mesmaMensagem(err, "já existe um grupo de clientes com este nome") {
		t.Errorf("grupo repetido = %v, esperado conflito com mensagem do grupo", err)
	}

	alfa, _ := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Alfa"})
	if _, err := c.grupos.Renomear(c.ctx, portsin.RenomearGrupoInput{UsuarioID: c.comercial, GrupoID: alfa.ID, Nome: "GRUPO ÔMEGA"}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("renomear para nome existente = %v, esperado conflito", err)
	}
	renomeado, err := c.grupos.Renomear(c.ctx, portsin.RenomearGrupoInput{UsuarioID: c.comercial, GrupoID: omega.ID, Nome: "Ômega Construções"})
	if err != nil || renomeado.NomeBusca != "omega construcoes" {
		t.Errorf("renomear = %+v (%v)", renomeado, err)
	}
	if _, err := c.grupos.Obter(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("obter inexistente = %v, esperado não encontrado", err)
	}

	if _, err := c.grupos.Desativar(c.ctx, alfa.ID, c.comercial); err != nil {
		t.Fatalf("desativar falhou: %v", err)
	}
	ativo := true
	ativos, err := c.grupos.Listar(c.ctx, portsin.ListarGruposInput{Ativo: &ativo})
	if err != nil || len(ativos) != 1 || ativos[0].ID != omega.ID {
		t.Errorf("grupos ativos = %+v (%v)", ativos, err)
	}
}

func TestObrasDoCliente(t *testing.T) {
	c := novoCenario(t)
	cliente := c.novoCliente(t, "11222333000181", "Construtora Águia", nil)
	inativo := c.novoCliente(t, "33400004000116", "Construtora Parada", nil)
	if _, err := c.clientes.Desativar(c.ctx, inativo.ID, c.comercial); err != nil {
		t.Fatalf("desativar cliente falhou: %v", err)
	}

	dados := dadosDeObra("Residencial Mar Azul")
	dados.CNPJ = texto("11.222.333/0001-81")
	dados.CEP = texto("58310-000")
	obra, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: c.comercial, ClienteID: cliente.ID, Dados: dados})
	if err != nil || obra.CNPJ == nil || *obra.Endereco.CEP != "58310000" || obra.Endereco.UF != "PB" {
		t.Fatalf("criar obra = %+v (%v)", obra, err)
	}

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		clienteID uuid.UUID
		alterar   func(*portsin.DadosObraInput)
		sentinela error
	}{
		{"sem perfil comercial", c.operador, cliente.ID, func(*portsin.DadosObraInput) {}, domain.ErrPermissao},
		{"cliente inexistente", c.comercial, uuid.New(), func(*portsin.DadosObraInput) {}, domain.ErrNotFound},
		{"cliente inativo", c.comercial, inativo.ID, func(*portsin.DadosObraInput) {}, domain.ErrValidacao},
		{"nome repetido", c.comercial, cliente.ID, func(d *portsin.DadosObraInput) { d.Nome = "RESIDENCIAL MAR AZUL" }, domain.ErrConflito},
		{"cnpj inválido", c.comercial, cliente.ID, func(d *portsin.DadosObraInput) { d.Nome = "Outra"; d.CNPJ = texto("123") }, domain.ErrValidacao},
		{"uf inválida", c.comercial, cliente.ID, func(d *portsin.DadosObraInput) { d.Nome = "Outra"; d.UF = "" }, domain.ErrValidacao},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			entrada := dadosDeObra("Residencial Novo")
			caso.alterar(&entrada)
			if _, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: caso.usuario, ClienteID: caso.clienteID, Dados: entrada}); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	alterada, err := c.obras.Alterar(c.ctx, portsin.AlterarObraInput{UsuarioID: c.comercial, ObraID: obra.ID, Dados: dadosDeObra("Mar Azul Fase 2")})
	if err != nil || alterada.Nome != "Mar Azul Fase 2" || alterada.CNPJ != nil {
		t.Errorf("alterar obra = %+v (%v)", alterada, err)
	}
	if _, err := c.obras.Desativar(c.ctx, obra.ID, c.comercial); err != nil {
		t.Fatalf("desativar obra falhou: %v", err)
	}
	if _, err := c.obras.Criar(c.ctx, portsin.CriarObraInput{UsuarioID: c.comercial, ClienteID: cliente.ID, Dados: dadosDeObra("Edifício Sol")}); err != nil {
		t.Fatalf("segunda obra falhou: %v", err)
	}

	ativa := true
	ativas, err := c.obras.Listar(c.ctx, portsin.ListarObrasInput{UsuarioID: c.operador, ClienteID: cliente.ID, Ativo: &ativa})
	if err != nil || len(ativas) != 1 || ativas[0].Nome != "Edifício Sol" {
		t.Errorf("obras ativas = %+v (%v)", ativas, err)
	}
	if _, err := c.obras.Obter(c.ctx, uuid.New(), c.operador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("obter obra inexistente = %v, esperado não encontrado", err)
	}
}

func TestDefinirFornecedorDoRepresentanteSoPeloAdministradorEPreservaNoAlterar(t *testing.T) {
	c := novoCenario(t)
	jose := c.novoRepresentante(t, "José Vendas", nil)
	fornecedorID := uuid.New()

	if _, err := c.representantes.DefinirFornecedor(c.ctx, portsin.DefinirFornecedorInput{
		UsuarioID: c.comercial, RepresentanteID: jose.ID, FornecedorID: &fornecedorID,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Fatalf("comercial definindo fornecedor = %v, esperado erro de permissão", err)
	}

	definido, err := c.representantes.DefinirFornecedor(c.ctx, portsin.DefinirFornecedorInput{
		UsuarioID: c.administrador, RepresentanteID: jose.ID, FornecedorID: &fornecedorID,
	})
	if err != nil || definido.FornecedorID == nil || *definido.FornecedorID != fornecedorID {
		t.Fatalf("definir fornecedor = %+v (%v)", definido, err)
	}

	alterado, err := c.representantes.Alterar(c.ctx, portsin.AlterarRepresentanteInput{
		UsuarioID: c.administrador, RepresentanteID: jose.ID,
		Dados: portsin.DadosRepresentanteInput{Nome: "José Representações", ComissaoProntaEntregaBP: 300},
	})
	if err != nil || alterado.FornecedorID == nil || *alterado.FornecedorID != fornecedorID {
		t.Fatalf("o PUT do cadastro não deveria apagar o fornecedor: %+v (%v)", alterado, err)
	}

	limpo, err := c.representantes.DefinirFornecedor(c.ctx, portsin.DefinirFornecedorInput{
		UsuarioID: c.administrador, RepresentanteID: jose.ID, FornecedorID: &uuid.Nil,
	})
	if err != nil || limpo.FornecedorID != nil {
		t.Fatalf("limpar fornecedor = %+v (%v)", limpo, err)
	}
}
