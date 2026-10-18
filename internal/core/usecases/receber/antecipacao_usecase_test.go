package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	"github.com/google/uuid"
)

func (c *cenario) novoFomento(ativo bool) uuid.UUID {
	fornecedor := &domainfornecedor.Fornecedor{ID: uuid.New(), Nome: "Fomento Alfa", Ativo: ativo}
	c.m.fornecedores[fornecedor.ID] = fornecedor

	return fornecedor.ID
}

func (c *cenario) borderoInput(fomento uuid.UUID, titulos ...*domainreceber.Titulo) portsin.CriarAntecipacaoInput {
	ids := make([]uuid.UUID, 0, len(titulos))
	var bruto int64
	for _, titulo := range titulos {
		ids = append(ids, titulo.ID)
		bruto += titulo.SaldoCentavos()
	}

	return portsin.CriarAntecipacaoInput{
		UsuarioID:            c.financeiro,
		EmpresaID:            c.mia,
		FornecedorID:         fomento,
		OperadoEm:            domainreceber.Hoje(time.Now().UTC()),
		TaxaBP:               300,
		ValorLiquidoCentavos: bruto - 5000,
		ContaBancariaID:      c.contaMia,
		TituloIDs:            ids,
	}
}

func TestCriarBorderoAntecipaOsTitulos(t *testing.T) {
	c := novoCenario(t)
	fomento := c.novoFomento(true)
	primeiro, segundo := c.novoTitulo(t), c.emBanco(t)

	antecipacao, err := c.borderos.Criar(c.ctx, c.borderoInput(fomento, primeiro, segundo))
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}
	if antecipacao.ValorBrutoCentavos != 300000 || antecipacao.EncargosCentavos() != 5000 || len(antecipacao.Titulos) != 2 {
		t.Fatalf("antecipação = %+v", antecipacao)
	}
	if c.m.titulos[primeiro.ID].Status != domainreceber.StatusAntecipado || c.m.titulos[segundo.ID].Status != domainreceber.StatusAntecipado {
		t.Fatal("os títulos ficam antecipados")
	}
	if historico := c.m.historicos[len(c.m.historicos)-1]; historico.Evento != domainreceber.EventoAntecipado {
		t.Fatalf("histórico = %+v", historico)
	}

	obtida, err := c.borderos.Obter(c.ctx, antecipacao.ID, c.financeiro)
	if err != nil || obtida.ID != antecipacao.ID {
		t.Fatalf("Obter = %+v, %v", obtida, err)
	}
	_, err = c.borderos.Obter(c.ctx, antecipacao.ID, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	daRed, err := c.borderos.Listar(c.ctx, portsin.ListarAntecipacoesInput{UsuarioID: c.financeiroRed})
	if err != nil || len(daRed) != 0 {
		t.Fatalf("o financeiro da Red não vê o borderô da MIA: %v, %v", daRed, err)
	}
	_, err = c.borderos.Listar(c.ctx, portsin.ListarAntecipacoesInput{UsuarioID: c.financeiro, Status: "vencida"})
	esperarErro(t, err, domain.ErrValidacao, "ativa ou estornada")
}

