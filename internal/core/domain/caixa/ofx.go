package caixa

import (
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
)

const (
	LimiteDeLancamentosDoExtrato = 5000
	maximoDeDigitosDoValor       = 15
	formatoDataOFX               = "20060102"
	formatoMomentoOFX            = "20060102150405"
)

const (
	SemCorrespondencia        = 0
	CorrespondenciaAproximada = 1
	CorrespondenciaExata      = 2
)

type ExtratoOFX struct {
	BancoID    string
	AgenciaID  string
	ContaID    string
	Inicio     *time.Time
	Fim        *time.Time
	Saldo      *SaldoInformado
	Transacoes []TransacaoOFX
}

type SaldoInformado struct {
	Centavos int64
	Em       time.Time
}

type TransacaoOFX struct {
	FITID         string
	Data          time.Time
	ValorCentavos int64
	Descricao     string
	Documento     string
}

type marcacaoOFX struct {
	nome  string
	fecha bool
	valor string
}

type transacaoBruta struct {
	fitid      string
	data       string
	valor      string
	memo       string
	nome       string
	cheque     string
	referencia string
}

type leituraOFX struct {
	extrato   *ExtratoOFX
	transacao *transacaoBruta
	contexto  string
	contas    int
	banco     string
	agencia   string
	conta     string
	saldo     string
	saldoEm   string
	inicio    string
	fim       string
}

func LerOFX(conteudo []byte) (*ExtratoOFX, error) {
	texto := textoUTF8(conteudo)

	inicio := strings.Index(strings.ToUpper(texto), "<OFX>")
	if inicio < 0 {
		return nil, domain.ErroValidacao("arquivo não é um extrato OFX")
	}

	leitura := &leituraOFX{extrato: &ExtratoOFX{}}
	for _, marcacao := range marcacoesOFX(texto[inicio:]) {
		if err := leitura.aplicar(marcacao); err != nil {
			return nil, err
		}
	}

	return leitura.concluir()
}

func (l *leituraOFX) aplicar(m marcacaoOFX) error {
	switch {
	case m.nome == "STMTTRN" && !m.fecha:
		l.transacao = &transacaoBruta{}
	case m.nome == "STMTTRN" && m.fecha:
		if l.transacao == nil {
			return nil
		}
		if err := l.fecharTransacao(); err != nil {
			return err
		}
		l.transacao = nil
	case !m.fecha && (m.nome == "BANKACCTFROM" || m.nome == "LEDGERBAL" || m.nome == "BANKTRANLIST"):
		l.contexto = m.nome
		l.banco, l.agencia, l.conta, l.saldo, l.saldoEm, l.inicio, l.fim = "", "", "", "", "", "", ""
	case m.fecha && m.nome == l.contexto:
		if err := l.fecharContexto(); err != nil {
			return err
		}
		l.contexto = ""
	case m.fecha:
	case l.transacao != nil:
		l.transacao.preencher(m.nome, m.valor)
	default:
		l.preencher(m.nome, m.valor)
	}

	return nil
}

func (l *leituraOFX) preencher(nome, valor string) {
	switch l.contexto {
	case "BANKACCTFROM":
		switch nome {
		case "BANKID":
			l.banco = valor
		case "BRANCHID":
			l.agencia = valor
		case "ACCTID":
			l.conta = valor
		}
	case "LEDGERBAL":
		switch nome {
		case "BALAMT":
			l.saldo = valor
		case "DTASOF":
			l.saldoEm = valor
		}
	case "BANKTRANLIST":
		switch nome {
		case "DTSTART":
			l.inicio = valor
		case "DTEND":
			l.fim = valor
		}
	}
}

func (l *leituraOFX) fecharContexto() error {
	switch l.contexto {
	case "BANKACCTFROM":
		return l.fecharConta()
	case "LEDGERBAL":
		l.fecharSaldo()
	case "BANKTRANLIST":
		l.fecharPeriodo()
	}

	return nil
}

func (l *leituraOFX) fecharConta() error {
	identificacao := func(banco, agencia, conta string) string {
		return semZerosAEsquerda(soDigitos(banco)) + "|" + semZerosAEsquerda(soDigitos(antesDoHifen(agencia))) + "|" +
			semZerosAEsquerda(soDigitosELetras(conta))
	}

	l.contas++
	if l.contas > 1 && identificacao(l.banco, l.agencia, l.conta) !=
		identificacao(l.extrato.BancoID, l.extrato.AgenciaID, l.extrato.ContaID) {
		return domain.ErroValidacao("arquivo OFX com mais de uma conta; exporte uma conta por arquivo")
	}

	l.extrato.BancoID, l.extrato.AgenciaID, l.extrato.ContaID = l.banco, l.agencia, l.conta

	return nil
}

