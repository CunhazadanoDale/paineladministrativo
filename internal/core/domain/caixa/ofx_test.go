package caixa_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

const ofxSGML = `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<SIGNONMSGSRSV1><SONRS><STATUS><CODE>0<SEVERITY>INFO</STATUS><DTSERVER>20261015120000[-3:BRT]<LANGUAGE>POR</SONRS></SIGNONMSGSRSV1>
<BANKMSGSRSV1>
<STMTTRNRS>
<TRNUID>1
<STATUS><CODE>0<SEVERITY>INFO</STATUS>
<STMTRS>
<CURDEF>BRL
<BANKACCTFROM>
<BANKID>0748
<BRANCHID>0812
<ACCTID>54321-0
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20261001000000[-3:BRT]
<DTEND>20261015000000[-3:BRT]
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20261005000000[-3:BRT]
<TRNAMT>-1500.00
<FITID>2026100501
<CHECKNUM>000123
<MEMO>PAGAMENTO   FORNECEDOR &amp; CIA
</STMTTRN>
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20261006
<TRNAMT>2.345,67
<FITID>2026100601
<NAME>TED RECEBIDA
<BANKACCTTO><BANKID>001<ACCTID>999</BANKACCTTO>
</STMTTRN>
</BANKTRANLIST>
<LEDGERBAL>
<BALAMT>10845.67
<DTASOF>20261015000000[-3:BRT]
</LEDGERBAL>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`

const ofxXML = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?OFX OFXHEADER="200" VERSION="211" SECURITY="NONE" OLDFILEUID="NONE" NEWFILEUID="NONE"?>
<OFX>
  <BANKMSGSRSV1>
    <STMTTRNRS>
      <STMTRS>
        <CURDEF>BRL</CURDEF>
        <BANKACCTFROM>
          <BANKID>341</BANKID>
          <BRANCHID>1234</BRANCHID>
          <ACCTID>98765-4</ACCTID>
          <ACCTTYPE>CHECKING</ACCTTYPE>
        </BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <TRNTYPE>FEE</TRNTYPE>
            <DTPOSTED>20261003100000</DTPOSTED>
            <TRNAMT>-12.9</TRNAMT>
            <FITID>A1</FITID>
            <MEMO>Tarifa pacote</MEMO>
          </STMTTRN>
        </BANKTRANLIST>
      </STMTRS>
    </STMTTRNRS>
  </BANKMSGSRSV1>
