package estoque

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const tamanhoMaximoSlug = 200

var mapaAcentos = map[rune]rune{
	'á': 'a', 'à': 'a', 'ã': 'a', 'â': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'õ': 'o', 'ô': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

type Slug struct {
	valor string
}

func NovoSlug(texto string) (Slug, error) {
	gerado := gerarSlug(texto)
	if gerado == "" {
		return Slug{}, domain.ErroValidacao("texto não gerou um slug válido")
	}
	if len(gerado) > tamanhoMaximoSlug {
		return Slug{}, domain.ErroValidacao("slug deve ter no máximo 200 caracteres")
	}

	return Slug{valor: gerado}, nil
}

func SlugDe(valor string) Slug {
	return Slug{valor: valor}
}

func (s Slug) Valor() string {
	return s.valor
}

func (s Slug) Vazio() bool {
	return s.valor == ""
}

func gerarSlug(texto string) string {
	var construido strings.Builder
	ultimoSeparador := false

	for _, letra := range strings.ToLower(strings.TrimSpace(texto)) {
		letra = semAcento(letra)

		if ehLetraOuDigitoASCII(letra) {
			construido.WriteRune(letra)
			ultimoSeparador = false
			continue
		}

		if !ultimoSeparador && construido.Len() > 0 {
			construido.WriteRune('-')
			ultimoSeparador = true
		}
	}

	return strings.Trim(construido.String(), "-")
}

func semAcento(letra rune) rune {
	if convertido, ok := mapaAcentos[letra]; ok {
		return convertido
	}

	return letra
}

func ehLetraOuDigitoASCII(letra rune) bool {
	return (letra >= 'a' && letra <= 'z') || (letra >= '0' && letra <= '9')
}
