package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoClientes struct {
	ctx            context.Context
	banco          *sqlx.DB
	grupos         *postgres.ClienteGrupoRepository
	representantes *postgres.RepresentanteRepository
	clientes       *postgres.ClienteRepository
	obras          *postgres.ClienteObraRepository
	usuarioID      uuid.UUID
	empresaID      uuid.UUID
	leadID         uuid.UUID
}

func novoCenarioClientes(t *testing.T) *cenarioBancoClientes {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	ctx := context.Background()

	cargoID, err := postgres.NewCargoRepository(banco).Criar(ctx, novoCargo("Vendedor"))
	if err != nil {
		t.Fatalf("não criei o cargo do cenário: %v", err)
	}
	usuarioID, err := postgres.NewUsuarioRepository(banco).Criar(ctx, novoUsuario(cargoID, "José Vendas", "jose@vendas.exemplo.com"))
	if err != nil {
		t.Fatalf("não criei o usuário do cenário: %v", err)
	}

	funilID := helpers.CriarFunil(t, banco, "Vendas")
	etapas := helpers.CriarEtapas(t, banco, funilID, "Contato")

	return &cenarioBancoClientes{
		ctx:            ctx,
		banco:          banco,
		grupos:         postgres.NewClienteGrupoRepository(banco),
		representantes: postgres.NewRepresentanteRepository(banco),
		clientes:       postgres.NewClienteRepository(banco),
		obras:          postgres.NewClienteObraRepository(banco),
		usuarioID:      usuarioID,
		empresaID:      novaEmpresaNoBanco(t, postgres.NewEmpresaRepository(banco), "11.222.333/0001-81", "MIA"),
		leadID:         helpers.CriarLead(t, banco, etapas[0], "Construtora Águia"),
	}
}

func textoDe(valor string) *string {
	return &valor
}

func (c *cenarioBancoClientes) novoGrupo(t *testing.T, nome string) *domaincliente.Grupo {
	t.Helper()

	grupo, err := domaincliente.NovoGrupo(nome)
	if err != nil {
		t.Fatalf("grupo de teste inválido: %v", err)
	}
	if _, err := c.grupos.Criar(c.ctx, grupo); err != nil {
		t.Fatalf("criação do grupo falhou: %v", err)
	}

	return grupo
}

func (c *cenarioBancoClientes) novoRepresentante(t *testing.T, nome string, documento string, usuarioID *uuid.UUID) *domaincliente.Representante {
	t.Helper()

	dados := domaincliente.DadosRepresentante{
		Nome:                  nome,
		UsuarioID:             usuarioID,
		ComissaoProntaEntrega: domaincliente.PercentualDe(300),
		ComissaoImportacao:    domaincliente.PercentualDe(150),
	}
	if documento != "" {
		valor, err := domaincliente.NovoDocumento(documento)
		if err != nil {
			t.Fatalf("documento de teste inválido: %v", err)
		}
		dados.Documento = &valor
	}

	representante, err := domaincliente.NovoRepresentante(dados)
	if err != nil {
		t.Fatalf("representante de teste inválido: %v", err)
	}
	if _, err := c.representantes.Criar(c.ctx, representante); err != nil {
		t.Fatalf("criação do representante falhou: %v", err)
	}

	return representante
}

func (c *cenarioBancoClientes) clienteDeTeste(t *testing.T, documento, nome string, tipo domaincliente.TipoCliente) *domaincliente.Cliente {
	t.Helper()

	valor, err := domaincliente.NovoDocumento(documento)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	cliente, err := domaincliente.NovoCliente(valor, domaincliente.DadosCliente{
		Nome:   nome,
		Tipo:   tipo,
		UF:     domainempresa.UFDe("PB"),
		Cidade: "João Pessoa",
	}, nil)
	if err != nil {
		t.Fatalf("cliente de teste inválido: %v", err)
	}

	return cliente
}

func (c *cenarioBancoClientes) novoCliente(t *testing.T, documento, nome string, tipo domaincliente.TipoCliente) *domaincliente.Cliente {
	t.Helper()

	cliente := c.clienteDeTeste(t, documento, nome, tipo)
	if _, err := c.clientes.Criar(c.ctx, cliente); err != nil {
		t.Fatalf("criação do cliente falhou: %v", err)
	}

	return cliente
}

