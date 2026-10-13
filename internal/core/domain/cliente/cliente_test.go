package cliente_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

func texto(valor string) *string {
	return &valor
}

func documentoDeTeste(t *testing.T, valor string) domaincliente.Documento {
	t.Helper()

	documento, err := domaincliente.NovoDocumento(valor)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}

	return documento
}

func dadosDeConstrutora() domaincliente.DadosCliente {
	return domaincliente.DadosCliente{
		Nome:                  "  Construtora Águia  LTDA ",
		NomeFantasia:          texto("  Águia "),
		Tipo:                  domaincliente.TipoClienteConstrutora,
		UF:                    domainempresa.UFDe("PB"),
		Cidade:                " João Pessoa ",
		EmailNF:               texto(" nf@aguia.com.br "),
		Telefone:              texto("   "),
		Observacoes:           "  pagamento sempre no dia 10 ",
		ExigeNumeroPedidoNaNF: true,
	}
}

func TestNovoClienteNormalizaENasceAtivo(t *testing.T) {
	leadID := uuid.New()
	nulo := uuid.Nil
	dados := dadosDeConstrutora()
	dados.GrupoID = &nulo

	cliente, err := domaincliente.NovoCliente(documentoDeTeste(t, "11.222.333/0001-81"), dados, &leadID)
	if err != nil {
		t.Fatalf("NovoCliente falhou: %v", err)
	}

	if cliente.ID == uuid.Nil || !cliente.Ativo || cliente.LeadID == nil || *cliente.LeadID != leadID {
		t.Errorf("cliente criado = %+v", cliente)
	}
	if cliente.Nome != "Construtora Águia  LTDA" || cliente.NomeBusca != "construtora aguia ltda" {
		t.Errorf("nome %q e nome de busca %q", cliente.Nome, cliente.NomeBusca)
	}
	if cliente.NomeFantasia == nil || *cliente.NomeFantasia != "Águia" || cliente.Cidade != "João Pessoa" {
		t.Errorf("nome fantasia %v e cidade %q", cliente.NomeFantasia, cliente.Cidade)
	}
	if cliente.EmailNF == nil || *cliente.EmailNF != "nf@aguia.com.br" || cliente.Telefone != nil {
		t.Errorf("e-mail %v e telefone %v", cliente.EmailNF, cliente.Telefone)
	}
	if cliente.Observacoes != "pagamento sempre no dia 10" || !cliente.ExigeNumeroPedidoNaNF || cliente.ProibeAntecipacao {
		t.Errorf("observações %q e regras %v/%v", cliente.Observacoes, cliente.ExigeNumeroPedidoNaNF, cliente.ProibeAntecipacao)
	}
	if cliente.GrupoID != nil {
		t.Error("grupo zerado deveria virar nulo")
	}
	if cliente.Documento.Pessoa() != domaincliente.PessoaJuridica {
		t.Error("CNPJ deveria dar pessoa jurídica")
	}
}

