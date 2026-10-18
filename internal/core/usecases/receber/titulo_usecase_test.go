package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	"github.com/google/uuid"
)

func TestCriarTituloManual(t *testing.T) {
	c := novoCenario(t)

	titulo := c.novoTitulo(t)

	if titulo.Status != domainreceber.StatusCarteira || titulo.EmpresaID != c.mia || titulo.ClienteID != c.cliente {
		t.Fatalf("título = %+v", titulo)
	}
	if len(c.m.historicos) != 1 || c.m.historicos[0].Evento != domainreceber.EventoCriado || c.m.historicos[0].UsuarioID != c.financeiro {
		t.Fatalf("histórico = %+v", c.m.historicos)
	}
}

func TestCriarTituloComContaECompetencia(t *testing.T) {
	c := novoCenario(t)
	input := c.criarInput(c.financeiro, c.mia)
	input.ContaBancariaID = &c.contaMia
	input.Dados.Competencia = "2026-09"

	titulo, err := c.titulos.Criar(c.ctx, input)
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusEmBanco || titulo.Competencia.String() != "2026-09" {
		t.Fatalf("título = %+v", titulo)
	}
}

func TestCriarTituloConfereDependencias(t *testing.T) {
	c := novoCenario(t)

	casos := map[string]struct {
		alterar func(*portsin.CriarTituloInput)
		erro    error
		trecho  string
	}{
		"sem empresa":            {func(i *portsin.CriarTituloInput) { i.EmpresaID = uuid.Nil }, domain.ErrValidacao, "empresa do título é obrigatória"},
		"empresa fora do acesso": {func(i *portsin.CriarTituloInput) { i.EmpresaID = c.red }, domain.ErrPermissao, "sem acesso"},
		"empresa inativa":        {func(i *portsin.CriarTituloInput) { i.EmpresaID = c.inativa }, domain.ErrValidacao, "empresa inativa"},
		"sem cliente":            {func(i *portsin.CriarTituloInput) { i.ClienteID = uuid.Nil }, domain.ErrValidacao, "cliente do título é obrigatório"},
		"cliente ausente":        {func(i *portsin.CriarTituloInput) { i.ClienteID = uuid.New() }, domain.ErrValidacao, "cliente do título não encontrado"},
		"cliente inativo":        {func(i *portsin.CriarTituloInput) { i.ClienteID = c.clienteInativo }, domain.ErrValidacao, "cliente inativo"},
		"categoria de despesa":   {func(i *portsin.CriarTituloInput) { i.Dados.CategoriaFinanceiraID = &c.despesa }, domain.ErrValidacao, "de receita"},
		"categoria raiz":         {func(i *portsin.CriarTituloInput) { i.Dados.CategoriaFinanceiraID = &c.raiz }, domain.ErrValidacao, "subcategoria ativa"},
		"categoria ausente":      {func(i *portsin.CriarTituloInput) { ausente := uuid.New(); i.Dados.CategoriaFinanceiraID = &ausente }, domain.ErrValidacao, "não encontrada"},
		"competência inválida":   {func(i *portsin.CriarTituloInput) { i.Dados.Competencia = "09/2026" }, domain.ErrValidacao, "AAAA-MM"},
		"conta de outra empresa": {func(i *portsin.CriarTituloInput) { i.ContaBancariaID = &c.contaRed }, domain.ErrValidacao, "mesma empresa"},
		"conta inativa":          {func(i *portsin.CriarTituloInput) { i.ContaBancariaID = &c.contaInativa }, domain.ErrValidacao, "conta bancária inativa"},
		"conta ausente":          {func(i *portsin.CriarTituloInput) { ausente := uuid.New(); i.ContaBancariaID = &ausente }, domain.ErrValidacao, "conta bancária não encontrada"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			input := c.criarInput(c.financeiro, c.mia)
			caso.alterar(&input)

			_, err := c.titulos.Criar(c.ctx, input)
			esperarErro(t, err, caso.erro, caso.trecho)
		})
	}
	if len(c.m.titulos) != 0 {
		t.Fatalf("nenhum título deveria ter sido gravado: %d", len(c.m.titulos))
	}
}

func TestSoFinanceiroEAdministradorUsamOsTitulos(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	_, err := c.titulos.Criar(c.ctx, c.criarInput(c.comercial, c.mia))
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")
	_, err = c.titulos.Obter(c.ctx, titulo.ID, c.comercial)
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")
	_, err = c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")
	_, err = c.titulos.Obter(c.ctx, titulo.ID, uuid.New())
	esperarErro(t, err, domain.ErrNotFound, "usuário")

	if _, err := c.titulos.Obter(c.ctx, titulo.ID, c.administrador); err != nil {
		t.Fatalf("administrador deveria ver o título: %v", err)
	}
}

