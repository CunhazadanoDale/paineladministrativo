package financeiro

import (
	"strconv"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	tamanhoMaximoAgencia       = 5
	tamanhoMaximoDigitoAgencia = 1
	tamanhoMaximoConta         = 20
	tamanhoMaximoDigitoConta   = 2
)

var semSeparadores = strings.NewReplacer(" ", "", ".", "")

type regraDeNumero struct {
	campo        string
	maximoNumero int
	maximoDigito int
}

var (
	regraAgencia = regraDeNumero{campo: "agência", maximoNumero: tamanhoMaximoAgencia, maximoDigito: tamanhoMaximoDigitoAgencia}
	regraConta   = regraDeNumero{campo: "conta", maximoNumero: tamanhoMaximoConta, maximoDigito: tamanhoMaximoDigitoConta}
)

type NumeroBancario struct {
	numero string
	digito string
}

func NovaAgencia(valor string, digito *string) (NumeroBancario, error) {
	return regraAgencia.novo(valor, digito)
}

func NovaConta(valor string, digito *string) (NumeroBancario, error) {
	return regraConta.novo(valor, digito)
}

func NumeroBancarioDe(numero string, digito *string) NumeroBancario {
	if digito == nil {
		return NumeroBancario{numero: numero}
	}

	return NumeroBancario{numero: numero, digito: *digito}
}

func (n NumeroBancario) Numero() string {
	return n.numero
}

func (n NumeroBancario) Digito() *string {
	if n.digito == "" {
		return nil
	}

	digito := n.digito

	return &digito
}

func (n NumeroBancario) Vazio() bool {
	return n.numero == ""
}

func (n NumeroBancario) Formatado() string {
	if n.digito == "" {
		return n.numero
	}

	return n.numero + "-" + n.digito
}

func (r regraDeNumero) novo(valor string, digitoInformado *string) (NumeroBancario, error) {
	numero, digitoJunto, temDigitoJunto := strings.Cut(strings.ToUpper(semSeparadores.Replace(valor)), "-")

	digito := ""
	if digitoInformado != nil {
		digito = strings.ToUpper(strings.TrimSpace(*digitoInformado))
	}
	if temDigitoJunto {
		if digito != "" {
			return NumeroBancario{}, domain.ErroValidacao("informe o dígito da " + r.campo + " uma vez só")
		}
		if digitoJunto == "" {
			return NumeroBancario{}, r.erroDigito()
		}
		digito = digitoJunto
	}

	if err := r.validarNumero(numero); err != nil {
		return NumeroBancario{}, err
	}
	if err := r.validarDigito(digito); err != nil {
		return NumeroBancario{}, err
	}

	return NumeroBancario{numero: numero, digito: digito}, nil
}

func (r regraDeNumero) validarNumero(numero string) error {
	if numero == "" {
		return domain.ErroValidacao(r.campo + " é obrigatória")
	}
	if len(numero) > r.maximoNumero || strings.Trim(numero, "0123456789") != "" {
		return domain.ErroValidacao(r.campo + " deve ter só dígitos, no máximo " + strconv.Itoa(r.maximoNumero))
	}
	if strings.Trim(numero, "0") == "" {
		return domain.ErroValidacao(r.campo + " não pode ser só zeros")
	}

	return nil
}

func (r regraDeNumero) validarDigito(digito string) error {
	if digito == "" {
		return nil
	}
	if len(digito) > r.maximoDigito || strings.Trim(digito, "0123456789X") != "" {
		return r.erroDigito()
	}

	return nil
}

func (r regraDeNumero) erroDigito() error {
	return domain.ErroValidacao("dígito da " + r.campo + " inválido")
}