func TestNovoClienteValidaDados(t *testing.T) {
	cnpj := documentoDeTeste(t, "11222333000181")
	cpf := documentoDeTeste(t, "52998224725")

	casos := []struct {
		nome      string
		documento domaincliente.Documento
		alterar   func(*domaincliente.DadosCliente)
		mensagem  string
	}{
		{"sem documento", domaincliente.Documento{}, func(*domaincliente.DadosCliente) {}, "documento do cliente é obrigatório"},
		{"sem nome", cnpj, func(d *domaincliente.DadosCliente) { d.Nome = " " }, "nome do cliente é obrigatório"},
		{"nome longo", cnpj, func(d *domaincliente.DadosCliente) { d.Nome = strings.Repeat("a", 201) }, "nome do cliente deve ter no máximo 200 caracteres"},
		{"sem cidade", cnpj, func(d *domaincliente.DadosCliente) { d.Cidade = "" }, "cidade do cliente é obrigatória"},
		{"tipo inválido", cnpj, func(d *domaincliente.DadosCliente) { d.Tipo = "revenda" }, ""},
		{"pf com CNPJ", cnpj, func(d *domaincliente.DadosCliente) { d.Tipo = domaincliente.TipoClientePF }, "cliente do tipo pf exige CPF, e CPF exige o tipo pf"},
		{"CPF de construtora", cpf, func(*domaincliente.DadosCliente) {}, "cliente do tipo pf exige CPF, e CPF exige o tipo pf"},
		{"uf inválida", cnpj, func(d *domaincliente.DadosCliente) { d.UF = "XX" }, ""},
		{"e-mail sem arroba", cnpj, func(d *domaincliente.DadosCliente) { d.EmailNF = texto("nf.aguia.com.br") }, "e-mail para NF inválido"},
		{"dois e-mails", cnpj, func(d *domaincliente.DadosCliente) { d.EmailNF = texto("a@b.com; c@d.com") }, "e-mail para NF inválido"},
		{"observação longa", cnpj, func(d *domaincliente.DadosCliente) { d.Observacoes = strings.Repeat("a", 2001) }, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeConstrutora()
			caso.alterar(&dados)

			_, err := domaincliente.NovoCliente(caso.documento, dados, nil)
			if !errors.Is(err, domain.ErrValidacao) {
				t.Fatalf("erro %v, esperado erro de validação", err)
			}
			if caso.mensagem != "" && err.Error() != domain.ErroValidacao(caso.mensagem).Error() {
				t.Errorf("mensagem %q, esperado %q", err.Error(), caso.mensagem)
			}
		})
	}

	dados := dadosDeConstrutora()
	dados.Tipo = domaincliente.TipoClientePF
	if _, err := domaincliente.NovoCliente(cpf, dados, nil); err != nil {
		t.Errorf("pessoa física com CPF = %v, esperado sucesso", err)
	}
}

func TestAlterarClientePreservaDocumentoELead(t *testing.T) {
	leadID := uuid.New()
	cliente, err := domaincliente.NovoCliente(documentoDeTeste(t, "11222333000181"), dadosDeConstrutora(), &leadID)
	if err != nil {
		t.Fatalf("NovoCliente falhou: %v", err)
	}
	documento := cliente.Documento
	agora := time.Now().UTC().Add(time.Hour)

	dados := dadosDeConstrutora()
	dados.Nome = "Águia Engenharia"
	dados.ProibeAntecipacao = true
	if err := cliente.Alterar(dados, agora); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if cliente.Documento != documento || cliente.LeadID == nil || *cliente.LeadID != leadID {
		t.Error("alterar não pode mudar documento nem lead")
	}
	if cliente.NomeBusca != "aguia engenharia" || !cliente.ProibeAntecipacao || !cliente.AtualizadoEm.Equal(agora) {
		t.Errorf("cliente alterado = %+v", cliente)
	}

	invalidos := dadosDeConstrutora()
	invalidos.Nome = ""
	if err := cliente.Alterar(invalidos, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alterar com nome vazio = %v, esperado erro de validação", err)
	}
	if cliente.Nome != "Águia Engenharia" {
		t.Error("alteração inválida não pode mudar o cliente")
	}

	cliente.Desativar(agora)
	if cliente.Ativo {
		t.Error("cliente deveria estar inativo")
	}
	cliente.Ativar(agora)
	if !cliente.Ativo {
		t.Error("cliente deveria voltar a ficar ativo")
	}
}

