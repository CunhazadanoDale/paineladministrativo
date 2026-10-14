package fornecedor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	fornecedorusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/fornecedor"
	"github.com/google/uuid"
)

type cenario struct {
	ctx           context.Context
	usecase       *fornecedorusecases.FornecedorUsecaseImpl
	fornecedores  *repositorioFornecedores
	usuarios      *repositorioUsuarios
	cargos        *repositorioCargos
	administrador uuid.UUID
	financeiro    uuid.UUID
	comercial     uuid.UUID
	semCargo      uuid.UUID
}

func novoCenario() *cenario {
	c := &cenario{
		ctx:          context.Background(),
		fornecedores: &repositorioFornecedores{itens: map[uuid.UUID]*domainfornecedor.Fornecedor{}},
		usuarios:     &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}},
		cargos:       &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}},
	}
	c.usecase = fornecedorusecases.NewFornecedorUsecase(c.fornecedores, c.usuarios, c.cargos)

	c.administrador = c.novoUsuario(&domainusuarios.Cargo{Nome: "Diretoria", Administrador: true})
	c.financeiro = c.novoUsuario(&domainusuarios.Cargo{Nome: "Tesouraria", Financeiro: true})
	c.comercial = c.novoUsuario(&domainusuarios.Cargo{Nome: "Vendas", Comercial: true})
	c.semCargo = uuid.New()
	c.usuarios.itens[c.semCargo] = &domainusuarios.Usuario{ID: c.semCargo, CargoID: uuid.New()}

	return c
}

func (c *cenario) novoUsuario(cargo *domainusuarios.Cargo) uuid.UUID {
	cargo.ID = uuid.New()
	c.cargos.itens[cargo.ID] = cargo

	id := uuid.New()
	c.usuarios.itens[id] = &domainusuarios.Usuario{ID: id, CargoID: cargo.ID}

	return id
}

func texto(valor string) *string {
	return &valor
}

func criarNacional(documento, nome string) portsin.CriarFornecedorInput {
	return portsin.CriarFornecedorInput{
		Pais:      "br",
		Documento: texto(documento),
		Dados: portsin.DadosFornecedorInput{
			Nome:   nome,
			UF:     texto("pb"),
			Cidade: texto("João Pessoa"),
			Email:  texto("vendas@exemplo.com.br"),
		},
	}
}

func criarEstrangeiro(pais, identificacao, nome string) portsin.CriarFornecedorInput {
	return portsin.CriarFornecedorInput{
		Pais: pais,
		Dados: portsin.DadosFornecedorInput{
			IdentificacaoEstrangeira: texto(identificacao),
			Nome:                     nome,
			MoedaPadrao:              "usd",
		},
	}
}

func (c *cenario) criar(t *testing.T, usuarioID uuid.UUID, input portsin.CriarFornecedorInput) *domainfornecedor.Fornecedor {
	t.Helper()

	input.UsuarioID = usuarioID
	fornecedor, err := c.usecase.Criar(c.ctx, input)
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	return fornecedor
}

func conferirErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) {
		t.Fatalf("erro = %v, esperado %v", err, sentinela)
	}
	if trecho != "" && !strings.Contains(err.Error(), trecho) {
		t.Errorf("erro %q não contém %q", err.Error(), trecho)
	}
}

func TestCriarFornecedorNacionalPeloFinanceiro(t *testing.T) {
	c := novoCenario()

	fornecedor := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica Paraíba"))

	if fornecedor.Pais != domainfornecedor.PaisBrasil || fornecedor.Documento.Valor() != "11222333000181" {
		t.Errorf("país %q e documento %q", fornecedor.Pais, fornecedor.Documento.Valor())
	}
	if fornecedor.MoedaPadrao != domainfornecedor.MoedaReal || fornecedor.UF == nil || *fornecedor.UF != "PB" || fornecedor.Versao != 1 {
		t.Errorf("fornecedor criado = %+v", fornecedor)
	}
	if _, ok := c.fornecedores.itens[fornecedor.ID]; !ok {
		t.Error("fornecedor não foi gravado")
	}
}

