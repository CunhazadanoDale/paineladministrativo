package cliente_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

func (c *cenario) novoLead(nome, email, telefone string) uuid.UUID {
	id := uuid.New()
	c.leadsRepo.itens[id] = &domainlead.Lead{ID: id, Nome: nome, Email: email, Telefone: telefone, Ativo: true}

	return id
}

func TestConverterLeadPreencheDadosDoLeadEVinculaUmaVez(t *testing.T) {
	c := novoCenario(t)
	leadID := c.novoLead("Construtora Águia", "contato@aguia.com.br", "83 98888-0000")
	dados := portsin.DadosClienteInput{Tipo: "construtora", UF: "PB", Cidade: "João Pessoa"}

	if _, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.operador, LeadID: leadID, Documento: "11222333000181", Dados: dados}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("converter sem perfil comercial = %v, esperado erro de permissão", err)
	}
	if _, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: uuid.New(), Documento: "11222333000181", Dados: dados}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("lead inexistente = %v, esperado não encontrado", err)
	}
	if _, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: leadID, Dados: dados}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("conversão sem documento = %v, esperado erro de validação", err)
	}

	cliente, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: leadID, Documento: "11.222.333/0001-81", Dados: dados})
	if err != nil {
		t.Fatalf("converter falhou: %v", err)
	}
	if cliente.LeadID == nil || *cliente.LeadID != leadID || cliente.Nome != "Construtora Águia" ||
		cliente.EmailNF == nil || *cliente.EmailNF != "contato@aguia.com.br" || cliente.Telefone == nil || *cliente.Telefone != "83 98888-0000" {
		t.Errorf("cliente convertido = %+v", cliente)
	}

	_, err = c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: leadID, Documento: "33400004000116", Dados: dados})
	if !errors.Is(err, domain.ErrConflito) || !mesmaMensagem(err, "lead já convertido em cliente") {
		t.Errorf("converter de novo = %v, esperado conflito de lead convertido", err)
	}

	outroLead := c.novoLead("Outra", "", "")
	explicito := dados
	explicito.Nome = "Nome informado"
	explicito.EmailNF = texto("nf@outra.com")
	convertido, err := c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: outroLead, Documento: "33400004000116", Dados: explicito})
	if err != nil || convertido.Nome != "Nome informado" || *convertido.EmailNF != "nf@outra.com" || convertido.Telefone != nil {
		t.Errorf("dados informados prevalecem sobre o lead = %+v (%v)", convertido, err)
	}

	terceiro := c.novoLead("Terceiro", "", "")
	_, err = c.clientes.ConverterLead(c.ctx, portsin.ConverterLeadInput{UsuarioID: c.comercial, LeadID: terceiro, Documento: "11222333000181", Dados: dados})
	if !errors.Is(err, domain.ErrConflito) || !mesmaMensagem(err, "já existe um cliente com este documento") {
		t.Errorf("converter com documento já usado = %v, esperado conflito de documento", err)
	}

	listados, err := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.operador, LeadID: &leadID})
	if err != nil || len(listados) != 1 || listados[0].ID != cliente.ID {
		t.Errorf("clientes do lead = %+v (%v)", listados, err)
	}
}

func linhaValida(numero int, documento, nome string) portsin.LinhaImportacao {
	return portsin.LinhaImportacao{Numero: numero, Documento: documento, Nome: nome, Tipo: "construtora", UF: "PB", Cidade: "João Pessoa"}
}

