package financeiro

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

const versaoInicial = 1

func conferirVersao(informada, atual int, campo string, divergente error) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão " + campo + " é obrigatória")
	}
	if informada != atual {
		return divergente
	}

	return nil
}