func TestIsolamentoDosTitulosPorEmpresa(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	_, err := c.titulos.Obter(c.ctx, titulo.ID, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = c.titulos.ListarHistorico(c.ctx, titulo.ID, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = c.titulos.Cancelar(c.ctx, portsin.CancelarTituloInput{UsuarioID: c.financeiroRed, TituloID: titulo.ID, Versao: 1, Motivo: "x"})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.financeiroRed, Empresa: &c.mia})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")

	daRed, err := c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.financeiroRed})
	if err != nil || len(daRed) != 0 {
		t.Fatalf("o financeiro da Red não vê títulos da MIA: %v, %v", daRed, err)
	}
	daMia, err := c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.financeiro})
	if err != nil || len(daMia) != 1 {
		t.Fatalf("o financeiro da MIA vê o título: %v, %v", daMia, err)
	}
	todos, err := c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.administrador})
	if err != nil || len(todos) != 1 || c.m.filtros[len(c.m.filtros)-1].EmpresaIDs != nil {
		t.Fatalf("o administrador lista sem restrição: %v, %v", todos, err)
	}
	if _, err := c.titulos.Obter(c.ctx, uuid.New(), c.financeiro); err == nil {
		t.Fatal("título ausente deveria ser 404")
	}
}

func TestListarValidaFiltrosEPreencheVencidos(t *testing.T) {
	c := novoCenario(t)

	_, err := c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.financeiro, Status: "vencido"})
	esperarErro(t, err, domain.ErrValidacao, "status do título")
	_, err = c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{UsuarioID: c.financeiro, Origem: "boleto"})
	esperarErro(t, err, domain.ErrValidacao, "origem deve ser")

	de := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	if _, err := c.titulos.Listar(c.ctx, portsin.ListarTitulosInput{
		UsuarioID: c.financeiro, Status: "em_banco", Origem: "pedido", Vencidos: true, VencimentoDe: &de,
	}); err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}
	filtro := c.m.filtros[len(c.m.filtros)-1]
	if filtro.Status != domainreceber.StatusEmBanco || filtro.Origem != domainreceber.OrigemPedido || filtro.VencidosEm == nil {
		t.Fatalf("filtro = %+v", filtro)
	}
	if !filtro.VencimentoDe.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) || filtro.Size != domain.TamanhoPaginaPadrao {
		t.Fatalf("data ou paginação = %v, %d", filtro.VencimentoDe, filtro.Size)
	}
}

func TestEditarTitulo(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)
	dados := c.dados()
	dados.ValorCentavos = 200000
	dados.Observacao = "reajuste"

	editado, err := c.titulos.Editar(c.ctx, portsin.EditarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, Dados: dados})
	if err != nil {
		t.Fatalf("Editar falhou: %v", err)
	}
	if editado.ValorCentavos != 200000 || editado.Versao != 2 {
		t.Fatalf("título = %+v", editado)
	}
	ultima := c.m.alteracoes[len(c.m.alteracoes)-1]
	if !ultima.ConferirCategoria || ultima.Historico.Evento != domainreceber.EventoEditado || ultima.Historico.Descricao != "alterados: valor, observação" {
		t.Fatalf("alteração = %+v, histórico = %+v", ultima, ultima.Historico)
	}

	igual, err := c.titulos.Editar(c.ctx, portsin.EditarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 2, Dados: dados})
	if err != nil || igual.Versao != 2 || len(c.m.alteracoes) != 1 {
		t.Fatalf("editar sem mudança não grava: %+v, %v", igual, err)
	}

	_, err = c.titulos.Editar(c.ctx, portsin.EditarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, Dados: dados})
	esperarErro(t, err, domain.ErrConflito, "alterado por outra pessoa")
	_, err = c.titulos.Editar(c.ctx, portsin.EditarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Dados: dados})
	esperarErro(t, err, domain.ErrValidacao, "versão do título é obrigatória")

	dados.CategoriaFinanceiraID = &c.despesa
	_, err = c.titulos.Editar(c.ctx, portsin.EditarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 2, Dados: dados})
	esperarErro(t, err, domain.ErrValidacao, "de receita")
}

