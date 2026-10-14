package cliente_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
)

func TestNovoDocumentoReconheceCPFECNPJ(t *testing.T) {
	casos := []struct {
		entrada   string
		valor     string
		formatado string
		pessoa    domaincliente.TipoPessoa
	}{
		{"529.982.247-25", "52998224725", "529.982.247-25", domaincliente.PessoaFisica},
		{" 11144477735 ", "11144477735", "111.444.777-35", domaincliente.PessoaFisica},
		{"11.222.333/0001-81", "11222333000181", "11.222.333/0001-81", domaincliente.PessoaJuridica},
		{"11222333000181", "11222333000181", "11.222.333/0001-81", domaincliente.PessoaJuridica},
	}

	for _, caso := range casos {
		t.Run(caso.entrada, func(t *testing.T) {
			documento, err := domaincliente.NovoDocumento(caso.entrada)
			if err != nil {
				t.Fatalf("NovoDocumento = %v", err)
			}
			if documento.Valor() != caso.valor || documento.Formatado() != caso.formatado || documento.Pessoa() != caso.pessoa {
				t.Errorf("documento = %q / %q / %q", documento.Valor(), documento.Formatado(), documento.Pessoa())
			}
		})
	}
}

func TestNovoDocumentoRejeitaInvalidos(t *testing.T) {
	for _, entrada := range []string{"", "529.982.247-26", "111.111.111-11", "11.222.333/0001-82", "12345", "abc", "5299822472"} {
		t.Run(entrada, func(t *testing.T) {
			if _, err := domaincliente.NovoDocumento(entrada); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("NovoDocumento(%q) = %v, esperado erro de validação", entrada, err)
			}
		})
	}
}

func TestDocumentoDeReconstroiPessoaPeloTamanho(t *testing.T) {
	if domaincliente.DocumentoDe("52998224725").Pessoa() != domaincliente.PessoaFisica {
		t.Error("documento de 11 dígitos deveria ser pessoa física")
	}
	if domaincliente.DocumentoDe("11222333000181").Pessoa() != domaincliente.PessoaJuridica {
		t.Error("documento de 14 caracteres deveria ser pessoa jurídica")
	}
}

func TestNomeDeBuscaNormalizaAcentosCaixaEEspacos(t *testing.T) {
	casos := map[string]string{
		"  Construtora   ÁGUIA  Ltda ": "construtora aguia ltda",
		"Concreção São João":           "concrecao sao joao",
		"MIA\tConstruções\n":           "mia construcoes",
	}

	for entrada, esperado := range casos {
		if obtido := domaincliente.NomeDeBusca(entrada); obtido != esperado {
			t.Errorf("NomeDeBusca(%q) = %q, esperado %q", entrada, obtido, esperado)
		}
	}
}

func TestPercentualEmPontosBase(t *testing.T) {
	for _, valido := range []int{0, 250, 10000} {
		if percentual, err := domaincliente.NovoPercentual(valido); err != nil || percentual.PontosBase() != valido {
			t.Errorf("NovoPercentual(%d) = %v, %v", valido, percentual, err)
		}
	}
	for _, invalido := range []int{-1, 10001} {
		if _, err := domaincliente.NovoPercentual(invalido); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoPercentual(%d) = %v, esperado erro de validação", invalido, err)
		}
	}
}

func TestTipoCliente(t *testing.T) {
	if tipo, err := domaincliente.NovoTipoCliente(" Construtora "); err != nil || tipo != domaincliente.TipoClienteConstrutora {
		t.Errorf("NovoTipoCliente = %q, %v", tipo, err)
	}
	if _, err := domaincliente.NovoTipoCliente("revenda"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tipo desconhecido = %v, esperado erro de validação", err)
	}
}