</OFX>`

func TestLerOFXVersaoSGML(t *testing.T) {
	extrato, err := domaincaixa.LerOFX([]byte(ofxSGML))
	if err != nil {
		t.Fatalf("LerOFX = %v", err)
	}

	if extrato.BancoID != "0748" || extrato.Banco() != "748" || extrato.AgenciaID != "0812" || extrato.ContaID != "54321-0" {
		t.Errorf("conta do extrato = %+v", extrato)
	}
	if extrato.Inicio == nil || !extrato.Inicio.Equal(data(2026, 10, 1)) || extrato.Fim == nil || !extrato.Fim.Equal(data(2026, 10, 15)) {
		t.Errorf("período = %v a %v", extrato.Inicio, extrato.Fim)
	}
	if extrato.Saldo == nil || extrato.Saldo.Centavos != 1084567 || !extrato.Saldo.Em.Equal(data(2026, 10, 15)) {
		t.Errorf("saldo = %+v", extrato.Saldo)
	}
	if len(extrato.Transacoes) != 2 {
		t.Fatalf("transações = %d, esperadas 2", len(extrato.Transacoes))
	}

	debito, credito := extrato.Transacoes[0], extrato.Transacoes[1]
	if debito.FITID != "2026100501" || debito.ValorCentavos != -150000 || !debito.Data.Equal(data(2026, 10, 5)) {
		t.Errorf("débito = %+v", debito)
	}
	if debito.Descricao != "PAGAMENTO   FORNECEDOR & CIA" || debito.Documento != "000123" {
		t.Errorf("descrição e documento do débito = %q, %q", debito.Descricao, debito.Documento)
	}
	if credito.ValorCentavos != 234567 || credito.Descricao != "TED RECEBIDA" || credito.Documento != "" {
		t.Errorf("crédito = %+v", credito)
	}
}

func TestLerOFXVersaoXML(t *testing.T) {
	extrato, err := domaincaixa.LerOFX([]byte(ofxXML))
	if err != nil {
		t.Fatalf("LerOFX = %v", err)
	}

	if extrato.Banco() != "341" || extrato.ContaID != "98765-4" || extrato.Saldo != nil || extrato.Inicio != nil {
		t.Errorf("extrato = %+v", extrato)
	}
	if len(extrato.Transacoes) != 1 || extrato.Transacoes[0].ValorCentavos != -1290 || extrato.Transacoes[0].Descricao != "Tarifa pacote" {
		t.Errorf("transações = %+v", extrato.Transacoes)
	}
}

func TestLerOFXConverteWindows1252(t *testing.T) {
	conteudo := []byte(strings.Replace(ofxSGML, "TED RECEBIDA", "TED RECEBIDA JOAO", 1))
	conteudo = append([]byte(nil), conteudo...)
	indice := strings.Index(string(conteudo), "JOAO")
	conteudo[indice+2] = 0xC3

	extrato, err := domaincaixa.LerOFX(conteudo)
	if err != nil {
		t.Fatalf("LerOFX = %v", err)
	}
	if descricao := extrato.Transacoes[1].Descricao; descricao != "TED RECEBIDA JOÃO" {
		t.Errorf("descrição convertida = %q", descricao)
	}
}

func TestLerOFXRecusaArquivosInvalidos(t *testing.T) {
	casos := map[string]string{
		"sem OFX":        "<html><body>extrato</body></html>",
		"cartão":         strings.NewReplacer("BANKACCTFROM", "CCACCTFROM").Replace(ofxSGML),
		"duas contas":    strings.Replace(ofxSGML, "<BANKTRANLIST>", "<BANKACCTFROM><BANKID>1<ACCTID>2</BANKACCTFROM><BANKTRANLIST>", 1),
		"data inválida":  strings.Replace(ofxSGML, "<DTPOSTED>20261006", "<DTPOSTED>2026-10", 1),
		"valor inválido": strings.Replace(ofxSGML, "<TRNAMT>-1500.00", "<TRNAMT>mil", 1),
		"três decimais":  strings.Replace(ofxSGML, "<TRNAMT>-1500.00", "<TRNAMT>-1500.125", 1),
	}

	for nome, conteudo := range casos {
		if _, err := domaincaixa.LerOFX([]byte(conteudo)); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestLerOFXLimitaQuantidadeDeLancamentos(t *testing.T) {
	transacao := "<STMTTRN><DTPOSTED>20261005<TRNAMT>1.00<FITID>X</STMTTRN>\n"
	conteudo := strings.Replace(ofxSGML, "<STMTTRN>", strings.Repeat(transacao, domaincaixa.LimiteDeLancamentosDoExtrato)+"<STMTTRN>", 1)

	_, err := domaincaixa.LerOFX([]byte(conteudo))
	if !errors.Is(err, domain.ErrValidacao) || !strings.Contains(err.Error(), "exporte um período menor") {
		t.Errorf("erro = %v, esperado limite de lançamentos", err)
	}
}

func TestValoresDoOFX(t *testing.T) {
	casos := map[string]int64{
		"-1500.00":  -150000,
		"1500":      150000,
		"+12.5":     1250,
		"2.345,67":  234567,
		"1,234.56":  123456,
		"0,07":      7,
		"-.50":      -50,
		"12.3400":   1234,
		" -9 876.5": -987650,
	}

	for valor, esperado := range casos {
		conteudo := strings.Replace(ofxXML, "<TRNAMT>-12.9</TRNAMT>", "<TRNAMT>"+valor+"</TRNAMT>", 1)
		extrato, err := domaincaixa.LerOFX([]byte(conteudo))
		if err != nil {
			t.Errorf("%q: %v", valor, err)
			continue
		}
		if obtido := extrato.Transacoes[0].ValorCentavos; obtido != esperado {
			t.Errorf("%q = %d, esperado %d", valor, obtido, esperado)
		}
	}
}

func TestExtratoReconheceAConta(t *testing.T) {
	conta := contaBancaria(t, domainfinanceiro.Sicredi, "0812", "54321", "0")

	casos := map[string]struct {
		banco, agencia, numero string
		esperado               bool
	}{
		"número e dígito com hífen":  {"748", "812", "54321-0", true},
		"dígito colado":              {"0748", "0812", "543210", true},
		"sem dígito":                 {"748", "", "0054321", true},
		"agência colada no número":   {"748", "", "081254321-0", true},
		"agência com dígito":         {"748", "0812-3", "54321-0", true},
		"sem banco no extrato":       {"", "812", "54321", true},
		"outro banco":                {"341", "812", "54321-0", false},
		"outra agência":              {"748", "0813", "54321-0", false},
		"outra conta":                {"748", "812", "54322-0", false},
		"conta que só termina igual": {"748", "812", "954321-0", false},
	}

	for nome, caso := range casos {
		extrato := &domaincaixa.ExtratoOFX{BancoID: caso.banco, AgenciaID: caso.agencia, ContaID: caso.numero}
		if obtido := extrato.EhDaConta(conta); obtido != caso.esperado {
			t.Errorf("%s: EhDaConta = %v, esperado %v", nome, obtido, caso.esperado)
		}
	}

	if (&domaincaixa.ExtratoOFX{ContaID: " - "}).IdentificaConta() {
		t.Error("ACCTID sem dígitos não identifica a conta")
	}
	if (&domaincaixa.ExtratoOFX{BancoID: "12345"}).Banco() != "" {
		t.Error("BANKID com mais de três dígitos não é código COMPE")
	}
}

func contaBancaria(t *testing.T, banco domainfinanceiro.Banco, agencia, numero, digito string) *domainfinanceiro.ContaBancaria {
	t.Helper()

	numeroAgencia, err := domainfinanceiro.NovaAgencia(agencia, nil)
	if err != nil {
		t.Fatalf("agência inválida: %v", err)
	}
	numeroConta, err := domainfinanceiro.NovaConta(numero, &digito)
	if err != nil {
		t.Fatalf("conta inválida: %v", err)
	}

	conta, err := domainfinanceiro.NovaContaBancaria(domainfinanceiro.DadosContaBancaria{
		EmpresaID: uuid.New(),
		Banco:     banco,
		Tipo:      domainfinanceiro.TipoContaCorrente,
		Agencia:   numeroAgencia,
		Conta:     numeroConta,
		Apelido:   "Conta de teste",
	})
	if err != nil {
		t.Fatalf("conta bancária inválida: %v", err)
	}

	return conta
}

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}