func TestGrupoDeClienteUnicoPeloNomeDeBusca(t *testing.T) {
	c := novoCenarioClientes(t)
	omega := c.novoGrupo(t, "Grupo Ômega")
	c.novoGrupo(t, "Alfa Incorporações")

	repetido, err := domaincliente.NovoGrupo("GRUPO OMEGA")
	if err != nil {
		t.Fatalf("grupo de teste inválido: %v", err)
	}
	if _, err := c.grupos.Criar(c.ctx, repetido); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("grupo com o mesmo nome de busca = %v, esperado conflito", err)
	}

	salvo, err := c.grupos.ObterPorNomeBusca(c.ctx, "grupo omega")
	if err != nil || salvo == nil || salvo.ID != omega.ID || salvo.Nome != "Grupo Ômega" {
		t.Fatalf("grupo por nome de busca = %+v (%v)", salvo, err)
	}

	encontrados, err := c.grupos.Listar(c.ctx, portsout.GrupoFiltro{PaginacaoFiltro: todasAsPaginas, Busca: "ÔMEG"})
	if err != nil || len(encontrados) != 1 || encontrados[0].ID != omega.ID {
		t.Errorf("busca por ômeg = %+v (%v)", encontrados, err)
	}

	omega.Desativar(time.Now().UTC())
	if ok, err := c.grupos.Atualizar(c.ctx, omega); err != nil || !ok {
		t.Fatalf("atualizar grupo = %v, %v", ok, err)
	}
	ativo := true
	ativos, err := c.grupos.Listar(c.ctx, portsout.GrupoFiltro{PaginacaoFiltro: todasAsPaginas, Ativo: &ativo})
	if err != nil || len(ativos) != 1 || ativos[0].Nome != "Alfa Incorporações" {
		t.Errorf("grupos ativos = %+v (%v)", ativos, err)
	}
}

func TestRepresentanteUnicoPorDocumentoEUsuario(t *testing.T) {
	c := novoCenarioClientes(t)
	jose := c.novoRepresentante(t, "José Vendas", "529.982.247-25", &c.usuarioID)
	c.novoRepresentante(t, "Maria Representações", "11.222.333/0001-81", nil)
	c.novoRepresentante(t, "Sem Documento", "", nil)

	salvo, err := c.representantes.Obter(c.ctx, jose.ID)
	if err != nil || salvo == nil || salvo.Documento == nil || salvo.Documento.Valor() != "52998224725" ||
		salvo.UsuarioID == nil || *salvo.UsuarioID != c.usuarioID || salvo.ComissaoProntaEntrega.PontosBase() != 300 {
		t.Fatalf("representante salvo = %+v (%v)", salvo, err)
	}
	if porUsuario, err := c.representantes.ObterPorUsuario(c.ctx, c.usuarioID); err != nil || porUsuario == nil || porUsuario.ID != jose.ID {
		t.Errorf("representante por usuário = %+v (%v)", porUsuario, err)
	}
	if porDocumento, err := c.representantes.ObterPorDocumento(c.ctx, "52998224725"); err != nil || porDocumento == nil || porDocumento.ID != jose.ID {
		t.Errorf("representante por documento = %+v (%v)", porDocumento, err)
	}
	if ausente, err := c.representantes.ObterPorUsuario(c.ctx, uuid.New()); err != nil || ausente != nil {
		t.Errorf("usuário sem representante = %+v, %v", ausente, err)
	}

	documento, _ := domaincliente.NovoDocumento("52998224725")
	mesmoDocumento, _ := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{Nome: "Outro", Documento: &documento})
	if _, err := c.representantes.Criar(c.ctx, mesmoDocumento); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("documento repetido = %v, esperado conflito", err)
	}
	mesmoUsuario, _ := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{Nome: "Outro", UsuarioID: &c.usuarioID})
	if _, err := c.representantes.Criar(c.ctx, mesmoUsuario); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("usuário repetido = %v, esperado conflito", err)
	}

	casos := map[string]int{"jose": 1, "REPRESENTAÇÕES": 1, "529.982": 1, "11222333": 1, "": 3, "%": 0}
	for busca, esperado := range casos {
		encontrados, err := c.representantes.Listar(c.ctx, portsout.RepresentanteFiltro{PaginacaoFiltro: todasAsPaginas, Busca: busca})
		if err != nil || len(encontrados) != esperado {
			t.Errorf("busca %q = %d (%v), esperado %d", busca, len(encontrados), err, esperado)
		}
	}

	if err := jose.Alterar(domaincliente.DadosRepresentante{Nome: "José Vendas", ComissaoImportacao: domaincliente.PercentualDe(200)}, time.Now().UTC()); err != nil {
		t.Fatalf("alterar falhou: %v", err)
	}
	if ok, err := c.representantes.Atualizar(c.ctx, jose); err != nil || !ok {
		t.Fatalf("atualizar representante = %v, %v", ok, err)
	}
	salvo, _ = c.representantes.Obter(c.ctx, jose.ID)
	if salvo.Documento != nil || salvo.UsuarioID != nil || salvo.ComissaoImportacao.PontosBase() != 200 {
		t.Errorf("representante atualizado = %+v", salvo)
	}
}

