package financeiro_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

func entradaDeConta(usuarioID, empresaID uuid.UUID, conta, apelido string) portsin.CriarContaBancariaInput {
	return portsin.CriarContaBancariaInput{
		UsuarioID: usuarioID,
		EmpresaID: empresaID,
		Banco:     "748",
		Tipo:      "corrente",
		Agencia:   "0812",
		Conta:     conta,
		Apelido:   apelido,
	}
}

func (c *cenario) novaConta(t *testing.T, usuarioID, empresaID uuid.UUID, conta, apelido string) *domainfinanceiro.ContaBancaria {
	t.Helper()

	criada, err := c.contas.Criar(c.ctx, entradaDeConta(usuarioID, empresaID, conta, apelido))
	if err != nil {
		t.Fatalf("criação da conta %q falhou: %v", apelido, err)
	}

	return criada
}

func TestCriarContaBancariaValidaENormaliza(t *testing.T) {
	c := novoCenario(t)

	entrada := entradaDeConta(c.financeiroMIA, c.mia, "54.321", " Sicredi movimento ")
	entrada.Tipo = " Pagamento "
	entrada.ContaDigito = texto("0")
	conta, err := c.contas.Criar(c.ctx, entrada)
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	if conta.EmpresaID != c.mia || conta.Banco != domainfinanceiro.Sicredi || conta.Tipo != domainfinanceiro.TipoContaPagamento {
		t.Errorf("conta criada = %+v", conta)
	}
	if conta.Agencia.Formatado() != "0812" || conta.Conta.Formatado() != "54321-0" || conta.Apelido != "Sicredi movimento" {
		t.Errorf("dados = %q %q %q", conta.Agencia.Formatado(), conta.Conta.Formatado(), conta.Apelido)
	}
	if gravada := c.contasRepo.itens[conta.ID]; gravada == nil || gravada.Versao != 1 || !gravada.Ativa {
		t.Errorf("conta gravada = %+v", gravada)
	}
}

func TestCriarContaBancariaExigePerfilFinanceiro(t *testing.T) {
	c := novoCenario(t)

	_, err := c.contas.Criar(c.ctx, entradaDeConta(c.comum, c.mia, "1", "Comum"))
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(uuid.New(), c.mia, "1", "Fantasma"))
	esperarErro(t, err, domain.ErrNotFound, "usuário não encontrado")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(uuid.Nil, c.mia, "1", "Sem usuário"))
	esperarErro(t, err, domain.ErrValidacao, "usuário não informado")

	if len(c.contasRepo.itens) != 0 {
		t.Fatalf("nenhuma conta deveria ter sido gravada, há %d", len(c.contasRepo.itens))
	}

	c.novaConta(t, c.administrador, c.red, "2", "Do administrador")
}

func TestCriarContaBancariaConfereEmpresa(t *testing.T) {
	c := novoCenario(t)

	_, err := c.contas.Criar(c.ctx, entradaDeConta(c.financeiroMIA, c.red, "1", "Red"))
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(c.financeiroMIA, c.inativa, "1", "Inativa"))
	esperarErro(t, err, domain.ErrValidacao, "empresa inativa não recebe conta bancária")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(c.administrador, uuid.New(), "1", "Inexistente"))
	esperarErro(t, err, domain.ErrValidacao, "empresa da conta bancária não encontrada")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(c.administrador, uuid.Nil, "1", "Sem empresa"))
	esperarErro(t, err, domain.ErrValidacao, "empresa da conta bancária não informada")
}

func TestCriarContaBancariaRecusaDadosInvalidos(t *testing.T) {
	c := novoCenario(t)

	casos := map[string]func(*portsin.CriarContaBancariaInput){
		"banco":   func(e *portsin.CriarContaBancariaInput) { e.Banco = "104" },
		"tipo":    func(e *portsin.CriarContaBancariaInput) { e.Tipo = "caixa" },
		"agência": func(e *portsin.CriarContaBancariaInput) { e.Agencia = "12a" },
		"conta":   func(e *portsin.CriarContaBancariaInput) { e.Conta = "000" },
		"dígito":  func(e *portsin.CriarContaBancariaInput) { e.Conta = "1-2"; e.ContaDigito = texto("3") },
		"apelido": func(e *portsin.CriarContaBancariaInput) { e.Apelido = " " },
	}

	for nome, alterar := range casos {
		t.Run(nome, func(t *testing.T) {
			entrada := entradaDeConta(c.financeiroMIA, c.mia, "1", "Conta")
			alterar(&entrada)

			_, err := c.contas.Criar(c.ctx, entrada)
			esperarErro(t, err, domain.ErrValidacao, "")
		})
	}

	if len(c.contasRepo.itens) != 0 {
		t.Fatalf("nenhuma conta deveria ter sido gravada, há %d", len(c.contasRepo.itens))
	}
}

