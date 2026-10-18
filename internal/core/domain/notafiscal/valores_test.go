package notafiscal_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
)

func TestNovaChaveValidaEDecompoe(t *testing.T) {
	valor := chaveDe(cnpjEmitente, "2026-10-10", 1, 123, "55")

	chave, err := domainnotafiscal.NovaChave(" " + valor + " ")
	if err != nil {
		t.Fatalf("NovaChave = %v", err)
	}

	casos := map[string][2]string{
		"uf":       {chave.CodigoUF(), "25"},
		"ano mês":  {chave.AnoMes(), "2610"},
		"cnpj":     {chave.CNPJEmitente(), cnpjEmitente},
		"modelo":   {chave.Modelo(), "55"},
		"emissão":  {chave.TipoEmissao(), "1"},
		"numérico": {chave.CodigoNumerico(), "12345678"},
	}
	for nome, caso := range casos {
		if caso[0] != caso[1] {
			t.Errorf("%s = %q, esperado %q", nome, caso[0], caso[1])
		}
	}
	if chave.Serie() != 1 || chave.Numero() != 123 {
		t.Errorf("série %d e número %d, esperado 1 e 123", chave.Serie(), chave.Numero())
	}
}

func TestNovaChaveAceitaCNPJAlfanumerico(t *testing.T) {
	valor := chaveDe(cnpjAlfanumerico, "2026-10-10", 1, 9, "55")

	chave, err := domainnotafiscal.NovaChave(valor)
	if err != nil {
		t.Fatalf("NovaChave alfanumérica = %v", err)
	}
	if chave.CNPJEmitente() != cnpjAlfanumerico {
		t.Errorf("CNPJ da chave = %q", chave.CNPJEmitente())
	}
}

func TestNovaChaveRecusaInvalidas(t *testing.T) {
	valida := chaveDe(cnpjEmitente, "2026-10-10", 1, 123, "55")
	trocado := valida[:43] + string(rune('0'+(int(valida[43]-'0')+1)%10))

	casos := map[string]string{
		"curta":              valida[:43],
		"dígito errado":      trocado,
		"letra fora do cnpj": "A" + valida[1:],
		"vazia":              "",
	}
	for nome, valor := range casos {
		if _, err := domainnotafiscal.NovaChave(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado validação", nome, err)
		}
	}
}

func TestCentavosDe(t *testing.T) {
	casos := map[string]int64{"0": 0, "10": 1000, "10.5": 1050, "1088.50": 108850, " 0.01 ": 1}
	for texto, esperado := range casos {
		valor, err := domainnotafiscal.CentavosDe(texto, "valor")
		if err != nil || valor != esperado {
			t.Errorf("CentavosDe(%q) = %d, %v; esperado %d", texto, valor, err, esperado)
		}
	}

	for _, texto := range []string{"", "1.234", "-1.00", "1,00", "abc", "1.", "99999999999999"} {
		if _, err := domainnotafiscal.CentavosDe(texto, "valor"); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("CentavosDe(%q) = %v, esperado validação", texto, err)
		}
	}

	if valor, err := domainnotafiscal.CentavosOpcionaisDe(" ", "valor"); err != nil || valor != 0 {
		t.Errorf("CentavosOpcionaisDe vazio = %d, %v", valor, err)
	}
}

func TestQuantidadeEValorUnitario(t *testing.T) {
	quantidade, err := domainnotafiscal.QuantidadeDe("10.5", "quantidade")
	if err != nil || quantidade != 105000 {
		t.Errorf("QuantidadeDe = %d, %v; esperado 105000", quantidade, err)
	}
	if _, err := domainnotafiscal.QuantidadeDe("0.0000", "quantidade"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("quantidade zero = %v, esperado validação", err)
	}
	if _, err := domainnotafiscal.QuantidadeDe("1.00001", "quantidade"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("quantidade com 5 casas = %v, esperado validação", err)
	}

	if valor, err := domainnotafiscal.ValorUnitarioDe("12.3456789012", "unitário"); err != nil || valor != "12.3456789012" {
		t.Errorf("ValorUnitarioDe = %q, %v", valor, err)
	}
	if _, err := domainnotafiscal.ValorUnitarioDe("1.12345678901", "unitário"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("unitário com 11 casas = %v, esperado validação", err)
	}
}