func TestClientePersisteTodosOsCamposEBuscaNormalizada(t *testing.T) {
	c := novoCenarioClientes(t)
	grupo := c.novoGrupo(t, "Grupo Águia")
	representante := c.novoRepresentante(t, "José Vendas", "", nil)

	cliente := c.clienteDeTeste(t, "11.222.333/0001-81", "Construtora Águia LTDA", domaincliente.TipoClienteConstrutora)
	cliente.NomeFantasia = textoDe("Águia")
	cliente.GrupoID = &grupo.ID
	cliente.RepresentanteID = &representante.ID
	cliente.EmpresaRelacionamentoID = &c.empresaID
	cliente.EmailNF = textoDe("nf@aguia.com.br")
	cliente.Telefone = textoDe("83 99999-0000")
	cliente.Observacoes = "pagamento no dia 10"
	cliente.ExigeNumeroPedidoNaNF = true
	cliente.ProibeAntecipacao = true
	cliente.LeadID = &c.leadID
	if _, err := c.clientes.Criar(c.ctx, cliente); err != nil {
		t.Fatalf("criação do cliente falhou: %v", err)
	}
	pessoa := c.novoCliente(t, "529.982.247-25", "José da Silva", domaincliente.TipoClientePF)
	c.novoCliente(t, "33.400.004/0001-16", "Loja Concreção", domaincliente.TipoClienteLoja)

	salvo, err := c.clientes.Obter(c.ctx, cliente.ID)
	if err != nil || salvo == nil {
		t.Fatalf("obter cliente = %+v (%v)", salvo, err)
	}
	if salvo.Documento.Valor() != "11222333000181" || salvo.Documento.Pessoa() != domaincliente.PessoaJuridica || salvo.NomeBusca != "construtora aguia ltda" {
		t.Errorf("documento e nome salvos = %+v", salvo)
	}
	if *salvo.GrupoID != grupo.ID || *salvo.RepresentanteID != representante.ID || *salvo.EmpresaRelacionamentoID != c.empresaID || *salvo.LeadID != c.leadID {
		t.Errorf("vínculos salvos = %+v", salvo)
	}
	if *salvo.NomeFantasia != "Águia" || *salvo.EmailNF != "nf@aguia.com.br" || !salvo.ExigeNumeroPedidoNaNF || !salvo.ProibeAntecipacao || salvo.UF != "PB" {
		t.Errorf("dados salvos = %+v", salvo)
	}
	if porDocumento, err := c.clientes.ObterPorDocumento(c.ctx, "52998224725"); err != nil || porDocumento == nil || porDocumento.ID != pessoa.ID || porDocumento.Documento.Pessoa() != domaincliente.PessoaFisica {
		t.Errorf("cliente por documento = %+v (%v)", porDocumento, err)
	}

	ativo := true
	casos := []struct {
		nome     string
		filtro   portsout.ClienteFiltro
		esperado int
	}{
		{"todos", portsout.ClienteFiltro{}, 3},
		{"busca sem acento", portsout.ClienteFiltro{Busca: "aguia"}, 1},
		{"busca com acento e caixa", portsout.ClienteFiltro{Busca: "CONCREÇÃO"}, 1},
		{"busca por documento mascarado", portsout.ClienteFiltro{Busca: "529.982"}, 1},
		{"busca por raiz de CNPJ", portsout.ClienteFiltro{Busca: "11222333"}, 1},
		{"curinga escapado", portsout.ClienteFiltro{Busca: "_"}, 0},
		{"só pontuação", portsout.ClienteFiltro{Busca: "./-"}, 0},
		{"grupo", portsout.ClienteFiltro{GrupoID: &grupo.ID}, 1},
		{"representante", portsout.ClienteFiltro{RepresentanteID: &representante.ID}, 1},
		{"tipo", portsout.ClienteFiltro{Tipo: domaincliente.TipoClientePF}, 1},
		{"uf", portsout.ClienteFiltro{UF: "SP"}, 0},
		{"ativos", portsout.ClienteFiltro{Ativo: &ativo}, 3},
		{"lead", portsout.ClienteFiltro{LeadID: &c.leadID}, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.filtro.PaginacaoFiltro = todasAsPaginas
			encontrados, err := c.clientes.Listar(c.ctx, caso.filtro)
			if err != nil || len(encontrados) != caso.esperado {
				t.Errorf("%d clientes (%v), esperado %d", len(encontrados), err, caso.esperado)
			}
		})
	}

	todos, _ := c.clientes.Listar(c.ctx, portsout.ClienteFiltro{PaginacaoFiltro: todasAsPaginas})
	if todos[0].NomeBusca != "construtora aguia ltda" || todos[2].NomeBusca != "loja concrecao" {
		t.Errorf("ordem pelo nome de busca = %s, %s, %s", todos[0].NomeBusca, todos[1].NomeBusca, todos[2].NomeBusca)
	}
}

