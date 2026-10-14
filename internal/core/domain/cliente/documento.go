package cliente

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
)

const (
	tamanhoCPF            = 11
	mensagemDocumentoRuim = "documento deve ser um CPF ou CNPJ válido"
)

type TipoPessoa string

const (
	PessoaFisica   TipoPessoa = "pf"
	PessoaJuridica TipoPessoa = "pj"
)

func TipoPessoaDe(valor string) TipoPessoa {
	return TipoPessoa(valor)
}

func (t TipoPessoa) String() string {
	return string(t)
}

type Documento struct {
	valor  string
	pessoa TipoPessoa
}

func NovoDocumento(valor string) (Documento, error) {
	limpo := semMascara(valor)
	if len(limpo) == tamanhoCPF && somenteDigitos(limpo) {
		if !cpfValido(limpo) {
			return Documento{}, domain.ErroValidacao(mensagemDocumentoRuim)
		}

		return Documento{valor: limpo, pessoa: PessoaFisica}, nil
	}

	cnpj, err := domainempresa.NovoCNPJ(valor)
	if err != nil {
		return Documento{}, domain.ErroValidacao(mensagemDocumentoRuim)
	}

	return Documento{valor: cnpj.Digitos(), pessoa: PessoaJuridica}, nil
}

func DocumentoDe(valor string) Documento {
	pessoa := PessoaJuridica
	if len(valor) == tamanhoCPF {
		pessoa = PessoaFisica
	}

	return Documento{valor: valor, pessoa: pessoa}
}

func (d Documento) Valor() string {
	return d.valor
}

func (d Documento) Pessoa() TipoPessoa {
	return d.pessoa
}

func (d Documento) Vazio() bool {
	return d.valor == ""
}

func (d Documento) Formatado() string {
	if d.pessoa == PessoaJuridica {
		return domainempresa.CNPJDe(d.valor).Formatado()
	}
	if len(d.valor) != tamanhoCPF {
		return d.valor
	}

	return d.valor[0:3] + "." + d.valor[3:6] + "." + d.valor[6:9] + "-" + d.valor[9:11]
}

func TermoDeDocumento(valor string) string {
	return strings.ToUpper(semMascara(valor))
}

func semMascara(valor string) string {
	return strings.NewReplacer(".", "", "-", "", "/", "", " ", "").Replace(strings.TrimSpace(valor))
}

func somenteDigitos(valor string) bool {
	for _, caractere := range valor {
		if caractere < '0' || caractere > '9' {
			return false
		}
	}

	return valor != ""
}

func cpfValido(digitos string) bool {
	if strings.Count(digitos, digitos[:1]) == len(digitos) {
		return false
	}

	return digitoCPF(digitos[:9], 10) == digitos[9] && digitoCPF(digitos[:10], 11) == digitos[10]
}

func digitoCPF(base string, pesoInicial int) byte {
	soma := 0
	for indice, caractere := range base {
		soma += int(caractere-'0') * (pesoInicial - indice)
	}

	resto := soma * 10 % 11
	if resto == 10 {
		resto = 0
	}

	return byte('0' + resto)
}