func TestCriarBorderoConfereDependencias(t *testing.T) {
	c := novoCenario(t)
	fomento := c.novoFomento(true)
	titulo := c.novoTitulo(t)

	input := c.borderoInput(c.novoFomento(false), titulo)
	_, err := c.borderos.Criar(c.ctx, input)
	esperarErro(t, err, domain.ErrValidacao, "fomento inativo")

	input = c.borderoInput(uuid.New(), titulo)
	_, err = c.borderos.Criar(c.ctx, input)
	esperarErro(t, err, domain.ErrValidacao, "fomento da antecipação não encontrado")

	input = c.borderoInput(fomento, titulo)
	input.ContaBancariaID = c.contaRed
	_, err = c.borderos.Criar(c.ctx, input)
	esperarErro(t, err, domain.ErrValidacao, "mesma empresa")

	input = c.borderoInput(fomento, titulo)
	input.EmpresaID = c.red
	_, err = c.borderos.Criar(c.ctx, input)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")

	c.m.clientes[c.cliente].ProibeAntecipacao = true
	_, err = c.borderos.Criar(c.ctx, c.borderoInput(fomento, titulo))
	esperarErro(t, err, domain.ErrValidacao, "cliente não permite antecipação: Construtora Alfa")

	c.m.clientes[c.cliente].ProibeAntecipacao = false
	input = c.borderoInput(fomento, titulo)
	input.TituloIDs = append(input.TituloIDs, uuid.New())
	_, err = c.borderos.Criar(c.ctx, input)
	esperarErro(t, err, domain.ErrNotFound, "título não encontrado")

	_, err = c.borderos.Criar(c.ctx, portsin.CriarAntecipacaoInput{UsuarioID: c.comercial, EmpresaID: c.mia})
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")

	if len(c.m.antecipacoes) != 0 || c.m.titulos[titulo.ID].Status != domainreceber.StatusCarteira {
		t.Fatal("nada é gravado quando a antecipação é recusada")
	}
}

func TestEstornarBordero(t *testing.T) {
	c := novoCenario(t)
	titulo := c.emBanco(t)
	antecipacao, err := c.borderos.Criar(c.ctx, c.borderoInput(c.novoFomento(true), titulo))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.borderos.Estornar(c.ctx, portsin.EstornarAntecipacaoInput{UsuarioID: c.financeiro, AntecipacaoID: antecipacao.ID, Versao: 2, Motivo: "x"})
	esperarErro(t, err, domain.ErrConflito, "alterada por outra pessoa")

	estornada, err := c.borderos.Estornar(c.ctx, portsin.EstornarAntecipacaoInput{
		UsuarioID: c.financeiro, AntecipacaoID: antecipacao.ID, Versao: 1, Motivo: "lançado no fomento errado",
	})
	if err != nil {
		t.Fatalf("Estornar falhou: %v", err)
	}
	if estornada.Status != domainreceber.StatusAntecipacaoEstornada || c.m.titulos[titulo.ID].Status != domainreceber.StatusEmBanco {
		t.Fatalf("antecipação = %+v, título = %+v", estornada, c.m.titulos[titulo.ID])
	}
	if len(c.m.estornos) != 1 || c.m.estornos[0].Titulos[0].Historico.Evento != domainreceber.EventoAntecipacaoEstornada {
		t.Fatalf("estorno = %+v", c.m.estornos)
	}
}

func (c *cenario) antecipado(t *testing.T) (*domainreceber.Titulo, *domainreceber.Antecipacao, uuid.UUID) {
	t.Helper()

	fomento := c.novoFomento(true)
	titulo := c.novoTitulo(t)
	antecipacao, err := c.borderos.Criar(c.ctx, c.borderoInput(fomento, titulo))
	if err != nil {
		t.Fatal(err)
	}

	return c.m.titulos[titulo.ID], antecipacao, fomento
}

