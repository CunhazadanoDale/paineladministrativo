package estoque_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func (c *cenario) novaEmpresa(ativa bool) uuid.UUID {
	id := uuid.New()
	c.empresasRepo.itens[id] = &domainempresa.Empresa{ID: id, NomeFantasia: "Empresa " + id.String()[:4], Ativa: ativa}

	return id
}

func (c *cenario) novoUsuarioComPerfil(nome string, administrador, financeiro bool, empresaIDs ...uuid.UUID) uuid.UUID {
	cargoID := uuid.New()
	c.cargosRepo.itens[cargoID] = &domainusuarios.Cargo{
		ID:            cargoID,
		Nome:          "Cargo " + nome,
		Ativo:         true,
		Administrador: administrador,
		Financeiro:    financeiro,
	}

	usuarioID := uuid.New()
	c.usuariosRepo.itens[usuarioID] = &domainusuarios.Usuario{ID: usuarioID, Nome: nome, CargoID: cargoID, Ativo: true}
	c.empresasRepo.acessos[usuarioID] = domainempresa.AcessoRestrito(empresaIDs)

	return usuarioID
}

func (c *cenario) novoLocal(t *testing.T, empresaID uuid.UUID, nome string) *domainestoque.Local {
	t.Helper()

	id, err := c.local.Criar(c.ctx, portsin.CriarLocalInput{UsuarioID: c.usuarioID, EmpresaID: empresaID, Nome: nome})
	if err != nil {
		t.Fatalf("criação do local falhou: %v", err)
	}

	local, err := c.local.Obter(c.ctx, id, c.usuarioID)
	if err != nil {
		t.Fatalf("busca do local falhou: %v", err)
	}

	return local
}

func (c *cenario) entradaNoLocal(t *testing.T, produto *domainestoque.Produto, local *domainestoque.Local, quantidade int, custoCentavos int64) {
	t.Helper()

	estoque, err := c.estoqueRepo.Carregar(c.ctx, produto.ID, []uuid.UUID{local.ID})
	if err != nil {
		t.Fatalf("carregar o estoque falhou: %v", err)
	}

	custo := domainestoque.CustoDe(custoCentavos)
	operacao, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:       domainestoque.TipoMovimentoEntrada,
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		Custo:      &custo,
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("entrada de domínio falhou: %v", err)
	}
	if err := c.estoqueRepo.Gravar(c.ctx, operacao); err != nil {
		t.Fatalf("gravar a entrada falhou: %v", err)
	}
}

func (c *cenario) produtoDeTeste(t *testing.T) *domainestoque.Produto {
	t.Helper()

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Cimento", raiz)

	return c.novoProduto(t, sub, "Cimento CP II")
}

