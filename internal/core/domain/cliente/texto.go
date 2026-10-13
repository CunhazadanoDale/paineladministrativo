package cliente

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func NomeDeBusca(nome string) string {
	minusculo := semAcento.Replace(strings.ToLower(nome))

	return strings.Join(strings.FieldsFunc(minusculo, unicode.IsSpace), " ")
}

type rotulo struct {
	nome     string
	feminino bool
}

func masculino(nome string) rotulo {
	return rotulo{nome: nome}
}

func feminino(nome string) rotulo {
	return rotulo{nome: nome, feminino: true}
}

func (r rotulo) obrigatorio() string {
	if r.feminino {
		return r.nome + " é obrigatória"
	}

	return r.nome + " é obrigatório"
}

func textoObrigatorio(valor string, campo rotulo, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", domain.ErroValidacao(campo.obrigatorio())
	}
	if utf8.RuneCountInString(valor) > maximo {
		return "", domain.ErroValidacao(campo.nome + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return valor, nil
}

func textoOpcional(valor *string, campo string, maximo int) (*string, error) {
	if valor == nil {
		return nil, nil
	}

	limpo := strings.TrimSpace(*valor)
	if limpo == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(limpo) > maximo {
		return nil, domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return &limpo, nil
}

func emailOpcional(valor *string, campo string) (*string, error) {
	email, err := textoOpcional(valor, campo, tamanhoMaximoEmail)
	if err != nil || email == nil {
		return email, err
	}

	usuario, dominio, ok := strings.Cut(*email, "@")
	if !ok || usuario == "" || dominio == "" || strings.ContainsAny(*email, " ,;") || strings.Contains(dominio, "@") {
		return nil, domain.ErroValidacao(campo + " inválido")
	}

	return email, nil
}

const (
	tamanhoMaximoEmail    = 255
	tamanhoMaximoTelefone = 40
)