func TestCriarFornecedorEstrangeiroPeloAdministrador(t *testing.T) {
	c := novoCenario()

	fornecedor := c.criar(t, c.administrador, criarEstrangeiro(" cn ", "91330200ma2", "Foshan Ceramics"))

	if fornecedor.Pais != "CN" || !fornecedor.Documento.Vazio() || *fornecedor.IdentificacaoEstrangeira != "91330200MA2" {
		t.Errorf("fornecedor criado = %+v", fornecedor)
	}
	if fornecedor.MoedaPadrao != domainfornecedor.MoedaDolar {
		t.Errorf("moeda = %q", fornecedor.MoedaPadrao)
	}
}

func TestCriarFornecedorDocumentoEmBrancoNoEstrangeiroEhIgnorado(t *testing.T) {
	c := novoCenario()

	input := criarEstrangeiro("US", "", "Ohio Tools")
	input.Documento = texto("   ")
	fornecedor := c.criar(t, c.financeiro, input)

	if !fornecedor.Documento.Vazio() || fornecedor.IdentificacaoEstrangeira != nil {
		t.Errorf("fornecedor criado = %+v", fornecedor)
	}
}

func TestCriarFornecedorExigePerfilFinanceiro(t *testing.T) {
	c := novoCenario()
	input := criarNacional("11.222.333/0001-81", "Cerâmica")

	casos := []struct {
		nome      string
		usuarioID uuid.UUID
		sentinela error
	}{
		{"comercial", c.comercial, domain.ErrPermissao},
		{"usuário sem cargo", c.semCargo, domain.ErrPermissao},
		{"usuário inexistente", uuid.New(), domain.ErrNotFound},
		{"usuário não informado", uuid.Nil, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			input.UsuarioID = caso.usuarioID
			_, err := c.usecase.Criar(c.ctx, input)
			conferirErro(t, err, caso.sentinela, "")
		})
	}
	if len(c.fornecedores.itens) != 0 {
		t.Error("nenhum fornecedor deveria ter sido gravado")
	}
}

func TestCriarFornecedorRecusaDadosInvalidos(t *testing.T) {
	c := novoCenario()

	estrangeiroComDocumento := criarEstrangeiro("CN", "", "Foshan")
	estrangeiroComDocumento.Documento = texto("11.222.333/0001-81")

	nacionalSemDocumento := criarNacional("", "Cerâmica")

	documentoInvalido := criarNacional("11.222.333/0001-00", "Cerâmica")

	paisInvalido := criarEstrangeiro("CHN", "", "Foshan")

	moedaInvalida := criarNacional("11.222.333/0001-81", "Cerâmica")
	moedaInvalida.Dados.MoedaPadrao = "GBP"

	ufInvalida := criarNacional("11.222.333/0001-81", "Cerâmica")
	ufInvalida.Dados.UF = texto("XX")

	nomeVazio := criarNacional("11.222.333/0001-81", "  ")

	casos := map[string]portsin.CriarFornecedorInput{
		"estrangeiro com documento": estrangeiroComDocumento,
		"nacional sem documento":    nacionalSemDocumento,
		"documento inválido":        documentoInvalido,
		"país inválido":             paisInvalido,
		"moeda fora da lista":       moedaInvalida,
		"uf inválida":               ufInvalida,
		"nome vazio":                nomeVazio,
	}

	for nome, input := range casos {
		t.Run(nome, func(t *testing.T) {
			input.UsuarioID = c.financeiro
			_, err := c.usecase.Criar(c.ctx, input)
			conferirErro(t, err, domain.ErrValidacao, "")
		})
	}
}