func TestCriarLocalValidaPerfilEmpresaENome(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	inativa := c.novaEmpresa(false)
	administradorDaRed := c.novoUsuarioComPerfil("Diretoria Red", true, false, red)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		empresaID uuid.UUID
		nomeLocal string
		sentinela error
	}{
		{"sem perfil de administrador", c.visitanteID, mia, "Galpão", domain.ErrPermissao},
		{"sem empresa", c.usuarioID, uuid.Nil, "Galpão", domain.ErrValidacao},
		{"empresa inexistente", c.usuarioID, uuid.New(), "Galpão", domain.ErrValidacao},
		{"empresa inativa", c.usuarioID, inativa, "Galpão", domain.ErrValidacao},
		{"empresa fora do acesso", administradorDaRed, mia, "Galpão", domain.ErrPermissao},
		{"empresa inexistente fora do acesso", administradorDaRed, uuid.New(), "Galpão", domain.ErrPermissao},
		{"sem nome", c.usuarioID, mia, "  ", domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.local.Criar(c.ctx, portsin.CriarLocalInput{UsuarioID: caso.usuario, EmpresaID: caso.empresaID, Nome: caso.nomeLocal})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	c.novoLocal(t, mia, "Galpão")
	_, err := c.local.Criar(c.ctx, portsin.CriarLocalInput{UsuarioID: c.usuarioID, EmpresaID: mia, Nome: "galpão"})
	if !errors.Is(err, domain.ErrConflito) || err.Error() != domain.ErroConflito("já existe um local com este nome na empresa").Error() {
		t.Errorf("nome repetido = %v, esperado conflito com mensagem do local", err)
	}
	if _, err := c.local.Criar(c.ctx, portsin.CriarLocalInput{UsuarioID: administradorDaRed, EmpresaID: red, Nome: "Galpão"}); err != nil {
		t.Errorf("mesmo nome em outra empresa = %v, esperado sucesso", err)
	}
}

func TestAlterarLocalRespeitaSaldoEVinculos(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	loja := c.novoLocal(t, mia, "Loja")
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 3, 100)

	if _, err := c.local.Renomear(c.ctx, portsin.RenomearLocalInput{UsuarioID: c.usuarioID, LocalID: loja.ID, Nome: "GALPÃO"}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("renomear para nome existente = %v, esperado conflito", err)
	}
	renomeado, err := c.local.Renomear(c.ctx, portsin.RenomearLocalInput{UsuarioID: c.usuarioID, LocalID: loja.ID, Nome: "Loja Centro"})
	if err != nil || renomeado.Nome != "Loja Centro" || renomeado.EmpresaID != mia {
		t.Errorf("renomear = %+v (%v)", renomeado, err)
	}
	if _, err := c.local.Renomear(c.ctx, portsin.RenomearLocalInput{UsuarioID: c.visitanteID, LocalID: loja.ID, Nome: "X"}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("renomear sem perfil = %v, esperado erro de permissão", err)
	}

	if _, err := c.local.Desativar(c.ctx, galpao.ID, c.usuarioID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("desativar local com saldo = %v, esperado erro de validação", err)
	}
	if salvo, _ := c.locaisRepo.Obter(c.ctx, galpao.ID); !salvo.Ativo {
		t.Error("local com saldo não pode ficar inativo")
	}

	desativado, err := c.local.Desativar(c.ctx, loja.ID, c.usuarioID)
	if err != nil || desativado.Ativo {
		t.Fatalf("desativar local vazio = %+v (%v)", desativado, err)
	}
	ativo := false
	inativos, err := c.local.Listar(c.ctx, portsin.ListarLocaisInput{UsuarioID: c.usuarioID, Ativo: &ativo})
	if err != nil || len(inativos) != 1 || inativos[0].ID != loja.ID {
		t.Errorf("locais inativos = %+v (%v), esperado só a loja", inativos, err)
	}
	if reativado, err := c.local.Ativar(c.ctx, loja.ID, c.usuarioID); err != nil || !reativado.Ativo {
		t.Errorf("ativar = %+v (%v)", reativado, err)
	}

	if err := c.local.Remover(c.ctx, galpao.ID, c.usuarioID); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remover local com estoque = %v, esperado erro de validação", err)
	}
	if err := c.local.Remover(c.ctx, loja.ID, c.usuarioID); err != nil {
		t.Errorf("remover local sem vínculo = %v", err)
	}
	if _, err := c.local.Obter(c.ctx, loja.ID, c.usuarioID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("local removido = %v, esperado não encontrado", err)
	}
}

