package caixa_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

func (c *cenario) recorrencia(empresaID uuid.UUID, dia int, valor int64) *domainsolicitacao.DespesaRecorrente {
	despesa := &domainsolicitacao.DespesaRecorrente{
		ID:            uuid.New(),
		EmpresaID:     empresaID,
		Valor:         domainsolicitacao.ValorDe(valor),
		DiaVencimento: domainsolicitacao.DiaVencimentoDe(dia),
		Inicio:        domainsolicitacao.CompetenciaDe(dia0().AddDate(-1, 0, 0)),
		Ativa:         true,
	}
	c.recorrenciasRepo.itens = append(c.recorrenciasRepo.itens, despesa)

	return despesa
}

func dia0() time.Time {
	return domaincaixa.Hoje(time.Now().UTC())
}

func TestProjecaoSomaPrevistosERecorrentesNaoGeradas(t *testing.T) {
	c := novoCenario(t)
	c.projecaoRepo.partida = domaincaixa.SaldoDePartida{ExtratoCentavos: 100000, EmTransitoCentavos: -2000, ContasConsideradas: 1}
	c.projecaoRepo.entradas = []domaincaixa.Previsto{{Data: dia0().AddDate(0, 0, 3), ValorCentavos: 30000}}
	c.projecaoRepo.saidas = []domaincaixa.Previsto{
		{Data: dia0().AddDate(0, 0, -2), ValorCentavos: 1000},
		{Data: dia0().AddDate(0, 0, 5), ValorCentavos: 4000, Estimado: true},
	}
	aluguel := c.recorrencia(c.mia, 10, 5000)
	c.recorrencia(c.red, 10, 777)
	mesPassado := domaincaixa.CompetenciaDe(dia0()).AddDate(0, -1, 0)
	c.projecaoRepo.geradas = map[portsout.RecorrenciaNaCompetencia]bool{
		{RecorrenciaID: aluguel.ID, Competencia: mesPassado}: true,
	}

	ate := dia0().AddDate(0, 0, 60)
	projecao, err := c.projecao.Projetar(c.ctx, portsin.ProjetarInput{UsuarioID: c.financeiroMIA, Ate: &ate})
	if err != nil {
		t.Fatalf("Projetar = %v", err)
	}
	if len(projecao.Dias) != 61 || !projecao.De.Equal(dia0()) || !projecao.Ate.Equal(ate) {
		t.Fatalf("período = %v a %v com %d dias", projecao.De, projecao.Ate, len(projecao.Dias))
	}
	if empresas := c.projecaoRepo.empresas[0]; len(empresas) != 2 {
		t.Errorf("empresas consultadas = %v", empresas)
	}

	var recorrentes, entradas, estimadas, vencidas int64
	for _, dia := range projecao.Dias {
		recorrentes += dia.RecorrentesCentavos
		entradas += dia.EntradasCentavos
		estimadas += dia.SaidasEstimadasCentavos
		vencidas += dia.SaidasVencidasCentavos
	}
	consultadas := len(c.projecaoRepo.consultadas)
	if consultadas < 2 {
		t.Fatalf("recorrências consultadas = %+v", c.projecaoRepo.consultadas)
	}
	for _, consultada := range c.projecaoRepo.consultadas {
		if consultada.RecorrenciaID != aluguel.ID {
			t.Errorf("recorrência de outra empresa consultada: %+v", consultada)
		}
	}
	if recorrentes != int64(consultadas-1)*5000 || entradas != 30000 || estimadas != 4000 || vencidas != 1000 {
		t.Errorf("recorrentes %d, entradas %d, estimadas %d, vencidas %d", recorrentes, entradas, estimadas, vencidas)
	}
	if esperado := 98000 + 30000 - 4000 - 1000 - recorrentes; projecao.SaldoFinalCentavos() != esperado {
		t.Errorf("saldo final = %d, esperado %d", projecao.SaldoFinalCentavos(), esperado)
	}
}

func TestProjecaoValidaPerfilEmpresaEHorizonte(t *testing.T) {
	c := novoCenario(t)

	_, err := c.projecao.Projetar(c.ctx, portsin.ProjetarInput{UsuarioID: c.comum})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
	_, err = c.projecao.Projetar(c.ctx, portsin.ProjetarInput{UsuarioID: c.financeiroMIA, Empresa: &c.red})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	ontem := dia0().AddDate(0, 0, -1)
	_, err = c.projecao.Projetar(c.ctx, portsin.ProjetarInput{UsuarioID: c.financeiroMIA, Ate: &ontem})
	esperarErro(t, err, domain.ErrValidacao, "a projeção começa hoje")

	padrao, err := c.projecao.Projetar(c.ctx, portsin.ProjetarInput{UsuarioID: c.administrador})
	if err != nil || len(padrao.Dias) != domaincaixa.HorizontePadraoDaProjecao+1 || c.projecaoRepo.empresas[0] != nil {
		t.Errorf("padrão do administrador = %d dias, empresas %v, %v", len(padrao.Dias), c.projecaoRepo.empresas, err)
	}
}

func TestMovimentoDiario(t *testing.T) {
	c := novoCenario(t)
	c.projecaoRepo.movimentos = []*domaincaixa.MovimentoDiario{{Data: dia0(), ContaBancariaID: c.sicrediMIA, EntradasCentavos: 10}}

	movimentos, err := c.projecao.MovimentoDiario(c.ctx, portsin.MovimentoDiarioInput{UsuarioID: c.financeiroMIA, ContaBancariaID: &c.sicrediMIA})
	if err != nil || len(movimentos) != 1 {
		t.Fatalf("MovimentoDiario = %+v, %v", movimentos, err)
	}
	filtro := c.projecaoRepo.filtroDiario
	if filtro.ContaBancariaID == nil || *filtro.ContaBancariaID != c.sicrediMIA || !filtro.De.Equal(domaincaixa.CompetenciaDe(dia0())) ||
		!filtro.Ate.Equal(dia0()) || len(filtro.EmpresaIDs) != 2 {
		t.Errorf("filtro = %+v", filtro)
	}

	_, err = c.projecao.MovimentoDiario(c.ctx, portsin.MovimentoDiarioInput{UsuarioID: c.financeiroMIA, ContaBancariaID: &c.itauRed})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	de, ate := dia0().AddDate(0, 0, -100), dia0()
	_, err = c.projecao.MovimentoDiario(c.ctx, portsin.MovimentoDiarioInput{UsuarioID: c.financeiroMIA, De: &de, Ate: &ate})
	esperarErro(t, err, domain.ErrValidacao, "no máximo 92 dias")
}