func TestClassificarNatureza(t *testing.T) {
	casos := []struct {
		nome          string
		entreEmpresas bool
		finalidade    domainnotafiscal.Finalidade
		cfop          string
		esperada      domainnotafiscal.Natureza
	}{
		{"venda", false, domainnotafiscal.FinalidadeNormal, "5102", domainnotafiscal.NaturezaVenda},
		{"simples faturamento", false, domainnotafiscal.FinalidadeNormal, "5922", domainnotafiscal.NaturezaVenda},
		{"remessa de entrega futura", false, domainnotafiscal.FinalidadeNormal, "6116", domainnotafiscal.NaturezaRemessa},
		{"bonificação", false, domainnotafiscal.FinalidadeNormal, "5910", domainnotafiscal.NaturezaRemessa},
		{"devolução", false, domainnotafiscal.FinalidadeDevolucao, "5202", domainnotafiscal.NaturezaDevolucao},
		{"devolução de remessa", false, domainnotafiscal.FinalidadeDevolucao, "5949", domainnotafiscal.NaturezaDevolucao},
		{"entre empresas", true, domainnotafiscal.FinalidadeDevolucao, "5102", domainnotafiscal.NaturezaTransferencia},
	}
	for _, caso := range casos {
		if natureza := domainnotafiscal.ClassificarNatureza(caso.entreEmpresas, caso.finalidade, caso.cfop); natureza != caso.esperada {
			t.Errorf("%s: natureza = %q, esperado %q", caso.nome, natureza, caso.esperada)
		}
	}

	if !domainnotafiscal.NaturezaVenda.GeraTitulo() || domainnotafiscal.NaturezaRemessa.GeraTitulo() ||
		domainnotafiscal.NaturezaTransferencia.GeraTitulo() || domainnotafiscal.NaturezaDevolucao.GeraTitulo() {
		t.Error("só venda gera título")
	}
}

func TestEnumsDoModulo(t *testing.T) {
	if natureza, err := domainnotafiscal.NovaNatureza(" Remessa "); err != nil || natureza != domainnotafiscal.NaturezaRemessa {
		t.Errorf("NovaNatureza = %q, %v", natureza, err)
	}
	if _, err := domainnotafiscal.NovaNatureza("bonificacao"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("natureza desconhecida = %v", err)
	}
	if status, err := domainnotafiscal.NovoStatus("CANCELADA"); err != nil || status != domainnotafiscal.StatusCancelada {
		t.Errorf("NovoStatus = %q, %v", status, err)
	}
	if _, err := domainnotafiscal.NovoStatus("denegada"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status desconhecido = %v", err)
	}
	if finalidade, err := domainnotafiscal.FinalidadeDoCodigo("4"); err != nil || finalidade != domainnotafiscal.FinalidadeDevolucao {
		t.Errorf("FinalidadeDoCodigo = %q, %v", finalidade, err)
	}
	if _, err := domainnotafiscal.FinalidadeDoCodigo("5"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("finalidade 5 = %v", err)
	}
	if tipo, err := domainnotafiscal.TipoOperacaoDoCodigo("0"); err != nil || tipo != domainnotafiscal.TipoOperacaoEntrada {
		t.Errorf("TipoOperacaoDoCodigo = %q, %v", tipo, err)
	}
	if tipo, err := domainnotafiscal.TipoEventoDoCodigo("110110"); err != nil || tipo != domainnotafiscal.TipoEventoCartaCorrecao {
		t.Errorf("TipoEventoDoCodigo = %q, %v", tipo, err)
	}
	if _, err := domainnotafiscal.TipoEventoDoCodigo("210200"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("evento de manifestação = %v", err)
	}
}
