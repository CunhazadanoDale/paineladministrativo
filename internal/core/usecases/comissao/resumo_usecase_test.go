package comissao_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
)

func TestResumoDoFinanceiroRespeitaOAcessoPorEmpresa(t *testing.T) {
	c := novoCenario(t)
	c.m.resumo = []domaincomissao.LinhaResumo{
		{EmpresaID: c.mia, RepresentanteID: c.carlos.ID, ComissoesCentavos: 1000, AjustesCentavos: -200, PagoCentavos: 800},
		{EmpresaID: c.mia, RepresentanteID: c.beatriz.ID, ComissoesCentavos: 500},
	}

	resumo, err := c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.financeiro, Competencia: "2026-09"})
	if err != nil {
		t.Fatalf("Resumo: %v", err)
	}
	if resumo.Competencia.String() != "2026-09" || len(resumo.Linhas) != 2 || resumo.TotalCentavos() != 1300 {
		t.Fatalf("resumo = %+v", resumo)
	}
	filtro := c.m.ultimoFiltroResumo
	if filtro.Competencia.String() != "2026-09" || filtro.EmpresaIDs[0] != c.mia || filtro.RepresentanteID != nil {
		t.Fatalf("filtro = %+v", filtro)
	}

	_, err = c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.financeiro, Competencia: "2026-09", Empresa: &c.red})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso a esta empresa")

	_, err = c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.financeiro})
	esperarErro(t, err, domain.ErrValidacao, "AAAA-MM")
}

func TestResumoDoRepresentanteMostraSoOProprio(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.usuarioCarlos, Competencia: "2026-09"}); err != nil {
		t.Fatalf("Resumo: %v", err)
	}
	if filtro := c.m.ultimoFiltroResumo; filtro.RepresentanteID == nil || *filtro.RepresentanteID != c.carlos.ID {
		t.Fatalf("filtro do representante = %+v", filtro)
	}

	_, err := c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.usuarioCarlos, Competencia: "2026-09", RepresentanteID: &c.beatriz.ID})
	esperarErro(t, err, domain.ErrPermissao, "outro representante")

	_, err = c.comissoes.Resumo(c.ctx, portsin.ResumoInput{UsuarioID: c.comercial, Competencia: "2026-09"})
	esperarErro(t, err, domain.ErrPermissao, "ver comissões")
}