func TestRepresentanteValidaENormaliza(t *testing.T) {
	documento := documentoDeTeste(t, "52998224725")
	usuarioID := uuid.New()
	nulo := uuid.Nil

	representante, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{
		Nome:                  " José  Vendas ",
		Documento:             &documento,
		UsuarioID:             &usuarioID,
		Email:                 texto(" jose@vendas.com "),
		ComissaoProntaEntrega: domaincliente.PercentualDe(300),
		ComissaoImportacao:    domaincliente.PercentualDe(150),
	})
	if err != nil {
		t.Fatalf("NovoRepresentante falhou: %v", err)
	}
	if representante.Nome != "José  Vendas" || representante.NomeBusca != "jose vendas" || !representante.Ativo {
		t.Errorf("representante criado = %+v", representante)
	}
	if representante.Email == nil || *representante.Email != "jose@vendas.com" || representante.ComissaoProntaEntrega.PontosBase() != 300 {
		t.Errorf("e-mail %v e comissão %d", representante.Email, representante.ComissaoProntaEntrega.PontosBase())
	}

	vazio := domaincliente.Documento{}
	if err := representante.Alterar(domaincliente.DadosRepresentante{Nome: "José", Documento: &vazio, UsuarioID: &nulo}, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if representante.Documento != nil || representante.UsuarioID != nil {
		t.Error("documento e usuário vazios deveriam virar nulos")
	}

	if _, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{Nome: "José", Email: texto("jose")}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("e-mail inválido = %v, esperado erro de validação", err)
	}
	if _, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem nome = %v, esperado erro de validação", err)
	}
}

func TestGrupoRenomeiaComNomeDeBusca(t *testing.T) {
	grupo, err := domaincliente.NovoGrupo(" Grupo Ômega ")
	if err != nil || grupo.Nome != "Grupo Ômega" || grupo.NomeBusca != "grupo omega" || !grupo.Ativo {
		t.Fatalf("NovoGrupo = %+v, %v", grupo, err)
	}
	if err := grupo.Renomear("Ômega Construções", time.Now().UTC()); err != nil || grupo.NomeBusca != "omega construcoes" {
		t.Errorf("Renomear = %+v, %v", grupo, err)
	}
	if _, err := domaincliente.NovoGrupo(""); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("grupo sem nome = %v, esperado erro de validação", err)
	}
}

func TestObraValidaEnderecoENormalizaCEP(t *testing.T) {
	cnpj, err := domainempresa.NovoCNPJ("33.400.004/0001-16")
	if err != nil {
		t.Fatalf("CNPJ de teste inválido: %v", err)
	}
	clienteID := uuid.New()
	dados := domaincliente.DadosObra{
		Nome: " Residencial Mar Azul ",
		CNPJ: &cnpj,
		Endereco: domaincliente.Endereco{
			Logradouro: " Av. Beira Mar ",
			Numero:     texto(" 100 "),
			Bairro:     texto(" "),
			Cidade:     "Cabedelo",
			UF:         domainempresa.UFDe("PB"),
			CEP:        texto("58.310-000"),
		},
	}

	obra, err := domaincliente.NovaObra(clienteID, dados)
	if err != nil {
		t.Fatalf("NovaObra falhou: %v", err)
	}
	if obra.ClienteID != clienteID || obra.Nome != "Residencial Mar Azul" || obra.CNPJ == nil || !obra.Ativo {
		t.Errorf("obra criada = %+v", obra)
	}
	endereco := obra.Endereco
	if endereco.Logradouro != "Av. Beira Mar" || *endereco.Numero != "100" || endereco.Bairro != nil || *endereco.CEP != "58310000" {
		t.Errorf("endereço = %+v", endereco)
	}

	casos := []struct {
		nome    string
		alterar func(*domaincliente.DadosObra)
	}{
		{"sem nome", func(d *domaincliente.DadosObra) { d.Nome = "" }},
		{"sem logradouro", func(d *domaincliente.DadosObra) { d.Endereco.Logradouro = " " }},
		{"sem cidade", func(d *domaincliente.DadosObra) { d.Endereco.Cidade = "" }},
		{"uf inválida", func(d *domaincliente.DadosObra) { d.Endereco.UF = "" }},
		{"cep curto", func(d *domaincliente.DadosObra) { d.Endereco.CEP = texto("5831") }},
		{"cep com letra", func(d *domaincliente.DadosObra) { d.Endereco.CEP = texto("5831000A") }},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			invalidos := dados
			caso.alterar(&invalidos)
			if _, err := domaincliente.NovaObra(clienteID, invalidos); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := domaincliente.NovaObra(uuid.Nil, dados); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("obra sem cliente = %v, esperado erro de validação", err)
	}
}
