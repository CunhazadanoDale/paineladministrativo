package caixa_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

func (c *cenario) subcategoria(natureza domainfinanceiro.Natureza) uuid.UUID {
	pai := uuid.New()
	categoria := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), Nome: "Sub " + natureza.String(), CategoriaPaiID: &pai, Natureza: natureza, Ativa: true}
	c.categoriasRepo.itens[categoria.ID] = categoria

	return categoria.ID
}

func TestClassificarLinhaDoExtrato(t *testing.T) {
	c := novoCenario(t)
	tarifa := c.lancamento(c.sicrediMIA, -2, -1290)
	despesa := c.subcategoria(domainfinanceiro.NaturezaDespesa)

	classificacao, err := c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: tarifa.ID, CategoriaFinanceiraID: despesa, Descricao: "Tarifa do pacote",
	})
	if err != nil {
		t.Fatalf("Classificar = %v", err)
	}
	lancamento := classificacao.Lancamento
	if lancamento.EmpresaID != c.mia || lancamento.ContaBancariaID != c.sicrediMIA || lancamento.ValorCentavos != -1290 ||
		lancamento.Descricao != "Tarifa do pacote" || lancamento.UsuarioID != c.financeiroMIA {
		t.Errorf("lançamento de caixa = %+v", lancamento)
	}
	if len(c.lancamentosRepo.classificacoes) != 1 || classificacao.Conciliacao.Modo != domaincaixa.ConciliacaoClassificacao {
		t.Errorf("classificação gravada = %+v", c.lancamentosRepo.classificacoes)
	}

	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: tarifa.ID, CategoriaFinanceiraID: c.subcategoria(domainfinanceiro.NaturezaReceita),
	})
	esperarErro(t, err, domain.ErrValidacao, "linha de débito só recebe categoria de despesa")
	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{UsuarioID: c.financeiroMIA, LancamentoID: tarifa.ID})
	esperarErro(t, err, domain.ErrValidacao, "categoria financeira é obrigatória")
	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: tarifa.ID, CategoriaFinanceiraID: uuid.New(),
	})
	esperarErro(t, err, domain.ErrValidacao, "categoria financeira não encontrada")
	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.financeiroRed, LancamentoID: tarifa.ID, CategoriaFinanceiraID: despesa,
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.comum, LancamentoID: tarifa.ID, CategoriaFinanceiraID: despesa,
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
	_, err = c.lancamentosCaixa.Classificar(c.ctx, portsin.ClassificarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: uuid.New(), CategoriaFinanceiraID: despesa,
	})
	esperarErro(t, err, domain.ErrNotFound, "lançamento do extrato não encontrado")
}

func TestTransferirEntreContas(t *testing.T) {
	c := novoCenario(t)
	aplicacao := c.novaConta(t, c.mia, domainfinanceiro.Sicredi, "0812", "99999", "", true)
	categoria := c.subcategoria(domainfinanceiro.NaturezaTransferencia)
	aplicado := c.lancamento(c.sicrediMIA, -3, -500000)

	registrada, err := c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.financeiroMIA, LancamentoID: aplicado.ID, ContraparteID: aplicacao, CategoriaFinanceiraID: categoria,
	})
	if err != nil {
		t.Fatalf("Transferir = %v", err)
	}
	if registrada.Transferencia.ContaOrigemID != c.sicrediMIA || registrada.Transferencia.ContaDestinoID != aplicacao ||
		registrada.Transferencia.EntreEmpresas || len(c.lancamentosRepo.transferencias) != 1 {
		t.Errorf("transferência = %+v", registrada.Transferencia)
	}

	paraRed := c.lancamento(c.sicrediMIA, -2, -1000)
	_, err = c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.financeiroMIA, LancamentoID: paraRed.ID, ContraparteID: c.itauRed, CategoriaFinanceiraID: categoria,
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")

	entreEmpresas, err := c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.administrador, LancamentoID: paraRed.ID, ContraparteID: c.itauRed, CategoriaFinanceiraID: categoria,
	})
	if err != nil || !entreEmpresas.Transferencia.EntreEmpresas {
		t.Errorf("transferência entre empresas = %+v, %v", entreEmpresas, err)
	}

	_, err = c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.financeiroMIA, LancamentoID: aplicado.ID, CategoriaFinanceiraID: categoria,
	})
	esperarErro(t, err, domain.ErrValidacao, "conta da outra ponta é obrigatória")
	_, err = c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.financeiroMIA, LancamentoID: aplicado.ID, ContraparteID: uuid.New(), CategoriaFinanceiraID: categoria,
	})
	esperarErro(t, err, domain.ErrValidacao, "conta da outra ponta não encontrada")
	_, err = c.lancamentosCaixa.Transferir(c.ctx, portsin.TransferirInput{
		UsuarioID: c.financeiroMIA, LancamentoID: aplicado.ID, ContraparteID: aplicacao,
		CategoriaFinanceiraID: c.subcategoria(domainfinanceiro.NaturezaDespesa),
	})
	esperarErro(t, err, domain.ErrValidacao, "só recebe categoria de transferencia")
}

func TestListarLancamentosETransferenciasRespeitaOAcesso(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.lancamentosCaixa.Listar(c.ctx, portsin.ListarCaixaInput{
		UsuarioID: c.financeiroRed, Filtro: domain.PaginacaoFiltro{Page: 3, Size: 20},
	}); err != nil {
		t.Fatalf("Listar = %v", err)
	}
	filtro := c.lancamentosRepo.filtros[0]
	if len(filtro.EmpresaIDs) != 1 || filtro.EmpresaIDs[0] != c.red || filtro.Limite != 20 || filtro.Deslocamento != 40 {
		t.Errorf("filtro = %+v", filtro)
	}

	_, err := c.lancamentosCaixa.ListarTransferencias(c.ctx, portsin.ListarCaixaInput{UsuarioID: c.financeiroRed, ContaBancariaID: &c.sicrediMIA})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.lancamentosCaixa.Listar(c.ctx, portsin.ListarCaixaInput{UsuarioID: c.financeiroRed, Empresa: &c.mia})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.lancamentosCaixa.Listar(c.ctx, portsin.ListarCaixaInput{
		UsuarioID: c.financeiroMIA, De: timePtr(dia(-1)), Ate: timePtr(dia(-3)),
	})
	esperarErro(t, err, domain.ErrValidacao, "data final anterior à inicial")
}