func TestClienteRespeitaUnicidadeEVinculos(t *testing.T) {
	c := novoCenarioClientes(t)
	cliente := c.novoCliente(t, "11.222.333/0001-81", "Construtora Águia", domaincliente.TipoClienteConstrutora)

	mesmoDocumento := c.clienteDeTeste(t, "11222333000181", "Outro nome", domaincliente.TipoClienteLoja)
	if _, err := c.clientes.Criar(c.ctx, mesmoDocumento); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("documento repetido = %v, esperado conflito", err)
	}

	grupoInexistente := c.clienteDeTeste(t, "33400004000116", "Loja", domaincliente.TipoClienteLoja)
	grupoID := uuid.New()
	grupoInexistente.GrupoID = &grupoID
	if _, err := c.clientes.Criar(c.ctx, grupoInexistente); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("grupo inexistente = %v, esperado erro de validação", err)
	}

	cliente.LeadID = &c.leadID
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE cliente SET lead_id = $1 WHERE id = $2`, c.leadID, cliente.ID); err != nil {
		t.Fatalf("vincular lead falhou: %v", err)
	}
	segundoDoLead := c.clienteDeTeste(t, "52998224725", "José", domaincliente.TipoClientePF)
	segundoDoLead.LeadID = &c.leadID
	if _, err := c.clientes.Criar(c.ctx, segundoDoLead); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("lead convertido duas vezes = %v, esperado conflito", err)
	}
	if err := postgres.NewLeadRepository(c.banco).Remover(c.ctx, c.leadID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("excluir lead convertido = %v, esperado erro de validação", err)
	}

	obra, err := domaincliente.NovaObra(cliente.ID, domaincliente.DadosObra{
		Nome:     "Residencial Mar Azul",
		Endereco: domaincliente.Endereco{Logradouro: "Av. Beira Mar", Cidade: "Cabedelo", UF: domainempresa.UFDe("PB")},
	})
	if err != nil {
		t.Fatalf("obra de teste inválida: %v", err)
	}
	if _, err := c.obras.Criar(c.ctx, obra); err != nil {
		t.Fatalf("criação da obra falhou: %v", err)
	}
	if _, err := c.clientes.Remover(c.ctx, cliente.ID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remover cliente com obra = %v, esperado erro de validação", err)
	}

	semVinculo := c.novoCliente(t, "33400004000116", "Loja Avulsa", domaincliente.TipoClienteLoja)
	if removido, err := c.clientes.Remover(c.ctx, semVinculo.ID); err != nil || !removido {
		t.Errorf("remover cliente sem vínculo = %v, %v", removido, err)
	}
	if removido, err := c.clientes.Remover(c.ctx, uuid.New()); err != nil || removido {
		t.Errorf("remover cliente inexistente = %v, %v", removido, err)
	}

	if err := cliente.Alterar(domaincliente.DadosCliente{
		Nome: "Águia Engenharia", Tipo: domaincliente.TipoClienteConstrutora, UF: domainempresa.UFDe("CE"), Cidade: "Fortaleza",
	}, time.Now().UTC()); err != nil {
		t.Fatalf("alterar falhou: %v", err)
	}
	if ok, err := c.clientes.Atualizar(c.ctx, cliente); err != nil || !ok {
		t.Fatalf("atualizar cliente = %v, %v", ok, err)
	}
	salvo, _ := c.clientes.Obter(c.ctx, cliente.ID)
	if salvo.NomeBusca != "aguia engenharia" || salvo.UF != "CE" || salvo.LeadID == nil || salvo.Documento.Valor() != "11222333000181" {
		t.Errorf("cliente atualizado = %+v", salvo)
	}
}

func TestObrasDoClienteUnicasPorNome(t *testing.T) {
	c := novoCenarioClientes(t)
	aguia := c.novoCliente(t, "11222333000181", "Construtora Águia", domaincliente.TipoClienteConstrutora)
	omega := c.novoCliente(t, "33400004000116", "Construtora Ômega", domaincliente.TipoClienteConstrutora)
	cnpj, _ := domainempresa.NovoCNPJ("11.222.333/0001-81")

	nova := func(clienteID uuid.UUID, nome string) (*domaincliente.Obra, error) {
		obra, err := domaincliente.NovaObra(clienteID, domaincliente.DadosObra{
			Nome: nome,
			CNPJ: &cnpj,
			Endereco: domaincliente.Endereco{
				Logradouro: "Av. Beira Mar", Numero: textoDe("100"), Cidade: "Cabedelo", UF: domainempresa.UFDe("PB"), CEP: textoDe("58310000"),
			},
		})
		if err != nil {
			t.Fatalf("obra de teste inválida: %v", err)
		}
		_, err = c.obras.Criar(c.ctx, obra)

		return obra, err
	}

	marAzul, err := nova(aguia.ID, "Residencial Mar Azul")
	if err != nil {
		t.Fatalf("criação da obra falhou: %v", err)
	}
	if _, err := nova(aguia.ID, "RESIDENCIAL MAR AZUL"); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("obra repetida no mesmo cliente = %v, esperado conflito", err)
	}
	if _, err := nova(omega.ID, "Residencial Mar Azul"); err != nil {
		t.Errorf("mesmo nome em outro cliente = %v, esperado sucesso", err)
	}
	if _, err := nova(aguia.ID, "Edifício Sol"); err != nil {
		t.Fatalf("segunda obra falhou: %v", err)
	}

	salva, err := c.obras.Obter(c.ctx, marAzul.ID)
	if err != nil || salva == nil || salva.CNPJ == nil || salva.CNPJ.Digitos() != "11222333000181" || *salva.Endereco.CEP != "58310000" || *salva.Endereco.Numero != "100" {
		t.Fatalf("obra salva = %+v (%v)", salva, err)
	}

	marAzul.Desativar(time.Now().UTC())
	if ok, err := c.obras.Atualizar(c.ctx, marAzul); err != nil || !ok {
		t.Fatalf("atualizar obra = %v, %v", ok, err)
	}

	ativa := true
	casos := []struct {
		nome     string
		filtro   portsout.ObraFiltro
		esperado int
	}{
		{"todas da Águia", portsout.ObraFiltro{ClienteID: aguia.ID}, 2},
		{"ativas da Águia", portsout.ObraFiltro{ClienteID: aguia.ID, Ativo: &ativa}, 1},
		{"da Ômega", portsout.ObraFiltro{ClienteID: omega.ID}, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.filtro.PaginacaoFiltro = todasAsPaginas
			obras, err := c.obras.Listar(c.ctx, caso.filtro)
			if err != nil || len(obras) != caso.esperado {
				t.Errorf("%d obras (%v), esperado %d", len(obras), err, caso.esperado)
			}
		})
	}
}