func (l *leituraOFX) fecharSaldo() {
	if l.saldo == "" || l.saldoEm == "" {
		return
	}

	centavos, errValor := centavosOFX(l.saldo)
	data, errData := dataOFX(l.saldoEm)
	if errValor != nil || errData != nil {
		return
	}
	if atual := l.extrato.Saldo; atual == nil || !data.Before(atual.Em) {
		l.extrato.Saldo = &SaldoInformado{Centavos: centavos, Em: data}
	}
}

func (l *leituraOFX) fecharPeriodo() {
	if inicio := dataOpcional(l.inicio); inicio != nil && (l.extrato.Inicio == nil || inicio.Before(*l.extrato.Inicio)) {
		l.extrato.Inicio = inicio
	}
	if fim := dataOpcional(l.fim); fim != nil && (l.extrato.Fim == nil || fim.After(*l.extrato.Fim)) {
		l.extrato.Fim = fim
	}
}

func (t *transacaoBruta) preencher(nome, valor string) {
	switch nome {
	case "FITID":
		t.fitid = valor
	case "DTPOSTED":
		t.data = valor
	case "TRNAMT":
		t.valor = valor
	case "MEMO":
		t.memo = valor
	case "NAME":
		t.nome = valor
	case "CHECKNUM":
		t.cheque = valor
	case "REFNUM":
		t.referencia = valor
	}
}

func (l *leituraOFX) fecharTransacao() error {
	numero := len(l.extrato.Transacoes) + 1
	if numero > LimiteDeLancamentosDoExtrato {
		return domain.ErroValidacao("extrato com mais de " + strconv.Itoa(LimiteDeLancamentosDoExtrato) +
			" lançamentos; exporte um período menor")
	}

	data, err := dataOFX(l.transacao.data)
	if err != nil {
		return domain.ErroValidacao("lançamento " + strconv.Itoa(numero) + " do extrato com data inválida")
	}

	valor, err := centavosOFX(l.transacao.valor)
	if err != nil {
		return domain.ErroValidacao("lançamento " + strconv.Itoa(numero) + " do extrato com valor inválido")
	}

	descricao := l.transacao.memo
	if strings.TrimSpace(descricao) == "" {
		descricao = l.transacao.nome
	}
	documento := l.transacao.cheque
	if strings.TrimSpace(documento) == "" {
		documento = l.transacao.referencia
	}

	l.extrato.Transacoes = append(l.extrato.Transacoes, TransacaoOFX{
		FITID:         strings.TrimSpace(l.transacao.fitid),
		Data:          data,
		ValorCentavos: valor,
		Descricao:     descricao,
		Documento:     documento,
	})

	return nil
}

func (l *leituraOFX) concluir() (*ExtratoOFX, error) {
	if l.contas == 0 {
		return nil, domain.ErroValidacao("extrato OFX sem conta bancária; extrato de cartão não é aceito")
	}

	return l.extrato, nil
}

func (e *ExtratoOFX) IdentificaConta() bool {
	return soDigitosELetras(e.ContaID) != ""
}

func (e *ExtratoOFX) Banco() string {
	banco := semZerosAEsquerda(soDigitos(e.BancoID))
	if banco == "" || len(banco) > 3 {
		return ""
	}

	return strings.Repeat("0", 3-len(banco)) + banco
}

func (e *ExtratoOFX) EhDaConta(conta *domainfinanceiro.ContaBancaria) bool {
	return e.Correspondencia(conta) > SemCorrespondencia
}

func (e *ExtratoOFX) Correspondencia(conta *domainfinanceiro.ContaBancaria) int {
	if banco := e.Banco(); banco != "" && banco != conta.Banco.String() {
		return SemCorrespondencia
	}

	agencia := conta.Agencia.Numero()
	if agenciaDoExtrato := soDigitos(antesDoHifen(e.AgenciaID)); agenciaDoExtrato != "" &&
		semZerosAEsquerda(agenciaDoExtrato) != semZerosAEsquerda(agencia) {
		return SemCorrespondencia
	}

	numero := conta.Conta.Numero()
	digito := ""
	if d := conta.Conta.Digito(); d != nil {
		digito = *d
	}

	antes, depois, temHifen := strings.Cut(e.ContaID, "-")
	semDigito := soDigitosELetras(antes)
	digitoDoExtrato := soDigitosELetras(depois)
	digitoDiverge := temHifen && digitoDoExtrato != "" && digito != "" && digitoDoExtrato != strings.ToUpper(digito)

	switch {
	case semDigito == "" || digitoDiverge:
		return SemCorrespondencia
	case semZerosAEsquerda(semDigito) == semZerosAEsquerda(numero):
		return CorrespondenciaExata
	case !temHifen && digito != "" && semZerosAEsquerda(semDigito) == semZerosAEsquerda(numero+digito):
		return CorrespondenciaAproximada
	case semDigito == agencia+numero || (!temHifen && digito != "" && semDigito == agencia+numero+digito):
		return CorrespondenciaAproximada
	default:
		return SemCorrespondencia
	}
}