func TestImportacaoGravaValidasERelataRejeitadas(t *testing.T) {
	c := novoCenario(t)
	representante, err := c.representantes.Criar(c.ctx, portsin.CriarRepresentanteInput{
		UsuarioID: c.administrador,
		Dados:     portsin.DadosRepresentanteInput{Nome: "José Vendas", Documento: texto("529.982.247-25")},
	})
	if err != nil {
		t.Fatalf("criar representante falhou: %v", err)
	}
	c.novoCliente(t, "55666777000181", "Já Cadastrada", nil)
	inativo, _ := c.grupos.Criar(c.ctx, portsin.CriarGrupoInput{UsuarioID: c.comercial, Nome: "Grupo Parado"})
	if _, err := c.grupos.Desativar(c.ctx, inativo.ID, c.comercial); err != nil {
		t.Fatalf("desativar grupo falhou: %v", err)
	}

	comGrupo := linhaValida(2, "11.222.333/0001-81", "Construtora Águia")
	comGrupo.Grupo = "Grupo Águia"
	comGrupo.RepresentanteDocumento = "52998224725"
	comGrupo.ExigeNumeroPedidoNaNF = "SIM"
	comGrupo.EmailNF = "nf@aguia.com.br"
	mesmoGrupo := linhaValida(3, "33400004000116", "Construtora Ômega")
	mesmoGrupo.Grupo = "GRUPO AGUIA"
	repetidaNoArquivo := linhaValida(4, "11222333000181", "Repetida")
	jaCadastrada := linhaValida(5, "55.666.777/0001-81", "Já Cadastrada")
	semNome := linhaValida(6, "111.444.777-35", "")
	semNome.Tipo = "pf"
	flagRuim := linhaValida(7, "123.456.789-09", "Pedro")
	flagRuim.Tipo = "pf"
	flagRuim.ProibeAntecipacao = "talvez"
	representanteDesconhecido := linhaValida(8, "77888999000181", "Loja X")
	representanteDesconhecido.RepresentanteDocumento = "111.444.777-35"
	grupoInativo := linhaValida(9, "88999000000198", "Loja Y")
	grupoInativo.Grupo = "Grupo Parado"
	documentoRuim := linhaValida(10, "123", "Loja Z")

	linhas := []portsin.LinhaImportacao{comGrupo, mesmoGrupo, repetidaNoArquivo, jaCadastrada, semNome, flagRuim, representanteDesconhecido, grupoInativo, documentoRuim}

	if _, err := c.importacao.Importar(c.ctx, portsin.ImportarClientesInput{UsuarioID: c.comercial, Linhas: linhas}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("importação pelo comercial = %v, esperado erro de permissão", err)
	}

	simulacao, err := c.importacao.Importar(c.ctx, portsin.ImportarClientesInput{UsuarioID: c.administrador, Linhas: linhas, Simular: true})
	if err != nil {
		t.Fatalf("simulação falhou: %v", err)
	}
	if !simulacao.Simulacao || simulacao.Total != 9 || simulacao.Importadas != 2 || len(simulacao.Rejeitadas) != 7 {
		t.Fatalf("simulação = %+v", simulacao)
	}
	if todos, _ := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.administrador}); len(todos) != 1 {
		t.Errorf("simulação gravou %d clientes, esperado só o que já existia", len(todos))
	}
	if grupos, _ := c.grupos.Listar(c.ctx, portsin.ListarGruposInput{Busca: "aguia"}); len(grupos) != 0 {
		t.Error("simulação não pode criar grupo")
	}

	relatorio, err := c.importacao.Importar(c.ctx, portsin.ImportarClientesInput{UsuarioID: c.administrador, Linhas: linhas})
	if err != nil {
		t.Fatalf("importação falhou: %v", err)
	}
	if relatorio.Simulacao || relatorio.Importadas != 2 || len(relatorio.Rejeitadas) != 7 {
		t.Fatalf("relatório = %+v", relatorio)
	}

	esperado := map[int]string{
		4:  "documento repetido no arquivo (linha 2)",
		5:  "documento já cadastrado",
		6:  "nome do cliente é obrigatório",
		7:  "proibe_antecipacao deve ser sim ou nao",
		8:  "representante não encontrado pelo documento",
		9:  "grupo de clientes inativo não recebe cliente",
		10: "documento deve ser um CPF ou CNPJ válido",
	}
	for _, rejeitada := range relatorio.Rejeitadas {
		if esperado[rejeitada.Linha] != rejeitada.Motivo {
			t.Errorf("linha %d rejeitada por %q, esperado %q", rejeitada.Linha, rejeitada.Motivo, esperado[rejeitada.Linha])
		}
	}

	importados, _ := c.clientes.Listar(c.ctx, portsin.ListarClientesInput{UsuarioID: c.administrador, Busca: "construtora"})
	if len(importados) != 2 || importados[0].GrupoID == nil || importados[1].GrupoID == nil || *importados[0].GrupoID != *importados[1].GrupoID {
		t.Fatalf("clientes importados = %+v, esperado os dois no mesmo grupo criado", importados)
	}
	for _, cliente := range importados {
		if cliente.Documento.Valor() == "11222333000181" && (cliente.RepresentanteID == nil || *cliente.RepresentanteID != representante.ID || !cliente.ExigeNumeroPedidoNaNF) {
			t.Errorf("cliente com representante e flag = %+v", cliente)
		}
	}
	if grupos, _ := c.grupos.Listar(c.ctx, portsin.ListarGruposInput{Busca: "aguia"}); len(grupos) != 1 {
		t.Errorf("grupos criados pela importação = %d, esperado 1", len(grupos))
	}

	novamente, err := c.importacao.Importar(c.ctx, portsin.ImportarClientesInput{UsuarioID: c.administrador, Linhas: linhas[:2]})
	if err != nil || novamente.Importadas != 0 || len(novamente.Rejeitadas) != 2 {
		t.Errorf("reimportar as mesmas linhas = %+v (%v), esperado nada sobrescrito", novamente, err)
	}
}

func TestImportacaoLimitaQuantidadeDeLinhas(t *testing.T) {
	c := novoCenario(t)
	linhas := make([]portsin.LinhaImportacao, 5001)

	if _, err := c.importacao.Importar(c.ctx, portsin.ImportarClientesInput{UsuarioID: c.administrador, Linhas: linhas}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("importação acima do limite = %v, esperado erro de validação", err)
	}
}