func TestEditarConfereOEstadoAntesDoCorpo(t *testing.T) {
	c := novoCenario(t)
	titulo := c.emBanco(t)

	_, err := c.titulos.Editar(c.ctx, portsin.EditarTituloInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, Dados: portsin.DadosEdicaoInput{Competencia: "x"},
	})
	esperarErro(t, err, domain.ErrConflito, "em carteira")
}

func TestProrrogarTitulo(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)
	novo := titulo.Vencimento.AddDate(0, 0, 15)

	prorrogado, err := c.titulos.Prorrogar(c.ctx, portsin.ProrrogarTituloInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, Vencimento: novo, Motivo: "cliente pediu",
	})
	if err != nil {
		t.Fatalf("Prorrogar falhou: %v", err)
	}
	if !prorrogado.Vencimento.Equal(novo) || prorrogado.Competencia != titulo.Competencia {
		t.Fatalf("título = %+v", prorrogado)
	}
	historico := c.m.historicos[len(c.m.historicos)-1]
	if historico.Evento != domainreceber.EventoProrrogado || *historico.DeStatus != domainreceber.StatusCarteira {
		t.Fatalf("histórico = %+v", historico)
	}
}

func TestEnviarERetirarDoBanco(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	_, err := c.titulos.EnviarAoBanco(c.ctx, portsin.EnviarAoBancoInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, ContaBancariaID: c.contaRed})
	esperarErro(t, err, domain.ErrValidacao, "mesma empresa")

	emBanco, err := c.titulos.EnviarAoBanco(c.ctx, portsin.EnviarAoBancoInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, ContaBancariaID: c.contaMia})
	if err != nil {
		t.Fatalf("EnviarAoBanco falhou: %v", err)
	}
	if emBanco.Status != domainreceber.StatusEmBanco || !c.m.alteracoes[0].ConferirConta {
		t.Fatalf("título = %+v", emBanco)
	}

	retirado, err := c.titulos.RetirarDoBanco(c.ctx, portsin.TransicaoTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 2})
	if err != nil {
		t.Fatalf("RetirarDoBanco falhou: %v", err)
	}
	if retirado.Status != domainreceber.StatusCarteira || retirado.ContaBancariaID != nil {
		t.Fatalf("título = %+v", retirado)
	}
	_, err = c.titulos.RetirarDoBanco(c.ctx, portsin.TransicaoTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 3})
	esperarErro(t, err, domain.ErrConflito, "não é possível")
}

func TestRegistrarEnvioBoleto(t *testing.T) {
	c := novoCenario(t)
	carteira := c.novoTitulo(t)

	_, err := c.titulos.RegistrarEnvioBoleto(c.ctx, portsin.RegistrarEnvioBoletoInput{
		UsuarioID: c.financeiro, TituloID: carteira.ID, Versao: 1, Canal: "fax", EnviadoEm: time.Now(),
	})
	esperarErro(t, err, domain.ErrConflito, "em banco")

	titulo := c.emBanco(t)
	_, err = c.titulos.RegistrarEnvioBoleto(c.ctx, portsin.RegistrarEnvioBoletoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, Canal: "fax", EnviadoEm: time.Now(),
	})
	esperarErro(t, err, domain.ErrValidacao, "canal de envio")

	enviado, err := c.titulos.RegistrarEnvioBoleto(c.ctx, portsin.RegistrarEnvioBoletoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, Canal: "email", EnviadoEm: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("RegistrarEnvioBoleto falhou: %v", err)
	}
	if *enviado.BoletoCanal != domainreceber.CanalBoletoEmail || enviado.Status != domainreceber.StatusEmBanco {
		t.Fatalf("título = %+v", enviado)
	}
	historico := c.m.historicos[len(c.m.historicos)-1]
	if historico.Evento != domainreceber.EventoBoletoEnviado || historico.Descricao != "canal: email" {
		t.Fatalf("histórico = %+v", historico)
	}
}

func TestCancelarTitulo(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	cancelado, err := c.titulos.Cancelar(c.ctx, portsin.CancelarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, Motivo: "lançado errado"})
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if cancelado.Status != domainreceber.StatusCancelado {
		t.Fatalf("título = %+v", cancelado)
	}

	historicos, err := c.titulos.ListarHistorico(c.ctx, titulo.ID, c.financeiro)
	if err != nil || len(historicos) != 2 || historicos[1].Descricao != "lançado errado" {
		t.Fatalf("histórico = %+v, %v", historicos, err)
	}
}