func TestCriarFornecedorRecusaDuplicidade(t *testing.T) {
	c := novoCenario()
	c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica A"))
	c.criar(t, c.financeiro, criarEstrangeiro("CN", "ABC123", "Foshan A"))

	input := criarNacional("11222333000181", "Cerâmica B")
	input.UsuarioID = c.financeiro
	_, err := c.usecase.Criar(c.ctx, input)
	conferirErro(t, err, domain.ErrConflito, "já existe um fornecedor com este documento")

	input = criarEstrangeiro("cn", " abc123 ", "Foshan B")
	input.UsuarioID = c.financeiro
	_, err = c.usecase.Criar(c.ctx, input)
	conferirErro(t, err, domain.ErrConflito, "já existe um fornecedor com esta identificação neste país")

	c.criar(t, c.financeiro, criarEstrangeiro("US", "ABC123", "Ohio"))
}

func TestCriarFornecedorRepassaConflitoDaCorrida(t *testing.T) {
	c := novoCenario()
	c.fornecedores.conflitoNaCriacao = true

	input := criarNacional("11.222.333/0001-81", "Cerâmica")
	input.UsuarioID = c.financeiro
	_, err := c.usecase.Criar(c.ctx, input)

	conferirErro(t, err, domain.ErrConflito, "já existe um fornecedor com este documento")
}

func TestCriarFornecedorRepassaFalhaDoBanco(t *testing.T) {
	c := novoCenario()
	c.fornecedores.falha = errBancoFora

	input := criarNacional("11.222.333/0001-81", "Cerâmica")
	input.UsuarioID = c.financeiro
	if _, err := c.usecase.Criar(c.ctx, input); !errors.Is(err, errBancoFora) {
		t.Errorf("erro = %v, esperado a falha do banco", err)
	}
}

func TestObterFornecedor(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	obtido, err := c.usecase.Obter(c.ctx, criado.ID)
	if err != nil || obtido.ID != criado.ID {
		t.Errorf("Obter = %v, %v", obtido, err)
	}

	_, err = c.usecase.Obter(c.ctx, uuid.New())
	conferirErro(t, err, domain.ErrNotFound, "fornecedor não encontrado")
}

func TestListarFornecedoresNormalizaFiltros(t *testing.T) {
	c := novoCenario()
	c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))
	c.criar(t, c.financeiro, criarEstrangeiro("CN", "", "Foshan"))

	ativo := true
	itens, err := c.usecase.Listar(c.ctx, portsin.ListarFornecedoresInput{
		Busca:  "  foshan ",
		Pais:   " cn ",
		Moeda:  "usd",
		Ativo:  &ativo,
		Filtro: domain.PaginacaoFiltro{Page: 0, Size: 0},
	})
	if err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}

	filtro := c.fornecedores.ultimoFiltro
	if filtro.Busca != "foshan" || filtro.Pais != "CN" || filtro.Moeda != domainfornecedor.MoedaDolar || filtro.Ativo == nil {
		t.Errorf("filtro repassado = %+v", filtro)
	}
	if filtro.Page < 1 || filtro.Size < 1 {
		t.Errorf("paginação não foi normalizada: %+v", filtro.PaginacaoFiltro)
	}
	if len(itens) != 1 || itens[0].Nome != "Foshan" {
		t.Errorf("itens = %+v", itens)
	}

	if _, err := c.usecase.Listar(c.ctx, portsin.ListarFornecedoresInput{}); err != nil {
		t.Fatalf("Listar sem filtro falhou: %v", err)
	}
	if c.fornecedores.ultimoFiltro.Pais != "" || c.fornecedores.ultimoFiltro.Moeda != "" {
		t.Errorf("filtro vazio virou %+v; país e moeda vazios não devem assumir BR e BRL", c.fornecedores.ultimoFiltro)
	}
}

func TestListarFornecedoresRecusaFiltroInvalido(t *testing.T) {
	c := novoCenario()

	_, err := c.usecase.Listar(c.ctx, portsin.ListarFornecedoresInput{Pais: "BRA"})
	conferirErro(t, err, domain.ErrValidacao, "país")

	_, err = c.usecase.Listar(c.ctx, portsin.ListarFornecedoresInput{Moeda: "GBP"})
	conferirErro(t, err, domain.ErrValidacao, "moeda")
}