func TestCriarContaBancariaDuplicada(t *testing.T) {
	c := novoCenario(t)
	c.novaConta(t, c.financeiroMIA, c.mia, "54321", "Movimento")

	_, err := c.contas.Criar(c.ctx, entradaDeConta(c.financeiroRed, c.red, "054321", "Outra"))
	esperarErro(t, err, domain.ErrConflito, "já existe uma conta com este banco, agência e número")

	_, err = c.contas.Criar(c.ctx, entradaDeConta(c.financeiroMIA, c.mia, "999", "MOVIMENTO"))
	esperarErro(t, err, domain.ErrConflito, "já existe uma conta com este apelido na empresa")
}

func TestContasBancariasIsoladasPorEmpresa(t *testing.T) {
	c := novoCenario(t)
	daMIA := c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")
	daRed := c.novaConta(t, c.financeiroRed, c.red, "2", "Red")

	visiveis := map[uuid.UUID]*domainfinanceiro.ContaBancaria{c.financeiroMIA: daMIA, c.financeiroRed: daRed}
	for usuarioID, propria := range visiveis {
		contas, err := c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: usuarioID})
		if err != nil {
			t.Fatalf("Listar falhou: %v", err)
		}
		if len(contas) != 1 || contas[0].ID != propria.ID {
			t.Errorf("usuário enxergou %d contas", len(contas))
		}
	}

	_, err := c.contas.Obter(c.ctx, daRed.ID, c.financeiroMIA)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: daRed.ID, Apelido: "Minha", Versao: 1})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.contas.Desativar(c.ctx, portsin.MudarSituacaoContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: daRed.ID, Versao: 1})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.contas.Ativar(c.ctx, portsin.MudarSituacaoContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: daRed.ID, Versao: 1})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.financeiroMIA, Empresa: &c.red})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	if gravada := c.contasRepo.itens[daRed.ID]; gravada.Apelido != "Red" || !gravada.Ativa || gravada.Versao != 1 {
		t.Errorf("conta da Red foi alterada: %+v", gravada)
	}
}

func TestListarContasBancariasPorPerfilDeAcesso(t *testing.T) {
	c := novoCenario(t)
	c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")
	c.novaConta(t, c.financeiroRed, c.red, "2", "Red")

	todas, err := c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.administrador})
	if err != nil || len(todas) != 2 || c.contasRepo.ultimoFiltro.EmpresaIDs != nil {
		t.Fatalf("administrador listou %d contas com filtro %v: %v", len(todas), c.contasRepo.ultimoFiltro.EmpresaIDs, err)
	}

	nenhuma, err := c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.semEmpresa})
	if err != nil || len(nenhuma) != 0 || c.contasRepo.ultimoFiltro.EmpresaIDs == nil {
		t.Fatalf("usuário sem empresa listou %d contas com filtro %v: %v", len(nenhuma), c.contasRepo.ultimoFiltro.EmpresaIDs, err)
	}

	daRed, err := c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.administrador, Empresa: &c.red})
	if err != nil || len(daRed) != 1 || daRed[0].EmpresaID != c.red {
		t.Fatalf("filtro por empresa = %d contas: %v", len(daRed), err)
	}

	_, err = c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.comum})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
}

func TestListarContasBancariasValidaFiltros(t *testing.T) {
	c := novoCenario(t)
	ativa := true

	_, err := c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{
		UsuarioID: c.administrador,
		Banco:     "403",
		Tipo:      "POUPANCA",
		Ativa:     &ativa,
		Filtro:    domain.PaginacaoFiltro{Page: 0, Size: 500},
	})
	if err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}
	filtro := c.contasRepo.ultimoFiltro
	if *filtro.Banco != domainfinanceiro.Cora || *filtro.Tipo != domainfinanceiro.TipoContaPoupanca || !*filtro.Ativa {
		t.Errorf("filtro = %+v", filtro)
	}
	if filtro.Page != 1 || filtro.Size != domain.TamanhoPaginaMaximo {
		t.Errorf("paginação = %+v", filtro.PaginacaoFiltro)
	}

	_, err = c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.administrador, Banco: "104"})
	esperarErro(t, err, domain.ErrValidacao, "banco não suportado")

	_, err = c.contas.Listar(c.ctx, portsin.ListarContasBancariasInput{UsuarioID: c.administrador, Tipo: "caixa"})
	esperarErro(t, err, domain.ErrValidacao, "tipo da conta")
}

func TestObterContaBancaria(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")

	lida, err := c.contas.Obter(c.ctx, conta.ID, c.administrador)
	if err != nil || lida.ID != conta.ID {
		t.Fatalf("Obter = %v, %v", lida, err)
	}

	_, err = c.contas.Obter(c.ctx, uuid.New(), c.financeiroMIA)
	esperarErro(t, err, domain.ErrNotFound, "conta bancária não encontrada")

	_, err = c.contas.Obter(c.ctx, conta.ID, c.comum)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
}