func marcacoesOFX(texto string) []marcacaoOFX {
	var marcacoes []marcacaoOFX

	for {
		abre := strings.IndexByte(texto, '<')
		if abre < 0 {
			return marcacoes
		}
		fecha := strings.IndexByte(texto[abre:], '>')
		if fecha < 0 {
			return marcacoes
		}
		fecha += abre

		nome := strings.TrimSpace(texto[abre+1 : fecha])
		texto = texto[fecha+1:]

		if nome == "" || nome[0] == '?' || nome[0] == '!' {
			continue
		}

		marcacao := marcacaoOFX{}
		if nome[0] == '/' {
			marcacao.fecha = true
			nome = nome[1:]
		}
		if campos := strings.Fields(nome); len(campos) > 0 {
			nome = campos[0]
		}
		marcacao.nome = strings.ToUpper(nome)

		if !marcacao.fecha {
			fim := strings.IndexByte(texto, '<')
			if fim < 0 {
				fim = len(texto)
			}
			marcacao.valor = decodificarEntidades(strings.TrimSpace(texto[:fim]))
		}

		marcacoes = append(marcacoes, marcacao)
	}
}

func dataOFX(valor string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if len(valor) < len(formatoDataOFX) {
		return time.Time{}, domain.ErroValidacao("data do OFX inválida")
	}

	dia, err := time.Parse(formatoDataOFX, valor[:len(formatoDataOFX)])
	if err != nil {
		return time.Time{}, err
	}

	abre := strings.IndexByte(valor, '[')
	if abre < 0 || len(valor) < len(formatoMomentoOFX) {
		return dia, nil
	}
	momento, err := time.Parse(formatoMomentoOFX, valor[:len(formatoMomentoOFX)])
	if err != nil {
		return dia, nil
	}

	deslocamento := valor[abre+1:]
	if fim := strings.IndexAny(deslocamento, ":]"); fim >= 0 {
		deslocamento = deslocamento[:fim]
	}
	horas, err := strconv.ParseFloat(strings.TrimSpace(deslocamento), 64)
	if err != nil {
		return dia, nil
	}

	fuso := time.FixedZone("OFX", int(horas*60*60))
	local := time.Date(momento.Year(), momento.Month(), momento.Day(), momento.Hour(), momento.Minute(), momento.Second(), 0, fuso)

	return DataDe(local.In(fusoDoGrupo())), nil
}

func dataOpcional(valor string) *time.Time {
	data, err := dataOFX(valor)
	if err != nil {
		return nil
	}

	return &data
}

func centavosOFX(valor string) (int64, error) {
	valor = strings.ReplaceAll(strings.TrimSpace(valor), " ", "")

	negativo := false
	switch {
	case strings.HasPrefix(valor, "-"):
		negativo = true
		valor = valor[1:]
	case strings.HasPrefix(valor, "+"):
		valor = valor[1:]
	}

	inteira, fracao := valor, ""
	if separador := max(strings.LastIndexByte(valor, '.'), strings.LastIndexByte(valor, ',')); separador >= 0 {
		inteira, fracao = valor[:separador], valor[separador+1:]
	}
	inteira = strings.NewReplacer(".", "", ",", "").Replace(inteira)

	if len(fracao) > 2 {
		if strings.Trim(fracao[2:], "0") != "" {
			return 0, domain.ErroValidacao("valor com mais de duas casas decimais")
		}
		fracao = fracao[:2]
	}
	fracao += strings.Repeat("0", 2-len(fracao))
	if inteira == "" {
		inteira = "0"
	}

	digitos := inteira + fracao
	if len(digitos) > maximoDeDigitosDoValor || strings.Trim(digitos, "0123456789") != "" {
		return 0, domain.ErroValidacao("valor do OFX inválido")
	}

	centavos, err := strconv.ParseInt(digitos, 10, 64)
	if err != nil {
		return 0, domain.ErroValidacao("valor do OFX inválido")
	}
	if negativo {
		centavos = -centavos
	}

	return centavos, nil
}

func antesDoHifen(valor string) string {
	antes, _, _ := strings.Cut(valor, "-")

	return antes
}

func soDigitos(valor string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, valor)
}

func soDigitosELetras(valor string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		case r >= 'A' && r <= 'Z':
			return r
		default:
			return -1
		}
	}, valor)
}

func semZerosAEsquerda(valor string) string {
	return strings.TrimLeft(valor, "0")
}