func TestRecomprarGeraContaAPagarAoFomento(t *testing.T) {
	c := novoCenario(t)
	titulo, antecipacao, fomento := c.antecipado(t)
	hoje := domainreceber.Hoje(time.Now().UTC())
	input := portsin.RecomprarInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, RecompradoEm: hoje, EncargosCentavos: 2500,
		Vencimento: hoje.AddDate(0, 0, 3), CategoriaFinanceiraID: c.despesa, Motivo: "cliente não pagou",
	}

	receita := input
	receita.CategoriaFinanceiraID = c.receita
	_, _, err := c.borderos.Recomprar(c.ctx, receita)
	esperarErro(t, err, domain.ErrValidacao, "de despesa")

	recomprado, recompra, err := c.borderos.Recomprar(c.ctx, input)
	if err != nil {
		t.Fatalf("Recomprar falhou: %v", err)
	}
	if recomprado.Status != domainreceber.StatusCarteira || recompra.ValorCentavos != 152500 || recompra.AntecipacaoID != antecipacao.ID {
		t.Fatalf("título = %+v, recompra = %+v", recomprado, recompra)
	}

	gravacao := c.m.recompras[0]
	conta := gravacao.ContaAPagar.Solicitacao
	if conta.Origem != domainsolicitacao.OrigemRecompra || *conta.OrigemID != recompra.ID || *conta.FornecedorID != fomento ||
		conta.Valor.Centavos() != 152500 || conta.Status != domainsolicitacao.StatusPendenteAprovacao || conta.SolicitanteID != c.financeiro ||
		conta.Competencia.String() != domainsolicitacao.CompetenciaDe(hoje).String() || conta.EmpresaID != c.mia {
		t.Fatalf("conta a pagar = %+v", conta)
	}
	if recompra.SolicitacaoID != conta.ID {
		t.Fatalf("recompra sem a conta: %+v", recompra)
	}

	_, _, err = c.borderos.Recomprar(c.ctx, portsin.RecomprarInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: recomprado.Versao})
	esperarErro(t, err, domain.ErrConflito, "só o título antecipado")
}

func TestReceberAntecipadoEmContaPropriaGeraORepasse(t *testing.T) {
	c := novoCenario(t)
	titulo, antecipacao, fomento := c.antecipado(t)
	hoje := domainreceber.Hoje(time.Now().UTC())

	semRepasse := c.receberInput(titulo, 150000)
	_, _, err := c.baixas.Receber(c.ctx, semRepasse)
	esperarErro(t, err, domain.ErrValidacao, "exige o repasse")

	comReceita := c.receberInput(titulo, 150000)
	comReceita.Repasse = &portsin.RepasseInput{Vencimento: hoje.AddDate(0, 0, 2), CategoriaFinanceiraID: c.receita}
	_, _, err = c.baixas.Receber(c.ctx, comReceita)
	esperarErro(t, err, domain.ErrValidacao, "de despesa")

	input := c.receberInput(titulo, 150000)
	input.JurosCentavos = 700
	input.Repasse = &portsin.RepasseInput{Vencimento: hoje.AddDate(0, 0, 2), CategoriaFinanceiraID: c.despesa}
	recebido, recebimento, err := c.baixas.Receber(c.ctx, input)
	if err != nil {
		t.Fatalf("Receber com repasse falhou: %v", err)
	}
	if recebido.Status != domainreceber.StatusRecebido || recebimento.Repasse == nil || recebimento.Repasse.AntecipacaoID != antecipacao.ID {
		t.Fatalf("título = %+v, recebimento = %+v", recebido, recebimento)
	}
	conta := c.m.baixas[0].ContaDoRepasse.Solicitacao
	if conta.Origem != domainsolicitacao.OrigemRepasseFomento || *conta.OrigemID != recebimento.Repasse.ID || *conta.FornecedorID != fomento ||
		conta.Valor.Centavos() != 150000 {
		t.Fatalf("conta do repasse = %+v", conta)
	}
}

func TestLiquidacaoNoFomentoNaoGeraRepasse(t *testing.T) {
	c := novoCenario(t)
	titulo, _, _ := c.antecipado(t)

	_, recebimento, err := c.baixas.Receber(c.ctx, portsin.ReceberInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: titulo.Versao, Tipo: "fomento",
		RecebidoEm: domainreceber.Hoje(time.Now().UTC()), ValorCentavos: 150000,
	})
	if err != nil {
		t.Fatalf("liquidação no fomento falhou: %v", err)
	}
	if recebimento.Repasse != nil || c.m.baixas[0].ContaDoRepasse != nil || recebimento.EntradaNoCaixaCentavos() != 0 {
		t.Fatalf("recebimento = %+v", recebimento)
	}
}
