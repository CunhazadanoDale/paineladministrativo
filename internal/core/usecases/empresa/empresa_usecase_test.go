package empresa_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/empresa"
	"github.com/google/uuid"
)

type cenario struct {
	ctx           context.Context
	usecase       *empresa.EmpresaUsecaseImpl
	repo          *repositorioEmpresas
	usuarios      *repositorioUsuarios
	cargos        *repositorioCargos
	administrador uuid.UUID
	operador      uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	usuarios := &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
	cargos := &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
	repo := &repositorioEmpresas{
		itens:    map[uuid.UUID]*domainempresa.Empresa{},
		vinculos: map[uuid.UUID]domainempresa.Vinculos{},
		usuarios: usuarios,
		cargos:   cargos,
	}

	c := &cenario{
		ctx:      context.Background(),
		usecase:  empresa.NewEmpresaUsecase(repo, usuarios, cargos),
		repo:     repo,
		usuarios: usuarios,
		cargos:   cargos,
	}
	c.administrador = c.novoUsuario(true)
	c.operador = c.novoUsuario(false)

	return c
}

func (c *cenario) novoUsuario(administrador bool) uuid.UUID {
	cargo := &domainusuarios.Cargo{ID: uuid.New(), Nome: "Cargo", Ativo: true, Administrador: administrador}
	c.cargos.itens[cargo.ID] = cargo

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.usuarios.itens[usuario.ID] = usuario

	return usuario.ID
}

func (c *cenario) criar(t *testing.T, cnpj, nomeFantasia string) uuid.UUID {
	t.Helper()

	id, err := c.usecase.Criar(c.ctx, portsin.CriarEmpresaInput{
		UsuarioID:        c.administrador,
		CNPJ:             cnpj,
		RazaoSocial:      nomeFantasia + " LTDA",
		NomeFantasia:     nomeFantasia,
		UF:               "pb",
		RegimeTributario: "presumido",
	})
	if err != nil {
		t.Fatalf("criação da empresa %q falhou: %v", nomeFantasia, err)
	}

	return id
}

func TestCriarEmpresaGravaNormalizada(t *testing.T) {
	c := novoCenario(t)
	id := c.criar(t, "11.222.333/0001-81", "MIA")

	salva := c.repo.itens[id]
	if salva == nil || salva.CNPJ.Digitos() != "11222333000181" || salva.UF.String() != "PB" || !salva.Ativa {
		t.Errorf("empresa salva %+v, esperado CNPJ só com dígitos, UF PB e ativa", salva)
	}
}

