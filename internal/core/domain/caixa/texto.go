package caixa

import (
	"strings"
	"unicode/utf8"
)

const marcaDeOrdemUTF8 = "\ufeff"

var entidadesOFX = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&nbsp;", " ")

var windows1252 = [32]rune{
	'€', 0x81, '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', 0x8D, 'Ž', 0x8F,
	0x90, '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ', 0x9D, 'ž', 'Ÿ',
}

func textoUTF8(conteudo []byte) string {
	if utf8.Valid(conteudo) {
		return strings.TrimPrefix(string(conteudo), marcaDeOrdemUTF8)
	}

	var texto strings.Builder
	texto.Grow(len(conteudo))
	for _, b := range conteudo {
		switch {
		case b >= 0x80 && b < 0xA0:
			texto.WriteRune(windows1252[b-0x80])
		default:
			texto.WriteRune(rune(b))
		}
	}

	return texto.String()
}

func decodificarEntidades(valor string) string {
	if !strings.Contains(valor, "&") {
		return valor
	}

	return entidadesOFX.Replace(valor)
}

func textoLimitado(valor string, maximo int) string {
	valor = strings.Join(strings.Fields(valor), " ")
	if utf8.RuneCountInString(valor) <= maximo {
		return valor
	}

	return strings.TrimSpace(string([]rune(valor)[:maximo]))
}