func TestTransferirEntreLocaisDaMesmaEmpresa(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	loja := c.novoLocal(t, mia, "Loja")
	daRed := c.novoLocal(t, red, "Depósito")
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 10, 150)
	c.entradaNoLocal(t, produto, daRed, 5, 150)
	administradorDaRed := c.novoUsuarioComPerfil("Diretoria Red", true, false, red)

	transferencia := func(usuario, produtoID, origem, destino uuid.UUID, quantidade int) error {
		_, err := c.movimento.Transferir(c.ctx, portsin.TransferirEstoqueInput{
			UsuarioID:      usuario,
			ProdutoID:      produtoID,
			LocalOrigemID:  origem,
			LocalDestinoID: destino,
			Quantidade:     quantidade,
		})
		return err
	}

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		produtoID uuid.UUID
		origem    uuid.UUID
		destino   uuid.UUID
		sentinela error
	}{
		{"sem perfil de administrador", c.visitanteID, produto.ID, galpao.ID, loja.ID, domain.ErrPermissao},
		{"origem inexistente", c.usuarioID, produto.ID, uuid.New(), loja.ID, domain.ErrValidacao},
		{"destino não informado", c.usuarioID, produto.ID, galpao.ID, uuid.Nil, domain.ErrValidacao},
		{"produto inexistente", c.usuarioID, uuid.New(), galpao.ID, loja.ID, domain.ErrNotFound},
		{"entre empresas", c.usuarioID, produto.ID, daRed.ID, galpao.ID, domain.ErrValidacao},
		{"locais fora do acesso", administradorDaRed, produto.ID, galpao.ID, loja.ID, domain.ErrPermissao},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := transferencia(caso.usuario, caso.produtoID, caso.origem, caso.destino, 2); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	movimento, err := c.movimento.Transferir(c.ctx, portsin.TransferirEstoqueInput{
		UsuarioID:      c.usuarioID,
		ProdutoID:      produto.ID,
		LocalOrigemID:  galpao.ID,
		LocalDestinoID: loja.ID,
		Quantidade:     4,
		Observacao:     "abastecer a loja",
	})
	if err != nil {
		t.Fatalf("transferência falhou: %v", err)
	}
	if movimento.Tipo != domainestoque.TipoMovimentoTransferencia || *movimento.LocalDestinoID != loja.ID || movimento.Observacao != "abastecer a loja" {
		t.Errorf("movimento devolvido = %+v", movimento)
	}

	saldo, err := c.movimento.Saldo(c.ctx, produto.ID, c.usuarioID)
	if err != nil {
		t.Fatalf("saldo falhou: %v", err)
	}
	quantidades := map[uuid.UUID]int{}
	for _, local := range saldo.Locais {
		quantidades[local.LocalID] = local.Quantidade
	}
	if saldo.Produto.Saldo.Quantidade() != 15 || quantidades[galpao.ID] != 6 || quantidades[loja.ID] != 4 || quantidades[daRed.ID] != 5 {
		t.Errorf("saldos depois da transferência: total %d, por local %v", saldo.Produto.Saldo.Quantidade(), quantidades)
	}
}

func TestSaldoDoProdutoMostraNaoAlocado(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	produto := c.produtoDeTeste(t)
	c.movimentar(t, produto, "entrada", 10)
	c.entradaNoLocal(t, produto, galpao, 4, 100)

	saldo, err := c.movimento.Saldo(c.ctx, produto.ID, c.usuarioID)
	if err != nil {
		t.Fatalf("saldo falhou: %v", err)
	}
	if saldo.Produto.Saldo.Quantidade() != 14 || saldo.NaoAlocado != 10 || len(saldo.Locais) != 1 || saldo.Locais[0].LocalNome != "Galpão" {
		t.Errorf("saldo = total %d, não alocado %d, locais %+v", saldo.Produto.Saldo.Quantidade(), saldo.NaoAlocado, saldo.Locais)
	}
	if _, err := c.movimento.Saldo(c.ctx, uuid.New(), c.usuarioID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("saldo de produto inexistente = %v, esperado não encontrado", err)
	}
}

func TestLotesExigemAdministradorOuFinanceiro(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 4, 100)
	c.entradaNoLocal(t, produto, galpao, 2, 300)
	financeiro := c.novoUsuarioComPerfil("Caio", false, true, mia)
	operador := c.novoUsuarioComPerfil("Lia", false, false, mia)

	if _, err := c.movimento.ListarLotes(c.ctx, portsin.ListarLotesInput{UsuarioID: operador, ProdutoID: produto.ID}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("lotes para operador = %v, esperado erro de permissão", err)
	}
	if _, err := c.movimento.ListarLotes(c.ctx, portsin.ListarLotesInput{UsuarioID: financeiro, ProdutoID: uuid.New()}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("lotes de produto inexistente = %v, esperado não encontrado", err)
	}

	lotes, err := c.movimento.ListarLotes(c.ctx, portsin.ListarLotesInput{UsuarioID: financeiro, ProdutoID: produto.ID, LocalID: &galpao.ID})
	if err != nil || len(lotes) != 2 || lotes[0].Custo.Centavos() != 100 || lotes[1].Custo.Centavos() != 300 {
		t.Errorf("lotes para o financeiro = %+v (%v), esperado os dois em ordem de entrada", lotes, err)
	}
}