func alterarInput(usuarioID, fornecedorID uuid.UUID, versao int, nome string) portsin.AlterarFornecedorInput {
	return portsin.AlterarFornecedorInput{
		UsuarioID:    usuarioID,
		FornecedorID: fornecedorID,
		Versao:       versao,
		Dados:        portsin.DadosFornecedorInput{Nome: nome, MoedaPadrao: "eur"},
	}
}

func TestAlterarFornecedorAvancaVersaoEMantemDocumento(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	alterado, err := c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, criado.ID, 1, "Cerâmica Nordeste"))
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}

	if alterado.Versao != 2 || alterado.Nome != "Cerâmica Nordeste" || alterado.MoedaPadrao != domainfornecedor.MoedaEuro {
		t.Errorf("fornecedor alterado = %+v", alterado)
	}
	if alterado.Documento.Valor() != "11222333000181" || alterado.UF != nil {
		t.Errorf("documento %q e uf %v", alterado.Documento.Valor(), alterado.UF)
	}
	if salvo := c.fornecedores.itens[criado.ID]; salvo.Versao != 2 || salvo.Nome != "Cerâmica Nordeste" {
		t.Errorf("fornecedor salvo = %+v", salvo)
	}
}

func TestAlterarFornecedorComVersaoVelhaDaConflito(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	if _, err := c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, criado.ID, 1, "Primeira")); err != nil {
		t.Fatalf("primeira alteração falhou: %v", err)
	}

	_, err := c.usecase.Alterar(c.ctx, alterarInput(c.administrador, criado.ID, 1, "Segunda"))
	conferirErro(t, err, domain.ErrConflito, "alterado por outra pessoa")

	if salvo := c.fornecedores.itens[criado.ID]; salvo.Nome != "Primeira" || salvo.Versao != 2 {
		t.Errorf("fornecedor salvo = %+v", salvo)
	}
}

func TestAlterarFornecedorPerdeACorridaNoBanco(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	c.fornecedores.antesDeAtualizar = func() {
		c.fornecedores.itens[criado.ID].Nome = "Gravado por outra pessoa"
		c.fornecedores.itens[criado.ID].Versao = 2
	}

	_, err := c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, criado.ID, 1, "Perdida"))
	conferirErro(t, err, domain.ErrConflito, "alterado por outra pessoa")

	if salvo := c.fornecedores.itens[criado.ID]; salvo.Nome != "Gravado por outra pessoa" || salvo.Versao != 2 {
		t.Errorf("a gravação perdida não deveria ter efeito: %+v", salvo)
	}
}

func TestAlterarFornecedorRemovidoDuranteAOperacao(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	c.fornecedores.antesDeAtualizar = func() {
		delete(c.fornecedores.itens, criado.ID)
	}

	_, err := c.usecase.Desativar(c.ctx, situacao(c.financeiro, criado.ID, 1))
	conferirErro(t, err, domain.ErrNotFound, "fornecedor não encontrado")
}

func TestAlterarFornecedorErrosDeEntrada(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	_, err := c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, criado.ID, 0, "Sem versão"))
	conferirErro(t, err, domain.ErrValidacao, "versão do fornecedor é obrigatória")

	_, err = c.usecase.Alterar(c.ctx, alterarInput(c.comercial, criado.ID, 1, "Comercial"))
	conferirErro(t, err, domain.ErrPermissao, "gerenciar fornecedores")

	_, err = c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, uuid.New(), 1, "Fantasma"))
	conferirErro(t, err, domain.ErrNotFound, "fornecedor não encontrado")

	_, err = c.usecase.Alterar(c.ctx, alterarInput(c.financeiro, criado.ID, 1, ""))
	conferirErro(t, err, domain.ErrValidacao, "nome do fornecedor")

	if salvo := c.fornecedores.itens[criado.ID]; salvo.Versao != 1 || salvo.Nome != "Cerâmica" {
		t.Errorf("fornecedor não deveria ter mudado: %+v", salvo)
	}
}

