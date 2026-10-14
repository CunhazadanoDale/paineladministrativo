package financeiro_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

func dadosDeConta(t *testing.T) domainfinanceiro.DadosContaBancaria {
	t.Helper()

	agencia, err := domainfinanceiro.NovaAgencia("0812", nil)
	if err != nil {
		t.Fatalf("agência de teste inválida: %v", err)
	}
	conta, err := domainfinanceiro.NovaConta("54321-0", nil)
	if err != nil {
		t.Fatalf("conta de teste inválida: %v", err)
	}

	return domainfinanceiro.DadosContaBancaria{
		EmpresaID: uuid.New(),
		Banco:     domainfinanceiro.Sicredi,
		Tipo:      domainfinanceiro.TipoContaCorrente,
		Agencia:   agencia,
		Conta:     conta,
		Apelido:   "  Sicredi movimento ",
	}
}

func TestNovaContaBancariaNasceAtivaNaVersaoInicial(t *testing.T) {
	dados := dadosDeConta(t)

	conta, err := domainfinanceiro.NovaContaBancaria(dados)
	if err != nil {
		t.Fatalf("NovaContaBancaria falhou: %v", err)
	}

	if conta.ID == uuid.Nil || conta.EmpresaID != dados.EmpresaID || !conta.Ativa || conta.Versao != 1 {
		t.Errorf("conta criada = %+v", conta)
	}
	if conta.Apelido != "Sicredi movimento" {
		t.Errorf("apelido = %q", conta.Apelido)
	}
	if conta.Agencia.Formatado() != "0812" || conta.Conta.Formatado() != "54321-0" || conta.Banco.Nome() != "Sicredi" {
		t.Errorf("dados bancários = %q %q %q", conta.Banco.Nome(), conta.Agencia.Formatado(), conta.Conta.Formatado())
	}
	if conta.CriadoEm.IsZero() || !conta.CriadoEm.Equal(conta.AtualizadoEm) {
		t.Errorf("datas = %v/%v", conta.CriadoEm, conta.AtualizadoEm)
	}
}

func TestNovaContaBancariaValidaDados(t *testing.T) {
	casos := map[string]func(*domainfinanceiro.DadosContaBancaria){
		"sem empresa":         func(d *domainfinanceiro.DadosContaBancaria) { d.EmpresaID = uuid.Nil },
		"banco fora":          func(d *domainfinanceiro.DadosContaBancaria) { d.Banco = domainfinanceiro.BancoDe("104") },
		"tipo inválido":       func(d *domainfinanceiro.DadosContaBancaria) { d.Tipo = domainfinanceiro.TipoContaDe("caixa") },
		"sem agência":         func(d *domainfinanceiro.DadosContaBancaria) { d.Agencia = domainfinanceiro.NumeroBancario{} },
		"sem conta":           func(d *domainfinanceiro.DadosContaBancaria) { d.Conta = domainfinanceiro.NumeroBancario{} },
		"sem apelido":         func(d *domainfinanceiro.DadosContaBancaria) { d.Apelido = "   " },
		"apelido muito longo": func(d *domainfinanceiro.DadosContaBancaria) { d.Apelido = strings.Repeat("a", 61) },
	}

	for nome, alterar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosDeConta(t)
			alterar(&dados)

			if _, err := domainfinanceiro.NovaContaBancaria(dados); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}
}

func TestContaBancariaAceitaApelidoNoLimite(t *testing.T) {
	dados := dadosDeConta(t)
	dados.Apelido = strings.Repeat("é", 60)

	if _, err := domainfinanceiro.NovaContaBancaria(dados); err != nil {
		t.Errorf("apelido com 60 caracteres falhou: %v", err)
	}
}

func TestContaBancariaAlterarApelido(t *testing.T) {
	conta, err := domainfinanceiro.NovaContaBancaria(dadosDeConta(t))
	if err != nil {
		t.Fatalf("NovaContaBancaria falhou: %v", err)
	}
	agora := conta.CriadoEm.Add(time.Hour)

	if err := conta.AlterarApelido(" Conta de pagamentos ", agora); err != nil {
		t.Fatalf("AlterarApelido falhou: %v", err)
	}
	if conta.Apelido != "Conta de pagamentos" || !conta.AtualizadoEm.Equal(agora) {
		t.Errorf("conta alterada = %q em %v", conta.Apelido, conta.AtualizadoEm)
	}

	if err := conta.AlterarApelido("", agora.Add(time.Hour)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("apelido vazio erro = %v", err)
	}
	if conta.Apelido != "Conta de pagamentos" || !conta.AtualizadoEm.Equal(agora) {
		t.Error("apelido inválido não pode alterar a conta")
	}
}

func TestContaBancariaAtivarEDesativarInformamMudanca(t *testing.T) {
	conta, err := domainfinanceiro.NovaContaBancaria(dadosDeConta(t))
	if err != nil {
		t.Fatalf("NovaContaBancaria falhou: %v", err)
	}
	agora := conta.CriadoEm.Add(time.Hour)

	if conta.Ativar(agora) {
		t.Error("ativar conta ativa não deveria mudar nada")
	}
	if !conta.Desativar(agora) || conta.Ativa || !conta.AtualizadoEm.Equal(agora) {
		t.Errorf("desativar = %+v", conta)
	}
	if conta.Desativar(agora.Add(time.Hour)) || !conta.AtualizadoEm.Equal(agora) {
		t.Error("desativar conta inativa não deveria mudar nada")
	}
	if !conta.Ativar(agora.Add(time.Hour)) || !conta.Ativa {
		t.Error("ativar conta inativa deveria reativar")
	}
}

func TestContaBancariaConferirVersao(t *testing.T) {
	conta, err := domainfinanceiro.NovaContaBancaria(dadosDeConta(t))
	if err != nil {
		t.Fatalf("NovaContaBancaria falhou: %v", err)
	}

	if err := conta.ConferirVersao(1); err != nil {
		t.Errorf("versão atual erro = %v", err)
	}
	if err := conta.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) || !strings.Contains(err.Error(), "versão da conta bancária é obrigatória") {
		t.Errorf("versão ausente erro = %v", err)
	}
	if err := conta.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) || !strings.Contains(err.Error(), "alterada por outra pessoa") {
		t.Errorf("versão divergente erro = %v", err)
	}
}