func TestAlterarContaBancariaConfereVersao(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroMIA, c.mia, "1", "Antigo")

	_, err := c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Apelido: "Novo"})
	esperarErro(t, err, domain.ErrValidacao, "versão da conta bancária é obrigatória")

	_, err = c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Apelido: "Novo", Versao: 2})
	esperarErro(t, err, domain.ErrConflito, "alterada por outra pessoa")

	alterada, err := c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Apelido: " Novo ", Versao: 1})
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if alterada.Apelido != "Novo" || alterada.Versao != 2 || c.contasRepo.itens[conta.ID].Versao != 2 {
		t.Errorf("conta alterada = %q versão %d", alterada.Apelido, alterada.Versao)
	}

	_, err = c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Apelido: "", Versao: 2})
	esperarErro(t, err, domain.ErrValidacao, "apelido da conta bancária é obrigatório")

	_, err = c.contas.Alterar(c.ctx, portsin.AlterarContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: uuid.New(), Apelido: "X", Versao: 1})
	esperarErro(t, err, domain.ErrNotFound, "conta bancária não encontrada")
}

func TestAtivarEDesativarContaBancaria(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")
	entrada := portsin.MudarSituacaoContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Versao: 1}

	semMudanca, err := c.contas.Ativar(c.ctx, entrada)
	if err != nil || !semMudanca.Ativa || semMudanca.Versao != 1 || c.contasRepo.gravacoes != 1 {
		t.Fatalf("ativar conta ativa = %+v, %v (%d gravações)", semMudanca, err, c.contasRepo.gravacoes)
	}

	desativada, err := c.contas.Desativar(c.ctx, entrada)
	if err != nil || desativada.Ativa || desativada.Versao != 2 {
		t.Fatalf("Desativar = %+v, %v", desativada, err)
	}

	_, err = c.contas.Desativar(c.ctx, entrada)
	esperarErro(t, err, domain.ErrConflito, "alterada por outra pessoa")

	entrada.Versao = 2
	if repetida, err := c.contas.Desativar(c.ctx, entrada); err != nil || repetida.Versao != 2 {
		t.Fatalf("desativar conta inativa = %+v, %v", repetida, err)
	}

	reativada, err := c.contas.Ativar(c.ctx, entrada)
	if err != nil || !reativada.Ativa || reativada.Versao != 3 {
		t.Fatalf("Ativar = %+v, %v", reativada, err)
	}
}

func TestAtivarContaBancariaDeEmpresaDesativada(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")
	entrada := portsin.MudarSituacaoContaBancariaInput{UsuarioID: c.financeiroMIA, ContaID: conta.ID, Versao: 1}
	if _, err := c.contas.Desativar(c.ctx, entrada); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
	c.empresasRepo.itens[c.mia].Ativa = false

	entrada.Versao = 2
	_, err := c.contas.Ativar(c.ctx, entrada)
	esperarErro(t, err, domain.ErrValidacao, "empresa inativa não recebe conta bancária")

	if gravada := c.contasRepo.itens[conta.ID]; gravada.Ativa || gravada.Versao != 2 {
		t.Errorf("conta gravada = ativa %v versão %d", gravada.Ativa, gravada.Versao)
	}
}

func TestRemoverContaBancariaSoPeloAdministrador(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroMIA, c.mia, "1", "MIA")

	esperarErro(t, c.contas.Remover(c.ctx, conta.ID, c.financeiroMIA), domain.ErrPermissao, "perfil sem permissão para esta operação")

	if err := c.contas.Remover(c.ctx, conta.ID, c.administrador); err != nil {
		t.Fatalf("Remover falhou: %v", err)
	}
	if _, ok := c.contasRepo.itens[conta.ID]; ok {
		t.Error("a conta deveria ter sido removida")
	}

	esperarErro(t, c.contas.Remover(c.ctx, conta.ID, c.administrador), domain.ErrNotFound, "conta bancária não encontrada")
}

func TestRemoverContaBancariaRespeitaOAcessoDoAdministrador(t *testing.T) {
	c := novoCenario(t)
	conta := c.novaConta(t, c.financeiroRed, c.red, "1", "Red")
	adminRestrito := c.novoUsuario("Admin restrito", true, false, domainempresa.AcessoRestrito([]uuid.UUID{c.mia}))

	esperarErro(t, c.contas.Remover(c.ctx, conta.ID, adminRestrito), domain.ErrPermissao, "perfil sem acesso a esta empresa")
}

func TestBancosDevolveOCatalogo(t *testing.T) {
	c := novoCenario(t)

	bancos := c.contas.Bancos()
	if len(bancos) != 7 || bancos[0] != domainfinanceiro.BancoDoBrasil || bancos[6] != domainfinanceiro.Sicredi {
		t.Errorf("catálogo = %v", bancos)
	}
}
