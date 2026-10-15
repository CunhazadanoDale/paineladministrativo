package solicitacao_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	solicitacaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/solicitacao"
	"github.com/google/uuid"
)

type repositorioResumoFalso struct {
	semanas   []domainsolicitacao.LinhaResumoSemanal
	atrasadas []domainsolicitacao.AtrasoPorEmpresa
	filtro    *portsout.FiltroResumo
}

var _ portsout.ResumoContasAPagarRepository = (*repositorioResumoFalso)(nil)

func (r *repositorioResumoFalso) ResumoSemanal(_ context.Context, filtro portsout.FiltroResumo) ([]domainsolicitacao.LinhaResumoSemanal, error) {
	r.filtro = &filtro
	return r.semanas, nil
}

func (r *repositorioResumoFalso) Atrasadas(_ context.Context, _ portsout.FiltroResumo) ([]domainsolicitacao.AtrasoPorEmpresa, error) {
	return r.atrasadas, nil
}

func (c *cenario) usecaseDeResumo(repo *repositorioResumoFalso) portsin.ResumoContasAPagarUseCase {
	return solicitacaousecases.NewResumoContasAPagarUsecase(repo, c.aprovadoresRepo, c.usuariosRepo, c.cargosRepo, c.empresasRepo)
}

func valorDe(ponteiro *int) int {
	if ponteiro == nil {
		return -1
	}

	return *ponteiro
}

func TestPendenciasContamSoAsFilasDoPerfilENoAcesso(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	aprovador := c.novoUsuario("Bruno", false, false)
	c.designar(t, aprovador)
	financeiro := c.novoUsuario("Fábio", false, true)
	administrador := c.novoUsuario("Admin", true, false)
	outra := c.novaEmpresa(true)
	c.restringirAcesso(financeiro, c.empresaID)

	c.criarSolicitacao(t, solicitante, 10000)
	aprovadaEmDia := c.criarSolicitacao(t, solicitante, 10000)
	aprovadaVencida := c.criarSolicitacao(t, solicitante, 10000)
	daOutra := c.criarSolicitacaoNaEmpresa(t, solicitante, outra)
	for _, id := range []uuid.UUID{aprovadaEmDia, aprovadaVencida, daOutra} {
		if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); err != nil {
			t.Fatalf("aprovação falhou: %v", err)
		}
	}
	c.solicitacoesRepo.itens[aprovadaVencida].Prazo = domainsolicitacao.PrazoPagamentoDe(time.Now().UTC().AddDate(0, 0, -3))

	casos := map[string]struct {
		usuarioID                       uuid.UUID
		aprovacao, financeiro, vencidas int
	}{
		"solicitante comum":         {solicitante, -1, -1, -1},
		"aprovador designado":       {aprovador, 1, -1, -1},
		"financeiro de uma empresa": {financeiro, -1, 2, 1},
		"administrador":             {administrador, 1, 3, 1},
	}

	for nome, caso := range casos {
		pendencias, err := c.solicitacao.Pendencias(c.ctx, caso.usuarioID)
		if err != nil {
			t.Fatalf("%s: pendências falharam: %v", nome, err)
		}
		if valorDe(pendencias.Aprovacao) != caso.aprovacao || valorDe(pendencias.Financeiro) != caso.financeiro || valorDe(pendencias.Vencidas) != caso.vencidas {
			t.Errorf("%s = aprovação %d, financeiro %d, vencidas %d; esperado %d, %d, %d", nome,
				valorDe(pendencias.Aprovacao), valorDe(pendencias.Financeiro), valorDe(pendencias.Vencidas),
				caso.aprovacao, caso.financeiro, caso.vencidas)
		}
	}
}

func TestResumoDeContasAPagarExigeFinanceiroERespeitaOAcesso(t *testing.T) {
	c := novoCenario(t)
	comum := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Fábio", false, true)
	outra := c.novaEmpresa(true)
	c.restringirAcesso(financeiro, c.empresaID)
	repo := &repositorioResumoFalso{}
	resumo := c.usecaseDeResumo(repo)

	if _, err := resumo.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: comum}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("sem perfil = %v", err)
	}
	if _, err := resumo.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: financeiro, Empresa: &outra}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v", err)
	}
	de := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	ate := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	if _, err := resumo.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: financeiro, De: &de, Ate: &ate}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("intervalo invertido = %v", err)
	}

	if _, err := resumo.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: financeiro}); err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}
	if len(repo.filtro.EmpresaIDs) != 1 || repo.filtro.EmpresaIDs[0] != c.empresaID {
		t.Errorf("empresas do filtro = %v, esperado só a do acesso", repo.filtro.EmpresaIDs)
	}
	hoje := time.Now().UTC()
	if !repo.filtro.De.Equal(domainsolicitacao.InicioDaSemana(hoje)) || repo.filtro.Hoje.Format("2006-01-02") != hoje.Format("2006-01-02") {
		t.Errorf("período padrão = %s, hoje = %s", repo.filtro.De, repo.filtro.Hoje)
	}
}

func TestResumoDeContasAPagarSomaOsTotais(t *testing.T) {
	c := novoCenario(t)
	administrador := c.novoUsuario("Admin", true, false)
	repo := &repositorioResumoFalso{
		semanas: []domainsolicitacao.LinhaResumoSemanal{
			{EmpresaID: c.empresaID, APagar: domainsolicitacao.TotalResumo{Quantidade: 2, ValorCentavos: 500}},
		},
		atrasadas: []domainsolicitacao.AtrasoPorEmpresa{
			{EmpresaID: c.empresaID, Vencido: domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 90}},
		},
	}

	resultado, err := c.usecaseDeResumo(repo).Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: administrador, Empresa: &c.empresaID})
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}
	if resultado.Totais.APagar.ValorCentavos != 500 || resultado.Totais.Vencido.ValorCentavos != 90 || len(resultado.Atrasadas) != 1 {
		t.Errorf("resumo = %+v", resultado)
	}
	if repo.filtro.EmpresaIDs[0] != c.empresaID {
		t.Errorf("empresa do filtro = %v", repo.filtro.EmpresaIDs)
	}
}