func TestCriarEmpresaValidaEntradaEPerfil(t *testing.T) {
	c := novoCenario(t)
	c.criar(t, "11.222.333/0001-81", "MIA")

	base := portsin.CriarEmpresaInput{
		UsuarioID:        c.administrador,
		CNPJ:             "33.400.004/0001-16",
		RazaoSocial:      "Red Diamond LTDA",
		NomeFantasia:     "Red Diamond",
		UF:               "SP",
		RegimeTributario: "real",
	}

	casos := []struct {
		nome      string
		ajuste    func(*portsin.CriarEmpresaInput)
		sentinela error
	}{
		{"sem perfil de administrador", func(i *portsin.CriarEmpresaInput) { i.UsuarioID = c.operador }, domain.ErrPermissao},
		{"cnpj inválido", func(i *portsin.CriarEmpresaInput) { i.CNPJ = "33.400.004/0001-17" }, domain.ErrValidacao},
		{"uf inválida", func(i *portsin.CriarEmpresaInput) { i.UF = "XX" }, domain.ErrValidacao},
		{"regime inválido", func(i *portsin.CriarEmpresaInput) { i.RegimeTributario = "mei" }, domain.ErrValidacao},
		{"razão social vazia", func(i *portsin.CriarEmpresaInput) { i.RazaoSocial = " " }, domain.ErrValidacao},
		{"cnpj repetido", func(i *portsin.CriarEmpresaInput) { i.CNPJ = "11222333000181" }, domain.ErrConflito},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			input := base
			caso.ajuste(&input)

			if _, err := c.usecase.Criar(c.ctx, input); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	if len(c.repo.itens) != 1 {
		t.Errorf("%d empresas gravadas, esperado só a primeira", len(c.repo.itens))
	}
}

func TestCriarEmpresaEmCorridaPeloMesmoCNPJDevolveAMensagemDoCNPJ(t *testing.T) {
	c := novoCenario(t)
	c.repo.conflitoNaCriacao = true

	_, err := c.usecase.Criar(c.ctx, portsin.CriarEmpresaInput{
		UsuarioID:        c.administrador,
		CNPJ:             "11.222.333/0001-81",
		RazaoSocial:      "MIA LTDA",
		NomeFantasia:     "MIA",
		UF:               "PB",
		RegimeTributario: "real",
	})
	if !errors.Is(err, domain.ErrConflito) || err.Error() != "operação em conflito com o estado atual: já existe uma empresa com este cnpj" {
		t.Errorf("erro %v, esperado conflito com a mensagem do CNPJ", err)
	}
}

func TestListarEObterRespeitamOAcesso(t *testing.T) {
	c := novoCenario(t)
	mia := c.criar(t, "11.222.333/0001-81", "MIA")
	red := c.criar(t, "33.400.004/0001-16", "Red Diamond")

	if err := c.usecase.DefinirVinculos(c.ctx, portsin.DefinirVinculosInput{
		UsuarioID:  c.administrador,
		AlvoID:     c.operador,
		EmpresaIDs: []uuid.UUID{mia},
	}); err != nil {
		t.Fatalf("vínculo falhou: %v", err)
	}

	doOperador, err := c.usecase.Listar(c.ctx, portsin.ListarEmpresasInput{UsuarioID: c.operador})
	if err != nil {
		t.Fatalf("listagem do operador falhou: %v", err)
	}
	if len(doOperador) != 1 || doOperador[0].ID != mia {
		t.Errorf("operador listou %d empresas, esperado só a MIA", len(doOperador))
	}

	doAdministrador, err := c.usecase.Listar(c.ctx, portsin.ListarEmpresasInput{UsuarioID: c.administrador})
	if err != nil || len(doAdministrador) != 2 {
		t.Errorf("administrador listou %d empresas (%v), esperado 2", len(doAdministrador), err)
	}

	if _, err := c.usecase.Obter(c.ctx, mia, c.operador); err != nil {
		t.Errorf("operador não obteve empresa do acesso: %v", err)
	}
	if _, err := c.usecase.Obter(c.ctx, red, c.operador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v, esperado erro de permissão", err)
	}
	if _, err := c.usecase.Obter(c.ctx, uuid.New(), c.administrador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("empresa inexistente = %v, esperado não encontrado", err)
	}
}

func TestUsuarioSemVinculoNaoListaNenhumaEmpresa(t *testing.T) {
	c := novoCenario(t)
	c.criar(t, "11.222.333/0001-81", "MIA")

	empresas, err := c.usecase.Listar(c.ctx, portsin.ListarEmpresasInput{UsuarioID: c.operador})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(empresas) != 0 {
		t.Errorf("usuário sem vínculo listou %d empresas, esperado nenhuma", len(empresas))
	}
}

func TestAlterarEAlternarAtivaPreservamCNPJ(t *testing.T) {
	c := novoCenario(t)
	id := c.criar(t, "11.222.333/0001-81", "MIA")

	if err := c.usecase.Alterar(c.ctx, portsin.AlterarEmpresaInput{
		UsuarioID:        c.administrador,
		EmpresaID:        id,
		RazaoSocial:      "MIA Construtora LTDA",
		NomeFantasia:     "Gigante PB",
		UF:               "PB",
		RegimeTributario: "real",
	}); err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}
	if err := c.usecase.AlternarAtiva(c.ctx, portsin.AlternarAtivaEmpresaInput{
		UsuarioID: c.administrador,
		EmpresaID: id,
		Ativa:     false,
	}); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	salva := c.repo.itens[id]
	if salva.NomeFantasia != "Gigante PB" || salva.RegimeTributario != domainempresa.RegimeReal || salva.Ativa {
		t.Errorf("empresa salva %+v, esperado nome e regime novos e inativa", salva)
	}
	if salva.CNPJ.Digitos() != "11222333000181" {
		t.Errorf("CNPJ mudou para %q", salva.CNPJ.Digitos())
	}

	if err := c.usecase.AlternarAtiva(c.ctx, portsin.AlternarAtivaEmpresaInput{
		UsuarioID: c.operador,
		EmpresaID: id,
		Ativa:     true,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("operador ativando empresa = %v, esperado erro de permissão", err)
	}
	if err := c.usecase.Alterar(c.ctx, portsin.AlterarEmpresaInput{
		UsuarioID:        c.administrador,
		EmpresaID:        uuid.New(),
		RazaoSocial:      "X",
		NomeFantasia:     "X",
		UF:               "PB",
		RegimeTributario: "real",
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("alteração de empresa inexistente = %v, esperado não encontrado", err)
	}
}

func TestDefinirVinculosValidaEmpresasEIgnoraRepeticao(t *testing.T) {
	c := novoCenario(t)
	mia := c.criar(t, "11.222.333/0001-81", "MIA")

	if err := c.usecase.DefinirVinculos(c.ctx, portsin.DefinirVinculosInput{
		UsuarioID:  c.administrador,
		AlvoID:     c.operador,
		EmpresaIDs: []uuid.UUID{mia, uuid.New()},
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("empresa inexistente = %v, esperado erro de validação", err)
	}

	if err := c.usecase.DefinirVinculos(c.ctx, portsin.DefinirVinculosInput{
		UsuarioID:   c.administrador,
		AlvoID:      c.operador,
		AcessoTodas: true,
		EmpresaIDs:  []uuid.UUID{mia, mia},
	}); err != nil {
		t.Fatalf("definição de vínculos falhou: %v", err)
	}

	vinculos, err := c.usecase.ObterVinculos(c.ctx, c.operador, c.administrador)
	if err != nil {
		t.Fatalf("consulta de vínculos falhou: %v", err)
	}
	if !vinculos.AcessoTodas || len(vinculos.EmpresaIDs) != 1 {
		t.Errorf("vínculos %+v, esperado acesso total e a MIA uma vez", vinculos)
	}

	if err := c.usecase.DefinirVinculos(c.ctx, portsin.DefinirVinculosInput{
		UsuarioID: c.operador,
		AlvoID:    c.operador,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("operador definindo o próprio acesso = %v, esperado erro de permissão", err)
	}
	if _, err := c.usecase.ObterVinculos(c.ctx, uuid.New(), c.administrador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("vínculos de usuário inexistente = %v, esperado não encontrado", err)
	}
}

func TestRemoverEmpresa(t *testing.T) {
	c := novoCenario(t)
	livre := c.criar(t, "11.222.333/0001-81", "MIA")
	vinculada := c.criar(t, "33.400.004/0001-16", "Red Diamond")

	if err := c.usecase.DefinirVinculos(c.ctx, portsin.DefinirVinculosInput{
		UsuarioID:  c.administrador,
		AlvoID:     c.operador,
		EmpresaIDs: []uuid.UUID{vinculada},
	}); err != nil {
		t.Fatalf("vínculo falhou: %v", err)
	}

	if err := c.usecase.Remover(c.ctx, vinculada, c.administrador); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remoção de empresa vinculada = %v, esperado erro de validação", err)
	}
	if err := c.usecase.Remover(c.ctx, livre, c.operador); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("remoção pelo operador = %v, esperado erro de permissão", err)
	}
	if err := c.usecase.Remover(c.ctx, livre, c.administrador); err != nil {
		t.Errorf("remoção de empresa livre falhou: %v", err)
	}
	if err := c.usecase.Remover(c.ctx, livre, c.administrador); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("segunda remoção = %v, esperado não encontrado", err)
	}
}
