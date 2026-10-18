package notafiscal

import (
	"strconv"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	tamanhoChave          = 44
	inicioCNPJNaChave     = 6
	inicioDVDoCNPJNaChave = 18
	inicioModeloNaChave   = 20
	mensagemChaveInvalida = "chave de acesso inválida"
)

type Chave struct {
	valor string
}

func NovaChave(valor string) (Chave, error) {
	limpo := strings.ToUpper(strings.TrimSpace(valor))
	if len(limpo) != tamanhoChave {
		return Chave{}, domain.ErroValidacao(mensagemChaveInvalida)
	}

	for posicao := 0; posicao < tamanhoChave; posicao++ {
		caractere := limpo[posicao]
		if posicao >= inicioCNPJNaChave && posicao < inicioDVDoCNPJNaChave {
			if !alfanumerico(caractere) {
				return Chave{}, domain.ErroValidacao(mensagemChaveInvalida)
			}
			continue
		}
		if !numerico(caractere) {
			return Chave{}, domain.ErroValidacao(mensagemChaveInvalida)
		}
	}

	if digitoVerificadorDaChave(limpo[:tamanhoChave-1]) != limpo[tamanhoChave-1] {
		return Chave{}, domain.ErroValidacao(mensagemChaveInvalida)
	}

	return Chave{valor: limpo}, nil
}

func ChaveDe(valor string) Chave {
	return Chave{valor: valor}
}

func (c Chave) Valor() string {
	return c.valor
}

func (c Chave) Vazia() bool {
	return c.valor == ""
}

func (c Chave) CodigoUF() string {
	return c.trecho(0, 2)
}

func (c Chave) AnoMes() string {
	return c.trecho(2, 6)
}

func (c Chave) CNPJEmitente() string {
	return c.trecho(inicioCNPJNaChave, inicioModeloNaChave)
}

func (c Chave) Modelo() string {
	return c.trecho(20, 22)
}

func (c Chave) Serie() int {
	return c.numero(22, 25)
}

func (c Chave) Numero() int {
	return c.numero(25, 34)
}

func (c Chave) TipoEmissao() string {
	return c.trecho(34, 35)
}

func (c Chave) CodigoNumerico() string {
	return c.trecho(35, 43)
}

func (c Chave) DigitoVerificador() string {
	return c.trecho(43, 44)
}

func (c Chave) trecho(inicio, fim int) string {
	if len(c.valor) != tamanhoChave {
		return ""
	}

	return c.valor[inicio:fim]
}

func (c Chave) numero(inicio, fim int) int {
	valor, err := strconv.Atoi(c.trecho(inicio, fim))
	if err != nil {
		return 0
	}

	return valor
}

func digitoVerificadorDaChave(base string) byte {
	soma := 0
	peso := 2
	for posicao := len(base) - 1; posicao >= 0; posicao-- {
		soma += int(base[posicao]-'0') * peso
		peso++
		if peso > 9 {
			peso = 2
		}
	}

	digito := 11 - soma%11
	if digito >= 10 {
		digito = 0
	}

	return byte('0' + digito)
}

func alfanumerico(caractere byte) bool {
	return numerico(caractere) || (caractere >= 'A' && caractere <= 'Z')
}

func numerico(caractere byte) bool {
	return caractere >= '0' && caractere <= '9'
}