func TestEstoqueDeEmpresasDiferentesFicaIsolado(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	galpaoMia := c.novoLocal(t, mia, "Galpão MIA")
	depositoRed := c.novoLocal(t, red, "Depósito Red")
	produto := c.produtoDeTeste(t)
	c.movimentar(t, produto, "entrada", 1)
	c.entradaNoLocal(t, produto, galpaoMia, 3, 100)
	c.entradaNoLocal(t, produto, depositoRed, 7, 200)

	ana := c.novoUsuarioComPerfil("Ana", false, true, mia)
	bia := c.novoUsuarioComPerfil("Bia", false, true, red)

	casos := []struct {
		nome    string
		usuario uuid.UUID
		empresa uuid.UUID
		proprio *domainestoque.Local
		alheio  *domainestoque.Local
	}{
		{"usuária da MIA", ana, mia, galpaoMia, depositoRed},
		{"usuária da Red", bia, red, depositoRed, galpaoMia},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			locais, err := c.local.Listar(c.ctx, portsin.ListarLocaisInput{UsuarioID: caso.usuario})
			if err != nil || len(locais) != 1 || locais[0].ID != caso.proprio.ID {
				t.Errorf("locais listados = %+v (%v), esperado só o próprio", locais, err)
			}
			if _, err := c.local.Obter(c.ctx, caso.alheio.ID, caso.usuario); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("obter local alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.local.Listar(c.ctx, portsin.ListarLocaisInput{UsuarioID: caso.usuario, Empresa: &caso.alheio.EmpresaID}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("filtrar pela empresa alheia = %v, esperado erro de permissão", err)
			}
			if _, err := c.local.ListarSaldos(c.ctx, portsin.ListarSaldosDoLocalInput{UsuarioID: caso.usuario, LocalID: caso.alheio.ID}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("saldos do local alheio = %v, esperado erro de permissão", err)
			}
			if _, err := c.local.Renomear(c.ctx, portsin.RenomearLocalInput{UsuarioID: caso.usuario, LocalID: caso.alheio.ID, Nome: "Tomado"}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("renomear local alheio = %v, esperado erro de permissão", err)
			}

			saldo, err := c.movimento.Saldo(c.ctx, produto.ID, caso.usuario)
			if err != nil || len(saldo.Locais) != 1 || saldo.Locais[0].LocalID != caso.proprio.ID {
				t.Errorf("saldo por local = %+v (%v), esperado só o próprio", saldo, err)
			}

			movimentos, err := c.movimento.Listar(c.ctx, portsin.ListarMovimentosInput{UsuarioID: caso.usuario, ProdutoID: &produto.ID})
			if err != nil || len(movimentos) != 2 {
				t.Fatalf("movimentos = %d (%v), esperado o próprio e o antigo sem local", len(movimentos), err)
			}
			for _, movimento := range movimentos {
				if movimento.LocalID != nil && *movimento.LocalID != caso.proprio.ID {
					t.Errorf("movimento do local alheio %s visível", *movimento.LocalID)
				}
			}
			if _, err := c.movimento.Listar(c.ctx, portsin.ListarMovimentosInput{UsuarioID: caso.usuario, LocalID: &caso.alheio.ID}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("movimentos filtrados pelo local alheio = %v, esperado erro de permissão", err)
			}

			lotes, err := c.movimento.ListarLotes(c.ctx, portsin.ListarLotesInput{UsuarioID: caso.usuario, ProdutoID: produto.ID})
			if err != nil || len(lotes) != 1 || lotes[0].LocalID != caso.proprio.ID {
				t.Errorf("lotes = %+v (%v), esperado só o do local próprio", lotes, err)
			}

			resumo, err := c.resumo.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: caso.usuario})
			if err != nil || len(resumo.UltimosMovimentos) != 2 {
				t.Errorf("últimos movimentos = %d (%v), esperado o próprio e o antigo", len(resumo.UltimosMovimentos), err)
			}
			if _, err := c.resumo.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: caso.usuario, Empresa: &caso.alheio.EmpresaID}); !errors.Is(err, domain.ErrPermissao) {
				t.Errorf("resumo da empresa alheia = %v, esperado erro de permissão", err)
			}
		})
	}

	semAcesso := c.novoUsuarioComPerfil("Sem acesso", false, true)
	if locais, err := c.local.Listar(c.ctx, portsin.ListarLocaisInput{UsuarioID: semAcesso}); err != nil || len(locais) != 0 {
		t.Errorf("locais para acesso vazio = %d (%v), esperado nenhum", len(locais), err)
	}
	if lotes, err := c.movimento.ListarLotes(c.ctx, portsin.ListarLotesInput{UsuarioID: semAcesso, ProdutoID: produto.ID}); err != nil || len(lotes) != 0 {
		t.Errorf("lotes para acesso vazio = %d (%v), esperado nenhum", len(lotes), err)
	}
}