func TestAlterarFornecedorRecusaIdentificacaoDeOutro(t *testing.T) {
	c := novoCenario()
	c.criar(t, c.financeiro, criarEstrangeiro("CN", "ABC123", "Foshan A"))
	outro := c.criar(t, c.financeiro, criarEstrangeiro("CN", "XYZ789", "Foshan B"))

	input := alterarInput(c.financeiro, outro.ID, 1, "Foshan B")
	input.Dados.IdentificacaoEstrangeira = texto("abc123")
	_, err := c.usecase.Alterar(c.ctx, input)
	conferirErro(t, err, domain.ErrConflito, "já existe um fornecedor com esta identificação neste país")

	input.Dados.IdentificacaoEstrangeira = texto("xyz789")
	if _, err := c.usecase.Alterar(c.ctx, input); err != nil {
		t.Errorf("manter a própria identificação deveria passar: %v", err)
	}
}

func situacao(usuarioID, fornecedorID uuid.UUID, versao int) portsin.SituacaoFornecedorInput {
	return portsin.SituacaoFornecedorInput{UsuarioID: usuarioID, FornecedorID: fornecedorID, Versao: versao}
}

func TestDesativarEAtivarFornecedor(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))
	gravacoesAntes := c.fornecedores.gravacoes

	desativado, err := c.usecase.Desativar(c.ctx, situacao(c.financeiro, criado.ID, 1))
	if err != nil || desativado.Ativo || desativado.Versao != 2 {
		t.Fatalf("Desativar = %+v, %v", desativado, err)
	}

	denovo, err := c.usecase.Desativar(c.ctx, situacao(c.financeiro, criado.ID, 2))
	if err != nil || denovo.Ativo || denovo.Versao != 2 {
		t.Errorf("Desativar já inativo = %+v, %v; esperado sem mudança de versão", denovo, err)
	}
	if c.fornecedores.gravacoes != gravacoesAntes+1 {
		t.Errorf("gravações = %d, esperado %d; desativar de novo não deveria gravar", c.fornecedores.gravacoes, gravacoesAntes+1)
	}

	_, err = c.usecase.Ativar(c.ctx, situacao(c.financeiro, criado.ID, 1))
	conferirErro(t, err, domain.ErrConflito, "alterado por outra pessoa")

	ativado, err := c.usecase.Ativar(c.ctx, situacao(c.administrador, criado.ID, 2))
	if err != nil || !ativado.Ativo || ativado.Versao != 3 {
		t.Errorf("Ativar = %+v, %v", ativado, err)
	}

	_, err = c.usecase.Ativar(c.ctx, situacao(c.comercial, criado.ID, 3))
	conferirErro(t, err, domain.ErrPermissao, "")
}

func TestRemoverFornecedorSoPeloAdministrador(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))

	err := c.usecase.Remover(c.ctx, criado.ID, c.financeiro)
	conferirErro(t, err, domain.ErrPermissao, "excluir fornecedores")

	if err := c.usecase.Remover(c.ctx, criado.ID, c.administrador); err != nil {
		t.Fatalf("Remover falhou: %v", err)
	}
	if _, ok := c.fornecedores.itens[criado.ID]; ok {
		t.Error("fornecedor deveria ter sido removido")
	}

	err = c.usecase.Remover(c.ctx, criado.ID, c.administrador)
	conferirErro(t, err, domain.ErrNotFound, "fornecedor não encontrado")
}

func TestRemoverFornecedorRepassaErroDoBanco(t *testing.T) {
	c := novoCenario()
	criado := c.criar(t, c.financeiro, criarNacional("11.222.333/0001-81", "Cerâmica"))
	c.fornecedores.falha = domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")

	err := c.usecase.Remover(c.ctx, criado.ID, c.administrador)
	conferirErro(t, err, domain.ErrValidacao, "registro em uso")
}
