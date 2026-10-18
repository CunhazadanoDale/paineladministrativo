package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

func (c *cenarioBancoTitulos) tituloDoResumo(t *testing.T, empresaID uuid.UUID, numero string, vencimento time.Time, status string, recebido int64) uuid.UUID {
	t.Helper()

	titulo := c.tituloDeDominio(t, numero, diaDoTitulo(12, 31))
	titulo.EmpresaID = empresaID
	if err := c.criar(titulo); err != nil {
		t.Fatalf("criar título do resumo falhou: %v", err)
	}
	conta := any(nil)
	if status == "em_banco" {
		conta = c.contaID
	}
	if _, err := c.banco.Exec(`
		UPDATE titulo_receber
		SET vencimento = $2, status = $3::varchar, recebido_centavos = $4, conta_bancaria_id = $5,
		    cancelado_motivo = CASE WHEN $3::varchar = 'cancelado' THEN 'lançado errado' END
		WHERE id = $1
	`, titulo.ID, vencimento, status, recebido, conta); err != nil {
		t.Fatalf("ajustar título do resumo falhou: %v", err)
	}

	return titulo.ID
}

func TestResumoDeContasAReceberSeparaAReceberAntecipadoEVencido(t *testing.T) {
	c := novoCenarioTitulos(t)
	resumos := postgres.NewContasAReceberResumoRepository(c.banco)
	outra := inserirEmpresa(t, c.banco, "45.723.174/0001-10", "Red Diamond")
	hoje := diaDoTitulo(10, 15)

	c.tituloDoResumo(t, c.empresaID, "1", diaDoTitulo(10, 20), "carteira", 0)
	c.tituloDoResumo(t, c.empresaID, "2", diaDoTitulo(10, 22), "em_banco", 30000)
	c.tituloDoResumo(t, c.empresaID, "3", diaDoTitulo(10, 21), "antecipado", 0)
	c.tituloDoResumo(t, c.empresaID, "4", diaDoTitulo(10, 13), "carteira", 0)
	c.tituloDoResumo(t, c.empresaID, "5", diaDoTitulo(10, 1), "em_banco", 0)
	c.tituloDoResumo(t, c.empresaID, "6", diaDoTitulo(10, 20), "recebido", 150000)
	c.tituloDoResumo(t, c.empresaID, "7", diaDoTitulo(10, 20), "cancelado", 0)
	c.tituloDoResumo(t, outra, "1", diaDoTitulo(10, 20), "carteira", 0)

	filtro := portsreceber.FiltroResumo{De: diaDoTitulo(10, 12), Ate: diaDoTitulo(10, 25), Hoje: hoje, EmpresaIDs: []uuid.UUID{c.empresaID}}
	semanas, err := resumos.ResumoSemanal(c.ctx, filtro)
	if err != nil {
		t.Fatalf("ResumoSemanal falhou: %v", err)
	}
	if len(semanas) != 2 {
		t.Fatalf("semanas = %+v", semanas)
	}
	primeira, segunda := semanas[0], semanas[1]
	if !primeira.Semana.Equal(diaDoTitulo(10, 12)) || primeira.Vencido != (domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 150000}) ||
		primeira.AReceber.Quantidade != 0 {
		t.Fatalf("semana de 12/10 = %+v", primeira)
	}
	if !segunda.Semana.Equal(diaDoTitulo(10, 19)) || segunda.AReceber != (domainreceber.TotalResumo{Quantidade: 2, SaldoCentavos: 270000}) ||
		segunda.Antecipado != (domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 150000}) || segunda.Vencido.Quantidade != 0 {
		t.Fatalf("semana de 19/10 = %+v", segunda)
	}

	atrasados, err := resumos.Atrasados(c.ctx, filtro)
	if err != nil || len(atrasados) != 1 || atrasados[0].Vencido != (domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 150000}) {
		t.Fatalf("atrasados = %+v, %v", atrasados, err)
	}

	todas, err := resumos.ResumoSemanal(c.ctx, portsreceber.FiltroResumo{De: diaDoTitulo(10, 12), Ate: diaDoTitulo(10, 25), Hoje: hoje})
	if err != nil || len(todas) != 3 {
		t.Fatalf("resumo sem filtro de empresa = %+v, %v", todas, err)
	}
	nenhuma, err := resumos.ResumoSemanal(c.ctx, portsreceber.FiltroResumo{De: diaDoTitulo(10, 12), Ate: diaDoTitulo(10, 25), Hoje: hoje, EmpresaIDs: []uuid.UUID{}})
	if err != nil || len(nenhuma) != 0 {
		t.Fatalf("resumo sem acesso = %+v, %v", nenhuma, err)
	}
}

func TestListarBoletosPendentes(t *testing.T) {
	c := novoCenarioTitulos(t)
	pendente := c.tituloDoResumo(t, c.empresaID, "1", diaDoTitulo(10, 18), "em_banco", 0)
	vencido := c.tituloDoResumo(t, c.empresaID, "2", diaDoTitulo(10, 10), "em_banco", 0)
	c.tituloDoResumo(t, c.empresaID, "3", diaDoTitulo(10, 30), "em_banco", 0)
	c.tituloDoResumo(t, c.empresaID, "4", diaDoTitulo(10, 18), "carteira", 0)
	enviado := c.tituloDoResumo(t, c.empresaID, "5", diaDoTitulo(10, 18), "em_banco", 0)
	if _, err := c.banco.Exec(`UPDATE titulo_receber SET boleto_enviado_em = now(), boleto_canal = 'email' WHERE id = $1`, enviado); err != nil {
		t.Fatal(err)
	}

	limite := diaDoTitulo(10, 20)
	esperarIDs(t, c.listar(t, portsreceber.TituloFiltro{BoletoPendenteAte: &limite}), vencido, pendente)
}
